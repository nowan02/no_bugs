package main

import (
	"debug/dwarf"
	"runtime"
	"strconv"
	"syscall"
)

func (dbgs Session) Update() {
	for _, line := range dbgs.Lines {
		if line.Num == dbgs.Context.CurrentLine {
			line.Current = true
		} else {
			line.Current = false
		}
	}
	// Update vars below:

}

func (dbgs *Session) Continue(SingleStep bool) {
	err := syscall.PtraceCont(dbgs.Context.Target.Proc.Pid, 0)
	ErrCheck(err)

	for {
		wpid, err := syscall.Wait4(dbgs.Context.Target.PGid*-1, &dbgs.Context.Target.Wstat, 0, nil)
		ErrCheck(err)

		if dbgs.Context.Target.Wstat.Exited() {
			if dbgs.Context.Target.Proc.Pid == wpid {
				break
			}
		} else {
			// Debugger is currently stopped at a breakpoint or used single step.
			if dbgs.Context.Target.Wstat.StopSignal() == syscall.SIGTRAP && dbgs.Context.Target.Wstat.TrapCause() != syscall.PTRACE_EVENT_CLONE {

				syscall.PtraceGetRegs(wpid, dbgs.Context.Target.Regs)

				// Temporary, does not support stepping into other areas.
				if dbgs.Context.Target.Regs.Rip < uint64(dbgs.Context.TextareaBegin) || dbgs.Context.Target.Regs.Rip > uint64(dbgs.Context.TextareaEnd) {
					println("End of main()")
					syscall.PtraceCont(wpid, 0)
					break
				}

				// We have to substract 1 from PC to get the breakpoint address, since
				// PC advances over INT3 then stops.
				setbp, err := dbgs.Context.StepOverBreakpoint()
				println("Stepover")
				ErrCheck(err)
				if setbp {
					// Breakpoint was found in the map
					println("Stopped at breakpoint:", strconv.FormatUint(dbgs.Context.Target.Regs.Rip, 16))

					entry, err := dbgs.Context.LookForSymbolByPC()

					ErrCheck(err)

					if entry == nil {
						println("No DWARF entry for current PC, skipping")
					} else {
						// Add call stack entry if it is a subprogram.
						if entry.Tag == dwarf.TagSubprogram {

							ret := dbgs.Context.GetCurrentReturnAddress()
							dbgs.Context.CallStack.Push(entry, ret)

							if entry != nil {
								println("Subprogram: ", entry.AttrField(dwarf.AttrName).Val.(string))
							}
						}
					}

					// When base pointer value changes, and the current rbp is an entry in the callstack, we exited the subprogram.
					if dbgs.Context.CallStack.Last().ReturnAddress != dbgs.Context.Target.Regs.Rip && dbgs.Context.CallStack.ContainsAddress(dbgs.Context.Target.Regs.Rip) {
						dbgs.Context.CallStack.Pop()
					}
				}

				PrintRegisters(dbgs.Context.Target.Regs)

				_, exists := dbgs.Context.UserBreakpoints[uintptr(dbgs.Context.Target.Regs.Rip)]

				// If stopped at user bp, or single step, hand back control
				if exists && !SingleStep {
					dbgs.Context.LookForLineNo()
					return
				} else {
					err := syscall.PtraceCont(dbgs.Context.Target.Proc.Pid, 0)
					ErrCheck(err)
					continue
				}
			}
		}
	}
	runtime.UnlockOSThread()
	println("Program likely exited.")
	dbgs.isRunning = false
}

func (dbgs Session) StepInto() bool {
	l_stack := len(dbgs.Context.CallStack.Stack)

	dbgs.Continue(true)

	// Lenght of the stack increased, which means step into was successful.
	if l_stack < len(dbgs.Context.CallStack.Stack) {
		return true
		// If not, the debugger performed a single step
	} else {
		return false
	}

}

func (dbgs Session) StepOutOf() bool {
	l_stack := len(dbgs.Context.CallStack.Stack)

	dbgs.Continue(false)

	// Lenght of the stack increased, which means step into was successful.
	if l_stack > len(dbgs.Context.CallStack.Stack) {
		return true
		// If not, the debugger performed a single step
	} else {
		return false
	}
}

func (dbgs Session) BreakOnLine(lineno int) {
	LineAddress := uintptr(dbgs.Context.TextareaBegin + dbgs.Context.Lines[lineno].Address)

	// refactor, every line already has a breakpoint...
	if dbgs.Lines[lineno-1].Breakpoint {
		dbgs.Lines[lineno-1].Breakpoint = false
		exists, err := dbgs.Context.RemoveBreakpoint(LineAddress)
		if err != nil {
			dbgs.logger.Fatalln("Error occured: ", err.Error())
		}

		if exists {
			dbgs.logger.Println("Breakpoint has been removed.")
		} else {
			dbgs.logger.Fatalln("Breakpoint didn't exist but was removed anyway.")
		}

	} else {
		dbgs.Lines[lineno-1].Breakpoint = true
		dbgs.Context.SetBreakpoint(LineAddress, false)
		dbgs.logger.Println("Breakpoint was set on line ", lineno, ".")
	}
}
