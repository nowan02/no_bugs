package symbol

import (
	"debug/dwarf"
	"log"
	"strconv"
)

// Leaves without parents are always Compilation Units.
type TreeLeaf struct {
	Parent   *TreeLeaf
	Self     *dwarf.Entry
	Children []*TreeLeaf
}

func InitNodes(Reader *dwarf.Reader, Parent *TreeLeaf) *TreeLeaf {
	entry, err := Reader.Next()

	if err != nil {
		log.Fatal(err)
	}

	if entry == nil {
		return nil
	}

	var Leaf = &TreeLeaf{
		Parent:   Parent,
		Self:     entry,
		Children: make([]*TreeLeaf, 0),
	}

	if entry.Children {
		for {
			Child := InitNodes(Reader, Leaf)

			if Child == nil {
				break
			}

			if Parent != nil {
				if Child.Self.Tag == Parent.Self.Tag || Child.Self.Tag == 0 {
					break
				}
			}

			Leaf.Children = append(Leaf.Children, Child)
		}
	}

	return Leaf
}

func (Context *DebugContext) InitializeSymbolTree() {
	Context.DwarfReader.Seek(0)
	for {
		CompUnit := InitNodes(Context.DwarfReader, nil)
		if CompUnit == nil {
			break
		}
		Context.SymbolTreeRoot = append(Context.SymbolTreeRoot, CompUnit)
	}
}

func (Context *DebugContext) SeekNode(Name string) *Variable {
	Context.DwarfReader.Seek(0)

	for _, compunit := range Context.SymbolTreeRoot {
		if compunit.Self.Tag == dwarf.TagCompileUnit && compunit.Self.AttrField(dwarf.AttrName).Val == Context.CurrentFile {
			newVar := Context.seek(Name, compunit)
			return newVar
		}
	}

	return nil
}

func (Context *DebugContext) seek(Name string, Parent *TreeLeaf) *Variable {
	var NewVar = &Variable{
		Name:  Name,
		Value: "Unknown",
		Type:  "Unknown",
	}

	if Parent.Children == nil {
		return NewVar
	}

	for _, child := range Parent.Children {
		if child.Self.Tag == dwarf.TagVariable && child.Self.AttrField(dwarf.AttrName).Val == Name {

			typeEntry, err := Context.LookForSymbolByDWARFOffset(dwarf.Offset(child.Self.AttrField(dwarf.AttrType).Attr))

			if err != nil {
				NewVar.Type = err.Error()
			}

			offs, err := VariableOffset(child.Self)

			if err != nil {
				NewVar.Value = err.Error()
			}

			NewVar.Type = typeEntry.AttrField(dwarf.AttrName).Attr.GoString()

			NewVar.Value = strconv.FormatInt(valueEntry, 10)

			return NewVar
		} else {
			return Context.seek(Name, child)
		}
	}

	return NewVar
}
