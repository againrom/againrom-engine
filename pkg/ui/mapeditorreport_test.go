package ui

import (
	"image"
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
)

// reportEdits is a host with a report: Save As is refused while refuse is set,
// carrying the findings; else it saves.
type reportEdits struct {
	noUnitAuthoring
	rows   []InspectionFinding
	refuse bool
	saves  int
	dirty  bool
}

func (p *reportEdits) Dirty() bool                        { return p.dirty }
func (p *reportEdits) CanUndo() bool                      { return false }
func (p *reportEdits) CanRedo() bool                      { return false }
func (p *reportEdits) Undo() (*InspectionDocument, error) { return nil, nil }
func (p *reportEdits) Redo() (*InspectionDocument, error) { return nil, nil }
func (p *reportEdits) Report() []InspectionFinding        { return p.rows }
func (p *reportEdits) SaveAs(path string) (string, error) {
	p.saves++
	if p.refuse {
		return "", &ReportRefusal{Findings: p.rows}
	}
	p.dirty = false
	return path, nil
}

func (p *reportEdits) Move(string, int, image.Point) (*InspectionDocument, error) { return nil, nil }

func reportEditor(t *testing.T, edits InspectionEdits) *MapEditor {
	t.Helper()
	open := func(source string) (*InspectionDocument, error) {
		v, err := NewViewer("editor", terrain.Grid{Width: 50, Height: 50, Tiles: make([]uint16, 2500)}, &terrain.Tileset{})
		if err != nil {
			return nil, err
		}
		d := &InspectionDocument{Viewer: v, Source: source, Width: 50, Height: 50, SourceBytes: []byte("map"), Edits: edits}
		for i := 0; i < 6; i++ {
			d.Records = append(d.Records, InspectionRecord{Kind: "Unit", Index: i, Section: 6, Cell: image.Pt(5+6*i, 30), Spatial: true, Size: image.Pt(1, 1)})
		}
		d.Records = append(d.Records, InspectionRecord{Kind: "Condition", Index: 2, Section: 7}, InspectionRecord{Kind: "Action", Index: 0, Section: 7})
		return d, nil
	}
	e := NewMapEditor(panelFont(), nil, open)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	e.Layout(800, 600)
	return e
}

var reportRows = []InspectionFinding{
	{Error: true, Rule: "script-reference", Kind: "Condition", Index: 2, Text: "condition 3 names unit 77"},
	{Error: true, Rule: "owner-slot", Kind: "Unit", Index: 4, Cell: image.Pt(29, 30), HasCell: true, Text: "owner slot 9"},
	{Error: true, Rule: "terrain-planes", Text: "the installed terrain refuses the map"},
	{Rule: "drop-cell", Kind: "Action", Index: 0, Cell: image.Pt(40, 8), HasCell: true, Text: "the drop cell is blocked"},
}

func TestReportKeyListsTheFindingsAndSelectingOneSelectsItsRecord(t *testing.T) {
	e := reportEditor(t, &reportEdits{rows: reportRows})
	if _, open := e.ReportOpen(); open {
		t.Fatal("a report is open before the key")
	}
	_ = e.Key("report")
	rows, open := e.ReportOpen()
	if !open || len(rows) != 4 || e.ReportSelected() != 0 {
		t.Fatalf("open %v rows %d selected %d", open, len(rows), e.ReportSelected())
	}
	record := func() (string, int) {
		if e.Selected() < 0 {
			return "", -1
		}
		r := e.doc.Records[e.Selected()]
		return r.Kind, r.Index
	}
	if k, i := record(); k != "Condition" || i != 2 {
		t.Fatalf("first finding selected %s %d", k, i)
	}
	cam := e.doc.Viewer.Camera()
	before := cam.X
	_ = e.Key("down")
	if k, i := record(); k != "Unit" || i != 4 || e.ReportSelected() != 1 {
		t.Fatalf("second finding selected %s %d", k, i)
	}
	if cam.X == before {
		t.Fatal("selecting a finding did not scroll the map to its unit")
	}
	// A finding that names no record selects none.
	_ = e.Key("down")
	if k, _ := record(); k != "" {
		t.Fatalf("a map-wide finding selected %s", k)
	}
	// A finding that names a cell and a record without one scrolls to the cell.
	cam.X = 0
	_ = e.Key("down")
	if k, i := record(); k != "Action" || i != 0 {
		t.Fatalf("fourth finding selected %s %d", k, i)
	}
	if cam.X == 0 {
		t.Fatal("the drop cell was not scrolled to")
	}
	_ = e.Key("down")
	if e.ReportSelected() != 3 {
		t.Fatal("Down passed the last finding")
	}
	// Clicking a row of the list popup selects it.
	chromeClick(e, listRowPoint(t, e, 1))
	if e.ReportSelected() != 1 {
		t.Fatalf("a click on row 1 selected %d", e.ReportSelected())
	}
	if pic := e.ChromeImage(); pic == nil {
		t.Fatal("no chrome")
	}
	if n := len(e.markerLines()); n == 0 {
		t.Fatal("flagged records draw no marker")
	}
	_ = e.Key("escape")
	if _, open := e.ReportOpen(); open {
		t.Fatal("Escape left the report open")
	}
	// R closes what R opened.
	_ = e.Key("report")
	_ = e.Key("report")
	if _, open := e.ReportOpen(); open {
		t.Fatal("R twice left the report open")
	}
}

