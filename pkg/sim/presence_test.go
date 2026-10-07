package sim

// Map presence (0164): instants 16, 17, 18, 32 and 33, the off-map bit they
// move, and the six readers that stop seeing an entity carrying it.
//
// Every fixture is a world and a hand-built script, on scriptitem_test.go's own
// precedent: the arms take an entity id, a group id and a cell, all three plain
// state in this tree, so nothing here needs a map to exercise it.

import (
	"reflect"
	"testing"
)

// prBounds is a small square. It is deliberately not cyBounds: several cases
// here fence a unit in with neighbours, and a 5x5 makes the fence three
// entities rather than eight.
var prBounds = Bounds{Width: 8, Height: 8}

const (
	prFirst  EntityID = 1
	prSecond EntityID = 2
	prThird  EntityID = 3
)

// prUnit is a plain living entity at (x, y) in the group named.
func prUnit(id EntityID, x, y int32, group uint32) Entity {
	return Entity{ID: id, X: x, Y: y, Group: group, HP: 10, MaxHP: 10, Reach: 1}
}

// prWorld is a world running s over the entities given.
func prWorld(t *testing.T, s *Script, ents ...Entity) *World {
	t.Helper()
	w, err := NewTerrainWorld(1, prBounds, ModeCanonical, Terrain{}, ents, s)
	if err != nil {
		t.Fatalf("NewTerrainWorld: %v", err)
	}
	return w
}

// prEnts is the world's entity records, and it is the "changed nothing"
// instrument every refusal here is measured with.
//
// IT IS NOT THE DIGEST, and that is a correction worth stating: World.Hash
// covers the tick counter and the generator state as well as the entities, both
// of which move on any advance and one of which these arms deliberately draw
// from. A refusal checked against the digest would be measuring the script
// cycle, not the arm.
func prEnts(w *World) []Entity { return w.Entities() }

// prAt reads one entity back by id.
func prAt(t *testing.T, w *World, id EntityID) Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("this world holds no entity %d", id)
	return Entity{}
}

// The five compiled records under test, spelt once each.
func prOffNode(u EntityID) ScriptInstant {
	return ScriptInstant{Op: ScriptInstantTakeOffMap, Unit: u, HasUnit: true}
}

func prOnNode(u EntityID) ScriptInstant {
	return ScriptInstant{Op: ScriptInstantReturnToMap, Unit: u, HasUnit: true}
}

func prSwapNode(a, b EntityID) ScriptInstant {
	return ScriptInstant{Op: ScriptInstantSwapOnMap, Unit: a, HasUnit: true, Unit2: b, HasUnit2: true}
}

func prGroupOffNode(g uint32) ScriptInstant {
	return ScriptInstant{Op: ScriptInstantGroupOffMap, Group: g, HasGroup: true}
}

func prGroupOnNode(g uint32) ScriptInstant {
	return ScriptInstant{Op: ScriptInstantGroupOnMap, Group: g, HasGroup: true}
}

// ---------------------------------------------------------------- AC-1

// TestTheRemovalSetsTheBitAndChangesNothingElse is AC-1. The whole entity is
// compared field by field through the record itself: the arm's untouched list
// is everything but the bit, so the check is an equality on the record with the
// bit put back rather than a list of fields somebody remembered to name.
func TestTheRemovalSetsTheBitAndChangesNothingElse(t *testing.T) {
	t.Parallel()

	a := prUnit(prFirst, 3, 3, 4)
	a.Owner = 7
	a.CommandGroup = 40
	w := prWorld(t, fireOnce(t, []ScriptInstant{prOffNode(prFirst)}, 0), a, prUnit(prSecond, 6, 6, 4))
	before := prAt(t, w, prFirst)
	runPass(t, w)

	got := prAt(t, w, prFirst)
	if !got.OffMap {
		t.Fatal("instant 16 left the unit on the map")
	}
	got.OffMap = false
	if got != before {
		t.Errorf("instant 16 changed more than the bit:\n got %+v\nwant %+v", got, before)
	}
	if other := prAt(t, w, prSecond); other.OffMap {
		t.Error("instant 16 took a unit it did not name off the map")
	}
}

// ---------------------------------------------------------------- AC-2

