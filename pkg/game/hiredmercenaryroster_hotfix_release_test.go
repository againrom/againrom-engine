package game

import (
	"fmt"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// A tavern-hired mercenary must keep drawing its own class record, not a
// hero body sheet, after the roster-duplicate reader fix above: it carries
// no hero body directory (canonicalizePartyAppearance skips a Hired()
// member, "he is drawn from his own class record"), so partyArt resolves
// nothing for it and its class comes from the entity's own TypeID, never
// from missionAppearanceArt's Start.Roster loop. That loop's new exclusion
// for an id in ms.Start.IDs is what a mercenary's id already is, being a
// real party member -- the same exclusion that protects a hero also leaves
// a mercenary's own class alone, whether or not its id was ever restated
// into a duplicate Start.Roster entry the way the old writer restated
// every entry-party id, hero and mercenary alike.
func TestReleaseHiredMercenaryKeepsItsOwnClassAfterOurSaveAndLoad(t *testing.T) {
	t.Run("healthy", func(t *testing.T) { releaseHiredMercenaryAfterSaveLoad(t, 3, false) })
	t.Run("legacy duplicate roster", func(t *testing.T) { releaseHiredMercenaryAfterSaveLoad(t, 3, true) })
}

func releaseHiredMercenaryAfterSaveLoad(t *testing.T, typ int, duplicate bool) {
	t.Helper()

	isPartyBody := func(units *terrain.UnitSet, class *terrain.UnitClass) bool {
		if class == nil {
			return false
		}
		for _, body := range units.Bodies {
			if body == class {
				return true
			}
		}
		return false
	}

	findHeroAndMerc := func(party []mapload.PartyMember, ids []sim.EntityID) (heroID, mercID sim.EntityID, merc mapload.PartyMember, heroFound, mercFound bool) {
		for i, member := range party {
			if i >= len(ids) {
				break
			}
			if member.MercenaryType == uint8(typ) && !mercFound {
				mercID, merc, mercFound = ids[i], member, true
				continue
			}
			if !member.Hired() && !heroFound {
				heroID, heroFound = ids[i], true
			}
		}
		return
	}

	mercDrawsOwnClass := func(mw *mapWorld, id sim.EntityID, class int32) bool {
		want := mw.units.Classes[class]
		if want == nil {
			return false
		}
		for _, draw := range mw.entityDraws() {
			if draw.ID == uint32(id) {
				return draw.Art == want
			}
		}
		return false
	}

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
	if _, ok := s.toggleMercenary(typ); !ok {
		t.Fatalf("hire type %d refused", typ)
	}
	if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpenerWith(target, f.Carried)(); err != nil {
		t.Fatalf("open mission %d with hired type %d: %v", target, typ, err)
	}

	heroID, mercID, merc, heroFound, mercFound := findHeroAndMerc(f.live.mission.party, f.live.mission.ids)
	if !heroFound || !mercFound {
		t.Fatalf("mission %d did not carry both a hero and hired type %d", target, typ)
	}
	if !isPartyBody(f.live.units, f.live.art[heroID]) {
		t.Fatalf("hero %d does not draw its own body before saving", heroID)
	}
	if isPartyBody(f.live.units, f.live.art[mercID]) {
		t.Fatalf("hired merc %d draws a hero body before saving", mercID)
	}
	if !mercDrawsOwnClass(f.live, mercID, merc.Class) {
		t.Fatalf("hired merc %d does not draw its own class %d before saving", mercID, merc.Class)
	}

	snap, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate {
		// The old writer, before the fix that stopped restating an entry-party
		// id into CurrentRoster, restated EVERY entry-party id unconditionally,
		// a hired mercenary's along with a hero's.
		for i, id := range snap.CurrentPartyIDs {
			if i >= len(snap.Party) {
				break
			}
			if id == heroID || id == mercID {
				snap.CurrentRoster[id] = mapload.CloneParty(snap.Party[i : i+1])[0]
			}
		}
	}
	saved, _, err := f.playerMissionSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}

	g := releaseFront(t)
	ag := g.App("hired mercenary reload")
	ag.Layout(1024, 768)
	reopened, _, err := g.RestoreOriginal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if err := ag.OpenMission(reopened); err != nil {
		t.Fatal(err)
	}

	reHeroID, reMercID, reMerc, heroFound, mercFound := findHeroAndMerc(g.live.mission.party, g.live.mission.ids)
	if !heroFound || !mercFound {
		t.Fatalf("reload did not carry both a hero and hired type %d", typ)
	}
	if !isPartyBody(g.live.units, g.live.art[reHeroID]) {
		t.Fatalf("after reload, hero %d does not draw its own body", reHeroID)
	}

	if isPartyBody(g.live.units, g.live.art[reMercID]) {
		t.Fatalf("after reload, hired merc %d draws a hero body", reMercID)
	}
	if !mercDrawsOwnClass(g.live, reMercID, reMerc.Class) {
		t.Fatalf("after reload, hired merc %d does not draw its own class %d", reMercID, reMerc.Class)
	}
	if !reMerc.Hired() || reMerc.MercenaryType != uint8(typ) {
		t.Fatalf("after reload, hired merc %d MercenaryType = %d Hired()=%v, want %d/true",
			reMercID, reMerc.MercenaryType, reMerc.Hired(), typ)
	}
	if reMerc.PlayerCharacter {
		t.Fatalf("after reload, hired merc %d is PlayerCharacter", reMercID)
	}
}

func TestReleaseHiredHorsemanKeepsItsOwnClassAfterOurSaveAndLoad(t *testing.T) {
	for _, typ := range []int{8, 9} {
		t.Run(fmt.Sprint(typ), func(t *testing.T) { releaseHiredMercenaryAfterSaveLoad(t, typ, false) })
	}
}
