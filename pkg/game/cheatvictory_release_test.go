package game

import (
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseCheatVictoryUsesCampaignCompletion(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mission   int
		continued bool
		successor int
		chapter   int
	}{
		{name: "main successor", mission: 10, successor: 20, chapter: 20},
		{name: "main town immediate", mission: 30, chapter: 40},
		{name: "main town after Continue", mission: 30, continued: true, chapter: 40},
		{name: "side town immediate", mission: 151, chapter: 150},
		{name: "side town after Continue", mission: 151, continued: true, chapter: 150},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			f.SetTipsOff(true)
			if tc.mission == 10 {
				f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Cheat victory", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
			} else {
				prepareAcceptedCampaignMission(t, f, tc.mission)
			}
			a := f.App("cheat campaign victory witness")
			a.SetCutscenes(nil)
			a.Layout(1024, 768)
			if err := a.OpenMission(f.MissionOpenerWith(tc.mission, f.NextParty())); err != nil {
				t.Fatal("campaign mission entry", err)
			}
			f.live.stopped = true
			helpCloseNotices(t, f, a)
			if err := a.HeadlessKey("0"); err != nil || !f.live.stopped {
				t.Fatal("campaign witness player pause", err)
			}
			w := f.live.world
			if w.Outcome() != sim.OutcomeUndecided || f.Town.Done(tc.mission) {
				t.Fatal("campaign mission was already completed before the chat command")
			}
			gold := int(w.Purse(sim.SelfSlot)) + chainPayment(t, f, tc.mission)
			cheatEnable(t, f, a)
			hash := w.Hash()
			cheatChat(t, a, "#victory")
			if _, kind, open := f.LiveNotice(); !open || kind != ui.NoticeSuccess {
				t.Fatal("campaign cheat victory did not open the success panel")
			}
			if pic, err := a.HeadlessNoticeFrame(); err != nil || pic == nil {
				t.Fatal("installed success panel", err)
			}
			if w.Outcome() != sim.OutcomeUndecided || w.Hash() != hash || f.Town.Done(tc.mission) {
				t.Fatal("client cheat victory changed the server outcome, world hash or campaign completion before acknowledgment")
			}
			if tc.continued {
				if err := a.HeadlessActivate("continue"); err != nil {
					t.Fatal("Continue control", err)
				}
				if a.Screen() != ui.ScreenMap || a.HeadlessNoticeOpen() || f.Town.Done(tc.mission) || w.Outcome() != sim.OutcomeUndecided || w.Hash() != hash {
					t.Fatalf("Continue: screen=%s open=%t done(%d)=%t worldOutcome=%v hashBefore=%x hashNow=%x; want map, closed panel, unfinished campaign mission and unchanged server world", a.Screen(), a.HeadlessNoticeOpen(), tc.mission, f.Town.Done(tc.mission), w.Outcome(), hash, w.Hash())
				}
				if err := a.HeadlessKey("escape"); err != nil || a.Screen() != ui.ScreenGameMenu {
					t.Fatal("delayed Victory game menu", a.Screen(), err)
				}
				for _, action := range []string{"end", "victory"} {
					if err := a.HeadlessGameMenuAction(action); err != nil {
						t.Fatalf("delayed Victory action %q: %v", action, err)
					}
				}
			} else if err := a.HeadlessActivate("victory"); err != nil {
				t.Fatal("Victory control", err)
			}
			if !f.Town.Done(tc.mission) || f.Town.Done(tc.chapter) || f.Town.Chapter() != tc.chapter || f.Town.Gold() != gold {
				t.Fatalf("cheat acknowledgment: screen=%s done(%d)=%t chapter=%d nextDone=%t gold=%d, want chapter=%d gold=%d", a.Screen(), tc.mission, f.Town.Done(tc.mission), f.Town.Chapter(), f.Town.Done(tc.chapter), f.Town.Gold(), tc.chapter, gold)
			}
			if tc.successor != 0 {
				if a.Screen() != ui.ScreenMap || f.live == nil || f.liveMission != tc.successor || f.live.world == w {
					t.Fatalf("cheat Victory destination: screen=%s live=%d, want successor %d", a.Screen(), f.liveMission, tc.successor)
				}
				cold, coldApp := cheatColdApp(t, f)
				if coldApp.Screen() != ui.ScreenMap || cold.liveMission != tc.successor || !cold.Town.Done(tc.mission) || cold.Town.Done(tc.successor) {
					t.Fatal("ordinary successor SAV lost cheat campaign completion", coldApp.Screen(), cold.liveMission)
				}
			} else {
				if a.Screen() != ui.ScreenTown {
					t.Fatalf("cheat Victory destination=%s, want town; message=%q", a.Screen(), a.HeadlessMessage())
				}
				for i := 0; i < 4000 && !f.townUI.AtTownSquare(); i++ {
					if err := a.HeadlessStep(); err != nil {
						t.Fatal("automatic town return", err)
					}
				}
				if a.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare() {
					t.Fatal("cheat Victory did not reach the usable town square")
				}
				for range 32 {
					if err := a.HeadlessStep(); err != nil {
						t.Fatal(err)
					}
				}
				if f.Town.Gold() != gold || !f.Town.Done(tc.mission) {
					t.Fatal("settled town repeated cheat completion or its payment")
				}
				store, _, _ := campaignSave(t, f, false)
				cold, coldApp := campaignCold(t, store)
				if coldApp.Screen() != ui.ScreenTown || !cold.townUI.AtTownSquare() || !cold.Town.Done(tc.mission) || cold.Town.Done(tc.chapter) || cold.Town.Chapter() != tc.chapter || cold.Town.Gold() != gold {
					t.Fatal("ordinary town SAV lost cheat campaign completion", coldApp.Screen(), cold.Town.Chapter(), cold.Town.Gold())
				}
			}
		})
	}
}