// TestTheRemovalIsIdempotent is AC-2, checked on the DIGEST rather than on the
// bit: a second removal that wrote anything at all would move the hash, and the
// bit alone cannot say whether it wrote the same value twice or wrote nothing.
func TestTheRemovalIsIdempotent(t *testing.T) {
	t.Parallel()

	w := prWorld(t, fireOnce(t, []ScriptInstant{prOffNode(prFirst)}, 0),
		prUnit(prFirst, 3, 3, 4))
	runPass(t, w)
	was := prEnts(w)

	i := indexOfEntity(w.entities, prFirst)
	if w.takeOffMap(i) {
		t.Error("a second removal reported that it changed the world")
	}
	if !reflect.DeepEqual(prEnts(w), was) {
		t.Error("a second removal wrote something")
	}
}

func TestTheRemovalRefusesEveryReferenceItCannotResolve(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what string
		node ScriptInstant
	}{
		{"a node naming no unit", ScriptInstant{Op: ScriptInstantTakeOffMap}},
		{"a node naming an id this world does not hold", prOffNode(99)},
		{"a return naming no unit", ScriptInstant{Op: ScriptInstantReturnToMap}},
		{"a return naming an id this world does not hold", prOnNode(99)},
		{"a swap naming one unit", ScriptInstant{Op: ScriptInstantSwapOnMap, Unit: prFirst, HasUnit: true}},
		{"a group removal naming no group", ScriptInstant{Op: ScriptInstantGroupOffMap}},
		{"a group return naming no group", ScriptInstant{Op: ScriptInstantGroupOnMap}},
	} {
		w := prWorld(t, fireOnce(t, []ScriptInstant{tc.node}, 0), prUnit(prFirst, 3, 3, 4))
		was := prEnts(w)
		runPass(t, w)
		if !reflect.DeepEqual(prEnts(w), was) {
			t.Errorf("%s changed an entity", tc.what)
		}
	}
}

// ---------------------------------------------------------------- AC-3, AC-18

// TestAnOffMapUnitOccupiesNoCell is AC-3, and it is the gate AC-18 reverts.
//
// It drives a REAL ROUTE rather than reading the occupancy plane, because
// counted has three readers — the seed, enterable's self-presence term and the
// flyer re-seed — and a test that seeded a scratch by hand would reach one of
// them. The walker is ordered onto the cell the removed unit stands on, and it
// gets there only if that unit holds no ground.
func TestAnOffMapUnitOccupiesNoCell(t *testing.T) {
	t.Parallel()

	// The blocker stands one cell from the walker, in the walker's way, and is
	// taken off the map on the first script pass.
	blocker := prUnit(prSecond, 4, 3, 9)
	walker := prUnit(prFirst, 3, 3, 4)
	walker.Speed = 10

	w := prWorld(t, fireOnce(t, []ScriptInstant{prOffNode(prSecond)}, 0), walker, blocker)
	runPass(t, w)
	if !prAt(t, w, prSecond).OffMap {
		t.Fatal("the blocker was not taken off the map")
	}

	scriptTicks(w, 40, []Command{{Kind: KindMoveTo, Entity: prFirst, X: 4, Y: 3}})
	if got := prAt(t, w, prFirst); got.X != 4 || got.Y != 3 {
		t.Errorf("the walker stopped at (%d,%d); the cell it was sent to is held by a unit that is off the map",
			got.X, got.Y)
	}
}

// TestAnOnMapUnitStillOccupiesItsCell is AC-18's control and the revert-check's
// other half: the SAME fixture with the removal node replaced by one naming
// nobody. Delete the OffMap test from counted and this case stays green while
// the one above stays green too — which is exactly what makes the pair a
// witness rather than one assertion that could pass for either reason.
func TestAnOnMapUnitStillOccupiesItsCell(t *testing.T) {
	t.Parallel()

	blocker := prUnit(prSecond, 4, 3, 9)
	walker := prUnit(prFirst, 3, 3, 4)
	walker.Speed = 10

	w := prWorld(t, fireOnce(t, []ScriptInstant{prOffNode(99)}, 0), walker, blocker)
	runPass(t, w)

	scriptTicks(w, 40, []Command{{Kind: KindMoveTo, Entity: prFirst, X: 4, Y: 3}})
	if got := prAt(t, w, prFirst); got.X == 4 && got.Y == 3 {
		t.Error("the walker stands on a cell an ON-MAP unit holds")
	}
}

// ---------------------------------------------------------------- AC-4

