package main

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"syscall"
)

type Tracee struct {
	Proc  *os.Process
	PGid  int
	Wstat syscall.WaitStatus // no initial value!
}

func main() {
	ExeName := "bin/example.out"

	ctx, err := InitContext(ExeName)
	ErrCheck(err)
	ctx.PrintState()

	Tracee := setup(ExeName, nil)
	TextAreaBegin, TextAreaEnd := FindTextareaLinux(Tracee.Proc.Pid, ExeName)
	SetBreakpoint(Tracee.Proc.Pid, TextAreaBegin+uintptr(ctx.entrypoint), ctx)

	debug(TextAreaBegin, TextAreaEnd, Tracee, ctx)

}

func debug(TextAreaBegin uintptr, TextAreaEnd uintptr, Tracee *Tracee, Context *DebugContext) {
	syscall.PtraceCont(Tracee.Proc.Pid, 0)

	breakpoint_stop := true

	regs := syscall.PtraceRegs{}

	for {
		wpid, err := syscall.Wait4(Tracee.PGid*-1, &Tracee.Wstat, 0, nil)
		ErrCheck(err)

		if Tracee.Wstat.Exited() {
			if Tracee.Proc.Pid == wpid {
				break
			}
		} else {
			if Tracee.Wstat.StopSignal() == syscall.SIGTRAP && Tracee.Wstat.TrapCause() != syscall.PTRACE_EVENT_CLONE {

				syscall.PtraceGetRegs(wpid, &regs)

				// Temporary, does not support stepping into other areas.
				if regs.Rip > uint64(TextAreaEnd) || regs.Rip < uint64(TextAreaBegin) {
					println("End of main()")
					syscall.PtraceCont(wpid, 0)
					runtime.UnlockOSThread()
					break
				}

				if breakpoint_stop {
					// At the entry of main, "push RBP" needs to be restored so we don't destroy the stack.
					println("Stopped at breakpoint ", strconv.FormatUint(regs.Rip-1, 16))

					ReplaceBreakpoint(wpid, uintptr(regs.Rip-1), &regs, Context)

					breakpoint_stop = false
				}

				PrintRegisters(&regs)
			}
		}

		println("\nCurrent stop signal: ", Tracee.Wstat.StopSignal().String())
		syscall.PtraceSingleStep(wpid)

		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
	}

	println("Program exited.")
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

	println("Process no. ", proc.Pid, " started with state: ", state.String())

	return proc, nil
}

func setup(name string, argv []string) *Tracee {
	proc, err := StartProcess(name, argv)

	ErrCheck(err)

	pgid, err := syscall.Getpgid(proc.Pid)

	ErrCheck(err)

	syscall.PtraceSetOptions(proc.Pid, syscall.PTRACE_O_TRACECLONE)
	syscall.PtraceSetOptions(proc.Pid, syscall.PTRACE_O_TRACEFORK)

	t := Tracee{
		Proc: proc,
		PGid: pgid,
	}

	return &t
}

// REFACTOR!
func ErrCheck(err error) {
	if err != nil {
		println(err.Error())
		panic(err)
	}
}
