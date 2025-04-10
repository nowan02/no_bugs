package main

import (
	"debug/dwarf"
	"debug/elf"
	"io"
	"log"
)

type DebugContext struct {
	dwarfinfo    *dwarf.Data
	lines        []*dwarf.LineEntry
	followed_sym []*dwarf.Entry
	breakpoints  []uint64

	current_file  string
	current_line  int
	current_instr string
	current_pc    uint64
}

// Initializes the debugger context, and sets a breakpoint at program entry.
func Init(ExePath string) (*DebugContext, error) {

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
	var line *dwarf.LineEntry
	var lines []*dwarf.LineEntry
	var syms []*dwarf.Entry
	var bps []uint64

	var ctx = &DebugContext{
		dwarfinfo:    dwarfinfo,
		lines:        lines,
		followed_sym: syms,
		breakpoints:  bps,

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

			entryPC := entry.AttrField(dwarf.AttrLowpc).Val.(uint64)
			filename := entry.AttrField(dwarf.AttrName).Val.(string)
			path := entry.AttrField(dwarf.AttrCompDir).Val.(string)

			ctx.current_file = path + "/" + filename

			ctx.breakpoints = append(ctx.breakpoints, entryPC)

			for {
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
