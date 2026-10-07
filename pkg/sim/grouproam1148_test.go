package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func widenedGroupRoamPin(old []byte) []byte {
	out := append(bytes.Clone(old), 0, 0, 0, 0)
	out[0] = 87
	return widenedScorchedPin(out)
}
func strippedGroupRoamPin(form []byte) []byte {
	out := strippedScorchedPin(form)
	if len(out) > 0 && out[0] >= 87 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		out = out[:len(out)-4-span]
		out[0] = 86
	}
	return out
}

func roamWorld1148(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(17, Bounds{Width: 80, Height: 80}, ModeCanonical, nil, []Entity{
		{ID: 1, Owner: 9, Group: 19, X: 30, Y: 30, HP: 100, MaxHP: 100},
		{ID: 2, Owner: 9, Group: 19, X: 31, Y: 30, HP: 100, MaxHP: 100},
	})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestGroupRoam1148CommandAndColdContinuation(t *testing.T) {
	w := roamWorld1148(t)
	if !groupOrderSupported(17) || groupOrderSupported(18) {
		t.Fatal("wrong support table")
	}
	w.groups[0].roamCounter = 50
	w.entities[0].HasTarget, w.entities[0].TargetX, w.entities[0].TargetY = true, 50, 50
	w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, HasGroup: true, Group: 19, Args: [10]int32{17, 70, 60}})
	g := w.groups[0]
	if g.order != 17 || g.commandedX != 30 || g.commandedY != 30 || g.roamCounter != 50 || w.entities[0].HasTarget {
		t.Fatal("command17 seed/stop/counter", g)
	}
	for range 130 {
		cold := retreatRoundTrip1089(t, w)
		Step(w, nil)
		Step(cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatal("Roam differs after cold LOAD")
		}
		for _, e := range w.entities {
			if e.HasTarget || e.X < 30 || e.X > 31 || e.Y != 30 {
				t.Fatal("Group cell was invented as member destination", e)
			}
		}
	}
	if w.groups[0].roamCounter == 50 || w.groups[0].commandedX == 30 && w.groups[0].commandedY == 30 {
		t.Fatal("Roam never evaluated")
	}
}

func TestGroupRoam1148RerollBoundariesAndNoValidCell(t *testing.T) {
	for _, tc := range []struct {
		distance int32
		counter  byte
		roll     bool
	}{{9, 50, true}, {10, 50, false}, {10, 51, true}} {
		w := roamWorld1148(t)
		w.entities[0].X = 30 + tc.distance
		before := w.rng
		dst, c, ok := w.rollRoamCell(cell{30, 30}, []int{0}, tc.counter)
		if !ok || (w.rng != before) != tc.roll || tc.roll && (c != 0 || dst == (cell{30, 30})) || !tc.roll && (c != 50 || dst != (cell{30, 30})) {
			t.Fatal("reroll threshold", tc, dst, c, ok)
		}
	}
	w, err := NewWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, nil, []Entity{{ID: 1, Owner: 9, Group: 19, X: 8, Y: 8}})
	if err != nil {
		t.Fatal(err)
	}
	w.cmdGroupOrder(ScriptInstant{HasGroup: true, Group: 19, Args: [10]int32{17}})
	before := w.Hash()
	w.decide(aiGroup{owner: 9, group: 19, members: []int{0}})
	if w.Hash() != before {
		t.Fatal("impossible reroll mutated or hung")
	}
}

func TestGroupRoam1148SavedSetterPreservesOtherRawFields(t *testing.T) {
	w := savedGroupWorld(t)
	g := &w.savedGroups.Groups[0]
	before := g.AI
	first := w.entities[indexOfEntity(w.entities, g.Members[0].Entity)]
	g.AI[0x15] = 37
	before[0x15] = 37
	w.cmdGroupOrder(ScriptInstant{HasGroup: true, Group: 19, Args: [10]int32{17, 70, 60}})
	before[0x20] = 17
	binary.LittleEndian.PutUint16(before[10:], uint16(first.X)|uint16(first.Y)<<8)
	if g.AI != before {
		t.Fatal("saved Group writes differ")
	}
	if !reflect.DeepEqual(retreatRoundTrip1089(t, w).savedGroups, w.savedGroups) {
		t.Fatal("saved command lost on LOAD")
	}
}

