package sim

import (
	"math"
	"testing"
)

const (
	stcMage  = EntityID(1)
	stcEnemy = EntityID(2)
)

// stcWorld builds a mage walking east along a row on an open map with a hostile
// unit, a lever and a sack in reach of the orders a player gives mid-step.
func stcWorld(t *testing.T) *World {
	t.Helper()
	mage := effectMage(stcMage, 4, 4, 1<<1|1<<18|1<<26)
	mage.Speed, mage.Capacity = 21, 1411
	mage.Mind, mage.MaxMana, mage.Mana = 30, 500, 500
	mage.AttackCharge, mage.AttackRelax = 8, 12
	enemy := spEnt(stcEnemy, 12, 9)
	enemy.Owner = 2
	enemy.HP, enemy.MaxHP = 5000, 5000
	rel := engRel(t, [3]uint32{SelfSlot, 2, relationHostile}, [3]uint32{2, SelfSlot, 2})
	lever := Structure{ID: 0, Col: 8, Row: 8, MaxHealth: 1, Width: 1, Height: 1, Kind: 28, Attach: 1, Blocking: 1}
	w, err := NewStructuredWorld(5, engBounds, ModeCanonical, Terrain{}, []Entity{mage, enemy}, nil, rel,
		[]Sack{{X: 7, Y: 6, Gold: 5}}, nil,
		[]SpellRule{hlArrow(), acbShield(), {ID: 26, ManaCost: 7, MaxRange: 10}}, GhostTemplate{}, []Structure{lever})
	if err != nil {
		t.Fatalf("NewStructuredWorld: %v", err)
	}
	return w
}

// stcFigure is where the figure is drawn, in cells: the cell it entered, less
// the part of the step still owed.
func stcFigure(e Entity) (float64, float64) {
	x, y := float64(e.X), float64(e.Y)
	if e.Transit > 0 && e.Stride.Present && e.TransitTotal > 0 {
		f := float64(e.Transit) / float64(e.TransitTotal)
		x -= float64(e.X-e.Stride.FromX) * f
		y -= float64(e.Y-e.Stride.FromY) * f
	}
	return x, y
}

// stcWalk starts the walk and runs it to the third tick of a step that still
// owes more than half its ticks.
func stcWalk(t *testing.T, w *World) {
	t.Helper()
	Step(w, []Command{MoveTo(stcMage, CellPoint{X: 30, Y: 4})})
	for range 2 {
		Step(w, nil)
	}
	if e := w.entities[0]; e.Transit < 4 {
		t.Fatalf("not mid-step: %d/%d", e.Transit, e.TransitTotal)
	}
}

// stcMaxJump runs ticks, applying cmds on the first, and returns the largest
// distance the figure moved in one tick until the step ends.
func stcMaxJump(w *World, cmds []Command, ticks int) float64 {
	px, py := stcFigure(w.entities[0])
	worst := 0.0
	for k := range ticks {
		if k == 0 {
			Step(w, cmds)
		} else {
			Step(w, nil)
		}
		x, y := stcFigure(w.entities[0])
		worst = math.Max(worst, math.Hypot(x-px, y-py))
		px, py = x, y
		if w.entities[0].Transit == 0 {
			break
		}
	}
	return worst
}

// One tick of ordinary motion is at most 1/10 of a cell at this speed.
const stcOrdinary = 0.1

type stcOrder struct {
	name string
	cmds []Command
}

func stcOrders() []stcOrder {
	return []stcOrder{
		{"move", []Command{MoveTo(stcMage, CellPoint{X: 4, Y: 12})}},
		{"attack", []Command{Attack(stcMage, stcEnemy)}},
		{"cast at a unit", []Command{Cast(stcMage, stcEnemy, 1)}},
		{"self shield", []Command{Cast(stcMage, stcMage, 18)}},
		{"cast at a cell", []Command{CastAt(stcMage, 26, CellPoint{X: 8, Y: 5})}},
		{"cast approach", []Command{CastAt(stcMage, 26, CellPoint{X: 40, Y: 40})}},
		{"structure use", []Command{UseStructure(stcMage, 0)}},
		{"pickup", []Command{PickUp(stcMage, CellPoint{X: 7, Y: 6})}},
		{"stand ground", []Command{GroupStance(stcMage, int32(orderStandGround), 1)}},
		{"guard", []Command{GroupStance(stcMage, int32(orderGuard), 1)}},
		{"group move", []Command{GroupMoveTo(stcMage, CellPoint{X: 4, Y: 12}, 1)}},
	}
}

