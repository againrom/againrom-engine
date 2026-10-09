package sim

import (
	"reflect"
	"testing"
)

func TestOriginalFacingOnlyChangesCurrentAndIdleDesiredDirection(t *testing.T) {
	w := poolImportWorld(t)
	before := w.Entities()
	if err := w.ImportOriginalActorFacings([]OriginalActorFacing{{ID: 1, Facing: 171}, {ID: 2, Facing: 0}}); err != nil {
		t.Fatal(err)
	}
	before[0].Facing, before[0].DesiredFacing = 171, 171
	if !reflect.DeepEqual(before, w.Entities()) {
		t.Fatal("facing import changed other actor fields")
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() {
		t.Fatalf("native facing roundtrip: %v", err)
	}
}

func TestOriginalFacingBatchFailureDoesNotWriteTheValidPrefix(t *testing.T) {
	for _, bad := range []OriginalActorFacing{{ID: 1}, {ID: 99}, {ID: 2}} {
		w := poolImportWorld(t)
		if bad.ID == 2 {
			w.entities[1].TurnRemaining, w.entities[1].TurnTotal = 8, 8
			w.entities[1].DesiredFacing = 128
		}
		before := w.Hash()
		if err := w.ImportOriginalActorFacings([]OriginalActorFacing{{ID: 1, Facing: 160}, bad}); err == nil || w.Hash() != before {
			t.Fatalf("late invalid %+v changed state or passed: %v", bad, err)
		}
	}
	for _, offMap := range []bool{false, true} {
		w := poolImportWorld(t)
		if offMap {
			w.entities[1].OffMap = true
		} else {
			w.entities[1].HP = 0
		}
		before := w.Hash()
		err := w.ImportOriginalActorFacings([]OriginalActorFacing{{ID: 1, Facing: 160}, {ID: 2, Facing: 224}})
		if offMap {
			if err != nil || !w.entities[1].OffMap || w.entities[1].Facing != 224 || w.entities[0].Facing != 160 {
				t.Fatal("current detached facing was not imported", err)
			}
		} else if err == nil || w.Hash() != before {
			t.Fatalf("dead target accepted or prefix written: %v", err)
		}
	}
	var absent *World
	if err := absent.ImportOriginalActorFacings(nil); err == nil {
		t.Fatal("nil world accepted")
	}
}

func TestOriginalFacingChangesTheNextMoveTurnAndSurvivesNativeContinuation(t *testing.T) {
	b := Bounds{Width: 7, Height: 7}
	w := mustWorldGrid(t, 1, b, ModeCanonical, openGrid(b), []Entity{turnActor(1, 3, 3, 0, 16)})
	if err := w.ImportOriginalActorFacings([]OriginalActorFacing{{ID: 1, Facing: 160}}); err != nil {
		t.Fatal(err)
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	cmd := []Command{{Entity: 1, X: 3, Y: 1}}
	Step(w, cmd)
	Step(&cold, cmd)
	e := w.entities[0]
	// North0 from southwest160 takes the shorter96-byte arc: ceil(96/16)=6.
	// Starting north by mistake would begin movement, not this stationary turn.
	if e.X != 3 || e.Y != 3 || e.Facing != 176 || e.DesiredFacing != 0 || e.TurnRemaining != 6 || e.TurnTotal != 6 {
		t.Fatalf("next move did not use imported direction: %+v", e)
	}
	for i := 0; i < 32; i++ {
		if w.Hash() != cold.Hash() {
			t.Fatalf("native continuation changed at step%d", i)
		}
		Step(w, nil)
		Step(&cold, nil)
	}
}
