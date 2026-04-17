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

func (Context *DebugContext) LookForSymbolByPC() (*dwarf.Entry, error) {

	offset := Context.Target.Regs.Rip - Context.TextareaBegin

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
// Since local variables always have a negative offset in relation to the stack base pointer
// positive values are assumed to be global offsets!
// Entry: DWARF entry of variable to resolve, length: number of bytes to read.
func (Context *DebugContext) getVariableValue(Entry *dwarf.Entry, length int) (uint64, error) {
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
			data = Context.PeekDataWrapper(uintptr(address), length)
		// Local offset
		case 0x91:
			offset_val = int64(binary.LittleEndian.Uint64(offset_bytes)) - 128 + 16
			address := int64(Context.Target.Regs.Rbp) + offset_val
			data = Context.PeekDataWrapper(uintptr(address), length)
		default:
			return 0, errors.New("location data was not in the expected format ([]byte)")
		}

		return uint64(binary.LittleEndian.Uint64(data)), nil
	} else {
		return 0, errors.New("Unable to get bytes from memory location of this variable")
	}
}

func (Context *DebugContext) getVariableType(Entry *dwarf.Entry) (*dwarf.Entry, error) {
	data := Entry.AttrField(dwarf.AttrType).Val

	if data == nil {
		return nil, errors.New("dwarf Symbol does not contain type attribute")
	}

	offs, ok := data.(dwarf.Offset)

	if ok {
		typeentry, err := Context.LookForSymbolByDWARFOffset(offs)

		if err != nil {
			return nil, errors.New("Unable to read type entry from offset.")
		}

		return typeentry, nil
	} else {
		return nil, errors.New("Unable to convert type offset.")
	}
}

func (Context *DebugContext) ResolveScopedVars() {
	Context.DwarfReader.Seek(0)
	Context.DwarfReader.Seek(Context.CallStack.Last().Self.Offset)

	for {
		current, err := Context.DwarfReader.Next()
		if current.Tag == 0 {
			break
		}
		if err != nil {
			Context.Logger.Println("ERROR: DwarfReader failed while resolving variables.")
			return
		}
		if current.Tag == dwarf.TagVariable || current.Tag == dwarf.TagFormalParameter {
			typeentry, err := Context.getVariableType(current)
			if err != nil {
				Context.Logger.Println("ERROR: ", err.Error())
			}

			switch typeentry.Tag {
			case dwarf.TagBaseType:
			case dwarf.TagPointerType:
			case dwarf.TagArrayType:
			default:
			}
		}
	}
}
