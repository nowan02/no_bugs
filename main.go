package main

import (
	"bufio"
	"debug/dwarf"
	"os"
	"runtime"
	"strconv"
	"syscall"
)

func main() {
	ExeName := "bin/empty.out"

	ctx, err := InitContext(ExeName)
	ErrCheck(err)

	ctx.tracee = setup(ExeName, nil)
	ctx.textarea_begin, ctx.textarea_end = FindTextareaLinux(ctx.tracee.Proc.Pid, ExeName)

	SetBreakpoint(ctx.tracee.Proc.Pid, uintptr(ctx.textarea_begin+ctx.entrypoint), ctx)

	for _, line := range ctx.lines {
		SetBreakpoint(ctx.tracee.Proc.Pid, uintptr(ctx.textarea_begin+line.Address), ctx)
	}

	aint, _ := LookForSymbolByName("a", dwarf.TagVariable, ctx)
	bint, _ := LookForSymbolByName("b", dwarf.TagVariable, ctx)

	type_offs, _ := aint.AttrField(dwarf.AttrType).Val.(dwarf.Offset)

	LookForSymbolByDWARFOffset(type_offs, ctx)

	ctx.followed_sym = append(ctx.followed_sym, aint, bint)

	debug(ctx)
}

func debug(Context *DebugContext) {
	syscall.PtraceCont(Context.tracee.Proc.Pid, 0)

	regs := syscall.PtraceRegs{}

	for {
		wpid, err := syscall.Wait4(Context.tracee.PGid*-1, &Context.tracee.Wstat, 0, nil)
		ErrCheck(err)

		if Context.tracee.Wstat.Exited() {
			if Context.tracee.Proc.Pid == wpid {
				break
			}
		} else {
			// Debugger is currently stopped at a breakpoint or used single step.
			if Context.tracee.Wstat.StopSignal() == syscall.SIGTRAP && Context.tracee.Wstat.TrapCause() != syscall.PTRACE_EVENT_CLONE {

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
					println("Stopped at breakpoint:", strconv.FormatUint(regs.Rip, 16))

					entry, _ := LookForSymbolByName("main", dwarf.TagSubprogram, Context)

					// Add call stack entry if it is a subprogram.
					if entry.Tag == dwarf.TagSubprogram {

						Context.callstack.Push(entry)

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

		println("\nCurrent stop signal: ", Context.tracee.Wstat.StopSignal().String())
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