func TestGroupRoam1148LiteralFooterAndAtomicRefusals(t *testing.T) {
	w := roamWorld1148(t)
	w.groups[0].roamCounter = 51
	b := strippedScorchedPin(mustMarshal(t, w))
	if b[0] != 87 || !bytes.Equal(b[len(b)-13:], []byte{9, 0, 0, 0, 19, 0, 0, 0, 51, 9, 0, 0, 0}) {
		t.Fatal("literal Group counter footer")
	}
	for _, mutate := range []func([]byte){
		func(b []byte) { b[len(b)-5] = 0 },
		func(b []byte) { b[len(b)-13] = 8 },
		func(b []byte) { binary.LittleEndian.PutUint32(b[len(b)-4:], 0xffffffff) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[len(b)-4:], 8) },
	} {
		bad := bytes.Clone(b)
		mutate(bad)
		before := w.Hash()
		if w.UnmarshalBinary(widenedScorchedPin(bad)) == nil || w.Hash() != before {
			t.Fatal("non-atomic malformed Group counter")
		}
	}
}

func TestGroupRoam1148Historical86Pins(t *testing.T) {
	for _, tc := range []struct {
		form []byte
		hash uint64
	}{{pinBytes, 0x346cf600ac185ef1}, {rtfBytes, 0x285c8ca36b96e898}} {
		old := strippedGroupRoamPin(tc.form)
		// pinBytes and rtfBytes both close on entityIDFloorLen=10 (see their own
		// var declarations); the widen chain up to widenedAttackNoticePin stops
		// at form93 to keep areaHeaderDigest1164's frozen digest untouched, so
		// form94's own outermost floor wrap is applied here explicitly.
		if old[0] != 86 || fnv1a(old) != tc.hash || !bytes.Equal(widenedEntityIDFloorPin(widenedGroupRoamPin(old), 10), tc.form) {
			t.Fatal("historical form86 changed outside the new footer/tag")
		}
	}
}

func importedRoamWorld1148(t *testing.T) *World {
	t.Helper()
	w := roamWorld1148(t)
	g := SavedGroup{ID: 71, Selector: 19, Members: []SavedGroupMember{{2, 2, true}, {1, 1, true}}}
	g.AI[0x45], g.AI[0x20], g.AI[0x15] = 1, 3, 50
	orders := []SavedActorOrder{{Entity: 1, State: 0xb}, {Entity: 2, State: 0xb}}
	for i := range orders {
		orders[i].Raw[10], orders[i].Raw[11] = uint8(30+i), 30
	}
	if err := w.ImportSavedGroups([]SavedGroup{g}, orders); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestGroupRoam1148CompiledTriggerAndPersistence(t *testing.T) {
	for _, imported := range []bool{false, true} {
		w := roamWorld1148(t)
		if imported {
			w = importedRoamWorld1148(t)
		}
		w.script = mustScript(t, []ScriptCheck{constCheck(0, 1), constCheck(1, 1)},
			[]ScriptInstant{{Op: ScriptInstantGroupOrder, HasGroup: true, Group: 19, Args: [10]int32{17, 70, 70}}},
			[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)}, Instants: acts(0), Once: true}})
		if len(w.script.Unsupported()) != 0 {
			t.Fatal("compiled command17 still reports a gap")
		}
		w.scriptPass(nil)
		if w.latches[0] == 0 || !imported && w.groups[0].order != 17 || imported && w.savedGroups.Groups[0].AI[0x20] != 17 {
			t.Fatal("trigger did not reach command17", imported)
		}
		for range 130 {
			cold := retreatRoundTrip1089(t, w)
			Step(w, nil)
			Step(cold, nil)
			if w.Hash() != cold.Hash() {
				t.Fatal("compiled Roam continuation diverged", imported)
			}
		}
	}
}

func TestGroupRoam1148RefusedSelectorIsAtomic(t *testing.T) {
	for _, fault := range []string{"no-selector", "absent", "empty", "unbound", "duplicate"} {
		w := savedGroupWorld(t)
		in := ScriptInstant{HasGroup: true, Group: 19, Args: [10]int32{17}}
		switch fault {
		case "no-selector":
			in.HasGroup = false
		case "absent":
			in.Group = 987
		case "empty":
			in.Group = 88
		case "unbound":
			w.savedGroups.Groups[0].Members[1].Bound = false
		case "duplicate":
			w.savedGroups.Groups[1].Selector = 19
		}
		before := w.Hash()
		w.cmdGroupOrder(in)
		if w.Hash() != before {
			t.Fatal("refused command partly changed state", fault)
		}
	}
}

