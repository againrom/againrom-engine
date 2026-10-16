package ui

func (e *MapEditor) helpStep() int {
	if e.font != nil {
		return max(18, e.font.Height()+2)
	}
	return 18
}

func (e *MapEditor) saveTop(h int) int { return h - max(21, e.helpStep()+2) }

func (e *MapEditor) helpTop(h int) int {
	return e.saveTop(h) - 6 - len(e.helpLines())*e.helpStep()
}
