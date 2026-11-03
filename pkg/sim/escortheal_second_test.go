package sim

import "testing"

// secondHealWorld is a defending escort beside a hurt charge. In the first
// game its book holds Heal as id 6. In the second game id 6 is Poison Cloud
// and Heal is id 24; the rows carry the arms the second game's table assigns.
func secondHealWorld(t *testing.T, second bool) (*World, uint32) {
	t.Helper()
	charge := laFighter(1, 2, 7, 20, 20)
	charge.HP, charge.MaxHP = 49, 100
	caster := spMage(2, 22, 20, 100, 50, 10, ^uint32(1))
	caster.Owner, caster.Group, caster.ScanRange = 2, 7, 10
	enemy := laFighter(3, 3, 9, 23, 20)
	healID := uint32(6)
	spells := []SpellRule{hlHeal()}
	if second {
		poison := SpellRule{ID: 6, ManaCost: 10, School: 5, MaxRange: 6, DamageMin: 10, DamageMax: 20, TargetsUnit: true, Damaging: true}
		heal := hlHeal()
		heal.ID, healID = 24, 24
		spells = []SpellRule{poison, heal}
		AssignSecondGameArms(spells)
	}
	w, err := NewStockedSpelledWorld(7, engBounds, ModeCanonical, Terrain{},
		[]Entity{charge, caster, enemy}, nil, engRel(t, [3]uint32{2, 3, 1}), nil, nil, spells)
	if err != nil {
		t.Fatal(err)
	}
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	w.ImportAutoHealing(2, 0)
	w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
		Unit: 1, HasUnit: true, Args: [scriptParams]int32{subCommandDefend, 3}})
	return w, healID
}

// TestEscortHealsWithTheGamesHeal pins that an escort casts the row whose arm
// heals: id 6 in the first game, id 24 in the second.
func TestEscortHealsWithTheGamesHeal(t *testing.T) {
	for _, second := range []bool{false, true} {
		w, want := secondHealWorld(t, second)
		if !w.escortHeal(1, 0) || len(w.bookCasts) != 1 {
			t.Fatalf("second=%v: the escort did not start a heal", second)
		}
		if got := uint32(w.bookCasts[0].Spell); got != want {
			t.Errorf("second=%v: the escort cast %d, want %d", second, got, want)
		}
	}
}

// TestUnarmedAutoHealResolvesTheGamesHeal is the same for the unarmed
// auto-heal.
func TestUnarmedAutoHealResolvesTheGamesHeal(t *testing.T) {
	for _, second := range []bool{false, true} {
		w, want := secondHealWorld(t, second)
		rule, ok := w.unarmedHeal(1)
		if !ok || uint32(rule.ID) != want {
			t.Errorf("second=%v: unarmed heal resolved %d (ok %v), want %d", second, rule.ID, ok, want)
		}
	}
}
