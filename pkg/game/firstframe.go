package game

// firstFrameSource is the source a mission's first frame restores its
// application state from: an original save's decoded document, or a mission
// file's state store. What a source supplies is data of it; a nil reader is
// a part that source does not restore.
type firstFrameSource struct {
	// application is the saved application state with the runtime ID of each
	// actor and structure its selection may name, and the book rows its
	// pressed index counts in; a nil state is no first frame.
	application func(mw *mapWorld) (*fileApplication, []uint16, error)
	// current is the engine action leaf whose view overlays the state.
	current func(mw *mapWorld) (*currentActionData, error)
	// fog is the explored plane the first frame installs over the one the
	// opener installed from the file (readOriginalFog).
	fog func(mw *mapWorld, cells int) ([]byte, error)
	// animation restores the animation clock current states.
	animation bool
}

// firstFrame is the source's application state over the viewer's current
// state; nil where the source holds none.
func (src *firstFrameSource) firstFrame(mw *mapWorld) (*SnapshotApplicationState, error) {
	app, rows, err := src.application(mw)
	if err != nil || app == nil {
		return nil, err
	}
	s, err := app.view(mw.view.SaveApplication(), rows)
	if err != nil {
		return nil, err
	}
	if src.current != nil {
		actions, err := src.current(mw)
		if err != nil {
			return nil, err
		}
		applyCurrentView(&s.View, app.raw, actions)
		s.Baseline = s.View
	}
	return s, nil
}
