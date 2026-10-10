package game

import (
	"testing"

	"againrom/pkg/data"
)

// Every object of every generator description that states a value carries a
// cite, and every cite is a claim, a ledger row or "owner".
func TestGeneratorDescriptionsEveryValueIsCited(t *testing.T) {
	if len(generatorJSON) == 0 {
		t.Fatal("no generator description")
	}
	for name, data := range generatorJSON {
		t.Run(name, func(t *testing.T) { checkDescriptionCites(t, data) })
	}
}

// The first game's description states the point-buy the hero model computes:
// the cost curve at every value, the bounds, the start and the budget.
func TestFirstGeneratorStatesThePointBuy(t *testing.T) {
	st := generatorDescriptions["rom1"].Detail.Stats
	if st.Floor != int(data.StatFloor) || st.Ceiling != int(data.StatCeiling) || st.Start != int(data.ChargenStat) || st.Budget != int(data.ChargenBudget) {
		t.Fatalf("bounds %d..%d, start %d, budget %d; want %d..%d, %d, %d", st.Floor, st.Ceiling, st.Start, st.Budget,
			data.StatFloor, data.StatCeiling, data.ChargenStat, data.ChargenBudget)
	}
	for v := 0; v <= st.Ceiling+1; v++ {
		if got, want := generatorCost(st.Cost, v), int(data.PointCost(int32(v))); got != want {
			t.Errorf("cost at %d = %d, want %d", v, got, want)
		}
	}
}