// TestAnOffMapHostileIsNotAcquired is AC-4: a group that would otherwise
// attack acquires nothing once its only candidate has left the map.
//
// IT CARRIES ITS OWN POSITIVE CONTROL, in the same fixture and before the
// removal, and the control is not decoration: written without one, this case
// passed with the acquisition gate DELETED, because the hunter it was built on
// never acquired its prey at any distance. An "acquires nothing" assertion is
// worth exactly as much as the proof that the same fixture acquires something.
func TestAnOffMapHostileIsNotAcquired(t *testing.T) {
	t.Parallel()

	fixture := func(t *testing.T) *World {
		t.Helper()
		hunter := engFighter(prFirst, 1, 3, 3)
		hunter.ScanRange = engSight
		prey := engFighter(prSecond, 2, 4, 3)
		w, err := NewRelatedWorld(1, prBounds, ModeCanonical, Terrain{},
			[]Entity{hunter, prey}, nil, acEnemies(t))
		if err != nil {
			t.Fatalf("NewRelatedWorld: %v", err)
		}
		return w
	}

	// The control: with the prey ON the map, this hunter acquires it.
	control := fixture(t)
	control.engagementPass()
	if _, held := engVictim(control, prFirst); !held {
		t.Fatal("the hunter acquires nothing even with its prey on the map; " +
			"this case would pass for the wrong reason")
	}

	// And with the prey off the map, the same decision finds no candidate.
	w := fixture(t)
	w.takeOffMap(indexOfEntity(w.entities, prSecond))
	w.engagementPass()
	if v, held := engVictim(w, prFirst); held {
		t.Errorf("the hunter acquired %d, a unit that is not on the map", v)
	}
}

// ---------------------------------------------------------------- AC-5

// TestAVictimTakenOffTheMapIsDropped is AC-5. The fight is allowed to start
// first, so the attacker is holding a victim at the moment the script removes
// it; the drop is then the removal's consequence and not a fight that never
// began.
func TestAVictimTakenOffTheMapIsDropped(t *testing.T) {
	t.Parallel()

	hunter := engFighter(prFirst, 1, 3, 3)
	hunter.ScanRange = engSight
	prey := engFighter(prSecond, 2, 4, 3)

	w, err := NewRelatedWorld(1, prBounds, ModeCanonical, Terrain{},
		[]Entity{hunter, prey}, nil, acEnemies(t))
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	w.engagementPass()
	if _, held := engVictim(w, prFirst); !held {
		t.Fatal("the hunter never acquired its victim; this case measures losing one")
	}

	w.takeOffMap(indexOfEntity(w.entities, prSecond))
	Step(w, nil)

	if v, held := engVictim(w, prFirst); held {
		t.Errorf("the attacker still holds %d, a victim that has left the map", v)
	}
}

func TestAnOffMapMemberIsNotDecidedFor(t *testing.T) {
	t.Parallel()

	fixture := func(t *testing.T) *World {
		t.Helper()
		hunter := engFighter(prFirst, 1, 3, 3)
		hunter.ScanRange = engSight
		prey := engFighter(prSecond, 2, 4, 3)
		w, err := NewRelatedWorld(1, prBounds, ModeCanonical, Terrain{},
			[]Entity{hunter, prey}, nil, acEnemies(t))
		if err != nil {
			t.Fatalf("NewRelatedWorld: %v", err)
		}
		return w
	}

	control := fixture(t)
	control.engagementPass()
	if _, held := engVictim(control, prFirst); !held {
		t.Fatal("the hunter decides on nothing even on the map; this case would pass for the wrong reason")
	}

	w := fixture(t)
	w.takeOffMap(indexOfEntity(w.entities, prFirst))
	w.engagementPass()
	if v, held := engVictim(w, prFirst); held {
		t.Errorf("a group decided for an off-map member and gave it %d to attack", v)
	}
}

// ---------------------------------------------------------------- AC-6

// TestAnOffMapUnitIsNotAdvanced is AC-6: a unit under a move order that is then
// taken off the map holds its coordinates for as many ticks as the walk would
// have taken.
func TestAnOffMapUnitIsNotAdvanced(t *testing.T) {
	t.Parallel()

	walker := prUnit(prFirst, 2, 2, 4)
	walker.Speed = 10
	w := prWorld(t, nil, walker)

	Step(w, []Command{{Kind: KindMoveTo, Entity: prFirst, X: 6, Y: 6}})
	w.takeOffMap(indexOfEntity(w.entities, prFirst))
	at := prAt(t, w, prFirst)

	for range 40 {
		Step(w, nil)
	}
	got := prAt(t, w, prFirst)
	if got.X != at.X || got.Y != at.Y {
		t.Errorf("an off-map unit walked from (%d,%d) to (%d,%d)", at.X, at.Y, got.X, got.Y)
	}
}

