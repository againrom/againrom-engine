package game

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// A companion that enters a mission carrying the quest document still holds
// it once the mission opens: mission start moves no item between party
// members (PARTY-037, PARTY-038). The mission opens and every registry Item
// row is one the party carries or wears.
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
	entry := func(p mapload.PartyMember) []uint16 {
		var codes []uint16
		for _, item := range mapload.MemberCarriedItems(p, f.Table) {
			codes = append(codes, item.Code)
		}
		slices.Sort(codes)
		return codes
	}
	before := [][]uint16{entry(hero), entry(companion)}
	for _, mission := range []int{10, 20} {
		if err := f.App("companion quest document").OpenMission(f.MissionOpenerWith(mission, []mapload.PartyMember{hero, companion})); err != nil {
			t.Fatalf("mission %d: %v", mission, err)
		}
		documents := make([]uint32, 2)
		var party, rows []string
		for i, id := range f.live.mission.ids {
			stacks, _ := f.live.world.CarriedStacks(id)
			var after []uint16
			for _, s := range stacks {
				if s.Code == uint16(data.QuestDocumentCode) {
					documents[i] += s.Count
				}
				for n := uint32(0); n < s.Count; n++ {
					after = append(after, s.Code)
				}
				party = append(party, fmt.Sprintf("%d:%d:%d", s.ObjectID, s.Code, s.Count))
			}
			slices.Sort(after)
			if !slices.Equal(after, before[i]) {
				t.Fatalf("mission %d member %d carries %#04x after the start, entered with %#04x", mission, i, after, before[i])
			}
			t.Logf("mission %d member %d carried before %#04x after %#04x", mission, i, before[i], after)
			worn, _ := f.live.world.EquippedItems(id)
			for _, v := range worn {
				if v.ObjectID != 0 {
					party = append(party, fmt.Sprintf("%d:%d:1", v.ObjectID, v.Code))
				}
			}
		}
		if documents[0] != 1 || documents[1] != 1 {
			t.Fatalf("mission %d quest documents hero=%d companion=%d, want 1 and 1", mission, documents[0], documents[1])
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
		snapshot, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatalf("mission %d snapshot: %v", mission, err)
		}
		raw, err := f.ExportCurrentSave(snapshot, label)
		if err != nil {
			t.Fatalf("mission %d SAV: %v", mission, err)
		}
		t.Logf("mission %d World hash %#016x SAV %d bytes sha256 %x", mission, f.live.world.Hash(), len(raw), sha256.Sum256(raw))
	}
}
