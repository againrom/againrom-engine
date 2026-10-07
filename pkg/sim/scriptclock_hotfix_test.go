package sim

import "testing"

// SAV-650: the saved session counter increments before both evaluators.
func TestScriptClock93PrecedesChecksAndSurvivesNativeLoad(t *testing.T) {
	for _, sessionClock := range []bool{false, true} {
		t.Run(map[bool]string{false: "native-clock", true: "original-clock"}[sessionClock], func(t *testing.T) {
			s := mustScript(t,
				[]ScriptCheck{{Op: ScriptCheckVariable, Register: 0, Args: [scriptParams]int32{93}}, constCheck(1, 42)},
				[]ScriptInstant{{Op: ScriptInstantIncVariable, Args: [scriptParams]int32{8}}},
				[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ), pair(93, 1, ScriptCmpEQ)}, Instants: acts(0), Once: true}})
			w := scriptWorld(t, s, nil)
			w.hasSessionClock = sessionClock
			w.registers[93] = 41
			scriptTicks(w, 6, nil)
			if got := w.ScriptRegister(93); got != 41 {
				t.Fatalf("counter advanced outside the script phase: %d", got)
			}
			cut, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold World
			if err := cold.UnmarshalBinary(cut); err != nil {
				t.Fatal(err)
			}
			for next := 0; next < 33; next++ {
				Step(w, nil)
				Step(&cold, nil)
				if w.Hash() != cold.Hash() {
					t.Fatalf("native continuation differs at step %d", next)
				}
				if next == 0 {
					if w.ScriptRegister(93) != 42 || w.ScriptRegister(0) != 42 || w.ScriptRegister(8) != 1 {
						t.Fatalf("first pass counter=%d check=%d trigger=%d; want42,42,1",
							w.ScriptRegister(93), w.ScriptRegister(0), w.ScriptRegister(8))
					}
				}
			}
			if w.ScriptRegister(93) != 44 || w.ScriptRegister(8) != 1 {
				t.Fatalf("three passes counter=%d trigger=%d; want44,1", w.ScriptRegister(93), w.ScriptRegister(8))
			}
		})
	}
}

func TestScriptClock93WrapsBeforeItsConsumers(t *testing.T) {
	s := mustScript(t,
		[]ScriptCheck{{Op: ScriptCheckVariable, Register: 0, Args: [scriptParams]int32{93}}}, nil, nil)
	w := scriptWorld(t, s, nil)
	w.registers[93] = 2147483647
	w.scriptPass(nil)
	if w.ScriptRegister(93) != -2147483648 || w.ScriptRegister(0) != -2147483648 {
		t.Fatalf("wrapped counter=%d check=%d", w.ScriptRegister(93), w.ScriptRegister(0))
	}
}

func TestScriptClock93EmptyProgramHasCanonicalNativeContinuation(t *testing.T) {
	for _, session := range []bool{false, true} {
		for _, empty := range []bool{false, true} {
			var s *Script
			if empty {
				s = mustScript(t, nil, nil, nil)
			}
			w := scriptWorld(t, s, nil)
			w.hasSessionClock = session
			w.registers[93] = 41
			cut, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold World
			if err := cold.UnmarshalBinary(cut); err != nil {
				t.Fatal(err)
			}
			for next := 0; next < 32; next++ {
				Step(w, nil)
				Step(&cold, nil)
				if w.Hash() != cold.Hash() {
					t.Fatalf("session=%v empty=%v step=%d: original and cold worlds differ", session, empty, next)
				}
			}
			want := int32(41)
			if session {
				want = 43
			}
			if got := w.ScriptRegister(93); got != want {
				t.Fatalf("session=%v empty=%v counter=%d want=%d", session, empty, got, want)
			}
		}
	}
}