func TestStepCompletesForEveryOrder(t *testing.T) {
	for _, o := range stcOrders() {
		t.Run(o.name, func(t *testing.T) {
			w := stcWorld(t)
			stcWalk(t, w)
			if j := stcMaxJump(w, o.cmds, 40); j > stcOrdinary {
				e := w.entities[0]
				t.Fatalf("the figure jumped %.3f of a cell; now %d,%d transit %d/%d", j, e.X, e.Y, e.Transit, e.TransitTotal)
			}
		})
	}
}

func stcShielded(w *World) bool {
	_, ok := effectIndex(w.attached, stcMage, 18)
	return ok
}

// A cast ordered mid-step waits for the step's end, then acts in place.
func TestCastOrderedMidStepActsAtTheStepsEnd(t *testing.T) {
	w := stcWorld(t)
	stcWalk(t, w)
	cell := w.entities[0].X
	Step(w, []Command{Cast(stcMage, stcMage, 18)})
	if p := w.entities[0].PendingOrder; p.Kind != PendingActorCast || p.Spell != 18 {
		t.Fatalf("the cast is not the waiting order: %+v", p)
	}
	for w.entities[0].Transit != 0 {
		if len(w.bookCasts) != 0 || w.entities[0].Mana != 500 || stcShielded(w) {
			t.Fatal("the cast acted before the step ended")
		}
		Step(w, nil)
	}
	for k := 0; k < 60 && !stcShielded(w); k++ {
		Step(w, nil)
	}
	e := w.entities[0]
	if !stcShielded(w) || e.Mana != 497 || e.X != cell || e.PendingOrder.Kind != PendingNone || e.HasTarget {
		t.Fatalf("shield %v mana %d cell %d (was %d) pending %+v target %v", stcShielded(w), e.Mana, e.X, cell, e.PendingOrder, e.HasTarget)
	}
}

// An order given later in the same step replaces the waiting cast.
func TestLaterOrderInTheStepReplacesTheWaitingCast(t *testing.T) {
	later := []struct {
		name string
		cmd  Command
		held func(w *World) bool
	}{
		{"cast", Cast(stcMage, stcMage, 18), func(w *World) bool { return stcShielded(w) }},
		{"move", MoveTo(stcMage, CellPoint{X: 4, Y: 12}), func(w *World) bool { return w.entities[0].Y > 4 }},
		{"attack", Attack(stcMage, stcEnemy), func(w *World) bool { return w.entities[0].HasAttackTarget }},
		{"structure use", UseStructure(stcMage, 0), func(w *World) bool { return w.structures[0].Field42 == 1 }},
		{"cast approach", CastAt(stcMage, 26, CellPoint{X: 40, Y: 40}), func(w *World) bool { return w.entities[0].HasTarget && w.entities[0].TargetX == 40 }},
		{"pickup", PickUp(stcMage, CellPoint{X: 7, Y: 6}), func(w *World) bool { return w.entities[0].PendingOrder.Kind == PendingPickup }},
	}
	for _, l := range later {
		t.Run(l.name, func(t *testing.T) {
			w := stcWorld(t)
			stcWalk(t, w)
			Step(w, []Command{CastAt(stcMage, 26, CellPoint{X: 8, Y: 5})})
			if w.entities[0].PendingOrder.Kind != PendingCellCast {
				t.Fatalf("the first cast did not wait: %+v", w.entities[0].PendingOrder)
			}
			Step(w, []Command{l.cmd})
			if l.name == "cast" {
				if p := w.entities[0].PendingOrder; p.Kind != PendingActorCast || p.Spell != 18 {
					t.Fatalf("the later cast did not replace the waiting one: %+v", p)
				}
			} else if p := w.entities[0].PendingOrder.Kind; p == PendingCellCast || p == PendingActorCast {
				t.Fatalf("the waiting cast survived the later %s: %+v", l.name, w.entities[0].PendingOrder)
			}
			mana := w.entities[0].Mana
			for k := 0; k < 60; k++ {
				Step(w, nil)
				if w.entities[0].X == 8 && w.entities[0].Y == 5 {
					t.Fatal("the replaced cell cast released and moved the mage")
				}
			}
			if !l.held(w) && l.name != "pickup" {
				t.Fatalf("the later order did not act")
			}
			if l.name != "cast" && w.entities[0].Mana < mana {
				t.Fatalf("mana spent by a replaced cast: %d -> %d", mana, w.entities[0].Mana)
			}
		})
	}
}

