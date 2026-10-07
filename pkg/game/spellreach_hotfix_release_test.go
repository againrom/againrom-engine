package game

import (
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
)

func TestReleaseFireballUsesPartyVisiblePointFromOwnerSave(t *testing.T) {
	path, _ := groundCorpusFile(t, "2026-09-15/mission111-portrait-input.ags",
		"522d5cd01dea60a6d64637884f0427b58888c329be3458259c14d17bce5afe35")
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	a := f.App("Fireball reach")
	defer a.StopAudio()
	_, list, load := agsSaveSeams(f, SaveStore{Dir: filepath.Dir(path)}, OriginalStore{}, nil)
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
	entryPointer1084(t, a, int(targetX), int(targetY))
	if len(live.pending) != 1 || live.pending[0].Kind != sim.KindCastAt ||
		live.pending[0].Spell != 2 || live.pending[0].X != targetX || live.pending[0].Y != targetY {
		t.Fatalf("Ordinary pointer produced wrong order: %+v", live.pending)
	}
	live.tick()
	got, _ := live.entity(id)
	wantMana := caster.Mana - int32(caster.Book.Slots[1].ManaCost)
	if got.Mana != wantMana || got.X != caster.X || got.Y != caster.Y {
		t.Fatalf("Party-visible Fireball did not begin in place: mana%d want%d cell%d,%d", got.Mana, wantMana, got.X, got.Y)
	}
	t.Logf("owner mission111: point61,111 Range12 Scan6, book/pointer admission paid%d mana in place", caster.Mana-got.Mana)
}
