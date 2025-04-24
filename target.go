package main

import (
	"os"
	"runtime"
	"syscall"
)

type Tracee struct {
	Proc  *os.Process
	PGid  int
	Wstat syscall.WaitStatus // no initial value!
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

func setup(name string, argv []string) *Tracee {
	proc, err := StartProcess(name, argv)

	ErrCheck(err)

	pgid, err := syscall.Getpgid(proc.Pid)

	ErrCheck(err)

	syscall.PtraceSetOptions(proc.Pid, syscall.PTRACE_O_TRACECLONE)
	syscall.PtraceSetOptions(proc.Pid, syscall.PTRACE_O_TRACEFORK)

	t := Tracee{
		Proc: proc,
		PGid: pgid,
	}

	return &t
}
