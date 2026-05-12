package symbol

import (
	"debug/dwarf"
	"errors"
	"io"
	"log"
	"no_bugs/ssr"
	"no_bugs/target"
	"os"
	"syscall"
)

// Implementation of symbolic layer
type DebugContext struct {
	Logger *log.Logger
	// Target
	Target *target.Tracee

	// DWARF debug info in a tree form
	SymbolTreeRoot []*TreeLeaf

	// DWARF reader instance
	DwarfReader *dwarf.Reader

	// Compilation unit line data
	Lines []*dwarf.LineEntry

	// Entries of followed symbols
	FollowedSym []*ssr.Variables

	// Contains offsets of breakpoints set by user.
	UserBreakpoints []uintptr

	// Breakpoint offsets (key) and their replaced data (value)
	SystemBreakpoints map[uintptr][]byte

	// Entry point offset of the executable
	Entrypoint uint64

	// Current line on source code
	CurrentLine int

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
	Logger := log.New(os.Stdout, "Symbolic layer: ", log.LstdFlags)

	Logger.Println("Getting DWARF information from executable.")
	dwarfinfo, err := Target.GetDwarfInfo()
	if err != nil {
		return nil, err
	}
	Logger.Println("DWARF info got, creating reader.")
	reader := dwarfinfo.Reader()

	Logger.Println("Initializing context.")
	var ctx = DebugContext{
		Logger:            Logger,
		Target:            Target,
		DwarfReader:       reader,
		SymbolTreeRoot:    make([]*TreeLeaf, 0),
		Lines:             make([]*dwarf.LineEntry, 0),
		FollowedSym:       make([]*ssr.Variables, 0),
		UserBreakpoints:   make([]uintptr, 0),
		SystemBreakpoints: make(map[uintptr][]byte),
		Entrypoint:        0,
		CallStack:         *NewCallStack(),

		TextareaBegin: 0,
		TextareaEnd:   0,
		CurrentLine:   0,
	}

	ctx.SearchEntryPoint()
	ctx.InitializeSymbolTree()
	err = ctx.InitializeLineData(dwarfinfo)

	if err != nil {
		return nil, err
	}

	Logger.Println("Context successfuly initialized")
	return &ctx, nil
}

func (ctx *DebugContext) InitializeLineData(DwarfInfo *dwarf.Data) error {
	ctx.Logger.Println("Reading line data.")
	for _, lvl1 := range ctx.SymbolTreeRoot {
		if lvl1.Self.Tag != dwarf.TagCompileUnit {
			continue
		}
		ctx.Logger.Println("Compilation unit found. Initializing DWARF line reader.")

		lrdr, err := DwarfInfo.LineReader(lvl1.Self)

		if err != nil {
			ctx.Logger.Fatalln("FATAL: DWARF line reader could not be created.")
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

			ctx.Lines = append(ctx.Lines, line)
		}
		ctx.Logger.Println(len(ctx.Lines), " lines read.")
	}

	return nil
}

func (ctx *DebugContext) SearchEntryPoint() {
	ctx.Logger.Println("Reading program entrypoint.")
	entry, err := ctx.LookForSymbolByName("main", dwarf.TagSubprogram)
	if err != nil {
		ctx.Logger.Fatalln("FATAL: Failed to read program entrypoint")
		panic(err)
	}

	if entry == nil {
		ctx.Logger.Fatalln("FATAL: No main() function found, can not determine entrypoint.")
		panic(errors.New("main function not found"))
	}

	data := entry.AttrField(dwarf.AttrLowpc).Val

	if data == nil {
		ctx.Logger.Fatalln("FATAL: Entry point could not be determined.")
		panic(errors.New("entry point was missing from subprogram main"))
	}

	entrypoint, ok := data.(uint64)

	if ok {
		ctx.Logger.Println("Entry point found!")
		ctx.Entrypoint = entrypoint
	} else {
		panic(errors.New("entrypoint had unexpected type (not uint64)"))
	}
}

func (ctx *DebugContext) Detach() {
	ctx.Logger.Println("Detaching from tracee.")
	err := syscall.PtraceDetach(ctx.Target.Proc.Pid)
	if err != nil {
		ctx.Logger.Println("Detach unsuccessful, process no longer exists.")
		return
	}
	ctx.Logger.Println("Detached tracee.")
	ctx.Target.Stop()
}