// SAVE with a waiting cast, a cold LOAD, then the same ticks and figure.
func TestWaitingCastSurvivesSaveAndLoadMidStep(t *testing.T) {
	w := stcWorld(t)
	stcWalk(t, w)
	Step(w, []Command{Cast(stcMage, stcMage, 18)})
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if back.entities[0].PendingOrder.Kind != PendingActorCast || back.entities[0].PendingOrder != w.entities[0].PendingOrder || back.Hash() != w.Hash() {
		t.Fatalf("LOAD changed the waiting order: %+v", back.entities[0].PendingOrder)
	}
	for k := 0; k < 80; k++ {
		Step(w, nil)
		Step(&back, nil)
		ax, ay := stcFigure(w.entities[0])
		bx, by := stcFigure(back.entities[0])
		if w.Hash() != back.Hash() || ax != bx || ay != by {
			t.Fatalf("tick %d diverged after LOAD", k)
		}
	}
	if !stcShielded(&back) {
		t.Fatal("the loaded world never cast the waiting spell")
	}
}

// Whichever tick of the walk the cast arrives on, the figure never jumps.
func TestCastOnAnyTickOfTheWalkNeverJumpsTheFigure(t *testing.T) {
	for at := 1; at < 45; at++ {
		w := stcWorld(t)
		Step(w, []Command{MoveTo(stcMage, CellPoint{X: 30, Y: 4})})
		for range at {
			Step(w, nil)
		}
		for _, c := range []Command{Cast(stcMage, stcMage, 18), Cast(stcMage, stcEnemy, 1)} {
			v := stcCopy(t, w)
			if j := stcMaxJump(v, []Command{c}, 3); j > stcOrdinary {
				t.Fatalf("tick %d: the figure jumped %.3f of a cell", at, j)
			}
		}
	}
}

func stcCopy(t *testing.T, w *World) *World {
	t.Helper()
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	return &back
}

// A mage pursuing its victim completes the step the same way.
func TestCastMidStepOfAPursuitCompletesTheStep(t *testing.T) {
	for _, c := range []Command{Cast(stcMage, stcMage, 18), Cast(stcMage, stcEnemy, 1)} {
		w := stcWorld(t)
		Step(w, []Command{Attack(stcMage, stcEnemy)})
		for range 2 {
			Step(w, nil)
		}
		if e := w.entities[0]; e.Transit < 4 || !e.HasAttackTarget {
			t.Fatalf("not mid-step in pursuit: %d/%d", e.Transit, e.TransitTotal)
		}
		if j := stcMaxJump(w, []Command{c}, 40); j > stcOrdinary {
			t.Fatalf("the figure jumped %.3f of a cell", j)
		}
		for k := 0; k < 80 && w.entities[0].Mana == 500; k++ {
			Step(w, nil)
		}
		if w.entities[0].Mana == 500 {
			t.Fatal("the waiting cast never acted")
		}
	}
}
