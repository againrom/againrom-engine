package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// 0159: what the front end's mission boundary does with a companion a script
// handed to the player.

// joinedMission is continuityMission's shape with a SECOND actor standing under
// the player, carrying a band type id and a roster template of his own — an
// actor that joined during the mission. The first actor is the party's own
// member, exactly as continuityMission builds him.
func joinedMission(t *testing.T, n int, op int32) *Mission {
	t.Helper()

	s, err := sim.NewScript(missionChecks(),
		[]sim.ScriptInstant{{Op: op}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	ents := []sim.Entity{
		{ID: 0, X: 10, Y: 10, HP: 20, MaxHP: 20, Owner: sim.SelfSlot,
			Domain: sim.DomainGround, TypeID: sim.HumanTypeID,
			SkillXP: continuityXP, GainsXP: true},
		{ID: 1, X: 12, Y: 10, HP: 15, MaxHP: 15, Owner: sim.SelfSlot,
			Domain: sim.DomainGround, TypeID: sim.HumanTypeID, GainsXP: true},
	}
	w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, make([]byte, worldFixtureW*worldFixtureH), ents, s)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	for i := 0; i < 64 && w.Outcome() == sim.OutcomeUndecided; i++ {
		sim.Step(w, nil)
	}
	return &Mission{
		Number: n, World: w,
		Party: []mapload.PartyMember{{ID: "hero", StartingHero: true, Class: 100}},
		Start: mapload.Start{
			IDs: []sim.EntityID{0},
			Roster: map[sim.EntityID]mapload.PartyMember{
				1: {ID: "join:1", Name: "companion", PlayerCharacter: true, Class: 101},
			},
		},
	}
}

func TestAWonMissionCarriesTheJoinedCompanion(t *testing.T) {
	f := continuityFront(t)
	f.continuity(20, joinedMission(t, 20, sim.ScriptInstantWin), listAdvance)()

	if len(f.Carried) != 2 {
		t.Fatalf("the front end carried %d members, want the hero and the companion", len(f.Carried))
	}
	if f.Carried[0].ID != "hero" {
		t.Errorf("the first carried member is %q, want the hero who walked in", f.Carried[0].ID)
	}
	if f.Carried[1].ID != "join:1" {
		t.Errorf("the second carried member is %q, want the companion's own identity",
			f.Carried[1].ID)
	}
	if f.Carried[1].StartingHero {
		t.Error("the companion crossed as the starting hero")
	}
	// NextParty is what every mission this front end opens by number resolves
	// its party through, so the companion reaching it is what says he is in the
	// next mission and not merely in a field.
	if got := len(f.NextParty()); got != 2 {
		t.Errorf("the next mission opens with %d members, want 2", got)
	}
}

func TestALostMissionCarriesNoJoinedCompanion(t *testing.T) {
	f := continuityFront(t)
	f.continuity(20, continuityMission(t, 20, sim.ScriptInstantWin), listAdvance)()
	if !f.Town.Open() {
		t.Fatal("the town did not open")
	}
	before := len(f.Carried)

	lost := joinedMission(t, 30, sim.ScriptInstantLose)
	if dest, _, _ := f.continuity(30, lost, menuAdvance)(); dest != ui.NoticeToMenu {
		t.Fatalf("destination %v, want NoticeToMenu", dest)
	}
	if len(f.Carried) != before {
		t.Errorf("a lost mission carried %d members, was %d: a defeat keeps nobody new",
			len(f.Carried), before)
	}
	for _, p := range f.Carried {
		if p.ID == "join:1" {
			t.Error("the companion crossed a boundary the party lost")
		}
	}
}

func TestFinishMissionWithNoRosterCarriesOnlyTheMembers(t *testing.T) {
	f := continuityFront(t)
	ms := joinedMission(t, 20, sim.ScriptInstantWin)
	f.FinishMission(20, ms.Party, ms.World, ms.Start.IDs)

	if len(f.Carried) != 1 {
		t.Errorf("FinishMission carried %d members, want only the one that walked in",
			len(f.Carried))
	}
}
