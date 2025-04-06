package main

//#include <sys/ptrace.h>
//#include <sys/types.h>
//#include <sys/syscall.h>
//#include <stdint.h>
//unsigned long instruction(int pid, uint64_t rip) {
//	return ptrace(PTRACE_PEEKDATA, pid, rip, NULL);
//}
import "C"
import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"syscall"
)

type tracee struct {
	Proc  *os.Process
	PGid  int
	Wstat syscall.WaitStatus // no initial value!
}

func main() {
	parseElf("bin/ai.out")
}

func debug() {
	tracee := setup("ai.out", nil)

	textarea_begin, textarea_end := FindTextareaLinux(tracee.Proc.Pid)

	breakpoint_address := textarea_begin + 0x18c

	syscall.PtracePokeData(tracee.Proc.Pid, breakpoint_address, []byte{0xCC})
	syscall.PtraceCont(tracee.Proc.Pid, 0)

	breakpoint_stop := true

	regs := syscall.PtraceRegs{}

	for {
		wpid, err := syscall.Wait4(tracee.PGid*-1, &tracee.Wstat, 0, nil)
		ErrCheck(err)

		if tracee.Wstat.Exited() {
			if tracee.Proc.Pid == wpid {
				break
			}
		} else {
			if tracee.Wstat.StopSignal() == syscall.SIGTRAP && tracee.Wstat.TrapCause() != syscall.PTRACE_EVENT_CLONE {

				syscall.PtraceGetRegs(wpid, &regs)

				// Temporary, does not support stepping into other areas.
				if regs.Rip > uint64(textarea_end) || regs.Rip < uint64(textarea_begin) {
					println("End of main()")
					syscall.PtraceCont(wpid, 0)
					runtime.UnlockOSThread()
					break
				}

				if breakpoint_stop {
					// At the entry of main, "push RBP" needs to be restored so we don't destroy the stack.
					println("Stopped at breakpoint ", strconv.FormatUint(regs.Rip-1, 16))

					ReplaceBreakpoint(wpid, uintptr(breakpoint_address), &regs, []byte{0xc9})

					breakpoint_stop = false
				}

				PrintRegisters(&regs)
			}
		}

		println("\nCurrent stop signal: ", tracee.Wstat.StopSignal().String())
		syscall.PtraceSingleStep(wpid)

		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
	}

	println("Program exited.")
}

func StartProcess(name string, argv []string) (*os.Process, error) {
	runtime.LockOSThread()
	proc, err := os.StartProcess(name, argv, &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		Sys: &syscall.SysProcAttr{
			Ptrace:    true,
			Pdeathsig: syscall.SIGCHLD,
		},
	})

	if err != nil {
		return nil, err
	}

	state, err := proc.Wait()

	if err != nil {
		return nil, err
	}

	println("Process no. ", proc.Pid, " started with state: ", state.String())

	return proc, nil
}

func setup(name string, argv []string) *tracee {
	proc, err := StartProcess(name, argv)

	ErrCheck(err)

	pgid, err := syscall.Getpgid(proc.Pid)

	ErrCheck(err)

	syscall.PtraceSetOptions(proc.Pid, syscall.PTRACE_O_TRACECLONE)
	syscall.PtraceSetOptions(proc.Pid, syscall.PTRACE_O_TRACEFORK)

	t := tracee{
		Proc: proc,
		PGid: pgid,
	}

	return &t
}

// REFACTOR!
func ErrCheck(err error) {
	if err != nil {
		println(err.Error())
		panic(err)
	}
}
