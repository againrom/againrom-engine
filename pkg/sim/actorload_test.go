package sim

import (
	"encoding/binary"
	"reflect"
	"testing"
)

// Independent append-only form-75 layout. Empty native worlds gain only a
// zero span; old versions never acquire a source bookkeeping marker.
func widenedActorLoadPin(old []byte) []byte {
	end := len(old) - 2500
	out := append([]byte(nil), old[:end]...)
	out = append(out, 0, 0, 0, 0)
	out = append(out, old[end:]...)
	out[0] = 75
	return out
}

func strippedActorLoadPin(form []byte) []byte {
	out := strippedSourceBindingPin(form)
	if out[0] >= 75 {
		end := len(out) - 2500 - 4
		span := int(binary.LittleEndian.Uint32(out[end:]) & 0x7fffffff)
		out = append(out[:end-span], out[end+4:]...)
		out[0] = 74
	}
	return out
}

func TestActorLoad1110CurrentWordsOrderedTransferAndNativeContinuation(t *testing.T) {
	w := srWorld(t, nil, nil, []Entity{
		{ID: 1, X: 1, Y: 1, HP: 100, MaxHP: 100, Speed: 20},
		{ID: 2, X: 2, Y: 1, HP: 100, MaxHP: 100, Speed: 21},
	})
	if err := w.DeclareItemWeights([]ItemWeight{{Code: 0x0e01, Weight: 5}, {Code: 0x0e02, Weight: 4}}); err != nil {
		t.Fatal(err)
	}
	one := ActorLoadSnapshot{Inventory: ActorLoad{Present: true, OwnWeight: 7, ContainerPresent: true, InsertIndex: 10000, Accumulator: 17}, Load: 53, Capacity: 31, Speed: 20,
		Movement: HumanMovement{Present: true, RawSpeed: 18, NativeSpeed: 20, Load: 53, Capacity: 31}}
	two := ActorLoadSnapshot{Inventory: ActorLoad{Present: true, OwnWeight: 2, ContainerPresent: true, InsertIndex: 1, Accumulator: -6}, Load: 0, Capacity: 31, Speed: 21,
		Movement: HumanMovement{Present: true, RawSpeed: 19, NativeSpeed: 21, Load: 0, Capacity: 31}}
	if err := w.ImportOriginalActorStock([]OriginalActorStock{
		{ID: 1, Carried: []ItemStack{StackItem(PlainItem(0x0e01), 2), StackItem(PlainItem(0x0e02), 1), StackItem(PlainItem(0x0e01), 1)}, LoadState: &one},
		{ID: 2, Carried: []ItemStack{StackItem(PlainItem(0x0e02), 1), StackItem(PlainItem(0x0e03), 1)}, LoadState: &two},
	}); err != nil {
		t.Fatal(err)
	}
	if w.entities[0].Load != 53 || w.entities[1].Load != 0 || len(w.carried[0]) != 3 {
		t.Fatal("LOAD derived or folded source state")
	}
	if rate, _, _, _ := w.StepRate(1, 2, 1); rate != 18 {
		t.Fatalf("source speed=%d", rate)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if back.Hash() != w.Hash() {
		t.Fatal("native LOAD changed current state")
	}
	for _, current := range []*World{w, &back} {
		if err := current.MoveCarried(1, 2, 0x0e01, 1); err != nil {
			t.Fatal(err)
		}
		if e := current.entities[0]; e.Load != 13 || e.ActorLoad.Accumulator != 12 || e.HumanMovement.Present {
			t.Fatalf("source transfer state %+v", e.CurrentActorLoad())
		}
		if e := current.entities[1]; e.Load != 2 || e.ActorLoad.Accumulator != -1 || !e.HumanMovement.Present || e.HumanMovement.RawSpeed != 19 {
			t.Fatalf("destination transfer state %+v", e.CurrentActorLoad())
		}
		got := []uint16{current.carried[1][0].Code, current.carried[1][1].Code, current.carried[1][2].Code}
		if !reflect.DeepEqual(got, []uint16{0x0e02, 0x0e01, 0x0e03}) {
			t.Fatalf("stored insertion index ignored: %x", got)
		}
		if len(current.carried[0]) != 3 {
			t.Fatal("unrelated equal source elements folded")
		}
	}
	if w.Hash() != back.Hash() {
		t.Fatal("next action differs after native LOAD")
	}
}

func TestActorLoad1110AbsentContainerAndSignedLoad(t *testing.T) {
	w := srWorld(t, nil, nil, []Entity{{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 10}, {ID: 2, X: 2, Y: 1, HP: 10, MaxHP: 10}})
	s := ActorLoadSnapshot{Inventory: ActorLoad{Present: true, OwnWeight: -7}, Load: -13, Capacity: 300, Speed: 20,
		Movement: HumanMovement{Present: true, RawSpeed: -12, NativeSpeed: 20, Load: -13, Capacity: 300}}
	if err := w.ImportOriginalActorStock([]OriginalActorStock{{ID: 1, LoadState: &s}, {ID: 2, Carried: []ItemStack{StackItem(PlainItem(0x0e01), 1)}}}); err != nil {
		t.Fatal(err)
	}
	hash := w.Hash()
	if err := w.MoveCarried(2, 1, 0x0e01, 1); err == nil || hash != w.Hash() {
		t.Fatal("absent destination mutated")
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if back.entities[0].Load != -13 || back.entities[0].ActorLoad.ContainerPresent {
		t.Fatal("signed/absent state lost")
	}
}
