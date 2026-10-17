package ui

// unitKey opens the Add window on a new unit.
func (e *MapEditor) unitKey() {
	if e.catalog {
		return
	}
	e.openAdd("Unit")
}
