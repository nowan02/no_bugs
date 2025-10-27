package main

//#include <sys/ptrace.h>
//#include <sys/types.h>
//#include <sys/syscall.h>
//#include <stdint.h>
//unsigned long PtraceCont(int pid) {
//	return ptrace(PTRACE_CONT, pid, NULL, 0);
//}
import "C"

import (
	"debug/dwarf"
	"net/http"
	"no_bugs/ssr"
	"no_bugs/symbol"
	"no_bugs/target"
	"strconv"
	"syscall"
	"text/template"
)

type Session struct {
	Context symbol.DebugContext
	Tracee  target.Tracee
	Lines   []ssr.Row
}

func main() {
	var DebugSession Session

	DebugSession.Setup()

	tmpl := template.Must(template.ParseFiles("ssr/template.html"))

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, DebugSession.Lines)
		DebugSession.Update()
	})

	mux.HandleFunc("/continue", func(w http.ResponseWriter, r *http.Request) {
		DebugSession.Continue(false)
		http.Redirect(w, r, "localhost:8080/", http.StatusTemporaryRedirect)
	})

	http.ListenAndServe(":8080", mux)
}

func (dbgs *Session) Setup() {
	ExeName := "../bin/empty.out"

	dbgs.Tracee = *target.Setup(ExeName, nil)

	ctx, err := symbol.InitContext(&dbgs.Tracee)
	ErrCheck(err)

	ctx.TextareaBegin, ctx.TextareaEnd = dbgs.Tracee.FindTextareaLinux()

	dbgs.Context = *ctx

	dbgs.Lines, err = ssr.ReadSourceFile("../bin/main.c")
	ErrCheck(err)

	// Place system breakpoint on all lines
	/*for _, line := range dbgs.Context.Lines {
		addr := dbgs.Context.TextareaBegin + line.Address
		dbgs.Context.SetBreakpoint(dbgs.Tracee.Proc.Pid, uintptr(addr), false)
	}*/
	dbgs.Context.SetBreakpoint(dbgs.Tracee.Proc.Pid, uintptr(ctx.TextareaBegin+ctx.Entrypoint), false)
	// WHAT?
}

func (dbgs *Session) Update() {

}

func (dbgs *Session) Continue(SingleStep bool) {
	err := syscall.PtraceCont(dbgs.Context.Target.Proc.Pid, 0)
	if err != nil {
		return
	}

	for {
		wpid, err := syscall.Wait4(dbgs.Tracee.PGid*-1, &dbgs.Tracee.Wstat, 0, nil)
		ErrCheck(err)

		if dbgs.Tracee.Wstat.Exited() {
			if dbgs.Tracee.Proc.Pid == wpid {
				break
			}
		} else {
			// Debugger is currently stopped at a breakpoint or used single step.
			if dbgs.Tracee.Wstat.StopSignal() == syscall.SIGTRAP && dbgs.Tracee.Wstat.TrapCause() != syscall.PTRACE_EVENT_CLONE {

				syscall.PtraceGetRegs(wpid, dbgs.Tracee.Regs)

				// Temporary, does not support stepping into other areas.
				/*if dbgs.Tracee.Regs.Rip < uint64(dbgs.Context.TextareaBegin) || dbgs.Tracee.Regs.Rip > uint64(dbgs.Context.TextareaEnd) {
					println("End of main()")
					syscall.PtraceCont(wpid, 0)
					runtime.UnlockOSThread()
					break
				}*/

				// We have to substract 1 from PC to get the breakpoint address, since
				// PC advances over INT3 then stops.
				setbp, err := dbgs.Context.StepOverBreakpoint(wpid, uintptr(dbgs.Tracee.Regs.Rip-1), dbgs.Tracee.Regs)
				ErrCheck(err)
				if setbp {
					// Breakpoint was found in the map
					println("Stopped at breakpoint:", strconv.FormatUint(dbgs.Tracee.Regs.Rip, 16))

					entry, err := dbgs.Context.LookForSymbolByPC(dbgs.Tracee.Regs.Rip)

					if err != nil {
						println("No DWARF entry for current PC, skipping")
					} else {
						// Add call stack entry if it is a subprogram.
						if entry.Tag == dwarf.TagSubprogram {

							dbgs.Context.CallStack.Push(entry, dbgs.Tracee.Regs.Rbp)

							if entry != nil {
								println("Subprogram: ", entry.AttrField(dwarf.AttrName).Val.(string))
							}
						}
					}
					// When base pointer value changes, we exited the subprogram.
					if dbgs.Context.CallStack.Last().Rbp != dbgs.Tracee.Regs.Rbp && len(dbgs.Context.CallStack.Stack) > 0 {
						dbgs.Context.CallStack.Pop()
					}
				}

				PrintRegisters(dbgs.Tracee.Regs)

				dbgs.Context.CurrentLine = symbol.LookForLineNo(&dbgs.Context, dbgs.Tracee.Regs)

				_, exists := dbgs.Context.SystemBreakpoints[uintptr(dbgs.Tracee.Regs.Rip-1)]

				// System breakpoints do not give the user control,
				// exception is when using step over.
				if exists && !SingleStep {
					err := syscall.PtraceCont(dbgs.Tracee.Proc.Pid, 0)
					ErrCheck(err)
					continue
				} else {
					return
				}
			}
		}
	}
	println("Program likely exited.")
}

// REFACTOR!
func ErrCheck(err error) {
	if err != nil {
		println(err.Error())
		panic(err)
	}
}
