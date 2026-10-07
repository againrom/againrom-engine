package game

import (
	"image"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestReleaseVictoryAppReturnsHomeAndBlocksPendingSave exercises the
// installed campaign/map assets and the real App input path. Mission completion
// is the controlled synthetic victory fixture, NOT ordinary M20/M30 gameplay.
func TestReleaseVictoryAppReturnsHomeAndBlocksPendingSave(t *testing.T) {
	for _, tc := range []struct {
		name        string
		continued   bool
		savePending bool
	}{
		{name: "immediate"},
		{name: "continued", continued: true},
		{name: "immediate/pending SAVE", savePending: true},
		{name: "continued/pending SAVE", continued: true, savePending: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			a := f.App("1088-controlled-victory-release")
			a.Layout(1024, 768)
			store := SaveStore{Dir: t.TempDir()}
			saveNumber := 0
			save, list, load := f.SaveSeams(store, OriginalStore{}, func() time.Time {
				saveNumber++
				return time.Date(2001, 1, 1, 0, 0, saveNumber, 0, time.UTC)
			})
			a.SetSaveSeams(save, list, load)
			key := func(name string) {
				t.Helper()
				if err := a.HeadlessKey(name); err != nil {
					t.Fatalf("key %q: %v", name, err)
				}
			}

			// Controlled precondition: establish the town boundary and boot its
			// native snapshot through App. A SAV town needs the actual generated
			// starting hero; the retired envelope path had let an empty fixture bypass SAV construction.
			f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Victory return", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
			f.FinishMission(20, f.Carried, nil, nil)
			for _, building := range []TownBuilding{TownTavern, TownShop, TownSchool} {
				for _, offer := range f.Town.Offers(building) {
					if offer.Mission > 0 {
						if _, ok := f.Town.Take(building, offer.Index); !ok {
							t.Fatalf("could not accept installed offer %+v", offer)
						}
					}
				}
			}
			available := f.Town.Available()
			if !containsMission(available, 30) {
				t.Fatalf("installed chapter did not offer mission30: %v", available)
			}
			if _, err := save(false); err != nil {
				t.Fatal(err)
			}
			key("load")
			key("enter")
			s := f.TownScreen().(*townScreen)
			if a.Screen() != ui.ScreenTown || !s.AtTownSquare() {
				t.Fatalf("setup town load screen=%s square=%v", a.Screen(), s.AtTownSquare())
			}
			if err := a.HeadlessActivate("GATES"); err != nil {
				t.Fatal(err)
			}
			before := s.WorldMapView()
			var missionPoint image.Point
			for _, mission := range before.Missions {
				if mission.Number == 30 {
					missionPoint = mission.Anchor
				}
			}
			homePoint := before.Position // observed at the installed home before departure
			if !before.AtHome || before.HideScrolls || missionPoint == homePoint {
				t.Fatalf("setup map does not offer an away mission: %+v", before)
			}
			x, y, err := a.HeadlessWorldMapMissionPoint(30)
			if err != nil {
				t.Fatal(err)
			}
			for _, edge := range []string{"press", "release"} {
				if err := a.HeadlessPointer(edge, x, y); err != nil {
					t.Fatal(err)
				}
			}
			for i := 0; i < 10000 && a.Screen() == ui.ScreenTown; i++ {
				if err := a.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			if a.Screen() != ui.ScreenMap || f.live == nil || f.live.mission.number != 30 || !s.AtWorldMap() {
				t.Fatalf("App scroll did not depart to mission30: screen=%s gates=%v", a.Screen(), s.AtWorldMap())
			}

			// Replace gameplay with an explicitly synthetic completed mission.
			// Completion choices, continuity, world-map travel and saving below
			// are production paths. No installed-map outcome is forced.
			ms, mw, v := victoryContinueDriver(t, 30)
			ms.Party = mapload.CloneParty(f.Carried)
			v.SetGameMenuContext(func() ui.GameMenuContext { return gameMenuContext(mw, true, "") })
			if err := a.OpenMission(func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
				return v, mw.paced, mw.enqueue, mw.setCadenceMode, mw.affect,
					f.continuity(30, ms, mw.advanceNotice), mw.attackOrCast, mw.grab, mw.stance, mw.march, nil
			}); err != nil {
				t.Fatal(err)
			}
			if tc.continued {
				key("escape")
				if f.Town.Done(30) || !containsMission(f.Town.Available(), 30) {
					t.Fatal("Continue completed the mission before Victory")
				}
				key("escape")
				for _, action := range []string{"end", "victory"} {
					if err := a.HeadlessGameMenuAction(action); err != nil {
						t.Fatalf("menu %q: %v", action, err)
					}
				}
			} else {
				key("enter")
			}

			wantAvailable := make([]int, 0, len(available)-1)
			for _, number := range available {
				if number != 30 {
					wantAvailable = append(wantAvailable, number)
				}
			}
			returnView := s.WorldMapView()
			if a.Screen() != ui.ScreenTown || !s.AtWorldMap() || !returnView.Returning || !returnView.HideScrolls || returnView.AtHome || returnView.Position != missionPoint || returnView.Destination != homePoint {
				t.Fatalf("Victory return view screen=%s gates=%v view=%+v", a.Screen(), s.AtWorldMap(), returnView)
			}
			if len(returnView.Route) < 2 || returnView.Route[0] != missionPoint || returnView.Route[len(returnView.Route)-1] != homePoint {
				t.Fatalf("homeward route endpoints: %v", returnView.Route)
			}
			if !f.Town.Done(30) || !reflect.DeepEqual(f.Town.Available(), wantAvailable) {
				t.Fatalf("completion changed wrong offers: done=%v available=%v want=%v", f.Town.Done(30), f.Town.Available(), wantAvailable)
			}
			var cached []int
			for _, mission := range returnView.Missions {
				cached = append(cached, mission.Number)
			}
			if !reflect.DeepEqual(cached, wantAvailable) {
				t.Fatalf("return cached offers=%v, want current offers=%v", cached, wantAvailable)
			}
			if _, hit := ui.WorldMapCardAt(returnView, ui.WorldMapCardRect(0).Min.Add(image.Pt(10, 10))); hit {
				t.Fatal("off-home return still exposes a scroll hit target")
			}
			gold, completed := f.Town.Gold(), f.Town.finishedCount()
			party := mapload.CloneParty(f.Carried)
			if gold != int(ms.World.Purse(sim.SelfSlot))+f.Campaign.Value().Reward(30) {
				t.Fatalf("completion payment=%d, want purse plus one reward=%d", gold, int(ms.World.Purse(sim.SelfSlot))+f.Campaign.Value().Reward(30))
			}

			if tc.savePending {
				for i := 0; i < 100 && s.WorldMapView().RouteShown == 0; i++ {
					if err := a.HeadlessStep(); err != nil {
						t.Fatal(err)
					}
				}
				key("f2")
				if a.Screen() != ui.ScreenTown || !s.WorldMapView().Returning {
					t.Fatal("pending F2 opened SAVE or resolved travel")
				}
				entries, err := store.List()
				if err != nil || len(entries) != 1 {
					t.Fatalf("pending F2 wrote a new slot: %v err=%v", entries, err)
				}
				key("f3")
				if a.Screen() != ui.ScreenLoad {
					t.Fatal("pending return blocked F3")
				}
				key("escape")
				if err := a.HeadlessGameMenuAction("save"); err == nil {
					t.Fatal("pending return enabled menu SAVE")
				}
				key("escape")
			}

			// No pointer, key, direct WorldMapTick, or manual Home action is
			// supplied here. App's own cadence must finish the route and arrive.
			retainedDriver := f.live
			for i := 0; i < 10000 && !s.AtTownSquare(); i++ {
				if a.Screen() != ui.ScreenTown || f.live != retainedDriver {
					t.Fatal("homeward travel reopened a mission")
				}
				if err := a.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			arrived := s.WorldMapView()
			if a.Screen() != ui.ScreenTown || !s.AtTownSquare() || arrived.Returning || len(arrived.Route) != 0 || arrived.Position != homePoint || !arrived.AtHome {
				t.Fatalf("automatic arrival failed: screen=%s square=%v view=%+v", a.Screen(), s.AtTownSquare(), arrived)
			}
			if !f.Town.Done(30) || f.Town.finishedCount() != completed || f.Town.Gold() != gold || !reflect.DeepEqual(f.Carried, party) || !reflect.DeepEqual(f.Town.Available(), wantAvailable) {
				t.Fatalf("return/load replayed campaign completion: done=%v count=%d/%d gold=%d/%d partyEqual=%v available=%v", f.Town.Done(30), f.Town.finishedCount(), completed, f.Town.Gold(), gold, reflect.DeepEqual(f.Carried, party), f.Town.Available())
			}
			if tc.savePending {
				key("f2")
				entries, err := store.List()
				if err != nil || len(entries) != 2 {
					t.Fatal("resolved city SAVE", entries, err)
				}
				encoded, err := store.Read(entries[0].Name)
				if err != nil {
					t.Fatal(err)
				}
				doc, err := sav.DecodeDocumentData(encoded)
				if err != nil || doc.Head.Mission != 0 || doc.World != nil {
					t.Fatal("resolved city save point", err)
				}
				actions, err := readCurrentActions(&doc)
				if err != nil || actions.WorldMapReturn != nil {
					t.Fatal("resolved city retained travel", err)
				}
				key("escape")
			}
			if err := a.HeadlessActivate("GATES"); err != nil {
				t.Fatal(err)
			}
			gates := s.WorldMapView()
			if !gates.AtHome || gates.HideScrolls || gates.Returning {
				t.Fatalf("fresh gates did not remain at home: %+v", gates)
			}
			for _, mission := range gates.Missions {
				if mission.Number == 30 {
					t.Fatal("completed mission30 scroll returned on fresh Gates entry")
				}
			}
			t.Logf("controlled Victory returned %v -> %v along %d points; gold%d unchanged; remaining offers%v", missionPoint, homePoint, len(returnView.Route), gold, wantAvailable)
		})
	}
}
