package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/mapload"
)

// emptyGroupCount counts the groups that hold no member.
func emptyGroupCount(groups [][]string) int {
	n := 0
	for _, g := range groups {
		if len(g) == 0 {
			n++
		}
	}
	return n
}

// A town SAVE of a campaign that never LOADed removes a hire group whose
// members all left the party without a refund, as it does after a LOAD: the
// town's Player has participant word zero (SAV-1251). The live town loses the
// group, the file does not hold it, and the cold LOAD's SAVE writes the same
// groups.
func TestNeverLoadedTownSaveDropsAnEmptiedHireGroup(t *testing.T) {
	f, newFront := cityRosterHiredFixture(t, []int{3}, []int{2})
	kept := slices.DeleteFunc(slices.Clone(f.Carried), func(m mapload.PartyMember) bool { return m.MercenaryType == 3 })
	if len(kept) != len(f.Carried)-2 {
		t.Fatalf("the fixture hired %d of type 3, want 2", len(f.Carried)-len(kept))
	}
	// The squad is lost as a mission loses it: no member returns and the
	// mission boundary ends the hire with none of the type alive.
	f.Carried = kept
	f.Town.mercenaryBoundary([16]int{})
	f.Town.settleCityGroups(f.Carried)
	if n := emptyGroupCount(liveGroupIDs(f.Town.cityGroups)); n != 1 {
		t.Fatalf("the lost hires left %d empty groups, want 1", n)
	}
	raw := currentTownSave(t, f)
	if n := emptyGroupCount(liveGroupIDs(f.Town.cityGroups)); n != 0 {
		t.Fatalf("the live town keeps %d empty groups after SAVE", n)
	}
	written := cityRosterOrdinary(t, raw, len(kept))
	if n := emptyGroupCount(written); n != 0 {
		t.Fatalf("the town SAV holds %d empty groups", n)
	}
	cold := cityProjectionLoad(t, raw, newFront)
	if again := cityRosterOrdinary(t, currentTownSave(t, cold), len(kept)); !reflect.DeepEqual(again, written) {
		t.Fatalf("the SAVE after cold LOAD writes groups %v, want %v", again, written)
	}
}
