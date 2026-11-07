package game

import (
	"os"
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/random"
	"againrom/pkg/secondshop"
)

// secondDropGroup is one row group of R2-ENGINE-942's table: the Units row
// names it holds, its item chance and its mask, and its row count.
type secondDropGroup struct {
	names  []string
	chance int32
	mask   uint32
	rows   int
}

// secondDropGroups is R2-ENGINE-942's table of the 66 Units rows that reach
// the death item roll. A name ending in an underscore is a prefix.
var secondDropGroups = []secondDropGroup{
	{[]string{"Orc_Sword", "Orc_Bow"}, 30, 0x3147ffff, 10},
	{[]string{"Ogre"}, 40, 0x21c7ffff, 5},
	{[]string{"Dragon"}, 100, 0x2c0fffff, 5},
	{[]string{"Urd"}, 100, 0, 1},
	{[]string{"Orc_Shaman"}, 30, 0x23c7ffff, 5},
	{[]string{"F_Zombie", "A_Zombie", "F_Skeleton", "A_Skeleton"}, 30, 0x31ffffff, 19},
	{[]string{"M_Skeleton"}, 30, 0x2e2e73f8, 3},
	{[]string{"Orc_ShamanLeader"}, 30, 0x64, 1},
	{[]string{"Necro_", "GoodNecroLeader", "GoodNecro"}, 30, 0x2e7fffff, 17},
}

// secondDropMagic is the mask's magic bit, 26 (R2-ENGINE-941).
const secondDropMagic = 1 << 26

// secondDropGroupOf is the group whose names hold row name n: its stem
// before a level suffix, or a prefix.
func secondDropGroupOf(n string) int {
	stem, _, _ := strings.Cut(n, ".")
	for g, group := range secondDropGroups {
		for _, name := range group.names {
			if stem == name || strings.HasSuffix(name, "_") && strings.HasPrefix(stem, name) {
				return g
			}
		}
	}
	return -1
}

// The installed second game's Units rows, read by the engine's one Units
// reader, hold R2-ENGINE-942's death item columns: 66 rows reach the roll,
// each with its group's chance and mask, none below type 64, and the table
// gives the draw. On the installed shop collections each row with the magic
// bit admits a candidate, each Dragon row's draw lists only books, scrolls,
// potions and staves, and each Ogre row's only enchanted weapons, shields
// and armour.
func TestReleaseSecondDeathDropUnitsRows(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the Units rows need a lawful install")
	}
	arch, err := OpenArchives(root)
	if err != nil {
		t.Fatal(err)
	}
	if arch.Game() != base.GameROM2 {
		t.Skip("the root holds the first game")
	}
	defs, err := LoadDefinitionsFor(arch.Containers, arch.Game())
	if err != nil {
		t.Fatal(err)
	}
	units := defs.Table.Units
	if defs.Table.DeathItems == nil {
		t.Fatal("the second game's table gives no death item draw")
	}
	tables, ok := secondShopTables(defs.Table)
	if !ok {
		t.Fatal("the install holds no shop collections")
	}
	counts := make([]int, len(secondDropGroups))
	total := 0
	for row := range units.Len() {
		d, err := data.NewUnitDef(units.EntryName(row), units.EntryParams(row))
		if err != nil || d.ItemChance <= 0 {
			continue
		}
		total++
		name := units.EntryName(row)
		t.Logf("row\t%s\t%d\t%#08x\t%d\t%d", name, d.ItemChance, uint32(d.ItemMask), d.ItemMin, d.ItemMax)
		g := secondDropGroupOf(name)
		if g < 0 || d.TypeID < 64 {
			t.Errorf("row %d %s: type %d, in no group of the claim", row, name, d.TypeID)
			continue
		}
		counts[g]++
		if want := secondDropGroups[g]; d.ItemChance != want.chance || uint32(d.ItemMask) != want.mask {
			t.Errorf("row %d %s: chance %d mask %#x, want %d %#x", row, name, d.ItemChance, uint32(d.ItemMask), want.chance, want.mask)
		}
		dragon, ogre := g == 2, g == 1
		pick := &random.MSVC{State: 1}
		cands := tables.Candidates(secondshop.Category{Min: d.ItemMin, Max: d.ItemMax, Mask: uint32(d.ItemMask)}, func() int { return int(pick.Rand()) })
		t.Logf("candidates\t%s\t%d", name, len(cands))
		if uint32(d.ItemMask)&secondDropMagic != 0 && len(cands) == 0 {
			t.Errorf("%s: no candidate, the list would be the six potions alone", name)
		}
		if !dragon && !ogre {
			continue
		}
		for seed := uint32(1); seed <= 20; seed++ {
			r := &random.MSVC{State: seed}
			list := tables.DeathItems(d.ItemMin, d.ItemMax, uint32(d.ItemMask), func() int { return int(r.Rand()) })
			if dragon && len(list) == 0 {
				t.Errorf("%s seed %d: the dragon's list is empty", name, seed)
			}
			for _, it := range list {
				part := it.Code >> 8 & 0xf
				if dragon && part != 0xe && part != 1 || ogre && (part == 0xe || len(it.Effects) == 0) {
					t.Errorf("%s seed %d: %#04x with %d effects is not of an admitted kind", name, seed, it.Code, len(it.Effects))
				}
			}
		}
	}
	if total != 66 {
		t.Errorf("%d rows reach the roll, want 66", total)
	}
	for g, group := range secondDropGroups {
		if counts[g] != group.rows {
			t.Errorf("group %v: %d rows, want %d", group.names, counts[g], group.rows)
		}
	}
}
