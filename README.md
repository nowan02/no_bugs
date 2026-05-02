# Debugger demo for thesis work

This project is a debugger demo made as a proof-of-concept to show the basic functionality of a debugger on X86_64 Linux systems which targets single source file binaries compiled from C.

## Features

* Html based GUI
* Logger on the console
* Commands:
    * Continue
    * Step into
    * Step out of
    * Step over
    * Start/stop traced process
* Local variable resolution:
    * Base types
    * Pointers of base types
    * Arrays
    * !!! structures are currently not supported!!!

## Setup
* Includes a demo file which contains all symbols the debugger supports. Currently it is hard coded, but you may change it inside main.go > Setup() function:
```go
func (dbgs *Session) Setup(commandChannel chan Command, resultChannel chan bool, wg *sync.WaitGroup, dp *Display) {
	runtime.LockOSThread()
	ExeName, err := filepath.Abs("/bin/demo.out") // Source binary!
	ErrCheck(err)

	tgt := target.Setup(ExeName, nil)

	dbgs.Context, err = symbol.InitContext(tgt)
	ErrCheck(err)

	dbgs.Context.TextareaBegin, dbgs.Context.TextareaEnd = dbgs.Context.Target.FindTextareaLinux()

	dp.Lines, err = ssr.ReadSourceFile("/bin/main.c") // Source code!
	ErrCheck(err)
    ...
```

* Binaries need to be compiled with these flags using GCC:

    `gcc -g -gdwarf-5 -gdwarf32 -O0 <input file> -o <output file>`

* What you need to run:
    * 64bit Linux-based operating system
    * Go 1.23.5 (The project used only used libraries that are present on the base install of the language.)
    * A web browser for the GUI

* Navigate to the root directory of the project, build then run it using these commands:

    `go build .` then
`go run .`

* The GUI will serve on the address localhost:8080/
* Log messages are displayed on the console to show stages.

    `GUI: 2026/05/02 16:59:07 Serving GUI on localhost:8080`

## GUI

The GUI supports display of local variables and 5 commands on the top row:

![Command row](/images/commands.png)

* **Continue**
    * Resumes program execution until a breakpoint is hit. If no breakpoint exists, the traced process will run, then the debugger exits
* **Step over**
    * If stopped at a function call, the debugger will skip the function body entirely and proceed to the next line. Otherwise it proceeds to the next line in the code.
* **Step into**
    * If stopped at a function call, the debugger will follow execution into that functions body and stop at the first statement. Otherwise proceeds to the next line in the code.
* **Step out of**
    * Skips the rest of the current function body, then stops at the first statement after the function returned. Otherwise proceeds to the next line in the code.
* **Start/Stop**
    * Starts the process to be debugged. The previous commands cannot be ran unless the traced process has started!
    * If the traced process is running, then it kills the process and exits the debugger.

Breakpoints can be placed by clicking on the blue line numbers on the left side of the source code display. If the number turns red, the breakpoint was successfully placed on that row. **The debugger will always stop at a breakpoint regardless of the command issued!**

![No breakpoint placed](/images/bp1.png)

If the line has a valid statement, a breakpoint can be placed.

![Breakpoint placed](/images/bp2.png)

Variables are displayed next to the source code, each are updated as the execution progresses.

The display format is **Type** | **Name** | **Value(s)**

![Variables in main()](/images/vars1.png)

 When the process enters a new scope, the local variables are cleared from the previous scope and the new ones load.

 ![Variables in outside_function()](/images/vars2.png)