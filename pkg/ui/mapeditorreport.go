package ui

import (
	"errors"
	"fmt"
	"image"
	"strings"
)

// InspectionFinding is one line of a map report: the rule a record breaks.
// Kind and Index address the record as InspectionRecord does; an empty Kind
// names the whole map. Cell is the cell the finding names, when HasCell.
type InspectionFinding struct {
	Error   bool
	Rule    string
	Kind    string
	Index   int
	Cell    image.Point
	HasCell bool
	Text    string
}

// Line is the finding as one line of text: the window's report rows and the
// command-line check print the same line.
func (f InspectionFinding) Line() string {
	severity := "warning"
	if f.Error {
		severity = "error"
	}
	at := "map"
	if f.Kind != "" {
		at = fmt.Sprintf("%s #%d", f.Kind, f.Index)
	}
	if f.HasCell {
		at += fmt.Sprintf(" cell %d,%d", f.Cell.X, f.Cell.Y)
	}
	return fmt.Sprintf("%s %s %s: %s", severity, f.Rule, at, f.Text)
}

// InspectionChecker is implemented by edits whose host checks the map against
// a rule table. Report answers the findings for the map as it now stands,
// errors first.
type InspectionChecker interface {
	Report() []InspectionFinding
}

// ReportRefusal is the error of an action the report refuses: the map holds at
// least one error finding. Ask is set when the action is a save the user may
// confirm with the errors kept; Why then or otherwise is the host's reason in
// the host's language, empty for the plain text.
type ReportRefusal struct {
	Findings []InspectionFinding
	Ask      bool
	Why      string
}

func (r *ReportRefusal) Error() string {
	e, w := CountFindings(r.Findings)
	if r.Why != "" {
		return fmt.Sprintf(r.Why, e, w)
	}
	return fmt.Sprintf("the map has %d errors and %d warnings; nothing was saved", e, w)
}

// InspectionErrorSave is implemented by edits whose host can write a map the
// report holds errors for. AllowErrors lets the next save write it; the host
// forgets the permission when that save ends.
type InspectionErrorSave interface {
	AllowErrors()
}

// InspectionSaveNote is implemented by edits whose host can finish a save
// without a part of it. SaveNote answers, in the host's words, what the last
// save did not write; it is empty when the save was whole.
type InspectionSaveNote interface {
	SaveNote() string
}

// CountFindings is the number of error and of warning findings.
func CountFindings(f []InspectionFinding) (errs, warnings int) {
	for _, x := range f {
		if x.Error {
			errs++
		} else {
			warnings++
		}
	}
	return errs, warnings
}

// refusal reports the report refusal err carries, if any.
func refusal(err error) (*ReportRefusal, bool) {
	var r *ReportRefusal
	return r, errors.As(err, &r)
}

// reportState is the window's report panel: the findings it lists, the one
// selected and the records they flag.
type reportState struct {
	open    bool
	rows    []InspectionFinding
	note    string
	sel     int
	flagged map[string]bool
	goals   []InspectionGoal // set while the rows are the script panel's arms
	// copies holds, by row, the text Ctrl+C copies when it differs from the
	// row's own text; hint is a line the panel shows under its rows.
	copies []string
	hint   string
	// confirm saves the map with its errors; set while the report is the
	// question a refused save asked, gone with the report.
	confirm func() error
}

func findingKey(kind string, index int) string { return fmt.Sprintf("%s#%d", kind, index) }

// ReportOpen reports whether the report panel is showing, and its findings.
func (e *MapEditor) ReportOpen() ([]InspectionFinding, bool) {
	return e.report.rows, e.report.open
}

// ReportSelected is the index of the selected finding, or -1.
func (e *MapEditor) ReportSelected() int {
	if !e.report.open || len(e.report.rows) == 0 {
		return -1
	}
	return e.report.sel
}

// reportKey is the report key: it checks the map as it now stands and shows
// the findings, or closes the panel when it already shows.
func (e *MapEditor) reportKey() {
	if e.report.open {
		e.leave(leaveReport)
		return
	}
	c, ok := e.checker()
	if !ok || e.catalog {
		return
	}
	e.leave(leaveDrag | leavePick | leavePrompt)
	rows := c.Report()
	errs, warnings := CountFindings(rows)
	e.openReport(rows, fmt.Sprintf("Report: %d errors, %d warnings", errs, warnings), nil)
	e.selectFinding(0)
}

func (e *MapEditor) checker() (InspectionChecker, bool) {
	if !e.editable() {
		return nil, false
	}
	c, ok := e.doc.Edits.(InspectionChecker)
	return c, ok
}

// afterSave shows the warnings of the map just saved.
func (e *MapEditor) afterSave() {
	c, ok := e.checker()
	if !ok {
		return
	}
	if rows := c.Report(); len(rows) > 0 {
		errs, warnings := CountFindings(rows)
		note := fmt.Sprintf("Saved with %d warnings", warnings)
		if errs > 0 {
			note = fmt.Sprintf("Saved with %d errors, %d warnings", errs, warnings)
		}
		e.openReport(rows, note, nil)
	}
}

