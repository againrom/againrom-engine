package sim

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func widenedSavedGroupPin(old []byte) []byte {
	out := append(append([]byte(nil), old...), 0, 0, 0, 0)
	out[0] = 78
	return widenedSavedStructurePin(out)
}

func savedGroupWorld(t *testing.T) *World {
	t.Helper()
	w := mustWorld(t, 113, Bounds{Width: 32, Height: 32}, []Entity{
		{ID: 10, Owner: 1, Group: 19, X: 8, Y: 8, HP: 100, MaxHP: 100, Speed: 20},
		{ID: 20, Owner: 1, Group: 19, X: 9, Y: 8, HP: 100, MaxHP: 100, Speed: 20},
		{ID: 30, Owner: 2, Group: 20, X: 10, Y: 8, HP: 100, MaxHP: 100, Speed: 20},
	})
	g := SavedGroup{ID: 71, Selector: 19, Words: []uint16{97, 99}, Path: []uint16{0x1111, 0x1212}}
	g.Members = []SavedGroupMember{{2, 20, true}, {1, 10, true}, {3, 30, true}}
	g.AI[0x20], g.AI[0x45], g.AI[0x38], g.AI[0x44] = 4, 1, 9, 7
	g.AI[10], g.AI[11] = 25, 26 // explicitly NOT actor Move destination
	orders := []SavedActorOrder{{Entity: 10}, {Entity: 20}, {Entity: 30}}
	for i := range orders {
		orders[i].Raw[10], orders[i].Raw[11] = 15+uint8(i), 14
	}
	empty := SavedGroup{ID: 72, Selector: 88}
	empty.AI[0x20], empty.AI[0x45] = 3, 1
	if err := w.ImportSavedGroups([]SavedGroup{g, empty}, orders); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestSavedGroupsMoveOrderAndEmptyDispatch(t *testing.T) {
	w := savedGroupWorld(t)
	w.engagementPass()
	for i, e := range w.entities {
		if !e.HasTarget || e.TargetX != 15+int32(i) || e.TargetY != 14 {
			t.Fatalf("actor destination replaced by Group cell: %+v", e)
		}
	}
	if w.savedGroups.Groups[1].AI[0x20] != 0xff {
		t.Fatal("empty Group not dispatched")
	}
	groups, orders, _ := w.SavedGroups()
	if groups[0].Members[0].Entity != 20 || groups[0].Owner.Class != 0 {
		t.Fatal("sorted membership or inferred owner")
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var fresh World
	if err := fresh.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	gotG, gotO, _ := fresh.SavedGroups()
	if !reflect.DeepEqual(groups, gotG) || !reflect.DeepEqual(orders, gotO) {
		t.Fatal("current registry lost in native SAVE")
	}
	Step(w, nil)
	Step(&fresh, nil)
	if w.Hash() != fresh.Hash() {
		t.Fatal("next step differs after fresh LOAD")
	}
}

func TestSavedGroupsSuccessorIsMutationSensitive(t *testing.T) {
	w := savedGroupWorld(t)
	g := &w.savedGroups.Groups[0]
	var visited []EntityID
	w.walkSavedMembers(g, func(i int) {
		id := w.entities[i].ID
		visited = append(visited, id)
		if id == 20 {
			w.detachSavedMember(id)
		}
	})
	if !reflect.DeepEqual(visited, []EntityID{20}) {
		t.Fatal(visited)
	}
	visited = nil
	w.walkSavedMembers(g, func(i int) { visited = append(visited, w.entities[i].ID) })
	if !reflect.DeepEqual(visited, []EntityID{10, 30}) {
		t.Fatal("fresh head lost", visited)
	}
	g.Members = g.Members[:1]
	visited = nil
	w.walkSavedMembers(g, func(i int) {
		visited = append(visited, w.entities[i].ID)
		if len(visited) == 1 {
			g.Members = append(g.Members, SavedGroupMember{3, 30, true})
		}
	})
	if !reflect.DeepEqual(visited, []EntityID{10, 30}) {
		t.Fatal("fresh appended successor lost", visited)
	}
}

func TestSavedGroupsPatrolCellCursorDuplicateAndLatch(t *testing.T) {
	w := savedGroupWorld(t)
	g := &w.savedGroups.Groups[0]
	g.AI[0x20] = 0
	o := w.savedOrder(10)
	o.State = 0xa
	o.Patrol = []uint16{0x0808, 0x0909, 0x0808, 0x0a0a}
	binary.LittleEndian.PutUint16(o.Raw[2:], 0x0808)
	binary.LittleEndian.PutUint32(o.Raw[4:], 0x2468)
	o.Raw[8] = 0xb
	w.savedActorDispatch(0)
	if binary.LittleEndian.Uint16(o.Raw[2:]) != 0x0909 || binary.LittleEndian.Uint32(o.Raw[4:]) != 1 || binary.LittleEndian.Uint16(o.Raw[:]) != 0x0808 {
		t.Fatalf("first equal/latch law lost: %x", o.Raw[:12])
	}
	w.entities[0].X, w.entities[0].Y = 9, 9
	w.savedActorDispatch(0)
	if binary.LittleEndian.Uint16(o.Raw[2:]) != 0x0808 {
		t.Fatal("ring successor lost")
	}
	w.entities[0].X, w.entities[0].Y = 8, 8
	w.savedActorDispatch(0)
	if binary.LittleEndian.Uint16(o.Raw[2:]) != 0x0909 {
		t.Fatal("duplicate treated as node cursor")
	}
	if !reflect.DeepEqual(g.Path, []uint16{0x1111, 0x1212}) {
		t.Fatal("actor path aliased Group path")
	}
}

func TestSavedGroupsPlayerCommandMovesCurrentMembership(t *testing.T) {
	w := savedGroupWorld(t)
	Step(w, []Command{{Kind: KindGroupMoveTo, Entity: 20, Group: 1, X: 20, Y: 21}, {Kind: KindGroupMoveTo, Entity: 10, Group: 1, X: 20, Y: 21}})
	if got := w.savedGroups.Groups[0].Members; len(got) != 1 || got[0].Entity != 30 {
		t.Fatal("old members retained", got)
	}
	newG := w.savedGroupFor(20)
	if newG == nil || newG.ID == 71 || newG.Members[0].Entity != 20 || newG.Members[1].Entity != 10 {
		t.Fatal("command group identity/order lost")
	}
	if o := w.savedOrder(20); o.Raw[10] == 0 || o.Raw[11] == 0 {
		t.Fatal("current destination not mutated")
	}
	w.handOver(0, 4)
	if w.savedGroupFor(10).Owner.Owner != 4 || len(newG.Members) != 1 {
		t.Fatal("handover missed current Group")
	}
}

func TestSavedGroupsAmbiguousAndUnboundRefuseAtomically(t *testing.T) {
	w := savedGroupWorld(t)
	w.savedGroups.Groups[1].Selector = 19
	before := w.Hash()
	w.cmdSavedGroupOrder(ScriptInstant{HasGroup: true, Group: 19, Args: [10]int32{4, 1, 1}})
	if before != w.Hash() || len(w.SavedGroupIssues()) == 0 {
		t.Fatal("ambiguous selector silently applied")
	}
	w.savedGroups.Groups[1].Selector = 88
	w.savedGroups.Groups[0].Members = append(w.savedGroups.Groups[0].Members, SavedGroupMember{Archive: 90})
	before = w.Hash()
	w.savedGroupPass(nil)
	// The separately empty Group may set ff; the affected Group is unchanged.
	if w.savedGroups.Groups[0].AI[0x20] != 4 || w.entities[0].HasTarget {
		t.Fatal("partial dispatch before missing member")
	}
	_ = before
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var fresh World
	if err := fresh.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if len(fresh.savedGroups.Groups[0].Members) != 4 {
		t.Fatal("unbound archive member dropped by SAVE")
	}
}

func TestSavedScriptSelectorResolvesToLaterOwner(t *testing.T) {
	w := savedGroupWorld(t)
	g, h := &w.savedGroups.Groups[0], &w.savedGroups.Groups[1]
	h.Selector, h.Members = g.Selector, append([]SavedGroupMember(nil), g.Members[2:]...)
	g.Members = g.Members[:2]
	g.Owner, h.Owner = SavedGroupReference{Class: 1, Owner: 1}, SavedGroupReference{Class: 1, Owner: 2}
	w.entities[2].Group = g.Selector
	w.script = mustScript(t, []ScriptCheck{groupCheck(0, 19), constCheck(1, 1)},
		[]ScriptInstant{{Op: ScriptInstantGroupOrder, HasGroup: true, Group: 19, Args: [10]int32{4, 11, 16}}},
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)}, Instants: acts(0), Once: true}})
	w.setRegisterAt(1, 1)
	w.scriptPass(nil)
	if issues := w.SavedGroupIssues(); len(issues) != 0 {
		t.Fatal("valid owned selectors reported as unavailable", issues)
	}
	if w.ScriptRegister(0) != 1 || !w.ScriptLatched(0) || !reflect.DeepEqual(w.groupMembers(19), []int{2}) {
		t.Fatalf("count, trigger or member selection did not resolve to the later owner: %d/%t/%v", w.ScriptRegister(0), w.ScriptLatched(0), w.groupMembers(19))
	}
	if g.AI[10] != 25 || g.AI[11] != 26 || h.AI[0x20] != 4 || h.AI[10] != 11 || h.AI[11] != 16 {
		t.Fatal("command did not reach the later owner's group alone", *g, *h)
	}
	for cut := range 2 {
		data, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var cold World
		if err := cold.UnmarshalBinary(data); err != nil || cold.Hash() != w.Hash() {
			t.Fatalf("cut %d: %v", cut, err)
		}
		for _, current := range []*World{w, &cold} {
			current.cmdSavedGroupOrder(ScriptInstant{HasGroup: true, Group: 19, HasUnit: true, Unit: 10,
				Args: [10]int32{subCommandFollow, 5}})
			if current.entities[2].EscortTarget != 10 || current.entities[0].EscortTarget != 0 || current.entities[1].EscortTarget != 0 {
				t.Fatal("per-member follow reached the earlier owner")
			}
			before := current.savedGroups.Groups[0].AI[0x20]
			current.cmdGroupRoam(19)
			first := current.entities[2]
			later := current.savedGroups.Groups[1]
			if later.AI[0x20] != orderRoam || later.AI[10] != byte(first.X) || later.AI[11] != byte(first.Y) || current.savedGroups.Groups[0].AI[0x20] != before {
				t.Fatal("roam did not retain the later owner's own first cell alone")
			}
		}
		if w.Hash() != cold.Hash() {
			t.Fatal("cold selector dispatch changed")
		}
		w = &cold
	}
	w.savedGroups.Groups[1].Owner = w.savedGroups.Groups[0].Owner
	before := w.Hash()
	w.cmdSavedGroupOrder(ScriptInstant{HasGroup: true, Group: 19, Args: [10]int32{4, 1, 1}})
	if before != w.Hash() {
		t.Fatal("same-owner selector ambiguity applied partially")
	}
}

