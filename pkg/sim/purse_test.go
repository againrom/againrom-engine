package sim

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestAPurseInTheActorValuesLeaf: the SAV actions leaf writes Purse only when
// it is set, so a save with no purse holds the same JSON as before, and the
// leaf restores a set purse.
func TestAPurseInTheActorValuesLeaf(t *testing.T) {
	e := Entity{ID: 1}
	raw, err := json.Marshal(e.Values())
	if err != nil || bytes.Contains(raw, []byte("Purse")) {
		t.Fatalf("a plain actor's leaf: %s (%v)", raw, err)
	}
	e.Purse = 90
	if raw, err = json.Marshal(e.Values()); err != nil || !bytes.Contains(raw, []byte(`"Purse":90`)) {
		t.Fatalf("a purse's leaf: %s (%v)", raw, err)
	}
	var v ActorValues
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	target := Entity{ID: 1}
	if err := target.restoreValues(v); err != nil || target.Purse != 90 {
		t.Fatalf("leaf restored purse %d (%v), want 90", target.Purse, err)
	}
}

// TestAPurseDropsWithTheUnitsItemsAndOnlyThere: a dying unit's purse joins
// its sack's gold; a unit that leaves nothing drops neither (DIV-4946).
func TestAPurseDropsWithTheUnitsItemsAndOnlyThere(t *testing.T) {
	w := mustStockedWorld(t, 1,
		[]Entity{
			{ID: 1, X: 3, Y: 3, Purse: 40},
			{ID: 2, X: 6, Y: 6, Purse: 25, SuppressCorpseLoot: true},
			{ID: 3, X: 8, Y: 1, Purse: 7},
		},
		[]Stock{{ID: 1, Items: []uint16{0x110}}, {ID: 2, Items: []uint16{0x210}}})

	Step(w, []Command{TerminalKill(1), TerminalKill(2), TerminalKill(3)})

	got := w.Sacks()
	if len(got) != 2 {
		t.Fatalf("Sacks() = %+v, want the two purses' sacks", got)
	}
	for _, s := range got {
		switch {
		case s.X == 3 && s.Y == 3:
			if s.Gold != 40 || !equalCodes(s.Items, []uint16{0x110}) {
				t.Errorf("sack %+v, want gold 40 holding 0x110", s)
			}
		case s.X == 8 && s.Y == 1:
			if s.Gold != 7 || len(s.Items) != 0 {
				t.Errorf("sack %+v, want gold 7 and no item", s)
			}
		default:
			t.Errorf("sack %+v at a cell no purse died on", s)
		}
	}
	for _, e := range w.Entities() {
		if e.Purse != 0 {
			t.Errorf("entity %d keeps purse %d after its death", e.ID, e.Purse)
		}
	}
}

// TestAPurseRidesItsOwnFormSection: a World with no purse keeps its bytes;
// one with a purse writes the purse form and decodes to the same World.
func TestAPurseRidesItsOwnFormSection(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 3, Y: 3}, {ID: 2, X: 6, Y: 6}}, nil)
	plain, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if plain[0] == purseFormVersion {
		t.Fatal("a World with no purse wrote the purse form")
	}
	w.entities[1].Purse = 300
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if form[0] != purseFormVersion {
		t.Fatalf("form version %d, want %d", form[0], purseFormVersion)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if back.entities[1].Purse != 300 || back.entities[0].Purse != 0 {
		t.Errorf("decoded purses %d and %d, want 0 and 300", back.entities[0].Purse, back.entities[1].Purse)
	}
	if back.Hash() != w.Hash() {
		t.Error("the decoded World hashes differently")
	}
}
