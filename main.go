package main

import (
	"debug/dwarf"
	"no_bugs/symbol"
	"no_bugs/target"
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

	ctx.SetBreakpoint(Tracee.Proc.Pid, uintptr(ctx.TextareaBegin+ctx.Entrypoint), true)

	debug(ctx, Tracee)
}

func serve() {

}

func Setup() (ctx *symbol.DebugContext, tracee *target.Tracee) {
	ExeName := "../bin/empty.out"

	Tracee := target.Setup(ExeName, nil)

	ctx, err := symbol.InitContext(Tracee)
	ErrCheck(err)

	// Place breakpoint on all subproram entries! Subprograms inside a comp unit are all on level 2 of the graph(?)
	for _, lvl2 := range ctx.SymbolTreeRoot {
		if lvl2.Self.Tag == dwarf.TagSubprogram {
			ctx.SetBreakpoint(Tracee.Proc.Pid, uintptr(ctx.TextareaBegin+uint64(lvl2.Self.Offset)), true)
			// When base pointer value changes, we exited the subprogram.
		}
	}

	return ctx, Tracee
}

func Continue(Context *symbol.DebugContext, Tracee *target.Tracee) {
	syscall.PtraceCont(Tracee.Proc.Pid, 0)

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

				syscall.PtraceGetRegs(wpid, Tracee.Regs)

				// Temporary, does not support stepping into other areas.
				if Tracee.Regs.Rip < uint64(Context.TextareaBegin) || Tracee.Regs.Rip > uint64(Context.TextareaEnd) {
					println("End of main()")
					syscall.PtraceCont(wpid, 0)
					runtime.UnlockOSThread()
					break
				}

				// We have to substract 1 from PC to get the breakpoint address, since
				// PC advances over INT3 then stops.
				if Context.StepOverBreakpoint(wpid, uintptr(Tracee.Regs.Rip-1), Tracee.Regs) {
					// Breakpoint was found in the map
					println("Stopped at breakpoint:", strconv.FormatUint(Tracee.Regs.Rip, 16))

					entry, err := Context.LookForSymbolByPC(Tracee.Regs.Rip)

					if err != nil {
						println("No DWARF entry for current PC, skipping")
					} else {
						// Add call stack entry if it is a subprogram.
						if entry.Tag == dwarf.TagSubprogram {

							Context.CallStack.Push(entry, Tracee.Regs.Rbp)

							if entry != nil {
								println("Subprogram: ", entry.AttrField(dwarf.AttrName).Val.(string))
							}
						}
					}
				}

				PrintRegisters(Tracee.Regs)

				Context.CurrentLine = symbol.LookForLineNo(Context, Tracee.Regs)

				_, exists := Context.SystemBreakpoints[uintptr(Tracee.Regs.Rip-1)]

				// System breakpoints do not give the user control
				if exists {
					syscall.PtraceCont(Tracee.Proc.Pid, 0)
					continue
				} else {
					return
				}
			}
		}
	}
}

func StepOver(Context *symbol.DebugContext, Tracee *target.Tracee) {
	// Special case for RETURN
	opcode := symbol.PeekDataWrapper(Tracee.Proc.Pid, uintptr(Tracee.Regs.Rip), 1)

	Context.SetBreakpoint(Tracee.Proc.Pid)
	// swap contents of systembreakpoints with userbreakpoints?
}

// REFACTOR!
func ErrCheck(err error) {
	if err != nil {
		println(err.Error())
		panic(err)
	}
}
