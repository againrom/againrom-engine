package game

import (
	"fmt"
	"image"
	"os"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestReleaseTriggerWizardOpensShippedTriggers opens every trigger of every
// installed map in the wizard: each opens, each offers Details, and the
// count of triggers every row of which is a ready item is logged. A row the
// wizard cannot read shows as "Other" and stays as it is.
func TestReleaseTriggerWizardOpensShippedTriggers(t *testing.T) {
	x := installedInspector(t)
	total, full, partial, rowsAll, keptAll := 0, 0, 0, 0, 0
	for _, key := range installedMapKeys(x) {
		e := x.Editor()
		e.Layout(1280, 800)
		if err := e.Open(key); err != nil {
			t.Fatal(key, err)
		}
		sc := e.Document().Script
		if sc == nil {
			continue
		}
		for i, tr := range sc.Triggers {
			e.Select(tr.Record, true)
			if !e.WizardShown() {
				t.Fatalf("%s trigger %d does not open the wizard", key, i+1)
			}
			if _, ok := e.PopupButtonRect(4); !ok {
				t.Fatalf("%s trigger %d offers no Details", key, i+1)
			}
			rows, kept := e.WizardRowCounts()
			total++
			rowsAll += rows
			keptAll += kept
			switch {
			case kept == 0:
				full++
			case kept < rows:
				partial++
			}
		}
	}
	if total == 0 || full == 0 {
		t.Fatalf("%d triggers, %d shown fully as rows", total, full)
	}
	t.Logf("%d triggers: %d show fully as rows, %d partly, %d only as Other; %d rows, %d kept as Other", total, full, partial, total-full-partial, rowsAll, keptAll)
}

// TestReleaseDropZoneMatchesTheShippedShape adds a drop zone with the tool
// on the cell scenario 10 drops on and compares the action and the trigger
// the tool writes with scenario 10's.
func TestReleaseDropZoneMatchesTheShippedShape(t *testing.T) {
	x := installedInspector(t)
	shipped := x.Editor()
	shipped.Layout(1280, 800)
	if err := shipped.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	sc := wizScript(t, shipped.Document().SourceBytes)
	d := drops(sc)
	if len(d) != 1 {
		t.Fatalf("scenario 10 has %d drop actions", len(d))
	}
	cell := image.Pt(int(d[0].Value[0]), int(d[0].Value[1]))
	dropShape(t, sc, d[0], cell)

	e, _ := openGoalMap(t, emptyScriptMap(t, 160))
	e.Fit()
	_ = e.Key("drop")
	clickCell(e, cell)
	got := wizScript(t, e.Document().SourceBytes)
	g := drops(got)
	if len(g) != 1 {
		t.Fatalf("the tool wrote %d drop actions", len(g))
	}
	dropShape(t, got, g[0], cell)
	if g[0].Label != d[0].Label || g[0].Type != d[0].Type || g[0].Value != d[0].Value {
		t.Fatalf("the tool's action %+v, scenario 10's %+v", g[0], d[0])
	}
	var want, have = sc.Triggers[0], got.Triggers[0]
	for _, tr := range sc.Triggers {
		if tr.Acts[0] == d[0].ID {
			want = tr
		}
	}
	want.Acts[0], have.Acts[0] = 0, 0
	if want != have {
		t.Fatalf("the tool's trigger %+v, scenario 10's %+v", have, want)
	}
	t.Logf("scenario 10: %q drop on %v, trigger %q once %d", d[0].Label, cell, want.Name, want.Once)
}

// TestReleaseTriggerWizardFrames writes, into AGAINROM_EDITOR_FRAMES, the
// wizard with the owner's example trigger, the Draw on map step and the
// drop-zone tool with two zones, drawn with the installed fonts.
func TestReleaseTriggerWizardFrames(t *testing.T) {
	x := installedInspector(t)
	dir := os.Getenv("AGAINROM_EDITOR_FRAMES")
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	e.Fit()
	wizNewTrigger(t, e, "Treasure")
	wizPress(t, e, "", 0)
	wizPress(t, e, "c0.p1", 0)
	from, to := e.CellPoint(image.Pt(60, 60)), e.CellPoint(image.Pt(70, 66))
	now := time.Unix(1, 0)
	e.Pointer(ui.Input{CursorX: from.X, CursorY: from.Y, PrimaryDown: true}, now)
	e.Pointer(ui.Input{CursorX: to.X, CursorY: to.Y, PrimaryDown: true}, now)
	editorShot(t, e, dir, "wizard-draw-on-map", image.Rectangle{})
	e.Pointer(ui.Input{CursorX: to.X, CursorY: to.Y}, now)
	wizPress(t, e, "+action", 0)
	wizPress(t, e, "+action", 0)
	r, ok := e.FieldRect(wizField(t, e, "a1.item"))
	trigClick(t, e, r, ok, "the second action's item")
	e.ScrollDropdown(-len(e.DropdownChoices()))
	r, ok = e.DropdownRowRect(1) // win the mission, in either language
	trigClick(t, e, r, ok, "the win item")
	if !strings.Contains(strings.Join(e.PopupLines(), "\n"), "60,60,70,66") {
		t.Fatalf("the drawn area is not in the row:\n%s", trigLines(e))
	}
	editorShot(t, e, dir, "wizard-example", image.Rectangle{})
	wizApply(t, e)
	editorShot(t, e, dir, "wizard-example-applied", image.Rectangle{})

	_ = e.Key("escape")
	_ = e.Key("drop")
	clickCell(e, image.Pt(45, 45))
	if len(e.Document().Drops) != 2 {
		t.Fatalf("the tool shows %d zones", len(e.Document().Drops))
	}
	editorShot(t, e, dir, "drop-tool-two-zones", image.Rectangle{})
}

// TestReleaseTriggerWizardMapPlaysTheTrigger adds the owner's example to the
// installed scenario 10 through the wizard, its area typed in the row, and
// starts the mission with the installed unit table and the starting hero.
// The hero walks three cells east; with the area on the cell the walk ends
// on, the gold and the win follow, and with the area elsewhere the same walk
// fires neither.
func TestReleaseTriggerWizardMapPlaysTheTrigger(t *testing.T) {
	x := installedInspector(t)
	archives, err := OpenArchives(os.Getenv("AGAINROM_ASSETS"))
	if err != nil {
		t.Fatal(err)
	}
	defs, err := LoadDefinitions(archives.Containers)
	if err != nil {
		t.Fatal(err)
	}
	// walk starts b and walks the hero; it reports the world and the cell
	// the walk ends on.
	walk := func(b []byte) (*sim.World, image.Point, uint32) {
		t.Helper()
		m, err := alm.Open(b)
		if err != nil {
			t.Fatal(err)
		}
		party := MissionPartyAs(false, defs.StartWeapon, defs.Bodies, defs.Table)
		ms, err := startMissionFromWith(m, "scenario/10.alm", 10, defs.Table, mapload.DifficultyNormal, party, nil)
		if err != nil || ms.RaiseErr != nil {
			t.Fatal(err, ms.RaiseErr)
		}
		w, hero := ms.World, ms.Start.IDs[0]
		purse := w.Purse(sim.SelfSlot)
		start := ms.Start.Cells[0]
		sim.Step(w, []sim.Command{sim.MoveTo(hero, sim.CellPoint{X: start.X + 3, Y: start.Y})})
		for i := 0; i < 16*20 && w.Outcome() == sim.OutcomeUndecided; i++ {
			sim.Step(w, nil)
		}
		h, _ := w.Entity(hero)
		return w, image.Pt(int(h.X), int(h.Y)), purse
	}
	shipped := x.Editor()
	shipped.Layout(1280, 800)
	if err := shipped.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	w, end, _ := walk(shipped.Document().SourceBytes)
	if w.Outcome() != sim.OutcomeUndecided {
		t.Fatalf("the shipped mission decided %d during the walk", w.Outcome())
	}
	for _, c := range []struct {
		name string
		box  image.Rectangle
		win  bool
	}{{"area on the walk's end", image.Rect(end.X-1, end.Y-1, end.X+1, end.Y+1), true},
		{"area elsewhere", image.Rect(end.X+20, end.Y+20, end.X+24, end.Y+24), false}} {
		t.Run(c.name, func(t *testing.T) {
			e := x.Editor()
			e.Layout(1280, 800)
			if err := e.Open("scenario/10.alm"); err != nil {
				t.Fatal(err)
			}
			closeMapPanel(t, e)
			wizNewTrigger(t, e, "Treasure")
			wizPress(t, e, "", 0)
			wizType(t, e, "c0.p1", fmt.Sprintf("%d,%d,%d,%d", c.box.Min.X, c.box.Min.Y, c.box.Max.X, c.box.Max.Y))
			wizPress(t, e, "+action", 0)
			wizPress(t, e, "+action", 0)
			r, ok := e.FieldRect(wizField(t, e, "a1.item"))
			trigClick(t, e, r, ok, "the second action's item")
			e.ScrollDropdown(-len(e.DropdownChoices()))
			r, ok = e.DropdownRowRect(1)
			trigClick(t, e, r, ok, "the win item")
			wizApply(t, e)
			w, got, purse := walk(e.Document().SourceBytes)
			if !got.In(c.box.Inset(-1)) && c.win || got != end && !c.win {
				t.Fatalf("the walk ends on %v, the shipped map's on %v", got, end)
			}
			won := w.Outcome() == sim.OutcomeWon
			if won != c.win || (w.Purse(sim.SelfSlot) == purse+500) != c.win {
				t.Fatalf("outcome %d, purse %d from %d; want win %v", w.Outcome(), w.Purse(sim.SelfSlot), purse, c.win)
			}
			t.Logf("walk ends on %v; area %v; outcome %d; purse %d from %d", end, c.box, w.Outcome(), w.Purse(sim.SelfSlot), purse)
		})
	}
}