// ---------------------------------------------------------------- AC-7

func TestAGroupsCountIsWholeWithItsMembersOffTheMap(t *testing.T) {
	t.Parallel()

	const g uint32 = 4
	s := mustScript(t,
		[]ScriptCheck{
			{Op: ScriptCheckGroupCount, Register: 0, Group: g, HasGroup: true},
			constCheck(1, 2),
		},
		[]ScriptInstant{prGroupOffNode(g)},
		[]ScriptTrigger{
			// Trigger 0 removes the group unconditionally on the first pass.
			{Pairs: [3]ScriptPair{pair(1, 1, ScriptCmpEQ)}, Instants: acts(0), Once: true, Latch: 0},
			// Trigger 1 latches only while the count still reads 2.
			{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)}, Latch: 1},
		})
	w := prWorld(t, s, prUnit(prFirst, 3, 3, g), prUnit(prSecond, 4, 4, g), prUnit(prThird, 6, 6, 9))

	runPass(t, w)
	if !prAt(t, w, prFirst).OffMap || !prAt(t, w, prSecond).OffMap {
		t.Fatal("the group removal left a member on the map")
	}
	// A second pass, so the count is read with both members already removed.
	scriptTicks(w, 32, nil)
	if !w.ScriptLatched(1) {
		t.Error("the group count stopped answering 2 once its members left the map; membership is not presence")
	}
	if prAt(t, w, prFirst).Group != g || prAt(t, w, prSecond).Group != g {
		t.Error("the group removal changed a member's group")
	}
}

// ---------------------------------------------------------------- AC-8

// TestTheReturnPutsTheUnitBackOnItsRetainedCell is AC-8: nothing authors a
// cell, so the unit lands on the one it never stopped carrying.
func TestTheReturnPutsTheUnitBackOnItsRetainedCell(t *testing.T) {
	t.Parallel()

	w := prWorld(t, fireOnce(t, []ScriptInstant{prOffNode(prFirst), prOnNode(prFirst)}, 0, 1),
		prUnit(prFirst, 3, 5, 4))
	runPass(t, w)

	got := prAt(t, w, prFirst)
	if got.OffMap {
		t.Fatal("the return left the unit off the map")
	}
	if got.X != 3 || got.Y != 5 {
		t.Errorf("the unit returned to (%d,%d), want the cell it was removed from, (3,5)", got.X, got.Y)
	}
}

// ---------------------------------------------------------------- AC-9

// TestAReturnWithNoFreeCellChangesNothing is AC-9. The retained cell and every
// cell the search can reach — the 4x4 window the random attempts span and the
// 3x3 the exhaustive stage scans, which the window contains — are filled with
// other units, so every attempt fails.
func TestAReturnWithNoFreeCellChangesNothing(t *testing.T) {
	t.Parallel()

	// The removed unit sits at (3,3). Fill x in 2..5 and y in 2..5, which is
	// [x-1, x+2] x [y-1, y+2] — the window the six random attempts span, and
	// which contains the 3x3 the exhaustive stage scans.
	//
	// (3,3) IS FILLED TOO, and it is the cell that matters: the removed unit
	// holds no ground once it is off the map, so the retained cell the search
	// tries first is free unless somebody else is standing on it.
	ents := []Entity{prUnit(prFirst, 3, 3, 4)}
	id := EntityID(10)
	for x := int32(2); x <= 5; x++ {
		for y := int32(2); y <= 5; y++ {
			ents = append(ents, prUnit(id, x, y, 9))
			id++
		}
	}

	w := prWorld(t, fireOnce(t, []ScriptInstant{prOffNode(prFirst)}, 0), ents...)
	runPass(t, w)
	if !prAt(t, w, prFirst).OffMap {
		t.Fatal("the removal did not take")
	}
	was := prEnts(w)
	wasSeed := w.rng.state

	if w.returnToMap(indexOfEntity(w.entities, prFirst)) {
		t.Fatal("the return reported success with every reachable cell occupied")
	}
	if !reflect.DeepEqual(prEnts(w), was) {
		t.Error("a failed return changed an entity")
	}
	if w.rng.state == wasSeed {
		t.Error("a failed return drew nothing; the search must consume its draws whether or not it succeeds")
	}
}

