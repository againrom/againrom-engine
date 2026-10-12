package game

import (
	"image"
	"testing"

	"againrom/pkg/sim"
)

// clickManaWorld is a mage with a full pool beside two hostile targets and
// three installed book rules, all of which pay when their wind-up starts.
func clickManaWorld(t *testing.T, casterX int32) (*mapWorld, *sim.World) {
	t.Helper()
	caster := sim.Entity{ID: 1, X: casterX, Y: 2, HP: 100, MaxHP: 100, Owner: sim.SelfSlot,
		Mind: 30, Mana: 100, MaxMana: 100, KnownSpells: 1<<1 | 1<<3 | 1<<26, AttackCharge: 12, AttackRelax: 4,
		ScanRange: 19}
	first := sim.Entity{ID: 2, X: 15, Y: 2, HP: 1000, MaxHP: 1000, Owner: 2, DyingTime: 200}
	second := sim.Entity{ID: 3, X: 15, Y: 4, HP: 1000, MaxHP: 1000, Owner: 2, DyingTime: 200}
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 64, Height: 16}, sim.ModeCanonical, nil,
		[]sim.Entity{caster, first, second}, nil, []sim.SpellRule{
			{ID: 1, ManaCost: 10, School: 1, MaxRange: 40, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true, Delivery: 2, EffectSpeed: 128},
			{ID: 3, ManaCost: 7, School: 1, MaxRange: 40, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true, Delivery: 2, EffectSpeed: 128},
			{ID: 26, ManaCost: 5, MaxRange: 6, Delivery: 1},
		})
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	mw := &mapWorld{world: w, swing: map[sim.EntityID]int{}, phase: map[sim.EntityID]sim.AttackPhase{},
		castRun: map[sim.EntityID]castRun{}, projectiles: sbSheets(), commanded: map[sim.EntityID]bool{},
		prev: map[sim.EntityID]image.Point{}, died: map[sim.EntityID]int{}, walk: map[sim.EntityID]walkClock{},
		fog: newFogPlane(64, 16)}
	return mw, w
}

// clickFrame advances one frame through the queue a click fills.
func clickFrame(mw *mapWorld) {
	cmds := mw.commands()
	mw.pending = mw.pending[:0]
	mw.commanded = map[sim.EntityID]bool{}
	mw.observeCasts(sim.StepObserved(mw.world, cmds))
}

func clickEntity(t *testing.T, w *sim.World, id sim.EntityID) sim.Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("entity %d absent", id)
	return sim.Entity{}
}

// firstHit runs frames until the target loses hit points and returns that frame.
func firstHit(t *testing.T, mw *mapWorld, target sim.EntityID, from int) int {
	t.Helper()
	for frame := from; frame < from+256; frame++ {
		if clickEntity(t, mw.world, target).HP < 1000 {
			return frame
		}
		clickFrame(mw)
	}
	t.Fatal("the cast never landed")
	return 0
}

// firstRelease runs frames until a delivery is in flight and returns that frame.
func firstRelease(mw *mapWorld, from int) int {
	frame := from
	for ; frame < from+256 && mw.world.PendingSpellDeliveries() == 0; frame++ {
		clickFrame(mw)
	}
	return frame
}

func TestRepeatedSpellClicksSpendTheCostOnceAndKeepTheWindUp(t *testing.T) {
	single, sw := clickManaWorld(t, 2)
	single.attackOrCast(1, 2, 1, 0, 0, false)
	clickFrame(single)
	if m := clickEntity(t, sw, 1).Mana; m != 90 {
		t.Fatalf("one click: mana %d, want 90", m)
	}
	want := firstHit(t, single, 2, 1)

	mw, w := clickManaWorld(t, 2)
	for frame := 0; frame < 6; frame++ {
		mw.attackOrCast(1, 2, 1, 0, 0, false)
		clickFrame(mw)
		if m := clickEntity(t, w, 1).Mana; m != 90 {
			t.Fatalf("click %d: mana %d, want 90 (one cost for one animation)", frame+1, m)
		}
	}
	if got := firstHit(t, mw, 2, 6); got != want {
		t.Fatalf("repeated clicks moved the landing to frame %d, want %d", got, want)
	}
	if m := clickEntity(t, w, 1).Mana; m != 90 {
		t.Fatalf("after release: mana %d, want 90", m)
	}
}

func TestRepeatedCellSpellClicksSpendTheCostOnce(t *testing.T) {
	mw, w := clickManaWorld(t, 2)
	for frame := 0; frame < 6; frame++ {
		mw.attackOrCast(1, 0, 26, 6, 2, true)
		clickFrame(mw)
		if m := clickEntity(t, w, 1).Mana; m != 95 {
			t.Fatalf("cell click %d: mana %d, want 95", frame+1, m)
		}
	}
	for frame := 0; frame < 64 && clickEntity(t, w, 1).X != 6; frame++ {
		clickFrame(mw)
	}
	if e := clickEntity(t, w, 1); e.X != 6 || e.Mana != 95 {
		t.Fatalf("Teleport after repeated clicks: at %d with mana %d, want 6 and 95", e.X, e.Mana)
	}
}

