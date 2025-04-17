package main

import (
	"bufio"
	"debug/dwarf"
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

	Tracee := setup(ExeName, nil)
	ctx.textarea_begin, ctx.textarea_end = FindTextareaLinux(Tracee.Proc.Pid, ExeName)

	SetBreakpoint(Tracee.Proc.Pid, uintptr(ctx.textarea_begin+ctx.entrypoint), ctx)

	debug(Tracee, ctx)
}

func debug(Tracee *Tracee, Context *DebugContext) {
	syscall.PtraceCont(Tracee.Proc.Pid, 0)

	regs := syscall.PtraceRegs{}

	for {
		wpid, err := syscall.Wait4(Tracee.PGid*-1, &Tracee.Wstat, 0, nil)
		ErrCheck(err)

		if Tracee.Wstat.Exited() {
			if Tracee.Proc.Pid == wpid {
				break
			}
		} else {
			// Debugger is currently stopped at a breakpoint or used single step.
			if Tracee.Wstat.StopSignal() == syscall.SIGTRAP && Tracee.Wstat.TrapCause() != syscall.PTRACE_EVENT_CLONE {

				syscall.PtraceGetRegs(wpid, &regs)

				Context.current_pc = regs.Rip

				// Temporary, does not support stepping into other areas.
				if regs.Rip > uint64(Context.textarea_end) || regs.Rip < uint64(Context.textarea_begin) {
					println("End of main()")
					syscall.PtraceCont(wpid, 0)
					runtime.UnlockOSThread()
					break
				}

				// We have to substract 1 from PC to get the breakpoint address, since
				// PC advances over INT3 then stops.
				if StepOverBreakpoint(wpid, uintptr(regs.Rip-1), &regs, Context) {
					// Breakpoint was found in the map
					println("Stopped at breakpoint:", strconv.FormatUint(regs.Rip-1, 16))

					entry, _ := LookForSymbolByPC(Context)

					// Add call stack entry if it is a subprogram.
					if entry.Tag == dwarf.TagSubprogram {

						ofs := GetCFAOffset(wpid, &regs)
						Context.callstack.Push(entry, ofs)

						if entry != nil {
							println("Subprogram: ", entry.AttrField(dwarf.AttrName).Val.(string))
						}
					}

					reader := bufio.NewReader(os.Stdin)
					reader.ReadString('\n')
				}

				PrintRegisters(&regs)
			}
		}

		println("\nCurrent stop signal: ", Tracee.Wstat.StopSignal().String())
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
		PrintRegisters(&regs)
		syscall.PtraceSingleStep(wpid)
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
