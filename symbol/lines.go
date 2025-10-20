package symbol

import (
	"syscall"
)

func LookForLineNo(Context *DebugContext, regs *syscall.PtraceRegs) int {
	// Only one file will be supported for now!
	for _, line := range Context.Lines {
		if line.Address >= regs.Rip && line.Address > Context.TextareaBegin && line.Address < Context.TextareaEnd {
			return line.Line
		}
	}

	return -1
}
