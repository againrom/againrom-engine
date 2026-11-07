package game

import (
	"bytes"
	"image"
	"os"
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// The unit window a map maker opens, the basic view, names a placed
// mercenary by his type and offers his level next to it; choosing another
// level keys that level's row as one undo step. A unit that is no mercenary
// has no level in either view.
func TestEditorPlainUnitWindowOffersTheMercenaryLevel(t *testing.T) {
	s := peopleIdentityEditor(t, "en")
	x := s.inspector
	if _, err := s.AddUnitIdentity(image.Pt(20, 20), personValue(5)); err != nil {
		t.Fatal(err)
	}
	m, _ := s.model.Map()
	index := len(m.AuthoredUnits) - 1
	if _, err := s.SetField(ui.KindUnit, index, "level", 3); err != nil {
		t.Fatal(err)
	}
	row := func() int {
		m, _ := s.model.Map()
		id, _, _ := mapload.IdentifyAt(m.AuthoredUnits[index], x.table, s.mission())
		return id.Row
	}
	if row() != 9 {
		t.Fatalf("level 3 keyed row %d, want NPC03_3 (row 9)", row())
	}
	level3 := append([]byte(nil), s.model.Bytes()...)
	doc, err := s.project(s.model)
	if err != nil {
		t.Fatal(err)
	}
	e := ui.NewMapEditor(nil, nil, func(string) (*ui.InspectionDocument, error) { return doc, nil })
	e.Layout(1280, 800)
	if err := e.Open("people.alm"); err != nil {
		t.Fatal(err)
	}
	e.SetPanelProfile(nil, os.ErrNotExist, nil)
	e.Select(recordOf(doc, "Unit", index), false)
	if !e.PlainView() {
		t.Fatal("the unit window opened technical")
	}
	has := func(key string) bool {
		return slices.ContainsFunc(e.ObjectFields(), func(f ui.InspectionField) bool { return f.Key == key })
	}
	if !has("identity") || !has("level") {
		t.Fatalf("the basic window lacks the identity or the level: %q", e.PopupLines())
	}
	fields := e.ObjectFields()
	if got := fields[fieldIndex(t, fields, "level")]; got.Value != 3 || got.Label != "Level" {
		t.Fatalf("level field = %+v, want Level 3", got)
	}
	idf := fields[fieldIndex(t, fields, "identity")]
	for _, c := range idf.Choices {
		if c.Value == idf.Value && c.Label != "NPC03_3" {
			t.Fatalf("the mercenary's identity is labelled %q, want NPC03_3", c.Label)
		}
	}
	if idf.Value != personValue(9) {
		t.Fatalf("identity value %#x, want the level-3 row's cell", idf.Value)
	}
	step := s.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindUnit}, "identity")
	var labels []string
	for _, g := range step.Groups {
		if g.Head == x.inspectionUTF8(x.word("editor.unit.groups.mercenaries")) {
			for _, v := range g.Values {
				i := slices.IndexFunc(step.Choices, func(c ui.InspectionChoice) bool { return c.Value == v })
				labels = append(labels, step.Choices[i].Label)
			}
		}
	}
	if want := []string{"NPC03_1", "NPC03_2", "NPC03_3", "NPC03_4", "NPC10_1", "NPC10_2", "NPC10_3", "NPC10_4"}; !slices.Equal(labels, want) {
		t.Fatalf("mercenary cells %q, want %q", labels, want)
	}
	c := &editorClicker{t: t}
	chooseField(t, e, c, "level", 2)
	if row() != 7 {
		t.Fatalf("level 2 keyed row %d, want NPC03_2 (row 7)", row())
	}
	level2 := append([]byte(nil), s.model.Bytes()...)
	if _, err := s.Undo(); err != nil || !bytes.Equal(level3, s.model.Bytes()) {
		t.Fatal("one undo did not restore level 3", err)
	}
	if _, err := s.SetField(ui.KindUnit, index, "identity", personValue(7)); err != nil || !bytes.Equal(level2, s.model.Bytes()) {
		t.Fatal("the level-2 cell wrote other bytes than the level field", err)
	}
	if _, err := s.Undo(); err != nil || !bytes.Equal(level3, s.model.Bytes()) {
		t.Fatal("one undo did not restore level 3 after the cell", err)
	}

	for _, plain := range []bool{true, false} {
		e.SetPanelProfile(nil, os.ErrNotExist, nil)
		e.Select(recordOf(doc, "Unit", 0), false)
		if !plain {
			r, ok := e.ViewSwitchRect()
			c.click(e, r, ok, "view switch")
		}
		if e.PlainView() != plain {
			t.Fatalf("plain view = %v, want %v", e.PlainView(), plain)
		}
		if has("level") {
			t.Fatalf("a unit that is no mercenary has a level field (plain %v)", plain)
		}
	}
}
