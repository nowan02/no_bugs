package main

import (
	"strconv"
	"syscall"
)

func PrintRegisters(registers *syscall.PtraceRegs) {
	println("Current Program state:")
	println("Rip: ", strconv.FormatUint(registers.Rip, 16))
	println("Rax: ", strconv.FormatUint(registers.Rax, 16))
	println("Rsp: ", strconv.FormatUint(registers.Rsp, 16))
	println("Rbp: ", strconv.FormatUint(registers.Rbp, 16))
	println("Rdx: ", strconv.FormatUint(registers.Rdx, 16))
	println("Rbx: ", strconv.FormatUint(registers.Rdx, 16))
	println("Rcx: ", strconv.FormatUint(registers.Rcx, 16))
	println("Rdi: ", strconv.FormatUint(registers.Rdi, 16))
	println("Rsi: ", strconv.FormatUint(registers.Rsi, 16))
}
