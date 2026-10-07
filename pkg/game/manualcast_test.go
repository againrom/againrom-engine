package game

import (
	"testing"

	"againrom/pkg/sim"
)

func TestManualCellCastCancelsPickupApproach(t *testing.T) {
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: poHeroID, Owner: 2, X: poHeroX, Y: poHeroY, HP: 100, MaxHP: 100,
			Mind: 60, Mana: 100, MaxMana: 100, KnownSpells: 1 << 26}}, nil,
		[]sim.SpellRule{{ID: 26, ManaCost: 7, MaxRange: 10}})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.ReplaceGroundSacks([]sim.Sack{{X: poSackX, Y: poSackY, Gold: 3}}); err != nil {
		t.Fatal(err)
	}
	mw := grabWorld(t, w, poHeroID, missionSource{})
	mw.grab(poHeroID, poSackX, poSackY, true)
	mw.attackOrCast(poHeroID, 0, 26, 6, 5, true)
	for tick := 0; tick < 64; tick++ {
		mw.tick()
	}
	e, _ := mw.entity(poHeroID)
	if e.X != 6 || e.Y != 5 || e.Mana != 93 || len(w.Sacks()) != 1 || e.HasTarget {
		t.Fatalf("pickup outlived manual Teleport: actor=%+v sacks=%+v", e, w.Sacks())
	}
}
