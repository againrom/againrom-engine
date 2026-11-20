package ui

import "fmt"

// InspectionTestPlay is implemented by edits whose host can play the map in
// the game. TestPlay writes the map as it now stands, unsaved edits included,
// where the game reads it and starts the game; it answers the line the window
// shows. A *ReportRefusal error means the report holds an error and nothing
// was written or started.
type InspectionTestPlay interface {
	TestPlay() (string, error)
}

// InspectionTestPlayGate is implemented by edits whose host can refuse every
// test play of the open map before it looks at the map; the answer is the
// refusal, nil when test play is open to the map.
type InspectionTestPlayGate interface {
	TestPlayRefusal() error
}

// applyUnitDraft writes the unit window's draft as Apply does.
func (e *MapEditor) applyUnitDraft() {
	if e.unitWindowShown() && e.objectDirty() {
		e.applyObject()
	}
}

// saveUnitDraft applies the draft before a save or test play; it answers the
// window's refusal.
func (e *MapEditor) saveUnitDraft() error {
	e.applyUnitDraft()
	if e.unitWindowShown() && e.objectDirty() {
		return fmt.Errorf("%s", e.obj.note)
	}
	return nil
}

// keyTestPlay plays the open map in the game. The editor stays open; the map,
// its saved file and its history are unchanged. A host refusal comes before
// any window; where the host offers characters the party window asks who
// plays and holds the notice's lines, else the notice asks on its own.
func (e *MapEditor) keyTestPlay(string) error {
	if !e.editable() {
		return nil
	}
	if err := e.saveUnitDraft(); err != nil {
		return err
	}
	e.leave(leaveDrag | leavePick | leavePrompt)
	if g, ok := Seam[InspectionTestPlayGate](e.doc.Edits); ok && g.TestPlayRefusal() != nil {
		return e.testPlay()
	}
	if e.askTestParty() || e.askNotice(noticeTestPlay) {
		return nil
	}
	return e.testPlay()
}

// testPlay starts the test play; a refusal shows the report.
func (e *MapEditor) testPlay() error {
	p, ok := Seam[InspectionTestPlay](e.doc.Edits)
	if !ok {
		e.message = e.plainWord("editor.testplay.unavailable")
		return nil
	}
	return e.testPlayBy(p.TestPlay)
}

// testPlayBy runs the host's test play and shows its line, or the report
// when it refuses.
func (e *MapEditor) testPlayBy(play func() (string, error)) error {
	line, err := play()
	if err != nil {
		if r, ok := refusal(err); ok {
			errs, warnings := CountFindings(r.Findings)
			e.openReportAtFirstFinding(r.Findings, fmt.Sprintf(e.word("editor.testplay.refused"), errs, warnings))
			return err
		}
		e.message = err.Error()
		return err
	}
	e.message = line
	return nil
}
