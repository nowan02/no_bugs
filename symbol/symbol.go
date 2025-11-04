package symbol

import (
	"debug/dwarf"
	"encoding/binary"
	"errors"
)

// Looks for a symbol in the symbol table and returns the corresponding entry.
// Name and tag should be specified.
// Returns nil if the symbol is not found.
func (Context *DebugContext) LookForSymbolByName(SymbolName string, SymbolType dwarf.Tag) (*dwarf.Entry, error) {
	Context.DwarfReader.Seek(0)

	for {
		entry, err := Context.DwarfReader.Next()

		if entry == nil {
			break
		}

		if err != nil {
			return nil, err
		}

		if entry.AttrField(dwarf.AttrName) != nil {
			if entry.AttrField(dwarf.AttrName).Val == SymbolName && entry.Tag == SymbolType {
				return entry, nil
			}
		}
	}

	return nil, nil
}

// Should be used when stopped at a breakpoint! It subtracts one from the current Rip to take
// account for that case.
func (Context *DebugContext) LookForSymbolByPC() (*dwarf.Entry, error) {

	offset := Context.Target.Regs.Rip - Context.TextareaBegin - 1

	Context.DwarfReader.Seek(0)

	for {
		entry, err := Context.DwarfReader.Next()

		if entry == nil {
			break
		}

		if err != nil {
			return nil, err
		}

		if entry.AttrField(dwarf.AttrLowpc) != nil {
			if entry.AttrField(dwarf.AttrLowpc).Val == offset && entry.Tag != dwarf.TagCompileUnit {
				return entry, nil
			}
		}
	}

	return nil, nil
}

func (Context *DebugContext) LookForSymbolByDWARFOffset(offset dwarf.Offset) (*dwarf.Entry, error) {

	Context.DwarfReader.Seek(offset)

	entry, err := Context.DwarfReader.Next()

	return entry, err
}

// Gets dwarf entry and calculates offset
// Since local variables always have a negative offset in relation to the stack pointer
// positive values are assumed to be global offsets!
func (Context *DebugContext) getVariableValue(Entry *dwarf.Entry, StackBase int64) (uint64, error) {
	data := Entry.AttrField(dwarf.AttrLocation).Val

	if data == nil {
		return 0, errors.New("dwarf Symbol does not contain location attribute")
	}

	DW_AT_location, ok := data.([]byte)

	if ok {
		Opcode := DW_AT_location[0]

		var offset_bytes []byte

		for i := 1; i < len(DW_AT_location); i++ { // get bytes after opcode
			offset_bytes = append(offset_bytes, DW_AT_location[i])
		}

		var offset_val int64 = 0
		var data []byte = make([]byte, 0)

		switch Opcode {
		// Global offset
		case 0x03:
			offset_val = int64(binary.LittleEndian.Uint64(offset_bytes))
			address := Context.TextareaBegin + uint64(offset_val)
			data = Context._peekDataWrapper(uintptr(address), 8)
		// Local offset
		case 0x91:
			offset_val = int64(binary.LittleEndian.Uint64(offset_bytes)) - 128 + 16
			data = Context._peekDataWrapper(uintptr(StackBase+offset_val), 8)
		default:
			return 0, errors.New("location data was not in the expected format ([]byte)")
		}

		return uint64(binary.LittleEndian.Uint64(data)), nil
	} else {
		return 0, errors.New("unable to get bytes from memory location of this variable")
	}
}
