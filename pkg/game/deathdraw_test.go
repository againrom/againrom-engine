package game

import (
	"fmt"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/sim"
)

type deathDrawRows struct {
	n int
	p map[int][]int32
	r map[int][]byte
	d map[int][]float64
}

func (r deathDrawRows) Len() int                     { return r.n }
func (r deathDrawRows) EntryName(i int) string       { return fmt.Sprint(i) }
func (r deathDrawRows) EntryStrings(int) []string    { return nil }
func (r deathDrawRows) EntryParams(i int) []int32    { return r.p[i] }
func (r deathDrawRows) EntryRaw(i int) []byte        { return r.r[i] }
func (r deathDrawRows) EntryDoubles(i int) []float64 { return r.d[i] }

func deathDrawTable(book, scroll int32, armor bool) *mapload.Table {
	sp := make([]int32, 22)
	sp[21] = book
	magic := map[int][]int32{}
	for i := 1; i <= 80; i++ {
		magic[i] = []int32{scroll}
	}
	empty := deathDrawRows{}
	table := &mapload.Table{Shapes: empty, Materials: empty, Armors: empty, Shields: empty,
		Weapons: empty, MagicItems: deathDrawRows{n: 81, p: magic}, Magic: empty,
		Spells: deathDrawRows{n: 30, p: map[int][]int32{1: sp}}}
	if armor {
		factor := []float64{0, 0, 1, 0, 0, 0, 0, 0, 10}
		table.Shapes = deathDrawRows{n: 1, d: map[int][]float64{0: factor}}
		table.Materials = deathDrawRows{n: 1, d: map[int][]float64{0: factor}}
		p := make([]int32, 16)
		p[2], p[4], p[15] = 300, 3, 1
		table.Armors = deathDrawRows{n: 2, p: map[int][]int32{1: p}, r: map[int][]byte{1: {1, 0}}}
	}
	return table
}

func deathDrawRow(typeID, gold, lo, span, chance, least, most int32, mask uint32) []int32 {
	p := make([]int32, 62)
	for i := range p {
		p[i] = -1
	}
	p[4], p[29] = 10, typeID
	p[38], p[39], p[40] = gold, lo, span
	p[41], p[42], p[43], p[44] = chance, least, most, int32(mask)
	return p
}

func deathDrawWorld(t *testing.T, row []int32, table *mapload.Table) (*sim.World, *int) {
	t.Helper()
	d, err := data.NewUnitDef("creature", row)
	if err != nil {
		t.Fatal(err)
	}
	script, err := sim.NewROM2Script(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	e := sim.NewActor(sim.ActorDefinition{TypeID: d.TypeID, HP: 10, MaxHP: 10,
		GoldChance: d.GoldChance, TreasureMin: d.TreasureMin, TreasureMax: d.TreasureMax,
		DeathItem: sim.DeathItemRoll{Chance: d.ItemChance, Min: d.ItemMin, Max: d.ItemMax, Mask: uint32(d.ItemMask)}},
		sim.ActorPlacement{ID: 1, X: 4, Y: 5})
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: 12, Height: 12}, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{e}, script, sim.Relations{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetRandom(random.Original, 1)
	calls := new(int)
	draw := secondDeathItems(table)
	w.SetDeathItemDraw(func(r sim.DeathItemRoll, rnd func() int) []sim.ItemInstance {
		return draw(r, func() int { *calls++; return rnd() })
	})
	return w, calls
}

func TestSecondDeathDrawEndsAtTheClaimBoundary(t *testing.T) {
	for _, tc := range []struct {
		name  string
		row   []int32
		table *mapload.Table
		calls int
		state uint64
		gold  uint32
		item  uint16
	}{
		{"dragon_books_only", deathDrawRow(71, 100, 3000, 6000, 100, 10, 10, 0x2c0fffff), deathDrawTable(10, 2, false), 29 + 6 + 1001, 1745243601, 6381, 0x0e49},
		{"ogre_failed_clones", deathDrawRow(66, 90, 150, 750, 40, 300, 1500, 0x21c7ffff), deathDrawTable(100000, 100000, true), 1001, 2662887517, 573, 0},
		{"dragon_no_candidate", deathDrawRow(71, 100, 3000, 6000, 100, 10, 10, 0x2c0fffff), deathDrawTable(100, 2, false), 29, 1423890337, 6381, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, calls := deathDrawWorld(t, tc.row, tc.table)
			sim.Step(w, []sim.Command{sim.TerminalKill(1)})
			if *calls != tc.calls || w.RandomState() != tc.state {
				t.Errorf("R2-ENGINE-941: generator calls %d, RNG %d; want %d, %d", *calls, w.RandomState(), tc.calls, tc.state)
			}
			sacks := w.Sacks()
			if len(sacks) != 1 || sacks[0].Gold != tc.gold {
				t.Fatalf("sacks %+v, want one with gold %d", sacks, tc.gold)
			}
			items := sacks[0].ItemInstances
			if tc.item == 0 {
				if len(items) != 0 {
					t.Fatalf("unexpected items %+v", items)
				}
			} else if len(items) != 1 || uint16(items[0].Code) != tc.item || items[0].Price != 2 {
				t.Fatalf("items %+v, want %#04x at price 2", items, tc.item)
			}
		})
	}
}
