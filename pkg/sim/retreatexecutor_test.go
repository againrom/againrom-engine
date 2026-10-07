package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func TestRetreatExecutorSeparatesCompletionClearAndEnteredZero(t *testing.T) {
	w := selectionWorld(t, retreatIntruder(2, 21, 20))
	w.orderAttack(0, 2)
	e := &w.entities[0]
	e.ActorState = actorStateRetreat
	e.AttackPhase, e.AttackCountdown = AttackCharging, 1
	e.Retreat = RetreatContinuation{Known: true, Pending: true, X: 17, Y: 20, Progress: 1, Counter: 2, Complete: true}
	if !w.stepRetreatExecutor(0) || e.Retreat.Progress != 0 || e.HasTarget || !e.Retreat.Pending || e.AttackPhase != AttackCharging {
		t.Fatal("completion-clear entered pending dispatch or reset phase", *e)
	}
	if w.stepRetreatExecutor(0) || !e.HasTarget || e.TargetX != 17 || e.Retreat.Pending {
		t.Fatal("entered-zero retained phase withheld pending move", *e)
	}
}

func TestRetreatExecutorActivityGateAppliesOnlyToEnteredZero(t *testing.T) {
	w := selectionWorld(t)
	w.entities[0].Owner = 2
	g := SavedGroup{ID: 1, Selector: 1, Owner: SavedGroupReference{Class: 1, Owner: 2}, Members: []SavedGroupMember{{Entity: 1, Bound: true}}}
	if err := w.ImportSavedGroups([]SavedGroup{g}, nil); err != nil {
		t.Fatal(err)
	}
	e := &w.entities[0]
	e.ActorState = actorStateRetreat
	e.Retreat = RetreatContinuation{Known: true, Pending: true, X: 17, Y: 20, Progress: 1, Counter: 2, Complete: true}
	w.stepRetreatExecutor(0)
	if e.Retreat.Progress != 0 {
		t.Fatal("inactive group blocked existing progress completion")
	}
	w.stepRetreatExecutor(0)
	if e.HasTarget || !e.Retreat.Pending {
		t.Fatal("inactive entered-zero executor dispatched pending")
	}
	w.savedGroups.Groups[0].AI[0x45] = 1
	w.stepRetreatExecutor(0)
	if !e.HasTarget || e.Retreat.Pending {
		t.Fatal("active group did not admit entered-zero executor")
	}
}

func TestRetreatActiveFailureTailReinstallsProgressWithoutResettingPhase(t *testing.T) {
	for source := range 4 {
		failed := source == 1 || source == 2
		w := selectionWorld(t, retreatIntruder(2, 20, 18), retreatIntruder(3, 21, 20))
		w.orderAttack(0, 2)
		e := &w.entities[0]
		e.ActorState = actorStateRetreat
		e.AttackPhase, e.AttackCountdown = AttackCharging, 1
		e.Retreat = RetreatContinuation{Known: true, Pending: true, X: 17, Y: 20, Progress: 1, Counter: 2, Complete: true, Failure: source == 1}
		if source >= 2 {
			motion := SavedActorMotion{Entity: e.ID, Current: source == 2}
			motion.Mover[0x98] = 1
			w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{motion}}
		}
		w.stepRetreatExecutor(0)
		if e.AttackPhase != AttackCharging || e.AttackCountdown != 1 {
			t.Fatal("tail reset retained phase")
		}
		if failed {
			if e.Retreat.Progress != 1 || e.Retreat.Counter != 0 || !e.HasAttackTarget || e.AttackTarget != 3 || e.Retreat.Pending || e.Retreat.Failure {
				t.Fatal("active tail did not replace victim and progress", *e)
			}
		} else if e.Retreat.Progress != 0 || !e.HasAttackTarget || e.AttackTarget != 2 || !e.Retreat.Pending {
			t.Fatal("quiet tail reinstalled progress", *e)
		}
		if source == 2 && w.motionFor(e.ID).Mover[0x98] != 0 || source == 3 && w.motionFor(e.ID).Mover[0x98] == 0 {
			t.Fatal("active/stale mover failure consumption")
		}
	}
}

