package sim

import "testing"

const (
	abpHero      EntityID = 1
	abpMage      EntityID = 2
	abpFoe       EntityID = 3
	abpNeighbour EntityID = 4
	abpSide      uint32   = 5
)

// abpWorld is a second-game world with the player's fighter at (30,40), a
// Darkness caster of a hostile owner at (40,40), a hostile fighter beside the
// hero and a computer unit of a third owner, hostile to the caster, at
// (50,50). Every Darkness pulse is applied by the same call a live cloud
// makes. A non-nil block is the passability grid.
func abpWorld(t *testing.T, block []byte) (*World, SpellRule) {
	t.Helper()
	rule := secondRow(SpellRule{ID: 14, Area: true, Distribution: distributionDiamond, Radius: 1, AreaDuration: 2,
		TargetsUnit: true, EffectKind: EffectScanRange, EffectMode: EffectDuration, EffectDuration: 200})
	mage := effectMage(abpMage, 40, 40, 1<<14)
	mage.Owner = abpSide
	mage.ScanRange = 1
	hero := engFighter(abpHero, SelfSlot, 30, 40)
	hero.HP, hero.MaxHP, hero.Reach = 1000, 1000, 1
	foe := engFighter(abpFoe, abpSide, 31, 40)
	foe.HP, foe.MaxHP = 1000, 1000
	neighbour := engFighter(abpNeighbour, 6, 50, 50)
	rel := engRel(t, [3]uint32{abpSide, SelfSlot, 1}, [3]uint32{SelfSlot, abpSide, 1},
		[3]uint32{abpSide, 6, 1}, [3]uint32{6, abpSide, 1})
	w, err := NewStockedSpelledWorld(1, Bounds{Width: 64, Height: 64}, ModeCanonical, Terrain{Block: block},
		[]Entity{hero, mage, foe, neighbour}, secondGameScript(t, nil, nil, nil), rel, nil, nil, []SpellRule{rule})
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	if !playsSecondGame(w) {
		t.Fatal("fixture: the world does not play the second game")
	}
	return w, rule
}

// abpPulse is one Darkness pulse of the hostile mage over the named unit's cell.
func abpPulse(t *testing.T, w *World, rule SpellRule, id EntityID) {
	t.Helper()
	e := laEnt(t, w, id)
	key := cellKey(e.X, e.Y)
	w.applyAreaCells(cellEffect{Key: key, Spell: rule.ID, Mode: areaModeCloud, Caster: abpMage, HasCaster: true}, rule, []uint16{key})
}

// A Darkness pulse of another owner does not erase a player's walk: the
// destination stands and no victim appears.
func TestAnAreaBlowLeavesAPlayerMoveOrderInForce(t *testing.T) {
	t.Parallel()
	w, rule := abpWorld(t, nil)
	dest := CellPoint{X: 20, Y: 30}
	Step(w, []Command{GroupMoveTo(abpHero, dest, 1)})
	abpPulse(t, w, rule, abpHero)
	e := laEnt(t, w, abpHero)
	if e.HasAttackTarget {
		t.Fatalf("the pulse replaced the walk with victim %d", e.AttackTarget)
	}
	if !e.HasTarget || e.TargetX != dest.X || e.TargetY != dest.Y {
		t.Fatalf("the destination is (%d,%d,%t) after the pulse, want (%d,%d,true)", e.TargetX, e.TargetY, e.HasTarget, dest.X, dest.Y)
	}
	for k := 0; k < 40; k++ {
		Step(w, nil)
	}
	if e := laEnt(t, w, abpHero); e.X != dest.X || e.Y != dest.Y {
		t.Fatalf("the walk ended at (%d,%d), want the ordered cell (%d,%d)", e.X, e.Y, dest.X, dest.Y)
	}
}

// A Darkness pulse of another owner does not take a fighter off the victim the
// player named.
func TestAnAreaBlowLeavesAPlayerAttackOrderInForce(t *testing.T) {
	t.Parallel()
	w, rule := abpWorld(t, nil)
	Step(w, []Command{Attack(abpHero, abpFoe)})
	for k := 0; k < 3; k++ {
		abpPulse(t, w, rule, abpHero)
		Step(w, nil)
		if e := laEnt(t, w, abpHero); !e.HasAttackTarget || e.AttackTarget != abpFoe {
			t.Fatalf("pulse %d: the ordered fighter holds victim %d/%v, want the foe %d", k, e.AttackTarget, e.HasAttackTarget, abpFoe)
		}
	}
}

