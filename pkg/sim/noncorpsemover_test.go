package sim

import "testing"

// A torn-down flyer keeps a row that says it leaves no body.
func TestTornDownMoverRowLeavesNoBody(t *testing.T) {
	body := spEnt(3, 10, 11)
	body.Domain, body.HP, body.Decay = DomainAir, noCorpseHP, DecayFallen
	w := spWorld(t, 9, nil, body, spEnt(8, 1, 1))
	Step(w, nil)
	if _, exists := w.Entity(3); exists {
		t.Fatal("fixture did not remove its body")
	}
	if rows := w.CurrentTerminalActors(); len(rows) != 1 || !rows[0].LeavesNoBody() {
		t.Fatal("departed row", rows)
	}
}

func TestLeavesNoBodyOnlyForTheTeardownPin(t *testing.T) {
	for _, tc := range []struct {
		row  CurrentTerminalActor
		want bool
	}{
		{CurrentTerminalActor{HP: -1001, Stage: 1}, true},
		{CurrentTerminalActor{HP: -1000, Stage: 1}, true},
		{CurrentTerminalActor{HP: decayGoneHP - 1, Stage: uint8(decayLast)}, false},
		{CurrentTerminalActor{HP: -30, Stage: uint8(DecayBones)}, false},
		{CurrentTerminalActor{HP: -30, Stage: 1}, false},
	} {
		if tc.row.LeavesNoBody() != tc.want {
			t.Fatal(tc.row, tc.want)
		}
	}
}
