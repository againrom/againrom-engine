package game

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// A companion that enters a mission carrying the quest document hands it to
// the starting hero before the mission opens. The mission's object registry
// is seeded from the party after that hand-over, so the mission opens and
// every registry Item row is one the party now carries or wears.
func TestReleaseCompanionQuestDocumentMissionEntry(t *testing.T) {
	f := releaseFront(t)
	hero := MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table)[0]
	companion := MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table)[0]
	companion.ID = "document-companion"
	carries := func(p mapload.PartyMember) bool {
		for _, item := range mapload.MemberCarriedItems(p, f.Table) {
			if item.Code == uint16(data.QuestDocumentCode) {
				return true
			}
		}
		return false
	}
	if !carries(hero) || !carries(companion) {
		t.Fatal("independently built members do not both carry the quest document")
	}
	for _, mission := range []int{10, 20} {
		if err := f.App("companion quest document").OpenMission(f.MissionOpenerWith(mission, []mapload.PartyMember{hero, companion})); err != nil {
			t.Fatalf("mission %d: %v", mission, err)
		}
		documents := make([]uint32, 2)
		var party, rows []string
		for i, id := range f.live.mission.ids {
			stacks, _ := f.live.world.CarriedStacks(id)
			for _, s := range stacks {
				if s.Code == uint16(data.QuestDocumentCode) {
					documents[i] += s.Count
				}
				party = append(party, fmt.Sprintf("%d:%d:%d", s.ObjectID, s.Code, s.Count))
			}
			worn, _ := f.live.world.EquippedItems(id)
			for _, v := range worn {
				if v.ObjectID != 0 {
					party = append(party, fmt.Sprintf("%d:%d:1", v.ObjectID, v.Code))
				}
			}
		}
		if documents[0] != 2 || documents[1] != 0 {
			t.Fatalf("mission %d quest documents hero=%d companion=%d, want 2 and 0", mission, documents[0], documents[1])
		}
		registry := f.live.world.SavedObjects()
		if registry == nil {
			t.Fatalf("mission %d has no object registry", mission)
		}
		for _, row := range registry.Items {
			rows = append(rows, fmt.Sprintf("%d:%d:%d", row.ID, row.Value.Code, row.Value.Count))
		}
		slices.Sort(party)
		slices.Sort(rows)
		if !slices.Equal(party, rows) {
			t.Fatalf("mission %d registry items %v, party carries and wears %v", mission, rows, party)
		}
	}
}
