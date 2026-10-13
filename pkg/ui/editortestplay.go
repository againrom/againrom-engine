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

// keyTestPlay plays the open map in the game. The editor stays open; the map,
// its saved file and its history are unchanged.
func (e *MapEditor) keyTestPlay(string) error {
	if !e.editable() {
		return nil
	}
	e.leave(leaveDrag | leavePick | leavePrompt)
	p, ok := e.doc.Edits.(InspectionTestPlay)
	if !ok {
		e.message = "Test play is not available for this map"
		return nil
	}
	line, err := p.TestPlay()
	if err != nil {
		if r, ok := refusal(err); ok {
			errs, warnings := CountFindings(r.Findings)
			e.openReport(r.Findings, fmt.Sprintf("Test play refused: %d errors, %d warnings", errs, warnings), nil)
			e.selectFinding(0)
			return err
		}
		e.message = err.Error()
		return err
	}
	e.message = line
	return nil
}
