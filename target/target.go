package target

import (
	"debug/dwarf"
	"debug/elf"
	"log"
	"os"
	"syscall"
)

type Tracee struct {
	ElfPath string
	Proc    *os.Process
	PGid    int
	Wstat   syscall.WaitStatus // no initial value!
	Regs    *syscall.PtraceRegs
	logger  *log.Logger
}

func _startProcess(name string, argv []string) (*os.Process, error) {
	logger := log.New(os.Stdout, "Target layer: ", log.LstdFlags)
	logger.Println("Starting tracee process.")
	proc, err := os.StartProcess(name, argv, &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		Sys: &syscall.SysProcAttr{
			Ptrace:    true,
			Pdeathsig: syscall.SIGCHLD,
		},
	})

	if err != nil {
		logger.Fatalln("FATAL: Process could not be started.")
		return nil, err
	}

	logger.Println("Telling process to wait.")
	state, err := proc.Wait()

	if err != nil {
		return nil, err
	}

	logger.Println("Process no. ", proc.Pid, "has started with state: ", state.String())

	return proc, nil
}

func Setup(name string, argv []string) *Tracee {
	logger := log.New(os.Stdout, "Target layer: ", log.LstdFlags)
	proc, err := _startProcess(name, argv)

	if err != nil {
		panic(err)
	}

	logger.Println("Getting PID of tracee process.")
	pgid, err := syscall.Getpgid(proc.Pid)

	if err != nil {
		logger.Fatalln("FATAL: Failed to get PID.")
		panic(err)
	}

	syscall.PtraceSetOptions(proc.Pid, syscall.PTRACE_O_TRACECLONE)
	syscall.PtraceSetOptions(proc.Pid, syscall.PTRACE_O_TRACEFORK)

	t := Tracee{
		ElfPath: name,
		Proc:    proc,
		PGid:    pgid,
		Regs:    &syscall.PtraceRegs{},
		logger:  logger,
	}
	logger.Println("Tracee process initialized.")
	return &t
}

func (t *Tracee) GetDwarfInfo() (*dwarf.Data, error) {
	t.logger.Println("Opening executable for reading.")
	elfFile, err := elf.Open(t.ElfPath)
	if err != nil {
		t.logger.Fatalln("FATAL: Executable could not be opened.")
		return nil, err
	}

	t.logger.Println("Read DWARF information from executable.")
	dwarfinfo, err := elfFile.DWARF()
	if err != nil {
		t.logger.Panicln("FATAL: DWARF data not present in executable.")
		return nil, err
	}

	t.logger.Println("Successfully read data, closing executable.")
	elfFile.Close()

	return dwarfinfo, nil
}