func TestSpellClicksWhileWalkingToACellSpendOnceAtTheWindUp(t *testing.T) {
	mw, w := clickManaWorld(t, 2)
	charged := -1
	for frame := 0; frame < 96 && clickEntity(t, w, 1).X != 14; frame++ {
		if charged < 0 {
			mw.attackOrCast(1, 0, 26, 14, 2, true)
		}
		clickFrame(mw)
		m := clickEntity(t, w, 1).Mana
		if charged < 0 && m != 100 {
			charged = frame
		}
		if charged >= 0 && m != 95 {
			t.Fatalf("frame %d: mana %d, want one cost of 5", frame, m)
		}
	}
	if e := clickEntity(t, w, 1); charged < 0 || e.X != 14 || e.Mana != 95 {
		t.Fatalf("approach Teleport: at %d with mana %d (wind-up frame %d)", e.X, e.Mana, charged)
	}
}

// Once the mana is spent the cast completes. Another cast clicked during the
// paid wind-up waits: the first cast lands, then the second order runs.
func TestASpellClickDuringAPaidWindUpRunsAfterTheCastLands(t *testing.T) {
	for _, tc := range []struct {
		name          string
		victim, spell uint32
		after         int32
	}{{"other target", 3, 1, 80}, {"other spell", 2, 3, 83}} {
		t.Run(tc.name, func(t *testing.T) {
			single, sw := clickManaWorld(t, 2)
			single.attackOrCast(1, 2, 1, 0, 0, false)
			clickFrame(single)
			want := firstRelease(single, 1)
			firstHit(t, single, 2, want)
			hp := clickEntity(t, sw, 2).HP

			mw, w := clickManaWorld(t, 2)
			mw.attackOrCast(1, 2, 1, 0, 0, false)
			clickFrame(mw)
			clickFrame(mw)
			mw.attackOrCast(1, tc.victim, tc.spell, 0, 0, false)
			clickFrame(mw)
			if m := clickEntity(t, w, 1).Mana; m != 90 {
				t.Fatalf("queued click: mana %d, want 90", m)
			}
			// A second delivery in flight moves the first one's landing by a
			// tick, as a fresh click after the release does; the release frame
			// is the cast's own clock.
			if got := firstRelease(mw, 3); got != want || clickEntity(t, w, 1).Mana != 90 {
				t.Fatalf("first cast released at frame %d, want %d; mana %d", got, want, clickEntity(t, w, 1).Mana)
			}
			firstHit(t, mw, 2, want)
			if clickEntity(t, w, 2).HP != hp {
				t.Fatalf("first cast hp %d, want %d", clickEntity(t, w, 2).HP, hp)
			}
			for frame := 0; frame < 256 && clickEntity(t, w, 1).Mana != tc.after; frame++ {
				clickFrame(mw)
			}
			if m := clickEntity(t, w, 1).Mana; m != tc.after {
				t.Fatalf("second order never started: mana %d, want %d", m, tc.after)
			}
			target := sim.EntityID(tc.victim)
			before := clickEntity(t, w, target).HP
			for frame := 0; frame < 256 && clickEntity(t, w, target).HP == before; frame++ {
				clickFrame(mw)
			}
			if clickEntity(t, w, target).HP == before || clickEntity(t, w, 1).Mana != tc.after {
				t.Fatalf("second cast: target hp %d, mana %d", clickEntity(t, w, target).HP, clickEntity(t, w, 1).Mana)
			}
		})
	}
}

// A move clicked during a paid wind-up waits too: the cast lands, then the
// caster walks.
func TestAMoveClickDuringAPaidWindUpWalksAfterTheCastLands(t *testing.T) {
	single, _ := clickManaWorld(t, 2)
	single.attackOrCast(1, 2, 1, 0, 0, false)
	clickFrame(single)
	want := firstHit(t, single, 2, 1)

	mw, w := clickManaWorld(t, 2)
	mw.attackOrCast(1, 2, 1, 0, 0, false)
	clickFrame(mw)
	mw.enqueue(1, 2, 9)
	clickFrame(mw)
	if e := clickEntity(t, w, 1); e.Mana != 90 || e.X != 2 || e.Y != 2 {
		t.Fatalf("move click: at (%d,%d) mana %d", e.X, e.Y, e.Mana)
	}
	got := firstHit(t, mw, 2, 2)
	if got != want || clickEntity(t, w, 1).Mana != 90 {
		t.Fatalf("cast landed at frame %d, want %d; mana %d", got, want, clickEntity(t, w, 1).Mana)
	}
	for frame := 0; frame < 256 && clickEntity(t, w, 1).Y != 9; frame++ {
		clickFrame(mw)
	}
	if e := clickEntity(t, w, 1); e.X != 2 || e.Y != 9 || e.Mana != 90 {
		t.Fatalf("walk after the cast: at (%d,%d) mana %d", e.X, e.Y, e.Mana)
	}
}

func TestSpellClicksDuringRecoverySpendOncePerAnimation(t *testing.T) {
	mw, w := clickManaWorld(t, 2)
	mw.attackOrCast(1, 2, 1, 0, 0, false)
	clickFrame(mw)
	for frame := 0; frame < 64 && clickEntity(t, w, 1).CastWait == 0; frame++ {
		clickFrame(mw)
	}
	if e := clickEntity(t, w, 1); e.CastWait == 0 || e.Mana != 90 {
		t.Fatalf("first cast: recovery %d, mana %d", e.CastWait, e.Mana)
	}
	for frame := 0; frame < 4; frame++ {
		mw.attackOrCast(1, 2, 1, 0, 0, false)
		clickFrame(mw)
		if m := clickEntity(t, w, 1).Mana; m != 80 {
			t.Fatalf("recovery click %d: mana %d, want 80 (one new animation)", frame+1, m)
		}
	}
}
