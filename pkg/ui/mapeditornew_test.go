package ui

import (
	"image"
	"strings"
	"testing"
)

// unitEdits is a host that adds a unit from nothing and records the call.
type unitEdits struct {
	reportEdits
	cells   []image.Point
	classes []int64
}

func (p *unitEdits) AddUnit(cell image.Point, class int64) (*InspectionDocument, error) {
	p.cells, p.classes = append(p.cells, cell), append(p.classes, class)
	return nil, nil
}

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

func TestUnitKeyNeedsAHostThatAddsUnits(t *testing.T) {
	plain := reportEditor(t, &reportEdits{})
	_ = plain.Key("unit")
	if plain.pick != "" {
		t.Fatalf("a host without AddUnit armed %q", plain.pick)
	}
	host := &unitEdits{}
	e := reportEditor(t, host)
	_ = e.Key("unit")
	if e.pick != "unit" {
		t.Fatalf("U armed %q", e.pick)
	}
	e.pickAt(image.Pt(300, 300))
	if e.prompt.kind != promptNewUnit {
		t.Fatalf("the click did not open the class prompt: prompt %v", e.prompt.kind)
	}
	if fields := e.promptFields(); len(fields) != 1 || fields[0].Key != "class" {
		t.Fatalf("prompt fields %+v", fields)
	}
	cell := e.prompt.cell
	e.prompt.path = "27"
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if len(host.cells) != 1 || host.cells[0] != cell || host.classes[0] != 27 || e.prompt.open() {
		t.Fatalf("AddUnit calls %v %v, prompt open %v", host.cells, host.classes, e.prompt.open())
	}
}

// The status strip joins a screen's help rows; each row stays short enough
// that a screen's rows fit the strip of the smallest window.
func TestEveryHelpRowFitsTheStatusStrip(t *testing.T) {
	e := reportEditor(t, &unitEdits{})
	for _, row := range EditorHelpLines() {
		if got := editorClip(e.font, row, 300); got != row {
			t.Errorf("%q is clipped to %q; a help row holds 300", row, got)
		}
	}
}

// tierEdits is a unit host whose class 70 has three tiers, default 2.
type tierEdits struct {
	unitEdits
	tiers []int64
}

func (p *tierEdits) UnitTiers(class int64) ([]InspectionChoice, int64) {
	if class != 70 {
		return nil, 0
	}
	return []InspectionChoice{{1, "1: Bear1"}, {2, "2: Bear2"}, {3, "3: Bear3"}}, 2
}

func (p *tierEdits) UnitTierLabel() string { return "Tier for the new unit" }

func (p *tierEdits) AddUnitTier(cell image.Point, class, tier int64) (*InspectionDocument, error) {
	p.cells, p.classes, p.tiers = append(p.cells, cell), append(p.classes, class), append(p.tiers, tier)
	return nil, nil
}

// A class with several tiers asks for the tier after the class, starting at
// the host's default and stepping with Down; one with a single tier adds at
// once.
func TestNewUnitAsksTheTierOfAMultiTierClass(t *testing.T) {
	host := &tierEdits{}
	e := reportEditor(t, host)
	_ = e.Key("unit")
	e.pickAt(image.Pt(300, 300))
	cell := e.prompt.cell
	e.prompt.path = "70"
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if e.prompt.kind != promptNewUnitTier || e.prompt.path != "2" || e.prompt.cell != cell || len(host.classes) != 0 {
		t.Fatalf("after the class: prompt %v text %q, adds %v", e.prompt.kind, e.prompt.path, host.classes)
	}
	if fields := e.promptFields(); len(fields) != 1 || fields[0].Key != "tier" || len(fields[0].Choices) != 3 {
		t.Fatalf("tier prompt fields %+v", fields)
	}
	_ = e.Key("down")
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if len(host.tiers) != 1 || host.tiers[0] != 3 || host.classes[0] != 70 || host.cells[0] != cell || e.prompt.open() {
		t.Fatalf("AddUnitTier calls %v %v %v, prompt open %v", host.cells, host.classes, host.tiers, e.prompt.open())
	}
	_ = e.Key("unit")
	e.pickAt(image.Pt(300, 300))
	e.prompt.path = "27"
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if len(host.classes) != 2 || host.classes[1] != 27 || len(host.tiers) != 1 || e.prompt.open() {
		t.Fatalf("a single-tier class: adds %v tiers %v, prompt open %v", host.classes, host.tiers, e.prompt.open())
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
