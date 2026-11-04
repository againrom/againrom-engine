package ui

import (
	"slices"
	"testing"
)

// The answers below are the string comparisons the editor made before a
// record kind carried its abilities. Each oracle is the old rule, written out
// over words; the test asks it and the kind the same question over every kind
// word and a word no kind has, and the answers must match.

var oldDragRank = map[string]int{"Unit": 1, "Sack": 2, "Structure": 3}

func oldScriptRecord(kind string) bool {
	return kind == "Action" || kind == "Condition" || kind == "Trigger"
}

func oldEditableKind(kind string) bool { return oldDragRank[kind] > 0 || kind == "Stock" }

func oldFieldKind(kind string) bool {
	return oldEditableKind(kind) || oldScriptRecord(kind) || kind == "Player" || kind == "Enchantment"
}

// oldListShows is the records list filter's switch.
func oldListShows(filter int, kind string, spatial bool, warning string) bool {
	show := filter == 0
	switch filter {
	case 1:
		show = kind == "Unit"
	case 2:
		show = kind == "Structure"
	case 3:
		show = kind == "Sack" || kind == "Stock"
	case 4:
		show = kind == "Scenery"
	case 5:
		show = !spatial
	case 6:
		show = warning != ""
	case editorPlayersFilter:
		show = kind == "Player"
	}
	return show
}

// oldBoxKinds is the selection box's kind for a list filter.
func oldBoxKinds(filter int) []string {
	switch filter {
	case 1, editorGroupsFilter:
		return []string{"Unit"}
	case 2:
		return []string{"Structure"}
	case 3:
		return []string{"Sack"}
	}
	return nil
}

var fixtureKindWords = []string{
	"Unit", "Structure", "Sack", "Stock", "Scenery", "Section", "Player", "Trigger",
	"Condition", "Action", "Message", "Script", "Enchantment", "Mod", "Drop", "Bogus", "",
}

func TestRecordKindAnswersMatchTheStringRules(t *testing.T) {
	for _, w := range fixtureKindWords {
		k := RecordKind(w)
		if k.String() != w {
			t.Errorf("%q: word id %q", w, k.String())
		}
		wantRank := oldDragRank[w]
		if k == KindEnchantment {
			wantRank = 4
		}
		if got, want := k.DragRank(), wantRank; got != want {
			t.Errorf("%q: drag rank %d, was %d", w, got, want)
		}
		if got, want := k.Placed(), oldDragRank[w] > 0; got != want {
			t.Errorf("%q: placed %v, was %v", w, got, want)
		}
		if got, want := k.Editable(), oldEditableKind(w); got != want {
			t.Errorf("%q: editable %v, was %v", w, got, want)
		}
		if got, want := k.Fielded(), oldFieldKind(w); got != want {
			t.Errorf("%q: fielded %v, was %v", w, got, want)
		}
		if got, want := k.Scripted(), oldScriptRecord(w); got != want {
			t.Errorf("%q: scripted %v, was %v", w, got, want)
		}
		if got, want := k.Owned(), w == "Unit" || w == "Structure"; got != want {
			t.Errorf("%q: owned %v, was %v", w, got, want)
		}
	}
}

func TestRecordKindListFilterMatchesTheStringRules(t *testing.T) {
	var records []InspectionRecord
	for _, w := range fixtureKindWords {
		for _, spatial := range []bool{false, true} {
			for _, warning := range []string{"", "blocked"} {
				records = append(records, InspectionRecord{Kind: RecordKind(w), Spatial: spatial, Warning: warning})
			}
		}
	}
	e := &MapEditor{doc: &InspectionDocument{Records: records}}
	for filter := 0; filter <= editorGroupsFilter+1; filter++ {
		if filter == editorScriptsFilter {
			continue
		}
		var want []int
		for i, r := range records {
			show := oldListShows(filter, string(r.Kind), r.Spatial, r.Warning)
			if filter == editorCellSpellsFilter {
				show = r.Kind == KindEnchantment && r.Spatial
			}
			if show {
				want = append(want, i)
			}
		}
		if got := e.recordsOf(filter); !slices.Equal(got, want) {
			t.Errorf("filter %d shows %v, was %v", filter, got, want)
		}
	}
}

func TestRecordKindBoxKindsMatchTheStringRules(t *testing.T) {
	for filter := 0; filter <= editorGroupsFilter+1; filter++ {
		e := &MapEditor{doc: &InspectionDocument{}}
		e.filter, e.listShown = filter, true
		var want []RecordKind
		for _, w := range oldBoxKinds(filter) {
			want = append(want, RecordKind(w))
		}
		if got := e.boxKinds(); !slices.Equal(got, want) {
			t.Errorf("filter %d: box kinds %v, was %v", filter, got, want)
		}
	}
}
