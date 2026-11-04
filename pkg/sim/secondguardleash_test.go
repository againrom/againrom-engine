package sim

import (
	"encoding/binary"
	"testing"
)

const (
	leashHero EntityID = 1
	leashSlot uint32   = 3
	// leashSight lets a member see past every radius these tests set, so
	// the leash and not sight decides what a member keeps.
	leashSight = 14
	// leashBase is the stored base radius +0x38 the fixtures give.
	leashBase uint8 = 6
)

// leashWorld is a second-game LOAD as an original file leaves it: the hero
// and one Group of slot 3 at programme byte 18 holding the members at
// cells, its AI bytes set by ai. Slot 3 is hostile to the hero; the hero's
// cell toward slot 3 is locked, so the hero never answers and no fight but
// the Group's own decides what the tests see. A second-game LOAD reads no
// member order.
func leashWorld(t *testing.T, hero cell, cells []cell, ai func(*SavedGroup)) *World {
	t.Helper()
	ents := []Entity{engFighter(leashHero, SelfSlot, hero.x, hero.y)}
	guard := SavedGroup{ID: 1, Selector: 1, Owner: SavedGroupReference{Class: 1, Owner: leashSlot}}
	for k, c := range cells {
		id := EntityID(2 + k)
		m := engFighter(id, leashSlot, c.x, c.y)
		m.Group, m.ScanRange, m.HP, m.MaxHP = 1, leashSight, 400, 400
		ents = append(ents, m)
		guard.Members = append(guard.Members, SavedGroupMember{Archive: uint16(2 + k), Entity: id, Bound: true})
	}
	rel := engRel(t, [3]uint32{leashSlot, SelfSlot, 1}, [3]uint32{SelfSlot, leashSlot, 2})
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{}, ents, secondGameScript(t, nil, nil, nil), rel)
	if err != nil {
		t.Fatal(err)
	}
	if !playsSecondGame(w) {
		t.Fatal("fixture: not a second-game World")
	}
	guard.AI[0x20], guard.AI[0x38], guard.AI[0x2d], guard.AI[0x45] = guardRouteB, leashBase, leashBase, 1
	binary.LittleEndian.PutUint32(guard.AI[0x48:], 1)
	if ai != nil {
		ai(&guard)
	}
	stand := SavedGroup{ID: 2, Owner: SavedGroupReference{Class: 1, Owner: SelfSlot},
		Members: []SavedGroupMember{{Archive: 1, Entity: leashHero, Bound: true}}}
	stand.AI[0x20], stand.AI[0x45] = orderStandGround, 1
	if err := w.ImportOriginalGroups([]SavedGroup{guard, stand}, nil); err != nil {
		t.Fatal(err)
	}
	return w
}

// leashLatched is a fixture's AI with the latch +0x30 set and working radius r,
// as a file whose Group has run its first-member roll.
func leashLatched(r uint8) func(*SavedGroup) {
	return func(g *SavedGroup) {
		g.AI[0x2d] = r
		binary.LittleEndian.PutUint32(g.AI[0x30:], 1)
	}
}

func leashGroup(t *testing.T, w *World) SavedGroup {
	t.Helper()
	groups, _, present := w.SavedGroups()
	if !present || len(groups) == 0 || groups[0].ID != 1 {
		t.Fatal("the guard Group left the registry")
	}
	return groups[0]
}

func leashAt(w *World, id EntityID) *Entity {
	return &w.entities[indexOfEntity(w.entities, id)]
}

// leashRun steps w through the next n group ticks: each ends on the step
// that runs the script pass and, after it, the Group decisions.
func leashRun(w *World, n int) {
	for range n {
		Step(w, nil)
		for !w.ScriptPassJustRan() {
			Step(w, nil)
		}
	}
}

// leashMove lifts actor id off the map and places it at (x, y), so the
// World's cell slots and traversal follow the move.
func leashMove(t *testing.T, w *World, id EntityID, x, y int32) {
	t.Helper()
	i := indexOfEntity(w.entities, id)
	if !w.takeOffMap(i) || !w.placeAt(i, x, y) {
		t.Fatalf("fixture: actor %d cannot be placed at (%d,%d)", id, x, y)
	}
}