func TestSavedGroupsRoamUsesItsOwnCellCounterAndNativeDraws(t *testing.T) {
	for _, counter := range []byte{0, 50, 51, 255} {
		w := mustWorld(t, 113, Bounds{128, 128}, []Entity{{ID: 10, Owner: 1, X: 64, Y: 64, HP: 10, MaxHP: 10, Speed: 10}})
		g := SavedGroup{ID: 1, Members: []SavedGroupMember{{1, 10, true}}}
		g.AI[0x20], g.AI[0x45], g.AI[0x15], g.AI[10], g.AI[11] = 0x11, 1, counter, 64, 64
		o := SavedActorOrder{Entity: 10}
		o.Raw[10], o.Raw[11] = 70, 71
		if err := w.ImportSavedGroups([]SavedGroup{g}, []SavedActorOrder{o}); err != nil {
			t.Fatal(err)
		}
		before := w.rng
		draw := before.uniform(7)
		// Independent compass table. Every ray from 64,64 is admissible.
		dx, dy := [8]int32{0, 1, 1, 1, 0, -1, -1, -1}, [8]int32{-1, -1, 0, 1, 1, 1, 0, -1}
		w.savedGroupPass(nil)
		got := w.savedGroups.Groups[0]
		if w.rng != before || got.AI[10] != byte(64+20*dx[draw]) || got.AI[11] != byte(64+20*dy[draw]) || got.AI[0x15] != 1 {
			t.Fatalf("Roam draw/counter %d: %+v", counter, got.AI)
		}
		if e := w.entities[0]; !e.HasTarget || e.TargetX != 70 || e.TargetY != 71 {
			t.Fatal("Roam copied Group cell into actor destination", e)
		}
	}
}

