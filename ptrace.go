package main

import (
	"os"
	"syscall"
)

type Tracee struct {
	Process *os.Process
}

func StartProcess(name string, argv []string) (*Tracee, error) {
	proc, err := os.StartProcess(name, argv, &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		Sys: &syscall.SysProcAttr{
			Ptrace:    true,
			Pdeathsig: syscall.SIGCHLD,
		},
	})

	trac := &Tracee{
		Process: proc,
	}

	if err != nil {
		return nil, err
	}

	return trac, nil
}

// Attach is only necessary when the Process was not started by the debugger itself, which is basically never.
func Attach(tracee *Tracee) error {
	err := syscall.PtraceAttach(tracee.Process.Pid)
	if err == syscall.EPERM {
		_, err := syscall.PtraceGetEventMsg(tracee.Process.Pid)
		if err != nil {
			return err
		}
	}

	if err != nil {
		return err
	}

	return nil
}

// Continue until tracee sends signal in input.
func (t *Tracee) Cont(sig syscall.Signal) error {
	return syscall.PtraceCont(t.Process.Pid, int(sig))
}

// Detach from the tracee
func (t *Tracee) Detatch(sig syscall.Signal) error {
	return syscall.PtraceDetach(t.Process.Pid)
}

// Event messages are sent by the tracee to indicate how the process continued.
func (t *Tracee) GetEventMessage(sig syscall.Signal) (uint, error) {
	return syscall.PtraceGetEventMsg(t.Process.Pid)
}

// Gets byte data from the address supplied, either a register or memory address.
func (t *Tracee) PeekData(addr uintptr, out []byte) (int, error) {
	return syscall.PtracePeekData(t.Process.Pid, addr, out)
}

// Write bytes to register or memory address.
func (t *Tracee) PokeData(addr uintptr, data []byte) (int, error) {
	return syscall.PtracePokeData(t.Process.Pid, addr, data)
}

// Options for tracee behaviour under certain circumstances. (eg.: on fork, on exit...)
func (t *Tracee) SetOptions(options int) error {
	return syscall.PtraceSetOptions(t.Process.Pid, options)
}

// Modifiy general purpose registers
func (t *Tracee) SetRegs(regs *syscall.PtraceRegs) error {
	return syscall.PtraceSetRegs(t.Process.Pid, regs)
}

// Step to next instruction.
func (t *Tracee) SingleStep() error {
	return syscall.PtraceSingleStep(t.Process.Pid)
}

func (t *Tracee) Wait() (syscall.WaitStatus, error) {
	state, err := t.Process.Wait()
	if err != nil {
		return state.Sys().(syscall.WaitStatus), err
	}
	return state.Sys().(syscall.WaitStatus), nil
}

// Send a syscall to the tracee and check the response
func (t *Tracee) Syscall(sig syscall.Signal) (uint64, error) {
	err := syscall.PtraceSyscall(t.Process.Pid, int(sig))
	if err != nil {
		return 0, err
	}

	// Wait4 waits for a status change in the tracee
	status := syscall.WaitStatus(0)
	_, err = syscall.Wait4(t.Process.Pid, &status, 0, nil)

	if err != nil {
		return 0, err
	}

	// Minek az RAX?
	regs, err := t.GetRegisters()
	if err != nil {
		return 0, err
	}

	return regs.Orig_rax, nil
}

// Get general purpose register values.
func (t *Tracee) GetRegisters() (*syscall.PtraceRegs, error) {
	regs := &syscall.PtraceRegs{}
	err := syscall.PtraceGetRegs(t.Process.Pid, regs)

	if err != nil {
		return nil, err
	}

	return regs, nil
}
