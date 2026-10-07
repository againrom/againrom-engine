package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// TestReleaseMissionCityBindsAChapterGrantedCompanionWithNoLiveEntity is the
// regression witness for the mission-return refusal a restored campaign hits
// when its town party carries a member missionCityProvenance's source
// document never described: a companion granted by addChapterCompanions
// straight into f.Carried, after the document was already frozen at mission
// entry. Before the fix, missionCityProvenance refused with "mission city
// source roster is not completely bound" the instant such a member reached
// city.Party; after it, the member is grafted the same way the native branch
// already constructs this exact companion.
//
// It reaches missionCityProvenance directly, the same way
// FinishMissionWithRoster's deferred retainMissionCity does, using a lawful
// install's own registry and Humans rows to build the companion template
// (mapload.CampaignNPCMember) and a genuinely opened mission's own captured
// document -- an engine-constructed fixture, not a saved game.
func TestReleaseMissionCityBindsAChapterGrantedCompanionWithNoLiveEntity(t *testing.T) {
	f := releaseFront(t)
	party := f.ChargenParty(ui.ChargenResult{Name: "Roster", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("roster")
	if err := app.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatalf("open mission 10: %v", err)
	}
	if _, _, err := f.LiveCompleteCampaign(); err != nil {
		t.Fatalf("finish mission 10: %v", err)
	}
	if err := app.OpenMission(f.MissionOpenerWith(20, f.NextParty())); err != nil {
		t.Fatalf("open mission 20: %v", err)
	}
	// Chapter 30 (the campaign's mission-20 successor) is the shipped chapter
	// that grants AddHero 22: exactly the addChapterCompanions call
	// FinishMissionWithRoster makes on a restored campaign's town arrival,
	// before retainMissionCity's own deferred call.
	companion, ok := mapload.CampaignNPCMember(f.Table, 22, 30, f.Carried)
	if !ok {
		t.Fatal("CampaignNPCMember(22, 30): the installed registry has no composed npc22 row")
	}
	city := Snapshot{Party: append(mapload.CloneParty(f.Carried), companion), Difficulty: f.Difficulty}
	document, err := snapshotSavedDocument(f.live.mission.state)
	if err != nil {
		t.Fatalf("snapshotSavedDocument: %v", err)
	}
	captured := Snapshot{SavedDocument: document}

	provenance, err := f.missionCityProvenance(f.townInstall(), city, captured, f.live.world)
	if err != nil {
		t.Fatalf("missionCityProvenance: %v, want the granted companion grafted onto the source document", err)
	}
	if got := len(provenance.Roster()); got != len(city.Party) {
		t.Fatalf("roster has %d characters, want %d (the walked-in hero plus the grafted companion)", got, len(city.Party))
	}
}
