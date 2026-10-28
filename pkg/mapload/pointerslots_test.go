package mapload

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// A gate-1 record names a Pointer by its type-4 id; its slot counts the
// action nodes before Position that the compiler takes as runtime nodes,
// below 0x10002 unsigned. The last record wins, a gate-0 record and a
// non-Pointer kind take none, and a Position past the nodes takes none.
func TestPointerSlotsCountTheNodesBeforeThePosition(t *testing.T) {
	objects := []alm.Object{
		{Kind: 0x4a, Field12: 42},
		{Kind: 0x30, Field12: 43},
		{Kind: 0x46, Field12: 44},
		{Kind: 0x48, Field12: 45},
		{Kind: 0x4b, Field12: 46},
	}
	actions := []alm.ScriptNode{{Opcode: 0x10002}, {Opcode: 2}, {Opcode: 0x80000000}, {Opcode: 7}, {Opcode: 2}}
	recs := []alm.StructureAction{
		{ID: 42, Gate: 1, Position: 1},
		{ID: 42, Gate: 1, Position: 4},
		{ID: 43, Gate: 1, Position: 4},
		{ID: 44, Gate: 0, Position: 4},
		{ID: 45, Gate: 1, Position: 5},
		{ID: 46, Gate: 2, Position: 2},
	}
	got := pointerSlots(objects, recs, actions)
	if want := []int32{2, 0, 0, 0, 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("slots %v, want %v", got, want)
	}
	if got := pointerSlots(objects, []alm.StructureAction{{ID: 44, Gate: 0, Position: 4}}, actions); got != nil {
		t.Fatalf("a gate-0 record gave slots %v", got)
	}
	if got := PointerSlots(&alm.Map{Objects: objects}); got != nil {
		t.Fatalf("a map without type-11c records gave slots %v", got)
	}
}

// An omitted node before the Pointer's node (unit 7 resolves to nothing) and
// a high-bit node: the Pointer's slot is the source slot the compiler stamps
// on the instant it built from the Pointer's node, while that instant sits at
// a lower program index.
func TestPointerSlotNamesTheInstantCompiledFromItsNode(t *testing.T) {
	msg := func(id, event uint32) alm.ScriptNode {
		n := alm.ScriptNode{Opcode: 2, ID: id}
		n.Type[0], n.Value[0] = 1, event
		return n
	}
	missing := alm.ScriptNode{Opcode: 12, ID: 3}
	missing.Type[0], missing.Value[0] = 4, 7
	actions := []alm.ScriptNode{msg(1, 5), {Opcode: 0x10002, ID: 2}, missing, {Opcode: 0x80000000, ID: 4}, msg(5, 19)}
	objects := []alm.Object{{Kind: 0x4a, Field12: 42}}
	slots := pointerSlots(objects, []alm.StructureAction{{ID: 42, Gate: 1, Position: 4}}, actions)
	s, rep, err := CompileROM2ScriptFrom(alm.Script{Actions: actions}, ScriptRefs{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.OmittedActions) != 1 || len(slots) != 1 || slots[0] != 2 {
		t.Fatalf("omitted %v slots %v, want one omitted node and slot 2", rep.OmittedActions, slots)
	}
	var hit []sim.ScriptInstant
	for _, in := range s.Instants() {
		if in.Slot == slots[0] {
			hit = append(hit, in)
		}
	}
	if len(hit) != 1 || hit[0].Op != 2 || hit[0].Args[0] != 19 {
		t.Fatalf("instants from slot %d: %+v, want the event 19 message", slots[0], hit)
	}
	if got := len(s.Instants()); got != 2 {
		t.Fatalf("%d instants, want 2", got)
	}
}