func TestSavedGroupsRoamRetentionThresholdAndImpossibleRectangle(t *testing.T) {
	w := savedGroupWorld(t)
	g := &w.savedGroups.Groups[0]
	g.AI[0x20], g.AI[0x15], g.AI[10], g.AI[11] = 0x11, 50, 25, 26
	rng := w.rng
	w.savedGroupPass(nil)
	if g.AI[0x15] != 51 || g.AI[10] != 25 || w.rng != rng {
		t.Fatal("Roam rerolled at 50 despite distant members")
	}
	before := w.Hash()
	w.savedGroupPass(nil)
	if w.Hash() != before || len(w.SavedGroupIssues()) == 0 {
		t.Fatal("impossible Roam failed atomically")
	}
}

func TestSavedGroupsPrimaryMissingOrderAtomicAndIdentityCommandIndependent(t *testing.T) {
	w := savedGroupWorld(t)
	w.savedGroups.Orders = w.savedGroups.Orders[:2] // third is missing, after two valid members
	w.savedGroupPass(nil)
	for _, e := range w.entities {
		if e.HasTarget {
			t.Fatal("partial move before missing operand", e)
		}
	}
	if len(w.SavedGroupIssues()) == 0 {
		t.Fatal("missing order not diagnosed")
	}
	w.savedGroups.Groups[1].Selector = 19
	Step(w, []Command{{Kind: KindGroupMoveTo, Entity: 10, Group: 1, X: 17, Y: 18}})
	if g := w.savedGroupFor(10); g.ID == 71 || len(g.Members) != 1 {
		t.Fatal("identity command blocked by unrelated selector collision")
	}
}

