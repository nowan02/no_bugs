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
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"text/template"
)

type Session struct {
	Context *symbol.DebugContext
	Lines   []ssr.Row
}

func main() {
	DebugSession := Session{
		Context: &symbol.DebugContext{},
		Lines:   make([]ssr.Row, 0),
	}

	DebugSession.Setup()

	tmpl := template.Must(template.ParseFiles("ssr/template.html"))

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("ssr/public"))
	mux.Handle("/public/", http.StripPrefix("/public/", fs))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, DebugSession.Lines)
		DebugSession.Update()
	})

	mux.HandleFunc("/continue", func(w http.ResponseWriter, r *http.Request) {
		DebugSession.Continue(false)
		http.Redirect(w, r, "http://localhost:8080/", http.StatusTemporaryRedirect)
	})

	mux.HandleFunc("/breakpoint", func(w http.ResponseWriter, r *http.Request) {
		line := r.URL.Query().Get("line")

		lineno, err := strconv.Atoi(line)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		DebugSession.Lines[lineno].Breakpoint = true

		LineAddress := uintptr(DebugSession.Context.TextareaBegin + DebugSession.Context.Lines[lineno].Address)

		DebugSession.Context.SetBreakpoint(DebugSession.Context.Target.Proc.Pid, LineAddress, false)
	})

	http.ListenAndServe(":8080", mux)
}

func (dbgs *Session) Setup() {
	ExeName, err := filepath.Abs("../bin/empty.out")
	ErrCheck(err)

	tgt := target.Setup(ExeName, nil)

	dbgs.Context, err = symbol.InitContext(tgt)
	ErrCheck(err)

	dbgs.Context.TextareaBegin, dbgs.Context.TextareaEnd = dbgs.Context.Target.FindTextareaLinux()

	dbgs.Lines, err = ssr.ReadSourceFile("../bin/main.c")
	ErrCheck(err)

	// Place system breakpoint on all lines
	/*for _, line := range dbgs.Context.Lines {
		addr := dbgs.Context.TextareaBegin + line.Address
		dbgs.Context.SetBreakpoint(dbgs.Context.Target.Proc.Pid, uintptr(addr), false)
	}*/
	dbgs.Context.SetBreakpoint(dbgs.Context.Target.Proc.Pid, uintptr(dbgs.Context.TextareaBegin+dbgs.Context.Entrypoint), false)
}

func (dbgs *Session) SetUserBreakpoint() {

}

func (dbgs *Session) Update() {

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
					runtime.UnlockOSThread()
					break
				}

				// We have to substract 1 from PC to get the breakpoint address, since
				// PC advances over INT3 then stops.
				setbp, err := dbgs.Context.StepOverBreakpoint(wpid, uintptr(dbgs.Context.Target.Regs.Rip-1), dbgs.Context.Target.Regs)
				ErrCheck(err)
				if setbp {
					// Breakpoint was found in the map
					println("Stopped at breakpoint:", strconv.FormatUint(dbgs.Context.Target.Regs.Rip, 16))

					entry, err := dbgs.Context.LookForSymbolByPC(dbgs.Context.Target.Regs.Rip)

					if err != nil {
						println("No DWARF entry for current PC, skipping")
					} else {
						// Add call stack entry if it is a subprogram.
						if entry.Tag == dwarf.TagSubprogram {

							dbgs.Context.CallStack.Push(entry, dbgs.Context.Target.Regs.Rbp)

							if entry != nil {
								println("Subprogram: ", entry.AttrField(dwarf.AttrName).Val.(string))
							}
						}
					}
					// When base pointer value changes, we exited the subprogram.
					if dbgs.Context.CallStack.Last().Rbp != dbgs.Context.Target.Regs.Rbp && len(dbgs.Context.CallStack.Stack) > 0 {
						dbgs.Context.CallStack.Pop()
					}
				}

				PrintRegisters(dbgs.Context.Target.Regs)

				dbgs.Context.LookForLineNo()

				_, exists := dbgs.Context.SystemBreakpoints[uintptr(dbgs.Context.Target.Regs.Rip-1)]

				// System breakpoints do not give the user control,
				// exception is when using step over.
				if exists && !SingleStep {
					err := syscall.PtraceCont(dbgs.Context.Target.Proc.Pid, 0)
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
