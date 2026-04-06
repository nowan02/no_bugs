package main

import (
	"debug/dwarf"
	"runtime"
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
		dbgs.logger.Fatalln("FATAL ERROR: ", err.Error())
		result <- false
		return
	}

	for {
		wpid, err := syscall.Wait4(dbgs.Context.Target.PGid*-1, &dbgs.Context.Target.Wstat, 0, nil)
		ErrCheck(err)

		if dbgs.Context.Target.Wstat.Exited() {
			if dbgs.Context.Target.Proc.Pid == wpid {
				dbgs.logger.Fatalln("FATAL ERROR: Traced process exited prematurely.")
				result <- false
				break
			}
		} else {
			// Debugger is currently stopped at a breakpoint or used single step.
			if dbgs.Context.Target.Wstat.StopSignal() == syscall.SIGTRAP && dbgs.Context.Target.Wstat.TrapCause() != syscall.PTRACE_EVENT_CLONE {

				dbgs.logger.Println("Updating register values.")
				syscall.PtraceGetRegs(wpid, dbgs.Context.Target.Regs)

				// Only the main textarea is supported, stepping into library functions is not.
				if dbgs.Context.Target.Regs.Rip < uint64(dbgs.Context.TextareaBegin) || dbgs.Context.Target.Regs.Rip > uint64(dbgs.Context.TextareaEnd) {
					dbgs.logger.Println("End of main(), let the program exit gracefully.")
					syscall.PtraceCont(wpid, 0)
					break
				}

				// We have to substract 1 from PC to get the breakpoint address, since
				// PC advances over INT3 then stops.
				setbp, err := dbgs.Context.StepOverBreakpoint()
				dbgs.logger.Println("Stepover performed.")
				ErrCheck(err)
				if setbp {
					// Breakpoint was found in the map
					dbgs.logger.Println("Stopped at breakpoint:", strconv.FormatUint(dbgs.Context.Target.Regs.Rip, 16))

					entry, err := dbgs.Context.LookForSymbolByPC()

					ErrCheck(err)

					if entry == nil {
						dbgs.logger.Println("No DWARF entry for current PC, skipping")
					} else {
						// Add call stack entry if it is a subprogram.
						if entry.Tag == dwarf.TagSubprogram {

							ret := dbgs.Context.GetCurrentReturnAddress()
							dbgs.Context.CallStack.Push(entry, ret)

							if entry != nil {
								dbgs.logger.Println("Subprogram: ", entry.AttrField(dwarf.AttrName).Val.(string), "was put on the stack.")
							}
						}
					}

					// When base pointer value changes, and the current rbp is an entry in the callstack, we exited the subprogram
					// CHECK IF MORE ELEMENTS NEED TO BE POPPED, NOT JUST THE LAST!
					if dbgs.Context.CallStack.Last().ReturnAddress != dbgs.Context.Target.Regs.Rip && dbgs.Context.CallStack.ContainsAddress(dbgs.Context.Target.Regs.Rip) {
						dbgs.logger.Println("Instruction pointer points to last entry's return address, pop top element from stack.")
						dbgs.Context.CallStack.Pop()
					}
				}

				_, exists := dbgs.Context.UserBreakpoints[uintptr(dbgs.Context.Target.Regs.Rip)]

				// If stopped at user bp, or single step, hand back control
				if exists && !SingleStep {
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
	dbgs.logger.Println("Traced process has exited or the debugger was detached.")
	dbgs.isRunning = false
}

func (dbgs Session) StepInto(result chan bool) {
	l_stack := len(dbgs.Context.CallStack.Stack)

	dbgs.Continue(true, result)
	if <-result {
		// Lenght of the stack increased, which means step into was successful.
		if l_stack < len(dbgs.Context.CallStack.Stack) {
			dbgs.logger.Println("Step into successful.")
			result <- true
			// If not, the debugger performed a single step
		} else {
			dbgs.logger.Println("Step into was not possible, performed single step instead.")
			result <- true
		}
	} else {
		dbgs.logger.Fatalln("Step into could not be performed, internal continue operation likely failed.")
	}
}

func (dbgs Session) StepOutOf(result chan bool) {
	l_stack := len(dbgs.Context.CallStack.Stack)

	dbgs.Continue(false, result)

	if <-result {
		// Lenght of the stack decreased, which means step out of was successful.
		if l_stack > len(dbgs.Context.CallStack.Stack) {
			dbgs.logger.Println("Step out of successful.")
			result <- true
			// If not, the debugger performed a single step
		} else {
			dbgs.logger.Println("Step out of was not possible, performed single step instead.")
			result <- true
		}
	} else {
		dbgs.logger.Fatalln("Step out of could not be performed, internal continue operation likely failed.")
		result <- false
	}
}

func (dbgs Session) BreakOnLine(lineno int, result chan bool) {
	LineAddress := uintptr(dbgs.Context.TextareaBegin + dbgs.Context.Lines[lineno].Address)

	// refactor, every line already has a breakpoint...
	if dbgs.Lines[lineno-1].Breakpoint {
		dbgs.Lines[lineno-1].Breakpoint = false
		exists, err := dbgs.Context.RemoveBreakpoint(LineAddress)
		if err != nil {
			dbgs.logger.Fatalln("Error occured: ", err.Error())
			result <- false
			return
		}

		if exists {
			dbgs.logger.Println("Breakpoint has been removed.")
			result <- true
		} else {
			dbgs.logger.Fatalln("Breakpoint didn't exist but was removed anyway.")
			result <- false
		}

	} else {
		dbgs.Lines[lineno-1].Breakpoint = true
		dbgs.Context.SetBreakpoint(LineAddress, false)
		dbgs.logger.Println("Breakpoint was set on line ", lineno, ".")
		result <- true
	}
}
