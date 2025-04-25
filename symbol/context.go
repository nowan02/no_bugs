package symbol

import (
	"debug/dwarf"
	"errors"
	"io"
	"no_bugs/target"
)

// Implementation of symbolic layer
type DebugContext struct {
	// Target
	Target *target.Tracee

	// DWARF debug info in a tree form
	SymbolTreeRoot []*TreeLeaf

	// DWARF reader instance
	DwarfReader *dwarf.Reader

	// Compilation unit line data
	Lines map[*dwarf.Entry][]*dwarf.LineEntry

	// Entries of followed symbols
	FollowedSym []*dwarf.Entry

	// Breakpoint offsets (key) and their replaced data (value)
	Breakpoints map[uintptr][]byte

	// Entry point offset of the executable
	Entrypoint uint64

	// Current file path
	CurrentFile string

	// Current line on source code
	CurrentLine int

	// Current instruction from source code
	CurrentInstr string

	// Start address of the text area of the executable
	TextareaBegin uint64

	// End address of the text area of the executable
	TextareaEnd uint64

	// Call stack of the program, stack based, where the last subprograms dwarf entry is at the top
	// which is the current subprogram the PC is in.
	CallStack CallStack
}

// Initializes the debugger context
func InitContext(Target *target.Tracee) (*DebugContext, error) {
	dwarfinfo, err := Target.GetDwarfInfo()
	if err != nil {
		return nil, err
	}

	reader := dwarfinfo.Reader()

	var ctx = &DebugContext{
		Target:         Target,
		DwarfReader:    reader,
		SymbolTreeRoot: make([]*TreeLeaf, 0),
		Lines:          make(map[*dwarf.Entry][]*dwarf.LineEntry),
		FollowedSym:    make([]*dwarf.Entry, 0),
		Breakpoints:    make(map[uintptr][]byte),
		Entrypoint:     0,
		CallStack:      *NewCallStack(),

		TextareaBegin: 0,
		TextareaEnd:   0,

		CurrentFile:  "",
		CurrentInstr: "",
		CurrentLine:  0,
	}

	ctx.SearchEntryPoint()
	ctx.InitializeSymbolTree()
	err = ctx.InitializeLineData(dwarfinfo)

	if err != nil {
		return nil, err
	}

	return ctx, nil
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

func (Context *DebugContext) InitializeLineData(DwarfInfo *dwarf.Data) error {
	for _, lvl1 := range Context.SymbolTreeRoot {
		if lvl1.Self.Tag != dwarf.TagCompileUnit {
			continue
		}

		lrdr, err := DwarfInfo.LineReader(lvl1.Self)

		if err != nil {
			return err
		}

		for {
			var line = &dwarf.LineEntry{}
			err = lrdr.Next(line)

			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}

			Context.Lines[lvl1.Self] = append(Context.Lines[lvl1.Self], line)
		}
	}

	return nil
}

func (ctx *DebugContext) SearchEntryPoint() {
	entry, err := ctx.LookForSymbolByName("main", dwarf.TagSubprogram)
	if err != nil {
		panic(err)
	}

	if entry == nil {
		panic(errors.New("main function not found"))
	}

	data := entry.AttrField(dwarf.AttrLowpc).Val

	if data == nil {
		panic(errors.New("entry point was missing from subprogram main"))
	}

	entrypoint, ok := data.(uint64)

	if ok {
		ctx.Entrypoint = entrypoint
	} else {
		panic(errors.New("entrypoint had unexpected type (not uint64)"))
	}
}

func (ctx *DebugContext) PrintState() {
	println("Breakpoints")
	for _, bp := range ctx.Breakpoints {
		println("\t", bp)
	}

	println("Source file:")
	println("\t", ctx.CurrentFile)
}

func (ctx DebugContext) PrintFollowedSyms() {
	println("Variables:")
	for _, v := range ctx.FollowedSym {
		offs, err := VariableOffset(v)

		if err != nil {
			print(err)
		}

		if offs < 0 {

		}
	}
}
