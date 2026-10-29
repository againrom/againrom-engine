package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// lossReasonText is a mission text with four failure entries, reasons 2
// to 5.
var lossReasonText = missionSource{"main/text/mission7.txt": []byte(
	"#failure2\r\nThe hero is dead.\r\n#failure3\r\nIglez is dead.\r\n#failure4\r\nDina is dead.\r\n#failure5\r\nThe caravan is lost.\r\n")}

// lossReasonMission is a second-game mission whose party is the hero (1),
// identity 22 (2), identity 23 (3) and a hire (4), beside an enemy unit 5,
// with the fallen entities at -10001 and the given script.
func lossReasonMission(t *testing.T, fallen []sim.EntityID, checks []sim.ScriptCheck, instants []sim.ScriptInstant, triggers []sim.ScriptTrigger) (*mapWorld, *ui.Viewer) {
	t.Helper()
	s, err := sim.NewROM2Script(append(missionChecks(), checks...), instants, triggers)
	if err != nil {
		t.Fatal(err)
	}
	ents := []sim.Entity{
		{ID: 1, Owner: sim.SelfSlot, X: 20, Y: 20, HP: 40, MaxHP: 40, TypeID: sim.HumanTypeID, Humanoid: true},
		{ID: 2, Owner: sim.SelfSlot, X: 22, Y: 20, HP: 40, MaxHP: 40, TypeID: sim.HumanTypeID, Humanoid: true},
		{ID: 3, Owner: sim.SelfSlot, X: 24, Y: 20, HP: 40, MaxHP: 40, TypeID: sim.HumanTypeID, Humanoid: true},
		{ID: 4, Owner: sim.SelfSlot, X: 26, Y: 20, HP: 40, MaxHP: 40, TypeID: sim.HumanTypeID, Humanoid: true},
		{ID: 5, Owner: 2, X: 40, Y: 40, HP: 40, MaxHP: 40, TypeID: sim.HumanTypeID, Humanoid: true},
	}
	for _, id := range fallen {
		ents[id-1].HP = -10001
	}
	w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, make([]byte, worldFixtureW*worldFixtureH), ents, s)
	if err != nil {
		t.Fatal(err)
	}
	mw, v := missionDriverFor(t, w, nil, lossReasonText)
	mw.mission.table = identityTable()
	mw.mission.party = []mapload.PartyMember{identityHero(),
		{ID: "npc:22", CompanionNPC: 22, PlayerCharacter: true},
		{ID: "npc:23", CompanionNPC: 23, PlayerCharacter: true},
		{ID: "hire", PlayerCharacter: true, MercenaryType: 1}}
	mw.mission.ids = []sim.EntityID{1, 2, 3, 4}
	mw.mission.guarded = []sim.EntityID{1, 2, 3, 4}
	mw.mission.failureText = secondGameFailureText(lossReasonText, TextCode{}, 7)
	return mw, v
}

// Each event that loses a second-game mission stores the reason
// missionLossReason gives it, and the notice shows that reason's failure
// entry; a reason outside the entries shows the generic body.
func TestMissionLossReasonPerEvent(t *testing.T) {
	lose := func(reason int32) []sim.ScriptInstant {
		return []sim.ScriptInstant{{Op: sim.ScriptInstantLose, Args: [10]int32{reason}}}
	}
	fire := []sim.ScriptTrigger{missionTrigger(3, 1, true, 0)}
	deadUnit := []sim.ScriptCheck{{Op: sim.ScriptCheckVIP, Register: 3, Unit: 5, HasUnit: true}}
	for _, tc := range []struct {
		name     string
		fallen   []sim.EntityID
		checks   []sim.ScriptCheck
		instants []sim.ScriptInstant
		triggers []sim.ScriptTrigger
		reason   int32
		want     string
	}{
		{name: "the hero falls", fallen: []sim.EntityID{1}, reason: 2, want: "The hero is dead.\r\n"},
		{name: "identity 22 falls", fallen: []sim.EntityID{2}, reason: 3, want: "Iglez is dead.\r\n"},
		{name: "identity 23 falls", fallen: []sim.EntityID{3}, reason: 4, want: "Dina is dead.\r\n"},
		{name: "a hire falls", fallen: []sim.EntityID{4}, reason: 0},
		{name: "instant 5 stores 5", instants: lose(5), triggers: fire, reason: 5, want: "The caravan is lost.\r\n"},
		{name: "instant 5 stores a reason past the entries", instants: lose(7), triggers: fire, reason: 7},
		{name: "check 18 counts a dead unit", fallen: []sim.EntityID{5}, checks: deadUnit, reason: 1},
		{name: "a fall on the step instant 5 stores 5", fallen: []sim.EntityID{3}, instants: lose(5), triggers: fire, reason: 4, want: "Dina is dead.\r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mw, v := lossReasonMission(t, tc.fallen, tc.checks, tc.instants, tc.triggers)
			missionSteps(mw, 1)
			want := tc.want
			if want == "" {
				want = missionOutcomeText(mw.view, sim.OutcomeLost)
			}
			if mw.mission.outcome != sim.OutcomeLost || mw.mission.lossReason != tc.reason {
				t.Fatalf("outcome %v with reason %d, want lost with reason %d", mw.mission.outcome, mw.mission.lossReason, tc.reason)
			}
			if body, kind, shown := v.NoticeState(); !shown || kind != ui.NoticeFailure || body != want {
				t.Fatalf("the notice shows %q/%v/%v, want %q", body, kind, shown, want)
			}
		})
	}
}

// The first loss takes the notice: a fall after the script's loss keeps
// the script's reason.
func TestMissionLossReasonKeepsTheFirstLoss(t *testing.T) {
	mw, v := lossReasonMission(t, nil, nil,
		[]sim.ScriptInstant{{Op: sim.ScriptInstantLose, Args: [10]int32{5}}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	missionSteps(mw, 1)
	if err := mw.world.HeadlessKill(1); err != nil {
		t.Fatal(err)
	}
	missionSteps(mw, 2)
	if mw.mission.lossReason != 5 {
		t.Fatalf("reason %d after the hero's later fall, want the script's 5", mw.mission.lossReason)
	}
	if body, _, shown := v.NoticeState(); !shown || body != "The caravan is lost.\r\n" {
		t.Fatalf("the notice shows %q/%v, want the script's failure5", body, shown)
	}
}
