package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// overrideTyper sets one field of a selected unit through its popup: it
// scrolls the field into view, types the value and presses Apply.
type overrideTyper struct {
	t *testing.T
	c *editorClicker
	e *ui.MapEditor
	i int
}

func (o overrideTyper) set(key string, v int64) {
	o.t.Helper()
	fields := objectEdits(o.t, o.e).Fields("Unit", o.i)
	f := fields[fieldIndex(o.t, fields, key)]
	switch {
	case len(f.Choices) != 0:
		o.choose(key, v)
	case f.Inherit && v == f.Absent:
		o.text(key, "")
	default:
		o.text(key, fmt.Sprint(v))
	}
}

// field scrolls field key into view and presses it.
func (o overrideTyper) field(key string) int {
	o.t.Helper()
	n := fieldIndex(o.t, objectEdits(o.t, o.e).Fields("Unit", o.i), key)
	o.e.ScrollDetails(-1000)
	r, ok := o.e.FieldRect(n)
	for k := 0; !ok && k < 60; k++ {
		o.e.ScrollDetails(1)
		r, ok = o.e.FieldRect(n)
	}
	o.c.click(o.e, r, ok, "the "+key+" field")
	return n
}

// text types s over field key and presses Apply.
func (o overrideTyper) text(key, s string) {
	o.t.Helper()
	o.field(key)
	for k := 0; k < 16; k++ {
		_ = o.e.Key("backspace")
	}
	o.e.TextInput(s)
	r, ok := o.e.PopupButtonRect(0)
	o.c.click(o.e, r, ok, "Apply")
}

// choose picks choice v of field key's drop-down and presses Apply.
func (o overrideTyper) choose(key string, v int64) {
	o.t.Helper()
	fields := objectEdits(o.t, o.e).Fields("Unit", o.i)
	f := fields[fieldIndex(o.t, fields, key)]
	o.field(key)
	row := slices.IndexFunc(f.Choices, func(c ui.InspectionChoice) bool { return c.Value == v })
	r, ok := o.e.DropdownRowRect(row)
	o.c.click(o.e, r, ok, fmt.Sprintf("%s choice %d", key, v))
	r, ok = o.e.PopupButtonRect(0)
	o.c.click(o.e, r, ok, "Apply")
}

// clear presses field key's Inherit button and Apply.
func (o overrideTyper) clear(key string) {
	o.t.Helper()
	n := o.field(key)
	r, ok := o.e.RowButtonRect(n, 0, false)
	o.c.click(o.e, r, ok, "the "+key+" Inherit button")
	r, ok = o.e.PopupButtonRect(0)
	o.c.click(o.e, r, ok, "Apply")
}

// popupText is the open popup's rows joined into one line of single spaces.
func popupText(e *ui.MapEditor) string {
	return strings.Join(strings.Fields(strings.Join(e.PopupLines(), " ")), " ")
}