func TestSavedGroupsGuardAndDeathMutateCurrentState(t *testing.T) {
	w := savedGroupWorld(t)
	g := &w.savedGroups.Groups[0]
	g.AI[0x20] = 0
	o := w.savedOrder(10)
	o.State = 0xb
	o.Raw[0], o.Raw[1] = 7, 7
	w.savedActorDispatch(0)
	if e := w.entities[0]; !e.HasTarget || e.TargetX != 7 || e.TargetY != 7 || o.Raw[8] != 1 {
		t.Fatal("saved guard post not used", e)
	}
	w.clearFelled(0)
	if len(g.Members) != 3 {
		t.Fatal("living health refresh detached membership")
	}
	w.entities[0].HP = -10
	w.entities[0].DyingTime = 2
	w.clearFelled(0)
	if w.entities[0].Decay != DecayFallen || len(g.Members) != 3 {
		t.Fatal("fall detached the member before teardown")
	}
	w.decayPass()
	if w.entities[0].Decay != DecayFallen || len(g.Members) != 3 {
		t.Fatal("dying dwell detached the member before teardown")
	}
	w.decayPass()
	if w.entities[0].Decay < DecayBones || len(g.Members) != 2 || g.AI[0x44] != 7 {
		t.Fatal("teardown lost membership/rate law")
	}
}

func TestSavedGroupsUnboundCountNeverWritesOrFiresFalseTrigger(t *testing.T) {
	w := savedGroupWorld(t)
	w.script = mustScript(t, []ScriptCheck{{Op: ScriptCheckGroupCount, HasGroup: true, Group: 19, Register: 0}, constCheck(1, 8)},
		[]ScriptInstant{{Op: ScriptInstantWin}}, []ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)}, Instants: acts(0), Once: true}})
	w.setRegisterAt(0, 8) // stale true must not fire; neither zero nor stale value is a result
	w.savedGroups.Groups[0].Members = append(w.savedGroups.Groups[0].Members, SavedGroupMember{Archive: 90})
	w.scriptPass(nil)
	if w.ScriptRegister(0) != 8 || w.won != 0 {
		t.Fatal("unsupported count fabricated value or false action")
	}
	w.savedGroups.Groups[0].Members = w.savedGroups.Groups[0].Members[:3]
	w.scriptPass(nil)
	if w.ScriptRegister(0) != 3 {
		t.Fatal("repaired count did not execute")
	}
}

