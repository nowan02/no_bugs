package main

//#include <sys/ptrace.h>
//#include <sys/types.h>
//#include <sys/syscall.h>
//#include <stdint.h>
//unsigned long instruction(int pid, uint64_t prev_rip) {
//	return ptrace(PTRACE_PEEKDATA, pid, prev_rip, NULL);
//}
import "C"
import (
	"os"
	"runtime"
	"syscall"
	"time"
)

type tracee struct {
	Proc  *os.Process
	PGid  int
	Wstat syscall.WaitStatus // no initial value!
}

func main() {
	tracee := setup("aslr", nil)

	syscall.PtracePokeData(tracee.Proc.Pid, FindTextareaLinux(tracee.Proc.Pid)+0x129, []byte{0xCC})
	syscall.PtraceCont(tracee.Proc.Pid, 0)

	prev_rip := uint64(0)

	prev_inst := uint64(0)

	regs := syscall.PtraceRegs{}

	for {
		wpid, err := syscall.Wait4(tracee.PGid*-1, &tracee.Wstat, 0, nil)
		ErrCheck(err)

		prev_rip = regs.Rip

		prev_inst = uint64(C.instruction(C.int(wpid), C.ulong(prev_rip)))

		regs = syscall.PtraceRegs{}

		if tracee.Wstat.Exited() {
			if tracee.Proc.Pid == wpid {
				break
			}
		} else {
			if tracee.Wstat.StopSignal() == syscall.SIGTRAP && tracee.Wstat.TrapCause() != syscall.PTRACE_EVENT_CLONE {

				syscall.PtraceGetRegs(wpid, &regs)

				format_littleendian(prev_rip, regs.Rip, prev_inst)
			}
		}

		if tracee.Wstat.StopSignal() == 11 || tracee.Wstat.StopSignal() == 4 {
			println("Segfault")
			break
		}

		println("Current stop signal: ", tracee.Wstat.StopSignal())
		syscall.PtraceSingleStep(wpid)
		time.Sleep(time.Millisecond * 100)
	}

	syscall.PtraceCont(tracee.Proc.Pid, 0)
	runtime.UnlockOSThread()
}

func StartProcess(name string, argv []string) (*os.Process, error) {
	runtime.LockOSThread()
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

	println(state.String())

	return proc, nil
}

func setup(name string, argv []string) *tracee {
	proc, err := StartProcess(name, argv)

	ErrCheck(err)

	pgid, err := syscall.Getpgid(proc.Pid)

	ErrCheck(err)

	syscall.PtraceSetOptions(proc.Pid, syscall.PTRACE_O_TRACECLONE)
	syscall.PtraceSetOptions(proc.Pid, syscall.PTRACE_O_TRACEFORK)

	t := tracee{
		Proc: proc,
		PGid: pgid,
	}

	return &t
}

func ErrCheck(err error) {
	if err != nil {
		println(err.Error())
		panic(err)
	}
}
