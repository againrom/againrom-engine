package ui

import (
	"errors"
	"testing"
)

// testPlayEdits is a host whose test play answers line or err.
type testPlayEdits struct {
	editorTextEdits
	plays int
	line  string
	err   error
}

func (p *testPlayEdits) TestPlay() (string, error) {
	p.plays++
	return p.line, p.err
}

func testPlayEditor(t *testing.T, edits InspectionEdits) *MapEditor {
	t.Helper()
	doc := &InspectionDocument{Viewer: litViewer(t), Source: "scenario/20.alm", Edits: edits}
	e := NewMapEditor(nil, nil, func(string) (*InspectionDocument, error) { return doc, nil })
	if err := e.Open("scenario/20.alm"); err != nil {
		t.Fatal(err)
	}
	return e
}

func TestTheTestPlayKeyPlaysTheMapAndKeepsTheWindow(t *testing.T) {
	edits := &testPlayEdits{line: "Test play: againrom.exe opens mission 21"}
	e := testPlayEditor(t, edits)
	if err := e.Key("testplay"); err != nil {
		t.Fatal(err)
	}
	if edits.plays != 1 || e.message != edits.line || e.doc == nil || e.doc.Source != "scenario/20.alm" {
		t.Fatalf("plays %d message %q", edits.plays, e.message)
	}
	edits.err = errors.New("Test play: the game program x is not beside the map editor")
	if err := e.Key("testplay"); err == nil || e.message != edits.err.Error() {
		t.Fatalf("failure: %v, message %q", err, e.message)
	}
	if !containsString(EditorHelpLines(), "P: players / F5: test play") {
		t.Fatal("the help does not name the test play key")
	}
}

func TestATestPlayTheReportRefusesShowsTheReport(t *testing.T) {
	findings := []InspectionFinding{{Error: true, Rule: "record-outside", Text: "outside"}, {Rule: "drop-cell", Text: "no drop"}}
	edits := &testPlayEdits{err: &ReportRefusal{Findings: findings}}
	e := testPlayEditor(t, edits)
	if err := e.Key("testplay"); err == nil {
		t.Fatal("a refused test play answered no error")
	}
	rows, open := e.ReportOpen()
	if !open || len(rows) != 2 || e.report.note != "Test play refused: 1 errors, 1 warnings" || e.message != "" {
		t.Fatalf("report open %v rows %d note %q message %q", open, len(rows), e.report.note, e.message)
	}
	// Loss control: a host without a test play names that and plays nothing.
	e = testPlayEditor(t, &editorTextEdits{})
	if err := e.Key("testplay"); err != nil || e.message != "Test play is not available for this map" {
		t.Fatalf("no test play: %v, message %q", err, e.message)
	}
}
