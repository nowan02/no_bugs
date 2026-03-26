package symbol

func (Context *DebugContext) LookForLineNo() {
	// Only one file will be supported for now!
	for _, line := range Context.Lines {
		if Context.TextareaBegin+line.Address == Context.Target.Regs.Rip {
			Context.CurrentLine = line.Line
			break
		}
	}
}
