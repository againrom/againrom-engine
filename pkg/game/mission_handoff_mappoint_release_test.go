package game

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func missionHandoffMapPointRoute(t *testing.T, path string) {
	t.Helper()
	_, want := missionHandoffDecode(t, "map-point-input", path)
	f, app := missionHandoffLoad(t, "map-point-cold-load", path)
	loaded := missionHandoffSampleNow(t, f)
	if app.Screen() != ui.ScreenMap || loaded.Mission != 131 || !reflect.DeepEqual(loaded.Campaign, want) {
		t.Fatalf("stage=map-point-cold-campaign: mission=%d fullCampaignEqual=%t FirstMapPoint=%t/%t", loaded.Mission, reflect.DeepEqual(loaded.Campaign, want), loaded.Campaign.FirstMapPoint, want.FirstMapPoint)
	}
	f.LiveAdvance(1)
	advanced := missionHandoffSampleNow(t, f)
	if advanced.Mission != 131 || advanced.Campaign.Main.Mission != 130 || advanced.Campaign.SelectedMission != 131 || advanced.Campaign.FirstMapPoint != want.FirstMapPoint {
		t.Fatal("stage=map-point-live-tick: campaign location changed")
	}
	missionHandoffStage(t, "map-point-live-tick", advanced.Campaign)
	// This controlled completion calls the production return seam. It does not
	// assert that the installed mission script has reached its victory condition.
	m := f.live.mission
	next, line := f.FinishMissionWithRoster(131, m.party, f.live.world, m.ids, m.state.Start.Roster)
	if next != 0 || !f.Town.Done(131) || f.Town.Chapter() != 130 || f.Town.Done(130) {
		t.Fatalf("stage=map-point-completion: next=%d main=%d sideDone=%t mainDone=%t: %s", next, f.Town.Chapter(), f.Town.Done(131), f.Town.Done(130), line)
	}
	screen := f.TownScreen().(*townScreen)
	screen.beginWorldMapReturn(131)
	if !screen.AtWorldMap() || !screen.WorldMapView().Returning {
		t.Fatal("stage=map-point-return: no homeward route")
	}
	gold := f.Town.Gold()
	for i := 0; i < 4000 && screen.AtWorldMap(); i++ {
		if action := screen.WorldMapTick(); action.Open != nil {
			t.Fatal("stage=map-point-return: return tick reopened a mission")
		}
	}
	if !screen.AtTownSquare() || screen.worldMap.returnMission != 0 || !f.Town.progress.firstMapPoint || f.Town.Gold() != gold {
		t.Fatal("stage=map-point-home: missing settled home or repeated payment")
	}
	missionHandoffStage(t, "map-point-home", map[string]any{"inputFirstMapPoint": want.FirstMapPoint, "firstMapPoint": f.Town.progress.firstMapPoint, "main": f.Town.Chapter(), "gold": gold})
	takeCampaignOffer(t, f, 130)
	screen.Choose(3)
	view := screen.WorldMapView()
	if !screen.AtWorldMap() || !view.AtHome || view.Returning || view.HideScrolls {
		t.Fatal("stage=map-point-next-gates: gates did not reopen at home with usable scrolls")
	}
	selected := false
	for i := 0; i < len(view.Missions); i++ {
		screen.WorldMapMove(1)
		view = screen.WorldMapView()
		if view.Selected >= 0 && view.Selected < len(view.Missions) && view.Missions[view.Selected].Number == 130 {
			selected = true
			break
		}
	}
	if !selected {
		t.Fatal("stage=map-point-next-gates: accepted main 130 has no selectable scroll")
	}
	// The selected scroll starts travel, and the ticks bring the party to the
	// mission, whose opener the arrival carries.
	var action ui.TownAction
	for i := 0; i < 4000 && action.Open == nil && screen.AtWorldMap(); i++ {
		action = screen.WorldMapTick()
	}
	if action.Open == nil {
		t.Fatal("stage=map-point-next-entry: selected scroll has no opener")
	}
	if err := app.OpenMission(action.Open); err != nil {
		t.Fatal("stage=map-point-next-entry:", err)
	}
	entered := missionHandoffSampleNow(t, f)
	if entered.Mission != 130 || entered.Campaign.Main.Mission != 130 || entered.Campaign.SelectedMission != 130 || entered.Campaign.FirstMapPoint || f.Town.Gold() != gold {
		t.Fatal("stage=map-point-next-entry: next mission location or completed payout changed")
	}
	missionHandoffStage(t, "map-point-next-entry", entered.Campaign)
}

func TestReleaseMissionHandoffMapPointRestore(t *testing.T) {
	if path := os.Getenv("AGAINROM_MISSION_HANDOFF_MAP_POINT_INPUT"); path != "" {
		t.Run(filepath.Base(filepath.Dir(path)), func(t *testing.T) { missionHandoffMapPointRoute(t, path) })
		return
	}
	f, app, _ := missionHandoffTown(t)
	if err := app.HeadlessActivate("GATES"); err != nil {
		t.Fatal("stage=map-point-source-gates:", err)
	}
	if err := app.HeadlessActivate("walk out to mission 131"); err != nil || app.Screen() != ui.ScreenMap {
		t.Fatal("stage=map-point-source-entry:", err)
	}
	before := missionHandoffSampleNow(t, f)
	store := SaveStore{Dir: missionHandoffDirectory(t, "map-point-source")}
	prepared, err := f.SaveDialogSeams(store, OriginalStore{}).Prepare(ui.SaveRequest{
		Directory: store.Dir, Name: "Map point source", Format: ui.SaveSAV, OnMap: true,
	})
	if err != nil {
		t.Fatal("stage=map-point-source-save-prepare:", err)
	}
	paths, err := prepared.Commit(false)
	if err != nil || len(paths) != 1 {
		t.Fatal("stage=map-point-source-save-commit:", paths, err)
	}
	if after := missionHandoffSampleNow(t, f); !reflect.DeepEqual(before, after) {
		t.Fatal("stage=map-point-source-save: SAVE changed World or Campaign")
	}
	raw, source := missionHandoffDecode(t, "map-point-source-bytes", paths[0])
	if source.FirstMapPoint || source.SelectedMission != 131 || source.Main.Mission != 130 {
		t.Fatal("stage=map-point-source-bytes: source is not ordinary selected mission 131")
	}
	for _, first := range []bool{false, true} {
		t.Run(fmt.Sprint(first), func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			doc.Campaign.Scalars[4] = 0
			if first {
				doc.Campaign.Scalars[4] = 1
			}
			encoded, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(missionHandoffDirectory(t, fmt.Sprint(first)), "Map point control.sav")
			if err := os.WriteFile(path, encoded, 0600); err != nil {
				t.Fatal(err)
			}
			_, want := missionHandoffDecode(t, "map-point-control-bytes", path)
			expected := source
			expected.FirstMapPoint = first
			if !reflect.DeepEqual(want, expected) {
				t.Fatal("stage=map-point-control-bytes: targeted flag edit changed another campaign field")
			}
			runSpellWitnessChild(t, path, "AGAINROM_MISSION_HANDOFF_MAP_POINT_INPUT")
		})
	}
}
