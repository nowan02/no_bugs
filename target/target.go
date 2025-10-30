package target

import (
	"debug/dwarf"
	"debug/elf"
	"os"
	"syscall"
)

type Tracee struct {
	ElfPath string
	Proc    *os.Process
	PGid    int
	Wstat   syscall.WaitStatus // no initial value!
	Regs    *syscall.PtraceRegs
}

func _startProcess(name string, argv []string) (*os.Process, error) {
	proc, err := os.StartProcess(name, argv, &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		Sys: &syscall.SysProcAttr{
			Ptrace:    true,
			Pdeathsig: syscall.SIGCHLD,
		},
	})

	if err != nil {
		return nil, err
	}

	state, err := proc.Wait()

	if err != nil {
		return nil, err
	}

	println("Process no. ", proc.Pid, "has started with state: ", state.String())

	return proc, nil
}

func Setup(name string, argv []string) *Tracee {
	proc, err := _startProcess(name, argv)

	if err != nil {
		panic(err)
	}

	pgid, err := syscall.Getpgid(proc.Pid)

	if err != nil {
		panic(err)
	}

	syscall.PtraceSetOptions(proc.Pid, syscall.PTRACE_O_TRACECLONE)
	syscall.PtraceSetOptions(proc.Pid, syscall.PTRACE_O_TRACEFORK)

	t := Tracee{
		ElfPath: name,
		Proc:    proc,
		PGid:    pgid,
		Regs:    &syscall.PtraceRegs{},
	}
	return &t
}

func (t *Tracee) GetDwarfInfo() (*dwarf.Data, error) {
	elfFile, err := elf.Open(t.ElfPath)
	if err != nil {
		return nil, err
	}

	dwarfinfo, err := elfFile.DWARF()
	if err != nil {
		return nil, err
	}

	elfFile.Close()

	return dwarfinfo, nil
}
