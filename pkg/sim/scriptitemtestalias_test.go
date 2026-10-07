package sim

import "testing"

// TestBothItemTestOpcodesShareOneSupportAndDispatchArm is the executable
// counterpart of TRIG-ITEMTEST-040's byte-identity result. Both numeric table
// entries cross NewScript, the ordinary phase-6 dispatch and the same result
// cases; no wrapper can accidentally implement only the common happy path.
func TestBothItemTestOpcodesShareOneSupportAndDispatchArm(t *testing.T) {
	for _, op := range []int32{ScriptCheckItemTestAlias, ScriptCheckItemTest} {
		op := op
		t.Run(checkOpcodeName(op), func(t *testing.T) {
			t.Parallel()
			for _, tc := range []struct {
				name  string
				node  ScriptCheck
				stock []Stock
				want  int32
			}{
				{"held in the named container",
					ScriptCheck{Op: op, Register: caRegister, Unit: caSubject, HasUnit: true, Item: caCure, HasItem: true},
					[]Stock{{ID: caSubject, Items: []uint16{caCure}}}, 1},
				{"different code in the named container",
					ScriptCheck{Op: op, Register: caRegister, Unit: caSubject, HasUnit: true, Item: caCure, HasItem: true},
					[]Stock{{ID: caSubject, Items: []uint16{caOther}}}, 0},
				{"held only in another container",
					ScriptCheck{Op: op, Register: caRegister, Unit: caSubject, HasUnit: true, Item: caCure, HasItem: true},
					[]Stock{{ID: caTarget, Items: []uint16{caCure}}}, 0},
				{"item field absent despite a stale code",
					ScriptCheck{Op: op, Register: caRegister, Unit: caSubject, HasUnit: true, Item: caCure},
					[]Stock{{ID: caSubject, Items: []uint16{caCure}}}, 0},
			} {
				t.Run(tc.name, func(t *testing.T) {
					s := mustScript(t, []ScriptCheck{tc.node}, nil, nil)
					if gaps := s.Unsupported(); len(gaps) != 0 {
						t.Fatalf("Unsupported() = %+v, want none", gaps)
					}
					w := caStockedWorld(t, s, tc.stock)
					for w.Tick()%16 != 6 {
						Step(w, nil)
					}
					trace := StepTraced(w, nil)
					if !trace.Pass || len(trace.Checks) != 1 {
						t.Fatalf("StepTraced pass=%v checks=%+v, want one check on the script pass",
							trace.Pass, trace.Checks)
					}
					run := trace.Checks[0]
					if run.Op != op || !run.Dispatched || !run.Wrote || run.HasSilence || run.Value != tc.want {
						t.Errorf("opcode %d run = %+v, want dispatched write of %d", op, run, tc.want)
					}
				})
			}
		})
	}
}

// TestBothItemTestOpcodesLeaveTheUnsupportedCensusSeamIntact proves both
// directions of the compile-time boundary: 12 and 17 disappear from the gap
// report, while a genuinely unsupported neighbour is still reported with its
// exact index. This catches a census filter that merely hides all checks.
func TestBothItemTestOpcodesLeaveTheUnsupportedCensusSeamIntact(t *testing.T) {
	for _, op := range []int32{ScriptCheckItemTestAlias, ScriptCheckItemTest} {
		s := mustScript(t, []ScriptCheck{
			{Op: op, Register: 0},
			{Op: scriptCheckSentinelOp, Register: 1},
		}, nil, nil)
		gaps := s.Unsupported()
		if len(gaps) != 1 || gaps[0].Kind != ScriptGapCheck || gaps[0].Op != scriptCheckSentinelOp || gaps[0].Index != 1 {
			t.Errorf("opcode %d Unsupported() = %+v, want only check %d at index 1",
				op, gaps, scriptCheckSentinelOp)
		}
	}
}

func checkOpcodeName(op int32) string {
	if op == ScriptCheckItemTestAlias {
		return "opcode 12"
	}
	return "opcode 17"
}
