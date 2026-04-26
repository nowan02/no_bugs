package main

import (
	"debug/dwarf"
	"slices"
	"strconv"
	"syscall"
	"time"
)

func (dbgs Session) Update(result chan bool, dp *Display) {
	for _, line := range dp.Lines {
		if line.Num == dbgs.Context.CurrentLine {
			line.Current = true
		} else {
			line.Current = false
		}
	}
	// Update vars below:
	dbgs.Context.ResolveVars()

	result <- true
}

func (dbgs *Session) Continue(SingleStep bool, result chan bool) {
	for {
		time.Sleep(50 * time.Millisecond)
		err := syscall.PtraceCont(dbgs.Context.Target.Proc.Pid, 0)
		if err != nil {
			dbgs.Context.Logger.Fatalln("FATAL ERROR: ", err.Error())
			result <- false
			return
		}

		wpid, err := syscall.Wait4(dbgs.Context.Target.PGid*-1, &dbgs.Context.Target.Wstat, 0, nil)
		if err != nil {
			dbgs.Context.Logger.Fatalln("FATAL ERROR: ", err.Error())
			result <- false
			return
		}

		if dbgs.Context.Target.Wstat.Exited() {
			if dbgs.Context.Target.Proc.Pid == wpid {
				dbgs.Context.Logger.Fatalln("Traced process exited.")
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
					dbgs.Context.Detach()
					return
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
							PrintRegisters(dbgs.Context.Target.Regs)

							// Push return address

							returnaddress := dbgs.Context.GetCurrentReturnAddress()

							dbgs.Context.CallStack.Push(entry, returnaddress)

							dbgs.Context.Logger.Println("Subprogram: ", entry.AttrField(dwarf.AttrName).Val.(string), "was put on the stack.")

						}
					}

					if dbgs.Context.CallStack.Last().ReturnAddress == dbgs.Context.Target.Regs.Rip {
						dbgs.Context.Logger.Println("Instruction pointer points to last entry's rbp, pop top element from stack.")
						dbgs.Context.CallStack.Pop()
					}
				}

				// If stopped at user bp, or step over was used, hand back control
				if slices.Contains(dbgs.Context.UserBreakpoints, uintptr(dbgs.Context.Target.Regs.Rip)) || SingleStep {
					dbgs.Context.LookForLineNo()
					result <- true
					return
				}
			}
		}
	}
	dbgs.isRunning = false
	result <- true
}

func (dbgs Session) StepInto(result chan bool) {
	l_stack := len(dbgs.Context.CallStack.Stack)

	localresult := make(chan bool, 1)

	dbgs.Continue(true, localresult)
	if <-localresult {
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
		dbgs.Context.Logger.Fatalln("ERROR: Step into could not be performed, internal continue operation likely failed.")
	}
}

func (dbgs Session) StepOutOf(result chan bool) {
	l_stack := len(dbgs.Context.CallStack.Stack)

	localresult := make(chan bool, 1)

	for {
		dbgs.Continue(true, localresult)
		if !<-localresult {
			dbgs.Context.Logger.Fatalln("ERROR: Step out of could not be performed, internal continue operation likely failed.")
			result <- false
			return
		}
		if l_stack > len(dbgs.Context.CallStack.Stack) ||
			slices.Contains(dbgs.Context.UserBreakpoints, uintptr(dbgs.Context.Target.Regs.Rip)) {
			break
		}
	}

	dbgs.Context.Logger.Println("Step out of successful.")
	result <- true
}

func (dbgs Session) StepOver(result chan bool) {
	l_stack := len(dbgs.Context.CallStack.Stack)

	localresult := make(chan bool, 1)

	dbgs.StepInto(localresult)
	if <-localresult {
		if l_stack < len(dbgs.Context.CallStack.Stack) && !slices.Contains(dbgs.Context.UserBreakpoints, uintptr(dbgs.Context.Target.Regs.Rip)) {
			dbgs.StepOutOf(localresult)
			if <-localresult {
				dbgs.Context.Logger.Println("Step out of successful.")
				result <- true
			}
		} else {
			dbgs.Context.Logger.Println("Step Over performed single step.")
			result <- true
		}
	} else {
		dbgs.Context.Logger.Fatalln("ERROR: Step Over could not performed, internal StepInto operation likely failed.")
		result <- false
	}
}

func (dbgs Session) BreakOnLine(lineno int, result chan bool, dp *Display) {
	offs := dbgs.Context.IsValidBreakpoint(lineno)
	if offs == 0 {
		result <- false
		return
	}
	LineAddress := uintptr(dbgs.Context.TextareaBegin + offs)

	if dp.Lines[lineno-1].Breakpoint {
		dp.Lines[lineno-1].Breakpoint = false
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
		dp.Lines[lineno-1].Breakpoint = true
		dbgs.Context.SetBreakpoint(LineAddress, false)
		dbgs.Context.Logger.Println("Breakpoint was set on line ", lineno, ".")
		result <- true
	}
}
