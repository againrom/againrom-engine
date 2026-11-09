package game

import (
	"bytes"
	"slices"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// lastButton presses the window's last button, Back when the window has one.
func lastButton(t *testing.T, e *ui.MapEditor, c *editorClicker) {
	t.Helper()
	n := 0
	for {
		if _, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: n + 1}); !ok {
			break
		}
		n++
	}
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: n})
	c.click(e, r, ok, "the last button")
}

// unitStock is the record of the selected unit's own stock, -1 for none or
// for a unit the person window does not edit.
func unitStock(e *ui.MapEditor) int {
	s, ok := e.Document().Edits.(*mapInspectionEdits)
	if !ok || e.Selected() < 0 {
		return -1
	}
	stock, err := s.unitTemplate(mapedit.KindUnit, e.Document().Records[e.Selected()].Index).stock()
	if err != nil || stock < 0 {
		return -1
	}
	return slices.IndexFunc(e.Document().Records, func(r ui.InspectionRecord) bool { return r.Kind == ui.KindStock && r.Index == stock })
}

// openUnitStock selects the selected unit's own stock as the records list
// selects it, the window the full sack editor stands in, and returns its
// record.
func openUnitStock(t *testing.T, e *ui.MapEditor) int {
	t.Helper()
	stock := unitStock(e)
	if stock < 0 {
		t.Fatal("the unit has no stock")
	}
	e.Select(stock, false)
	if e.Selected() != stock {
		t.Fatalf("selecting the stock shows record %d, want %d", e.Selected(), stock)
	}
	return stock
}

// The unit's own stock, selected, edits in the window a selected stock
// takes, over the game's host, in both views: an edit there writes the
// bytes the host's field writer writes and undoes and redoes exactly with
// the stock's window kept.
func TestEditorUnitSackEditsInTheStockWindow(t *testing.T) {
	for _, plain := range []bool{true, false} {
		s, raw := viewsMap(t)
		c := &editorClicker{t: t}
		e := viewsWindow(t, s, plain, ui.KindUnit)
		e.Layout(1920, 1080)
		stock := openUnitStock(t, e)
		if e.PlainView() != plain {
			t.Fatalf("plain %v: the sack's window changed the view", plain)
		}
		if plain {
			openBlocks(t, e, c)
		}
		typeField(t, e, c, "use1", 77)
		twin, _ := viewsMap(t)
		if _, err := twin.SetFields(ui.KindStock, e.Document().Records[stock].Index, []ui.InspectionFieldValue{{Key: "use1", Value: 77, Step: 1}}); err != nil {
			t.Fatal(err)
		}
		want := twin.model.Bytes()
		if bytes.Equal(raw, want) || !bytes.Equal(s.model.Bytes(), want) {
			t.Fatalf("plain %v: the sack's window wrote other bytes than the field writer", plain)
		}
		for _, step := range []struct {
			key  string
			want []byte
		}{{"undo", raw}, {"redo", want}} {
			if err := e.Key(step.key); err != nil || !bytes.Equal(s.model.Bytes(), step.want) || e.Selected() != stock {
				t.Fatalf("plain %v: %s gives other bytes or leaves the sack (record %d): %v", plain, step.key, e.Selected(), err)
			}
		}
	}
}

// The release proof of map bytes runs on a map whose units own no stock,
// where K adds one, in both views.
func TestEditorUnitSackBytesMatchTheWriters(t *testing.T) {
	x, path := identifiedUnits(t)
	for _, plain := range []bool{true, false} {
		if !unitSackBytes(t, x, path, plain) {
			t.Fatal("the map holds no unit")
		}
	}
}

// K on a unit opens the stock it adds in the stock's window, with Back to
// the unit.
func TestEditorStockKeyOpensTheNewStockWithBack(t *testing.T) {
	e := openIdentifiedUnits(t)
	unit := recordIndex(t, e, "Unit", 0)
	e.Select(unit, false)
	if err := e.Key("stock"); err != nil {
		t.Fatal(err)
	}
	if r := e.Document().Records[e.Selected()]; r.Kind != ui.KindStock {
		t.Fatalf("K selects a %s", r.Kind)
	}
	// Esc closes the notice K leaves; the stock's window stays.
	if _ = e.Key("escape"); e.Message() != "" || e.Document().Records[e.Selected()].Kind != ui.KindStock {
		t.Fatalf("Esc leaves the notice %q on record %d", e.Message(), e.Selected())
	}
	lastButton(t, e, &editorClicker{t: t})
	if e.Selected() != unit {
		t.Fatalf("Back shows record %d, want the unit %d", e.Selected(), unit)
	}
}
