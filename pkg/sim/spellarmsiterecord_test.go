package sim

import "testing"

// spellArmSiteRecord is each answer a World site gives about an arm, as the
// sites gave it before the arm table held it. creatureAim is the first
// game's aim of the arm (the second game's aims are keyed by row id, in their
// own table). healID is the spell id an escort's heal and the unarmed
// auto-heal ask a caster's book for: the first game's Heal id, asked in both
// games.
var spellArmSiteRecord = map[string]armAnswers{
	"creatureAim": bothGames(map[uint16]int{
		1: int(creatureAimVictim), 11: int(creatureAimVictim), 13: int(creatureAimVictim), 14: int(creatureAimVictim),
		20: int(creatureAimVictim), 27: int(creatureAimVictim), 28: int(creatureAimVictim),
		5: int(creatureAimCaster), 6: int(creatureAimCaster), 10: int(creatureAimCaster), 15: int(creatureAimCaster),
		16: int(creatureAimCaster), 18: int(creatureAimCaster), 22: int(creatureAimCaster), 23: int(creatureAimCaster),
		24: int(creatureAimCaster),
		2:  int(creatureAimVictimCell), 3: int(creatureAimVictimCell), 7: int(creatureAimVictimCell), 8: int(creatureAimVictimCell),
		12: int(creatureAimVictimCell), 17: int(creatureAimVictimCell), 19: int(creatureAimVictimCell), 21: int(creatureAimVictimCell),
		9: int(creatureAimStepCell), 26: int(creatureAimStepCell),
	}),
	"healID": bothGames(armSet(6)),
	// burst: a blast of the row ends in a burst record; speedFlight: the
	// row's delivery waits the Euclidean distance over its effect speed.
	// Both read the row id, in both games.
	"burst":       bothGames(armSet(2)),
	"speedFlight": bothGames(armSet(2)),
}

// siteBursts is how many burst records one radius-0 blast of a row with id
// id lays, the row running Fire Ball's own area arm; a table that refuses
// the row lays none.
func siteBursts(id uint16, second bool) int {
	rule := blastRule(0)
	rule.ID, rule.Arm, rule.Second = id, 2, second
	w, err := NewStockedSpelledWorld(7, Bounds{Width: 32, Height: 32}, ModeCanonical,
		Terrain{}, []Entity{effectMage(1, 2, 2, 1<<2)}, nil, Relations{}, nil, nil, []SpellRule{rule})
	if err != nil {
		return 0
	}
	w.SetBurstPhases(11)
	w.landAreaFacing(rule, 0, 1, true, 2, 2, 16, 16, 0, false, nil)
	return len(burstCells(w))
}

// siteSpeedFlight reports a delivery of a row with id id, effect speed 3,
// that waits the Euclidean rather than the Chebyshev distance.
func siteSpeedFlight(id uint16, second bool) bool {
	d := spellDelivery{Rule: SpellRule{ID: id, Arm: 2, EffectSpeed: 3, Second: second}, X: 10, Y: 10}
	return (&World{}).spellDeliveryRemaining(d) != spellDeliveryTicks(d.Rule, 0, 0, 10, 10)
}

// siteHealWorld is a defending escort whose book knows ids 1 to 31, each a
// defensive heal, beside a hurt charge: the cast it starts names the id its
// heal asks for.
func siteHealWorld(t *testing.T, second bool) *World {
	t.Helper()
	charge := laFighter(1, 2, 7, 20, 20)
	charge.HP, charge.MaxHP = 49, 100
	caster := spMage(2, 22, 20, 100, 50, 10, ^uint32(1))
	caster.Owner, caster.Group, caster.ScanRange = 2, 7, 10
	enemy := laFighter(3, 3, 9, 23, 20)
	var spells []SpellRule
	for id := uint16(1); id <= 31; id++ {
		heal := hlHeal()
		heal.ID, heal.Defensive, heal.Second = id, true, second
		spells = append(spells, heal)
	}
	w, err := NewStockedSpelledWorld(7, engBounds, ModeCanonical, Terrain{},
		[]Entity{charge, caster, enemy}, nil, engRel(t, [3]uint32{2, 3, 1}), nil, nil, spells)
	if err != nil {
		t.Fatal(err)
	}
	if second {
		w.rom2 = &rom2ScriptState{}
	}
	w.ImportAutoHealing(2, 0)
	w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
		Unit: 1, HasUnit: true, Args: [scriptParams]int32{subCommandDefend, 3}})
	return w
}

// siteHealIDs is the id the escort's heal casts and the id the unarmed
// auto-heal resolves, in one game.
func siteHealIDs(t *testing.T, second bool) (escort, unarmed uint32) {
	w := siteHealWorld(t, second)
	if rule, ok := w.unarmedHeal(1); ok {
		unarmed = uint32(rule.ID)
	}
	if w.escortHeal(1, 0) && len(w.bookCasts) == 1 {
		escort = uint32(w.bookCasts[0].Spell)
	}
	return escort, unarmed
}

// siteAnswers asks each World site its question about one arm.
func siteAnswers(t *testing.T, arm uint16, second bool, escort, unarmed uint32) map[string]int {
	heal := 0
	if escort == uint32(arm) && unarmed == uint32(arm) {
		heal = 1
	}
	return map[string]int{
		"creatureAim": int(creatureAimOf(uint32(arm))),
		"healID":      heal,
		"burst":       boolAnswer(siteBursts(arm, second) == 1),
		"speedFlight": boolAnswer(siteSpeedFlight(arm, second)),
	}
}

// TestSpellArmSitesKeepTheirRecord asks every recorded World site about
// every probed arm in both games: each answers its record.
func TestSpellArmSitesKeepTheirRecord(t *testing.T) {
	for _, second := range []bool{false, true} {
		escort, unarmed := siteHealIDs(t, second)
		if escort == 0 || unarmed == 0 {
			t.Fatalf("second=%v: the escort heal cast %d, the unarmed heal resolved %d", second, escort, unarmed)
		}
		for _, arm := range armProbe() {
			for name, got := range siteAnswers(t, arm, second, escort, unarmed) {
				if want := spellArmSiteRecord[name].of(arm, second); got != want {
					t.Errorf("%s: arm %d second=%v answers %d, record %d", name, arm, second, got, want)
				}
			}
		}
	}
	for _, id := range []uint32{65, 1 << 16, 1<<32 - 1} {
		if creatureAimOf(id) != creatureAimNone {
			t.Errorf("creature aim %d is not none", id)
		}
	}
	if got := fireBallBurstPicture(); got != 13 {
		t.Errorf("the burst picture is %d, record 13", got)
	}
}
