package game

import (
	"image"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// fallenCandidate is a living installed Human whose class has a corpse class.
func fallenCandidate(t *testing.T, x *MapInspector) (string, int) {
	t.Helper()
	return unitWhere(t, x, "a living Human with a corpse class", func(u alm.Unit, r mapload.Resolution, s unitSheet) bool {
		c := x.units.Classes[int32(u.ClassID)]
		return r.Found() && r.Arm == mapload.ArmHumansByType && s.Life == ui.LifeAlive && c != nil && c.Corpse != nil && len(c.Corpse.TierFrames(0)) > 0 && !hasTail(u)
	})
}

// TestReleaseEditorHealthZeroUnitDrawsFallen: at Health 0 the editor draws a
// placement with the game's held fall frame, from the same bytes, in the same
// colours; the standing pose is not that frame.
func TestReleaseEditorHealthZeroUnitDrawsFallen(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	key, i := fallenCandidate(t, x)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open(key); err != nil {
		t.Fatal(err)
	}
	standing, ok := e.Document().Viewer.EntityOf(uint32(i))
	if !ok || standing.Frame == nil || standing.Life != ui.LifeAlive {
		t.Fatalf("%s unit %d: no standing entity %+v", key, i, standing)
	}
	d, err := objectEdits(t, e).SetField("Unit", i, "hp", 0)
	if err != nil {
		t.Fatal(err)
	}
	fallen, ok := d.Viewer.EntityOf(uint32(i))
	if !ok || fallen.Frame == nil {
		t.Fatal("no fallen entity")
	}
	m, err := alm.Open(d.SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	mw, err := openMapWorld(m, x.table, x.units, d.Viewer)
	if err != nil {
		t.Fatal(err)
	}
	mw.usePlayerColours(m) // as a mission start does
	mw.entityDraws()
	mw.scene += 1 << 20
	var want *ui.MapEntity
	for _, d := range mw.entityDraws() {
		if d.ID == uint32(i) {
			want = &d
			break
		}
	}
	if want == nil || want.Life == ui.LifeAlive || want.Frame == nil {
		t.Fatalf("the game draws placement %d alive or not at all: %+v", i, want)
	}
	same := func(a, b ui.MapEntity) bool {
		return a.Art == b.Art && a.Mirror == b.Mirror && a.Frame.Width == b.Frame.Width && a.Frame.Height == b.Frame.Height &&
			reflect.DeepEqual(a.Frame.Pixels, b.Frame.Pixels) && a.Frame.Palette == b.Frame.Palette
	}
	if !same(fallen, *want) || fallen.Life != want.Life {
		t.Fatalf("%s unit %d: the editor's Health 0 frame differs from the game's: art %v/%v mirror %v/%v size %dx%d/%dx%d palette %v pixels %v life %d/%d",
			key, i, fallen.Art == want.Art, want.Art != nil, fallen.Mirror, want.Mirror, fallen.Frame.Width, fallen.Frame.Height, want.Frame.Width, want.Frame.Height,
			fallen.Frame.Palette == want.Frame.Palette, reflect.DeepEqual(fallen.Frame.Pixels, want.Frame.Pixels), fallen.Life, want.Life)
	}
	if same(standing, *want) {
		t.Fatal("loss control: the standing pose is the fallen frame")
	}
	t.Logf("%s unit %d class %d: Health 0 draws corpse class %q life %d", key, i, m.Units[i].ClassID, fallen.Art.Name, fallen.Life)
}

// TestReleaseEditorUnitBlockFrames writes the owner frames of the unit block,
// the fallen unit, a message's whole text and a moved Add window, rendered
// headless from the install, as PNGs into AGAINROM_EDITOR_FRAMES; without it
// the same steps run.
func TestReleaseEditorUnitBlockFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	dir, lang := os.Getenv("AGAINROM_EDITOR_FRAMES"), installLanguage(x)
	shot := func(e *ui.MapEditor, name string) {
		t.Helper()
		// Glyphs of the windows a popup lies over are hidden under it.
		cover, _ := e.PromptRect()
		writeEditorFrame(t, e, filepath.Join(dir, name+"-"+lang+".png"), dir != "", cover)
	}
	c := &editorClicker{t: t, now: time.Unix(1, 0)}
	key, i := fallenCandidate(t, x)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open(key); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	rec := recordOf(e.Document(), "Unit", i)
	e.Select(rec, true)
	typer := overrideTyper{t: t, c: c, e: e, i: i}
	vitals := x.inspectionUTF8(x.word("editor.unit.group.vitals"))
	scrollTo(t, e, vitals)
	shot(e, "unit-inherited")
	typer.set("body", 40)
	typer.set("hp", 50)
	e.Select(rec, true)
	scrollTo(t, e, vitals)
	shot(e, "unit-overridden")
	typer.text("hp", "0")
	e.Select(rec, true)
	scrollTo(t, e, vitals)
	shot(e, "unit-health0")
	_ = e.Key("escape")
	e.ShowCell(e.Document().Records[rec].Cell)
	shot(e, "unit-fallen")

	e = x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	targets := e.Document().Script.Targets[ui.ScriptMessageParameter]
	action, longest := -1, 0
	for n, a := range e.Document().Script.Actions {
		k := slices.IndexFunc(targets, func(m ui.InspectionScriptTarget) bool { return !m.NewMessage && m.Value == a.Values[0] })
		if a.Opcode == 2 && a.Types[0] == 1 && k >= 0 && len(targets[k].Text) > longest {
			action, longest = n, len(targets[k].Text)
		}
	}
	if action < 0 {
		t.Fatal("scenario/10.alm names no message")
	}
	e.Select(recordIndex(t, e, "Action", action), false)
	k := slices.IndexFunc(targets, func(m ui.InspectionScriptTarget) bool {
		return !m.NewMessage && m.Value == e.Document().Script.Actions[action].Values[0]
	})
	shown, whole := strings.Join(strings.Fields(popupText(e)), " "), strings.Join(strings.Fields(targets[k].Text), " ")
	if !strings.Contains(shown, whole) || !strings.Contains(targets[k].Choice, "...") && strings.Contains(targets[k].Text, "\n") {
		t.Fatalf("action %d shows not the whole message %q:%s", action, whole, shown)
	}
	t.Logf("action %d shows its whole message of %d bytes; its row shows %q", action, longest, targets[k].Choice)
	shot(e, "message-whole")

	_ = e.Key("escape")
	_ = e.Key("scripts")
	_ = e.Key("player")
	before, after := titleDrag(t, e, image.Pt(-300, 160))
	if after.Min != before.Min.Add(image.Pt(-300, 160)) {
		t.Fatalf("the Add window moved from %v to %v", before, after)
	}
	shot(e, "add-moved")
}
