package main

import (
	"debug/dwarf"
	"encoding/binary"
	"errors"
)

// Looks for a symbol in the symbol table and returns the corresponding entry.
// Name and tag should be specified.
// Returns nil if the symbol is not found.
func LookForSymbolByName(SymbolName string, SymbolType dwarf.Tag, Context *DebugContext) (*dwarf.Entry, error) {
	reader := Context.dwarfinfo.Reader()
	for {
		entry, err := reader.Next()

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

func LookForSymbolByPC(Context *DebugContext) (*dwarf.Entry, error) {
	reader := Context.dwarfinfo.Reader()

	offset := Context.current_pc - Context.textarea_begin - 1

	for {
		entry, err := reader.Next()

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

func LookForSymbolByDWARFOffset(offset dwarf.Offset, Context *DebugContext) (*dwarf.Entry, error) {
	reader := Context.dwarfinfo.Reader()

	reader.Seek(offset)

	entry, err := reader.Next()

	return entry, err
}

// Gets dwarf entry and calculates offset
// Since local variables always have a negative offset in relation to the stack pointer
// positive values are assumed to be global offsets!
func VariableOffset(Entry *dwarf.Entry) (int64, error) {
	data := Entry.AttrField(dwarf.AttrLocation).Val

	if data == nil {
		return 0, errors.New("dwarf Symbol does not contain location attribute")
	}

	DW_AT_location, ok := data.([]byte)

	if ok {
		Opcode := DW_AT_location[0]

		var offset_val []byte

		for i := 1; i < len(DW_AT_location); i++ { // get bytes after opcode
			offset_val = append(offset_val, DW_AT_location[i])
		}

		switch Opcode {
		case 0x03:
			return int64(binary.LittleEndian.Uint64(offset_val)), nil
		case 0x91:
			return int64(binary.LittleEndian.Uint64(offset_val)) - 128 + 16, nil
		}

	}

	return 0, errors.New("location data was not in the expected format ([]byte)")
}
