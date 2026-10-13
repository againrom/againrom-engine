package game

import (
	"testing"

	"againrom/pkg/sim"
)

// fogMover is a local-player walker with a three-cell sight on the open fog
// fixture grid.
func fogMover(x, y int32) sim.Entity {
	return sim.Entity{ID: 1, X: x, Y: y, HP: 100, MaxHP: 100, Owner: sim.SelfSlot, ScanRange: 3,
		TokenSize: 1, Speed: 64}
}

func fogMoverWorld(t *testing.T, ents ...sim.Entity) *mapWorld {
	t.Helper()
	return newMapWorld(fogTestWorld(t, ents), nil, nil, fogTestViewer(t))
}

// visibleCovers reports the first cell the world's sight for the local player
// lights and the visible layer does not, or -1.
func visibleMisses(mw *mapWorld) int {
	for i, lit := range mw.world.Sight(sim.SelfSlot) {
		if lit != 0 && mw.fog.visible[i] == 0 {
			return i
		}
	}
	return -1
}

func TestMovingUnitLightsNewGroundOnItsOwnTick(t *testing.T) {
	mw := fogMoverWorld(t, fogMover(10, 22))
	mw.pending = append(mw.pending, sim.MoveTo(1, sim.CellPoint{X: 34, Y: 22}))
	moved := 0
	start := int32(10)
	for n := 0; n < 3*fogPeriod; n++ {
		mw.tick()
		if e, _ := mw.entity(1); e.X != start {
			start = e.X
			moved++
		}
		if i := visibleMisses(mw); i >= 0 {
			t.Fatalf("world tick %d: cell %d is in the unit's sight and not in the visible layer", mw.world.Tick(), i)
		}
	}
	if moved < 10 {
		t.Fatalf("the unit changed cell %d times; the fixture does not walk", moved)
	}
}

func TestGroundAMovingUnitLeftStaysLitUntilThePeriod(t *testing.T) {
	mw := fogMoverWorld(t, fogMover(10, 22))
	const left = 22*45 + 8 // lit at the start, out of sight once the unit is past x=11
	if mw.fog.visible[left] == 0 {
		t.Fatal("the start cell's neighbour is not lit at the start")
	}
	mw.pending = append(mw.pending, sim.MoveTo(1, sim.CellPoint{X: 34, Y: 22}))
	sawLeft := false
	for n := 1; n < fogPeriod; n++ {
		mw.tick()
		if e, _ := mw.entity(1); e.X > 13 {
			if mw.world.Sight(sim.SelfSlot)[left] != 0 {
				t.Fatal("the unit still sees the cell it left")
			}
			if mw.fog.visible[left] == 0 {
				t.Fatalf("world tick %d: ground the unit left went dark before the period", mw.world.Tick())
			}
			sawLeft = true
		}
	}
	if !sawLeft {
		t.Fatal("the unit never left the start cell's sight inside one period")
	}
	mw.tick()
	if mw.fog.visible[left] != 0 {
		t.Fatalf("world tick %d: the period did not clear ground the unit left", mw.world.Tick())
	}
}

func TestAStandingUnitsLayerIsTheOneItsPeriodsMake(t *testing.T) {
	mw := fogMoverWorld(t, fogMover(22, 22))
	want := append([]byte(nil), mw.fog.visible...)
	wantExplored := append([]byte(nil), mw.fog.explored...)
	for n := 0; n < 2*fogPeriod; n++ {
		mw.tick()
		for i := range want {
			if mw.fog.visible[i] != want[i] || mw.fog.explored[i] != wantExplored[i] {
				t.Fatalf("world tick %d: cell %d differs for a standing unit", mw.world.Tick(), i)
			}
		}
	}
}

// TestAnAutoCastingMageCastsOnlyAtWhatTheFogShows is the player's report: the
// mage's automatic cast picks its target by the mage's own sight on the tick
// the mage steps into range, and the enemy it picks is lit on the screen on
// that same tick.
func TestAnAutoCastingMageCastsOnlyAtWhatTheFogShows(t *testing.T) {
	mage := sim.Entity{ID: 1, X: 10, Y: 22, HP: 100, MaxHP: 100, Owner: sim.SelfSlot, TokenSize: 1,
		Speed: 64, Mind: 60, Mana: 60, MaxMana: 60, KnownSpells: 1 << 1, AutoSpell: 1,
		ScanRange: 3, AttackCharge: 1, AttackRelax: 1}
	mage.Skill[1] = 30
	enemy := sim.Entity{ID: 2, X: 22, Y: 22, HP: 1000, MaxHP: 1000, Owner: 2, TokenSize: 1}
	var rel sim.Relations
	rel.Set(sim.SelfSlot, 2, 1)
	rel.Set(2, sim.SelfSlot, 1)
	arrow := sim.SpellRule{ID: 1, ManaCost: 3, School: 1, MaxRange: 7, DamageMin: 4, DamageMax: 8,
		TargetsUnit: true, Damaging: true}
	w, err := sim.NewStockedSpelledWorld(1, fogTestBounds, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{mage, enemy}, nil, rel, nil, nil, []sim.SpellRule{arrow})
	if err != nil {
		t.Fatal(err)
	}
	mw := newMapWorld(w, nil, nil, fogTestViewer(t))
	enemyCell := 22*45 + 22
	if mw.fog.visible[enemyCell] != 0 {
		t.Fatal("the enemy is lit before the mage moves")
	}
	mw.pending = append(mw.pending, sim.MoveTo(1, sim.CellPoint{X: 20, Y: 22}))
	for n := 0; n < 2*fogPeriod-1; n++ {
		mw.tick()
		m, _ := mw.entity(1)
		if m.Mana == 60 {
			if mw.fog.visible[enemyCell] != 0 && w.Sight(sim.SelfSlot)[enemyCell] == 0 {
				t.Fatalf("world tick %d: the enemy is lit and the mage does not see it", w.Tick())
			}
			continue
		}
		if mw.fog.visible[enemyCell] == 0 {
			t.Fatalf("world tick %d: the mage cast at an enemy the visible layer does not show", w.Tick())
		}
		return
	}
	t.Fatal("the mage never cast before the second period turned over; the fixture does not exercise the report")
}

// The fog is presentation state: a walked world's hash is the hash of the
// same world stepped with no fog at all.
func TestTheStampLeavesTheWorldHashAlone(t *testing.T) {
	ents := []sim.Entity{fogMover(10, 22), {ID: 2, X: 20, Y: 20, HP: 50, MaxHP: 50, Owner: 2, TokenSize: 1}}
	mw := fogMoverWorld(t, ents...)
	bare := fogTestWorld(t, ents)
	order := sim.MoveTo(1, sim.CellPoint{X: 30, Y: 22})
	mw.pending = append(mw.pending, order)
	sim.Step(bare, []sim.Command{order})
	for n := 0; n < 2*fogPeriod; n++ {
		mw.tick()
		if n > 0 {
			sim.Step(bare, nil)
		}
		if mw.world.Hash() != bare.Hash() {
			t.Fatalf("world tick %d: hash differs from the unfogged run", mw.world.Tick())
		}
	}
}
