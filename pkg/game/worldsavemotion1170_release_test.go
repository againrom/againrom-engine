package game

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func world1170MotionContinuation(t *testing.T, mode string) {
	worldMotionContinuation(t, mode)
}

// Use the actual mission MapOrder and sample before SAVE. The AGS control is
// encoded, decoded and restored into another FrontEnd; it is not a World clone
// or a comparison between projected Documents.
func worldMotionContinuation(t *testing.T, mode string) {
	t.Helper()
	turning, healing, savOnlyHealing := mode == "turn", mode == "heal" || mode == "heal_sav", mode == "heal_sav"
	f := releaseFront(t)
	_, raw := groundCorpusFile(t, "2026-08-15/game0016.sav", "5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345")
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal(err)
	}
	app := f.App("current motion continuation")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	party, err := source.Party()
	if err != nil {
		t.Fatal(err)
	}
	var key uint32
	for _, p := range party {
		if p.Hero {
			key = p.Key
		}
	}
	var hero sim.Entity
	for _, e := range f.live.world.Entities() {
		if e.SourceBinding.Identity == key {
			hero = e
		}
	}
	if hero.SourceBinding.RuntimeID == 0 {
		t.Fatal("authentic hero missing")
	}
	if healing {
		for range 16 {
			f.live.tick()
		}
		if savOnlyHealing {
			f.live.pending = append(f.live.pending, sim.Command{Kind: sim.KindDropCarried, Entity: hero.ID, Spell: 0, X: hero.X + 1, Y: hero.Y})
		}
		headlessDamage(t, f.live.world, hero.ID, 3)
		f.live.tick()
		hero = world1170Entity(t, f, hero.SourceBinding.RuntimeID)
	}
	_, _, order, _, _, _, _, _, _, _, err := open()
	if err != nil {
		t.Fatal(err)
	}
	order(uint32(hero.ID), int(hero.X+3), int(hero.Y))
	for range 100 {
		f.live.tick()
		hero = world1170Entity(t, f, hero.SourceBinding.RuntimeID)
		if turning && hero.Turning() && hero.TurnRemaining <= hero.TurnTotal/2 || !turning && hero.Stride.Present && hero.Transit > 1 && (!healing || world1170NativeTurnsFinished(f.live.world)) {
			break
		}
	}
	if turning && !hero.Turning() || !turning && (!hero.Stride.Present || hero.Transit <= 1) {
		t.Fatal("ordinary MapOrder did not reach the requested motion cut")
	}
	var native *FrontEnd
	var encoded []byte
	if !savOnlyHealing {
		s, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err = EncodeSave(s, "explicit native continuation control")
		if err != nil {
			t.Fatal(err)
		}
		native = world1170NativeMotionLoad(t, encoded)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	ext := ".sav"
	name, err := save(true)
	if err != nil || !strings.HasSuffix(name, ext) {
		t.Fatalf("ordinary motion SAVE %q, want %s: %v", name, ext, err)
	}
	cold, _ := loadSAVWindow(t, store, name)
	if savOnlyHealing {
		assertSackIdentityMatch(t, f.live.world, cold.live.world, "first ordinary SAV cold load")
	}
	clear(raw)
	clear(encoded)
	var second *FrontEnd
	goal := [2]int32{hero.TargetX*256 + 128, hero.TargetY*256 + 128}
	liveArrival, coldArrival, different := -1, -1, 0
	healStep, healed := -1, int32(0)
	t.Logf("runtime %d: drawn facing %d, native turn %d/%d, crossing %d/%d, goal %v", hero.SourceBinding.RuntimeID, hero.DrawnFacing(), hero.TurnRemaining, hero.TurnTotal, hero.Transit, hero.TransitTotal, goal)
	for step := 0; step <= 160; step++ {
		a := world1170Entity(t, f, hero.SourceBinding.RuntimeID)
		b := world1170Entity(t, cold, hero.SourceBinding.RuntimeID)
		apos, bpos := world1170Position(f.live.world, a), world1170Position(cold.live.world, b)
		if native != nil && f.live.world.Hash() != native.live.world.Hash() {
			t.Fatalf("true AGS cold control lost complete World at sample %d", step)
		}
		if a.TurnRemaining != b.TurnRemaining || a.TurnTotal != b.TurnTotal || a.Transit != b.Transit || a.TransitTotal != b.TransitTotal || a.ActionClock != b.ActionClock || a.HP != b.HP || a.Mana != b.Mana {
			t.Fatalf("current action/clock/pools differ at sample %d: turn %d/%d transit %d/%d clock %+v/%+v HP%d/%d Mana%d/%d", step, a.TurnRemaining, b.TurnRemaining, a.Transit, b.Transit, a.ActionClock, b.ActionClock, a.HP, b.HP, a.Mana, b.Mana)
		}
		if apos != bpos || a.DrawnFacing() != b.DrawnFacing() {
			if different == 0 {
				t.Errorf("current motion sample %d: position %v/%v, drawn facing %d/%d", step, apos, bpos, a.DrawnFacing(), b.DrawnFacing())
			}
			different++
		}
		if apos == goal && liveArrival < 0 {
			liveArrival = step
		}
		if bpos == goal && coldArrival < 0 {
			coldArrival = step
		}
		// A second ordinary SAV after the first boundary must retain the
		// remaining crossing as well. No source store exists in either cold
		// FrontEnd, and no new movement order replaces the one under test.
		if step == 7 {
			secondSave, _, _ := cold.SaveSeams(store, OriginalStore{}, nil)
			name, err := secondSave(true)
			if err != nil || !strings.HasSuffix(name, ".sav") {
				t.Fatalf("second current crossing SAV %q: %v", name, err)
			}
			second, _ = loadSAVWindow(t, store, name)
			if savOnlyHealing {
				assertSackIdentityMatch(t, cold.live.world, second.live.world, "second ordinary SAV cold load")
			}
		}
		if second != nil {
			c := world1170Entity(t, second, hero.SourceBinding.RuntimeID)
			if apos != world1170Position(second.live.world, c) || a.DrawnFacing() != c.DrawnFacing() {
				t.Fatalf("second SAV motion differs at sample %d", step)
			}
		}
		if step == 160 {
			break
		}
		f.live.tickWithCastSink(func(events []sim.CastEvent) {
			for _, event := range events {
				if event.Target == hero.ID && event.Spell == 6 && event.HealthRestored > 0 {
					if healStep < 0 {
						healStep = step + 1
					}
					healed += event.HealthRestored
				}
			}
		})
		cold.live.tick()
		if native != nil {
			native.live.tick()
		}
		if second != nil {
			second.live.tick()
		}
	}
	if liveArrival < 0 || coldArrival != liveArrival {
		t.Errorf("full route arrival live %d cold %d", liveArrival, coldArrival)
	}
	if healing {
		if healStep != 7 || healed != 3 {
			t.Fatalf("real pending Heal control changed: first step %d, health restored %d", healStep, healed)
		}
		t.Logf("ordinary SAV preserves the pending Heal: first release at continuation step %d, restored %d HP", healStep, healed)
	}
	if native != nil {
		t.Logf("161 samples through ordinary %s: differences %d, arrival live %d cold %d; true AGS control preserves full World throughout", ext, different, liveArrival, coldArrival)
	} else {
		t.Logf("161 samples through ordinary SAV: differences %d, arrival live %d cold %d", different, liveArrival, coldArrival)
	}
}