// ---------------------------------------------------------------- AC-10

// TestAReturnFallsBackToTheNeighbourhood is AC-10: the retained cell is taken,
// so the unit comes back somewhere inside the window the search reaches.
func TestAReturnFallsBackToTheNeighbourhood(t *testing.T) {
	t.Parallel()

	w := prWorld(t, fireOnce(t, []ScriptInstant{prOffNode(prFirst)}, 0),
		prUnit(prFirst, 3, 3, 4), prUnit(prSecond, 3, 3, 9))
	runPass(t, w)

	if !w.returnToMap(indexOfEntity(w.entities, prFirst)) {
		t.Fatal("the return failed with a whole neighbourhood free")
	}
	got := prAt(t, w, prFirst)
	if got.OffMap {
		t.Fatal("the return reported success and left the bit set")
	}
	if got.X < 2 || got.X > 5 || got.Y < 2 || got.Y > 5 {
		t.Errorf("the unit came back at (%d,%d), outside the window the search reaches", got.X, got.Y)
	}
	if got.X == 3 && got.Y == 3 {
		t.Error("the unit came back onto the cell another unit holds")
	}
}

// ---------------------------------------------------------------- AC-11, AC-12

// TestTheSwapRemovesTheFirstAndPlacesTheSecond is AC-11.
func TestTheSwapRemovesTheFirstAndPlacesTheSecond(t *testing.T) {
	t.Parallel()

	w := prWorld(t, fireOnce(t, []ScriptInstant{prSwapNode(prFirst, prSecond)}, 0),
		prUnit(prFirst, 3, 3, 4), prUnit(prSecond, 6, 6, 9))
	runPass(t, w)

	gone := prAt(t, w, prFirst)
	if !gone.OffMap {
		t.Error("the swap left its first unit on the map")
	}
	if gone.X != 3 || gone.Y != 3 {
		t.Errorf("the swap moved its first unit to (%d,%d); a removal writes no position", gone.X, gone.Y)
	}
	back := prAt(t, w, prSecond)
	if back.OffMap {
		t.Error("the swap left its second unit off the map")
	}
	if back.X < 2 || back.X > 5 || back.Y < 2 || back.Y > 5 {
		t.Errorf("the second unit stands at (%d,%d), outside the window around the first's cell (3,3)",
			back.X, back.Y)
	}
}

// TestTheSwapRefusesOneUnitNamedTwice is AC-12's second clause. The two
// references resolving to one entity is refused whole, on the give-all's own
// ground, so the named unit is neither removed nor placed.
func TestTheSwapRefusesOneUnitNamedTwice(t *testing.T) {
	t.Parallel()

	w := prWorld(t, fireOnce(t, []ScriptInstant{prSwapNode(prFirst, prFirst)}, 0),
		prUnit(prFirst, 3, 3, 4))
	was := prEnts(w)
	runPass(t, w)

	if !reflect.DeepEqual(prEnts(w), was) {
		t.Error("a swap naming one unit twice changed an entity")
	}
	if prAt(t, w, prFirst).OffMap {
		t.Error("a swap naming one unit twice took it off the map")
	}
}

// ---------------------------------------------------------------- AC-13, AC-14

// TestTheGroupArmsRemoveAndReturnEveryMember is AC-13's first clause and AC-14.
// One pass removes the group, a second returns it, and the members that were
// never in it are untouched throughout.
func TestTheGroupArmsRemoveAndReturnEveryMember(t *testing.T) {
	t.Parallel()

	const g uint32 = 4
	s := mustScript(t,
		[]ScriptCheck{constCheck(0, 1), constCheck(1, 2)},
		[]ScriptInstant{prGroupOffNode(g), prGroupOnNode(g)},
		[]ScriptTrigger{
			{Pairs: [3]ScriptPair{pair(0, 0, ScriptCmpEQ)}, Instants: acts(0), Once: true, Latch: 0},
			// Latch 1 is written only after latch 0 has, because the second
			// trigger's condition is the first's own latch register read back.
			{Pairs: [3]ScriptPair{pair(1, 1, ScriptCmpEQ)}, Instants: acts(1), Once: true, Latch: 1},
		})
	w := prWorld(t, s, prUnit(prFirst, 3, 3, g), prUnit(prSecond, 6, 6, g), prUnit(prThird, 1, 1, 9))

	// The whole script fires in one pass: trigger 0 removes, trigger 1 returns.
	runPass(t, w)

	for _, id := range []EntityID{prFirst, prSecond} {
		if got := prAt(t, w, id); got.OffMap {
			t.Errorf("entity %d is still off the map after the group return", id)
		}
	}
	if got := prAt(t, w, prThird); got.OffMap || got.X != 1 || got.Y != 1 {
		t.Error("the group arms touched an entity outside the named group")
	}
	// Each member searched from its OWN retained cell, so each is back where it
	// was: nothing is carried between members.
	if got := prAt(t, w, prFirst); got.X != 3 || got.Y != 3 {
		t.Errorf("the first member returned to (%d,%d), want (3,3)", got.X, got.Y)
	}
	if got := prAt(t, w, prSecond); got.X != 6 || got.Y != 6 {
		t.Errorf("the second member returned to (%d,%d), want (6,6)", got.X, got.Y)
	}
}

