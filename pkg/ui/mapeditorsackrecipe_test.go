package ui

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/words"
)

// recipeSackHost is the unit's-sack fixture whose stock's first item links
// a recipe: the host lists the recipe's fields after the item's recipe
// row, keyed recipe1.<field>, and offers Duplicate and the recipe copies.
type recipeSackHost struct {
	*sackHost
}

func (h *recipeSackHost) Fields(kind RecordKind, index int) []InspectionField {
	out := h.sackHost.Fields(kind, index)
	if kind != KindStock {
		return out
	}
	var listed []InspectionField
	for _, f := range out {
		switch f.Key {
		case "item1", "item2":
			f.Choices = append([]InspectionChoice{{0, "Remove this item"}, {-1, "Duplicate item"}}, f.Choices...)
		case "recipe1", "recipe2":
			f.Choices = []InspectionChoice{{0, "No recipe (plain item)"}, {-1, "New private recipe"}, {-2, "Copy for this item: Recipe 1"}, {1, "Recipe 1"}}
		}
		listed = append(listed, f)
		if f.Key == "recipe1" {
			listed = append(listed,
				InspectionField{Key: "recipe1.power", Label: "Item 1 / Spell power", Parent: "item1", Value: 7},
				InspectionField{Key: "recipe1.addeffect", Label: "Item 1 / Add effect kind (1..255)", Parent: "item1"})
		}
	}
	return listed
}

// recipeSackEditor is sackEditor over a recipeSackHost, the stock selected.
func recipeSackEditor(t *testing.T, lang, view string) *MapEditor {
	t.Helper()
	inner, _ := viewsEditor(t, lang)
	h := &recipeSackHost{&sackHost{wideHost: &wideHost{viewsHost: inner.doc.Edits.(*viewsHost)}}}
	h.values["Stock/gold"], h.values["Stock/item1"], h.values["Stock/item2"], h.values["Stock/recipe1"] = 12, 0x101, 0x102, 1
	document := func(string) (*InspectionDocument, error) {
		d := sackDocument(h.sackHost)
		d.Edits = h
		return d, nil
	}
	e := NewMapEditor(wordsFont(lang), []EditorMap{{Key: "views", Label: "Views"}}, document)
	e.SetWords(words.For(lang))
	e.SetHostTextEncoder(func(s string) string {
		encoded, _ := textinput.EncodeText(s, e.font.Selector, textinput.Substitute)
		return encoded
	})
	e.SetPanelProfile([]byte(fmt.Sprintf(`{"left":{"minimized":false,"width":260},"right":{"minimized":false,"width":520,"view":%q}}`, view)), nil, nil)
	e.Layout(1920, 1080)
	if err := e.Open("views"); err != nil {
		t.Fatal(err)
	}
	e.Select(sackStock, false)
	return e
}

// The sack's window lists an item's recipe fields beside the item's
// recipe row, in the plain view's details block and in the technical
// view, and its item and recipe rows offer Duplicate and the copies.
func TestMapEditorSackViewListsTheRecipeRows(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, view := range []string{"plain", "technical"} {
			t.Run(lang+"/"+view, func(t *testing.T) {
				e := recipeSackEditor(t, lang, view)
				if view == "plain" {
					e.toggleBlock("details")
					if got := e.PlainFieldBlocks()["recipe1.power"]; got != "details" {
						t.Fatalf("recipe1.power stands in block %q", got)
					}
				}
				if _, _, ok := e.objectSpec(); !ok {
					t.Fatal("the stock shows no window")
				}
				row := func(key string) int {
					return e.fieldRow(slices.IndexFunc(e.obj.fields, func(f InspectionField) bool { return f.Key == key }))
				}
				own, power, effect, next := row("recipe1"), row("recipe1.power"), row("recipe1.addeffect"), row("recipe2")
				if own < 0 || !(own < power && power < effect && effect < next) {
					t.Fatalf("rows recipe1 %d, power %d, add effect %d, recipe2 %d:\n%s", own, power, effect, next, strings.Join(e.PopupLines(), "\n"))
				}
				for _, key := range []string{"item1", "recipe1"} {
					i := slices.IndexFunc(e.obj.fields, func(f InspectionField) bool { return f.Key == key })
					rows := choiceRows(e.obj.fields[i])
					if !slices.ContainsFunc(rows, func(r PickerRow) bool { return r.Text == "Duplicate item" || r.Text == "Copy for this item: Recipe 1" }) {
						t.Errorf("%s lists %v", key, rows)
					}
				}
			})
		}
	}
}
