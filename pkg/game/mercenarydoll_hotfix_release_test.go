package game

import (
	"fmt"
	"testing"
)

func TestReleaseHiredHumanMercenaryHasAReadOnlyMissionDoll(t *testing.T) {
	for _, typ := range []int{3, 8, 9} {
		t.Run(fmt.Sprint(typ), func(t *testing.T) { releaseHiredMissionDoll(t, typ) })
	}
}

func releaseHiredMissionDoll(t *testing.T, typ int) {
	f := releaseFront(t)
	target := releaseMercenaryChapter(t, f.Campaign.Value())
	town := NewTown(f.Campaign.Value())
	for mission := range f.Campaign.Value().Chapters {
		if mission < target {
			town.Won(mission)
		}
	}
	town.gold = 1_000_000
	f.Town = town
	f.Carried = f.NextParty()
	f.arriveInTown()
	f.Town.mercEnabled[typ] = true
	s := f.TownScreen().(*townScreen)
	s.room = roomTavern
	s.composeShopFaces()

	wantPicture := expectedInstalledMercenaryTalkPicture(t, f, typ)
	if _, ok := s.toggleMercenary(typ); !ok {
		t.Fatalf("hire type %d refused", typ)
	}
	if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpenerWith(target, f.Carried)(); err != nil {
		t.Fatalf("open mission %d with hired type %d: %v", target, typ, err)
	}

	index := -1
	for i, member := range f.live.mission.party {
		if member.MercenaryType == uint8(typ) {
			index = i
			break
		}
	}
	if index < 0 || index >= len(f.live.mission.ids) {
		t.Fatalf("hired type %d is absent from live mission party", typ)
	}
	id := f.live.mission.ids[index]
	member := f.live.mission.party[index]
	dir, face := memberFigure(member)
	if fig, ok := f.live.figures[id]; !ok || fig != (figureID{Dir: dir, Face: face, Horse: typ == 8 || typ == 9}) {
		t.Fatalf("hired entity %d portrait identity = %+v / %v, want %s face %d", id, fig, ok, dir, face)
	}
	if pic := f.live.unitPicture(id, member.Class); pic == nil || !imagesEqual(pic, wantPicture) {
		t.Fatalf("hired entity %d class %d has no composed portrait", id, member.Class)
	}
	wantArt := f.Units.Classes[member.Class]
	var found bool
	for _, draw := range f.live.entityDraws() {
		if draw.ID != uint32(id) {
			continue
		}
		found = true
		if draw.Art != wantArt || draw.Frame == nil {
			t.Fatalf("hired entity %d draw = art %p frame %p, want class %d art %p with a frame",
				id, draw.Art, draw.Frame, member.Class, wantArt)
		}
		break
	}
	if !found {
		t.Fatalf("hired entity %d has no world draw", id)
	}
	primary := f.live.invSubject.ID
	f.live.switchInventorySubject(uint32(id))
	if f.live.invSubject.ID != primary {
		t.Fatalf("hired entity %d became interactive inventory subject %d; doll must stay read-only", id, f.live.invSubject.ID)
	}
}
