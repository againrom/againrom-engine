package game

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/mapedit"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// openIdentifiedUnits opens a map of two units with script identities, so
// each can own a stock.
func openIdentifiedUnits(t *testing.T) *ui.MapEditor {
	t.Helper()
	b := synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12), Units: []synth.ALMUnit{
		{X: 3*256 + 128, Y: 4*256 + 77, ClassID: 40, ClassSubID: 2, UnitID: 11},
		{X: 7*256 + 3, Y: 8*256 + 5, ClassID: 40, ClassSubID: 2, UnitID: 12},
	}})
	table := authoringTable()
	table.Names = data.ItemNames{0x0101: "first", 0x0202: "second"}
	x := &MapInspector{tiles: &terrain.Tileset{}, table: table}
	path := filepath.Join(t.TempDir(), "source.alm")
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	e.Document().Viewer.SetFlat(true)
	return e
}

// stockedAuthoring is the authoring map with both its units given a stock
// holding a different item.
func stockedAuthoring(t *testing.T) *ui.MapEditor {
	t.Helper()
	e := openIdentifiedUnits(t)
	for i, item := range []int64{0x0101, 0x0202} {
		e.Select(recordIndex(t, e, "Unit", i), false)
		if err := e.Key("stock"); err != nil {
			t.Fatal(err)
		}
		recordFieldInput(t, e, "additem", item)
	}
	return e
}

func stockRows(e *ui.MapEditor) []ui.InspectionRecord {
	var out []ui.InspectionRecord
	for _, r := range e.Document().Records {
		if r.Kind == ui.KindStock {
			out = append(out, r)
		}
	}
	return out
}

func TestUnitDeletionRemovesItsStockFromTheDocument(t *testing.T) {
	e := stockedAuthoring(t)
	host := e.Document().Edits.(*mapInspectionEdits)
	before := stockRows(e)
	if len(before) != 2 {
		t.Fatalf("%d stocks", len(before))
	}
	raw := bytes.Clone(e.Document().SourceBytes)
	d, err := host.Delete("Unit", 0)
	if err != nil || d == nil {
		t.Fatal("unit deletion refused", err)
	}
	var left []ui.InspectionRecord
	for _, r := range d.Records {
		if r.Kind == ui.KindStock {
			left = append(left, r)
		}
	}
	if len(left) != 1 || host.model.Count(mapedit.KindLoot) != 1 || host.model.UnitCount() != 1 {
		t.Fatalf("the document lists %d stocks after the delete", len(left))
	}
	if left[0].Index != 0 || left[0].Label != before[1].Label || !reflect.DeepEqual(left[0].Details, before[1].Details) {
		t.Fatalf("the surviving stock is not the second one: %+v", left[0])
	}
	if _, err := host.Undo(); err != nil || !bytes.Equal(host.model.Bytes(), raw) {
		t.Fatal("one Undo did not restore the unit and its stock", err)
	}
}

// A stock selected by one command stays the same stock when another command
// deletes a unit and the stock's index falls.
func TestUnitDeletionKeepsAnotherSelectedStock(t *testing.T) {
	e := stockedAuthoring(t)
	before := stockRows(e)
	e.Select(recordIndex(t, e, "Stock", 1), false)
	_, must := editorMCP(t, e)
	must("delete_record", map[string]any{"kind": "Unit", "index": 0})
	left := stockRows(e)
	if len(left) != 1 || left[0].Label != before[1].Label {
		t.Fatalf("stock rows after the delete: %+v", left)
	}
	if sel := e.Selected(); sel < 0 || e.Document().Records[sel].Kind != ui.KindStock || e.Document().Records[sel].Label != before[1].Label {
		t.Fatalf("the selection is %d, not the second stock", sel)
	}
}

func TestUnitDeletionClearsASelectedStockOfThatUnit(t *testing.T) {
	e := stockedAuthoring(t)
	e.Select(recordIndex(t, e, "Stock", 0), false)
	_, must := editorMCP(t, e)
	must("delete_record", map[string]any{"kind": "Unit", "index": 0})
	if sel := e.Selected(); sel >= 0 {
		t.Fatalf("the removed stock is still selected: %+v", e.Document().Records[sel])
	}
}
