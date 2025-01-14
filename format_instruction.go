package main

import (
	"strconv"
)

// Formats the output of the current instruction by stripping excess data not on the current line.
func format_littleendian(current_rip uint64, next_rip uint64, instruction uint64) {
	//offset := (next_rip - current_rip) * 2 // byte offset hex

	inst_str := strconv.FormatUint(instruction, 16)

	println("Rip: ", strconv.FormatUint(current_rip, 16), " ", inst_str)
}
