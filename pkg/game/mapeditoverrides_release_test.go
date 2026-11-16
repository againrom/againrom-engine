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
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// overrideTyper sets one field of a selected unit through its popup: it
// scrolls the field into view, types the value and presses Apply.
type overrideTyper struct {
	t      *testing.T
	c      *editorClicker
	e      *ui.MapEditor
	i      int
	decode func([]byte) string
}

func (o overrideTyper) section(section string) {
	o.t.Helper()
	if tab, ok := o.e.FindControl(ui.ControlQuery{Role: ui.ControlObjectSection, Key: section}); ok {
		o.c.click(o.e, tab, true, "the field section")
	}
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
	fields := objectEdits(o.t, o.e).Fields("Unit", o.i)
	n := fieldIndex(o.t, fields, key)
	o.section(fields[n].Section)
	o.e.ScrollDetails(-1000)
	r, ok := o.e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: n})
	for k := 0; !ok && k < 60; k++ {
		o.e.ScrollDetails(1)
		r, ok = o.e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: n})
	}
	o.c.click(o.e, r, ok, "the "+key+" field")
	return n
}

// text types s over field key and presses Apply. The press puts the caret
// where it lands, which can be inside the value, so the old value is
// selected whole and removed before s is typed.
func (o overrideTyper) text(key, s string) {
	o.t.Helper()
	o.field(key)
	_ = o.e.Key("selectall")
	_ = o.e.Key("backspace")
	o.e.TextInput(s)
	r, ok := o.e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 0})
	o.c.click(o.e, r, ok, "Apply")
}

// choose picks choice v of field key's drop-down and presses Apply; a row
// the list does not show is chosen by typing its label.
func (o overrideTyper) choose(key string, v int64) {
	o.t.Helper()
	fields := objectEdits(o.t, o.e).Fields("Unit", o.i)
	f := fields[fieldIndex(o.t, fields, key)]
	o.field(key)
	row := slices.IndexFunc(f.Choices, func(c ui.InspectionChoice) bool { return c.Value == v })
	if r, ok := o.e.FindControl(ui.ControlQuery{Role: ui.ControlDropdownRow, N: row}); ok || row < 0 {
		o.c.click(o.e, r, ok, fmt.Sprintf("%s choice %d", key, v))
	} else {
		label := f.Choices[row].Label
		if o.decode != nil {
			label = o.decode([]byte(label))
		}
		o.e.TextInput(label)
		_ = o.e.Key("enter")
	}
	r, ok := o.e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 0})
	o.c.click(o.e, r, ok, "Apply")
}

// clear presses field key's Inherit button and Apply.
func (o overrideTyper) clear(key string) {
	o.t.Helper()
	n := o.field(key)
	r, ok := o.e.FindControl(ui.ControlQuery{Role: ui.ControlRowButton, N: n, K: 0, Flag: false})
	o.c.click(o.e, r, ok, "the "+key+" Inherit button")
	r, ok = o.e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 0})
	o.c.click(o.e, r, ok, "Apply")
}

