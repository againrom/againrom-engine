package sim

import "testing"

const (
	sabCaster   EntityID = 1
	sabMember   EntityID = 2
	sabHero     EntityID = 3
	sabStranger EntityID = 4
	sabLoner    EntityID = 5
	// sabSide is the caster's and the member's owner, sabOther a hostile
	// owner of the stranger; the loner has no owner.
	sabSide  uint32 = 5
	sabOther uint32 = 6
)

// secondAreaBlowWorld is a second-game world with a Darkness caster of owner
// sabSide at (2,2), a member of the same owner at (40,40) attacking the hero
// beside it, a stranger of a hostile owner at (40,41) and an ownerless loner
// at (41,41). diagonal is the caster owner's relation to itself.
func secondAreaBlowWorld(t *testing.T, diagonal byte) (*World, SpellRule) {
	t.Helper()
	rule := SpellRule{ID: 17, Area: true, Distribution: distributionDiamond, Radius: 1, AreaDuration: 2,
		TargetsUnit: true, EffectKind: EffectScanRange, EffectMode: EffectDuration, EffectDuration: 200}
	caster := effectMage(sabCaster, 2, 2, 1<<17)
	caster.Owner = sabSide
	hero := engFighter(sabHero, SelfSlot, 41, 40)
	hero.HP, hero.MaxHP = 1000, 1000
	loner := engFighter(sabLoner, 0, 41, 41)
	rel := engRel(t, [3]uint32{sabSide, sabSide, uint32(diagonal)},
		[3]uint32{sabSide, SelfSlot, 1}, [3]uint32{SelfSlot, sabSide, 1},
		[3]uint32{sabSide, sabOther, 1}, [3]uint32{sabOther, sabSide, 1})
	w, err := NewStockedSpelledWorld(1, Bounds{Width: 64, Height: 64}, ModeCanonical, Terrain{},
		[]Entity{caster, engFighter(sabMember, sabSide, 40, 40), hero, engFighter(sabStranger, sabOther, 40, 41), loner},
		secondGameScript(t, nil, nil, nil), rel, nil, nil, []SpellRule{rule})
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	if !playsSecondGame(w) {
		t.Fatal("fixture: the world does not play the second game")
	}
	if !w.orderAttack(indexOfEntity(w.entities, sabMember), sabHero) {
		t.Fatal("fixture: the member did not take the hero")
	}
	return w, rule
}

func sabRelation(w *World, from, to uint32) byte {
	i, ok := relationIndex(from, to)
	if !ok || w.relations.cells == nil {
		return 0
	}
	return w.relations.cells[i]
}

// sabCells are the three occupied cells the area covers.
var sabCells = []uint16{cellKey(40, 40), cellKey(40, 41), cellKey(41, 41)}

// checkSecondAreaBlow states the answer of each occupant to the caster's
// Darkness: the member of the caster's own owner runs the hit hook (caster,
// member) and keeps the hero (R2-ENGINE-985, R2-ENGINE-987, R2-ENGINE-989);
// the diagonal changes only when its low two bits are clear (R2-ENGINE-988);
// the ownerless loner gets neither the hook nor an order (R2-ENGINE-985 step
// 6); the stranger of another owner is still ordered at the caster (DIV-1600).
func checkSecondAreaBlow(t *testing.T, w *World, diagonal, want byte) {
	t.Helper()
	m := laEnt(t, w, sabMember)
	if !m.HasAttackTarget || m.AttackTarget != sabHero {
		t.Errorf("the same-owner member holds victim %d/%v, want the hero %d it was attacking", m.AttackTarget, m.HasAttackTarget, sabHero)
	}
	if n := w.attackNoticeAt(indexOfEntity(w.entities, sabMember)); n.Cell != 2|2<<8 {
		t.Errorf("the same-owner member remembers cell %#x, want the caster's (2,2): the hit hook did not run", n.Cell)
	}
	if got := sabRelation(w, sabSide, sabSide); got != want {
		t.Errorf("the caster owner's diagonal is %#x after the blow from %#x, want %#x", got, diagonal, want)
	}
	if l := laEnt(t, w, sabLoner); l.HasAttackTarget {
		t.Errorf("the ownerless loner holds victim %d, want none", l.AttackTarget)
	}
	if n := w.attackNoticeAt(indexOfEntity(w.entities, sabLoner)); n.Cell != 0 {
		t.Errorf("the ownerless loner remembers cell %#x, want none", n.Cell)
	}
	if s := laEnt(t, w, sabStranger); !s.HasAttackTarget || s.AttackTarget != sabCaster {
		t.Errorf("the stranger of another owner holds victim %d/%v, want the caster %d", s.AttackTarget, s.HasAttackTarget, sabCaster)
	}
}

// The live area pulse: a second-game Darkness over a member of the caster's
// own owner does not turn it on the caster.
func TestASecondGameSameOwnerDarknessKeepsTheMembersTarget(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ diagonal, want byte }{{0x12, 0x12}, {0x10, 0x11}} {
		w, rule := secondAreaBlowWorld(t, c.diagonal)
		area := cellEffect{Key: cellKey(40, 40), Spell: rule.ID, Mode: areaModeCloud, Caster: sabCaster, HasCaster: true}
		w.applyAreaCells(area, rule, sabCells)
		checkSecondAreaBlow(t, w, c.diagonal, c.want)
	}
}

// A Darkness pulse resumed from a SAV payload answers as the live pulse.
func TestASecondGameLoadedSameOwnerDarknessKeepsTheMembersTarget(t *testing.T) {
	t.Parallel()
	live, rule := secondAreaBlowWorld(t, 0x12)
	if !live.landArea(rule, 0, sabCaster, true, 2, 2, 40, 40, nil) {
		t.Fatal("fixture: the Darkness cloud did not land")
	}
	rows, err := live.NativeAreaSaveStates()
	if err != nil || len(rows) != 1 {
		t.Fatalf("fixture: area save states %d, %v", len(rows), err)
	}
	w, _ := secondAreaBlowWorld(t, 0x12)
	w.applySavedAreaPayload(SavedAreaDriver{Key: cellKey(40, 40), Spell: rule.ID, Mode: areaModeCloud},
		SavedSpellEffect{AE44: &rows[0].Payload}, sabCells, sabCaster)
	checkSecondAreaBlow(t, w, 0x12, 0x12)
}
