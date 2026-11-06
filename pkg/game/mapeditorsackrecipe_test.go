package game

import (
	"bytes"
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// sackRecipeMap is the views fixture in lang ("" or "russian") whose unit's
// sack holds item 1 linking recipe R (power 7) and item 2 with no recipe;
// with shared, the ground sack's first item links R too. It returns the
// host, the unit's stock and R.
func sackRecipeMap(t *testing.T, lang string, shared bool) (*mapInspectionEdits, int, int) {
	t.Helper()
	s, _ := viewsMap(t)
	if s.inspector.archives == nil {
		t.Fatal("the fixture has no archives to name the language")
	}
	s.inspector.archives.Base.Profile.Language = lang
	s.inspector.recordOptions = nil
	stock, err := s.model.UnitStock(0)
	if err != nil || stock < 0 {
		t.Fatalf("the unit has no stock: %v", err)
	}
	recipe, err := s.model.AddItemRecipe(stock, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.model.SetField(mapedit.KindEnchantment, recipe, "power", 7); err != nil {
		t.Fatal(err)
	}
	if err := s.model.SetField(mapedit.KindLoot, stock, "additem", 0x101); err != nil {
		t.Fatal(err)
	}
	if shared {
		if err := s.model.SetField(mapedit.KindLoot, groundSack(t, s), "recipe1", int64(recipe+1)); err != nil {
			t.Fatal(err)
		}
	}
	return s, stock, recipe
}

// groundSack is the fixture's ground sack's loot record.
func groundSack(t *testing.T, s *mapInspectionEdits) int {
	t.Helper()
	m, _ := s.model.Map()
	loot, _ := m.Loot()
	for i, r := range loot.Records {
		if r.Owner == 0 {
			return i
		}
	}
	t.Fatal("no ground sack")
	return -1
}

// sackRecipeWindow selects the unit's own stock at 1920x1080 in lang,
// in the plain or the technical view, every plain block open.
func sackRecipeWindow(t *testing.T, s *mapInspectionEdits, lang string, plain bool) (*ui.MapEditor, *editorClicker) {
	t.Helper()
	c := &editorClicker{t: t}
	e := viewsWindow(t, s, plain, ui.KindUnit)
	e.SetWords(words.For(lang))
	e.SetHostTextEncoder(s.inspector.inspectionUTF8)
	e.Layout(1920, 1080)
	openUnitStock(t, e)
	if plain {
		openBlocks(t, e, c)
	}
	return e, c
}

// recipeOf is the recipe record item element of stock links, -1 for none.
func recipeOf(t *testing.T, s *mapInspectionEdits, stock, element int) int {
	t.Helper()
	items := stockItems(t, s, stock)
	if element < 1 || element > len(items) {
		t.Fatalf("stock %d has %d items, want item %d", stock, len(items), element)
	}
	return int(items[element-1].TileMarkerIndex) - 1
}

func recipeBytes(t *testing.T, s *mapInspectionEdits, recipe int) []byte {
	t.Helper()
	b, err := s.model.Record(mapedit.KindEnchantment, recipe)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func recipePower(t *testing.T, s *mapInspectionEdits, recipe int) int64 {
	t.Helper()
	v, err := s.model.FieldValue(mapedit.KindEnchantment, recipe, "power")
	if err != nil {
		t.Fatal(err)
	}
	return v
}

var sackRecipeCases = []struct {
	lang  string
	plain bool
}{{"", true}, {"", false}, {"russian", true}, {"russian", false}}

// Duplicate, chosen on an item of the unit's sack in the sack's window,
// appends a copy of the item whose recipe is its own copy of the
// original's: the bytes the unit's carried-item Duplicate writes.
func TestEditorSackViewDuplicatesAnItemWithItsRecipe(t *testing.T) {
	for _, tc := range sackRecipeCases {
		t.Run(fmt.Sprintf("%s/plain=%v", tc.lang, tc.plain), func(t *testing.T) {
			s, stock, recipe := sackRecipeMap(t, tc.lang, false)
			twin, _, _ := sackRecipeMap(t, tc.lang, false)
			if _, err := twin.SetField(ui.KindUnit, 0, "carry1", -1); err != nil {
				t.Fatal(err)
			}
			e, c := sackRecipeWindow(t, s, tc.lang, tc.plain)
			chooseField(t, e, c, "item1", stockDuplicate)
			items := stockItems(t, s, stock)
			if len(items) != 3 || items[2].ItemCode() != items[0].ItemCode() {
				t.Fatalf("Duplicate gave the items %v", items)
			}
			copied := recipeOf(t, s, stock, 3)
			if copied < 0 || copied == recipe || !bytes.Equal(recipeBytes(t, s, copied), recipeBytes(t, s, recipe)) {
				t.Fatalf("the copy links recipe %d, the original %d", copied, recipe)
			}
			if !bytes.Equal(s.model.Bytes(), twin.model.Bytes()) {
				t.Fatal("the sack's Duplicate wrote other bytes than the carried item's")
			}
		})
	}
}

// A recipe chosen as a copy of an existing one is the item's own: an
// effect edit on it leaves the template and the items linking the
// template as they were.
func TestEditorSackViewCopiesARecipeForOneItem(t *testing.T) {
	for _, tc := range sackRecipeCases {
		t.Run(fmt.Sprintf("%s/plain=%v", tc.lang, tc.plain), func(t *testing.T) {
			s, stock, recipe := sackRecipeMap(t, tc.lang, true)
			template := recipeBytes(t, s, recipe)
			e, c := sackRecipeWindow(t, s, tc.lang, tc.plain)
			chooseField(t, e, c, "recipe2", -int64(recipe+1)-1)
			copied := recipeOf(t, s, stock, 2)
			if copied < 0 || copied == recipe || !bytes.Equal(recipeBytes(t, s, copied), template) {
				t.Fatalf("item 2 links recipe %d, a copy of %d wanted", copied, recipe)
			}
			if tc.plain {
				openBlocks(t, e, c)
			}
			typeField(t, e, c, "recipe2.power", 9)
			if recipeOf(t, s, stock, 2) != copied || recipePower(t, s, copied) != 9 {
				t.Fatalf("the edit reached recipe %d, power %d", recipeOf(t, s, stock, 2), recipePower(t, s, copied))
			}
			ground := groundSack(t, s)
			if !bytes.Equal(recipeBytes(t, s, recipe), template) || recipeOf(t, s, stock, 1) != recipe || recipeOf(t, s, ground, 1) != recipe {
				t.Fatal("the edit of the copy changed the template or another item's link")
			}
		})
	}
}

// An effect edit in the sack's window on an item whose recipe another item
// links copies the recipe for that item first: the other item keeps its
// effects, the bytes are the carried item's copy-on-write edit's, and
// undo and redo are exact.
func TestEditorSackViewEffectEditCopiesASharedRecipe(t *testing.T) {
	for _, tc := range sackRecipeCases {
		t.Run(fmt.Sprintf("%s/plain=%v", tc.lang, tc.plain), func(t *testing.T) {
			s, stock, recipe := sackRecipeMap(t, tc.lang, true)
			raw := bytes.Clone(s.model.Bytes())
			twin, _, _ := sackRecipeMap(t, tc.lang, true)
			if _, err := twin.SetField(ui.KindUnit, 0, "carryrecipe1.power", 9); err != nil {
				t.Fatal(err)
			}
			e, c := sackRecipeWindow(t, s, tc.lang, tc.plain)
			typeField(t, e, c, "recipe1.power", 9)
			own := recipeOf(t, s, stock, 1)
			if own == recipe || recipePower(t, s, own) != 9 || recipePower(t, s, recipe) != 7 || recipeOf(t, s, groundSack(t, s), 1) != recipe {
				t.Fatalf("item 1 links %d (power %d), the shared recipe %d has power %d", own, recipePower(t, s, own), recipe, recipePower(t, s, recipe))
			}
			want := bytes.Clone(s.model.Bytes())
			if !bytes.Equal(want, twin.model.Bytes()) {
				t.Fatal("the sack's effect edit wrote other bytes than the carried item's")
			}
			for _, step := range []struct {
				key  string
				want []byte
			}{{"undo", raw}, {"redo", want}} {
				if err := e.Key(step.key); err != nil || !bytes.Equal(s.model.Bytes(), step.want) {
					t.Fatalf("%s gives other bytes: %v", step.key, err)
				}
			}
		})
	}
}

// A recipe number typed into an item's recipe row of a selected stock and
// entered links that recipe as it stands, as before the copies joined the
// list; 0 unlinks.
func TestEditorSackViewTypedRecipeNumberLinksTheSharedRecipe(t *testing.T) {
	for _, plain := range []bool{true, false} {
		s, stock, recipe := sackRecipeMap(t, "", false)
		e := viewsWindow(t, s, plain, ui.KindStock)
		e.Layout(1920, 1080)
		e.Select(recordIndex(t, e, "Stock", stock), false)
		recordFieldInput(t, e, "recipe2", int64(recipe+1))
		m, _ := s.model.Map()
		if recipeOf(t, s, stock, 2) != recipe || len(m.Enchantments) != recipe+1 {
			t.Fatalf("plain %v: item 2 links %d of %d recipes, the shared %d wanted", plain, recipeOf(t, s, stock, 2), len(m.Enchantments), recipe)
		}
		recordFieldInput(t, e, "recipe2", 0)
		if recipeOf(t, s, stock, 2) != -1 {
			t.Fatalf("plain %v: 0 left item 2 on recipe %d", plain, recipeOf(t, s, stock, 2))
		}
	}
}

// The sack's window lists, in the plain and the technical view, each
// item's Duplicate, its recipe's choices (none, a new recipe, a copy of
// each recipe, each recipe shared) and its recipe's fields beside it.
func TestEditorSackViewListsTheItemRows(t *testing.T) {
	for _, tc := range sackRecipeCases {
		t.Run(fmt.Sprintf("%s/plain=%v", tc.lang, tc.plain), func(t *testing.T) {
			s, _, recipe := sackRecipeMap(t, tc.lang, false)
			e, c := sackRecipeWindow(t, s, tc.lang, tc.plain)
			x := s.inspector
			fields := e.ObjectFields()
			has := func(key string, value int64, label string) {
				t.Helper()
				f := fields[fieldIndex(t, fields, key)]
				if !slices.ContainsFunc(f.Choices, func(c ui.InspectionChoice) bool { return c.Value == value && (label == "" || c.Label == label) }) {
					t.Errorf("%s offers no choice %d %q: %v", key, value, label, f.Choices)
				}
			}
			has("item1", stockDuplicate, x.inspectionUTF8(x.word("editor.inventory.duplicate")))
			has("recipe1", 0, "")
			has("recipe1", -1, x.inspectionUTF8(x.word("editor.equipment.newrecipe")))
			has("recipe1", int64(recipe+1), "")
			copyLabel := x.inspectionUTF8(fmt.Sprintf(x.word("editor.inventory.recipecopy"), fmt.Sprintf("Recipe %d: spell0, 0 effects", recipe+1)))
			has("recipe2", -int64(recipe+1)-1, copyLabel)
			for _, key := range []string{"item1", "recipe1", "recipe1.power", "recipe1.addeffect", "recipe2"} {
				if !showField(t, e, c, key) {
					t.Errorf("%s does not show", key)
				}
			}
			if slices.ContainsFunc(fields, func(f ui.InspectionField) bool { return f.Key == "recipe2.power" }) {
				t.Error("item 2, with no recipe, lists recipe fields")
			}
			if tc.plain {
				blocks := e.PlainFieldBlocks()
				if blocks["recipe1.power"] != "details" {
					t.Errorf("recipe1.power stands in block %q", blocks["recipe1.power"])
				}
			}
		})
	}
}
