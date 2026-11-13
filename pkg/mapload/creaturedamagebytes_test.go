package mapload

import (
	"testing"

	"againrom/pkg/formats/alm"
)

// damageRowTable is cheatActorTable with its first creature's damage columns
// and selector set, under the game's creature arithmetic r.
func damageRowTable(r *CreatureDifficulty, min, max, selector int32) *Table {
	t := cheatActorTable()
	t.CreatureDifficulty = r
	t.Units = ghostResistanceCollection{
		{},
		{name: "First Creature", params: cheatActorParams(38, map[int]int32{0: 23, 4: 40, 11: min, 12: max, 13: selector, 29: 64, 30: 2})},
	}
	return t
}

// placedDamage is the damage pair and always-hits mark of the creature a map
// places from table's first row.
func placedDamage(t *testing.T, table *Table) (base, spread int32, alwaysHits bool) {
	t.Helper()
	w, _, err := FromALMRoster(&alm.Map{Width: 40, Height: 40, Units: []alm.Unit{originPlacement()}}, table, DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	e, ok := w.Entity(0)
	if !ok {
		t.Fatal("the placement built no actor")
	}
	return e.DamageBase, e.DamageSpread, e.AlwaysHits
}

// A second-game creature's damage pair is what its row reader stores in two
// bytes and its strike reads back (R2-ENGINE-483, R2-ENGINE-478 step 1): an
// arm-0 pair with a minimum of 128 or more, or a spread above the minimum
// plus 255, keeps whole fifteens, the base at most 127 of them and the spread
// at most 255; any other pair is stored in the bytes as it is; the
// always-hits arm is stored unencoded. The first game keeps the columns.
// Fails while the second game keeps the exact columns.
func TestASecondGameCreatureDealsTheDamageItsRowBytesHold(t *testing.T) {
	for _, tc := range []struct {
		name               string
		min, max, selector int32
		second, first      [2]int32
		alwaysHits         bool
	}{
		{"strong minimum", 200, 300, -1, [2]int32{195, 90}, [2]int32{200, 100}, false},
		{"wide spread", 100, 500, -1, [2]int32{90, 390}, [2]int32{100, 400}, false},
		{"capped", 2000, 6000, 0, [2]int32{1905, 3825}, [2]int32{2000, 4000}, false},
		{"within the bytes", 20, 50, -1, [2]int32{20, 30}, [2]int32{20, 30}, false},
		{"edge of the bytes", 127, 382, -1, [2]int32{127, 255}, [2]int32{127, 255}, false},
		{"spread past a byte under the minimum plus 255", 100, 400, -1, [2]int32{100, 44}, [2]int32{100, 300}, false},
		{"always hits", 20, 50, 3, [2]int32{20, 30}, [2]int32{20, 30}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, game := range []struct {
				name string
				r    *CreatureDifficulty
				want [2]int32
			}{
				{"second game", SecondGameCreatureDifficulty, tc.second},
				{"first game", nil, tc.first},
			} {
				base, spread, always := placedDamage(t, damageRowTable(game.r, tc.min, tc.max, tc.selector))
				if base != game.want[0] || spread != game.want[1] || always != tc.alwaysHits {
					t.Errorf("%s: base %d spread %d always hits %v, want %d %d %v",
						game.name, base, spread, always, game.want[0], game.want[1], tc.alwaysHits)
				}
			}
		})
	}
}