// A move ordered behind a loaded attack cycle waits beside the victim; the
// pulse leaves the destination it waits with.
func TestAnAreaBlowLeavesAMoveHeldBehindALoadedCycle(t *testing.T) {
	t.Parallel()
	w, rule := abpWorld(t, nil)
	Step(w, []Command{Attack(abpHero, abpFoe)})
	for k := 0; k < 8 && laEnt(t, w, abpHero).AttackPhase == AttackReady; k++ {
		Step(w, nil)
	}
	if e := laEnt(t, w, abpHero); e.AttackPhase == AttackReady {
		t.Fatal("fixture: the fighter never loaded a cycle")
	}
	dest := CellPoint{X: 20, Y: 30}
	Step(w, []Command{GroupMoveTo(abpHero, dest, 1)})
	if e := laEnt(t, w, abpHero); !e.HasTarget || e.AttackPhase == AttackReady {
		t.Fatalf("fixture: the move did not wait behind the cycle: target %v, phase %d", e.HasTarget, e.AttackPhase)
	}
	abpPulse(t, w, rule, abpHero)
	if e := laEnt(t, w, abpHero); !e.HasTarget || e.TargetX != dest.X || e.TargetY != dest.Y {
		t.Fatalf("the held destination is (%d,%d,%t) after the pulse, want (%d,%d,true)", e.TargetX, e.TargetY, e.HasTarget, dest.X, dest.Y)
	}
}

// A hemmed-in fighter's walk ends at the stall limit and the group sends it
// again on its next pass. A pulse in that gap leaves the order to the pass.
func TestAnAreaBlowLeavesAWalkRefusedAtTheStallLimitInForce(t *testing.T) {
	t.Parallel()
	block := make([]byte, 64*64)
	for _, c := range [][2]int32{{29, 39}, {30, 39}, {31, 39}, {29, 40}, {29, 41}, {30, 41}, {31, 41}} {
		block[c[1]*64+c[0]] = 1
	}
	w, rule := abpWorld(t, block)
	wakeGroups(w)
	dest := CellPoint{X: 20, Y: 30}
	Step(w, []Command{GroupMoveTo(abpHero, dest, 1)})
	for k := 0; k < 64 && laEnt(t, w, abpHero).HasTarget; k++ {
		Step(w, nil)
	}
	if laEnt(t, w, abpHero).HasTarget {
		t.Fatal("fixture: the walk was never refused")
	}
	abpPulse(t, w, rule, abpHero)
	if e := laEnt(t, w, abpHero); e.HasAttackTarget {
		t.Fatalf("the pulse replaced the refused walk with victim %d", e.AttackTarget)
	}
	for k := 0; k < 32 && !laEnt(t, w, abpHero).HasTarget; k++ {
		Step(w, nil)
	}
	if e := laEnt(t, w, abpHero); !e.HasTarget || e.TargetX != dest.X || e.TargetY != dest.Y || e.HasAttackTarget {
		t.Fatalf("after the group's pass: destination (%d,%d,%t), victim %d/%v", e.TargetX, e.TargetY, e.HasTarget, e.AttackTarget, e.HasAttackTarget)
	}
}

// A player's fighter with no order is not sent at the caster by the pulse.
func TestAnAreaBlowDoesNotOrderAnIdlePlayerUnit(t *testing.T) {
	t.Parallel()
	w, rule := abpWorld(t, nil)
	abpPulse(t, w, rule, abpHero)
	if e := laEnt(t, w, abpHero); e.HasAttackTarget || e.HasTarget {
		t.Errorf("the idle fighter holds victim %d/%v and destination %v, want no order", e.AttackTarget, e.HasAttackTarget, e.HasTarget)
	}
}

// The control: a computer unit of another owner is still ordered at the
// caster, the standing answer of DIV-1600.
func TestAnAreaBlowStillOrdersAComputerUnitAtTheCaster(t *testing.T) {
	t.Parallel()
	w, rule := abpWorld(t, nil)
	abpPulse(t, w, rule, abpNeighbour)
	if e := laEnt(t, w, abpNeighbour); !e.HasAttackTarget || e.AttackTarget != abpMage {
		t.Errorf("the computer unit holds victim %d/%v, want the caster %d", e.AttackTarget, e.HasAttackTarget, abpMage)
	}
}
