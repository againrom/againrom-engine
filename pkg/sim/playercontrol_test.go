package sim

import "testing"

func TestThePlayerUnitTestCountsANeutralUnitWithNeitherSide(t *testing.T) {
	type answer struct{ player, computer bool }
	table := []PlayerParticipant{{PlayerID: 91, Value: 1}, {PlayerID: 103, Value: 0}}
	for _, tc := range []struct {
		name  string
		table bool
		want  map[uint32]answer
	}{
		{"no participants table", false, map[uint32]answer{0: {}, SelfSlot: {player: true}, 7: {computer: true}}},
		{"participants table reverses the slots", true, map[uint32]answer{0: {}, SelfSlot: {computer: true}, 7: {player: true}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := mustWorld(t, 1, Bounds{Width: 8, Height: 8}, []Entity{
				{ID: 1, Owner: 0, X: 1, Y: 1}, {ID: 2, Owner: SelfSlot, X: 2, Y: 1}, {ID: 3, Owner: 7, X: 3, Y: 1}})
			if tc.table {
				players := []SavedGroupPlayer{{ID: 91, Slot: SelfSlot}, {ID: 103, Slot: 7}}
				if err := w.RestoreCurrentPlayers(players, table, true); err != nil {
					t.Fatal(err)
				}
			}
			for _, e := range w.entities {
				got := answer{w.playerControlledUnit(e), w.computerControlledUnit(e)}
				if want := tc.want[e.Owner]; got != want || w.playerControlledSlot(e.Owner) != want.player {
					t.Errorf("owner %d: player %v computer %v slot %v, want %+v",
						e.Owner, got.player, got.computer, w.playerControlledSlot(e.Owner), want)
				}
			}
		})
	}
}
