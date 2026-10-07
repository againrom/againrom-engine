package ui

func (a *App) SetMapEntryObserver(entered func(*Viewer)) {
	if a == nil || a.flow == nil {
		return
	}
	a.mapEntry = entered
	a.mapEntryViewer = a.flow.viewer
}
