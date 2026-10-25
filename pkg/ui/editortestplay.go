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

// keyTestPlay plays the open map in the game. The editor stays open; the map,
// its saved file and its history are unchanged. A host refusal comes before
// any notice.
func (e *MapEditor) keyTestPlay(string) error {
	if !e.editable() {
		return nil
	}
	e.leave(leaveDrag | leavePick | leavePrompt)
	if g, ok := e.doc.Edits.(InspectionTestPlayGate); ok && g.TestPlayRefusal() != nil {
		return e.testPlay()
	}
	if e.askNotice(noticeTestPlay) {
		return nil
	}
	return e.testPlay()
}

// testPlay starts the test play; a refusal shows the report.
func (e *MapEditor) testPlay() error {
	p, ok := e.doc.Edits.(InspectionTestPlay)
	if !ok {
		e.message = "Test play is not available for this map"
		return nil
	}
	line, err := p.TestPlay()
	if err != nil {
		if r, ok := refusal(err); ok {
			errs, warnings := CountFindings(r.Findings)
			e.openReport(r.Findings, fmt.Sprintf(e.word("editor.testplay.refused"), errs, warnings), nil)
			e.selectFinding(0)
			return err
		}
		e.message = err.Error()
		return err
	}
	e.message = line
	return nil
}
