package game

import (
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestReleaseMission100AmuletPickedAfterTheWin plays mission 100 the way the
// win comes first: the captive (unit 245) is healed and unit 168 killed while
// unit 146 still holds the amulet, so the win trigger's take nodes find no
// hero holding it (TRIG-TAKEITEM-038). Continue keeps the party on the map;
// the hero kills 146 and picks the amulet out of its sack. The farm trigger
// takes it from ordinal 10001 when a party unit comes within 3 cells of the
// farm. Without that walk nothing in the script takes it again, and End
// Quest's Victory carries it to town. What the original does with a quest
// item at mission end is an open research question, not a rule this build
// holds.
func TestReleaseMission100AmuletPickedAfterTheWin(t *testing.T) {
	for _, farm := range []bool{false, true} {
		name := "town"
		if farm {
			name = "farm"
		}
		t.Run(name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Amulet after the win", Choices: []int{0, 0, 3}, Stats: []int{31, 27, 24, 29}})
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
			app := f.App("mission 100 amulet after the win")
			if err := app.OpenMission(f.MissionOpenerWith(100, f.NextParty())); err != nil {
				t.Fatal(err)
			}
			mw := f.live
			w := mw.world
			hero := mw.mission.ids[0]
			units := mapload.ScriptUnits(mw.mission.state.Map, mw.mission.party)
			closeNotices := func() {
				for i := 0; i < 8; i++ {
					_, kind, up := f.LiveNotice()
					if !up {
						return
					}
					target := "notice"
					if kind == ui.NoticeSuccess {
						target = "continue"
					}
					if err := app.HeadlessActivate(target); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err := w.HeadlessHeal(units[245]); err != nil {
				t.Fatal(err)
			}
			if err := w.HeadlessKill(units[168]); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 300 && w.Outcome() != sim.OutcomeWon; i++ {
				closeNotices()
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			if w.Outcome() != sim.OutcomeWon {
				t.Fatal("mission 100 did not reach script victory")
			}
			closeNotices()
			if app.Screen() != ui.ScreenMap {
				t.Fatal("Continue left the map", app.Screen())
			}
			for i, id := range mw.mission.ids {
				if c, _ := w.Carried(id); slices.Contains(c, mission100Amulet) {
					t.Fatalf("hero %d holds the amulet at the win: %v", i, c)
				}
			}
			if c, _ := w.Carried(units[146]); !slices.Contains(c, mission100Amulet) {
				t.Fatalf("unit 146 does not hold the amulet at the win: %v", c)
			}
			if err := w.HeadlessKill(units[146]); err != nil {
				t.Fatal(err)
			}
			var sx, sy int32
			found := false
			for i := 0; i < 60 && !found; i++ {
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
				for _, s := range w.Sacks() {
					if slices.Contains(s.Items, mission100Amulet) {
						sx, sy, found = s.X, s.Y, true
					}
				}
			}
			if !found {
				t.Fatal("unit 146 dropped no sack with the amulet")
			}
			if err := w.HeadlessPlace(hero, sx+1, sy); err != nil {
				t.Fatal(err)
			}
			mw.orderPickup(hero, sx, sy)
			held := false
			for i := 0; i < 200 && !held; i++ {
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
				c, _ := w.Carried(hero)
				held = slices.Contains(c, mission100Amulet)
			}
			if !held {
				t.Fatal("the hero did not pick the amulet up")
			}
			if farm {
				if err := w.HeadlessPlace(hero, 79, 131); err != nil {
					t.Fatal(err)
				}
				for i := 0; i < 200 && held; i++ {
					if err := app.HeadlessStep(); err != nil {
						t.Fatal(err)
					}
					c, _ := w.Carried(hero)
					held = slices.Contains(c, mission100Amulet)
				}
				if held {
					t.Fatal("the farm trigger did not take the amulet")
				}
				closeNotices()
			}
			if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenGameMenu {
				t.Fatalf("game menu: %v on %s", err, app.Screen())
			}
			if err := app.HeadlessGameMenuAction("end"); err != nil {
				t.Fatal("End Quest:", err)
			}
			if err := app.HeadlessGameMenuAction("victory"); err != nil {
				t.Fatal("Victory:", err)
			}
			for i := 0; i < 4000 && (app.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare()); i++ {
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			if app.Screen() != ui.ScreenTown {
				t.Fatal("Victory did not return to town", app.Screen())
			}
			holders := 0
			for _, p := range f.Carried {
				if slices.Contains(p.Carried, mission100Amulet) {
					holders++
				}
			}
			want := 1
			if farm {
				want = 0
			}
			if holders != want {
				t.Fatalf("%d members bring the amulet to town, want %d", holders, want)
			}
		})
	}
}
