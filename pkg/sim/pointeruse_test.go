package sim

import (
	"reflect"
	"testing"
)

// pointerWorld is a second-game world with one 1x1 Pointer at (8,8) whose
// action slot is slot, and two message instants from source slots 0 and 1.
func pointerWorld(t *testing.T, kind uint16, slot int32) *World {
	t.Helper()
	return pointerWorldWith(t, kind, slot, []ScriptInstant{{Op: ScriptInstantMessage, Args: [10]int32{7}}, {Slot: 1, Op: ScriptInstantMessage, Args: [10]int32{19}}})
}

func pointerWorldWith(t *testing.T, kind uint16, slot int32, instants []ScriptInstant) *World {
	t.Helper()
	s, err := NewROM2Script(nil, instants, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := Structure{ID: 0, Kind: kind, UseAmount: slot, Col: 8, Row: 8, Width: 1, Height: 1, Attach: 1, Blocking: 1}
	a := structureCombatActor()
	a.Owner = SelfSlot
	w, err := NewStructuredWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical,
		Terrain{}, []Entity{a}, s, Relations{}, nil, nil, nil, GhostTemplate{}, []Structure{p})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// A use of a Pointer walks the actor within range 1 of a 1-wide structure
// (R2-ENGINE-607) and on arrival runs the instant at its slot once.
func TestPointerUseWalksThenRunsItsSlotOnce(t *testing.T) {
	w := pointerWorld(t, 0x4a, 1)
	messages := StepReported(w, []Command{UseStructure(0, 0)}).ScriptMessages
	if len(w.StructureUses()) != 1 {
		t.Fatal("the Pointer use was not admitted")
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatalf("a world approaching a Pointer did not reload: %v", err)
	}
	for range 400 {
		messages = append(messages, StepReported(w, nil).ScriptMessages...)
		Step(&cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatal("the reloaded approach diverged")
		}
		if len(w.StructureUses()) == 0 {
			break
		}
	}
	if len(w.StructureUses()) != 0 {
		t.Fatalf("the Pointer use did not finish: actor at %d,%d target %v", w.entities[0].X, w.entities[0].Y, w.entities[0].HasTarget)
	}
	if d := cellOf(&w.entities[0]).chebyshevTo(cell{8, 8}); d > 1 {
		t.Fatalf("the actor stopped %d cells from the Pointer", d)
	}
	for range 40 {
		messages = append(messages, StepReported(w, nil).ScriptMessages...)
	}
	if !reflect.DeepEqual(messages, []int32{19}) {
		t.Fatalf("messages %v, want [19]", messages)
	}
}

func TestPointerWithoutAPositiveSlotIsNotUsable(t *testing.T) {
	for _, tc := range []struct {
		kind uint16
		slot int32
		want bool
	}{{0x46, 1, true}, {0x4b, 3, true}, {0x4a, 0, false}, {0x45, 1, false}, {0x4c, 1, false}} {
		s := Structure{Kind: tc.kind, UseAmount: tc.slot, Width: 1, Height: 1}
		if s.Usable() != tc.want {
			t.Errorf("kind %#x slot %d usable %v", tc.kind, tc.slot, !tc.want)
		}
	}
	w := pointerWorld(t, 0x4a, 0)
	Step(w, []Command{UseStructure(0, 0)})
	if len(w.StructureUses()) != 0 {
		t.Fatal("a Pointer with slot 0 admitted a use")
	}
}

func TestPointerSlotPastTheProgramRunsNothing(t *testing.T) {
	w := pointerWorld(t, 0x4a, 2)
	var messages []int32
	messages = append(messages, StepReported(w, []Command{UseStructure(0, 0)}).ScriptMessages...)
	for range 400 {
		messages = append(messages, StepReported(w, nil).ScriptMessages...)
	}
	if len(messages) != 0 {
		t.Fatalf("messages %v", messages)
	}
}

// pointerMessages uses the Pointer and returns the messages of 400 steps.
func pointerMessages(w *World) []int32 {
	messages := StepReported(w, []Command{UseStructure(0, 0)}).ScriptMessages
	for range 400 {
		messages = append(messages, StepReported(w, nil).ScriptMessages...)
	}
	return messages
}

// The compiler omitted source slot 1, so the instant from slot 2 is the
// program's second: a Pointer naming slot 2 runs it, and one naming the
// omitted slot 1 runs nothing. Indexing the program by the slot fails both.
func TestPointerRunsTheInstantCompiledFromItsSlot(t *testing.T) {
	instants := []ScriptInstant{{Op: ScriptInstantMessage, Args: [10]int32{7}}, {Slot: 2, Op: ScriptInstantMessage, Args: [10]int32{19}}}
	if got := pointerMessages(pointerWorldWith(t, 0x4a, 2, instants)); !reflect.DeepEqual(got, []int32{19}) {
		t.Fatalf("slot 2 gave messages %v, want [19]", got)
	}
	if got := pointerMessages(pointerWorldWith(t, 0x4a, 1, instants)); len(got) != 0 {
		t.Fatalf("the omitted slot 1 gave messages %v", got)
	}
}

// The slots ride the byte form's script slot section; a form without it
// reads slot 0 for every instant, which no Pointer names.
func TestScriptSlotsSurviveTheByteForm(t *testing.T) {
	w := pointerWorldWith(t, 0x4a, 2, []ScriptInstant{{Op: ScriptInstantMessage, Args: [10]int32{7}}, {Slot: 2, Op: ScriptInstantMessage, Args: [10]int32{19}}})
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if form[0] != scriptSlotFormVersion {
		t.Fatalf("form version %d, want %d", form[0], scriptSlotFormVersion)
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if got := cold.script.instants[1].Slot; got != 2 || cold.Hash() != w.Hash() {
		t.Fatalf("reloaded slot %d, hash equal %v", got, cold.Hash() == w.Hash())
	}
	if got := pointerMessages(&cold); !reflect.DeepEqual(got, []int32{19}) {
		t.Fatalf("reloaded messages %v", got)
	}
	plain := pointerWorld(t, 0x4a, 1)
	plain.script.instants[1].Slot = 0
	old, err := plain.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if old[0] == scriptSlotFormVersion {
		t.Fatal("a program without slots wrote the slot section")
	}
	var oldCold World
	if err := oldCold.UnmarshalBinary(old); err != nil {
		t.Fatal(err)
	}
	if got := pointerMessages(&oldCold); len(got) != 0 {
		t.Fatalf("an older form's Pointer gave messages %v", got)
	}
	bad := append([]byte(nil), form...)
	bad[len(bad)-9-4*2-4]++
	if err := new(World).UnmarshalBinary(bad); err == nil {
		t.Fatal("a slot count that is not the instant count was read")
	}
}

func TestRestoreStructureUseMetadataFillsOnlyAnEmptyPointerSlot(t *testing.T) {
	w := pointerWorld(t, 0x4a, 0)
	w.RestoreStructureUseMetadata([]Structure{{ID: 0, Kind: 0x4a, UseAmount: 5}})
	if w.structures[0].UseAmount != 5 {
		t.Fatalf("slot %d, want 5", w.structures[0].UseAmount)
	}
	w.RestoreStructureUseMetadata([]Structure{{ID: 0, Kind: 0x4a, UseAmount: 6}})
	if w.structures[0].UseAmount != 5 {
		t.Fatal("a held slot was replaced")
	}
	f := pointerWorld(t, 15, 0)
	f.RestoreStructureUseMetadata([]Structure{{ID: 0, Kind: 15, UseAmount: 9}})
	if f.structures[0].UseAmount != 0 {
		t.Fatal("a fountain's held amount was replaced")
	}
}
