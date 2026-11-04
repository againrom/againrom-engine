package game

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// secondCastingBatMission is the guard mission with Player 2 hostile to the
// party both ways and a bat of Player 2 four cells from it. With slot, the
// bat's Units row holds Spell 1 14 at Probability 10, as Bat_Sonic.4 does
// (R2-SESSION-341); without, its Spell cells are -1. Spell 14 is a
// cell-aimed row.
func secondCastingBatMission(slot bool) secondMissionContent {
	c := secondGuardMission()
	c.players = 2
	c.stances = map[[2]int]uint16{{0, 1}: 1, {1, 0}: 1}
	p := make([]int32, 62)
	for i := range p {
		p[i] = -1
	}
	p[4], p[29], p[30], p[54], p[55] = 20000, 70, 4, 30, secondSpellServerID
	if slot {
		p[48], p[49] = 14, 10
	}
	c.creatures = map[int]dbEntry{2: {name: "Bat_Sonic.4", params: p}}
	c.units = append(c.units, alm.Unit{X: 54<<8 | 0x80, Y: 96<<8 | 0x80, ClassID: 9, ClassSubID: 1,
		ServerID: secondSpellServerID, Owner: 2, UnitID: secondSpellUnitID, GroupID: 6})
	aimed := make([]int32, 22)
	aimed[0], aimed[1], aimed[2], aimed[4], aimed[5], aimed[6], aimed[8], aimed[9], aimed[11] = 3, 5, 3, 2, 1, 6, 3, 4, 15
	aimed[16], aimed[17], aimed[18] = -1, -1, -1
	c.spells = map[int]dbEntry{14: {name: "spell", params: aimed, strings: []string{"scanRange=-2:duration 1"}}}
	return c
}

// secondBatCasts is the tick of the bat's first cast of spell 14 within
// 2000 ticks of w, or -1.
func secondBatCasts(t *testing.T, w *sim.World) int {
	t.Helper()
	bat := secondSpellCreature(t, w)
	for tick := range 2000 {
		for _, ev := range sim.StepObserved(w, nil) {
			if ev.Caster == bat.ID && ev.Spell == 14 {
				return tick
			}
		}
	}
	return -1
}

// A second-game bat whose Units row holds Spell 1 14 at Probability 10
// holds slot 14 with threshold 3270 and, with an enemy in reach, casts
// spell 14 by the attack planner's slot rule (R2-ENGINE-651, R2-ENGINE-832):
// after a campaign mission start and after LOAD of a SAVE made at that
// start. A bat whose row holds no Spell cell never casts it.
func TestSecondBatCastsItsSlotSpell(t *testing.T) {
	c := secondCastingBatMission(true)
	f, app, screen, _ := secondBagFrontOf(t, secondBagSample(), c)
	secondBagEnter(t, f, app, screen)
	want := [sim.CreatureSpellSlots]sim.CreatureSpell{{ID: 14, Threshold: 10 * sim.CreatureSpellScale}}
	if got := secondSpellCreature(t, f.live.world).CreatureSpells; got != want {
		t.Fatalf("the bat holds slots %v, want %v", got, want)
	}
	raw := secondPageSave(t, f, app)
	if tick := secondBatCasts(t, f.live.world); tick < 0 {
		t.Fatal("after the mission start the bat never casts spell 14")
	}
	loaded, _ := secondMaskLoad(t, c, raw)
	if got := secondSpellCreature(t, loaded.live.world).CreatureSpells; got != want {
		t.Fatalf("LOAD: the bat holds slots %v, want %v", got, want)
	}
	if tick := secondBatCasts(t, loaded.live.world); tick < 0 {
		t.Fatal("after LOAD the bat never casts spell 14")
	}

	plain, papp, pscreen, _ := secondBagFrontOf(t, secondBagSample(), secondCastingBatMission(false))
	secondBagEnter(t, plain, papp, pscreen)
	if tick := secondBatCasts(t, plain.live.world); tick >= 0 {
		t.Fatalf("a bat with no slot casts spell 14 at tick %d", tick)
	}
}
