package ui

// HeadlessEntityPoint locates a visible hit; it sends no event and changes no
// selection. A witness still activates the production pointer handler itself.
func (a *App) HeadlessEntityPoint(id uint32) (int, int, error) {
	v, err := a.headlessViewer()
	if err != nil {
		return 0, 0, err
	}
	return v.InspectionPoint(InspectionSubject{InspectionUnit, id})
}