func TestSavedGroupsFreshSuccessorSurvivesRegistryReallocation(t *testing.T) {
	w := savedGroupWorld(t)
	var visited []EntityID
	w.walkSavedMembers(&w.savedGroups.Groups[0], func(i int) {
		visited = append(visited, w.entities[i].ID)
		if len(visited) == 1 {
			w.commandSavedGroup([]int{i}, orderGuard, cell{})
		}
	})
	if !reflect.DeepEqual(visited, []EntityID{20}) {
		t.Fatal("stale Group pointer traversed removed current", visited)
	}
}

func TestSavedGroupsMalformedNativePayloadIsAtomic(t *testing.T) {
	w := savedGroupWorld(t)
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	groupEnd := len(form) - entityIDFloorLen - spellDeliverySpanLen - 56 // absent Structure, Player, Stride, motion, plane, object, carried-resume, action-clock and Group-counter footers
	start := groupEnd - 4 - int(binary.LittleEndian.Uint32(form[groupEnd-4:]))
	for _, kind := range []string{"count", "binding", "identity", "span"} {
		bad := append([]byte(nil), form...)
		switch kind {
		case "count":
			binary.LittleEndian.PutUint32(bad[start:], ^uint32(0))
		case "binding":
			bad[start+4+119+4+4+6] = 2 // first member Bound, after two 2-word lists
		case "identity":
			binary.LittleEndian.PutUint32(bad[start+4:], 0)
		case "span":
			binary.LittleEndian.PutUint32(bad[groupEnd-4:], ^uint32(0))
		}
		before := w.Hash()
		if err := w.UnmarshalBinary(bad); err == nil || w.Hash() != before {
			t.Fatalf("%s partially published: %v", kind, err)
		}
	}
}

func TestSavedGroupsRateReadsLiveGroupNotStaleEntityCache(t *testing.T) {
	w := savedGroupWorld(t)
	w.entities[0].Speed, w.entities[0].Load, w.entities[0].Capacity = 0, 100, 1
	w.entities[0].GroupSpeed = 29
	if r, _, _, ok := w.StepRate(10, 9, 8); !ok || r != 7 {
		t.Fatalf("saved Group override %d/%t", r, ok)
	}
	w.savedGroups.Groups[0].AI[0x44] = 11
	if r, _, _, ok := w.StepRate(10, 9, 8); !ok || r != 11 {
		t.Fatalf("current Group rate ignored %d/%t", r, ok)
	}
	w.savedGroups.Groups[0].AI[0x44] = 0
	if _, _, _, ok := w.StepRate(10, 9, 8); ok {
		t.Fatal("stale Entity.GroupSpeed resurrected after saved rate became zero")
	}
}

func TestSavedGroupsNativeEscortCommandsDispatchAndSurviveSave(t *testing.T) {
	for _, command := range []int32{subCommandDefend, subCommandFollow} {
		w := savedGroupWorld(t)
		// This mixed-owner synthetic Group must remain active while its
		// foreign escort target occupies an excluded coarse-grid corner.
		binary.LittleEndian.PutUint32(w.savedGroups.Groups[0].AI[0x48:], 1)
		w.entities[2].X, w.entities[2].Y = 24, 24
		o := w.savedOrder(10)
		binary.LittleEndian.PutUint32(o.Raw[0x10:], 0xdecafbad) // not a native ID
		w.cmdSavedGroupOrder(ScriptInstant{HasGroup: true, Group: 19, HasUnit: true, Unit: 30, Args: [10]int32{command, 1}})
		o = w.savedOrder(10)
		if !o.Authored || binary.LittleEndian.Uint32(o.Raw[0x10:]) != 0xdecafbad || !w.entities[0].HasEscortTarget || w.entities[0].EscortTarget != 30 {
			t.Fatal("native binding conflated with raw target", o, w.entities[0])
		}
		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var fresh World
		if err := fresh.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
		if !fresh.savedOrder(10).Authored {
			t.Fatal("authored order flag lost")
		}
		for range 20 {
			Step(w, nil)
			Step(&fresh, nil)
			if w.Hash() != fresh.Hash() {
				t.Fatal("escort next step changed across SAVE")
			}
		}
		if e := w.entities[0]; e.X == 8 && e.Y == 8 {
			t.Fatal("native escort did not dispatch", command, e)
		}
	}
}