func TestTacticalContinuationBytesAndActionLossControls(t *testing.T) {
	w := selectionWorld(t, retreatIntruder(2, 20, 18), retreatIntruder(3, 20, 22))
	w.takeOffMap(1)
	if !w.returnToMap(1) {
		t.Fatal("return refused")
	}
	w.entities[0].ActorState = actorStateRetreat
	w.entities[0].Retreat = RetreatContinuation{Known: true, Pending: true, X: 17, Y: 20, Progress: 1, Counter: 2, Complete: true}
	form := mustMarshal(t, w)
	if form[0] != tacticalFormVersion {
		t.Fatal("non-default tactical state absent from bytes")
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if cold.Hash() != w.Hash() || !reflect.DeepEqual(cold.ActorTraversal(), []EntityID{1, 3, 2}) {
		t.Fatal("bytes lost tactical state")
	}
	span := int(binary.LittleEndian.Uint32(form[len(form)-9:]))
	stripped := bytes.Clone(form[:len(form)-9-span])
	stripped[0] = form[len(form)-5]
	var lost World
	if err := lost.UnmarshalBinary(stripped); err != nil {
		t.Fatal(err)
	}
	if lost.Hash() == w.Hash() {
		t.Fatal("suffix removal did not change hash")
	}
	actions := actionCopy(t, w.Actions())
	if err := lost.RestoreActions(actions, nil); err != nil {
		t.Fatal(err)
	}
	if lost.Hash() != w.Hash() {
		t.Fatal("action supplement lost tactical state")
	}
	actions.ActorTraversal, actions.Actors[0].Retreat = nil, nil
	if err := lost.RestoreActions(actions, nil); err != nil {
		t.Fatal(err)
	}
	if lost.Hash() == w.Hash() {
		t.Fatal("action loss was invisible")
	}
	w.reacquireWithinReach(0)
	lost.reacquireWithinReach(0)
	if w.entities[0].AttackTarget != 2 || lost.entities[0].AttackTarget != 3 {
		t.Fatal("lost traversal did not change next production selection")
	}
}

func TestLoadedActorTraversalUsesPlayerGroupMemberOrder(t *testing.T) {
	w := selectionWorld(t, retreatIntruder(2, 20, 18), retreatIntruder(3, 20, 22))
	groups := []SavedGroup{
		{ID: 1, Selector: 1, Owner: SavedGroupReference{Class: 1, Owner: SelfSlot}, Members: []SavedGroupMember{{Entity: 1, Bound: true}}},
		{ID: 2, Selector: 2, Owner: SavedGroupReference{Class: 1, Owner: 3}, Members: []SavedGroupMember{{Entity: 3, Bound: true}, {Entity: 2, Bound: true}}},
	}
	if err := w.ImportSavedGroups(groups, nil); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportSavedGroupPlayers([]SavedGroupPlayer{{ID: 7, Slot: SelfSlot}, {ID: 9, Slot: 3}}, []SavedGroupContainer{{GroupID: 1, PlayerID: 7}, {GroupID: 2, PlayerID: 9}}); err != nil {
		t.Fatal(err)
	}
	w.RebuildLoadedActorTraversal()
	if !reflect.DeepEqual(w.ActorTraversal(), []EntityID{1, 3, 2}) {
		t.Fatal("LOAD reconstructed identity order", w.ActorTraversal())
	}
}

func TestActorTraversalIdentityPermutationPreservesSelectionAndBytes(t *testing.T) {
	w := selectionWorld(t, retreatIntruder(2, 20, 18), retreatIntruder(3, 20, 22))
	if err := w.RestoreActorTraversal([]EntityID{1, 3, 2}); err != nil {
		t.Fatal(err)
	}
	before := mustMarshal(t, w)
	if err := w.RestoreActorIdentities(map[EntityID]EntityID{1: 8, 2: 3, 3: 1}, nil); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.ActorTraversal(), []EntityID{8, 1, 3}) || w.entities[w.reacquisitionVictim(indexOfEntity(w.entities, 8))].ID != 3 {
		t.Fatal("identity permutation changed traversal or next selection", w.ActorTraversal())
	}
	if err := w.RestoreActorIdentities(map[EntityID]EntityID{8: 1, 3: 2, 1: 3}, nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("inverse permutation changed explicit traversal bytes")
	}
}

