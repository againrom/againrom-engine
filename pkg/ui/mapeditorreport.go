package ui

import (
	"errors"
	"fmt"
	"image"
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

// ReportRefusal is the error of a save the report refuses: the map holds at
// least one error finding.
type ReportRefusal struct{ Findings []InspectionFinding }

func (r *ReportRefusal) Error() string {
	e, w := CountFindings(r.Findings)
	return fmt.Sprintf("the map has %d errors and %d warnings; nothing was saved", e, w)
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
// selected, and the list offset the record list had before it opened.
type reportState struct {
	open    bool
	rows    []InspectionFinding
	note    string
	sel     int
	flagged map[string]bool
	offset  int
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

// openReport shows rows under a note, leaving the record list's offset to be
// restored when the panel closes.
func (e *MapEditor) openReport(rows []InspectionFinding, note string) {
	if !e.report.open {
		e.report.offset = e.offset
	}
	e.report.open, e.report.rows, e.report.note, e.report.sel = true, rows, note, 0
	e.report.flagged = map[string]bool{}
	for _, f := range rows {
		if f.Kind != "" {
			e.report.flagged[findingKey(f.Kind, f.Index)] = true
		}
	}
	e.catalog, e.offset, e.listWheel = false, 0, 0
}

func (e *MapEditor) closeReport() {
	if !e.report.open {
		return
	}
	e.offset = e.report.offset
	e.report = reportState{}
	e.listWheel = 0
}

// reportKey is the report key: it checks the map as it now stands and shows
// the findings, or closes the panel when it already shows.
func (e *MapEditor) reportKey() {
	if e.report.open {
		e.closeReport()
		return
	}
	c, ok := e.checker()
	if !ok || e.catalog {
		return
	}
	e.cancelRecordDrag()
	e.pick, e.fieldMode, e.pathMode = "", false, false
	rows := c.Report()
	errs, warnings := CountFindings(rows)
	e.openReport(rows, fmt.Sprintf("Report: %d errors, %d warnings", errs, warnings))
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
		_, warnings := CountFindings(rows)
		e.openReport(rows, fmt.Sprintf("Saved with %d warnings", warnings))
	}
}

// refuseSave shows the report a refused save carries and keeps it on screen.
func (e *MapEditor) refuseSave(r *ReportRefusal) {
	e.pathMode, e.pathSave, e.fieldMode = false, false, false
	errs, warnings := CountFindings(r.Findings)
	e.openReport(r.Findings, fmt.Sprintf("Save refused: %d errors, %d warnings", errs, warnings))
	e.selectFinding(0)
}

// selectFinding selects the record finding i names and scrolls the map to it;
// a finding that names a cell and no spatial record scrolls to the cell.
func (e *MapEditor) selectFinding(i int) {
	if e.doc == nil || i < 0 || i >= len(e.report.rows) {
		return
	}
	e.report.sel = i
	f := e.report.rows[i]
	listOffset := e.offset
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
	e.offset = listOffset
	if !jumped && f.HasCell {
		e.centerOn(f.Cell)
	}
	if rows := e.visibleRows(); i < e.offset {
		e.offset = i
	} else if i >= e.offset+rows {
		e.offset = i - rows + 1
	}
}

// reportStep moves the selected finding by delta, as the arrow keys do.
func (e *MapEditor) reportStep(delta int) {
	if n := len(e.report.rows); e.report.open && n > 0 {
		e.selectFinding(max(0, min(n-1, e.report.sel+delta)))
	}
}

// reportLabel is the finding as one list row.
func reportLabel(f InspectionFinding) string {
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
