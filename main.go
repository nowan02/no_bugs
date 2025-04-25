package main

import (
	"bufio"
	"debug/dwarf"
	"no_bugs/symbol"
	"no_bugs/target"
	"os"
	"runtime"
	"strconv"
	"syscall"
)

func main() {
	ExeName := "../bin/empty.out"

	Tracee := target.Setup(ExeName, nil)

	ctx, err := symbol.InitContext(Tracee)
	ErrCheck(err)

	ctx.TextareaBegin, ctx.TextareaEnd = Tracee.FindTextareaLinux()

	ctx.SetBreakpoint(Tracee.Proc.Pid, uintptr(ctx.TextareaBegin+ctx.Entrypoint))

	debug(ctx, Tracee)
}

func debug(Context *symbol.DebugContext, Tracee *target.Tracee) {
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

				// Temporary, does not support stepping into other areas.
				if regs.Rip < uint64(Context.TextareaBegin) || regs.Rip > uint64(Context.TextareaEnd) {
					println("End of main()")
					syscall.PtraceCont(wpid, 0)
					runtime.UnlockOSThread()
					break
				}

				// We have to substract 1 from PC to get the breakpoint address, since
				// PC advances over INT3 then stops.
				if Context.StepOverBreakpoint(wpid, uintptr(regs.Rip-1), &regs) {
					// Breakpoint was found in the map
					println("Stopped at breakpoint:", strconv.FormatUint(regs.Rip, 16))

					entry, _ := Context.LookForSymbolByName("main", dwarf.TagSubprogram)

					// Add call stack entry if it is a subprogram.
					if entry.Tag == dwarf.TagSubprogram {

						Context.CallStack.Push(entry, regs.Rbp)

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

// REFACTOR!
func ErrCheck(err error) {
	if err != nil {
		println(err.Error())
		panic(err)
	}
}
