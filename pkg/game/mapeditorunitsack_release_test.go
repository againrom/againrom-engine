package game

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// TestReleaseMapEditorUnitSackBytes opens the first installed maps that
// hold a unit, at 1920x1080 in both views. Save with no edit writes the
// bytes the map writer writes for the source. An item added in the window
// of the unit's own stock, selected (or K, for a unit without a stock),
// writes the bytes the host's own stock and field writers write. Each root
// runs it.
func TestReleaseMapEditorUnitSackBytes(t *testing.T) {
	x := installedInspector(t)
	maps := 0
	for _, entry := range x.Maps {
		if maps == 3 {
			break
		}
		key := "loose/" + entry.Source
		if entry.FromArchive {
			key = "scenario/" + entry.Source
		}
		if !unitSackBytes(t, x, key, true) {
			continue
		}
		unitSackBytes(t, x, key, false)
		maps++
	}
	if maps == 0 {
		t.Fatal("no installed map holds a unit")
	}
	t.Logf("maps=%d: Save without an edit and a stock edit through the sack's window wrote the writers' bytes", maps)
}

// unitSackBytes runs the release proof on map key in one view; false when
// the map holds no unit.
func unitSackBytes(t *testing.T, x *MapInspector, key string, plain bool) bool {
	t.Helper()
	open := func() *ui.MapEditor {
		e := x.Editor()
		view := map[bool]string{true: "plain", false: "technical"}[plain]
		e.SetPanelProfile([]byte(`{"right":{"minimized":true,"view":"`+view+`","width":520}}`), nil, nil)
		e.Layout(1920, 1080)
		if err := e.Open(key); err != nil {
			t.Fatal(key, err)
		}
		return e
	}
	e := open()
	unit := recordOf(e.Document(), string(ui.KindUnit), 0)
	if unit < 0 {
		return false
	}
	source := bytes.Clone(e.Document().SourceBytes)
	s := e.Document().Edits.(*mapInspectionEdits)
	ed, err := mapedit.New(source)
	if err != nil {
		t.Fatal(key, err)
	}
	want, err := ed.SaveBytes()
	if err != nil {
		t.Fatal(key, err)
	}
	// The save writes as the user's answer to the check's question does: a
	// map with findings saves unchanged.
	s.AllowErrors()
	path, err := s.SaveAs(filepath.Join(t.TempDir(), "unchanged.alm"))
	if err != nil {
		t.Fatal(key, err)
	}
	if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, want) {
		t.Fatalf("%s: Save without an edit wrote other bytes than the map writer: %v", key, err)
	}
	c := &editorClicker{t: t}
	e.Select(unit, true)
	items := x.recordCatalog("item")
	if len(items) == 0 {
		t.Fatalf("%s: the item catalogue is empty", key)
	}
	item := items[0].Value
	twin := open().Document().Edits.(*mapInspectionEdits)
	before := bytes.Clone(s.model.Bytes())
	var stock int
	if unitStock(e) >= 0 {
		stock = e.Document().Records[openUnitStock(t, e)].Index
	} else {
		if err := e.Key("stock"); err != nil {
			t.Fatal(key, err)
		}
		_ = e.Key("escape")
		stock = e.Document().Records[e.Selected()].Index
		if _, err := twin.AddStock(0); err != nil {
			t.Fatal(key, err)
		}
	}
	// The map maker's edits, each chosen from its drop-down and applied: an
	// item added, a new recipe of its own, a duplicate that copies the
	// recipe, and the added item's recipe made a copy of the duplicate's.
	edit := func(field string, value int64) {
		t.Helper()
		if plain {
			openBlocks(t, e, c)
		}
		chooseField(t, e, c, field, value)
		if _, err := twin.SetFields(ui.KindStock, stock, []ui.InspectionFieldValue{{Key: field, Value: value, Step: 1}}); err != nil {
			t.Fatal(key, field, err)
		}
		if got := s.model.Bytes(); !bytes.Equal(got, twin.model.Bytes()) {
			t.Fatalf("%s plain %v: %s = %d in the sack's window wrote other bytes than the host's writers", key, plain, field, value)
		}
	}
	edit("additem", item)
	added := len(stockItems(t, s, stock))
	edit(fmt.Sprintf("recipe%d", added), -1)
	edit(fmt.Sprintf("item%d", added), -1)
	copied := stockItems(t, s, stock)
	if len(copied) != added+1 || copied[added].TileMarkerIndex == copied[added-1].TileMarkerIndex || copied[added].TileMarkerIndex == 0 {
		t.Fatalf("%s plain %v: Duplicate gave the items %v", key, plain, copied)
	}
	edit(fmt.Sprintf("recipe%d", added), -int64(copied[added].TileMarkerIndex)-1)
	if got := s.model.Bytes(); bytes.Equal(got, before) {
		t.Fatalf("%s plain %v: the sack's window wrote nothing", key, plain)
	}
	return true
}

// stockItems are the items of the host's stock record.
func stockItems(t *testing.T, s *mapInspectionEdits, stock int) []alm.LootElement {
	t.Helper()
	m, err := s.model.Map()
	if err != nil {
		t.Fatal(err)
	}
	loot, err := m.Loot()
	if err != nil || stock < 0 || stock >= len(loot.Records) {
		t.Fatalf("stock %d: %v", stock, err)
	}
	return loot.Records[stock].Elements
}