// TestReleaseMapEditorOverridePopupWritesTheSetFieldBytes sets each kind of
// override field of an installed Human placement through its popup: the
// bytes equal SetField's on the bytes before, the edit moves no byte outside
// the field, the label carries the template's value, Undo and Redo restore
// both images, and typing the absent value writes the bytes before. A value
// outside the field's width is refused with its reason, and the gear bit is
// neither offered nor accepted on a creature.
func TestReleaseMapEditorOverridePopupWritesTheSetFieldBytes(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	key, i := unitWhere(t, x, "an equipped Human of the type-key arm without overrides", func(u alm.Unit, r mapload.Resolution, s unitSheet) bool {
		return !hasTail(u) && gearOffered(u, x.table) && slices.ContainsFunc(s.Worn[:], func(it sim.ItemInstance) bool { return !it.Empty() })
	})
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open(key); err != nil {
		t.Fatal(err)
	}
	rec := recordOf(e.Document(), "Unit", i)
	e.Select(rec, true)
	raw := e.Document().SourceBytes
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	template, err := unitSheets(m, x.table)
	if err != nil {
		t.Fatal(err)
	}
	type fieldCase struct {
		key   string
		v     int64
		width int
	}
	typer := overrideTyper{t: t, c: &editorClicker{t: t, now: time.Unix(1, 0)}, e: e, i: i}
	for _, c := range []fieldCase{{"body", 33, 1}, {"absorption", 240, 1}, {"skill.pike", 7, 1},
		{"protection.astral", 55, 1}, {"hp", 50, 2}, {"mana", 7, 2}, {"gear", 1, 1}} {
		before := e.Document().SourceBytes
		fields := objectEdits(t, e).Fields("Unit", i)
		f := fields[fieldIndex(t, fields, c.key)]
		label := f.Label
		current, err := alm.Open(before)
		if err != nil {
			t.Fatal(err)
		}
		now, err := unitSheets(current, x.table)
		if err != nil {
			t.Fatal(err)
		}
		if was, ok := overrideTemplate(now[i].Values, c.key); ok && (!f.Inherit || f.Inherited != fmt.Sprint(was)) {
			t.Fatalf("%s: the empty field's hint %q is not the game's %d", c.key, f.Inherited, was)
		}
		typer.set(c.key, c.v)
		edited := e.Document().SourceBytes
		witness, err := mapedit.New(before)
		if err != nil {
			t.Fatal(err)
		}
		if err := witness.SetField(mapedit.KindUnit, i, c.key, c.v); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(edited, witness.Bytes()) {
			t.Fatalf("%s %s unit %d: the popup's bytes differ from SetField %d", key, c.key, i, c.v)
		}
		changed := 0
		for n := range before {
			if before[n] != edited[n] {
				changed++
			}
		}
		if changed == 0 || changed > c.width {
			t.Fatalf("%s moved %d bytes, want 1..%d", c.key, changed, c.width)
		}
		if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, before) {
			t.Fatalf("%s: Undo does not restore the bytes before: %v", c.key, err)
		}
		if err := e.Key("redo"); err != nil || !bytes.Equal(e.Document().SourceBytes, edited) {
			t.Fatalf("%s: Redo does not restore the edit: %v", c.key, err)
		}
		e.Select(rec, true)
		t.Logf("%s unit %d: %s %d moved %d bytes; label %q", key, i, c.key, c.v, changed, label)
	}

	// The edited placement: the popup lists each override beside the
	// template's value, and its Normal-difficulty sheet is the game's spawn of
	// the edited bytes.
	text := popupText(e)
	name := x.overrideName("body")
	want := x.inspectionUTF8(fmt.Sprintf(x.word("editor.unit.override.item"), name, 33, template[i].Values.Body))
	if !strings.Contains(text, want) || !strings.Contains(text, x.inspectionUTF8(x.word("editor.unit.override.skipped"))) {
		t.Fatalf("the popup lacks %q or the skipped line:\n%s", want, text)
	}
	edited, err := alm.Open(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	sheets, err := unitSheets(edited, x.table)
	if err != nil {
		t.Fatal(err)
	}
	got := sheets[i]
	if got.Values.Absorption != 240 || got.Values.Skill[4] != 7 || got.Values.Protection[4] != 55 || got.Values.Mana != 7 || got.Values.HP != 50 ||
		slices.ContainsFunc(got.Worn[:], func(it sim.ItemInstance) bool { return !it.Empty() }) {
		t.Fatalf("the sheet of the edited placement: %+v, worn %v", got.Values, got.Worn)
	}

	// Clearing writes the absent value: the bytes return to the installed
	// ones, each field as its own edit.
	for _, k := range []string{"body", "absorption", "skill.pike", "protection.astral", "hp", "mana", "gear"} {
		typer.set(k, overrideAbsent(k))
		e.Select(rec, true)
	}
	if !bytes.Equal(e.Document().SourceBytes, raw) {
		t.Fatal("clearing every override did not restore the installed bytes")
	}
	if !strings.Contains(popupText(e), x.inspectionUTF8(x.word("editor.unit.override.none"))) {
		t.Fatalf("a placement without overrides does not say so:\n%s", popupText(e))
	}

	// A Shooting byte that is the record's only skill byte is written, and
	// the sheet keeps the template's Shooting, which is what the game applies;
	// beside a Pike byte the sheet shows it (DIV-3581).
	shooting := func() int32 {
		t.Helper()
		m, err := alm.Open(e.Document().SourceBytes)
		if err != nil {
			t.Fatal(err)
		}
		sheets, err := unitSheets(m, x.table)
		if err != nil {
			t.Fatal(err)
		}
		return sheets[i].Values.Skill[5]
	}
	was := template[i].Values.Skill[5]
	typer.set("skill.shooting", int64(was+3))
	e.Select(rec, true)
	written, err := mapedit.New(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	if v, err := written.FieldValue(mapedit.KindUnit, i, "skill.shooting"); err != nil || v != int64(was+3) || shooting() != was {
		t.Fatalf("a lone Shooting %d: field %d (%v), sheet Shooting %d, want the template's %d", was+3, v, err, shooting(), was)
	}
	typer.set("skill.pike", 7)
	e.Select(rec, true)
	if shooting() != was+3 {
		t.Fatalf("Shooting %d beside Pike 7: sheet Shooting %d", was+3, shooting())
	}
	for _, k := range []string{"skill.shooting", "skill.pike"} {
		typer.set(k, overrideAbsent(k))
		e.Select(rec, true)
	}
	if !bytes.Equal(e.Document().SourceBytes, raw) {
		t.Fatal("clearing Shooting and Pike did not restore the installed bytes")
	}

	typer.set("body", 256)
	if !bytes.Equal(e.Document().SourceBytes, raw) || !strings.Contains(popupText(e), "outside 0..255") {
		t.Fatalf("Body 256 was not refused with its reason:\n%s", popupText(e))
	}

	ckey, j := unitWhere(t, x, "a creature", func(u alm.Unit, r mapload.Resolution, s unitSheet) bool {
		return r.Arm == mapload.ArmUnits && r.Found()
	})
	e = x.Editor()
	e.Layout(1280, 800)
	if err := e.Open(ckey); err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(objectEdits(t, e).Fields("Unit", j), func(f ui.InspectionField) bool { return f.Key == "gear" }) {
		t.Fatal("a creature's popup offers the gear bit")
	}
	before := e.Document().SourceBytes
	if _, err := objectEdits(t, e).SetField("Unit", j, "gear", 1); err == nil || !strings.Contains(err.Error(), "bit 7 reaches only") {
		t.Fatalf("gear on a creature was not refused with its reason: %v", err)
	}
	if !bytes.Equal(e.Document().SourceBytes, before) {
		t.Fatal("a refused gear bit wrote")
	}
}

// TestReleaseMapEditorOverrideFrames writes the unit popup's override frames,
// rendered headless from the install, as PNGs into AGAINROM_EDITOR_FRAMES;
// without it the same steps run. In each frame the installed fonts keep every
// glyph, except the key strip under the refusal's grown popup.
func TestReleaseMapEditorOverrideFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	dir := os.Getenv("AGAINROM_EDITOR_FRAMES")
	shot := func(e *ui.MapEditor, name string, cover image.Rectangle) {
		t.Helper()
		writeEditorFrame(t, e, filepath.Join(dir, name+".png"), dir != "", cover)
	}
	sheetHeading := x.inspectionUTF8(x.word("editor.unit.sheet"))
	overHeading := x.inspectionUTF8(x.word("editor.unit.overrides"))
	c := &editorClicker{t: t, now: time.Unix(1, 0)}
	opened := func(key string, i int) (*ui.MapEditor, overrideTyper) {
		t.Helper()
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		e.Select(recordOf(e.Document(), "Unit", i), true)
		return e, overrideTyper{t: t, c: c, e: e, i: i}
	}
	fieldShot := func(e *ui.MapEditor, i int, key, name string) {
		t.Helper()
		n := fieldIndex(t, objectEdits(t, e).Fields("Unit", i), key)
		e.ScrollDetails(-1000)
		for k := 0; k < 60; k++ {
			if _, ok := e.FieldRect(n); ok {
				break
			}
			e.ScrollDetails(1)
		}
		shot(e, name, image.Rectangle{})
	}

	key, i := unitWhere(t, x, "an equipped Human of the type-key arm without overrides", func(u alm.Unit, r mapload.Resolution, s unitSheet) bool {
		return !hasTail(u) && gearOffered(u, x.table) && slices.ContainsFunc(s.Worn[:], func(it sim.ItemInstance) bool { return !it.Empty() })
	})
	e, typer := opened(key, i)
	fieldShot(e, i, "body", "human-fields-template")
	scrollTo(t, e, sheetHeading)
	shot(e, "human-sheet-before", image.Rectangle{})
	for _, s := range []struct {
		key string
		v   int64
	}{{"body", 40}, {"reaction", 45}, {"skill.blade", 9}, {"protection.fire", 60}, {"gear", 1}} {
		typer.set(s.key, s.v)
		e.Select(recordOf(e.Document(), "Unit", i), true)
	}
	fieldShot(e, i, "body", "human-fields-set")
	scrollTo(t, e, sheetHeading)
	shot(e, "human-sheet-after", image.Rectangle{})
	scrollTo(t, e, overHeading)
	shot(e, "human-overrides-set", image.Rectangle{})
	for _, k := range []string{"body", "reaction", "skill.blade", "protection.fire", "gear"} {
		typer.set(k, overrideAbsent(k))
		e.Select(recordOf(e.Document(), "Unit", i), true)
	}
	scrollTo(t, e, overHeading)
	shot(e, "human-overrides-cleared", image.Rectangle{})
	// The refusal's two note lines grow the popup over the key strip.
	typer.set("body", 256)
	shot(e, "human-refused", image.Rect(0, 760, 1280, 800))

	key, i = unitWhere(t, x, "a creature", func(u alm.Unit, r mapload.Resolution, s unitSheet) bool {
		return r.Arm == mapload.ArmUnits && r.Found() && !hasTail(u)
	})
	e, typer = opened(key, i)
	typer.set("protection.water", 80)
	e.Select(recordOf(e.Document(), "Unit", i), true)
	fieldShot(e, i, "protection.water", "creature-fields-set")
	scrollTo(t, e, overHeading)
	shot(e, "creature-overrides-set", image.Rectangle{})
}
