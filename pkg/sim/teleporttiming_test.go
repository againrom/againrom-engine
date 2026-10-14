package sim

import "testing"

// teleportTimingSteps admits one Teleport and steps n more ticks, reporting
// each tick: index 0 is the admitting tick S.
func teleportTimingSteps(t *testing.T, w *World, cmd Command, n int) []Report {
	t.Helper()
	out := []Report{StepReported(w, []Command{cmd})}
	for range n {
		out = append(out, StepReported(w, nil))
	}
	return out
}

// TestATeleportRelocatesWhenItsCountdownReachesZero is MAGIC-TELEPORT-370 and
// MAGIC-TELEPORT-371: the admitting tick S reports the one cast and moves
// nothing; the caster stays on its cell through S+charge-1 and moves on
// S+charge, which reports the relocation and no second cast.
func TestATeleportRelocatesWhenItsCountdownReachesZero(t *testing.T) {
	for _, charge := range []int32{8, 11} {
		for _, form := range []string{"cell", "unit"} {
			w := teleportHiddenWorld(t, -1)
			w.entities[0].AttackCharge = charge
			cmd := CastAt(1, 26, CellPoint{X: 6, Y: 3})
			if form == "unit" {
				beacon := spEnt(2, 6, 3)
				beacon.Domain = DomainAir
				w.entities = append(w.entities, beacon)
				w.routes = append(w.routes, nil)
				cmd = Cast(1, 2, 26)
			}
			reports := teleportTimingSteps(t, w, cmd, int(charge)+2)
			if c := reports[0].Casts; len(c) != 1 || c[0].Spell != 26 || c[0].FromX != 2 || c[0].ToX != 6 || c[0].AtCell != (form == "cell") {
				t.Fatalf("%s charge %d: admitting tick reported %+v, want the one Teleport from (2,3) to (6,3)", form, charge, c)
			}
			for k, r := range reports {
				if k > 0 && len(r.Casts) != 0 {
					t.Errorf("%s charge %d: tick S+%d reported a second cast %+v", form, charge, k, r.Casts)
				}
				if moved := len(r.Relocations) == 1 && r.Relocations[0].ToX == 6; moved != (k == int(charge)) {
					t.Errorf("%s charge %d: tick S+%d relocations %+v, want one only on S+%d", form, charge, k, r.Relocations, charge)
				}
			}
			if got := spAt(t, w, 1); got.X != 6 || got.Y != 3 || got.Mana != 140 {
				t.Fatalf("%s charge %d: caster at (%d,%d) with mana %d, want (6,3) and 140", form, charge, got.X, got.Y, got.Mana)
			}
		}
	}
}

// TestATeleportSavedInItsCountdownRelocatesOnTheSameTick: a SAVE in the
// countdown carries the cast in flight in the action continuation the SAV
// writer stores, and in the native byte form; either LOAD relocates the
// caster on the tick the uninterrupted world does.
func TestATeleportSavedInItsCountdownRelocatesOnTheSameTick(t *testing.T) {
	const charge = 8
	for after := 1; after < charge; after++ {
		live := teleportHiddenWorld(t, -1)
		live.entities[0].AttackCharge = charge
		StepReported(live, []Command{CastAt(1, 26, CellPoint{X: 6, Y: 3})})
		for range after - 1 {
			Step(live, nil)
		}
		native := worldRoundTripForTest(t, live)
		carried := teleportHiddenWorld(t, -1)
		carried.entities[0].AttackCharge = charge
		carried.tick, carried.entities[0] = live.tick, live.entities[0]
		if err := carried.RestoreActions(live.Actions(), nil); err != nil {
			t.Fatalf("after %d: RestoreActions: %v", after, err)
		}
		for _, w := range []*World{live, native, carried} {
			for k := after; k < charge+2; k++ {
				r := StepReported(w, nil)
				if moved := len(r.Relocations) == 1; moved != (k == charge) {
					t.Fatalf("saved %d ticks after admission: tick S+%d relocated=%v, want only S+%d", after, k, moved, charge)
				}
			}
		}
		if live.Hash() != native.Hash() {
			t.Fatalf("saved %d ticks after admission: native reload diverged", after)
		}
		if got := spAt(t, carried, 1); got.X != 6 || got.Mana != 140 {
			t.Fatalf("saved %d ticks after admission: continuation reload ended at x=%d mana %d", after, got.X, got.Mana)
		}
	}
}

// TestATurningTeleportIsAdmittedWhenItsCasterFacesTheDestination is MOVE-107,
// MAGIC-340 and MAGIC-TELEPORT-370: the cast is admitted only once the caster
// faces the destination, and the countdown runs from that admission. For a
// destination ahead, to the side and behind a caster with a nonzero turn rate,
// the one cast is reported on the tick the caster's facing reaches the
// heading, and the relocation follows exactly charge ticks later.
func TestATurningTeleportIsAdmittedWhenItsCasterFacesTheDestination(t *testing.T) {
	const charge = 8
	for _, rate := range []int32{8, 22} {
		for _, to := range []CellPoint{{X: 2, Y: 0}, {X: 5, Y: 0}, {X: 6, Y: 3}, {X: 6, Y: 6}, {X: 2, Y: 6}} {
			w := teleportHiddenWorld(t, -1)
			w.entities[0].AttackCharge, w.entities[0].RotationSpeed = charge, rate
			heading := cellHeading(2, 3, to.X, to.Y)
			reports := teleportTimingSteps(t, w, CastAt(1, 26, to), 40)
			presented, relocated := -1, -1
			for k, r := range reports {
				for range r.Casts {
					if presented >= 0 {
						t.Fatalf("rate %d to %v: a second cast on S+%d", rate, to, k)
					}
					presented = k
				}
				if len(r.Relocations) == 1 && r.Relocations[0].ToX == to.X && r.Relocations[0].ToY == to.Y {
					relocated = k
				}
			}
			if presented < 0 || relocated != presented+charge {
				t.Fatalf("rate %d to %v: cast reported on S+%d, relocation on S+%d; want the relocation %d ticks after the cast", rate, to, presented, relocated, charge)
			}
			// Replay to the presentation tick: the caster faces the heading
			// there and on no earlier tick.
			w = teleportHiddenWorld(t, -1)
			w.entities[0].AttackCharge, w.entities[0].RotationSpeed = charge, rate
			teleportTimingSteps(t, w, CastAt(1, 26, to), presented)
			if got := spAt(t, w, 1).Facing; got != heading {
				t.Fatalf("rate %d to %v: the cast was reported on S+%d with facing %d, want the heading %d", rate, to, presented, got, heading)
			}
			if presented > 0 {
				w = teleportHiddenWorld(t, -1)
				w.entities[0].AttackCharge, w.entities[0].RotationSpeed = charge, rate
				teleportTimingSteps(t, w, CastAt(1, 26, to), presented-1)
				if got := spAt(t, w, 1).Facing; got == heading {
					t.Fatalf("rate %d to %v: the caster faced the heading on S+%d, before the cast was reported on S+%d", rate, to, presented-1, presented)
				}
			}
			t.Logf("rate %d to %v: heading %d, cast S+%d, relocation S+%d", rate, to, heading, presented, relocated)
		}
	}
}
