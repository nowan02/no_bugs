package symbol

func (Context *DebugContext) LookForLineNo() {
	// Only one file will be supported for now!
	for _, line := range Context.Lines {
		if line.Address >= Context.Target.Regs.Rip && line.Address > Context.TextareaBegin && line.Address < Context.TextareaEnd {
			Context.CurrentLine = line.Line
		}
	}
}
