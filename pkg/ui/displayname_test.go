package ui

import (
	"fmt"
	"testing"
)

// previousPanelSubjectName is the card's former name rule, the oracle.
func previousPanelSubjectName(s PanelSubject) (string, bool) {
	if s.Name != "" {
		return s.Name, true
	}
	if s.Kind != InspectionStructure && s.UnitNameIndex >= 0 && s.UnitNameIndex < len(s.Words.UnitNames) && s.Words.UnitNames[s.UnitNameIndex] != "" {
		return s.Words.UnitNames[s.UnitNameIndex], true
	}
	return "", false
}

// displayNameTables are fixture tables for the two games, some rows empty.
func displayNameTables() map[string]Words {
	var first, second Words
	for i := range first.UnitNames {
		if i%7 == 5 {
			continue
		}
		first.UnitNames[i] = fmt.Sprintf("Class %d", i)
		second.UnitNames[i] = fmt.Sprintf("Класс %d", i)
	}
	second.UnitNames[0] = ""
	return map[string]Words{"first": first, "second": second, "empty": {}}
}

func TestDisplayNameGivesTheCardNameForEachSubjectKind(t *testing.T) {
	kinds := []InspectionKind{InspectionNone, InspectionUnit, InspectionStructure}
	for table, words := range displayNameTables() {
		for _, kind := range kinds {
			for _, own := range []string{"", "Own name"} {
				for index := -2; index <= len(words.UnitNames)+1; index++ {
					s := PanelSubject{Kind: kind, Name: own, UnitNameIndex: index, Words: words}
					wantName, wantOK := previousPanelSubjectName(s)
					gotName, gotOK := DisplayName(kind, own, index, &words)
					if gotName != wantName || gotOK != wantOK {
						t.Fatalf("%s kind %d own %q index %d: DisplayName = %q %t, want %q %t", table, kind, own, index, gotName, gotOK, wantName, wantOK)
					}
					if gotName, gotOK = PanelSubjectName(s); gotName != wantName || gotOK != wantOK {
						t.Fatalf("%s kind %d own %q index %d: PanelSubjectName = %q %t, want %q %t", table, kind, own, index, gotName, gotOK, wantName, wantOK)
					}
				}
			}
		}
	}
}

func TestDisplayNameWithoutAWordTableShowsOnlyTheOwnName(t *testing.T) {
	if name, ok := DisplayName(InspectionUnit, "", 3, nil); ok || name != "" {
		t.Fatalf("no table: %q %t", name, ok)
	}
	if name, ok := DisplayName(InspectionStructure, "Tower", 3, nil); !ok || name != "Tower" {
		t.Fatalf("own name without a table: %q %t", name, ok)
	}
}
