package sim

import "testing"

// allySightWorld is sideSightWorld with the scout in slot 3 under rel.
func allySightWorld(t *testing.T, second bool, rel Relations) *World {
	t.Helper()
	rule := castDistanceRow(second, false)
	caster := spMage(1, 20, 8, 30, 100, 100, 1<<rule.ID)
	caster.Owner, caster.ScanRange, caster.Speed = SelfSlot, 5, 60
	caster.AttackCharge, caster.AttackRelax = 8, 4
	target := spEnt(2, 13, 8)
	target.Owner, target.HP, target.MaxHP, target.TokenSize = 2, 5000, 5000, 1
	scout := spEnt(3, 12, 12)
	scout.Owner, scout.ScanRange, scout.TokenSize = 3, 12, 1
	w, err := NewStockedSpelledWorld(7, Bounds{Width: 40, Height: 16}, ModeCanonical,
		Terrain{}, []Entity{caster, target, scout}, nil, rel, nil, nil, []SpellRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	for i := range w.groups {
		if w.groups[i].owner != SelfSlot {
			w.groups[i].order = orderStandGround
		}
	}
	return w
}

// TestTheSecondGameAutomaticCastTakesWhatAnAllySees: the second game's
// automatic cast takes an enemy only a unit of a player allied on the
// caster's row (ownOrAllied) sees; ownerSees keeps the own player's units.
func TestTheSecondGameAutomaticCastTakesWhatAnAllySees(t *testing.T) {
	t.Parallel()

	enemy := [][3]uint32{{SelfSlot, 2, relationHostile}, {2, SelfSlot, relationHostile}}
	rel := func(cells ...[3]uint32) Relations { return engRel(t, append(cells, enemy...)...) }
	for _, tc := range []struct {
		name   string
		second bool
		rel    Relations
		cast   bool
	}{
		{name: "allied both ways", second: true, rel: rel([3]uint32{SelfSlot, 3, relationLocked}, [3]uint32{3, SelfSlot, relationLocked}), cast: true},
		{name: "allied on the caster's row only", second: true, rel: rel([3]uint32{SelfSlot, 3, relationLocked}), cast: true},
		{name: "allied on the scout's row only", second: true, rel: rel([3]uint32{3, SelfSlot, relationLocked})},
		{name: "allied bit with the hostile bit", second: true, rel: rel([3]uint32{SelfSlot, 3, relationLocked | relationHostile})},
		{name: "neutral", second: true, rel: rel()},
		{name: "hostile", second: true, rel: rel([3]uint32{SelfSlot, 3, relationHostile}, [3]uint32{3, SelfSlot, relationHostile})},
		{name: "first game: an ally lends no sight", rel: rel([3]uint32{SelfSlot, 3, relationLocked}, [3]uint32{3, SelfSlot, relationLocked})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := allySightWorld(t, tc.second, tc.rel)
			if w.ownerSees(SelfSlot, 2) {
				t.Fatal("ownerSees: the player's own units see the enemy, want only the scout")
			}
			if !w.ownerSees(3, 2) {
				t.Fatal("ownerSees: the scout does not see the enemy")
			}
			x, edge, ok := castDistanceRelease(t, w, func(w *World) { w.entities[0].AutoSpell = 1 })
			if ok != tc.cast || ok && (x != 20 || edge != 7) {
				t.Fatalf("release %v from x=%d at edge distance %d, want %v from x=20 at 7", ok, x, edge, tc.cast)
			}
		})
	}
}
