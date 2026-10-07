package game

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func victoryWorldMapFront(t *testing.T) (*FrontEnd, *townScreen) {
	t.Helper()
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Archives: &Archives{Containers: worldMapFixtureFS(t)}, Font: resolved(missionFont(), nil)}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	f.Town.Arrive()
	f.Town.announceMission(30)
	s := f.TownScreen().(*townScreen)
	s.Choose(3)
	s.WorldMapMove(1)
	// Depart through the same persistent gates screen: the ticks carry the
	// selected mission's travel to its arrival.
	for i := 0; i < 200 && s.WorldMapView().Selected >= 0; i++ {
		s.WorldMapTick()
	}
	return f, s
}

func TestVictory1088ReplacesCompletedScrollAndReturnsAutomatically(t *testing.T) {
	for _, continued := range []bool{false, true} {
		t.Run(map[bool]string{false: "immediate", true: "after Continue"}[continued], func(t *testing.T) {
			f, s := victoryWorldMapFront(t)
			ms, mw, _ := victoryContinueDriver(t, 30)
			advance := f.continuity(30, ms, mw.advanceNotice)
			if continued {
				if dest, _, _ := advance(ui.NoticeContinue); dest != ui.NoticeStay || f.Town.Done(30) {
					t.Fatal("Continue completed the mission")
				}
			}
			if dest, _, open := advance(ui.NoticeVictory); dest != ui.NoticeToTown || open != nil {
				t.Fatalf("Victory destination=%v opener=%v", dest, open != nil)
			}
			v := s.WorldMapView()
			if !s.AtWorldMap() || !v.Returning || !v.HideScrolls || v.Position != image.Pt(32, 23) || v.Destination != image.Pt(2, 3) {
				t.Fatalf("return view: %+v", v)
			}
			for _, m := range v.Missions {
				if m.Number == 30 {
					t.Fatal("completed mission remains cached")
				}
			}
			gold, party := f.Town.Gold(), f.Carried
			s.WorldMapMove(1)
			if act := s.WorldMapChoose(); act.Open != nil {
				t.Fatal("return Enter opened a mission")
			}
			for tick := 0; tick < 20 && s.AtWorldMap(); tick++ {
				if act := s.WorldMapTick(); act.Open != nil {
					t.Fatal("return tick reopened a mission")
				}
			}
			if !s.AtTownSquare() || s.worldPosition != image.Pt(2, 3) || s.worldMap.returnMission != 0 {
				t.Fatal("return failed to arrive at the square automatically")
			}
			if dest, _, _ := advance(ui.NoticeVictory); dest != ui.NoticeStay {
				t.Fatal("Victory ran twice")
			}
			if f.Town.Gold() != gold || !reflect.DeepEqual(f.Carried, party) {
				t.Fatal("arrival repeated campaign completion")
			}
			if _, ok := f.Town.Take(TownShop, 0); !ok {
				t.Fatal("the chapter after the win offers nothing at the shop")
			}
			s.Choose(3)
			if !s.WorldMapView().AtHome || s.WorldMapView().HideScrolls {
				t.Fatal("gates reopened away from home")
			}
		})
	}
}

func TestVictoryLegacySnapshotReturnResolvesWithoutReplay(t *testing.T) {
	f, s := victoryWorldMapFront(t)
	ms, mw, _ := victoryContinueDriver(t, 30)
	f.continuity(30, ms, mw.advanceNotice)(ui.NoticeVictory)
	s.WorldMapTick()
	pending := SnapshotMapReturn{Mission: 30, Shown: s.WorldMapView().RouteShown}
	if _, _, err := f.Snapshot(false); err == nil {
		t.Fatal("pending return admitted SAVE")
	}
	s.Back()
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Mission != 0 || len(snap.World) != 0 || snap.WorldMapReturn != nil {
		t.Fatal("resolved city retained a third save point")
	}
	snap.WorldMapReturn = &pending
	form, err := EncodeSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(form)
	if err != nil {
		t.Fatal(err)
	}
	g, _ := victoryWorldMapFront(t)
	if _, town, err := g.Restore(decoded); err != nil || !town {
		t.Fatalf("restore=%v town=%v", err, town)
	}
	back := g.TownScreen().(*townScreen)
	if !back.AtTownSquare() || back.worldPosition != image.Pt(2, 3) || back.worldMap.returnMission != 0 {
		t.Fatal("legacy return was not resolved to the city")
	}
	gold := g.Town.Gold()
	back.WorldMapClick(ui.WorldMapCardRect(0).Min.Add(image.Pt(1, 1)))
	back.WorldMapTick()
	if !back.AtTownSquare() || g.Town.Gold() != gold || !g.Town.Done(30) {
		t.Fatal("resumed return did not complete without replay")
	}
}

