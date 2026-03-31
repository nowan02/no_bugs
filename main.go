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
	"net/http"
	"no_bugs/ssr"
	"no_bugs/symbol"
	"no_bugs/target"
	"path/filepath"
	"runtime"
	"strconv"
	"text/template"
)

type Session struct {
	Context   *symbol.DebugContext
	Lines     []*ssr.Row
	isRunning bool
}

type Command struct {
	cmd  string
	arg1 interface{}
}

func main() {
	runtime.LockOSThread()
	var DebugSession Session
	DebugSession.isRunning = false

	commandChannel := make(chan Command, 1)
	errorChannel := make(chan error, 1)

	fs := http.FileServer(http.Dir("ssr/public"))
	http.Handle("/public/", http.StripPrefix("/public/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		DebugSession.Update()
		tmpl := template.Must(template.ParseFiles("ssr/template.html"))
		tmpl.Execute(w, DebugSession.Lines)
	})

	http.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		if !DebugSession.isRunning {
			go DebugSession.Setup(commandChannel, errorChannel, DebugSession)
		}
		w.Header().Add("Content-Type", "")
		http.Redirect(w, r, "http://localhost:8080/", http.StatusTemporaryRedirect)
	})

	http.HandleFunc("/continue", func(w http.ResponseWriter, r *http.Request) {
		DebugSession.Continue(false)
		w.Header().Add("Content-Type", "")
		http.Redirect(w, r, "http://localhost:8080/", http.StatusTemporaryRedirect)
	})

	http.HandleFunc("/breakpoint", func(w http.ResponseWriter, r *http.Request) {
		line := r.URL.Query().Get("line")

		lineno, err := strconv.Atoi(line)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		LineAddress := uintptr(DebugSession.Context.TextareaBegin + DebugSession.Context.Lines[lineno].Address)

		// refactor, every line already has a breakpoint...
		if DebugSession.Lines[lineno-1].Breakpoint {
			DebugSession.Lines[lineno-1].Breakpoint = false
			DebugSession.Context.RemoveBreakpoint(LineAddress)
		} else {
			DebugSession.Lines[lineno-1].Breakpoint = true
			DebugSession.Context.SetBreakpoint(LineAddress, false)
		}

		w.Header().Add("Content-Type", "")
		http.Redirect(w, r, "http://localhost:8080", http.StatusTemporaryRedirect)
	})

	http.ListenAndServe(":8080", nil)
}

func (dbgs *Session) Setup(commandChannel chan Command, errorChannel chan<- error, debugSession Session) {
	ExeName, err := filepath.Abs("../bin/empty.out")
	ErrCheck(err)

	tgt := target.Setup(ExeName, nil)

	dbgs.Context, err = symbol.InitContext(tgt)
	ErrCheck(err)

	dbgs.Context.TextareaBegin, dbgs.Context.TextareaEnd = dbgs.Context.Target.FindTextareaLinux()

	dbgs.Lines, err = ssr.ReadSourceFile("../bin/main.c")
	ErrCheck(err)

	// Place system breakpoint on all lines
	for _, line := range dbgs.Context.Lines {
		addr := dbgs.Context.TextareaBegin + line.Address
		if addr != dbgs.Context.TextareaBegin+dbgs.Context.Entrypoint {
			dbgs.Context.SetBreakpoint(uintptr(addr), true)
		}
	}

	// User breakpoint on the entry main()
	dbgs.Context.SetBreakpoint(uintptr(dbgs.Context.TextareaBegin+dbgs.Context.Entrypoint), false)

	dbgs.isRunning = true

	for {
		select {
		case cmd := <-commandChannel:
			HandleCommand(cmd, debugSession)

		}
	}
}

// REFACTOR!
func ErrCheck(err error) {
	if err != nil {
		println(err.Error())
		panic(err)
	}
}
