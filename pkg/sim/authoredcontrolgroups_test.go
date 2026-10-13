package sim

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func controlGroupColdWorld(t *testing.T, w *World) *World {
	t.Helper()
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() {
		t.Fatal("control group changed on cold restore", err)
	}
	return &cold
}

func TestAuthoredControlGroupActivityAndColdRestore(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		sourceOwner, sourceControl uint32
		memberOwner, memberControl uint32
		forced                     bool
		want                       byte
	}{
		{"human-other-covers-computer-self", 7, 0, SelfSlot, 1, false, 1},
		{"human-other-covers-whole-word-self", 7, 0, SelfSlot, 0x100, false, 1},
		{"computer-other-does-not-cover", 7, 1, 8, 1, false, 0},
		{"whole-word-other-does-not-cover", 7, 0x100, 8, 1, false, 0},
		{"computer-self-does-not-cover", SelfSlot, 1, 8, 1, false, 0},
		{"human-member-keeps-raw-count", 7, 0, 8, 0, false, 17},
		{"force-counts-computer-self", 7, 1, SelfSlot, 0x100, true, 1},
		{"force-does-not-count-human", 7, 1, 8, 0, true, 17},
		{"unowned-does-not-cover", 0, 0, 8, 1, false, 0},
		{"unowned-member-counts", 7, 0, 0, 0, false, 1},
		{"force-counts-unowned-member", 7, 1, 0, 0, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := mustWorld(t, 1, Bounds{Width: 128, Height: 128}, []Entity{
				{ID: 1, Owner: tc.sourceOwner, X: 8, Y: 8, HP: 20, MaxHP: 20},
				{ID: 2, Owner: tc.memberOwner, X: 16, Y: 8, HP: 20, MaxHP: 20},
			})
			g := SavedGroup{ID: 71, Members: []SavedGroupMember{{Entity: 2, Bound: true}}}
			g.AI[0x45] = 17
			if tc.forced {
				binary.LittleEndian.PutUint32(g.AI[0x48:], 0x100)
			}
			if err := w.ImportSavedGroups([]SavedGroup{g}, nil); err != nil {
				t.Fatal(err)
			}
			if err := w.RestoreCurrentPlayers([]SavedGroupPlayer{{ID: 91, Slot: tc.sourceOwner}, {ID: 103, Slot: tc.memberOwner}},
				[]PlayerParticipant{{PlayerID: 91, Value: tc.sourceControl}, {PlayerID: 103, Value: tc.memberControl}}, true); err != nil {
				t.Fatal(err)
			}
			cold := controlGroupColdWorld(t, w)
			for range 2 {
				w.refreshSavedGroupActivity()
				cold.refreshSavedGroupActivity()
				if got := w.savedGroups.Groups[0].AI[0x45]; got != tc.want {
					t.Fatalf("activity %d, want %d", got, tc.want)
				}
				if w.Hash() != cold.Hash() {
					t.Fatal("activity rebuild changed after cold restore")
				}
			}
		})
	}
}

func TestAuthoredControlCoverageWakesSavedGroup(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		sourceOwner, sourceControl uint32
		memberOwner                uint32
		awake                      bool
	}{
		{"human-other-wakes-computer-self", 7, 0, SelfSlot, true},
		{"human-other-wakes-computer-other", 7, 0, 8, true},
		{"computer-self-keeps-group-asleep", SelfSlot, 1, 8, false},
		{"whole-word-other-keeps-group-asleep", 7, 0x100, 8, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := engWorld(t, engRel(t, [3]uint32{tc.memberOwner, tc.sourceOwner, 1}),
				engFighter(1, tc.sourceOwner, 8, 8), engFighter(2, tc.memberOwner, 9, 8))
			g := SavedGroup{ID: 71, Members: []SavedGroupMember{{Entity: 2, Bound: true}}}
			g.AI[0x20] = orderStandGround
			if err := w.ImportSavedGroups([]SavedGroup{g}, []SavedActorOrder{{Entity: 2}}); err != nil {
				t.Fatal(err)
			}
			if err := w.RestoreCurrentPlayers([]SavedGroupPlayer{{ID: 91, Slot: tc.sourceOwner}, {ID: 103, Slot: tc.memberOwner}},
				[]PlayerParticipant{{PlayerID: 91, Value: tc.sourceControl}, {PlayerID: 103, Value: 1}}, true); err != nil {
				t.Fatal(err)
			}
			cold := controlGroupColdWorld(t, w)
			for tick := 0; tick <= scriptPassPhase; tick++ {
				Step(w, nil)
				Step(cold, nil)
				if w.Hash() != cold.Hash() {
					t.Fatalf("activity dispatch changed after cold restore at tick %d", tick)
				}
			}
			e := entityAt(t, w, 2)
			if e.HasAttackTarget != tc.awake || tc.awake && e.AttackTarget != 1 {
				t.Fatalf("activity gate left victim %v/%d, want acquired %v", e.HasAttackTarget, e.AttackTarget, tc.awake)
			}
			want := byte(0)
			if tc.awake {
				want = 5
			}
			if o := w.savedOrder(2); o == nil || o.Raw[8] != want {
				t.Fatalf("activity gate pending order %+v, want %d", o, want)
			}
		})
	}
}

