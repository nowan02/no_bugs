package main

import (
	"debug/dwarf"
	"runtime"
	"slices"
	"strconv"
	"syscall"
)

func (dbgs Session) Update(result chan bool) {
	for _, line := range dbgs.Lines {
		if line.Num == dbgs.Context.CurrentLine {
			line.Current = true
		} else {
			line.Current = false
		}
	}
	// Update vars below:

}

func (dbgs *Session) Continue(SingleStep bool, result chan bool) {
	err := syscall.PtraceCont(dbgs.Context.Target.Proc.Pid, 0)
	if err != nil {
		dbgs.Context.Logger.Fatalln("FATAL ERROR: ", err.Error())
		result <- false
		return
	}

	for {
		wpid, err := syscall.Wait4(dbgs.Context.Target.PGid*-1, &dbgs.Context.Target.Wstat, 0, nil)
		ErrCheck(err)

		if dbgs.Context.Target.Wstat.Exited() {
			if dbgs.Context.Target.Proc.Pid == wpid {
				dbgs.Context.Logger.Fatalln("FATAL ERROR: Traced process exited prematurely.")
				result <- false
				break
			}
		} else {
			// Debugger is currently stopped at a breakpoint or used single step.
			if dbgs.Context.Target.Wstat.StopSignal() == syscall.SIGTRAP && dbgs.Context.Target.Wstat.TrapCause() != syscall.PTRACE_EVENT_CLONE {

				dbgs.Context.Logger.Println("Updating register values.")
				syscall.PtraceGetRegs(wpid, dbgs.Context.Target.Regs)

				// Only the main textarea is supported, stepping into library functions is not.
				if dbgs.Context.Target.Regs.Rip < uint64(dbgs.Context.TextareaBegin) || dbgs.Context.Target.Regs.Rip > uint64(dbgs.Context.TextareaEnd) {
					dbgs.Context.Logger.Println("End of main(), let the program exit gracefully.")
					syscall.PtraceCont(wpid, 0)
					break
				}

				// We have to substract 1 from PC to get the breakpoint address, since
				// PC advances over INT3 then stops.
				setbp, err := dbgs.Context.StepOverBreakpoint()
				dbgs.Context.Logger.Println("Stepover performed.")
				ErrCheck(err)
				if setbp {
					// Breakpoint was found in the map
					dbgs.Context.Logger.Println("Stopped at breakpoint:", strconv.FormatUint(dbgs.Context.Target.Regs.Rip, 16))

					entry, err := dbgs.Context.LookForSymbolByPC()

					ErrCheck(err)

					if entry == nil {
						dbgs.Context.Logger.Println("No DWARF entry for current PC, skipping")
					} else {
						// Add call stack entry if it is a subprogram.
						if entry.Tag == dwarf.TagSubprogram {

							ret := dbgs.Context.GetCurrentReturnAddress()
							dbgs.Context.CallStack.Push(entry, ret)

							dbgs.Context.Logger.Println("Subprogram: ", entry.AttrField(dwarf.AttrName).Val.(string), "was put on the stack.")

						}
					}

					// When base pointer value changes, and the current rbp is an entry in the callstack, we exited the subprogram
					// CHECK IF MORE ELEMENTS NEED TO BE POPPED, NOT JUST THE LAST!
					if dbgs.Context.CallStack.Last().ReturnAddress != dbgs.Context.Target.Regs.Rip && dbgs.Context.CallStack.ContainsAddress(dbgs.Context.Target.Regs.Rip) {
						dbgs.Context.Logger.Println("Instruction pointer points to last entry's return address, pop top element from stack.")
						dbgs.Context.CallStack.Pop()
					}
				}

				// If stopped at user bp, or step over was used, hand back control
				if slices.Contains(dbgs.Context.UserBreakpoints, uintptr(dbgs.Context.Target.Regs.Rip)) && !SingleStep {
					dbgs.Context.LookForLineNo()
					result <- true
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
	dbgs.Context.Logger.Println("Traced process has exited or the debugger was detached.")
	dbgs.isRunning = false
}

func (dbgs Session) StepInto(result chan bool) {
	l_stack := len(dbgs.Context.CallStack.Stack)

	dbgs.Continue(true, result)
	if <-result {
		// Lenght of the stack increased, which means step into was successful.
		if l_stack < len(dbgs.Context.CallStack.Stack) {
			dbgs.Context.Logger.Println("Step into successful.")
			result <- true
			// If not, the debugger performed a single step
		} else {
			dbgs.Context.Logger.Println("Step into was not possible, performed single step instead.")
			result <- true
		}
	} else {
		dbgs.Context.Logger.Fatalln("Step into could not be performed, internal continue operation likely failed.")
	}
}

func (dbgs Session) StepOutOf(result chan bool) {
	l_stack := len(dbgs.Context.CallStack.Stack)

	dbgs.Continue(false, result)

	if <-result {
		// Lenght of the stack decreased, which means step out of was successful.
		if l_stack > len(dbgs.Context.CallStack.Stack) {
			dbgs.Context.Logger.Println("Step out of successful.")
			result <- true
			// If not, the debugger performed a single step
		} else {
			dbgs.Context.Logger.Println("Step out of was not possible, performed single step instead.")
			result <- true
		}
	} else {
		dbgs.Context.Logger.Fatalln("Step out of could not be performed, internal continue operation likely failed.")
		result <- false
	}
}

func (dbgs Session) StepOver() {
	//TODO
}

func (dbgs Session) BreakOnLine(lineno int, result chan bool) {
	LineAddress := uintptr(dbgs.Context.TextareaBegin + dbgs.Context.Lines[lineno].Address)

	if dbgs.Lines[lineno-1].Breakpoint {
		dbgs.Lines[lineno-1].Breakpoint = false
		exists, err := dbgs.Context.RemoveBreakpoint(LineAddress)
		if err != nil {
			dbgs.Context.Logger.Fatalln("Error occured: ", err.Error())
			result <- false
			return
		}

		if exists {
			dbgs.Context.Logger.Println("Breakpoint has been removed.")
			result <- true
		} else {
			dbgs.Context.Logger.Fatalln("Breakpoint didn't exist but was removed anyway.")
			result <- false
		}

	} else {
		dbgs.Lines[lineno-1].Breakpoint = true
		dbgs.Context.SetBreakpoint(LineAddress, false)
		dbgs.Context.Logger.Println("Breakpoint was set on line ", lineno, ".")
		result <- true
	}
}
