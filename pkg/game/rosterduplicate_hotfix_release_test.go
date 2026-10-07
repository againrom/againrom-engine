package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// An engine build that once restated the entry party into CurrentRoster
// (fixed by resume.go's Snapshot no longer doing so, before this file was
// added) already wrote files whose Start.Roster names a party hero -- an id
// restoreCurrentPartyMembers copies into Start.Roster unfiltered, and a
// healthy Start.Roster never names to begin with. missionAppearanceArt's own
// roster loop had no exclusion for an id that is also a current party
// member, so each such hero drew as the roster's equipment-derived NPC class
// instead of its own body, on a direct LOAD of such a file and again after
// this engine's own SAVE and LOAD of it.
//
// This builds that exact duplicate synthetically: no owner file is read. A
// genuine Snapshot's own CurrentPartyIDs are cloned into CurrentRoster, the
// same restatement the old writer bug made, before asking this engine's own
// current writer to encode the result -- "a corpus file the engine writes in
// the test". It then checks both properties the reader must now have: the
// duplicate-carrying file's own LOAD draws every hero as its own body, and
// saving that loaded session again does not restate the duplicate for a
// later LOAD to repeat.
func TestReleaseHeroDrawsItsOwnBodyFromALegacyDuplicateRosterEntry(t *testing.T) {
	_, saved := groundCorpusFile(t, "2026-09-24/game0017-victory.sav", "a30de01ec3d96d58a3a244b9a536a3e49c04a4d45098cb828dabb908756488c3")

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

	f := releaseFront(t)
	a := f.App("legacy duplicate roster load")
	a.Layout(1024, 768)
	open, _, err := f.RestoreOriginal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.CurrentPartyIDs) == 0 {
		t.Fatal("fixture opened with no party")
	}
	for i, id := range s.CurrentPartyIDs {
		if i >= len(s.Party) {
			break
		}
		s.CurrentRoster[id] = mapload.CloneParty(s.Party[i : i+1])[0]
	}
	duplicated, _, err := f.playerMissionSave(s, label)
	if err != nil {
		t.Fatal(err)
	}

	g := releaseFront(t)
	ag := g.App("legacy duplicate roster reload")
	ag.Layout(1024, 768)
	reopened, _, err := g.RestoreOriginal(duplicated)
	if err != nil {
		t.Fatal(err)
	}
	if err := ag.OpenMission(reopened); err != nil {
		t.Fatal(err)
	}
	partyIDs := append([]sim.EntityID(nil), g.live.mission.ids...)
	for _, id := range partyIDs {
		if !isPartyBody(g.live.units, g.live.art[id]) {
			t.Fatalf("loading a pre-existing duplicate roster entry: party actor %d art %q is not its own body",
				id, releaseArtName(g.live, id))
		}
	}

	s2, label2, err := g.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range s2.CurrentPartyIDs {
		if _, dup := s2.CurrentRoster[id]; dup {
			t.Fatalf("party actor %d is restated into CurrentRoster on resave; the duplicate survives", id)
		}
	}
	resaved, _, err := g.playerMissionSave(s2, label2)
	if err != nil {
		t.Fatal(err)
	}

	h := releaseFront(t)
	ah := h.App("legacy duplicate roster resaved reload")
	ah.Layout(1024, 768)
	reopenedAgain, _, err := h.RestoreOriginal(resaved)
	if err != nil {
		t.Fatal(err)
	}
	if err := ah.OpenMission(reopenedAgain); err != nil {
		t.Fatal(err)
	}
	for _, id := range partyIDs {
		if !isPartyBody(h.live.units, h.live.art[id]) {
			t.Fatalf("after resaving a pre-existing duplicate roster entry: party actor %d art %q is not its own body",
				id, releaseArtName(h.live, id))
		}
	}
}
