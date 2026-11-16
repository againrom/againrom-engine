package game

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"againrom/pkg/ui"
)

// installItemsPage is the items page of a new mod folder over the install at
// root, as the editor opens it.
func installItemsPage(t *testing.T, root string) (*MapInspector, *mapInspectionEdits) {
	t.Helper()
	x, err := NewMapInspector(root)
	if err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	if err := e.Key("new"); err != nil {
		t.Fatal(err)
	}
	_ = e.Key("clear")
	e.TextInput("40x56")
	for range 2 {
		if err := e.Key("enter"); err != nil {
			t.Fatal(err)
		}
	}
	x.setEditorMod(nil)
	dir := filepath.Join(t.TempDir(), "authored-map")
	d, err := e.Document().Edits.(*mapInspectionEdits).SetWorkspaceField("package", "folder", dir)
	if err != nil || d == nil {
		t.Fatalf("make %s: %v", dir, err)
	}
	return x, d.Edits.(*mapInspectionEdits)
}

// Every install item the items page offers is accepted, in each language of
// install: its name is the game's own, in UTF-8, and no weapon is named by a
// code.
func TestItemsPageOffersEveryInstallItemByTheGamesName(t *testing.T) {
	assets := os.Getenv("AGAINROM_ASSETS")
	if assets == "" {
		t.Skip("no AGAINROM_ASSETS: needs lawful install")
	}
	x, _ := installItemsPage(t, assets)
	items := x.installItems()
	if len(items) == 0 {
		t.Fatal("the install offers no item")
	}
	cyrillic := 0
	for _, it := range items {
		if !utf8.ValidString(it.shown) || it.shown == "" || strings.Contains(it.shown, "0x") || strings.Contains(it.shown, "?") {
			t.Errorf("item %q is shown as %q", it.name, it.shown)
		}
		if strings.IndexFunc(it.shown, unicode.IsUpper) >= 0 && strings.IndexFunc(it.shown, func(r rune) bool { return r > 0x400 && r < 0x500 }) >= 0 {
			cyrillic++
		}
	}
	t.Logf("%s: %d items, %d with Cyrillic names", assets, len(items), cyrillic)
	offered := 0
	for _, key := range []string{itemsNewArmour, itemsNewWeapon} {
		_, page := installItemsPage(t, assets)
		f, ok := itemsField(page, key)
		if !ok {
			t.Fatalf("no %s field", key)
		}
		for _, c := range f.Choices {
			if c.Value == 0 {
				continue
			}
			offered++
			if strings.Contains(c.Label, "?") {
				t.Errorf("%s choice %q shows a question mark", key, c.Label)
			}
			_, page := installItemsPage(t, assets)
			if _, err := page.SetWorkspaceField(ui.ModItemsScope, key, strconv.FormatInt(c.Value, 10)); err != nil {
				t.Errorf("%s choice %q is refused: %v", key, c.Label, err)
			}
		}
	}
	want := 0
	for _, it := range items {
		if it.table == "weapon" || armourBase(it) {
			want++
		}
	}
	if offered != want || offered == 0 {
		t.Fatalf("offered %d choices of %d items", offered, want)
	}
}
