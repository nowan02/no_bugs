package symbol

func (ctx *DebugContext) LookForLineNo() {
	// Only one file will be supported for now!
	for _, line := range ctx.Lines {
		if ctx.TextareaBegin+line.Address == ctx.Target.Regs.Rip {
			ctx.Logger.Println("Currently on source line ", line.Line)
			ctx.CurrentLine = line.Line
			return
		}
	}
	ctx.Logger.Fatalln("FATAL: Could not current determine line number, instruction pointer might be point outside of text area.")
}

// Checks if source line number can be a valid breakpoint.
// If it is, returns the offset of that line.
// If not, return 0.
func (ctx *DebugContext) IsValidBreakpoint(lineno int) uint64 {
	for _, line := range ctx.Lines {
		if line.Line == lineno && line.IsStmt {
			ctx.Logger.Println("Line ", lineno, " can be a valid breakpoint.")
			return line.Address
		}
	}
	ctx.Logger.Println("Line ", lineno, " is not a valid breakpoint.")
	return 0
}
