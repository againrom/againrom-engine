package sim

import "testing"

// castFacingWorld is a computer mage (owner 2) at (8,8) facing north with a
// turn rate of 4, knowing Fire Arrow, and its victim 2 at (8,10), straight
// behind it, in the second game or the first.
func castFacingWorld(t *testing.T, second bool) *World {
	t.Helper()
	mage := spMage(1, 8, 8, 30, 100, 100, 1<<1)
	mage.Owner, mage.Speed, mage.AttackCharge, mage.AttackRelax = 2, 30, 8, 4
	mage.RotationSpeed, mage.Facing, mage.DesiredFacing = 4, 0, 0
	victim := spEnt(2, 8, 10)
	victim.Owner, victim.HP, victim.MaxHP = 1, 500, 500
	rel := engRel(t, [3]uint32{1, 2, relationHostile}, [3]uint32{2, 1, relationHostile})
	w := hlWorld(t, 7, rel, []SpellRule{hlArrow()}, mage, victim)
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	return w
}

// castFacingRelease steps w up to n ticks and returns the caster's facing,
// drawn facing and wanted heading at its next release, the tick the cast
// animation starts.
func castFacingRelease(t *testing.T, w *World, n int) (facing, drawn, want uint8) {
	t.Helper()
	for range n {
		for _, ev := range StepObserved(w, nil) {
			if ev.Caster == 1 && ev.Spell == 1 {
				e := w.entities[0]
				return e.Facing, e.DrawnFacing(), w.headingBetween(e, w.entities[1])
			}
		}
	}
	t.Fatalf("no Fire Arrow released in %d ticks", n)
	return
}

// A computer caster faces its target when its cast starts (MOVE-107,
// MAGIC-340): the first cast turns it from north to the victim behind it,
// and the retained order's next cast turns it again after the victim moved
// to its other side. Before, the first game's group pass cut the first turn
// and the cast charged on; the second game's re-entry cast without a turn.
func TestAComputerCasterFacesItsTargetWhenItsCastStarts(t *testing.T) {
	for _, second := range []bool{false, true} {
		name := map[bool]string{false: "first game", true: "second game"}[second]
		t.Run(name, func(t *testing.T) {
			w := castFacingWorld(t, second)
			if !w.orderAttack(0, 2) {
				t.Fatal("fixture: the engage cast nothing")
			}
			if f, d, want := castFacingRelease(t, w, 80); f != want || d != want {
				t.Fatalf("first cast: facing %d drawn %d, want %d", f, d, want)
			}
			w.entities[1].Y = 6
			if f, d, want := castFacingRelease(t, w, 120); f != want || d != want || want != 0 {
				t.Fatalf("re-entered cast: facing %d drawn %d, want %d (north)", f, d, want)
			}
		})
	}
}

// castFacingManualHash is the first-game World hash after a player's unit
// cast at a victim behind its caster, taken before the cast facing gate: a
// cast whose turn the admission already starts runs tick for tick as before.
const castFacingManualHash = 0xc8eb1e73484b1639

// Control: a first-game player's cast at a unit behind it turns at admission
// and releases facing it, and the run hashes as before.
func TestAFirstGamePlayerCastTurnsAsBefore(t *testing.T) {
	w := castFacingWorld(t, false)
	w.entities[0].Owner, w.entities[1].Owner = SelfSlot, 2
	cmds := []Command{Cast(1, 2, 1)}
	for k := range 60 {
		var c []Command
		if k == 0 {
			c = cmds
		}
		StepObserved(w, c)
	}
	if e := w.entities[0]; e.Facing != w.headingBetween(e, w.entities[1]) {
		t.Fatalf("facing %d, want %d", e.Facing, w.headingBetween(e, w.entities[1]))
	}
	if got := w.Hash(); got != castFacingManualHash {
		t.Fatalf("first-game hash %#x, want %#x", got, uint64(castFacingManualHash))
	}
}