func TestAuthoredControlSavedStandGroundUsesMemberAndColdRestore(t *testing.T) {
	for _, owner := range []uint32{7, SelfSlot, 0} {
		for _, control := range []uint32{0, 1, 0x100} {
			t.Run(fmt.Sprintf("owner%d-control%x", owner, control), func(t *testing.T) {
				human := owner != 0 && control == 0
				w := engWorld(t, engRel(t, [3]uint32{owner, 3, 1}),
					engFighter(1, owner, 5, 10), engFighter(2, 3, 30, 30))
				g := SavedGroup{ID: 71, Owner: SavedGroupReference{Class: 1, Owner: 9}, Members: []SavedGroupMember{{Entity: 1, Bound: true}}}
				g.AI[0x20], g.AI[0x45] = orderStandGround, 1
				o := SavedActorOrder{Entity: 1, State: uint32(actorStateGuard)}
				o.Raw[8] = 5
				if err := w.ImportSavedGroups([]SavedGroup{g}, []SavedActorOrder{o}); err != nil {
					t.Fatal(err)
				}
				players := []SavedGroupPlayer{{ID: 91, Slot: owner}, {ID: 103, Slot: 9}}
				if err := w.ImportSavedGroupPlayers(players, []SavedGroupContainer{{GroupID: 71, PlayerID: 103}}); err != nil {
					t.Fatal(err)
				}
				if err := w.ImportSavedPlayerFormations([]SavedPlayerFormation{{PlayerID: 91, CommandID: int16(owner)}, {PlayerID: 103, CommandID: 9}},
					[]SavedGroupOwner{{GroupID: 71, PlayerID: 103}}); err != nil {
					t.Fatal(err)
				}
				otherControl := uint32(0)
				if human {
					otherControl = 1
				}
				if err := w.RestorePlayerParticipants([]PlayerParticipant{{PlayerID: 91, Value: control}, {PlayerID: 103, Value: otherControl}}, true); err != nil {
					t.Fatal(err)
				}
				e := &w.entities[0]
				e.ActorState = actorStateGuard
				e.AttackTarget, e.HasAttackTarget = 2, true
				e.AttackCharge = 4
				e.AttackPhase, e.AttackCountdown = AttackCharging, 3
				e.TargetX, e.TargetY, e.HasTarget = 30, 30, true
				if err := attackFault(*e); err != nil {
					t.Fatal("loaded-strike fixture", err)
				}
				cold := controlGroupColdWorld(t, w)
				w.savedDecision(&w.savedGroups.Groups[0], orderStandGround, nil)
				cold.savedDecision(&cold.savedGroups.Groups[0], orderStandGround, nil)
				if w.Hash() != cold.Hash() {
					t.Fatal("member control changed after cold restore")
				}
				got := entityAt(t, w, 1)
				wantOrder, wantPending := byte(0), uint8(PendingNone)
				if !human {
					wantOrder, wantPending = 0xb, PendingRelease
				}
				if pending := w.savedOrder(1).Raw[8]; pending != wantOrder || got.PendingOrder.Kind != wantPending {
					t.Fatalf("member control %#x gave order %#x/pending %d, want %#x/%d", control, pending, got.PendingOrder.Kind, wantOrder, wantPending)
				}
				if !got.HasAttackTarget || got.AttackTarget != 2 || got.AttackPhase != AttackCharging || got.AttackCharge != 4 || got.AttackCountdown != 3 {
					t.Fatal("decision reset the loaded strike", got)
				}
				if human && (!got.HasTarget || got.TargetX != got.X || got.TargetY != got.Y) || !human && got.HasTarget {
					t.Fatal("decision kept the wrong pursuit destination", got)
				}
				controlGroupColdWorld(t, w)
			})
		}
	}
}