func TestActorTraversalPresenceEditsPreserveSurvivorsAndRejectBadIdentities(t *testing.T) {
	w := selectionWorld(t, retreatIntruder(2, 20, 18), retreatIntruder(3, 20, 22))
	if err := w.RestoreActorTraversal([]EntityID{1, 3, 2}); err != nil {
		t.Fatal(err)
	}
	actors := w.ActorTraversal()
	actors[0] = 99
	if w.ActorTraversal()[0] != 1 {
		t.Fatal("traversal reader exposed live storage")
	}
	actions := actionCopy(t, w.Actions())
	actions.Actors[1].OffMap = true
	if err := w.RestoreActions(actions, nil); err != nil || !reflect.DeepEqual(w.ActorTraversal(), []EntityID{1, 3}) {
		t.Fatal("presence edit lost survivors", err, w.ActorTraversal())
	}
	actions = actionCopy(t, w.Actions())
	actions.Actors[1].OffMap = false
	if err := w.RestoreActions(actions, nil); err != nil || !reflect.DeepEqual(w.ActorTraversal(), []EntityID{1, 3, 2}) || w.reacquisitionVictim(0) != 1 {
		t.Fatal("return edit failed tail append or next selection", err, w.ActorTraversal())
	}
	before := w.Hash()
	for _, bad := range [][]EntityID{{1, 3, 3}, {1, 3, 99}} {
		actions = actionCopy(t, w.Actions())
		actions.ActorTraversal = &bad
		if err := w.RestoreActions(actions, nil); err == nil || w.Hash() != before {
			t.Fatal("invalid traversal identity was admitted or mutated World", bad, err)
		}
	}
	if err := w.RestoreActorTraversal([]EntityID{1, 3}); err == nil || w.Hash() != before {
		t.Fatal("native traversal accepted an incomplete list")
	}
}

func TestActorTraversalLivingAdmissionKeepsUnchangedBytesAndAppendsNewActors(t *testing.T) {
	w := selectionWorld(t, retreatIntruder(2, 20, 18), retreatIntruder(3, 20, 22))
	if err := w.RestoreActorTraversal([]EntityID{1, 3, 2}); err != nil {
		t.Fatal(err)
	}
	e := w.entities[1]
	e.SourceBinding = SourceBinding{Class: 1, ArchiveIndex: 2, Identity: 0x01000002}
	e.ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 1}}
	e.TokenSize = 1
	w.entities[1] = e
	before := mustMarshal(t, w)
	if err := w.ImportOriginalLivingActors([]OriginalLivingActor{{Entity: e}}); err != nil || !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("unchanged living admission changed bytes", err)
	}
	e.OffMap = true
	if err := w.ImportOriginalLivingActors([]OriginalLivingActor{{Entity: e}}); err != nil || !reflect.DeepEqual(w.ActorTraversal(), []EntityID{1, 3}) {
		t.Fatal("living admission did not unlink off-map actor", err, w.ActorTraversal())
	}
	e.OffMap = false
	if err := w.ImportOriginalLivingActors([]OriginalLivingActor{{Entity: e}}); err != nil || !reflect.DeepEqual(w.ActorTraversal(), []EntityID{1, 3, 2}) {
		t.Fatal("living admission did not append returned actor", err, w.ActorTraversal())
	}
	e.ID, e.Y = 4, 23
	e.SourceBinding.ArchiveIndex, e.SourceBinding.Identity = 4, 0x01000004
	if err := w.ImportOriginalLivingActors([]OriginalLivingActor{{Entity: e, New: true}}); err != nil || !reflect.DeepEqual(w.ActorTraversal(), []EntityID{1, 3, 2, 4}) {
		t.Fatal("living admission did not append new actor", err, w.ActorTraversal())
	}
}

