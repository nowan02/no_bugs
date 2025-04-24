package main

import (
	"debug/dwarf"
	"debug/elf"
	"io"
	"log"
)

type DebugContext struct {
	// Tracee
	tracee *Tracee

	// DWARF debug info
	dwarfinfo *dwarf.Data

	// Source code line data
	lines []*dwarf.LineEntry

	// Entries of followed symbols
	followed_sym []*dwarf.Entry

	// Breakpoint offsets (key) and their replaced data (value)
	breakpoints map[uintptr][]byte

	// Entry point offset of the executable
	entrypoint uint64

	// Current file path
	current_file string

	// Current line on source code
	current_line int

	// Current instruction from source code
	current_instr string

	// Current position of the program count RIP
	current_pc uint64

	// Start address of the text area of the executable
	textarea_begin uint64

	// End address of the text area of the executable
	textarea_end uint64

	// Call stack of the program, stack based, where the last subprograms dwarf entry is at the top
	// which is the current subprogram the PC is in.
	callstack CallStack
}

// Initializes the debugger context
func InitContext(ExePath string) (*DebugContext, error) {

	elfFile, err := elf.Open(ExePath)
	if err != nil {
		return nil, err
	}

	dwarfinfo, err := elfFile.DWARF()
	if err != nil {
		return nil, err
	}

	elfFile.Close()

	reader := dwarfinfo.Reader()

	var lrdr *dwarf.LineReader
	var lines []*dwarf.LineEntry
	var syms []*dwarf.Entry
	var bps = make(map[uintptr][]byte)

	var ctx = &DebugContext{
		tracee:       nil,
		dwarfinfo:    dwarfinfo,
		lines:        lines,
		followed_sym: syms,
		breakpoints:  bps,
		entrypoint:   0,
		callstack:    *NewCallStack(),

		textarea_begin: 0,
		textarea_end:   0,

		current_file:  "",
		current_instr: "",
		current_line:  0,
		current_pc:    0,
	}

	for {
		entry, err := reader.Next()

		if err != nil {
			log.Fatal(err)
		}

		if entry == nil {
			break
		}

		if entry.Tag == dwarf.TagCompileUnit {
			lrdr, err = dwarfinfo.LineReader(entry)
			if err != nil {
				return nil, err
			}

			filename := entry.AttrField(dwarf.AttrName).Val.(string)
			path := entry.AttrField(dwarf.AttrCompDir).Val.(string)

			ctx.current_file = path + "/" + filename

			for {
				var line = &dwarf.LineEntry{}
				err = lrdr.Next(line)

				if err != nil {
					if err == io.EOF {
						break
					}
					return nil, err
				}

				ctx.lines = append(ctx.lines, line)
			}
		}

		if entry.Tag == dwarf.TagSubprogram && entry.AttrField(dwarf.AttrName).Val == "main" {
			ctx.entrypoint = entry.AttrField(dwarf.AttrLowpc).Val.(uint64)
			return ctx, nil
		}
	}
	return nil, nil
}

func (ctx DebugContext) PrintState() {
	println("Breakpoints")
	for _, bp := range ctx.breakpoints {
		println("\t", bp)
	}

	println("Source file:")
	println("\t", ctx.current_file)
}

func (ctx DebugContext) PrintFollowedSyms() {
	println("Variables:")
	for _, v := range ctx.followed_sym {
		offs, err := VariableOffset(v)

		if err != nil {
			print(err)
		}

		if offs < 0 {

		}
	}
}
