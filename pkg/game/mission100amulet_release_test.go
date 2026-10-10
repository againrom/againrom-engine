package game

import (
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// mission100Amulet is the turtle master's amulet: Target_Item 11 of
// 100.alm's take nodes, a quest-class code.
const mission100Amulet = 0x0e18 + 11

// TestReleaseMission100VictoryTakesTheTurtleAmulet carries the amulet in every
// hero's pack into mission 100, heals the captive (unit 245, authored at
// health 0, UNIT-144) and kills unit 168. The win trigger's take node empties
// ordinal 10001 and its twin trigger's nodes empty 10002..10005
// (TRIG-TAKEITEM-038, TRIG-HEROTPL-076), so no hero brings the amulet to town,
// on the live mission and after a SAVE and cold LOAD.
func TestReleaseMission100VictoryTakesTheTurtleAmulet(t *testing.T) {
	for _, cold := range []bool{false, true} {
		name := "live"
		if cold {
			name = "cold"
		}
		t.Run(name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Amulet witness", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
			f.arriveInTown()
			for _, mission := range f.Campaign.Value().Main {
				if mission >= 100 {
					break
				}
				f.addChapterCompanions(mission)
				f.Town.Won(mission)
			}
			brian, ok := mapload.CampaignNPCMember(f.Table, 25, 0, f.Carried)
			if !ok {
				t.Fatal("no Brian to carry")
			}
			f.Carried = append(f.Carried, brian)
			takeCampaignOffer(t, f, 100)
			party := f.NextParty()
			for i := range party {
				party[i].Carried = append(party[i].Carried, mission100Amulet)
				party[i].CarriedItems = append(party[i].CarriedItems, sim.PlainItem(mission100Amulet))
			}
			app := f.App("mission 100 amulet")
			if err := app.OpenMission(f.MissionOpenerWith(100, party)); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 20; i++ {
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			if cold {
				store, _, _ := campaignSave(t, f, true)
				f, app = campaignCold(t, store)
			}
			w := f.live.world
			held := 0
			for _, id := range f.live.mission.ids {
				if c, _ := w.Carried(id); slices.Contains(c, mission100Amulet) {
					held++
				}
			}
			if held != len(party) {
				t.Fatalf("%d of %d heroes enter the mission holding the amulet", held, len(party))
			}
			units := mapload.ScriptUnits(f.live.mission.state.Map, f.live.mission.party)
			if err := w.HeadlessHeal(units[245]); err != nil {
				t.Fatal(err)
			}
			if err := w.HeadlessKill(units[168]); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 300 && w.Outcome() != sim.OutcomeWon; i++ {
				if _, kind, up := f.LiveNotice(); up && kind != ui.NoticeSuccess {
					if err := app.HeadlessActivate("notice"); err != nil {
						t.Fatal(err)
					}
				}
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			if w.Outcome() != sim.OutcomeWon {
				t.Fatal("mission 100 did not reach script victory")
			}
			for i, id := range f.live.mission.ids {
				if c, _ := w.Carried(id); slices.Contains(c, mission100Amulet) {
					t.Errorf("hero %d still holds the amulet after victory: %v", i, c)
				}
			}
			for i := 0; i < 4000 && (app.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare()); i++ {
				if _, kind, up := f.LiveNotice(); up && kind == ui.NoticeSuccess {
					if err := app.HeadlessActivate("notice"); err != nil {
						t.Fatal(err)
					}
				}
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			if app.Screen() != ui.ScreenTown {
				t.Fatal("victory did not return to town", app.Screen())
			}
			for _, p := range f.Carried {
				if slices.Contains(p.Carried, mission100Amulet) {
					t.Errorf("%s brings the amulet to town", p.Name)
				}
			}
		})
	}
}
