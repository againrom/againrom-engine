package sim

import (
	"encoding/binary"
	"slices"
	"testing"
)

func TestCurrentOrderTargetProjectionKeepsInactiveOperand(t *testing.T) {
	actor := Entity{HasAttackTarget: true}
	for _, tc := range []struct {
		name  string
		state uint32
		inner byte
		want  uint32
	}{
		{"idle-turn", 0xb, 1, 0x4380},
		{"completed", 0xc, 0, 0x4380},
		{"attack", 3, 0, 0x7698},
		{"pursuit", 0xb, 5, 0x7698},
	} {
		t.Run(tc.name, func(t *testing.T) {
			order := SavedActorOrder{State: tc.state}
			order.Raw[8] = tc.inner
			binary.LittleEndian.PutUint32(order.Raw[0xc:], 0x4380)
			got := ProjectActorOrderTargets(actor, order, 0x7698, 0)
			if target := binary.LittleEndian.Uint32(got.Raw[0xc:]); target != tc.want {
				t.Fatalf("order target %x, want %x", target, tc.want)
			}
		})
	}
}

func TestCurrentOrderOrdinalsKeepPayloadAndAtomicity(t *testing.T) {
	w := mustWorld(t, 7, Bounds{Width: 4, Height: 4}, []Entity{{ID: 1}, {ID: 2}})
	w.savedGroups = &savedGroupState{Orders: []SavedActorOrder{{Entity: 2}, {Entity: 1}}}
	w.savedGroups.Orders[0].Raw[0x40], w.savedGroups.Orders[1].Raw[0x40] = 17, 29
	for cycle := 0; cycle < 2; cycle++ {
		actions := w.Actions()
		cold := worldRoundTripForTest(t, w)
		slices.Reverse(cold.savedGroups.Orders)
		if err := cold.RestoreActions(actions, nil); err != nil || cold.Hash() != w.Hash() {
			t.Fatal("archive traversal changed current order insertion sequence", cycle, err)
		}
		w = cold
	}
	actions := w.Actions()
	cold := worldRoundTripForTest(t, w)
	slices.Reverse(cold.savedGroups.Orders)
	cold.savedOrder(2).Raw[0x40] = 83
	if err := cold.RestoreActions(actions, nil); err != nil || cold.savedGroups.Orders[0].Entity != 2 || cold.savedGroups.Orders[0].Raw[0x40] != 83 {
		t.Fatal("native order ordinal replaced an ordinary payload edit", err)
	}
	for _, kind := range []string{"duplicate", "missing", "outside"} {
		bad := actionCopy(t, actions)
		switch kind {
		case "duplicate":
			*bad.Actors[0].Order.Ordinal = *bad.Actors[1].Order.Ordinal
		case "missing":
			bad.Actors[0].Order.Ordinal = nil
		case "outside":
			*bad.Actors[0].Order.Ordinal = 2
		}
		before := cold.Hash()
		if err := cold.RestoreActions(bad, nil); err == nil || cold.Hash() != before {
			t.Fatal("invalid ordinal policy partially adopted an action", kind, err)
		}
	}
	legacy := actionCopy(t, actions)
	for i := range legacy.Actors {
		legacy.Actors[i].Order.Ordinal = nil
	}
	slices.Reverse(cold.savedGroups.Orders)
	before := cold.Hash()
	if err := cold.RestoreActions(legacy, nil); err != nil || cold.Hash() != before {
		t.Fatal("historical continuation without ordinals reordered ordinary rows", err)
	}
}
