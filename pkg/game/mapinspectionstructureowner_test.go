package game

import (
	"testing"

	"againrom/internal/synth"
)

// A structure record of the editor's document carries its owner roster slot,
// the low word of the placement's +0x0e field, the way a unit record carries
// its owner: the player colours tint a structure by it.
func TestInspectionStructureRecordsCarryTheirOwnerSlot(t *testing.T) {
	b := synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12), Objects: []synth.ALMObject{
		{X: 5 << 8, Y: 5 << 8, Kind: 1, Field0E: 0x00070003},
		{X: 9 << 8, Y: 5 << 8, Kind: 1},
	}, Units: []synth.ALMUnit{{X: 12 << 8, Y: 12 << 8, ClassID: 300}}})
	e, _ := openGoalMap(t, b)
	owners := map[string][]int{}
	for _, r := range e.Document().Records {
		if r.Kind == "Structure" || r.Kind == "Unit" {
			owners[r.Kind] = append(owners[r.Kind], r.Player)
		}
	}
	if got := owners["Structure"]; len(got) != 2 || got[0] != 3 || got[1] != 0 {
		t.Fatalf("structure owners %v, want the low word 3 and none", got)
	}
	if got := owners["Unit"]; len(got) != 1 {
		t.Fatalf("unit owners %v", got)
	}
}
