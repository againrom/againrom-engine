package sim

import (
	"encoding/binary"
	"testing"
)

// missionStartWorld is a second-game mission start: the hero, and slot 3's
// Groups 1 (members at cells) and 2 (one member at (30,30)), with no
// original file read, over terrain[0] when one is given.
func missionStartWorld(t *testing.T, cells []cell, terrain ...Terrain) *World {
	t.Helper()
	ents := []Entity{engFighter(leashHero, SelfSlot, 5, 5)}
	for k, c := range cells {
		m := engFighter(EntityID(2+k), leashSlot, c.x, c.y)
		m.Group, m.ScanRange, m.HP, m.MaxHP = 1, leashSight, 400, 400
		ents = append(ents, m)
	}
	other := engFighter(EntityID(2+len(cells)), leashSlot, 30, 30)
	other.Group = 2
	ents = append(ents, other)
	rel := engRel(t, [3]uint32{leashSlot, SelfSlot, 1}, [3]uint32{SelfSlot, leashSlot, 2})
	ground := Terrain{}
	if len(terrain) > 0 {
		ground = terrain[0]
	}
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, ground, ents, secondGameScript(t, nil, nil, nil), rel)
	if err != nil {
		t.Fatal(err)
	}
	if !playsSecondGame(w) {
		t.Fatal("fixture: not a second-game World")
	}
	return w
}

// A second-game mission start holds the Group registry the first game's
// constructor builds: one Group per map Group at programme byte 18, and its
// first group tick runs the guard tick, whose latch rolls +0x2d = +0x38 + 4 +
// U{-1,0,1} once (R2-ENGINE-841, R2-ENGINE-843, R2-ENGINE-848).
func TestSecondGameMissionStartHoldsTheGroupRegistry(t *testing.T) {
	w := missionStartWorld(t, []cell{{20, 20}, {22, 20}})
	groups, orders, present := w.SavedGroups()
	if !present {
		t.Fatal("a second-game mission start holds no Group registry")
	}
	var guards []SavedGroup
	for _, g := range groups {
		if g.Owner.Owner == leashSlot {
			guards = append(guards, g)
		}
	}
	if len(guards) != 2 || guards[0].Selector != 1 || len(guards[0].Members) != 2 || guards[1].Selector != 2 || len(guards[1].Members) != 1 {
		t.Fatalf("slot 3's Groups %+v, want Group 1 with two members and Group 2 with one", guards)
	}
	for _, g := range guards {
		if g.AI[0x20] != guardRouteB || g.AI[0x45] != 1 {
			t.Fatalf("Group %d programme %d active %d, want 18 and 1", g.Selector, g.AI[0x20], g.AI[0x45])
		}
	}
	if len(orders) < 3 {
		t.Fatalf("the registry holds %d member orders, want one per guard member", len(orders))
	}
	leashRun(w, 1)
	var g SavedGroup
	groups, _, _ = w.SavedGroups()
	for _, h := range groups {
		if h.Owner.Owner == leashSlot && h.Selector == 1 {
			g = h
		}
	}
	if latch := binary.LittleEndian.Uint32(g.AI[0x30:]); latch != 1 {
		t.Fatalf("the first group tick left latch +0x30 = %d, want the guard tick's 1", latch)
	}
	if d := int(g.AI[0x2d]) - int(g.AI[0x38]); d < 3 || d > 5 {
		t.Fatalf("+0x2d - +0x38 = %d after the first group tick, want 3..5", d)
	}
}

// A World without a Group registry is never refused a save: the SAVE writes
// from a copy holding the registry the constructor builds from the current
// state, and the World itself stays as it was.
func TestASaveOfAWorldWithoutARegistryWritesTheConstructedGroups(t *testing.T) {
	w := missionStartWorld(t, []cell{{20, 20}})
	want, wantOrders, _ := w.SavedGroups()
	w.savedGroups = nil
	saved := w.WithGroupRegistry()
	if _, _, present := w.SavedGroups(); present || saved == w {
		t.Fatal("the SAVE's copy changed the World it was taken from")
	}
	groups, orders, present := saved.SavedGroups()
	if !present || len(groups) != len(want) || len(orders) != len(wantOrders) {
		t.Fatalf("the SAVE's copy holds %d Groups and %d orders (present %v), want %d and %d", len(groups), len(orders), present, len(want), len(wantOrders))
	}
	for k := range groups {
		if groups[k].Selector != want[k].Selector || len(groups[k].Members) != len(want[k].Members) {
			t.Fatalf("Group %d is %+v, want %+v", k, groups[k], want[k])
		}
	}
	if held := missionStartWorld(t, []cell{{20, 20}}); held.WithGroupRegistry() != held {
		t.Fatal("a World holding its registry is copied for the SAVE")
	}
}