func TestSavedGroupsImportedEscortCannotBorrowCoincidentNativeID(t *testing.T) {
	w := savedGroupWorld(t)
	w.savedGroups.Groups[0].AI[0x20] = 0
	o := w.savedOrder(10)
	o.State = 8
	binary.LittleEndian.PutUint32(o.Raw[0x10:], 30)
	before := w.entities[0]
	w.savedGroupPass(nil)
	if w.entities[0] != before || o.Authored || len(w.SavedGroupIssues()) == 0 {
		t.Fatal("raw target became native binding")
	}
	// A later real bound command repairs only its own operation.
	w.cmdSavedGroupOrder(ScriptInstant{HasGroup: true, Group: 19, HasUnit: true, Unit: 30, Args: [10]int32{subCommandFollow, 1}})
	if !w.savedOrder(10).Authored || w.entities[0].EscortTarget != 30 {
		t.Fatal("valid native command remained blocked")
	}
}

func TestSavedGroupsRoamRejectedDirectionsConsumeDraws(t *testing.T) {
	w := mustWorld(t, 3, Bounds{48, 48}, []Entity{{ID: 10, Owner: 1, X: 8, Y: 8, HP: 10, MaxHP: 10, Speed: 10}})
	g := SavedGroup{ID: 1, Members: []SavedGroupMember{{1, 10, true}}}
	g.AI[0x20], g.AI[0x45], g.AI[10], g.AI[11] = 0x11, 1, 8, 8
	o := SavedActorOrder{Entity: 10}
	o.Raw[10], o.Raw[11] = 20, 20
	if err := w.ImportSavedGroups([]SavedGroup{g}, []SavedActorOrder{o}); err != nil {
		t.Fatal(err)
	}
	w.savedGroupPass(nil)
	// Independent SplitMix64 arithmetic for seed3 gives directions0,5,4:
	// reject north and southwest, accept south. No original CRT claim.
	if w.rng.state != 0xdaa66d2c7ddf7442 || w.savedGroups.Groups[0].AI[10] != 8 || w.savedGroups.Groups[0].AI[11] != 28 || w.savedGroups.Groups[0].AI[0x15] != 1 {
		t.Fatalf("rejection draw protocol: %x %+v", w.rng.state, w.savedGroups.Groups[0].AI)
	}
}

func savedIncomingEscortWorld(t *testing.T) *World {
	w := savedGroupWorld(t)
	// Keep this mixed-owner escort fixture independent of spatial activation.
	binary.LittleEndian.PutUint32(w.savedGroups.Groups[0].AI[0x48:], 1)
	for i := range w.entities {
		e := &w.entities[i]
		e.SourceBinding = SourceBinding{Class: 1, ArchiveIndex: uint16(i + 1), Identity: 1000 + uint32(i)}
		e.ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 1}}
	}
	w.entities[2].X, w.entities[2].Y = 24, 24
	w.savedGroups.Groups[0].AI[0x20] = 0
	w.savedGroups.Groups[1].AI[0x20] = 0xff
	o := w.savedOrder(10)
	o.State, o.EscortTarget, o.EscortBound = 8, 30, true
	binary.LittleEndian.PutUint32(o.Raw[0x10:], 1002)
	o.Raw[0x70] = 1
	return w
}

