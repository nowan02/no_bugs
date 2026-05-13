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
	"sync"
	"text/template"
)

type Display struct {
	Lines     []*ssr.Row
	Variables []*ssr.Variables
}

type Session struct {
	Context   *symbol.DebugContext
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
	var Display Display
	DebugSession.isRunning = false
	DebugSession.logger = log.New(os.Stdout, "GUI: ", log.LstdFlags)

	commandChannel := make(chan Command, 1)
	resultChannel := make(chan bool, 1)

	fs := http.FileServer(http.Dir("ssr/public"))
	http.Handle("/public/", http.StripPrefix("/public/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if DebugSession.isRunning {
			DebugSession.logger.Println("Updating UI...")

			cmd := &Command{
				cmd:  "update",
				arg1: nil,
			}

			DebugSession.logger.Println("Sending command...")
			commandChannel <- *cmd

			DebugSession.logger.Println("Command sent. Awaiting result...")
			await(resultChannel, DebugSession.logger)
			DebugSession.logger.Println("Update performed, executing template.")
		}

		tmpl := template.Must(template.ParseFiles("ssr/template.html"))
		tmpl.Execute(w, Display)
	})

	http.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		if !DebugSession.isRunning {
			DebugSession.logger.Println("Starting tracee...")

			var wg sync.WaitGroup

			wg.Add(1)

			go DebugSession.Setup(commandChannel, resultChannel, &wg, &Display)

			wg.Wait()
		} else {
			cmd := &Command{
				cmd:  "stop",
				arg1: nil,
			}

			DebugSession.logger.Println("Sending command...")
			commandChannel <- *cmd

			DebugSession.logger.Println("Command sent. Awaiting result...")
			await(resultChannel, DebugSession.logger)
		}
		w.Header().Add("Content-Type", "")
		http.Redirect(w, r, "http://localhost:8080/", http.StatusTemporaryRedirect)
	})

	http.HandleFunc("/continue", func(w http.ResponseWriter, r *http.Request) {
		if !DebugSession.isRunning {
			DebugSession.logger.Println("Continue issued but tracee is not running! Start the program first.")
			w.Header().Add("Content-Type", "")
			http.Redirect(w, r, "http://localhost:8080/", http.StatusTemporaryRedirect)
			return
		}

		DebugSession.logger.Println("Continue until next breakpoint if exists.")
		cmd := &Command{
			cmd:  "continue",
			arg1: nil,
		}

		DebugSession.logger.Println("Command sent...")
		commandChannel <- *cmd

		DebugSession.logger.Println("Awaiting result.")
		await(resultChannel, DebugSession.logger)

		// If the continue loop exits, sets isRunning to false.
		if !DebugSession.isRunning {
			DebugSession.logger.Println("Tracee stopped, resetting.")
			cmd := &Command{
				cmd:  "stop",
				arg1: nil,
			}

			DebugSession.logger.Println("Sending command...")
			commandChannel <- *cmd

			DebugSession.logger.Println("Command sent. Awaiting result...")
			await(resultChannel, DebugSession.logger)

			DebugSession.logger.Println("Stopping debugger.")
			os.Exit(0)
		}

		w.Header().Add("Content-Type", "")
		http.Redirect(w, r, "http://localhost:8080", http.StatusTemporaryRedirect)
	})

	http.HandleFunc("/stepover", func(w http.ResponseWriter, r *http.Request) {
		if !DebugSession.isRunning {
			DebugSession.logger.Println("Step over issued but tracee is not running! Start the program first.")
			w.Header().Add("Content-Type", "")
			http.Redirect(w, r, "http://localhost:8080/", http.StatusTemporaryRedirect)
			return
		}

		cmd := &Command{
			cmd:  "stepover",
			arg1: nil,
		}

		DebugSession.logger.Println("Sending command...")
		commandChannel <- *cmd

		DebugSession.logger.Println("Command sent. Awaiting result...")
		await(resultChannel, DebugSession.logger)

		w.Header().Add("Content-Type", "")
		http.Redirect(w, r, "http://localhost:8080", http.StatusTemporaryRedirect)

	})

	http.HandleFunc("/stepinto", func(w http.ResponseWriter, r *http.Request) {
		if !DebugSession.isRunning {
			DebugSession.logger.Println("Step into issued but tracee is not running! Start the program first.")
			w.Header().Add("Content-Type", "")
			http.Redirect(w, r, "http://localhost:8080/", http.StatusTemporaryRedirect)
			return
		}

		cmd := &Command{
			cmd:  "stepinto",
			arg1: nil,
		}

		DebugSession.logger.Println("Sending command...")
		commandChannel <- *cmd

		DebugSession.logger.Println("Command sent. Awaiting result...")
		await(resultChannel, DebugSession.logger)

		w.Header().Add("Content-Type", "")
		http.Redirect(w, r, "http://localhost:8080", http.StatusTemporaryRedirect)

	})

	http.HandleFunc("/stepoutof", func(w http.ResponseWriter, r *http.Request) {
		if !DebugSession.isRunning {
			DebugSession.logger.Println("Step out of issued but tracee is not running! Start the program first.")
			w.Header().Add("Content-Type", "")
			http.Redirect(w, r, "http://localhost:8080/", http.StatusTemporaryRedirect)
			return
		}

		cmd := &Command{
			cmd:  "stepoutof",
			arg1: nil,
		}

		DebugSession.logger.Println("Sending command...")
		commandChannel <- *cmd

		DebugSession.logger.Println("Command sent. Awaiting result...")
		await(resultChannel, DebugSession.logger)

		w.Header().Add("Content-Type", "")
		http.Redirect(w, r, "http://localhost:8080", http.StatusTemporaryRedirect)
	})

	http.HandleFunc("/breakpoint", func(w http.ResponseWriter, r *http.Request) {
		if !DebugSession.isRunning {
			DebugSession.logger.Println("Tracee is not running! Start the program first.")
			w.Header().Add("Content-Type", "")
			http.Redirect(w, r, "http://localhost:8080/", http.StatusTemporaryRedirect)
			return
		}
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

	DebugSession.logger.Println("Serving GUI on localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func (dbgs *Session) Setup(commandChannel chan Command, resultChannel chan bool, wg *sync.WaitGroup, dp *Display) {
	runtime.LockOSThread()
	ExeName, err := filepath.Abs("./bin/demo.out")
	ErrCheck(err)

	tgt := target.Setup(ExeName, nil)

	dbgs.Context, err = symbol.InitContext(tgt)
	ErrCheck(err)

	dbgs.Context.TextareaBegin, dbgs.Context.TextareaEnd = dbgs.Context.Target.FindTextareaLinux()

	dp.Lines, err = ssr.ReadSourceFile("./bin/main.c")
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

	wg.Done()

	for {
		cmd := <-commandChannel
		dbgs.Context.Logger.Println("Command received: ", cmd.cmd)
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
				dbgs.BreakOnLine(lineno, resultChannel, dp)
			} else {
				dbgs.logger.Fatalln("ERROR: first argument for this command was not an integer.")
			}
		case "stepover":
			dbgs.StepOver(resultChannel)
		case "stop":
			dbgs.Stop()
		case "update":
			dbgs.Update(resultChannel, dp)
		default:
			dbgs.logger.Println("Invalid command.")
			resultChannel <- false
			continue
		}
	}
}

func ErrCheck(err error) {
	if err != nil {
		println(err.Error())
		panic(err)
	}
}
