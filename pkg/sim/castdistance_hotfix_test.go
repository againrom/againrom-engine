package sim

import (
	"fmt"
	"testing"
)

// castDistanceRow: Fire Arrow or Drain Life, range 7 at power below 30.
func castDistanceRow(second bool, drain bool) SpellRule {
	if !second {
		return SpellRule{ID: 1, Delivery: 2, EffectSpeed: 200, ManaCost: 3, School: 1, MaxRange: 7,
			DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true, Distribution: 1}
	}
	if drain {
		return SpellRule{ID: 26, Delivery: 1, ManaCost: 20, School: 5, MaxRange: 7, DamageMin: 4,
			DamageMax: 6, TargetsUnit: true, Distribution: 1, Arm: 11, Second: true}
	}
	return SpellRule{ID: 1, Delivery: 2, EffectSpeed: 200, ManaCost: 3, School: 1, MaxRange: 7,
		DamageMin: 5, DamageMax: 10, TargetsUnit: true, Damaging: true, Distribution: 1, Arm: 1, Second: true}
}

// castDistanceWorld: a mage of sight 5 at (20, 8) and a standing target whose
// nearest body cell is d cells west.
func castDistanceWorld(t *testing.T, second bool, rule SpellRule, d int32, size uint8, casterOwner uint32) *World {
	t.Helper()
	caster := spMage(1, 20, 8, 30, 100, 100, 1<<rule.ID)
	caster.Owner, caster.ScanRange, caster.Speed = casterOwner, 5, 60
	caster.AttackCharge, caster.AttackRelax = 8, 4
	target := spEnt(2, 20-d-int32(size-1), 8)
	target.Owner, target.HP, target.MaxHP, target.TokenSize = 3-casterOwner, 5000, 5000, size
	w, err := NewStockedSpelledWorld(7, Bounds{Width: 40, Height: 16}, ModeCanonical,
		Terrain{}, []Entity{caster, target}, nil, acEnemies(t), nil, nil, []SpellRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	for i := range w.groups {
		if w.groups[i].owner == target.Owner {
			w.groups[i].order = orderStandGround
		}
	}
	return w
}

// castDistanceRelease reports the caster's x and edge distance at its first
// release at the target.
func castDistanceRelease(t *testing.T, w *World, order func(*World)) (x int32, edge uint8, ok bool) {
	t.Helper()
	order(w)
	for range 400 {
		for _, ev := range StepObserved(w, nil) {
			if ev.Caster == 1 && ev.Target == 2 {
				c, tg := spAt(t, w, 1), spAt(t, w, 2)
				return c.X, edgeDistance(&c, &tg), true
			}
		}
	}
	return 0, 0, false
}

// TestASecondGameMageCastsFromTheSpellRangeNotItsSight: a second-game book
// cast at a unit asks no sight and measures the edge distance against the
// spell's range (R2-ENGINE-1143, R2-ENGINE-1136). The first game keeps its
// sight rule.
func TestASecondGameMageCastsFromTheSpellRangeNotItsSight(t *testing.T) {
	t.Parallel()

	ordered := func(spell uint16) func(*World) {
		return func(w *World) { Step(w, []Command{spCast(1, 2, uint32(spell))}) }
	}
	attack := func(w *World) {
		if !w.orderAttack(0, 2) {
			t.Fatal("attack order refused")
		}
	}
	for _, tc := range []struct {
		name     string
		second   bool
		drain    bool
		d        int32
		size     uint8
		owner    uint32
		computer bool
		refusal  string
		x        int32
		edge     uint8
	}{
		{name: "Fire Arrow at 7", second: true, d: 7, size: 1, owner: 1, x: 20, edge: 7},
		{name: "Drain Life at 7", second: true, drain: true, d: 7, size: 1, owner: 1, x: 20, edge: 7},
		{name: "Fire Arrow at 8 walks one cell", second: true, d: 8, size: 1, owner: 1, refusal: refusalTargetOutOfRange, x: 19, edge: 7},
		{name: "a two-cell target whose nearest cell is 7 away", second: true, d: 7, size: 2, owner: 1, x: 20, edge: 7},
		{name: "a computer mage's attack at 7", second: true, d: 7, size: 1, owner: 2, computer: true, x: 20, edge: 7},
		{name: "first game: the caster's sight still gates", d: 7, size: 1, owner: 1, refusal: refusalTargetNotVisible},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := castDistanceRow(tc.second, tc.drain)
			w := castDistanceWorld(t, tc.second, rule, tc.d, tc.size, tc.owner)
			if got := w.BookSpellRefusal(1, 2, uint32(rule.ID)); got != tc.refusal {
				t.Fatalf("refusal %q, want %q", got, tc.refusal)
			}
			order := ordered(rule.ID)
			if tc.computer {
				order = attack
			}
			x, edge, ok := castDistanceRelease(t, w, order)
			if !tc.second {
				if ok {
					t.Fatalf("first game: released at edge distance %d from x=%d past the caster's sight", edge, x)
				}
				return
			}
			if !ok || x != tc.x || edge != tc.edge {
				t.Fatalf("release %v from x=%d at edge distance %d, want from x=%d at %d", ok, x, edge, tc.x, tc.edge)
			}
		})
	}
}

// TestTheSecondGameAutomaticCastKeepsSightAndAnchorCells: the candidate test
// keeps the caster's sight (R2-ENGINE-502) and the anchor cells
// (R2-ENGINE-348).
func TestTheSecondGameAutomaticCastKeepsSightAndAnchorCells(t *testing.T) {
	t.Parallel()

	for _, d := range []int32{5, 7} {
		t.Run(fmt.Sprint(d), func(t *testing.T) {
			w := castDistanceWorld(t, true, castDistanceRow(true, false), d, 1, 1)
			w.entities[0].AutoSpell = 1
			_, edge, ok := castDistanceRelease(t, w, func(*World) {})
			if ok != (d == 5) || ok && edge != 5 {
				t.Fatalf("automatic cast at %d: released %v at %d", d, ok, edge)
			}
		})
	}
	w := castDistanceWorld(t, true, castDistanceRow(true, false), 7, 2, 1)
	caster, wide := spAt(t, w, 1), spAt(t, w, 2)
	if w.withinCastReach(&caster, &wide, 7, reachChoice) || !w.withinCastReach(&caster, &wide, 7, reachOrder) {
		t.Fatal("second game: the candidate test does not compare the anchor cells, or the order does not compare the nearest cell")
	}
}
