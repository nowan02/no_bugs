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
	"os"
	"runtime"
	"syscall"
	"time"
)

type tracee struct {
	Proc  *os.Process
	PGid  int
	Wstat syscall.WaitStatus // no initial value!
}

func main() {
	tracee := setup("ai.out", nil)

	textarea_begin, textarea_end := FindTextareaLinux(tracee.Proc.Pid)

	breakpoint_address := textarea_begin + 0x187

	syscall.PtracePokeData(tracee.Proc.Pid, breakpoint_address, []byte{0xCC})
	syscall.PtraceCont(tracee.Proc.Pid, 0)

	breakpoint_stop := true

	current_inst := uint64(0)

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

				current_inst = uint64(C.instruction(C.int(wpid), C.ulong(regs.Rip)))

				// THIS DOES NOT SUPPORT STEPPING INTO CLIB!!!
				if regs.Rip > uint64(textarea_end) || regs.Rip < uint64(textarea_begin) {
					println("End of main()")
					syscall.PtraceCont(wpid, 0)
					runtime.UnlockOSThread()
					break
				}

				// Program stops one byte after the breakpoint, we get the break instruction address by decrementing RIP
				// Pretty retarded but it is what it is
				if breakpoint_stop {
					// At the entry of main, "push RBP" needs to be restored so we don't destroy the stack.
					println("Stopped at breakpoint, replacing with original")

					syscall.PtracePokeData(wpid, uintptr(breakpoint_address), []byte{0xb8})

					regs.Rip -= 1

					syscall.PtraceSetRegs(wpid, &regs)

					breakpoint_stop = false
				} else {
					format_littleendian(regs.Rip, regs.Rip, current_inst)
				}
			}
		}

		println("Current stop signal: ", tracee.Wstat.StopSignal().String())
		syscall.PtraceSingleStep(wpid)
		time.Sleep(time.Millisecond * 100)
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

	println(state.String())

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

func ErrCheck(err error) {
	if err != nil {
		println(err.Error())
		panic(err)
	}
}
