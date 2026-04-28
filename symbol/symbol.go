package symbol

import (
	"debug/dwarf"
	"encoding/binary"
	"errors"
	"no_bugs/ssr"
	"strconv"
	"strings"
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

func decode_sleb128(data []byte) int64 {
	var result int64
	shift := 0

	var bytedata byte

	for i := range len(data) {
		bytedata = data[i]
		result |= int64(bytedata&0x7f) << shift
		shift += 7
		if (bytedata & 0x80) == 0 {
			break
		}
	}

	if shift < 64 && bytedata&0x40 != 0 {
		result |= ^0 << shift
	}

	return result
}

// Gets dwarf entry and calculates offset
// Since local variables always have a negative offset in relation to the stack base pointer
// positive values are assumed to be global offsets!
// Entry: DWARF entry of variable to resolve, length: number of bytes to read.
// Length: number of bytes to read from address
func (Context *DebugContext) getVariableValue(Entry *dwarf.Entry, length int) ([]byte, error) {
	ent := Entry.AttrField(dwarf.AttrLocation)

	if ent == nil {
		return nil, errors.New("dwarf Symbol does not contain location attribute")
	}

	data := Entry.AttrField(dwarf.AttrLocation).Val

	DW_AT_location, ok := data.([]byte)

	if ok {
		Opcode := DW_AT_location[0]

		offset_bytes := make([]byte, 0)

		for i := 1; i < len(DW_AT_location); i++ { // get bytes after opcode
			offset_bytes = append(offset_bytes, DW_AT_location[i])
		}

		// Expand offset to 8 bytes
		for len(offset_bytes) < 8 {
			offset_bytes = append(offset_bytes, 0x00)
		}

		offset_val := decode_sleb128(offset_bytes)
		data := make([]byte, 0)

		switch Opcode {
		// Global offset
		case 0x03:
			address := Context.TextareaBegin + uint64(offset_val)
			data = Context.PeekDataWrapper(uintptr(address), length)
		// Local offset
		case 0x91:
			offset_val = offset_val + 16
			address := Context.Target.Regs.Rbp + uint64(offset_val)
			data = Context.PeekDataWrapper(uintptr(address), length)
		default:
			return nil, errors.New("location data was not in the expected format ([]byte)")
		}

		// Expand data to 8 bytes in length
		for len(data) < 8 {
			data = append(data, 0x00)
		}
		return data, nil
	} else {
		return nil, errors.New("Unable to get bytes from memory location of this variable")
	}
}

func (Context *DebugContext) GetEncoding(Entry *dwarf.Entry) (signed bool, bytesize int64, name string) {
	sign := false
	size, ok := Entry.AttrField(dwarf.AttrByteSize).Val.(int64)
	if !ok {
		size = 0
	}

	varname := Entry.AttrField(dwarf.AttrName).Val.(string)

	enc, ok := Entry.AttrField(dwarf.AttrEncoding).Val.(int64)
	if ok {
		switch enc {
		case 5:
			sign = true
		case 6:
			sign = true
		case 13:
			sign = true
		}
	}

	return sign, size, varname
}

// Returns the type entry of the given DWARF entry if it exists.
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

// Resolves a single variable, prepares it for frontend.
func (Context *DebugContext) ResolveSingle(signed bool, bytesize int64, typename string, value []byte) *ssr.Variables {

	vari := &ssr.Variables{
		Vartype: typename,
		Values:  make([]string, 0),
		Name:    "Unknown",
	}

	if len(value) != 0 {
		if strings.Contains(strings.ToLower(typename), "int") {
			if signed {
				vari.Values = append(vari.Values, strconv.FormatInt(int64(binary.LittleEndian.Uint64(value)), 10))
			} else {
				vari.Values = append(vari.Values, strconv.FormatUint(binary.LittleEndian.Uint64(value), 10))
			}
			// float
		}
		if strings.Contains(strings.ToLower(typename), "char") {
			if signed {
				vari.Values = append(vari.Values, string(value))
			} else {
				vari.Values = append(vari.Values, strconv.FormatUint(binary.LittleEndian.Uint64(value), 10))
			}
		}
	}

	return vari
}

func (Context *DebugContext) ResolveVars() {
	subp, err := Context.LookForSymbolByPC()

	if err != nil {
		Context.Logger.Fatalln("FATAL: Could not resolve a symbol by Program Counter.")
		return
	}

	if subp != nil {
		if subp.Tag == dwarf.TagSubprogram {
			Context.Logger.Println("Currently stopped at function entry, skipping variable resolution.")
			return
		}
	}
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

				sign, size, name := Context.GetEncoding(typeentry)
				data, err := Context.getVariableValue(current, int(size))
				if err != nil {
					Context.Logger.Println("ERROR: ", err.Error())
				}

				out := Context.ResolveSingle(sign, size, name, data)

				out.Name = current.AttrField(dwarf.AttrName).Val.(string)

				Context.AppendSymbol(out)

				Context.DwarfReader.Seek(current.Offset)
				Context.DwarfReader.Next()
			case dwarf.TagPointerType:

				ptr, err := Context.getVariableType(current)
				if err != nil {
					Context.Logger.Println("ERROR: ", err.Error())
				}

				ptrtype, err := Context.getVariableType(ptr)

				if err != nil {
					Context.Logger.Println("ERROR: ", err.Error())
				}

				if ptrtype.Tag == dwarf.TagConstType {
					ptrtype, err = Context.getVariableType(ptrtype)
					if err != nil {
						Context.Logger.Println("ERROR: ", err.Error())
					}
				}

				sign, size, name := Context.GetEncoding(ptrtype)

				data, err := Context.getVariableValue(current, 8)

				addr := binary.LittleEndian.Uint64(data)

				if err != nil {
					Context.Logger.Println("ERROR: ", err.Error())
				}

				data = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

				if addr != 0 {
					data = make([]byte, 0)
					// If a pointer is char* type, read until null termination 0x00.
					if strings.Contains(name, "char") {
						i := uint64(0)
						for {
							char := Context.PeekDataWrapper(uintptr(addr+i), 1)[0]
							data = append(data, char)
							// if no 0x00, read until we reach the end of the stack frame.
							if char == 0x00 || uintptr(addr+i) > uintptr(Context.Target.Regs.Rsp) {
								break
							}
							i++
						}
					} else {
						data, err = Context.getVariableValue(current, int(size))
						if err != nil {
							Context.Logger.Println("ERROR: ", err.Error())
							data = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
						}
					}
				}

				out := Context.ResolveSingle(sign, size, name, data)

				out.Name = "*" + current.AttrField(dwarf.AttrName).Val.(string)

				Context.AppendSymbol(out)

				Context.DwarfReader.Seek(current.Offset)
				Context.DwarfReader.Next()

			case dwarf.TagArrayType:
				arraytype, err := Context.getVariableType(typeentry)
				if err != nil {
					Context.Logger.Println("ERROR: ", err.Error())
				}

				Context.DwarfReader.Seek(typeentry.Offset)
				// First returns the array type
				Context.DwarfReader.Next()
				// the next entry after an array type is the associated subrange
				subrangetype, err := Context.DwarfReader.Next()

				if err != nil {
					Context.Logger.Println("ERROR:", err.Error())
				}

				sign, size, name := Context.GetEncoding(arraytype)

				// Get the size of the array from subrange
				arraysize, ok := subrangetype.AttrField(dwarf.AttrUpperBound).Val.(int64)
				if !ok {
					arraysize = 0
				} else {
					// upper bound attribute reflects the maximal index, starts from 0
					arraysize++
				}

				arrayvar := Context.ResolveSingle(sign, size, name, make([]byte, 0))
				arrayvar.Vartype = arrayvar.Vartype + "[" + strconv.FormatInt(arraysize-1, 10) + "]"
				arrayvar.Name = current.AttrField(dwarf.AttrName).Val.(string)

				data, err := Context.getVariableValue(current, int(arraysize*size))

				if err != nil {
					Context.Logger.Println("ERROR:", err.Error())
				}

				var resolvedelement *ssr.Variables

				for i := int64(0); i < arraysize; i++ {
					elem_data := make([]byte, 8)
					copy(elem_data, data[i*size:i*size+size])
					resolvedelement = Context.ResolveSingle(sign, size, name, elem_data)
					arrayvar.Values = append(arrayvar.Values, resolvedelement.Values[0])
				}

				Context.AppendSymbol(arrayvar)

				Context.DwarfReader.Seek(current.Offset)
				Context.DwarfReader.Next()
			default:
				Context.Logger.Println("Unimplemented variable resolution, skipping.")
			}
		}
	}
}

// Appends a symbol to the followed symbols, if a symbol already exists, replace values.
func (ctx *DebugContext) AppendSymbol(variable *ssr.Variables) {
	for i := 0; i < len(ctx.FollowedSym); i++ {
		if ctx.FollowedSym[i].Name == variable.Name {
			ctx.FollowedSym[i] = variable
			return
		}
	}
	ctx.FollowedSym = append(ctx.FollowedSym, variable)
}
