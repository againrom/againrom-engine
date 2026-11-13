package sim

import "testing"

// thresholdActors are the actors both games build in the threshold test:
// a computer Human mage, archer, fighter and flying archer, a player's Human
// mage and a computer creature carrying its row's thresholds.
func thresholdActors() []Entity {
	mage := Entity{ID: 1, Owner: 3, X: 2, Y: 2, HP: 100, MaxHP: 100, MaxMana: 50, Mana: 50, Reach: 1, Humanoid: true}
	archer := Entity{ID: 2, Owner: 3, X: 4, Y: 2, HP: 80, MaxHP: 80, Reach: 6, Humanoid: true}
	fighter := Entity{ID: 3, Owner: 3, X: 6, Y: 2, HP: 60, MaxHP: 60, Reach: 1, Withdraw: 5, Humanoid: true}
	flyer := Entity{ID: 4, Owner: 3, X: 8, Y: 2, HP: 40, MaxHP: 40, Reach: 6, Domain: DomainAir, Humanoid: true}
	hero := Entity{ID: 5, Owner: SelfSlot, X: 10, Y: 2, HP: 90, MaxHP: 90, MaxMana: 30, Mana: 30, Reach: 1, Humanoid: true}
	beast := Entity{ID: 6, Owner: 3, X: 12, Y: 2, HP: 30, MaxHP: 30, Reach: 1, Withdraw: 7, Wimpy: 3}
	return []Entity{mage, archer, fighter, flyer, hero, beast}
}

// A second-game computer Human backs off at Withdraw and flees at Wimpy as
// its class gives them: a spellbook gives HPmax and HPmax/4, a range above
// 1 on a unit whose movement type is not 3 gives HPmax/2 and 0, any other
// Human 0 and 0 (R2-ENGINE-506). A player's Human keeps the player's
// setting and a creature its row's values. The first game keeps every value
// the actor was built with.
func TestSecondComputerHumansTakeTheClassThresholds(t *testing.T) {
	want := map[EntityID][2]int32{1: {100, 25}, 2: {40, 0}, 3: {0, 0}, 4: {0, 0}, 5: {0, 0}, 6: {7, 3}}
	w := secondWorld(t, 1, Bounds{Width: 16, Height: 16}, nil, thresholdActors()...)
	for _, e := range w.Entities() {
		if got := [2]int32{e.Withdraw, e.Wimpy}; got != want[e.ID] {
			t.Errorf("second game: actor %d holds Withdraw/Wimpy %v, want %v", e.ID, got, want[e.ID])
		}
	}
	first, err := NewSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, nil, thresholdActors(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	built := map[EntityID][2]int32{}
	for _, e := range thresholdActors() {
		built[e.ID] = [2]int32{e.Withdraw, e.Wimpy}
	}
	for _, e := range first.Entities() {
		if got := [2]int32{e.Withdraw, e.Wimpy}; got != built[e.ID] {
			t.Errorf("first game: actor %d holds Withdraw/Wimpy %v, want its built %v", e.ID, got, built[e.ID])
		}
	}
}

// guardFloorGroup is the mission-start Group of slot 3 holding two members
// at (20,20) and (22,20) with the given sight, in a World running script s.
func guardFloorGroup(t *testing.T, s *Script, sight uint8) SavedGroup {
	t.Helper()
	a, b := engFighter(2, leashSlot, 20, 20), engFighter(3, leashSlot, 22, 20)
	a.Group, b.Group, a.ScanRange, b.ScanRange = 1, 1, sight, sight
	hero := engFighter(leashHero, SelfSlot, 40, 20)
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{}, []Entity{hero, a, b}, s, Relations{})
	if err != nil {
		t.Fatal(err)
	}
	groups, _, present := w.SavedGroups()
	if !present {
		t.Fatal("fixture: no Group registry at mission start")
	}
	for _, g := range groups {
		if g.Owner.Owner == leashSlot {
			return g
		}
	}
	t.Fatal("fixture: no Group of slot 3")
	return SavedGroup{}
}

// The second game's mode-18 setter with no range argument writes 20 to the
// working radius +0x2d and the base +0x38 when the members' widest sum +0x2c
// is below MinimalGuardRange, and the sum itself otherwise; it clears +0x39
// (R2-ENGINE-843). The first game's guard setter raises the sum to
// MinimalGuardRange (AI-450).
func TestSecondMissionStartGuardBelowTheFloorTakesTwenty(t *testing.T) {
	second := func() *Script { return secondGameScript(t, nil, nil, nil) }
	for _, tc := range []struct {
		name                  string
		script                *Script
		sight                 uint8
		programme, sum, value uint8
	}{
		{"second game below the floor", second(), 5, secondGuardRoute, 6, 20},
		{"second game at or above the floor", second(), 9, secondGuardRoute, 10, 10},
		{"first game below the floor", nil, 5, orderGuard, 6, minimalGuardRange},
	} {
		g := guardFloorGroup(t, tc.script, tc.sight)
		if g.AI[0x20] != tc.programme || g.AI[0x2c] != tc.sum || g.AI[0x2d] != tc.value || g.AI[0x38] != tc.value || g.AI[0x39] != 0 {
			t.Errorf("%s: programme %d sum %d radius %d base %d +0x39 %d, want %d, %d, %d, %d and 0", tc.name,
				g.AI[0x20], g.AI[0x2c], g.AI[0x2d], g.AI[0x38], g.AI[0x39], tc.programme, tc.sum, tc.value, tc.value)
		}
	}
}

// A second-game Summon's new Group takes the guard setter with no range
// argument: programme 1, the member's geometry and +0x2d = +0x38 = the
// widest sum raised to MinimalGuardRange, +0x39 zero; the member holds order
// 0xb (R2-ENGINE-788). The first game's cheat summon, the same builder,
// keeps its order-none Group.
func TestSecondSummonGroupTakesTheGuardSetter(t *testing.T) {
	w := summonWorld(t, 0x25, 0)
	if !w.ordinaryEffectPayload(0, 0, w.spells[0], 50) {
		t.Fatal("Summon did not land")
	}
	i := indexOfEntity(w.entities, 2)
	g := w.savedGroupFor(2)
	if i < 0 || g == nil {
		t.Fatal("no summoned creature in a Group")
	}
	sum := w.entities[i].ScanRange
	want := max(sum, minimalGuardRange)
	if g.AI[0x20] != orderGuard || g.AI[0x2c] != sum || g.AI[0x2d] != want || g.AI[0x38] != want || g.AI[0x39] != 0 {
		t.Fatalf("summoned Group programme %d sum %d radius %d base %d +0x39 %d, want %d, %d, %d, %d and 0",
			g.AI[0x20], g.AI[0x2c], g.AI[0x2d], g.AI[0x38], g.AI[0x39], orderGuard, sum, want, want)
	}
	if o := w.savedOrder(2); w.entities[i].ActorState != actorStateGuard || o == nil || o.State != uint32(actorStateGuard) {
		t.Fatalf("summoned creature state %#x order %v, want 0xb", w.entities[i].ActorState, o)
	}
	cloneWorld(t, w)

	c := cheatWorld(t)
	id, err := c.CheatSummon(Entity{Owner: SelfSlot, HP: 20, MaxHP: 20, Reach: 1, TokenSize: 1}, nil, [EquipSlots]ItemInstance{}, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if g := c.savedGroupFor(id); g == nil || g.AI[0x20] != orderNone || g.AI[0x2d] != 0 {
		t.Fatalf("first game: the cheat summon's Group %v, want programme 0 and radius 0", g)
	}
}
