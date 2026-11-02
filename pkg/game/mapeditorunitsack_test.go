package game

import (
	"bytes"
	"slices"
	"testing"

	"againrom/pkg/ui"
)

// lastButton presses the window's last button, Back when the window has one.
func lastButton(t *testing.T, e *ui.MapEditor, c *editorClicker) {
	t.Helper()
	n := 0
	for {
		if _, ok := e.PopupButtonRect(n + 1); !ok {
			break
		}
		n++
	}
	r, ok := e.PopupButtonRect(n)
	c.click(e, r, ok, "the last button")
}

// openSackRow presses the selected unit's sack row and returns the record
// of the stock it selects. The unit's basic view shows no bag: from it the
// switch turns to the detailed view, whose bag opens the sack, and turns
// the sack's window back to the basic view.
func openSackRow(t *testing.T, e *ui.MapEditor, c *editorClicker) int {
	t.Helper()
	basic := e.PlainView()
	if basic {
		if showField(t, e, c, "sack") {
			t.Fatalf("the unit's basic view shows the bag: %q", e.PopupLines())
		}
		r, ok := e.ViewSwitchRect()
		c.click(e, r, ok, "view switch")
	}
	fields := e.ObjectFields()
	sack := fields[fieldIndex(t, fields, "sack")]
	stock := slices.IndexFunc(e.Document().Records, func(r ui.InspectionRecord) bool { return r.Kind == sack.Link.Kind && r.Index == sack.Link.Index })
	if sack.Link.Kind != ui.KindStock || stock < 0 {
		t.Fatalf("the bag's sack row links %v", sack.Link)
	}
	if !showField(t, e, c, "sack") {
		t.Fatalf("the bag shows no sack row: %q", e.PopupLines())
	}
	n := slices.Index(e.PopupRecordLinks(), stock)
	e.ScrollDetails(-1000)
	r, ok := e.PopupRowRect(n)
	for k := 0; !ok && n >= 0 && k < 80; k++ {
		e.ScrollDetails(1)
		r, ok = e.PopupRowRect(n)
	}
	c.click(e, r, ok, "the sack row")
	if e.Selected() != stock {
		t.Fatalf("the sack row selects record %d, want the stock %d", e.Selected(), stock)
	}
	if basic {
		r, ok := e.ViewSwitchRect()
		c.click(e, r, ok, "view switch")
	}
	return stock
}

// The unit's sack opens from the unit's bag, over the game's host, in the
// window a selected stock takes, in both views: an edit there writes the
// bytes the host's field writer writes, undoes and redoes exactly with the
// sack's window kept, and Back returns to the unit.
func TestEditorUnitSackEditsInTheStockWindow(t *testing.T) {
	for _, plain := range []bool{true, false} {
		s, raw := viewsMap(t)
		c := &editorClicker{t: t}
		e := viewsWindow(t, s, plain, ui.KindUnit)
		e.Layout(1920, 1080)
		unit := e.Selected()
		stock := openSackRow(t, e, c)
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
		lastButton(t, e, c)
		if e.Selected() != unit {
			t.Fatalf("plain %v: Back shows record %d, want the unit %d", plain, e.Selected(), unit)
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

// K on a unit opens the stock it adds in the window the sack row opens,
// with Back to the unit.
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
