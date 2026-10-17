package ui

import (
	"strings"
	"testing"
)

func TestNewKeyOpensTheCatalogueAtItsFirstRow(t *testing.T) {
	e := reportEditor(t, &reportEdits{})
	e.catalog = false
	_ = e.Key("report")
	if err := e.Key("new"); err != nil {
		t.Fatal(err)
	}
	e.listSpec()
	if top, _ := e.model.Visible(); !e.catalog || top != 0 || e.model.Len() > 1 && e.model.Selection() != 1 {
		t.Fatalf("catalogue %v top %d selection %d after N", e.catalog, top, e.model.Selection())
	}
	if _, open := e.ReportOpen(); open {
		t.Fatal("the report stayed open")
	}
}

// The status strip joins a screen's help rows; each row stays short enough
// that a screen's rows fit the strip of the smallest window.
func TestEveryHelpRowFitsTheStatusStrip(t *testing.T) {
	e := reportEditor(t, &reportEdits{})
	for _, row := range EditorHelpLines() {
		if got := editorClip(e.font, row, 300); got != row {
			t.Errorf("%q is clipped to %q; a help row holds 300", row, got)
		}
	}
}

// The status strip names the record on the hovered cell by its hover line;
// a heading line of a record becomes a popup heading row.
func TestHoverLineAndHeadingLine(t *testing.T) {
	e := reportEditor(t, &reportEdits{})
	e.doc.Records[1].Hover = "Bear2  tier 2  health 40/40"
	e.doc.Records[1].Lines = []InspectionLine{{Text: "Characteristics", Role: "heading"}, {Text: "Health 40/40"}}
	e.hoverCell, e.hoverOK = e.doc.Records[1].Cell, true
	if s := e.statusText(); !strings.Contains(s, "Bear2  tier 2  health 40/40") {
		t.Fatalf("status %q", s)
	}
	e.hoverCell = e.doc.Records[2].Cell
	if s := e.statusText(); strings.Contains(s, "Bear2") {
		t.Fatalf("status %q names a record off the hovered cell", s)
	}
	e.Select(1, false)
	s, _, ok := e.objectSpec()
	if !ok {
		t.Fatal("no popup")
	}
	found := false
	for _, r := range s.Rows {
		found = found || r.Kind == popupHeading && r.Label == "Characteristics"
	}
	if !found {
		t.Fatal("the heading line is not a heading row")
	}
}
