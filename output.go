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

func format_littleendian(current_rip uint64, prev_rip uint64, prev_inst uint64) {
	//offset := (next_rip - current_rip) * 2 // byte offset hex

	inst_str := strconv.FormatUint(prev_inst, 16)

	offset := int(current_rip-prev_rip) * 2

	formatted_inst := ""

	for i := len(inst_str) - 1; i >= len(inst_str)-offset; i-- {
		formatted_inst += string(inst_str[i])
	}

	println("Current instruction: ", formatted_inst)
}
