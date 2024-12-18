package main

import (
	"os"
	"runtime"
	"strconv"
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

	syscall.PtracePokeData(tracee.Proc.Pid, FindTextareaLinux(tracee.Proc.Pid)+0x130, []byte{0xCC})
	syscall.PtraceCont(tracee.Proc.Pid, 0)

	data := make([]byte, 0, 1024)

	for {
		wpid, err := syscall.Wait4(tracee.PGid*-1, &tracee.Wstat, 0, nil)
		ErrCheck(err)

		if tracee.Wstat.Exited() {
			if tracee.Proc.Pid == wpid {
				break
			}
		} else {
			if tracee.Wstat.StopSignal() == syscall.SIGTRAP && tracee.Wstat.TrapCause() != syscall.PTRACE_EVENT_CLONE {
				regs := syscall.PtraceRegs{}

				syscall.PtraceGetRegs(wpid, &regs)

				syscall.PtracePokeData(tracee.Proc.Pid, FindTextareaLinux(tracee.Proc.Pid)+0x130, []byte{0x55})

				_, err := syscall.PtracePeekText(wpid, uintptr(regs.Rip), data)

				ErrCheck(err)

				println("Rip: ", strconv.FormatUint(regs.Rip, 16), " ", data)

				syscall.PtraceSingleStep(wpid)

				time.Sleep(time.Millisecond * 500)
			}
		}
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
