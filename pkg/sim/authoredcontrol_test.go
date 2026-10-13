package sim

import "testing"

func TestAuthoredControlFarMovementAndColdRestore(t *testing.T) {
	for _, tc := range []struct {
		name    string
		owner   uint32
		control uint32
		present bool
		detour  bool
	}{
		{"human-other-slot", 7, 0, true, true},
		{"computer-other-slot", 7, 1, true, false},
		{"computer-zero-low-byte", 7, 0x100, true, false},
		{"computer-self-slot", SelfSlot, 1, true, false},
		{"legacy-self-slot", SelfSlot, 0, false, true},
		{"legacy-other-slot", 7, 0, false, false},
		{"legacy-unowned", 0, 0, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := mustWorldGrid(t, 1, engBounds, ModeCanonical, farWallGrid(),
				[]Entity{{ID: 1, X: farWallStartX, Y: farWallStartY, Owner: tc.owner}})
			var rows []PlayerParticipant
			if tc.present {
				rows = []PlayerParticipant{{PlayerID: 91, Value: tc.control}}
			}
			if err := w.RestoreCurrentPlayers([]SavedGroupPlayer{{ID: 91, Slot: tc.owner}}, rows, tc.present); err != nil {
				t.Fatal(err)
			}
			raw, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold World
			if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() {
				t.Fatal("control carrier changed on cold restore", err)
			}
			command := []Command{MoveTo(1, CellPoint{X: farWallGoalX, Y: farWallGoalY})}
			Step(w, command)
			Step(&cold, command)
			arrived, west := uint64(0), farWallStartX
			for tick := 0; tick < 160; tick++ {
				if cold.Hash() != w.Hash() {
					t.Fatalf("cold control changed movement at tick %d", tick)
				}
				e := w.entities[0]
				west = min(west, e.X)
				if e.X == farWallGoalX && e.Y == farWallGoalY && arrived == 0 {
					arrived = w.tick
				}
				Step(w, nil)
				Step(&cold, nil)
			}
			if tc.detour {
				if arrived < 30 {
					t.Fatalf("human control did not walk the long detour: arrival tick %d", arrived)
				}
			} else if arrived != 0 || west < 18 {
				t.Fatalf("computer control walked the detour: arrival tick %d, west %d", arrived, west)
			}
		})
	}
}

func TestAuthoredControlAmbiguousSlotKeepsLegacyRule(t *testing.T) {
	for _, owner := range []uint32{SelfSlot, 7} {
		w := mustWorld(t, 1, Bounds{Width: 8, Height: 8}, []Entity{{ID: 1, Owner: owner, X: 1, Y: 1}})
		players := []SavedGroupPlayer{{ID: 91, Slot: owner}, {ID: 103, Slot: owner}}
		for _, rows := range [][]PlayerParticipant{
			{{PlayerID: 91, Value: 0}, {PlayerID: 103, Value: 1}},
			{{PlayerID: 91, Value: 1}, {PlayerID: 103, Value: 0}},
		} {
			if err := w.RestoreCurrentPlayers(players, rows, true); err != nil {
				t.Fatal(err)
			}
			want := staticBudget
			if owner == SelfSlot {
				want = flatBudget
			}
			if got := w.farBudgetFor(0); got != want {
				t.Fatalf("slot %d selected one of two exact Players: budget %d, want %d", owner, got, want)
			}
		}
	}
}

func TestAuthoredControlEscortIdleAndColdRestore(t *testing.T) {
	for _, control := range []uint32{0, 1, 0x100} {
		w := engWorld(t, engRel(t),
			laFighter(1, 7, 7, 5, 5), laFighter(2, 7, 7, 7, 5))
		if err := w.RestoreCurrentPlayers([]SavedGroupPlayer{{ID: 91, Slot: 7}},
			[]PlayerParticipant{{PlayerID: 91, Value: control}}, true); err != nil {
			t.Fatal(err)
		}
		w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
			Unit: 1, HasUnit: true, Args: [scriptParams]int32{subCommandFollow, 3}})
		raw, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var cold World
		if err := cold.UnmarshalBinary(raw); err != nil {
			t.Fatal(err)
		}
		esPass(w)
		esPass(&cold)
		want := escortOrderNone
		if control != 0 {
			want = escortOrderIdle
		}
		if got := laEnt(t, w, 2).EscortOrder; got != want {
			t.Fatalf("control %#x produced escort order %d, want %d", control, got, want)
		}
		if cold.Hash() != w.Hash() {
			t.Fatal("escort control changed after cold restore")
		}
	}
}
