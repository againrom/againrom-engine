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
	table := &Table{Names: data.ItemNames{0x0101: "Shipped"}}
	for _, c := range []struct {
		table           *Table
		code            data.ItemCode
		saved, fallback string
		want            string
	}{
		{table, 0x0101, "Saved", "Composed", "Saved"},
		{table, 0x0101, "", "Composed", "Shipped"},
		{table, 0x0102, "", "Composed", "Composed"},
		{nil, 0x0101, "", "Composed", "Composed"},
	} {
		if got := ItemDisplayName(c.table, c.code, c.saved, c.fallback); got != c.want {
			t.Errorf("ItemDisplayName(%#04x, %q, %q) = %q, want %q", c.code, c.saved, c.fallback, got, c.want)
		}
	}
}

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