func TestSavedGroupsIncomingEscortAtomicAdmissionAndNativeValidation(t *testing.T) {
	for _, failure := range []string{"miss", "stage", "target-class", "target-key"} {
		w := savedIncomingEscortWorld(t)
		// B precedes A in CURRENT membership. A valid earlier move must not
		// happen before discovering the later unsupported target operation.
		w.savedOrder(20).State = 1
		o := w.savedOrder(10)
		switch failure {
		case "miss":
			o.EscortBound, o.EscortTarget = false, 0
		case "stage":
			o.RepairStage, o.EscortBound, o.EscortTarget = 2, false, 0
		case "target-class":
			w.entities[2].SourceBinding.Class = 0
		case "target-key":
			w.entities[2].SourceBinding.Identity = 900
		}
		before := w.Hash()
		w.savedGroupPass(nil)
		if w.Hash() != before || len(w.SavedGroupIssues()) == 0 {
			t.Fatal("partial primary before late escort refusal", failure)
		}
	}
	w := savedIncomingEscortWorld(t)
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	groupEnd := len(form) - entityIDFloorLen - spellDeliverySpanLen - 56
	start := groupEnd - 4 - int(binary.LittleEndian.Uint32(form[groupEnd-4:]))
	// Literal layout: count4, Group119 + two2-word lists8 + members21,
	// emptyGroup119, orderCount4, Entity4/State4/Authored1/Stage1/Target4.
	bad := append([]byte(nil), form...)
	binary.LittleEndian.PutUint32(bad[start+4+119+8+21+119+4+10:], 20)
	before := w.Hash()
	if err := w.UnmarshalBinary(bad); err == nil || w.Hash() != before {
		t.Fatal("late invalid native escort partially adopted", err)
	}
	var fresh World
	if err := fresh.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	for range 20 {
		Step(w, nil)
		Step(&fresh, nil)
		if w.Hash() != fresh.Hash() {
			t.Fatal("incoming bound escort continuation changed")
		}
	}
	if w.entities[0].X == 8 && w.entities[0].Y == 8 {
		t.Fatal("incoming escort remained metadata")
	}
	w.remove([]EntityID{30})
	o := w.savedOrder(10)
	if o.EscortBound || o.EscortTarget != 0 || binary.LittleEndian.Uint32(o.Raw[0x10:]) != 1002 {
		t.Fatal("expired binding lost retained source key", o)
	}
	if _, err := w.MarshalBinary(); err != nil {
		t.Fatal("expired target made SAVE unavailable", err)
	}
}

func TestSavedGroupsIncomingEscortZeroRangeUsesRawBranchAndFallbackStop(t *testing.T) {
	w := savedIncomingEscortWorld(t)
	w.entities[2].X, w.entities[2].Y = 10, 8
	w.entities[0].ScanRange = 8
	o := w.savedOrder(10)
	o.Raw[0x70] = 0
	w.savedGroupPass(nil)
	if !w.entities[0].HasTarget || o.Raw[8] != 4 || o.Raw[0x14] != 8 || o.Raw[0x70] != 0 {
		t.Fatal("zero saved range coerced or compared against fallback", o)
	}
}

func TestSavedGroupsIncomingEscortAlreadyAdmittedDeadExactKey(t *testing.T) {
	w := savedIncomingEscortWorld(t)
	w.entities[2].SourceBinding = SourceBinding{}
	w.entities[2].ActorLoad = ActorLoad{}
	w.entities[2].MapUnitID = 50
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{deadInput(30, 3, -100)}); err != nil {
		t.Fatal(err)
	}
	o := w.savedOrder(10)
	binary.LittleEndian.PutUint32(o.Raw[0x10:], 130) // exact admitted corpse Token key
	if issue := w.savedPrimaryIssue(&w.savedGroups.Groups[0]); issue != "" {
		t.Fatal("target corpse stage incorrectly used as follower repair stage", issue)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var fresh World
	if err := fresh.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	for range 20 {
		Step(w, nil)
		Step(&fresh, nil)
		if w.Hash() != fresh.Hash() {
			t.Fatal("corpse escort changed across SAVE")
		}
	}
	if e := w.entities[0]; !e.HasEscortTarget || e.EscortTarget != 30 || e.X == 8 && e.Y == 8 {
		t.Fatal("exact corpse target did not drive movement", e)
	}
	// A second materialized actor claiming that source key is not first-wins.
	w.entities[1].SourceBinding.Identity = 130
	if savedEscortBindingValid(*w.savedOrder(10), w.entities, w.originalDead) {
		t.Fatal("ambiguous source key selected first actor")
	}
	w.entities[1].SourceBinding.Identity = 1001
	binary.LittleEndian.PutUint32(w.savedOrder(10).Raw[0x10:], 50) // MapID is not the key
	if savedEscortBindingValid(*w.savedOrder(10), w.entities, w.originalDead) {
		t.Fatal("corpse map identity substituted for source key")
	}
}
