package game

import (
	"bytes"
	"image"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// personLines are every line of the person window over its sections: each
// section's rows and the window's notes.
func personLines(t *testing.T, e *ui.MapEditor, c *editorClicker) []string {
	t.Helper()
	var out []string
	for _, section := range []string{"identity", "equipment", "inventory", "stats"} {
		if tab, ok := e.ObjectSectionRect(section); ok {
			c.click(e, tab, true, "section "+section)
		}
		e.ScrollDetails(-1000)
		for range 80 {
			out = append(out, e.PopupLines()...)
			e.ScrollDetails(1)
		}
	}
	return out
}

// The person window shows, in both views, the rows the map maker keeps: who
// the unit is under the label Kind, its player under Player, its group, its
// name, face and sex. It shows no cell row, no Fields heading, no facing,
// no explanation under the name, no dialogue speaker, no map keys, no used
// by lines, no count of stored overrides and no key hint.
func TestEditorPersonWindowShowsTheKeptRows(t *testing.T) {
	for _, plain := range []bool{false, true} {
		s, _ := viewsMap(t)
		c := &editorClicker{t: t}
		e := viewsWindow(t, s, plain, ui.KindUnit)
		x := s.inspector
		fields := e.ObjectFields()
		if got := fields[fieldIndex(t, fields, "identity")].Label; got != "Kind" {
			t.Errorf("plain %v: who the unit is reads %q, want Kind", plain, got)
		}
		if got := fields[fieldIndex(t, fields, "owner")].Label; got != "Player" {
			t.Errorf("plain %v: the unit's player reads %q, want Player", plain, got)
		}
		keep := []string{"identity", "owner", "group", "name"}
		if !plain {
			keep = append(keep, "face", "sex")
		}
		for _, key := range keep {
			if !showField(t, e, c, key) {
				t.Errorf("plain %v: %s does not show", plain, key)
			}
		}
		for _, key := range []string{"facing", "speaker", "keys"} {
			if showField(t, e, c, key) {
				t.Errorf("plain %v: %s shows", plain, key)
			}
		}
		note := x.word("editor.unit.name.note")
		for _, line := range personLines(t, e, c) {
			for _, gone := range []string{x.word("editor.object.cell") + " ", x.word("editor.plain.position") + " ", x.word("editor.usedby.title"),
				x.word("editor.unit.speaker"), x.word("editor.unit.keys"), x.word("editor.unit.field.facing"), "Stored stat overrides", "Tab: next field"} {
				if strings.HasPrefix(line, gone) {
					t.Errorf("plain %v: the window shows %q", plain, line)
				}
			}
			if line == x.word("editor.object.fields") || len(line) > 15 && strings.Contains(note, line) {
				t.Errorf("plain %v: the window shows %q", plain, line)
			}
		}
	}
	for _, c := range []struct{ lang, kind, player string }{{"", "Kind", "Player"}, {"ru", "Вид", "Игрок"}} {
		book := words.For(c.lang)
		if got := book.Text("editor.unit.field.identity"); got != c.kind || book.Text("editor.plain.who") != c.kind {
			t.Errorf("%q: who the unit is reads %q, want %q", c.lang, got, c.kind)
		}
		if got := book.Text("editor.unit.field.owner"); got != c.player || book.Text("editor.plain.player") != c.player {
			t.Errorf("%q: the unit's player reads %q, want %q", c.lang, got, c.player)
		}
	}
}

// The person window's bag is the pick that adds an item to the unit's sack
// and the sack's numbered items: an item chosen there and Applied writes the
// bytes the host's sack writer writes for the same choice, and the new item
// then stands in the bag's list; removing a listed item writes what the
// writer writes.
func TestEditorPersonBagWritesThroughTheSackWriter(t *testing.T) {
	s, raw := viewsMap(t)
	c := &editorClicker{t: t}
	e := viewsWindow(t, s, false, ui.KindUnit)
	before := 0
	for _, f := range e.ObjectFields() {
		if strings.HasPrefix(f.Key, "carry") && f.Parent == "" && f.Key != "carryadd" {
			before++
			if !showField(t, e, c, f.Key) {
				t.Errorf("the bag does not list %s", f.Key)
			}
		}
	}
	if before == 0 {
		t.Fatal("the fixture's unit carries nothing")
	}
	chooseField(t, e, c, "carryadd", 0x0e08)
	twin, _ := viewsMap(t)
	if _, err := twin.SetFields(ui.KindUnit, 0, []ui.InspectionFieldValue{{Key: "carryadd", Value: 0x0e08, Step: 1}}); err != nil {
		t.Fatal(err)
	}
	want := twin.model.Bytes()
	if bytes.Equal(raw, want) || !bytes.Equal(s.model.Bytes(), want) {
		t.Fatal("the bag's item pick wrote other bytes than the sack writer")
	}
	after := 0
	for _, f := range e.ObjectFields() {
		if strings.HasPrefix(f.Key, "carry") && f.Parent == "" && f.Key != "carryadd" {
			after++
		}
	}
	if after != before+1 {
		t.Fatalf("the bag lists %d items after the pick, want %d", after, before+1)
	}
	chooseField(t, e, c, "carry1", 0)
	if _, err := twin.SetFields(ui.KindUnit, 0, []ui.InspectionFieldValue{{Key: "carry1", Value: 0, Step: 1}}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(s.model.Bytes(), twin.model.Bytes()) {
		t.Fatal("removing a bag item wrote other bytes than the sack writer")
	}
}

// The view switch of every window that offers it stands in the button row,
// on the line of Apply and left of every button; a press turns the view,
// and the switch stays in the button row in both views.
func TestEditorViewSwitchStandsInTheButtonRow(t *testing.T) {
	for _, kind := range []ui.RecordKind{ui.KindUnit, ui.KindStructure, ui.KindSack} {
		s, _ := viewsMap(t)
		c := &editorClicker{t: t}
		e := viewsWindow(t, s, true, kind)
		for _, plain := range []bool{true, false, true} {
			if e.PlainView() != plain {
				t.Fatalf("%s: plain %v, want %v", kind, e.PlainView(), plain)
			}
			sw, ok := e.ViewSwitchRect()
			apply, applied := e.PopupButtonRect(0)
			if !ok || !applied {
				t.Fatalf("%s plain %v: switch %v button %v", kind, plain, ok, applied)
			}
			if sw.Min.Y != apply.Min.Y || sw.Dy() != apply.Dy() {
				t.Errorf("%s plain %v: the switch %v is not on Apply's line %v", kind, plain, sw, apply)
			}
			for n := 0; ; n++ {
				b, ok := e.PopupButtonRect(n)
				if !ok {
					break
				}
				if b.Min.Y == sw.Min.Y && sw.Max.X > b.Min.X {
					t.Errorf("%s plain %v: the switch %v is not left of button %d %v", kind, plain, sw, n, b)
				}
			}
			c.click(e, sw, true, "view switch")
		}
	}
}

// A unit whose facing, cell, map keys and dialogue speaker are not the
// defaults keeps each of them, byte for byte, through an edit Applied in
// the person window of either view, which shows none of them: the written
// map equals the one the host's field writer writes for the edit alone.
func TestEditorPersonWindowKeepsTheValuesItDoesNotShow(t *testing.T) {
	prepare := func(t *testing.T) *mapInspectionEdits {
		s, _ := viewsMap(t)
		if _, err := s.SetField(ui.KindUnit, 0, "facing", 5); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Move(ui.KindUnit, 0, image.Pt(12, 9)); err != nil {
			t.Fatal(err)
		}
		s.unitViews = nil
		return s
	}
	shown := func(s *mapInspectionEdits) []int64 {
		var out []int64
		for _, key := range []string{"facing", "class", "tier", "definition", "npcflag"} {
			v, err := s.model.FieldValue(mapedit.KindUnit, 0, key)
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, v)
		}
		x, y, err := s.model.Position(mapedit.KindUnit, 0)
		if err != nil {
			t.Fatal(err)
		}
		speaker, miss := s.UnitSpeaker(0)
		return append(out, int64(x), int64(y), speaker, int64(miss.Cause), miss.Record)
	}
	for _, plain := range []bool{true, false} {
		s := prepare(t)
		raw := s.model.Bytes()
		kept := shown(s)
		if kept[0] != 5 || kept[1] == 0 {
			t.Fatalf("the fixture's unit has facing %d and class key %d", kept[0], kept[1])
		}
		c := &editorClicker{t: t}
		e := viewsWindow(t, s, plain, ui.KindUnit)
		typeField(t, e, c, "hp", 50)
		twin := prepare(t)
		if _, err := twin.SetFields(ui.KindUnit, 0, []ui.InspectionFieldValue{{Key: "hp", Value: 50, Step: 1}}); err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(raw, s.model.Bytes()) || !bytes.Equal(s.model.Bytes(), twin.model.Bytes()) {
			t.Fatalf("plain %v: Apply wrote other bytes than the field writer for the edit alone", plain)
		}
		if got := shown(s); !slices.Equal(got, kept) {
			t.Fatalf("plain %v: the unit's facing, keys, cell and speaker %v, want %v", plain, got, kept)
		}
	}
}
