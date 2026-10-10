package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// mission130Ticks bounds the drive of mission 130 with no player action.
const mission130Ticks = 4000

// TestReleaseMission130WinsWithoutItsTenThousandTwoCompanion enters mission
// 130 through the campaign with the chapter companions, once without npc 22,
// who resolves 10002, and once with her. Without her, check 45 is not built
// and T2's pair reads register 0, the breeder-alive check; T2 sets the
// mission state to 2, and T9's slot names the unbuilt action 39 and runs
// subscript 0, Force Mission Complete (TRIG-BIND-010; the owner's run of the
// original, DIV-2794). The mission is won with no player action and the
// victory returns to the town. With npc 22 nothing wins in the same drive.
func TestReleaseMission130WinsWithoutItsTenThousandTwoCompanion(t *testing.T) {
	for _, companion := range []bool{false, true} {
		name := "without-npc22"
		if companion {
			name = "with-npc22"
		}
		t.Run(name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Mission 130 witness", Choices: []int{0, 0, 3}, Stats: []int{31, 27, 24, 29}})
			f.arriveInTown()
			for _, mission := range f.Campaign.Value().Main {
				if mission >= 130 {
					break
				}
				f.addChapterCompanions(mission)
				f.Town.Won(mission)
			}
			// The campaign carries npc 22 by chapter 130; 10003..10005 are npc
			// 23..25 (TRIG-HEROTPL-076), carried here as the owner's kit party
			// carries three companions besides the primary.
			for npc := 23; npc <= 25; npc++ {
				member, ok := mapload.CampaignNPCMember(f.Table, int32(npc), 0, f.Carried)
				if !ok {
					t.Fatalf("no npc %d to carry", npc)
				}
				f.Carried = append(f.Carried, member)
			}
			takeCampaignOffer(t, f, 130)
			var party []mapload.PartyMember
			for _, p := range f.NextParty() {
				if companion || p.CompanionNPC != 22 {
					party = append(party, p)
				}
			}
			app := f.App("mission 130 without 10002")
			if err := app.OpenMission(f.MissionOpenerWith(130, party)); err != nil {
				t.Fatal(err)
			}
			ms := f.live.mission.state
			refs := campaignScriptRefs(ms.Map, f.Table, party, 130)
			roles := 0
			for v := uint32(10003); v <= 10005; v++ {
				if _, ok := refs.Roles[v]; ok {
					roles++
				}
			}
			if refs.HasCompanion != companion || roles != 3 {
				t.Fatalf("party of %d resolves 10002 %v and %d of 10003..10005; want 10002 %v and all three",
					len(party), refs.HasCompanion, roles, companion)
			}
			w := f.live.world
			start := w.Tick()
			won := false
			for frames := 0; frames < 4*mission130Ticks && w.Tick()-start < mission130Ticks; frames++ {
				if _, kind, up := f.LiveNotice(); up {
					if kind == ui.NoticeSuccess {
						won = true
						break
					}
					if kind != ui.NoticeDialogue {
						t.Fatalf("notice %v at tick %d", kind, w.Tick()-start)
					}
					if err := app.HeadlessActivate("notice"); err != nil {
						t.Fatal(err)
					}
				}
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			if companion {
				if won || w.Outcome() != sim.OutcomeUndecided {
					t.Fatalf("with npc 22 the mission ends with outcome %v after %d ticks", w.Outcome(), w.Tick()-start)
				}
				return
			}
			if !won || w.Outcome() != sim.OutcomeWon {
				t.Fatalf("without npc 22: notice won=%v, outcome %v after %d ticks", won, w.Outcome(), w.Tick()-start)
			}
			t.Logf("mission 130 without npc 22 shows the victory after %d ticks", w.Tick()-start)
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal("victory acknowledgment:", err)
			}
			for i := 0; i < 4000 && (app.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare()); i++ {
				if err := app.HeadlessStep(); err != nil {
					t.Fatal("return frame:", err)
				}
			}
			if app.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare() || !f.Town.Open() || f.Town.Chapter() == 130 {
				t.Fatalf("the victory returned to screen %s, town open=%v chapter %d; want the town square past chapter 130",
					app.Screen(), f.Town.Open(), f.Town.Chapter())
			}
		})
	}
}
