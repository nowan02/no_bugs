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
	"log"
	"net/http"
	"no_bugs/ssr"
	"no_bugs/symbol"
	"no_bugs/target"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"text/template"
)

type Session struct {
	Context   *symbol.DebugContext
	Lines     []*ssr.Row
	isRunning bool
	logger    *log.Logger
}

type Command struct {
	cmd  string
	arg1 interface{}
}

func await(ret chan bool, log *log.Logger) {
	for {
		result := <-ret
		if result {
			log.Println("Command ran successfully.")
			return
		} else {
			log.Println("Command ran with error.")
			return
		}
	}
}

func main() {
	var DebugSession Session
	DebugSession.isRunning = false
	DebugSession.logger = log.New(os.Stdout, "GUI: ", log.LstdFlags)

	commandChannel := make(chan Command, 1)
	resultChannel := make(chan bool, 1)

	fs := http.FileServer(http.Dir("ssr/public"))
	http.Handle("/public/", http.StripPrefix("/public/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		DebugSession.logger.Println("Updating UI...")
		DebugSession.Update(resultChannel)
		DebugSession.logger.Println("Update successful, executing template.")
		tmpl := template.Must(template.ParseFiles("ssr/template.html"))
		tmpl.Execute(w, DebugSession.Lines)
	})

	http.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		if !DebugSession.isRunning {
			DebugSession.logger.Println("Starting tracee...")
			go DebugSession.Setup(commandChannel, resultChannel)
		}
		w.Header().Add("Content-Type", "")
		http.Redirect(w, r, "http://localhost:8080/", http.StatusTemporaryRedirect)
	})

	http.HandleFunc("/continue", func(w http.ResponseWriter, r *http.Request) {
		DebugSession.logger.Println("Continue until next breakpoint if exists.")
		cmd := &Command{
			cmd:  "continue",
			arg1: nil,
		}

		DebugSession.logger.Println("Command sent...")
		commandChannel <- *cmd

		DebugSession.logger.Println("Awaiting result.")
		await(resultChannel, DebugSession.logger)

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

		cmd := &Command{
			cmd:  "breakpoint",
			arg1: lineno,
		}

		DebugSession.logger.Println("Sending command...")
		commandChannel <- *cmd

		DebugSession.logger.Println("Command sent. Awaiting result...")
		await(resultChannel, DebugSession.logger)

		w.Header().Add("Content-Type", "")
		http.Redirect(w, r, "http://localhost:8080", http.StatusTemporaryRedirect)
	})

	http.ListenAndServe(":8080", nil)
}

func (dbgs *Session) Setup(commandChannel chan Command, resultChannel chan bool) {
	runtime.LockOSThread()
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
		dbgs.Context.SetBreakpoint(uintptr(addr), true)
	}

	// User breakpoint on the entry main()
	dbgs.Context.SetBreakpoint(uintptr(dbgs.Context.TextareaBegin+dbgs.Context.Entrypoint), false)

	dbgs.Continue(false, resultChannel)
	await(resultChannel, dbgs.logger)

	dbgs.isRunning = true

	for {
		cmd := <-commandChannel
		dbgs.logger.Println("Command received.")
		switch cmd.cmd {
		case "continue":
			dbgs.Continue(false, resultChannel)
		case "singlestep":
			dbgs.Continue(true, resultChannel)
		case "stepinto":
			dbgs.StepInto(resultChannel)
		case "stepoutof":
			dbgs.StepOutOf(resultChannel)
		case "breakpoint":
			lineno, ok := cmd.arg1.(int)
			if ok {
				dbgs.BreakOnLine(lineno, resultChannel)
			} else {
				dbgs.logger.Fatalln("ERROR: first argument for this command was not an integer.")
			}
		case "stepover":
			dbgs.StepOver()
		case "stop":
			dbgs.Context.Detach()
			dbgs.logger.Println("Debugger detached, handing back control to OS.")
			os.Exit(0)
		default:
			dbgs.logger.Println("Invalid command.")
			continue
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