func TestGroupRoam1148KeepsActorStateButSupersedesPatrol(t *testing.T) {
	w := roamWorld1148(t)
	w.cmdGroupOrder(ScriptInstant{HasGroup: true, Group: 19, Args: [10]int32{14, 50, 50}})
	before := w.entities[0]
	w.cmdGroupRoam(19)
	e := w.entities[0]
	if e.ActorState != actorStatePatrol || e.PatrolHeadX != before.PatrolHeadX || e.PatrolTailX != before.PatrolTailX || e.PatrolLeg != before.PatrolLeg || e.HasTarget {
		t.Fatal("command17 rebuilt or continued the old patrol ring")
	}
	for range 32 {
		Step(w, nil)
	}
	if w.entities[0].X != 30 || w.entities[0].Y != 30 || w.entities[0].HasTarget {
		t.Fatal("old actor patrol decided alongside Roam")
	}
	w.commandPatrol([]int{0}, cell{50, 50})
	for range 32 {
		Step(w, nil)
	}
	if w.entities[0].X == 30 && w.entities[0].Y == 30 && !w.entities[0].HasTarget {
		t.Fatal("new player patrol did not leave the Roam group")
	}
	_ = retreatRoundTrip1089(t, w)
}

func TestGroupRoam1148CounterThresholdAndReplacement(t *testing.T) {
	w := roamWorld1148(t)
	w.groups[0].order, w.groups[0].roamCounter = 17, 50
	w.groups[0].commandedX, w.groups[0].commandedY = 50, 50
	g := aiGroup{owner: 9, group: 19, members: []int{0, 1}}
	w.decide(g)
	if w.groups[0].roamCounter != 51 || w.groups[0].commandedX != 50 || w.groups[0].commandedY != 50 {
		t.Fatal("counter50 did not increment after evaluation")
	}
	w.decide(g)
	if w.groups[0].roamCounter != 1 || w.groups[0].commandedX == 50 && w.groups[0].commandedY == 50 {
		t.Fatal("counter51 did not reroll/reset then increment")
	}
	w.groups[0].roamCounter = 255
	w.cmdGroupOrder(ScriptInstant{HasGroup: true, Group: 19, Args: [10]int32{3}})
	if retreatRoundTrip1089(t, w).groups[0].roamCounter != 255 {
		t.Fatal("replacement order lost retained counter")
	}
	w.cmdGroupRoam(19)
	w.decide(g)
	if w.groups[0].roamCounter != 1 {
		t.Fatal("counter255 did not reset before increment")
	}
}

func TestGroupRoam1148BoundedRerollFallback(t *testing.T) {
	w, err := NewWorld(94, Bounds{17, 40}, ModeCanonical, nil, []Entity{{ID: 1, X: 8, Y: 8}})
	if err != nil {
		t.Fatal(err)
	}
	// Only south reaches the inset rectangle. This literal SplitMix64 seed
	// draws no south direction in its first32 calls; the state is seed+32*gamma.
	dst, counter, ok := w.rollRoamCell(cell{8, 8}, []int{0}, 0)
	if !ok || dst != (cell{8, 28}) || counter != 0 || w.rng.state != 0xc6ef372fe94f82fe {
		t.Fatal("bounded fallback did not terminate at32 draws", dst, counter, w.rng)
	}
}

func TestGroupRoam1148SavedStopWritesAndRetainedOrder(t *testing.T) {
	w := importedRoamWorld1148(t)
	o := w.savedOrder(1)
	o.State, o.Patrol = 0xa, []uint16{0x1e1e, 0x3232}
	binary.LittleEndian.PutUint16(o.Raw[2:], 0x3232)
	o.Raw[8], o.Raw[0x14] = 5, 99
	for _, at := range []int{0x38, 0x50, 0x60} {
		binary.LittleEndian.PutUint32(o.Raw[at:], 0x12345678)
	}
	before := *o
	before.Patrol = append([]uint16(nil), o.Patrol...)
	w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{Entity: 1, ActorAction: 7}}}
	binary.LittleEndian.PutUint32(w.motionFor(1).Mover[0x7c:], 0xffffffff)
	w.cmdGroupRoam(19)
	want := before
	want.Raw[0x14] = uint8(w.entities[0].Reach)
	for _, at := range []int{0x38, 0x50, 0x60} {
		clear(want.Raw[at : at+4])
	}
	if !reflect.DeepEqual(*w.savedOrder(1), want) || w.motionFor(1).ActorAction != 0 || binary.LittleEndian.Uint32(w.motionFor(1).Mover[0x7c:]) != 0 {
		t.Fatal("STOP changed state/ring/destination or omitted a retained field")
	}
}