// A second-game guard B member chasing the hero drops the chase on the group
// tick the hero stands beyond the stored radius +0x2d of the centroid, then
// walks to its post on each tick and holds the idle order 0xb there
// (R2-ENGINE-841, R2-ENGINE-842, R2-ENGINE-844 case 6). The latch counter
// +0x34 counts the ticks (R2-ENGINE-843).
func TestSecondGuardLeashEndsThePursuitBeyondTheStoredRadius(t *testing.T) {
	const radius = 7 // the held circle's base + 4 would be 10
	w := leashWorld(t, cell{24, 20}, []cell{{20, 20}}, leashLatched(radius))
	leashRun(w, 1)
	if v, ok := engVictim(w, 2); !ok || v != leashHero {
		t.Fatalf("the member holds victim %v/%v inside the leash, want the hero", v, ok)
	}
	m := leashAt(w, 2)
	post := cell{int32(m.PostX), int32(m.PostY)}
	// The hero stands twelve cells off; the member closes at most four cells
	// before the next tick, so the hero is still beyond the leash then.
	leashMove(t, w, leashHero, m.X+radius+5, m.Y)
	leashRun(w, 1)
	g := leashGroup(t, w)
	h := leashAt(w, leashHero)
	d := cellOf(h).chebyshevTo(cell{int32(g.AI[0x28]), int32(g.AI[0x29])})
	if d <= radius {
		t.Fatalf("fixture: the hero stands %d cells from the centroid, inside the leash %d", d, radius)
	}
	if v, ok := engVictim(w, 2); ok {
		t.Fatalf("the member still chases %v with the hero %d cells from the leash's centre", v, d)
	}
	// The file's latch is set, so each of the two ticks counts.
	if g.AI[0x2d] != radius || binary.LittleEndian.Uint32(g.AI[0x30:]) != 1 || binary.LittleEndian.Uint32(g.AI[0x34:]) != 2 {
		t.Fatalf("radius %d latch %d counter %d, want %d, 1 and 2", g.AI[0x2d], binary.LittleEndian.Uint32(g.AI[0x30:]), binary.LittleEndian.Uint32(g.AI[0x34:]), radius)
	}
	for n := 0; n < 30; n++ {
		m := leashAt(w, 2)
		if m.X == post.x && m.Y == post.y && arrived(*m) {
			break
		}
		if !m.HasTarget || m.TargetX != post.x || m.TargetY != post.y {
			t.Fatalf("decision %d: the member at (%d,%d) walks to (%d,%d)/%v, want its post %v", n, m.X, m.Y, m.TargetX, m.TargetY, m.HasTarget, post)
		}
		leashRun(w, 1)
	}
	leashRun(w, 1)
	m = leashAt(w, 2)
	if m.X != post.x || m.Y != post.y || m.HasAttackTarget {
		t.Fatalf("the member stands at (%d,%d) victim %v, want its post %v", m.X, m.Y, m.HasAttackTarget, post)
	}
	if o := w.savedOrder(2); o == nil || o.Raw[8] != 0xb {
		t.Fatalf("the member at its post holds order %v, want 0xb", o)
	}
}

// The first member's roll draws +0x2d = +0x38 + 4 + U{-1,0,1} once, clears
// the counter and sets the latch; later ticks count and keep the radius. A
// byte-form round trip in the middle keeps the three fields and the next
// tick's decision (R2-ENGINE-843).
func TestSecondGuardRadiusRollsOnceAndSurvivesAColdLoad(t *testing.T) {
	// The hero at (36,20) is past the member's sight and near enough to keep
	// the Group active.
	w := leashWorld(t, cell{36, 20}, []cell{{20, 20}}, nil)
	leashRun(w, 1)
	g := leashGroup(t, w)
	rolled := g.AI[0x2d]
	if rolled < leashBase+3 || rolled > leashBase+5 || binary.LittleEndian.Uint32(g.AI[0x30:]) != 1 || binary.LittleEndian.Uint32(g.AI[0x34:]) != 0 {
		t.Fatalf("after the first tick radius %d latch %d counter %d, want %d..%d, 1 and 0", rolled, binary.LittleEndian.Uint32(g.AI[0x30:]), binary.LittleEndian.Uint32(g.AI[0x34:]), leashBase+3, leashBase+5)
	}
	cold := cloneWorld(t, w)
	leashMove(t, w, leashHero, 20+int32(rolled), 20)
	leashMove(t, cold, leashHero, 20+int32(rolled), 20)
	for n := 1; n <= 3; n++ {
		leashRun(w, 1)
		leashRun(cold, 1)
		a, b := leashGroup(t, w), leashGroup(t, cold)
		if a.AI != b.AI || w.Hash() != cold.Hash() {
			t.Fatalf("tick %d: the cold World's Group or state departs from the live one", n)
		}
		if a.AI[0x2d] != rolled || binary.LittleEndian.Uint32(a.AI[0x34:]) != uint32(n) {
			t.Fatalf("tick %d: radius %d counter %d, want %d and %d", n, a.AI[0x2d], binary.LittleEndian.Uint32(a.AI[0x34:]), rolled, n)
		}
	}
	if v, ok := engVictim(w, 2); !ok || v != leashHero {
		t.Fatalf("the member holds victim %v/%v with the hero at the rolled radius, want the hero", v, ok)
	}
}