func TestSaveAsWithAnErrorShowsTheReportAndKeepsItOnScreen(t *testing.T) {
	edits := &reportEdits{rows: reportRows, refuse: true, dirty: true}
	e := reportEditor(t, edits)
	_ = e.Key("saveas")
	e.TextInput("out.alm")
	err := e.Key("enter")
	if _, ok := err.(*ReportRefusal); !ok {
		t.Fatalf("Save As: %v", err)
	}
	rows, open := e.ReportOpen()
	if !open || len(rows) != 4 || e.prompt.open() || e.message != "" || !e.Dirty() || e.doc.Source != "scenario/10.alm" {
		t.Fatalf("open %v rows %d path %v message %q dirty %v source %q", open, len(rows), e.prompt.open(), e.message, e.Dirty(), e.doc.Source)
	}
	if !strings.HasPrefix(e.report.note, "Save refused: 3 errors, 1 warnings") {
		t.Fatalf("note %q", e.report.note)
	}
	_ = e.Key("pagedown")
	if _, open := e.ReportOpen(); !open {
		t.Fatal("scrolling closed the report")
	}
	// The same save of a map without errors goes ahead and lists the warnings.
	edits.refuse, edits.rows = false, reportRows[3:]
	_ = e.Key("saveas")
	e.TextInput("out.alm")
	if err := e.Key("enter"); err != nil || e.Dirty() || e.doc.Source != "out.alm" {
		t.Fatalf("Save As: %v dirty %v source %q", err, e.Dirty(), e.doc.Source)
	}
	if rows, open := e.ReportOpen(); !open || len(rows) != 1 || !strings.HasPrefix(e.report.note, "Saved with 1 warnings") {
		t.Fatalf("open %v rows %d note %q", open, len(rows), e.report.note)
	}
	// A map with no finding saves and shows no report.
	edits.rows = nil
	edits.dirty = true
	_ = e.Key("saveas")
	e.TextInput("clean.alm")
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if _, open := e.ReportOpen(); open {
		t.Fatal("a clean save opened a report")
	}
}

// reportModEdits saves into a mod.
type reportModEdits struct {
	reportEdits
}

func (p *reportModEdits) ModTarget() string { return "x" }
func (p *reportModEdits) SaveIntoMod() (string, error) {
	_, err := p.SaveAs("mods/x/map.alm")
	return "mods/x/map.alm", err
}

func TestSaveIntoAModWithAnErrorShowsTheReport(t *testing.T) {
	edits := &reportModEdits{reportEdits{rows: reportRows, refuse: true, dirty: true}}
	e := reportEditor(t, edits)
	err := e.Key("saveas")
	if _, ok := err.(*ReportRefusal); !ok {
		t.Fatalf("Save: %v", err)
	}
	if rows, open := e.ReportOpen(); !open || len(rows) != 4 || e.message != "" || e.doc.Source != "scenario/10.alm" {
		t.Fatalf("open %v rows %d message %q source %q", open, len(rows), e.message, e.doc.Source)
	}
	edits.refuse, edits.rows = false, reportRows[3:]
	if err := e.Key("saveas"); err != nil || e.doc.Source != "mods/x/map.alm" {
		t.Fatalf("Save: %v source %q", err, e.doc.Source)
	}
	if rows, open := e.ReportOpen(); !open || len(rows) != 1 {
		t.Fatalf("warnings after the save: open %v rows %d", open, len(rows))
	}
}

func TestReportKeyDoesNothingForAHostWithoutAReport(t *testing.T) {
	e := reportEditor(t, &editorTextEdits{})
	_ = e.Key("report")
	if _, open := e.ReportOpen(); open {
		t.Fatal("a host without a report opened one")
	}
}

func TestFindingLineNamesSeverityRuleRecordAndCell(t *testing.T) {
	f := InspectionFinding{Error: true, Rule: "owner-slot", Kind: "Unit", Index: 4, Cell: image.Pt(29, 30), HasCell: true, Text: "owner slot 9"}
	if got := f.Line(); got != "error owner-slot Unit #4 cell 29,30: owner slot 9" {
		t.Fatal(got)
	}
	if got := (InspectionFinding{Rule: "terrain-planes", Text: "x"}).Line(); got != "warning terrain-planes map: x" {
		t.Fatal(got)
	}
}