func TestVictory1088InvalidReturnRestoreIsAtomic(t *testing.T) {
	for _, pending := range []SnapshotMapReturn{{Mission: 0}, {Mission: -1}, {Mission: 30, Shown: -1}} {
		f, s := victoryWorldMapFront(t)
		town, cache := f.Town, s.worldMap
		if _, _, err := f.Restore(Snapshot{Open: true, WorldMapReturn: &pending}); err == nil {
			t.Fatalf("accepted malformed return %+v", pending)
		}
		if f.Town != town || s.worldMap != cache {
			t.Fatal("refused restore changed the active game")
		}
	}
}

func TestVictory1088BackFinishesReturnBeforeReopeningGates(t *testing.T) {
	f, s := victoryWorldMapFront(t)
	ms, mw, _ := victoryContinueDriver(t, 30)
	f.continuity(30, ms, mw.advanceNotice)(ui.NoticeVictory)
	if !s.Back() || !s.AtTownSquare() {
		t.Fatal("Back failed to finish the return")
	}
	if _, ok := f.Town.Take(TownShop, 0); !ok {
		t.Fatal("the chapter after the win offers nothing at the shop")
	}
	s.Choose(3)
	if !s.WorldMapView().AtHome || s.WorldMapView().HideScrolls {
		t.Fatal("Back stranded the party away from home")
	}
}

func TestVictory1088NonWinningReturnsKeepTheRetryAndPurse(t *testing.T) {
	for _, failedOpen := range []bool{false, true} {
		t.Run(map[bool]string{false: "defeat", true: "opener failure"}[failedOpen], func(t *testing.T) {
			f, s := victoryWorldMapFront(t)
			gold, offers := f.Town.Gold(), f.Town.Available()
			if failedOpen {
				// The synthetic archive provides the route, but no mission ALM.
				_, _, _, _, _, _, _, _, _, _, err := s.worldMapMissionAction(30).Open()
				if err == nil {
					t.Fatal("missing mission unexpectedly opened")
				}
			} else {
				ms := continuityMission(t, 30, sim.ScriptInstantLose)
				if dest, _, _ := f.continuity(30, ms, menuAdvance)(); dest != ui.NoticeToMenu {
					t.Fatal("defeat did not exit to menu")
				}
				if s.WorldMapView().Returning || f.Town.Done(30) || f.Town.Gold() != gold || !reflect.DeepEqual(f.Town.Available(), offers) {
					t.Fatal("defeat started a return or changed completion/economy")
				}
				return
			}
			if !s.WorldMapView().Returning {
				t.Fatal("non-winning exit stranded the away map")
			}
			for i := 0; i < 20 && s.AtWorldMap(); i++ {
				s.WorldMapTick()
			}
			if !s.AtTownSquare() {
				t.Fatal("non-winning return did not arrive")
			}
			s.Choose(3)
			if s.WorldMapView().HideScrolls || f.Town.Done(30) || f.Town.Gold() != gold || !reflect.DeepEqual(f.Town.Available(), offers) {
				t.Fatal("return hid retry or changed completion/economy")
			}
			if _, hit := ui.WorldMapCardAt(s.WorldMapView(), ui.WorldMapCardRect(0).Min.Add(image.Pt(1, 1))); !hit {
				t.Fatal("retry is not hittable")
			}
		})
	}
}

func TestVictory1088ImportedAwayMapReturnsAndPersistsHome(t *testing.T) {
	f := restoredWorldMapFront(t, false, true)
	s := f.TownScreen().(*townScreen)
	s.Choose(3)
	if !s.WorldMapView().Returning || !s.WorldMapView().HideScrolls {
		t.Fatal("imported away map is stranded")
	}
	s.WorldMapTick()
	s.WorldMapClick(image.Pt(2, 2))
	s.WorldMapTick()
	if !s.AtTownSquare() {
		t.Fatal("imported return did not enter town")
	}
	snap, _, err := f.Snapshot(false)
	if err != nil || !snap.Campaign.FirstMapPoint || snap.WorldMapReturn != nil {
		t.Fatalf("home snapshot=%+v err=%v", snap.Campaign, err)
	}
	if _, _, err := f.Restore(snap); err != nil {
		t.Fatal(err)
	}
	s.Choose(3)
	if !s.WorldMapView().AtHome || s.WorldMapView().Returning {
		t.Fatal("completed return replayed on load")
	}
}