// noteSave sets the status line after a save: the errors the map was saved
// with, then what the host did not write.
func (e *MapEditor) noteSave(errs int) {
	var parts []string
	if errs > 0 {
		parts = append(parts, fmt.Sprintf(e.word("editor.saved.errors"), errs))
	}
	if n, ok := e.doc.Edits.(InspectionSaveNote); ok && n.SaveNote() != "" {
		parts = append(parts, n.SaveNote())
	}
	e.saveNote, e.saveNoteFor = strings.Join(parts, ". "), e.doc.Source
}

// refuseSave shows the report a refused save carries and keeps it on screen.
func (e *MapEditor) refuseSave(r *ReportRefusal) {
	e.leave(leavePrompt)
	errs, warnings := CountFindings(r.Findings)
	e.openReport(r.Findings, fmt.Sprintf("Save refused: %d errors, %d warnings", errs, warnings), nil)
	e.selectFinding(0)
}

// askSaveWithErrors shows the report of a save the map's errors stopped and
// waits for Enter or S, which run again with the errors allowed; Esc closes
// the report and writes nothing.
func (e *MapEditor) askSaveWithErrors(r *ReportRefusal, again func() error) {
	e.leave(leavePrompt)
	errs, _ := CountFindings(r.Findings)
	e.openReport(r.Findings, fmt.Sprintf(e.word("editor.save.errors"), errs), nil)
	e.selectFinding(0)
	e.report.confirm = func() error {
		if a, ok := e.doc.Edits.(InspectionErrorSave); ok {
			a.AllowErrors()
		}
		e.saveErrors = errs
		return again()
	}
}

// selectFinding selects the record finding i names and scrolls the map to it;
// a finding that names a cell and no spatial record scrolls to the cell.
func (e *MapEditor) selectFinding(i int) {
	if i < 0 || i >= len(e.report.rows) {
		return
	}
	e.report.sel = i
	if e.doc == nil {
		return
	}
	f := e.report.rows[i]
	e.selected, e.focusedReference = -1, 0
	jumped := false
	if f.Kind != "" {
		for n, r := range e.doc.Records {
			if r.Kind == f.Kind && r.Index == f.Index {
				e.Select(n, true)
				jumped = r.Spatial
				break
			}
		}
	}
	if !jumped && f.HasCell {
		e.centerOn(f.Cell)
	}
}

// selectedFinding is the selected finding of the open report.
func (e *MapEditor) selectedFinding() (InspectionFinding, bool) {
	if !e.report.open || e.report.sel < 0 || e.report.sel >= len(e.report.rows) {
		return InspectionFinding{}, false
	}
	return e.report.rows[e.report.sel], true
}

// reportCopyText is the text Ctrl+C copies from report row n: the value the
// row was opened with, a script arm's list text, a key list row's own text, or
// a finding's line.
func (e *MapEditor) reportCopyText(n int) string {
	switch {
	case n < 0 || n >= len(e.report.rows):
		return ""
	case n < len(e.report.copies) && e.report.copies[n] != "":
		return e.report.copies[n]
	case n < len(e.report.goals):
		return e.goalText(e.report.goals[n])
	case e.report.rows[n].Rule == keyListRule:
		return e.report.rows[n].Text
	}
	return e.report.rows[n].Line()
}

// keyCopy is Ctrl+C: with a report panel showing it copies the selected row to
// the text clipboard, unless a group of records is selected; otherwise it
// copies the map selection.
func (e *MapEditor) keyCopy(key string) error {
	if e.listKind() != "report" || e.selectionShown() || e.report.sel >= len(e.report.rows) {
		return e.selectionKey(key)
	}
	if err := textClipboard.write(e.reportCopyText(e.report.sel)); err != nil {
		e.message = err.Error()
		return err
	}
	return nil
}

// reportStep moves the selected finding by delta, as the arrow keys do.
func (e *MapEditor) reportStep(delta int) {
	if n := len(e.report.rows); e.report.open && n > 0 {
		e.selectFinding(max(0, min(n-1, e.report.sel+delta)))
	}
}

// keyListRule is the rule of the F1 key list's rows, which the report window
// shows; each row is one help line.
const keyListRule = "keys"

// reportLabel is the finding as one list row; a key list row is its line.
func reportLabel(f InspectionFinding) string {
	if f.Rule == keyListRule {
		return f.Text
	}
	mark := "W"
	if f.Error {
		mark = "E"
	}
	if f.Kind == "" {
		return fmt.Sprintf("%s %s map", mark, f.Rule)
	}
	return fmt.Sprintf("%s %s %s #%d", mark, f.Rule, f.Kind, f.Index)
}

// reportMarkers are the cell marks of the selected finding that names a cell.
func (e *MapEditor) reportCrosses() []image.Point {
	if !e.report.open || e.report.sel >= len(e.report.rows) {
		return nil
	}
	if f := e.report.rows[e.report.sel]; f.HasCell {
		return []image.Point{f.Cell}
	}
	return nil
}
