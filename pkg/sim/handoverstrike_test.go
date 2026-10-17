package sim

import (
	"fmt"
	"testing"
)

// The join (AI-643) moves an actor into a one-member Group of the new Player
// under Stand Ground and stores no progress, phase or victim, and the order 3
// evaluation afterwards reads the new Player's control word (AI-633, AI-383).
// The old owner holds the opposite control word, so an evaluation that read
// the old Player fails the pending-order check.
func TestAHandOverKeepsTheLoadedStrikeAndTheEvaluationReadsTheNewOwner(t *testing.T) {
	for _, control := range []uint32{0, 1} {
		t.Run(fmt.Sprintf("control%d", control), func(t *testing.T) {
			const oldOwner, newOwner = 7, 5
			w := engWorld(t, engRel(t, [3]uint32{oldOwner, 3, 1}, [3]uint32{newOwner, 3, 1}),
				engFighter(1, oldOwner, 5, 10), engFighter(2, 3, 30, 30))
			g := SavedGroup{ID: 71, Owner: SavedGroupReference{Class: 1, Owner: oldOwner}, Members: []SavedGroupMember{{Entity: 1, Bound: true}}}
			g.AI[0x20], g.AI[0x45] = orderStandGround, 1
			o := SavedActorOrder{Entity: 1, State: uint32(actorStateGuard)}
			o.Raw[8] = 5
			if err := w.ImportSavedGroups([]SavedGroup{g}, []SavedActorOrder{o}); err != nil {
				t.Fatal(err)
			}
			players := []SavedGroupPlayer{{ID: 91, Slot: oldOwner}, {ID: 95, Slot: newOwner}}
			if err := w.ImportSavedGroupPlayers(players, []SavedGroupContainer{{GroupID: 71, PlayerID: 91}}); err != nil {
				t.Fatal(err)
			}
			if err := w.ImportSavedPlayerFormations([]SavedPlayerFormation{{PlayerID: 91, CommandID: oldOwner}, {PlayerID: 95, CommandID: newOwner}},
				[]SavedGroupOwner{{GroupID: 71, PlayerID: 91}}); err != nil {
				t.Fatal(err)
			}
			if err := w.RestorePlayerParticipants([]PlayerParticipant{{PlayerID: 91, Value: 1 - control}, {PlayerID: 95, Value: control}}, true); err != nil {
				t.Fatal(err)
			}
			e := &w.entities[0]
			e.ActorState = actorStateGuard
			e.AttackTarget, e.HasAttackTarget = 2, true
			e.AttackCharge = 4
			e.AttackPhase, e.AttackCountdown = AttackCharging, 3
			e.TargetX, e.TargetY, e.HasTarget = 30, 30, true
			e.PostX, e.PostY = 9, 9
			if err := attackFault(*e); err != nil {
				t.Fatal("loaded-strike fixture", err)
			}

			w.handOver(0, newOwner)

			got := entityAt(t, w, 1)
			if got.Owner != newOwner {
				t.Fatalf("owner %d after the hand-over, want %d", got.Owner, newOwner)
			}
			if !got.HasAttackTarget || got.AttackTarget != 2 || got.AttackPhase != AttackCharging || got.AttackCharge != 4 || got.AttackCountdown != 3 {
				t.Fatal("the hand-over reset the loaded strike", got)
			}
			if got.ActorState != actorStateAcquire || got.PostX != got.X || got.PostY != got.Y {
				t.Fatalf("state %#x post (%d,%d), want 0xc at the own cell (%d,%d)", got.ActorState, got.PostX, got.PostY, got.X, got.Y)
			}
			held := w.savedGroupFor(1)
			if held == nil || held.ID == 71 || held.AI[0x20] != orderStandGround || len(held.Members) != 1 {
				t.Fatalf("the actor's Group after the hand-over is %+v, want a new one-member Stand Ground Group", held)
			}
			if pending := w.savedOrder(1).Raw[8]; pending != 0 {
				t.Fatalf("pending order %#x after the hand-over, want 0", pending)
			}
			if !got.HasTarget || got.TargetX != got.X || got.TargetY != got.Y {
				t.Fatal("the hand-over did not hold the own cell behind the loaded strike", got)
			}

			w.savedDecision(held, orderStandGround, nil)
			got = entityAt(t, w, 1)
			wantOrder, wantPending := byte(0), uint8(PendingNone)
			if control != 0 {
				wantOrder, wantPending = 0xb, PendingRelease
			}
			if pending := w.savedOrder(1).Raw[8]; pending != wantOrder || got.PendingOrder.Kind != wantPending {
				t.Fatalf("new owner control %d gave order %#x/pending %d, want %#x/%d", control, pending, got.PendingOrder.Kind, wantOrder, wantPending)
			}
			if !got.HasAttackTarget || got.AttackTarget != 2 || got.AttackPhase != AttackCharging {
				t.Fatal("the evaluation reset the loaded strike", got)
			}
			if control == 0 && (!got.HasTarget || got.TargetX != got.X || got.TargetY != got.Y) || control != 0 && got.HasTarget {
				t.Fatal("the evaluation kept the wrong held destination", got)
			}
			controlGroupColdWorld(t, w)
		})
	}
}

// The native dialect's hand-over (a second-game World, no Group registry)
// stands the member in a new one-member Stand Ground Group with state 0xc and
// the post at its own cell, and keeps the loaded strike (AI-643, AI-351). Here
// standHandedOverMember's two stores are the only writers of state and post.
func TestANativeHandOverStandsAtTheOwnCellAndKeepsTheLoadedStrike(t *testing.T) {
	const oldOwner, newOwner = 7, 5
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{},
		[]Entity{engFighter(1, oldOwner, 5, 10), engFighter(2, 3, 30, 30)}, secondGameScript(t, nil, nil, nil),
		engRel(t, [3]uint32{oldOwner, 3, 1}, [3]uint32{newOwner, 3, 1}))
	if err != nil {
		t.Fatal(err)
	}
	if !w.SecondGame() || !w.groupDialectNative() {
		t.Fatal("fixture: not a native-dialect World")
	}
	e := &w.entities[0]
	e.ActorState = actorStateGuard
	e.AttackTarget, e.HasAttackTarget = 2, true
	e.AttackCharge = 4
	e.AttackPhase, e.AttackCountdown = AttackCharging, 3
	e.TargetX, e.TargetY, e.HasTarget = 30, 30, true
	e.PostX, e.PostY = 9, 9
	if err := attackFault(*e); err != nil {
		t.Fatal("loaded-strike fixture", err)
	}

	w.handOver(0, newOwner)

	got := entityAt(t, w, 1)
	if got.Owner != newOwner {
		t.Fatalf("owner %d after the hand-over, want %d", got.Owner, newOwner)
	}
	if !got.HasAttackTarget || got.AttackTarget != 2 || got.AttackPhase != AttackCharging || got.AttackCharge != 4 || got.AttackCountdown != 3 {
		t.Fatal("the hand-over reset the loaded strike", got)
	}
	if got.ActorState != actorStateAcquire || got.PostX != got.X || got.PostY != got.Y {
		t.Fatalf("state %#x post (%d,%d), want 0xc at the own cell (%d,%d)", got.ActorState, got.PostX, got.PostY, got.X, got.Y)
	}
	if order, _, ok := w.groupState(newOwner, got.Group); !ok || got.Group == 0 || order != orderStandGround {
		t.Fatalf("Group %d of the new owner: order %d, recorded %v; want a Stand Ground Group", got.Group, order, ok)
	}
}
