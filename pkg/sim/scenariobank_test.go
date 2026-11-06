package sim

import "testing"

// The objective writer against the quest machine's state table: each mode of
// instant 36 from each listed state, its stored value and its event.
func TestWriteObjectiveFollowsTheStateTable(t *testing.T) {
	for _, tc := range []struct {
		name            string
		old, mode, want int32
		event           int32
	}{
		{"mode 1 opens state 0", 0, 1, 1, 0},
		{"mode 1 keeps state 1", 1, 1, 1, 0},
		{"mode 1 keeps state 3", 3, 1, 3, 0},
		{"mode 1 keeps state 5", 5, 1, 5, 0},
		{"mode 2 completes state 0", 0, 2, 3, ObjectiveEventDone},
		{"mode 2 completes state 1", 1, 2, 3, ObjectiveEventDone},
		{"mode 2 repeats on state 3 silently", 3, 2, 3, 0},
		{"mode 2 completes state 5", 5, 2, 3, ObjectiveEventDone},
		{"mode 2 leaves state 4", 4, 2, 4, 0},
		{"mode 2 over bit 2 is silent", 6, 2, 3, 0},
		{"mode 4 from state 0", 0, 4, 5, ObjectiveEventMode4},
		{"mode 4 from state 3", 3, 4, 5, ObjectiveEventMode4},
		{"mode 4 repeats", 5, 4, 5, ObjectiveEventMode4},
		{"mode 4 from state 4", 4, 4, 5, ObjectiveEventMode4},
		{"mode 0 stores 0", 3, 0, 0, 0},
		{"mode 8 stores 8", 1, 8, 8, 0},
		{"negative mode stored", 0, -9, -9, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bank := NewGameScenarioBank()
			bank[ObjectiveSlotBase+2] = tc.old
			before := bank
			event := WriteObjective(&bank, 2, tc.mode)
			if bank[ObjectiveSlotBase+2] != tc.want || event != tc.event {
				t.Fatalf("old %d mode %d: slot %d event %d, want %d event %d", tc.old, tc.mode, bank[ObjectiveSlotBase+2], event, tc.want, tc.event)
			}
			before[ObjectiveSlotBase+2] = tc.want
			if bank != before {
				t.Fatal("the write reached another slot")
			}
		})
	}
	for _, n := range []int64{-753, 1024 - 752, 1 << 40, -1 << 40} {
		bank := NewGameScenarioBank()
		before := bank
		if event := WriteObjective(&bank, n, 4); event != 0 || bank != before {
			t.Fatalf("objective %d outside the bank wrote %v or raised %d", n, bank != before, event)
		}
	}
	bank := NewGameScenarioBank()
	if WriteObjective(&bank, -752, 4) != ObjectiveEventMode4 || bank[0] != 5 {
		t.Fatal("objective -752 reaches slot 0 as the unguarded store does")
	}
}

// The clearing points: NewGame clears every slot and sets slot 768 to 10; a
// location entry clears 752..767 and nothing else; SetVar writes one slot.
func TestScenarioBankClearingPoints(t *testing.T) {
	fresh := NewGameScenarioBank()
	for i, v := range fresh {
		if want := int32(0); i == 768 {
			want = 10
			if v != want {
				t.Fatalf("NewGame slot 768 = %d", v)
			}
		} else if v != want {
			t.Fatalf("NewGame slot %d = %d", i, v)
		}
	}
	var bank ScenarioBank
	for i := range bank {
		bank[i] = int32(i + 1)
	}
	EnterLocation(&bank)
	for i, v := range bank {
		cleared := i >= 752 && i <= 767
		if cleared && v != 0 || !cleared && v != int32(i+1) {
			t.Fatalf("after EnterLocation slot %d = %d", i, v)
		}
	}
	if !SetScenario(&bank, 760, 7) || bank[760] != 7 || SetScenario(&bank, 1024, 1) || SetScenario(&bank, -1, 1) {
		t.Fatal("SetVar did not write one slot inside the bank only")
	}
}
