package game

import (
	"strconv"
	"strings"

	"againrom/pkg/herokind"
	"againrom/pkg/mod"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// itemSuit is one "for" choice of an added item row: the suit and the word
// book key of its label.
type itemSuit struct {
	suit int32
	word string
}

// editorTables are the install's tables the map editor's routes read: the
// one value of each kind, held by the install's gameFiles. A table the
// research does not name for a game is empty there.
type editorTables struct {
	// objectKinds are the structure kinds listed in Objects, which their
	// class fields cannot separate from the fixed bridges and floor marks.
	objectKinds []int32
	// handoutBuildings are the offer tables of the building choice, in its
	// order.
	handoutBuildings []town.OfferTable
	// heroChoices are the class and sex fields' choices in order: the word
	// the mission table holds and the word book key of its label. Choice 0
	// names no word, so the row leaves that axis open.
	heroChoices map[string][3][2]string
	// itemSuits are the "for" choices of an added row, by choice value less
	// one.
	itemSuits []itemSuit
	// addKinds are the record kinds the Add routes and palettes assemble.
	addKinds []ui.RecordKind
}

var firstEditorTables = editorTables{
	// The Magic Symbol (structures.reg ID 27).
	objectKinds:      []int32{27},
	handoutBuildings: []town.OfferTable{town.TavernTable, town.ShopTable, town.SchoolTable},
	heroChoices: map[string][3][2]string{
		"class": {{"", "editor.mod.anyhero"}, {herokind.Fighter, "editor.message.hero.fighter"}, {herokind.Mage, "editor.message.hero.mage"}},
		"sex":   {{"", "editor.mod.anyhero"}, {herokind.Male, "editor.unit.look.male"}, {herokind.Female, "editor.unit.look.female"}},
	},
	itemSuits: []itemSuit{{mod.SuitAny, "editor.items.for.any"}, {mod.SuitFighter, "editor.items.for.fighter"}, {mod.SuitMage, "editor.items.for.mage"}},
	addKinds: []ui.RecordKind{ui.KindEnchantment, ui.KindUnit, ui.KindStructure, ui.KindSack, ui.KindPlayer,
		ui.KindTrigger, ui.KindMessage, ui.KindAction, ui.KindCondition},
}

// The second game's maps are viewed only: the palettes list the catalog's
// units, structures and sacks, and the type-9 cell records the format's
// field table admits. No claim names its Objects kinds, mod
// offer tables, hero words or item suits.
var secondEditorTables = editorTables{
	addKinds: []ui.RecordKind{ui.KindEnchantment, ui.KindUnit, ui.KindStructure, ui.KindSack, ui.KindPlayer},
}

// editorTables is the open install's tables.
func (x *MapInspector) editorTables() editorTables {
	return filesOf(x.installGame()).editorTables
}

// isObjectKind reports that structure kind id lists in Objects.
func (t editorTables) isObjectKind(id int32) bool {
	for _, k := range t.objectKinds {
		if k == id {
			return true
		}
	}
	return false
}

// heroWord is the mission table word of key's choice value: the choice
// index, or the word itself.
func (t editorTables) heroWord(key, value string) (string, bool) {
	value = strings.TrimSpace(value)
	for i, c := range t.heroChoices[key] {
		if value == strconv.Itoa(i) || value != "" && value == c[0] {
			return c[0], true
		}
	}
	return "", false
}