func inactiveRetreatFailureWorld(t *testing.T, mover bool) *World {
	t.Helper()
	w := selectionWorld(t, retreatIntruder(2, 20, 18), retreatIntruder(3, 21, 20))
	w.entities[0].Owner = 2
	w.relations.Set(2, 3, 1)
	group := SavedGroup{ID: 1, Selector: 1, Owner: SavedGroupReference{Class: 1, Owner: 2}, Members: []SavedGroupMember{{Entity: 1, Bound: true}}}
	if err := w.ImportSavedGroups([]SavedGroup{group}, nil); err != nil {
		t.Fatal(err)
	}
	e := &w.entities[0]
	e.ActorState = actorStateRetreat
	e.Retreat = RetreatContinuation{Known: true, Pending: true, X: 17, Y: 20, Failure: !mover}
	if mover {
		motion := SavedActorMotion{Entity: e.ID, Current: true}
		motion.Mover[0x98] = 1
		w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{motion}}
	}
	if w.retreatExecutorActive(0) {
		t.Fatal("fixture executor must be inactive")
	}
	return w
}

func TestRetreatInactiveEnteredZeroDoesNotConsumeFailure(t *testing.T) {
	for _, mover := range []bool{false, true} {
		w := inactiveRetreatFailureWorld(t, mover)
		before := w.entities[0]
		if w.stepRetreatExecutor(0) || !reflect.DeepEqual(w.entities[0], before) {
			t.Fatal("inactive entered-zero changed pending/failure or physical state", mover, w.entities[0])
		}
		if mover && w.motionFor(before.ID).Mover[0x98] != 1 {
			t.Fatal("inactive entered-zero consumed current mover failure")
		}
	}
}

func TestRetreatFailureActivityBoundaryControls(t *testing.T) {
	for _, mover := range []bool{false, true} {
		for _, enteredNonzero := range []bool{false, true} {
			w := inactiveRetreatFailureWorld(t, mover)
			e := &w.entities[0]
			wantPhase, wantCountdown := AttackReady, int32(0)
			if enteredNonzero {
				w.orderAttack(0, 2)
				if mover {
					w.motionFor(e.ID).Current, w.motionFor(e.ID).Issue = true, ""
				}
				e.AttackPhase, e.AttackCountdown = AttackCharging, 1
				e.Retreat.Progress, e.Retreat.Counter, e.Retreat.Complete = 1, 2, true
				wantPhase, wantCountdown = AttackCharging, 1
			} else {
				w.savedGroups.Groups[0].AI[0x45] = 1
			}
			if w.stepRetreatExecutor(0) || !e.HasAttackTarget || e.AttackTarget != 3 || e.Retreat.Progress != 1 || e.Retreat.Counter != 0 || e.Retreat.Pending || e.Retreat.Failure || e.Retreat.Complete {
				t.Fatal("admitted failure tail lost replacement/progress", mover, enteredNonzero, *e)
			}
			if e.AttackPhase != wantPhase || e.AttackCountdown != wantCountdown {
				t.Fatal("failure tail reset the retained physical phase")
			}
			if mover && w.motionFor(e.ID).Mover[0x98] != 0 {
				t.Fatal("admitted failure tail did not consume mover failure")
			}
		}
	}
}

func TestRetreatCompletionClearRemainsSerializable(t *testing.T) {
	for _, phase := range []AttackPhase{AttackCharging, AttackBoundaryOne} {
		w := selectionWorld(t, retreatIntruder(2, 21, 20))
		w.orderAttack(0, 2)
		e := &w.entities[0]
		e.ActorState, e.AttackPhase = actorStateRetreat, phase
		if phase == AttackCharging {
			e.AttackCountdown = 1
		}
		e.Retreat = RetreatContinuation{Known: true, Pending: true, X: 17, Y: 20, Progress: 1, Counter: 2, Complete: true}
		countdown := e.AttackCountdown
		Step(w, nil)
		if !e.HasAttackTarget || e.AttackTarget != 2 || e.AttackPhase != phase || e.AttackCountdown != countdown || e.Retreat.Progress != 0 || !e.Retreat.Pending || e.HasTarget || w.entities[1].HP != 100 {
			t.Fatal("completion-clear dispatched or lost/advanced physical carrier", *e)
		}
		var cold World
		if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil || cold.Hash() != w.Hash() {
			t.Fatal("completion-clear is not a readable byte boundary", phase, err)
		}
		Step(w, nil)
		Step(&cold, nil)
		if cold.Hash() != w.Hash() || w.entities[0].Retreat.Pending || w.entities[0].AttackPhase != AttackReady || w.entities[1].HP != 100 {
			t.Fatal("cold completion-clear changed next pending action")
		}
	}
}
