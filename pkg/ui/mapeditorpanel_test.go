package ui

import "testing"

// A report that lists more rows than the active brush's palette has is drawn
// as the report: the height palette holds two rows, the report three.
func TestMapEditorPanelDrawsAReportLongerThanTheBrushPalette(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	for _, k := range []string{"height", "report"} {
		if err := e.Key(k); err != nil {
			t.Fatal(err)
		}
	}
	if rows, open := e.ReportOpen(); !open || len(rows) <= len(e.brushRows()) || !e.brushActive() {
		t.Fatalf("setup: report open %v with %d rows, brush active %v with %d rows", open, len(rows), e.brushActive(), len(e.brushRows()))
	}
	if img := e.PanelImage(768); img == nil {
		t.Fatal("no panel")
	}
}