func TestAGroupArmNamingAnUnusedGroupChangesNothing(t *testing.T) {
	t.Parallel()

	w := prWorld(t, fireOnce(t, []ScriptInstant{prGroupOffNode(77)}, 0),
		prUnit(prFirst, 3, 3, 4))
	was := prEnts(w)
	runPass(t, w)
	if !reflect.DeepEqual(prEnts(w), was) {
		t.Error("a group removal naming a group no entity carries changed an entity")
	}
}

// ---------------------------------------------------------------- AC-15

// TestTheOffMapBitSurvivesTheByteForm is AC-15: the round trip carries it, and
// the digest separates a world holding it from one that does not.
func TestTheOffMapBitSurvivesTheByteForm(t *testing.T) {
	t.Parallel()

	w := prWorld(t, nil, prUnit(prFirst, 3, 3, 4), prUnit(prSecond, 6, 6, 9))
	on := w.Hash()
	w.takeOffMap(indexOfEntity(w.entities, prFirst))
	off := w.Hash()

	if on == off {
		t.Error("a world with a unit off the map hashes like one with it on the map")
	}

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if !prAt(t, &back, prFirst).OffMap {
		t.Error("the decoded world put the removed unit back on the map")
	}
	if prAt(t, &back, prSecond).OffMap {
		t.Error("the decoded world took a unit off the map that was never removed")
	}
	if back.Hash() != off {
		t.Error("the round trip changed the digest")
	}
}

func TestTheOffMapByteIsRefusedOutsideZeroAndOne(t *testing.T) {
	t.Parallel()

	w := prWorld(t, nil, prUnit(prFirst, 3, 3, 4))
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	// The single entity record's own off-map byte: the header, the three
	// planes, then +223 into the first record — 0166 widened the spell mark's
	// remaining ticks to a word in front of it, moving this byte up one.
	o := headerLen + 3*int(gridCells(prBounds)) + 223
	form[o] = 2

	var back World
	if err := back.UnmarshalBinary(form); err == nil {
		t.Error("an off-map byte of 2 was accepted")
	}
}

// ---------------------------------------------------------------- AC-17

// TestTheFivePresenceOpcodesAreNoLongerUnsupported is AC-17: the compile-time
// report stops naming them, which is the whole of the census fall. It asks the
// support table AND builds a script, so the answer the report gives and the
// answer the dispatch gives are both witnessed.
func TestTheFivePresenceOpcodesAreNoLongerUnsupported(t *testing.T) {
	t.Parallel()

	ops := []int32{
		ScriptInstantTakeOffMap, ScriptInstantReturnToMap, ScriptInstantSwapOnMap,
		ScriptInstantGroupOffMap, ScriptInstantGroupOnMap,
	}
	var nodes []ScriptInstant
	for _, op := range ops {
		if !scriptInstantSupported(ScriptInstant{Op: op}) {
			t.Errorf("opcode %d is still reported unsupported", op)
		}
		nodes = append(nodes, ScriptInstant{Op: op})
	}

	s := fireOnce(t, nodes, 0, 1, 2, 3)
	for _, g := range s.Unsupported() {
		for _, op := range ops {
			if g.Kind == ScriptGapInstant && g.Op == op {
				t.Errorf("the compiled script still reports opcode %d as unsupported", op)
			}
		}
	}
	if len(s.InertTriggers()) != 0 {
		t.Error("a trigger naming these nodes is marked inert; implementing an instant arms no trigger and inerts none")
	}
}
