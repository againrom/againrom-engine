package game

import (
	"reflect"
	"strconv"
	"testing"

	"againrom/pkg/mapload"
)

func TestForcedShopStaffRetriesAnUnaffordableSpell(t *testing.T) {
	candidate := ordinaryCandidate()
	candidate.Price, candidate.ForcedCast = 167, true
	table := &mapload.Table{Magic: enchantMagic{}, Spells: enchantSpells{1: 50, 13: 1000}}
	draws := &fixedDraws{values: []int{1, 0, 0}}
	item, ok := shopEnchantedItem(candidate, 1000, table, draws)
	spell, power, cast := item.Instance().CastSpell()
	if !ok || !cast || spell != 1 || power != 1 || item.Price != 733 || !reflect.DeepEqual(draws.calls, []int{5, 5, 12}) {
		t.Fatalf("retry: ok=%v cast=%v spell=%d power=%d price=%d draws=%v", ok, cast, spell, power, item.Price, draws.calls)
	}
}

func TestForcedShopStaffDiscardsItsHundredthAttempt(t *testing.T) {
	candidate := ordinaryCandidate()
	candidate.Price, candidate.ForcedCast = 167, true
	table := &mapload.Table{Magic: enchantMagic{}, Spells: enchantSpells{1: 50, 13: 1000}}
	for _, attempt := range []int{99, 100, 101} {
		t.Run(strconv.Itoa(attempt), func(t *testing.T) {
			values := make([]int, attempt+1)
			for i := 0; i < attempt-1; i++ {
				values[i] = 1
			}
			draws := &fixedDraws{values: values}
			item, ok := shopEnchantedItem(candidate, 1000, table, draws)
			if ok != (attempt == 99) {
				t.Fatalf("attempt %d: ok=%v item=%+v", attempt, ok, item)
			}
			wantCalls := min(attempt, 100)
			if attempt <= 100 {
				wantCalls++
			}
			if len(draws.calls) != wantCalls {
				t.Fatalf("attempt %d: draws=%v want %d", attempt, draws.calls, wantCalls)
			}
		})
	}
}
