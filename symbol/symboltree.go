package symbol

import (
	"debug/dwarf"
	"log"
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

func (Context *DebugContext) SeekEntry() {
	Context.DwarfReader.Seek(0)

	for _, compunit := range Context.SymbolTreeRoot {
		compunit.
	}
}
