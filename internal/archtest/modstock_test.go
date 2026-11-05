package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckModStockNamesEachShape drives the scan with synthetic sources,
// one planted instance per shape, beside the clean forms and named entries.
func TestCheckModStockNamesEachShape(t *testing.T) {
	const reader = "package game\nfunc modStock(t *mapload.Table, p modStockPolicy) []modStocked {\n" +
		"for _, m := range t.Mods.Items { if m.Stock { secondshop.Stocked(m.Code, m.Price) } }\nreturn nil }\n"
	cases := []struct {
		name, path, src, want string
	}{
		{"a second shop reader", "pkg/game/a.go", "package game\nfunc modShopStock(t *mapload.Table) { for range t.Mods.Items {} }\n", "modShopStock reads the merged mod items"},
		{"the writer", "pkg/game/moditemlayer.go", "package game\nfunc (f *FrontEnd) setModItems() { f.Table.Mods.Items = nil }\n", ""},
		{"the carry", "pkg/game/mods.go", "package game\nfunc (f *FrontEnd) SetMods() { _ = mapload.ModContext{Items: f.Table.Mods.Items} }\n", ""},
		{"the layer lookup", "pkg/mapload/layers.go", "package mapload\nfunc LayerItem(t *Table, code uint16) (ModItem, bool) { return LayerItemIn(t.Mods.Items, code) }\n", ""},
		{"a mod context read", "pkg/game/a.go", "package game\nfunc f(x *FrontEnd) { _ = x.modContext().Items }\n", "f reads the mod context's items"},
		{"the save's mod context read", "pkg/game/currentsave.go", "package game\nfunc (f *FrontEnd) ExportCurrentSave() { _ = f.modContext().Items }\n", ""},
		{"a stock flag on a list item", "pkg/game/a.go", "package game\nfunc f(items []mapload.ModItem) { for _, m := range items { _ = m.Stock } }\n", "f reads a mod item's Stock"},
		{"a price on a mod item", "pkg/game/a.go", "package game\nfunc f(m mapload.ModItem) int32 { return m.Price }\n", "f reads a mod item's Price"},
		{"a stock flag on a layer answer", "pkg/game/a.go", "package game\nfunc f(t *mapload.Table) bool { m, _ := mapload.LayerItem(t, 1); return m.Stock }\n", "f reads a mod item's Stock"},
		{"a stock flag inside mapload", "pkg/mapload/b.go", "package mapload\nfunc f(items []ModItem) { var m ModItem; m = items[0]; _ = m.Stock }\n", "f reads a mod item's Stock"},
		{"another type's stock", "pkg/game/a.go", "package game\nfunc f(w *sim.World, r mod.ItemRow) { _ = w.Stock(); _ = r.Stock; _ = sim.Stock{} }\n", ""},
		{"a classification", "pkg/game/a.go", "package game\nfunc f() { secondshop.Stocked(1, 2) }\n", "f classifies a mod code by secondshop.Stocked"},
		{"a classification inside secondshop", "pkg/secondshop/c.go", "package secondshop\nfunc f() { Stocked(1, 2) }\n", "f classifies a mod code by secondshop.Stocked"},
		{"the first shop", "pkg/game/shopstockprofile.go", "package game\nfunc (s *Shop) stockProfile(t *mapload.Table) { modStock(t, firstModStock(0, 0, 1)) }\n", ""},
		{"the second shop", "pkg/game/secondtownshop.go", "package game\nfunc secondShopTables(t *mapload.Table) { modStock(t, secondModStock) }\n", ""},
		{"a third caller", "pkg/game/a.go", "package game\nfunc f(t *mapload.Table) { modStock(t, secondModStock) }\n", "f calls modStock"},
		{"a second reader in its file", modStockFile, reader + "func g(t *mapload.Table) { for range t.Mods.Items {} }\n", "g reads the merged mod items"},
		{"no reader", modStockFile, "package game\n", "found 0 modStock functions"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{modStockFile: reader}
			files[tc.path] = tc.src
			var got []string
			for _, v := range CheckModStock(files) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckModStock = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckModStock = %v, want one naming %q", got, tc.want)
			}
		})
	}
	if vs := CheckModStock(nil); len(vs) != 1 {
		t.Fatalf("CheckModStock(nil) = %v, want one violation", vs)
	}
}

// TestModStockHasOneReader is the live-tree half.
func TestModStockHasOneReader(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadSackPickupSources(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files[modStockFile]; !ok {
		t.Fatalf("LoadSackPickupSources found no %s among %d files", modStockFile, len(files))
	}
	for _, v := range CheckModStock(files) {
		t.Error(v)
	}
}
