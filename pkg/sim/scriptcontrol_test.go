package sim

import (
	"bytes"
	"testing"
)

func TestAControlledProgramRunsOnlyThroughTheOrdinaryScriptPass(t *testing.T) {
	base := sttWorld(t, sttScript(t,
		[]ScriptCheck{{Op: ScriptCheckConstant, Register: 7, Args: [scriptParams]int32{9}}},
		nil, nil), sttEnts())
	before, err := base.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary(base): %v", err)
	}

	probe := sttScript(t,
		[]ScriptCheck{{Op: ScriptCheckConstant, Register: 5, Args: [scriptParams]int32{3}}},
		[]ScriptInstant{{Op: ScriptInstantIncVariable, Args: [scriptParams]int32{5}}},
		[]ScriptTrigger{{
			Instants: [4]int32{0, ScriptNone, ScriptNone, ScriptNone}, Once: true, Latch: 12,
		}})
	controlled, err := NewControlledScriptWorld(base, probe)
	if err != nil {
		t.Fatalf("NewControlledScriptWorld: %v", err)
	}
	if got := controlled.ScriptRegister(5); got != 3 {
		t.Fatalf("controlled constant preset = %d, want 3", got)
	}
	tr := sttPass(controlled)
	if len(tr.Firings) != 1 || len(tr.Firings[0].Instants) != 1 ||
		tr.Firings[0].Instants[0].Outcome != ScriptInstantStateChanged {
		t.Fatalf("ordinary pass trace = %+v", tr)
	}
	if got := controlled.ScriptRegister(5); got != 4 {
		t.Errorf("ordinary dispatch left register 5 at %d, want 4", got)
	}

	after, err := base.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary(base after witness): %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("building or running the controlled copy changed its source mission")
	}
}

func TestAControlledProgramKeepsMissionStateOutsideTheProgram(t *testing.T) {
	base := sttWorld(t, nil, sttEnts())
	base.ghost = GhostTemplate{Class: 1, RotationSpeed: 16}
	base.sourceDerive = func(s SourceActor, _ int32, _ Rules) (SourceActor, error) { s.Sight = 123; return s, nil }
	probe := sttScript(t, nil, nil, nil)
	controlled, err := NewControlledScriptWorld(base, probe)
	if err != nil {
		t.Fatalf("NewControlledScriptWorld: %v", err)
	}
	if got, want := controlled.Entities(), base.Entities(); len(got) != len(want) || got[1] != want[1] {
		t.Errorf("controlled world entities = %+v, want mission entities %+v", got, want)
	}
	if controlled.Bounds() != base.Bounds() {
		t.Errorf("controlled bounds = %+v, want %+v", controlled.Bounds(), base.Bounds())
	}
	if controlled.ghost != base.ghost || controlled.sourceDerive == nil {
		t.Fatal("controlled copy lost installed construction rules")
	}
	if got, err := controlled.sourceDerive(SourceActor{}, 0, Rules{}); err != nil || got.Sight != 123 {
		t.Fatal("controlled copy changed its arithmetic rule")
	}
}

func TestControlledScriptWorldRequiresBothInputs(t *testing.T) {
	base := sttWorld(t, nil, nil)
	probe := sttScript(t, nil, nil, nil)
	if _, err := NewControlledScriptWorld(nil, probe); err == nil {
		t.Fatal("NewControlledScriptWorld accepted a nil mission world")
	}
	if _, err := NewControlledScriptWorld(base, nil); err == nil {
		t.Fatal("NewControlledScriptWorld accepted a nil replacement program")
	}
}
