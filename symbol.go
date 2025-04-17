package main

import "debug/dwarf"

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
