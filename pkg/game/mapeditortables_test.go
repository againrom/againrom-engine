package game

import (
	"reflect"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/herokind"
	"againrom/pkg/mod"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// The first game's tables hold the values the editor held before they
// became per-install data.
func TestFirstEditorTablesKeepTheirValues(t *testing.T) {
	got := filesOf(base.GameROM1).editorTables
	want := editorTables{
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
	if !reflect.DeepEqual(got, want) {
		t.Errorf("first-game tables %+v, want %+v", got, want)
	}
	if !got.isObjectKind(27) || got.isObjectKind(26) {
		t.Error("the Magic Symbol alone lists in Objects")
	}
	if w, ok := got.heroWord("class", "1"); !ok || w != herokind.Fighter {
		t.Errorf("class choice 1 is %q, %v", w, ok)
	}
}

// The second game's tables hold the palettes' four kinds and nothing the
// research does not name.
func TestSecondEditorTablesHoldOnlyWhatIsNamed(t *testing.T) {
	got := filesOf(base.GameROM2).editorTables
	if len(got.objectKinds)+len(got.handoutBuildings)+len(got.heroChoices)+len(got.itemSuits) != 0 {
		t.Errorf("second-game tables hold rows no claim names: %+v", got)
	}
	if want := []ui.RecordKind{ui.KindEnchantment, ui.KindUnit, ui.KindStructure, ui.KindSack, ui.KindPlayer}; !reflect.DeepEqual(got.addKinds, want) {
		t.Errorf("second-game add kinds %v, want %v", got.addKinds, want)
	}
}
