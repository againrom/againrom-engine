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

// fileStoreFirstFrame is a mission file without a complete document: the
// state store's application leaves, read at LOAD with the actor IDs of the
// import's source bindings (readFileApplication). The store holds no engine
// action leaf, so it supplies no current view and no animation; its fog
// plane is the opener's, and the first frame adds none.
func fileStoreFirstFrame() *firstFrameSource {
	return &firstFrameSource{
		application: func(mw *mapWorld) (*fileApplication, []uint16, error) {
			return mw.mission.state.fileApplication, mw.spells().BookRows, nil
		},
	}
}

// documentFirstFrame is an original save's decoded document: its
// application leaves with the actor IDs of the document's actor table, its
// current view and animation, and its fog plane over the opener's.
func documentFirstFrame() *firstFrameSource {
	return &firstFrameSource{
		application: documentApplication,
		current: func(mw *mapWorld) (*currentActionData, error) {
			return readCurrentActions(mw.mission.state.savedDocument.Document)
		},
		fog: func(mw *mapWorld, cells int) ([]byte, error) {
			return originalFogPlane(mw.mission.state.savedDocument.Document, cells)
		},
		animation: true,
	}
}

func documentApplication(mw *mapWorld) (*fileApplication, []uint16, error) {
	document := mw.mission.state.savedDocument
	if document == nil || document.Document == nil {
		return nil, nil, nil
	}
	raw, err := readOriginalApplicationState(document.Document)
	if err != nil {
		return nil, nil, err
	}
	ids, err := applicationActorIDs(document.Document, document.Actors, mw.world)
	if err != nil {
		return nil, nil, err
	}
	return &fileApplication{raw: raw, ids: ids, structures: applicationStructureIDs(mw.world)}, originalBookIDs, nil
}

// firstFrameSourceOf is the source of a mission's first frame: the decoded
// document of an original save, else a mission file's state store; nil
// where the mission has neither.
func firstFrameSourceOf(ms *Mission, original bool) *firstFrameSource {
	switch {
	case ms == nil:
		return nil
	case original:
		return documentFirstFrame()
	case ms.fileApplication != nil:
		return fileStoreFirstFrame()
	}
	return nil
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
