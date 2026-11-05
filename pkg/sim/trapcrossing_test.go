package sim

import "testing"

// A walking unit's cell trap is requested on the call whose cell byte changes,
// after the position has entered the trap's cell, not on the step's start call
// (MOVE-194). Call c counts the step's start as call 1; a positive axis stride
// s crosses on call ceil(128/s) and a negative one on call floor(128/s)+1
// (MOVE-195). A standing unit requests nothing (MOVE-196).

func trapCrossingWorld(t *testing.T, x, y int32) *World {
	t.Helper()
	w, err := NewSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, nil,
		[]Entity{{ID: 7, X: x, Y: y, HP: 1000, MaxHP: 1000, Domain: DomainGround, Speed: 16}}, nil,
		[]SpellRule{{ID: 13, School: 3, DamageMin: 20, DamageMax: 20, Damaging: true, TargetsUnit: true}})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.DeclareCellTails([]CellTail{{X: 3, Y: 3, Bytes: [6]byte{13, 1, 9, 10}}}); err != nil {
		t.Fatal(err)
	}
	return w
}

func crossingCall(step int8) int {
	if step > 0 {
		return (128 + int(step) - 1) / int(step)
	}
	return 128/int(-step) + 1
}

func TestCellTrapIsRequestedOnTheStepsCrossingCall(t *testing.T) {
	for _, tc := range []struct {
		name string
		x, y int32
		axis func(NativeStride) int8
	}{
		{"east", 2, 3, func(s NativeStride) int8 { return s.StepX }},
		{"west", 4, 3, func(s NativeStride) int8 { return s.StepX }},
		{"south", 3, 2, func(s NativeStride) int8 { return s.StepY }},
		{"north", 3, 4, func(s NativeStride) int8 { return s.StepY }},
		{"north-east crosses with its y axis", 2, 4, func(s NativeStride) int8 { return s.StepY }},
		{"south-west crosses with its y axis", 4, 2, func(s NativeStride) int8 { return s.StepY }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := trapCrossingWorld(t, tc.x, tc.y)
			cmds := []Command{MoveTo(7, CellPoint{X: 3, Y: 3})}
			start, call := 0, 0
			for tick := 1; tick <= 80; tick++ {
				Step(w, cmds)
				cmds = nil
				e := w.entities[0]
				if start == 0 && e.X == 3 && e.Y == 3 {
					if !e.Stride.Present {
						t.Fatal("the step retained no stride")
					}
					start, call = tick, crossingCall(tc.axis(e.Stride))
					if call < 2 {
						t.Fatalf("stride %+v crosses on its start call", e.Stride)
					}
				}
				requested := len(w.casts) != 0
				want := start != 0 && tick == start+call-1
				if requested != want {
					t.Fatalf("tick %d (step start %d, crossing call %d): requested=%v casts=%+v", tick, start, call, requested, w.casts)
				}
				if want && (len(w.casts) != 1 || w.casts[0].Target != 7 || w.casts[0].Spell != 13) {
					t.Fatalf("crossing request = %+v", w.casts)
				}
			}
			if start == 0 {
				t.Fatal("the unit never stepped onto the trap cell")
			}
		})
	}
}

func TestCellTrapStandingUnitIsNotCastAgain(t *testing.T) {
	w := trapCrossingWorld(t, 2, 3)
	cmds := []Command{MoveTo(7, CellPoint{X: 3, Y: 3})}
	requests := 0
	for tick := 0; tick < 120; tick++ {
		Step(w, cmds)
		cmds = nil
		requests += len(w.casts)
	}
	if e := w.entities[0]; e.X != 3 || e.Y != 3 || e.Transit != 0 {
		t.Fatalf("the unit did not come to rest on the trap: %+v", e)
	}
	if requests != 1 {
		t.Fatalf("a step and its rest requested %d casts, want 1", requests)
	}
}
