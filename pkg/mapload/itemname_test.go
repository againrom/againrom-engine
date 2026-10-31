package mapload

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/data"
)

func TestItemDisplayNameOrder(t *testing.T) {
	composed := func(*Table, ItemSubject) []string { return []string{"Composed"} }
	table := &Table{Names: data.ItemNames{0x0101: "Shipped", 0x0103: ""}, ItemNamer: composed}
	rows := &ItemRows{Table: ItemRowMagic, Row: 1}
	for _, c := range []struct {
		table *Table
		s     ItemSubject
		want  string
		known bool
	}{
		{table, ItemSubject{Code: 0x0101, Saved: "Saved"}, "Saved", true},
		{table, ItemSubject{Code: 0x0101}, "Shipped", true},
		{table, ItemSubject{Code: 0x0101, Keyless: true}, "Composed", false},
		{table, ItemSubject{Code: 0x0102}, "Composed", false},
		{table, ItemSubject{Code: 0x0102, Form: ItemNameWeapon}, "Composed", false},
		{table, ItemSubject{Code: 0x0103}, "", true},
		{table, ItemSubject{Code: 0x0103, Rows: rows}, "Composed", false},
		{table, ItemSubject{Code: 0x0102, Form: ItemNameEditorList}, "item 0x0102", false},
		{table, ItemSubject{Code: 0x0102, Form: ItemNameEditorRecord}, "unresolved item name", false},
		{nil, ItemSubject{Code: 0x0101}, data.ItemCode(0x0101).Name(), false},
		{nil, ItemSubject{Code: 0x0101, Form: ItemNameWeapon}, "", false},
		{nil, ItemSubject{Code: 0x0101, Rows: rows}, "", false},
	} {
		got := ResolveItemName(c.table, c.s)
		if got.Text() != c.want || got.Known != c.known || ItemDisplayName(c.table, c.s) != c.want {
			t.Errorf("ResolveItemName(%+v) = %q known %v, want %q known %v", c.s, got.Text(), got.Known, c.want, c.known)
		}
	}
}

// TestRowItemNamerNamesTheRows names a second-game book by its spell, a
// magic item by its row and equipment by its row, material and class, one
// line each, and a row past a table's end by nothing.
func TestRowItemNamerNamesTheRows(t *testing.T) {
	table := &Table{Names: data.ItemNames{}, ItemNamer: RowItemNamer,
		Spells: dbRows{"", "Fire Arrow"}, MagicItems: dbRows{"", "Potion"}, Armors: dbRows{"Cloak"},
		Materials: scaleRows{"Iron"}, Shapes: scaleRows{"Common"}}
	for _, c := range []struct {
		rows ItemRows
		want []string
	}{
		{ItemRows{Table: ItemRowBook, Spell: 1}, []string{"Fire Arrow"}},
		{ItemRows{Table: ItemRowMagic, Row: 1}, []string{"Potion"}},
		{ItemRows{Table: ItemRowArmor}, []string{"Cloak", "Iron", "Common"}},
		{ItemRows{Table: ItemRowShield}, []string{"", "Iron", "Common"}},
		{ItemRows{Table: ItemRowArmor, Row: 1, Material: 1, Class: -1}, []string{"", "", ""}},
	} {
		got := ResolveItemName(table, ItemSubject{Keyless: true, Rows: &c.rows}).Lines
		if strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("rows %+v: %q, want %q", c.rows, got, c.want)
		}
	}
}

type dbRows []string

func (c dbRows) Len() int                  { return len(c) }
func (c dbRows) EntryName(i int) string    { return c[i] }
func (c dbRows) EntryParams(int) []int32   { return nil }
func (c dbRows) EntryStrings(int) []string { return nil }

type scaleRows []string

func (c scaleRows) Len() int                   { return len(c) }
func (c scaleRows) EntryName(i int) string     { return c[i] }
func (c scaleRows) EntryDoubles(int) []float64 { return nil }

// TestItemNameOneRoute fails when production code outside this resolver reads
// the shipped item-name table to name an item. The cheat factory's reverse
// lookup, a name to its code, is the one other reader.
func TestItemNameOneRoute(t *testing.T) {
	allowed := map[string]bool{"pkg/mapload/itemname.go": true, "pkg/mapload/cheatfactory.go": true}
	root := filepath.Join("..", "..")
	var found []string
	for _, dir := range []string{"pkg", "cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(raw), ".NameFor(") && !allowed[rel] {
				found = append(found, rel)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(found) > 0 {
		t.Fatalf("item names read outside ItemDisplayName: %v", found)
	}
}
