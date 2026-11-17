package game

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// unitWindowFixture is the views map's unit window in mode plain, in the
// language lang names, at 1920x1080.
func unitWindowFixture(t *testing.T, lang string, plain bool) (*mapInspectionEdits, *ui.MapEditor) {
	t.Helper()
	s, _ := viewsMap(t)
	s.inspector.archives.Base.Profile.Language = lang
	s.inspector.recordOptions = nil
	e := viewsWindow(t, s, plain, ui.KindUnit)
	e.SetWords(words.For(lang))
	e.SetHostTextEncoder(s.inspector.inspectionUTF8)
	e.Layout(1920, 1080)
	return s, e
}

// The unit's bag lists its gold first, then its loot (what it leaves and
// what the corpse drops), then the items it carries: in the fields the
// game's host gives and in the detailed mode's rows. The normal mode shows
// no bag. In English and Russian.
func TestEditorUnitBagListsGoldThenLootThenItems(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		for _, plain := range []bool{false, true} {
			s, e := unitWindowFixture(t, lang, plain)
			x := s.inspector
			fields := e.ObjectFields()
			var bag []string
			for _, f := range fields {
				if f.Section == "inventory" && f.Parent == "" {
					bag = append(bag, f.Key)
				}
			}
			rank := func(key string) int {
				switch {
				case key == unitGoldField:
					return 0
				case key == "noloot" || key == "loot":
					return 1
				}
				return 2
			}
			if len(bag) < 4 || bag[0] != unitGoldField || !slices.IsSortedFunc(bag, func(a, b string) int { return rank(a) - rank(b) }) {
				t.Errorf("%q plain %v: the bag's fields stand %v, want gold, loot, then items", lang, plain, bag)
			}
			gold, noloot, add := fieldIndex(t, fields, unitGoldField), fieldIndex(t, fields, "noloot"), fieldIndex(t, fields, "carryadd")
			if plain {
				for _, i := range []int{gold, noloot, add} {
					if _, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: i}); ok {
						t.Errorf("%q: the normal mode shows the bag's %s", lang, fields[i].Key)
					}
				}
				continue
			}
			lines := e.PopupLines()
			row := func(i int) int {
				label := x.inspectionUTF8(fields[i].Label) + " "
				return slices.IndexFunc(lines, func(line string) bool { return strings.HasPrefix(line, label) })
			}
			loot := slices.Index(lines, fields[fieldIndex(t, fields, "loot")].Lines[0].Text)
			if at := []int{row(gold), row(noloot), loot, row(add)}; at[0] < 0 || !slices.IsSorted(at) {
				t.Errorf("%q: the bag's rows gold, loot, corpse drops, carried item stand at %v", lang, at)
			}
		}
	}
}

// animalKept are the fields the generic object window let a map maker edit
// for a creature placed on the units arm.
var animalKept = []string{
	"identity", "owner", "group", "facing", "name", "hp", "hpmax", "mana",
	"body", "mind", "spirit", "reaction", "defence", "absorption", "speed", "sight",
	"skill.blade", "skill.axe", "skill.bludgeon", "skill.pike", "skill.shooting",
	"protection.fire", "protection.water", "protection.air", "protection.earth", "protection.astral",
	"noloot",
}

// unitSectionLayout is the window's sections in the order its shown fields
// stand, each once; it fails the test for a shown field of no section.
func unitSectionLayout(t *testing.T, e *ui.MapEditor) []string {
	t.Helper()
	var out []string
	for i, f := range e.ObjectFields() {
		if _, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: i}); !ok {
			continue
		}
		if f.Section == "" {
			t.Errorf("field %s stands in no section of the unit window", f.Key)
		}
		if !slices.Contains(out, f.Section) {
			out = append(out, f.Section)
		}
	}
	return out
}

// A creature opens the unit window a person opens: the view switch, every
// field in a section, its sections in the person's order less the ones it
// has no data for, and every field the generic window edited for it shown
// and editable.
func TestEditorAnimalOpensThePersonsUnitWindow(t *testing.T) {
	_, person := unitWindowFixture(t, "", false)
	_, animal := openAuthoring(t)
	animal.Layout(1920, 1080)
	animal.Select(1, false)
	c := &editorClicker{t: t}
	for name, e := range map[string]*ui.MapEditor{"person": person, "animal": animal} {
		if _, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlViewSwitch}); !ok {
			t.Errorf("the %s's window offers no normal and detailed mode", name)
		}
	}
	human, creature := unitSectionLayout(t, person), unitSectionLayout(t, animal)
	if !slices.Contains(human, "equipment") || !slices.Contains(creature, "identity") {
		t.Fatalf("the person's sections %v, the creature's %v", human, creature)
	}
	at := 0
	for _, section := range creature {
		for at < len(human) && human[at] != section {
			at++
		}
		if at == len(human) {
			t.Fatalf("the creature's sections %v are not the person's %v, in order", creature, human)
		}
	}
	fields := animal.ObjectFields()
	for _, key := range animalKept {
		f := fields[fieldIndex(t, fields, key)]
		if f.ReadOnly || f.Section == "" || !showField(t, animal, c, key) {
			t.Errorf("the creature's %s is not editable in the unit window (read-only %v, section %q)", key, f.ReadOnly, f.Section)
		}
	}
}