func TestGroupRoam1148SameSelectorAcrossNativeOwners(t *testing.T) {
	w, err := NewWorld(17, Bounds{80, 80}, ModeCanonical, nil, []Entity{
		{ID: 8, Owner: 2, Group: 19, X: 40, Y: 40},
		{ID: 7, Owner: 2, Group: 19, X: 30, Y: 30},
		{ID: 6, Owner: 3, Group: 19, X: 50, Y: 50},
	})
	if err != nil {
		t.Fatal(err)
	}
	w.groups[0].roamCounter, w.groups[1].roamCounter = 33, 44
	w.cmdGroupRoam(19)
	if w.groups[0].commandedX != 0 || w.groups[0].order == 17 || w.groups[1].commandedX != 50 || w.groups[1].order != 17 {
		t.Fatal("selector reached the earlier owner or missed the later owner's own first member")
	}
	cold := retreatRoundTrip1089(t, w)
	if !reflect.DeepEqual(w.groups, cold.groups) {
		t.Fatal("owner/group counter binding changed on LOAD")
	}
	b := mustMarshal(t, w)
	for _, edit := range []func([]byte){
		func(b []byte) { copy(b[len(b)-29:len(b)-21], b[len(b)-38:len(b)-30]) },
		func(b []byte) { b[len(b)-29] = 1 },
	} {
		bad := bytes.Clone(b)
		edit(bad)
		before := cold.Hash()
		if err := cold.UnmarshalBinary(bad); err == nil || before != cold.Hash() {
			t.Fatal("late duplicate/descending counter partly adopted")
		}
	}
}

func TestGroupRoam1148WithdrawalKeepsEvaluationAcrossSave(t *testing.T) {
	a := withdrawalFighter(1, 2, 30, 30, 100)
	a.Group, a.Withdraw, a.Speed, a.ScanRange = 19, 100, 1, 20
	b := withdrawalFighter(2, 3, 31, 30, 100)
	b.ScanRange = 0
	w, err := NewRelatedWorld(17, Bounds{80, 80}, ModeCanonical,
		Terrain{}, []Entity{a, b}, nil, engRel(t, [3]uint32{2, 3, 1}))
	if err != nil {
		t.Fatal(err)
	}
	w.script = mustScript(t,
		[]ScriptCheck{constCheck(0, 1), constCheck(1, 1)},
		[]ScriptInstant{{Op: ScriptInstantGroupOrder, HasGroup: true,
			Group: 19, Args: [10]int32{17}}},
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)},
			Instants: acts(0), Once: true}})
	w.tick = scriptPassPhase
	Step(w, nil)
	if w.latches[0] == 0 || !w.entities[0].HasTarget || w.entities[0].HasAttackTarget {
		t.Fatal("command17 did not reach automatic withdrawal")
	}
	before := w.groups[0].roamCounter
	cold := retreatRoundTrip1089(t, w)
	for range 16 {
		Step(w, nil)
		Step(cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatal("withdrawal Roam changed after cold reload")
		}
	}
	if w.groups[0].roamCounter == before {
		t.Fatal("withdrawal removed the live Roam group from evaluation")
	}
}

func TestGroupRoam1148MovingMemberCountsTowardDistance(t *testing.T) {
	w := roamWorld1148(t)
	w.groups[0].order, w.groups[0].roamCounter = orderRoam, 50
	w.groups[0].commandedX, w.groups[0].commandedY = 30, 30
	w.entities[1].X = 50
	w.entities[1].HasTarget, w.entities[1].TargetX, w.entities[1].TargetY = true, 60, 30
	groups := w.aiGroups()
	if len(groups) != 1 || len(groups[0].members) != 2 {
		t.Fatal("moving Roam member vanished from the distance population", groups)
	}
	rng := w.rng
	w.decide(groups[0])
	if w.groups[0].roamCounter != 51 || w.groups[0].commandedX != 30 || w.groups[0].commandedY != 30 || w.rng != rng {
		t.Fatal("far moving member did not prevent the proximity reroll")
	}
}
