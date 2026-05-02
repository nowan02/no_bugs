package symbol

//#include <sys/ptrace.h>
//#include <sys/types.h>
//#include <sys/syscall.h>
//#include <stdint.h>
//unsigned long PeekData(int Pid, uint64_t rip) {
//	return ptrace(PTRACE_PEEKDATA, Pid, rip, NULL);
//}
import "C"

import (
	"errors"
	"slices"
	"strconv"
	"syscall"
)

// Wrapper for C function, go's implementation is broken
// Read n bytes of data from location
func (Context *DebugContext) PeekDataWrapper(address uintptr, length int) []byte {
	data := make([]byte, length)

	for i := 0; i < length; i++ {
		data[i] = byte(C.PeekData(C.int(Context.Target.Proc.Pid), C.uint64_t(address)))
		address++
	}

	return data
}

// Sets breakpoint and saves replaced data to context
// systemcreated is false when user places a breakpoint
func (Context *DebugContext) SetBreakpoint(address uintptr, systemcreated bool) error {

	if systemcreated {
		original := Context.PeekDataWrapper(address, 1)

		if original[0] == 0xCC {
			Context.Logger.Println("Address ", strconv.FormatUint(uint64(address), 16), " already has a breakpoint.")
			return errors.New("Existing breakpoint can not be replaced with another breakpoint.")
		}

		_, err := syscall.PtracePokeData(Context.Target.Proc.Pid, address, []byte{0xCC})
		if err != nil {
			return err
		}

		Context.SystemBreakpoints[address] = original
	} else {
		Context.UserBreakpoints = append(Context.UserBreakpoints, address)
	}
	return nil
}

// Replaces INT3, steps over then places the breakpoint back.
// Use this to handle a breakpoint you currently stopped at.
// Returns true if breakpoint exists and was handled, false if the breakpoint does not exist
func (Context *DebugContext) StepOverBreakpoint() (bool, error) {

	data, exists := Context.SystemBreakpoints[uintptr(Context.Target.Regs.Rip-1)]

	if exists {

		_, err := syscall.PtracePokeData(Context.Target.Proc.Pid, uintptr(Context.Target.Regs.Rip-1), data)
		if err != nil {
			return false, err
		}

		// INT3 stops the program after its evaluation,
		// set back PC with 1 byte after replacement to rerun the correct instruction.
		Context.Target.Regs.Rip -= 1
		err = syscall.PtraceSetRegs(Context.Target.Proc.Pid, Context.Target.Regs)
		if err != nil {
			return false, err
		}

		err = syscall.PtraceSingleStep(Context.Target.Proc.Pid)
		if err != nil {
			return false, err
		}

		Context.SetBreakpoint(uintptr(Context.Target.Regs.Rip), true)

		return exists, nil
	}

	return exists, nil
}

// Removes a breakpoint, only use it if the breakpoint was previously handled
// by StepOverBreakpoint or was not hit at all.
// Returns true if breakpoint exists and was handled, false if the breakpoint does not exist
func (Context *DebugContext) RemoveBreakpoint(address uintptr) (bool, error) {
	data, exists := Context.SystemBreakpoints[address]
	if exists {
		_, err := syscall.PtracePokeData(Context.Target.Proc.Pid, address, data)
		if err != nil {
			return false, err
		}

		if slices.Contains(Context.UserBreakpoints, address) {
			index := slices.Index(Context.UserBreakpoints, address)
			Context.UserBreakpoints = slices.Delete(Context.UserBreakpoints, index, index)
		} else {
			delete(Context.SystemBreakpoints, address)
		}
	}
	return exists, nil
}