// A two-member Group's centroid +0x28 is the members' floored mean cell on
// each tick, and the leash measures from it (R2-ENGINE-841, R2-ENGINE-842).
func TestSecondGuardCentroidMovesWithItsMembers(t *testing.T) {
	const radius = 8
	w := leashWorld(t, cell{36, 20}, []cell{{20, 20}, {21, 20}}, leashLatched(radius))
	leashRun(w, 1)
	if g := leashGroup(t, w); g.AI[0x28] != 20 || g.AI[0x29] != 20 {
		t.Fatalf("centroid (%d,%d), want the floored mean (20,20)", g.AI[0x28], g.AI[0x29])
	}
	// The second member stands at (30,20): the mean is (25,20), and the hero
	// eight cells beyond it is inside the leash, though thirteen from the
	// first member and from the old centre.
	leashMove(t, w, 3, 30, 20)
	b := indexOfEntity(w.entities, 3)
	w.entities[b].PostX, w.entities[b].PostY = 30, 20
	w.syncSavedPost(b)
	leashMove(t, w, leashHero, 25+radius, 20)
	leashRun(w, 1)
	if g := leashGroup(t, w); g.AI[0x28] != 25 || g.AI[0x29] != 20 {
		t.Fatalf("centroid (%d,%d), want (25,20)", g.AI[0x28], g.AI[0x29])
	}
	if v, ok := engVictim(w, 2); !ok || v != leashHero {
		t.Fatalf("the first member holds victim %v/%v with the hero inside the moved leash, want the hero", v, ok)
	}
}

// A struck member does not answer a striker outside the clipped list: the
// second game takes DIV-1523's clipped rule, as the first game does. A
// striker inside the leash is taken on the next group tick.
func TestSecondGuardStrikerOutsideTheLeashIsNotJoined(t *testing.T) {
	const radius = 3
	// Members at (20,20) and (30,20): the centroid is (25,20). The striker's
	// recorded cell (45,20) lies past both members' sight, so only the blow's
	// record lists it, twenty cells from the centroid.
	w := leashWorld(t, cell{45, 20}, []cell{{20, 20}, {30, 20}}, leashLatched(radius))
	// The first tick gives each member its order block, which holds the
	// blow's record from then on.
	leashRun(w, 1)
	w.rememberAttacker(indexOfEntity(w.entities, leashHero), indexOfEntity(w.entities, 3))
	leashRun(w, 1)
	for _, id := range []EntityID{2, 3} {
		if v, ok := engVictim(w, id); ok {
			t.Fatalf("member %d took the striker %v twenty cells from the centroid, beyond the leash %d", id, v, radius)
		}
	}
	leashMove(t, w, leashHero, 26, 20)
	w.rememberAttacker(indexOfEntity(w.entities, leashHero), indexOfEntity(w.entities, 3))
	leashRun(w, 1)
	if v, ok := engVictim(w, 3); !ok || v != leashHero {
		t.Fatalf("the struck member holds victim %v/%v with the striker inside the leash, want the hero", v, ok)
	}
}

// The second game's mission start, which holds the Group registry, clips an
// engaged striker to its circle as well (DIV-1523, DIV-4502).
func TestSecondGameMissionStartDoesNotJoinAStrikerOutsideTheCircle(t *testing.T) {
	// The striker's recorded cell (40,20) lies 19 cells from the members'
	// centre (21,20), past sight and past the held circle.
	a := engFighter(2, leashSlot, 20, 20)
	b := engFighter(3, leashSlot, 22, 20)
	a.Group, b.Group = 1, 1
	hero := engFighter(leashHero, SelfSlot, 40, 20)
	rel := engRel(t, [3]uint32{leashSlot, SelfSlot, 1}, [3]uint32{SelfSlot, leashSlot, 2})
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{}, []Entity{hero, a, b}, secondGameScript(t, nil, nil, nil), rel)
	if err != nil {
		t.Fatal(err)
	}
	if !playsSecondGame(w) || (w.savedGroups == nil) {
		t.Fatal("fixture: not a second-game mission start holding the registry")
	}
	order, base, ok := w.groupState(leashSlot, 1)
	if !ok || order != orderGuard {
		t.Fatalf("fixture: slot 3's Group holds order %d/%v, want Guard", order, ok)
	}
	if int64(noticeRadius(base)) >= 19 {
		t.Fatalf("fixture: the held circle %d reaches the striker", noticeRadius(base))
	}
	w.rememberAttacker(indexOfEntity(w.entities, leashHero), indexOfEntity(w.entities, 3))
	leashRun(w, 1)
	for _, id := range []EntityID{2, 3} {
		if v, ok := engVictim(w, id); ok {
			t.Fatalf("member %d took the striker %v outside the circle", id, v)
		}
	}
}
