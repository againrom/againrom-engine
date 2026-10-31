package secondshop

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/random"
)

// fixtureRows is a data.bin collection of n rows with parameters, the raw
// class words admission reads and the class or material doubles.
type fixtureRows struct {
	n       int
	params  map[int][]int32
	raw     map[int][]byte
	doubles map[int][]float64
}

func (c fixtureRows) Len() int                     { return c.n }
func (c fixtureRows) EntryParams(i int) []int32    { return c.params[i] }
func (c fixtureRows) EntryRaw(i int) []byte        { return c.raw[i] }
func (c fixtureRows) EntryDoubles(i int) []float64 { return c.doubles[i] }

// equipmentRow is an equipment row at price, worn on part, with use word use
// and class 0 made of material 1.
func equipmentRow(price, part, use int32) []int32 {
	p := make([]int32, 16)
	p[colPrice], p[colPart], p[colWeaponUse] = price, part, use
	return p
}

// sortTables hold armour rows 1, 3 and 4 and shield rows 1 and 2 of class 0
// and material 1, at the factor 1 and the level 100; admission refuses armour
// row 2 of that class and material. Armour rows 1 and 3 and shield row 1 set
// the fighter bit.
func sortTables() Tables {
	factors := []float64{0, 0, 1, 0, 0, 0, 0, 0, 10}
	word := []byte{1 << 1, 0}
	return Tables{
		Shapes:    fixtureRows{n: 1, doubles: map[int][]float64{0: factors}},
		Materials: fixtureRows{n: 2, doubles: map[int][]float64{1: factors}},
		Armors: fixtureRows{n: 5, params: map[int][]int32{1: equipmentRow(300, 3, 1), 3: equipmentRow(200, 3, 1), 4: equipmentRow(100, 6, 2)},
			raw: map[int][]byte{1: word, 3: word, 4: word}},
		Shields: fixtureRows{n: 3, params: map[int][]int32{1: equipmentRow(50, 2, 1), 2: equipmentRow(400, 2, 2)},
			raw: map[int][]byte{1: word, 2: word}},
	}
}

// A category's list is sorted once the fill ends (R2-ENGINE-710): rows with
// the fighter bit first, then the row ascending, then the price ascending,
// across the armour and shield tables.
func TestSecondShopShelfSortsByTheClaimsKeys(t *testing.T) {
	tb := sortTables()
	r := Record{{Min: 0, Max: 100000, Draws: 200, Bound: 0, Mask: kindArmor | kindShield | 1<<classShift | 1<<1}}
	want := []string{"shield 1 50", "armour 1 300", "armour 3 200", "shield 2 400", "armour 4 100"}
	for seed := uint32(1); seed <= 5; seed++ {
		g := &random.MSVC{State: seed}
		s := tb.Fill(r, func() int { return int(g.Rand()) })
		var got []string
		for _, it := range s[0] {
			name := map[Kind]string{Armor: "armour", Shield: "shield"}[it.Table]
			got = append(got, fmt.Sprintf("%s %d %d", name, it.Row, it.Price))
		}
		if !slices.Equal(got, want) {
			t.Fatalf("seed %d: shelf %q, want %q", seed, got, want)
		}
	}
}
