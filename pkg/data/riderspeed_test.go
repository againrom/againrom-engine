package data_test

import (
	"testing"

	"againrom/pkg/data"
)

// Rider type ids add 10 to speed.
func TestRiderTypeIDsTakeTheSpeedBonus(t *testing.T) {
	hero := data.Hero{Body: 30, Reaction: 30}
	for _, tc := range []struct {
		typeID int32
		want   int32
	}{{19, 28}, {21, 28}, {7, 18}, {12, 18}, {33, 18}, {0, 18}} {
		def := data.HumanDef{TypeID: tc.typeID, Body: 30, Reaction: 30}
		if got := hero.Recompute(def.Profile(), data.Loadout{}).Speed; got != tc.want {
			t.Errorf("typeID %d: speed %d, want %d", tc.typeID, got, tc.want)
		}
	}
	low := data.Hero{Body: 30, Reaction: 8}
	if got := low.Recompute(data.Profile{Rider: true}, data.Loadout{}).Speed; got != 18 {
		t.Errorf("rider below the branch: speed %d, want 18", got)
	}
	if got := hero.Recompute(data.Profile{}, data.Loadout{}).Speed; got != 18 {
		t.Errorf("empty profile: speed %d, want 18", got)
	}
}
