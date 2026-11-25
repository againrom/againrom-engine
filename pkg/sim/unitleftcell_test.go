package sim

import "testing"

func TestUnitLeftCellWalkAndTeleport(t *testing.T) {
	for _, route := range []string{"walk", "teleport"} {
		t.Run(route, func(t *testing.T) {
			cs := []ScriptCheck{{Op: 0x4105, HasUnit: true, Unit: 1, Args: [10]int32{10, 10}}, constCheck(1, 1)}
			s, err := NewExtendedScript(100, cs, []ScriptInstant{{Op: ScriptInstantIncVariable, Args: [10]int32{7}}},
				[]ScriptTrigger{{Instants: acts(), Conditions: []ScriptPair{pair(0, 1, ScriptCmpEQ)}, Actions: []int32{0}}})
			if err != nil {
				t.Fatal(err)
			}
			w := scriptWorld(t, s, []Entity{{ID: 1, Owner: SelfSlot, X: 10, Y: 10, HP: 100, MaxHP: 100}})
			scriptTicks(w, 32, nil)
			if w.ScriptRegister(7) != 0 {
				t.Fatal("fired on the starting cell")
			}
			if route == "teleport" {
				if err := w.DeclareCellTails([]CellTail{firstTeleport}); err != nil {
					t.Fatal(err)
				}
				w.stepCentred(0)
				w.stepScriptCasts(nil)
				if e, _ := w.Entity(1); e.X != 30 || e.Y != 40 {
					t.Fatalf("teleport reached (%d,%d)", e.X, e.Y)
				}
				scriptTicks(w, 16, nil)
			} else {
				scriptTicks(w, 16, []Command{MoveTo(1, CellPoint{X: 10, Y: 11})})
			}
			if got := w.ScriptRegister(7); got != 1 {
				t.Fatalf("first pass away fired %d times, want 1", got)
			}
			scriptTicks(w, 16, nil)
			if got := w.ScriptRegister(7); got != 2 {
				t.Fatalf("second pass away fired %d times, want 2", got)
			}
			if route == "walk" {
				scriptTicks(w, 32, []Command{MoveTo(1, CellPoint{X: 10, Y: 10})})
				if got := w.ScriptRegister(7); got != 2 {
					t.Fatalf("return to the starting cell fired again: %d", got)
				}
			}
		})
	}
}

func TestUnitLeftCellIsInertOnPlainMachines(t *testing.T) {
	checks := []ScriptCheck{{Op: 0x4105, Unit: 1, HasUnit: true}, constCheck(1, 0)}
	trigger := ScriptTrigger{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)}, Instants: acts(0)}
	for _, build := range []func([]ScriptCheck, []ScriptInstant, []ScriptTrigger) (*Script, error){NewScript, NewROM2Script} {
		s, err := build(checks, []ScriptInstant{{Op: ScriptInstantIncVariable, Args: [10]int32{7}}}, []ScriptTrigger{trigger})
		if err != nil {
			t.Fatal(err)
		}
		if !s.Triggers()[0].Inert {
			t.Fatalf("dialect %v executed the extension in an original-format trigger", s.Dialect())
		}
	}
}

func TestUnitLeftCellUsesTheRetainedCell(t *testing.T) {
	for _, tc := range []struct {
		name    string
		hp, y   int32
		removed bool
		want    int32
	}{
		{"dead on the starting cell", -10, 10, false, 0},
		{"dead on another cell", -10, 11, false, 1},
		{"removed on the starting cell", -10, 10, true, 0},
		{"removed on another cell", -10, 11, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := ScriptCheck{Op: 0x4105, Unit: 1, HasUnit: true, Args: [10]int32{10, 10}}
			s, err := NewExtendedScript(100, []ScriptCheck{c}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			var ents []Entity
			if !tc.removed {
				ents = []Entity{{ID: 1, X: 10, Y: tc.y, HP: tc.hp, MaxHP: 100}}
			}
			w := scriptWorld(t, s, ents)
			if tc.removed {
				w.currentTerminalActors = []CurrentTerminalActor{{ID: 1, Cell: uint16(tc.y<<8 | 10), Stage: uint8(DecayBones), HP: tc.hp}}
			}
			w.scriptPass(nil)
			if got := w.checkResult(0); got != tc.want {
				t.Fatalf("check = %d, want %d", got, tc.want)
			}
		})
	}
}
