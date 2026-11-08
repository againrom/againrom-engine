package sim

import (
	"strings"
	"testing"
)

func TestSecondGamePartialImportKeepsHealthAboveMaximum(t *testing.T) {
	w := guardRouteWorld(t, true, 0)
	if err := w.ImportOriginalActorPools([]OriginalActorPools{{ID: guardRouteFoe, HP: 300, MaxHP: 132}}); err != nil {
		t.Fatal(err)
	}
	e := &w.entities[indexOfEntity(w.entities, guardRouteFoe)]
	if e.HP != 300 || e.MaxHP != 132 {
		t.Fatalf("pools %d/%d, want 300/132", e.HP, e.MaxHP)
	}
}

func TestPartialImportStillRefusesOtherImpossiblePools(t *testing.T) {
	for _, tc := range []struct {
		name   string
		second bool
		pool   OriginalActorPools
	}{
		{"first game health above maximum", false, OriginalActorPools{ID: guardRouteFoe, HP: 300, MaxHP: 132}},
		{"second game mana above maximum", true, OriginalActorPools{ID: guardRouteFoe, HP: 7, MaxHP: 31, Mana: 65535, MaxMana: 101}},
		{"second game negative mana", true, OriginalActorPools{ID: guardRouteFoe, HP: 7, MaxHP: 31, Mana: -1, MaxMana: 101}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := guardRouteWorld(t, tc.second, 0)
			err := w.ImportOriginalActorPools([]OriginalActorPools{tc.pool})
			if err == nil || !strings.Contains(err.Error(), "unsupported pools") {
				t.Fatalf("got %v, want the pool refusal", err)
			}
		})
	}
}
