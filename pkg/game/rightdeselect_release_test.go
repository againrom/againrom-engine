package game

import (
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
)

func TestReleaseRightClickAfterBookCastDeselectsTheMage(t *testing.T) {
	path, _ := ownerCurrentSAV(t, "mission111-portrait-input")
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	a := f.App("Right click deselect")
	defer a.StopAudio()
	_, list, load := f.SaveSeams(SaveStore{Dir: filepath.Dir(path)}, OriginalStore{}, nil)
	a.SetSaveSeams(nil, list, load)
	groundAppLoad(t, a, list, filepath.Base(path))
	a.Layout(1024, 768)
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 16 && a.HeadlessNoticeOpen(); i++ {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	live, w := f.live, f.live.world
	const id, targetX, targetY = sim.EntityID(84), int32(61), int32(111)
	caster, ok := live.entity(id)
	if !ok || caster.X != 58 || caster.Y != 116 || caster.ScanRange != 6 {
		t.Fatal("Owner fixture no longer supplies the reported caster")
	}
	if w.Sight(sim.SelfSlot)[targetY*w.Bounds().Width+targetX] == 0 {
		t.Fatal("Target must be visible to the party")
	}
	var rule sim.SpellRule
	for _, r := range w.Spells() {
		if r.ID == 2 {
			rule = r
			break
		}
	}
	if got := sim.SpellCharacteristicsFor(sim.Rules{}, caster, rule).Range; got != 12 {
		t.Fatalf("Owner's cached Fireball range=%d, want12", got)
	}
	book, _ := selectedSpellbook(sim.Rules{}, []sim.Entity{caster}, w.Spells(), live.spellNames, live.view.Words(), nil)
	for _, entry := range book {
		if entry.ID == 2 && (!entry.PointTarget || entry.Unavailable) {
			t.Fatal("Installed Fireball did not retain its point-target type")
		}
		if entry.ID == 1 && entry.PointTarget {
			t.Fatal("Installed Fire Arrow lost its unit-target type")
		}
	}
	inspectionCentre(live, int(caster.X), int(caster.Y))
	live.push()
	if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	live.push()
	if _, _, err := a.HeadlessSpellPoint(2); err != nil {
		if err := a.HeadlessKey("book"); err != nil {
			t.Fatal(err)
		}
	}
	// Restored HUD preferences become visible on an ordinary idle frame.
	if err := a.HeadlessPointer("hover", -1, -1); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		x, y, err := a.HeadlessSpellPoint(2)
		if err != nil {
			t.Fatal(err)
		}
		for _, edge := range []string{"press", "release"} {
			if err := a.HeadlessPointer(edge, x, y); err != nil {
				t.Fatal(err)
			}
		}
		if _, selected, armed := live.view.QuickSpellState(); selected == 2 && armed {
			break
		}
	}
	if _, selected, armed := live.view.QuickSpellState(); selected != 2 || !armed {
		t.Fatal("The book click did not arm Fireball")
	}
	inspectionCentre(live, int(targetX), int(targetY))
	pressDropCell(t, a, int(targetX), int(targetY))
	if len(live.pending) != 1 || live.pending[0].Kind != sim.KindCastAt ||
		live.pending[0].Spell != 2 || live.pending[0].X != targetX || live.pending[0].Y != targetY {
		t.Fatalf("Ordinary pointer produced wrong order: %+v", live.pending)
	}
	if sel := a.HeadlessSelection(); len(sel) != 1 || sel[0] != uint32(id) {
		t.Fatalf("Before the right click the selection is %v, want the mage", sel)
	}
	if err := a.HeadlessPointer("right-press", 400, 300); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("right-release", 400, 300); err != nil {
		t.Fatal(err)
	}
	if sel := a.HeadlessSelection(); len(sel) != 0 {
		t.Fatalf("The right click after a book cast left the selection %v, want empty", sel)
	}
}
