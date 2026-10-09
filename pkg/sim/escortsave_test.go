package sim

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
)

func escortSaveWorld(t *testing.T) *World {
	t.Helper()
	w := mustWorld(t, 91, Bounds{7, 7}, []Entity{
		{ID: 1, X: 2, Y: 2, HP: 10, MaxHP: 10},
		{ID: 2, X: 3, Y: 2, HP: 10, MaxHP: 10, RotationSpeed: 16},
	})
	e := &w.entities[1]
	e.ActorState = actorStateFollow
	e.EscortTarget, e.HasEscortTarget, e.EscortRange = 1, true, 3
	e.EscortOrder, e.EscortTurnPending = escortOrderIdle, true
	e.requestFacing(128)
	return w
}

func TestEscortResiduesSurviveBinaryAndActionContinuations(t *testing.T) {
	w := escortSaveWorld(t)
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if form[0] != escortFormVersion || !bytes.Equal(form[len(form)-4:], []byte("ESC1")) {
		t.Fatal("escort residues have no native suffix")
	}
	if err := CheckSaveForm(form); err != nil {
		t.Fatal("escort native form refused at the save boundary", err)
	}
	var binaryWorld, actionWorld World
	for _, cold := range []*World{&binaryWorld, &actionWorld} {
		if err := cold.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
	}
	a := actionCopy(t, w.Actions())
	actionWorld.entities[1].EscortOrder, actionWorld.entities[1].EscortTurnPending = 0, false
	if err := actionWorld.RestoreActions(a, nil); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 12; tick++ {
		for _, cold := range []*World{&binaryWorld, &actionWorld} {
			if cold.Hash() != w.Hash() || cold.entities[1].DrawnFacing() != w.entities[1].DrawnFacing() {
				t.Fatalf("tick %d: cold escort differs", tick)
			}
			Step(cold, nil)
		}
		Step(w, nil)
	}
}

func TestEscortResidueHistoricalFormsDefaultToZero(t *testing.T) {
	w := escortSaveWorld(t)
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	start := len(form) - 9 - int(binary.LittleEndian.Uint32(form[len(form)-9:]))
	old := bytes.Clone(form[:start])
	old[0] = form[len(form)-5]
	cold := escortSaveWorld(t)
	if err := cold.UnmarshalBinary(old); err != nil {
		t.Fatal(err)
	}
	if cold.entities[1].EscortOrder != 0 || cold.entities[1].EscortTurnPending {
		t.Fatal("historical native form retained receiver residues")
	}
	if cold.Hash() == w.Hash() {
		t.Fatal("native loss control did not change the state hash")
	}
	w.entities[1].EscortOrder, w.entities[1].EscortTurnPending = 0, false
	a := actionCopy(t, w.Actions())
	encoded, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("EscortOrder")) || bytes.Contains(encoded, []byte("EscortTurnPending")) {
		t.Fatal("zero escort fields widen historical action records")
	}
	cold.entities[1].EscortOrder, cold.entities[1].EscortTurnPending = escortOrderIdle, true
	if err := cold.RestoreActions(a, nil); err != nil {
		t.Fatal(err)
	}
	if cold.Hash() != w.Hash() {
		t.Fatal("historical action record did not clear receiver residues")
	}
}

func TestEscortResidueEachFieldHasALossControl(t *testing.T) {
	for _, tc := range []struct {
		name    string
		order   uint8
		pending bool
	}{
		{"close", escortOrderClose, false},
		{"idle", escortOrderIdle, false},
		{"pending", escortOrderNone, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := escortSaveWorld(t)
			w.entities[1].EscortOrder, w.entities[1].EscortTurnPending = tc.order, tc.pending
			form, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold World
			if err := cold.UnmarshalBinary(form); err != nil {
				t.Fatal(err)
			}
			roundtrip, err := cold.MarshalBinary()
			if err != nil || !bytes.Equal(form, roundtrip) {
				t.Fatal("escort native bytes changed on round trip", err)
			}
			a := actionCopy(t, w.Actions())
			a.Actors[1].EscortOrder, a.Actors[1].EscortTurnPending = 0, false
			if err := cold.RestoreActions(a, nil); err != nil {
				t.Fatal(err)
			}
			if cold.Hash() == w.Hash() {
				t.Fatal("removing an escort field did not change the state hash")
			}
		})
	}
}

func TestEscortResidueDecoderRejectsMalformedSuffixAtomically(t *testing.T) {
	w := escortSaveWorld(t)
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	start := len(form) - 9 - int(binary.LittleEndian.Uint32(form[len(form)-9:]))
	for _, tc := range []struct {
		name   string
		mutate func([]byte)
	}{
		{"count", func(b []byte) { binary.LittleEndian.PutUint32(b[start:], ^uint32(0)) }},
		{"span", func(b []byte) { binary.LittleEndian.PutUint32(b[len(b)-9:], ^uint32(0)) }},
		{"order", func(b []byte) { b[start+8] = escortOrderIdle + 1 }},
		{"pending", func(b []byte) { b[start+9] = 2 }},
		{"actor", func(b []byte) { binary.LittleEndian.PutUint32(b[start+4:], 99) }},
		{"zero", func(b []byte) { b[start+8], b[start+9] = 0, 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := bytes.Clone(form)
			tc.mutate(bad)
			cold := escortSaveWorld(t)
			before := cold.Hash()
			if cold.UnmarshalBinary(bad) == nil {
				t.Fatal("malformed escort suffix accepted")
			}
			if CheckSaveForm(bad) == nil {
				t.Fatal("malformed escort suffix passed save admission")
			}
			if cold.Hash() != before {
				t.Fatal("failed decode changed receiver")
			}
		})
	}
}

func TestEscortResidueActionsRejectInvalidStateAtomically(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*ActorContinuation)
	}{
		{"order", func(v *ActorContinuation) { v.EscortOrder = escortOrderIdle + 1 }},
		{"guard", func(v *ActorContinuation) {
			v.ActorState = actorStateGuard
			v.EscortTarget, v.HasEscortTarget, v.EscortRange = 0, false, 0
		}},
		{"target", func(v *ActorContinuation) { v.HasEscortTarget = false }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := escortSaveWorld(t)
			a := actionCopy(t, w.Actions())
			tc.mutate(&a.Actors[1])
			before := w.Hash()
			if w.RestoreActions(a, nil) == nil {
				t.Fatal("invalid escort action accepted")
			}
			if w.Hash() != before {
				t.Fatal("failed action restore changed receiver")
			}
		})
	}
}

func TestEscortResiduesClearWithOrderAndConstruction(t *testing.T) {
	w := escortSaveWorld(t)
	w.entities[1].clearEscort()
	if w.entities[1].EscortOrder != 0 || w.entities[1].EscortTurnPending {
		t.Fatal("clearEscort retained escort residues")
	}
	e := Entity{ID: 1, HP: 10, MaxHP: 10, EscortOrder: escortOrderIdle, EscortTurnPending: true}
	cold := mustWorld(t, 1, Bounds{7, 7}, []Entity{e})
	if cold.entities[0].EscortOrder != 0 || cold.entities[0].EscortTurnPending {
		t.Fatal("constructor retained escort residues without an order")
	}
}