func assertSackIdentityMatch(t *testing.T, left, right *sim.World, stage string) {
	t.Helper()
	leftSack, leftContainer, leftItem := savedSackRecords(t, left, stage+" live")
	rightSack, rightContainer, rightItem := savedSackRecords(t, right, stage+" cold")
	if leftSack != rightSack {
		t.Fatalf("%s Sack 44 differs: live=%+v cold=%+v", stage, leftSack, rightSack)
	}
	if !reflect.DeepEqual(leftContainer, rightContainer) {
		t.Fatalf("%s Sack 44 contents differ: live=%+v cold=%+v", stage, leftContainer, rightContainer)
	}
	if !reflect.DeepEqual(leftItem, rightItem) {
		t.Fatalf("%s Item 19 identity/value differs: live=%+v cold=%+v", stage, leftItem, rightItem)
	}
}

func savedSackRecords(t *testing.T, world *sim.World, stage string) (sim.SavedSackObject, sim.SavedObjectContainer, sim.SavedItemObject) {
	t.Helper()
	registry := world.SavedObjects()
	if registry == nil || len(registry.Sacks) != 5 || !slices.Contains(registry.SackRoots, sim.SavedObjectID(44)) {
		t.Fatalf("%s lost the fifth native Sack root: registry=%+v", stage, registry)
	}
	var sack *sim.SavedSackObject
	for i := range registry.Sacks {
		if registry.Sacks[i].ID == 44 {
			candidate := registry.Sacks[i]
			sack = &candidate
			break
		}
	}
	if sack == nil || sack.Retired {
		t.Fatalf("%s Sack 44 is missing or retired: %+v", stage, sack)
	}
	container, ok := savedSackContainer(registry, 44)
	if !ok || !container.Present || len(container.Items) != 1 || container.Items[0] != 19 {
		t.Fatalf("%s Sack 44 lost Item 19 contents: %+v, present=%v", stage, container, ok)
	}
	var item *sim.SavedItemObject
	for i := range registry.Items {
		if registry.Items[i].ID == 19 {
			candidate := registry.Items[i]
			item = &candidate
			break
		}
	}
	if item == nil {
		t.Fatalf("%s Sack 44 references missing Item 19 registry row", stage)
	}
	return *sack, container, *item
}

func TestReleaseCurrentWorldCombinedDropDamageHealSAV(t *testing.T) {
	worldMotionContinuation(t, "heal_sav")
}

func world1170NativeMotionLoad(t *testing.T, encoded []byte) *FrontEnd {
	t.Helper()
	s, _, err := DecodeSave(encoded)
	if err != nil {
		t.Fatal(err)
	}
	cold := releaseFront(t)
	cold.SetDeterministicFrames(true)
	open, town, err := cold.Restore(s)
	if err != nil || town {
		t.Fatal(err)
	}
	app := cold.App("true native cold continuation")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	return cold
}

func world1170NativeTurnsFinished(w *sim.World) bool {
	for _, e := range w.Entities() {
		if e.Turning() {
			return false
		}
	}
	return true
}
