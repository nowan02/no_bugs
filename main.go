package main

import (
	"strconv"
	"syscall"
)

func main() {
	Traced, err := StartProcess("aslr", nil)

	if err != nil {
		println("Process can't be started. Exiting.")
		return
	}

	Traced.SetOptions(syscall.PTRACE_O_TRACECLONE)
	Traced.SetOptions(syscall.PTRACE_O_TRACEFORK)

	Traced.Wait4()

	regs, _ := Traced.GetRegisters()

	println(strconv.FormatUint(regs.Rip, 16))

	Traced.SingleStep()

	Traced.Wait4()

	println(Traced.Wstat.TrapCause())

	regs, _ = Traced.GetRegisters()

	println("Rip: ", strconv.FormatUint(regs.Rip, 16))

	out := []byte{}

	Traced.PeekData(uintptr(regs.Rip), out)

	println(out)

	Traced.Cont(syscall.SIGCONT)
}