// popupText is the open popup's rows joined into one line of single spaces.
func popupText(e *ui.MapEditor) string {
	return strings.Join(strings.Fields(strings.Join(e.PopupLines(), " ")), " ")
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
	c := &editorClicker{t: t, now: time.Unix(1, 0)}
	opened := func(key string, i int) (*ui.MapEditor, overrideTyper) {
		t.Helper()
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		// The person setup window opens over the Map panel's rows, which it
		// hides; the frames show the window alone, as the owner frames do.
		closeMapPanel(t, e)
		e.Select(recordOf(e.Document(), "Unit", i), true)
		return e, overrideTyper{t: t, c: c, e: e, i: i, decode: x.messageUTF8}
	}
	fieldShot := func(e *ui.MapEditor, i int, key, name string) {
		t.Helper()
		fields := objectEdits(t, e).Fields("Unit", i)
		n := fieldIndex(t, fields, key)
		overrideTyper{t: t, c: c, e: e, i: i}.section(fields[n].Section)
		e.ScrollDetails(-1000)
		for k := 0; k < 60; k++ {
			if _, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: n}); ok {
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
	beforeLook := bytes.Clone(e.Document().SourceBytes)
	lookMap, err := alm.Open(beforeLook)
	if err != nil {
		t.Fatal(err)
	}
	figureDir, face, ok := mapload.PlacedFigure(lookMap.Units[i], x.table)
	if !ok {
		t.Fatal("selected Human has no figure")
	}
	chosen := int64(1)
	if face == 1 {
		chosen = 2
	}
	typer.choose("face", chosen)
	e.Select(recordOf(e.Document(), "Unit", i), true)
	sex := int64(1)
	if figureDir.Female() {
		sex = 0
	}
	typer.choose("sex", sex)
	e.Select(recordOf(e.Document(), "Unit", i), true)
	_, _, picture, err := objectEdits(t, e).UnitSettings(i, nil)
	if err != nil || picture == nil {
		t.Fatal("native look has no installed doll preview", err)
	}
	fieldShot(e, i, "face", "human-face-sex-doll")
	_ = e.Key("undo")
	_ = e.Key("undo")
	if !bytes.Equal(beforeLook, e.Document().SourceBytes) {
		t.Fatal("look Undo changed unrelated native bytes")
	}
	e.Select(recordOf(e.Document(), "Unit", i), true)
	host := objectEdits(t, e)
	fields := host.Fields("Unit", i)
	npc, value, tried := int32(-1), int64(0), 0
	for _, choice := range fields[fieldIndex(t, fields, "identity")].Choices {
		id, ok := x.identityOf(choice.Value)
		if !ok || id.Kind != mapload.UnitNamed {
			continue
		}
		if tried++; tried > 10 {
			break
		}
		if _, _, pic, err := host.UnitSettings(i, []ui.InspectionFieldValue{{Key: "identity", Value: choice.Value}}); err == nil && pic != nil {
			npc, value = id.NPC, choice.Value
			break
		}
	}
	if npc < 0 {
		t.Fatal("no drawable named character among the first ten")
	}
	typer.choose("identity", value)
	e.Select(recordOf(e.Document(), "Unit", i), true)
	named, err := alm.Open(e.Document().SourceBytes)
	if err != nil || named.Units[i].Flags&1 == 0 || int32(named.Units[i].ClassSubID) != npc {
		t.Fatal("the identity choice did not write the named character", err)
	}
	fieldShot(e, i, "identity", "human-npc-record-doll")
	_ = e.Key("undo")
	if !bytes.Equal(beforeLook, e.Document().SourceBytes) {
		t.Fatal("NPC Undo changed unrelated native bytes")
	}
	e.Select(recordOf(e.Document(), "Unit", i), true)
	fields = objectEdits(t, e).Fields("Unit", i)
	choices := fields[fieldIndex(t, fields, "carryadd")].Choices
	if len(choices) < 2 {
		t.Fatal("carried item catalog is empty")
	}
	// A carried item is added in the unit's sack's window, the stock's: the
	// records list selects one the unit has, and K adds one it lacks.
	undos := 1
	if stock := unitStock(e); stock >= 0 {
		e.Select(stock, true)
	} else {
		if err := e.Key("stock"); err != nil {
			t.Fatal(err)
		}
		_ = e.Key("escape")
		undos++
	}
	recordFieldInput(t, e, "additem", choices[1].Value)
	e.Select(recordOf(e.Document(), "Unit", i), true)
	fields = objectEdits(t, e).Fields("Unit", i)
	// The person window's bag lists the added item.
	carried := slices.IndexFunc(fields, func(f ui.InspectionField) bool {
		_, _, ok := carriedField(f.Key)
		return ok && f.Parent == "" && f.Key != "carryadd" && f.Value == choices[1].Value
	})
	if carried < 0 {
		t.Fatal("the bag does not list the added item")
	}
	fieldShot(e, i, fields[carried].Key, "human-carried-and-derived-loot")
	for _, size := range []image.Point{{800, 600}, {640, 480}} {
		e.Layout(size.X, size.Y)
		fieldShot(e, i, fields[carried].Key, fmt.Sprintf("human-setup-%dx%d", size.X, size.Y))
	}
	e.Layout(1280, 800)
	for range undos {
		_ = e.Key("undo")
	}
	if !bytes.Equal(beforeLook, e.Document().SourceBytes) {
		t.Fatal("carried item Undo changed unrelated bytes")
	}
	e.Select(recordOf(e.Document(), "Unit", i), true)
	fieldShot(e, i, "body", "human-fields-template")
	for _, s := range []struct {
		key string
		v   int64
	}{{"body", 40}, {"reaction", 45}, {"skill.blade", 9}, {"protection.fire", 60}, {"gear", 1}} {
		typer.set(s.key, s.v)
		e.Select(recordOf(e.Document(), "Unit", i), true)
	}
	fieldShot(e, i, "body", "human-fields-set")
	for _, k := range []string{"body", "reaction", "skill.blade", "protection.fire", "gear"} {
		typer.set(k, overrideAbsent(k))
		e.Select(recordOf(e.Document(), "Unit", i), true)
	}
	fieldShot(e, i, "body", "human-fields-cleared")
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
}
