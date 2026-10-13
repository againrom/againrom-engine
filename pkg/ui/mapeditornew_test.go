package ui

import (
	"image"
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
	e.catalog, e.offset = false, 3
	_ = e.Key("report")
	if err := e.Key("new"); err != nil {
		t.Fatal(err)
	}
	if !e.catalog || e.offset != 0 {
		t.Fatalf("catalogue %v offset %d after N", e.catalog, e.offset)
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
	if !e.pathMode || !e.fieldMode || e.sackCell == nil || !e.newUnit {
		t.Fatalf("the click did not open the class prompt: path %v field %v cell %v unit %v", e.pathMode, e.fieldMode, e.sackCell, e.newUnit)
	}
	if fields := e.promptFields(); len(fields) != 1 || fields[0].Key != "class" {
		t.Fatalf("prompt fields %+v", fields)
	}
	cell := *e.sackCell
	e.path = "27"
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if len(host.cells) != 1 || host.cells[0] != cell || host.classes[0] != 27 || e.pathMode {
		t.Fatalf("AddUnit calls %v %v, prompt open %v", host.cells, host.classes, e.pathMode)
	}
}

func TestEveryHelpRowFitsTheRail(t *testing.T) {
	e := reportEditor(t, &unitEdits{})
	for _, row := range EditorHelpLines() {
		if got := editorClip(e.font, row, 300); got != row {
			t.Errorf("%q is clipped to %q; the rail holds 300", row, got)
		}
	}
}
