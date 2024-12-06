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

func (t *Tracee) Cont(sig syscall.Signal) error {
	return syscall.PtraceCont(t.Process.Pid, int(sig))
}

func (t *Tracee) Detatch(sig syscall.Signal) error {
	return syscall.PtraceDetach(t.Process.Pid)
}

func (t *Tracee) GetEventMessage(sig syscall.Signal) (uint, error) {
	return syscall.PtraceGetEventMsg(t.Process.Pid)
}

func (t *Tracee) GetRegisters() (*syscall.PtraceRegs, error) {
	regs := &syscall.PtraceRegs{}
	err := syscall.PtraceGetRegs(t.Process.Pid, regs)

	if err != nil {
		return nil, err
	}

	return regs, nil
}

func (t *Tracee) PeekData(addr uintptr, out []byte) (int, error) {
	return syscall.PtracePeekData(t.Process.Pid, addr, out)
}

func (t *Tracee) PeekText(addr uintptr, out []byte) (int, error) {
	return syscall.PtracePeekText(t.Process.Pid, addr, out)
}

func (t *Tracee) PokeData(addr uintptr, data []byte) (int, error) {
	return syscall.PtracePokeData(t.Process.Pid, addr, data)
}

func (t *Tracee) PokeText(addr uintptr, data []byte) (int, error) {
	return syscall.PtracePokeText(t.Process.Pid, addr, data)
}

func (t *Tracee) SetOptions(options int) error {
	return syscall.PtraceSetOptions(t.Process.Pid, options)
}

func (t *Tracee) SetRegs(regs *syscall.PtraceRegs) error {
	return syscall.PtraceSetRegs(t.Process.Pid, regs)
}

func (t *Tracee) SingleStep() error {
	return syscall.PtraceSingleStep(t.Process.Pid)
}

func (t *Tracee) Syscall(sig syscall.Signal) (uint64, error) {
	err := syscall.PtraceSyscall(t.Process.Pid, int(sig))
	if err != nil {
		return 0, err
	}

	status := syscall.WaitStatus(0)
	_, err = syscall.Wait4(t.Process.Pid, &status, 0, nil)
	if err != nil {
		return 0, err
	}

	regs, err := t.GetRegisters()
	if err != nil {
		return 0, err
	}

	return regs.Orig_rax, nil
}
