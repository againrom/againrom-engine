package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

type missionHandoffSample struct {
	World    []byte
	Hash     uint64
	Mission  int
	Campaign sav.CampaignProjection
}

type missionHandoffProof struct {
	SHA256  string
	Samples []missionHandoffSample
}

func missionHandoffDirectory(t *testing.T, leaf string) string {
	t.Helper()
	root := os.Getenv("AGAINROM_MISSION_HANDOFF_OUTPUT")
	if root == "" {
		root = t.TempDir()
		t.Setenv("AGAINROM_MISSION_HANDOFF_OUTPUT", root)
	}
	if !filepath.IsAbs(root) {
		t.Fatal("witness needs an explicit absolute AGAINROM_MISSION_HANDOFF_OUTPUT")
	}
	dir := filepath.Join(root, strings.ReplaceAll(t.Name(), "/", "_"), leaf)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func missionHandoffStage(t *testing.T, stage string, value any) {
	t.Helper()
	row := struct {
		Stage string
		PID   int
		Value any
	}{stage, os.Getpid(), value}
	raw, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("stage=%s %s", stage, raw)
	path := filepath.Join(missionHandoffDirectory(t, "evidence"), "stages.jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := file.Write(append(raw, '\n'))
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatal(writeErr, closeErr)
	}
}

func missionHandoffDecode(t *testing.T, stage, path string) ([]byte, sav.CampaignProjection) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("stage=%s read: %v", stage, err)
	}
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatalf("stage=%s open: %v", stage, err)
	}
	projection, present, err := file.Campaign()
	if err != nil || !present {
		t.Fatalf("stage=%s campaign present=%t: %v", stage, present, err)
	}
	missionHandoffStage(t, stage, struct {
		Path, SHA256 string
		Header       uint32
		Campaign     sav.CampaignProjection
	}{path, fmt.Sprintf("%x", sha256.Sum256(raw)), file.Head.Mission, projection})
	return raw, projection
}

func missionHandoffLoad(t *testing.T, stage, path string) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	app := f.App("mission handoff LOAD")
	app.Layout(1024, 768)
	app.SetCutscenes(nil)
	f.ConfigureSaveSeams(app, SaveStore{Dir: filepath.Dir(path)}, OriginalStore{}, nil)
	if err := app.HeadlessActivate("load game"); err != nil {
		t.Fatalf("stage=%s open LOAD: %v", stage, err)
	}
	rows := app.HeadlessRows()
	if app.Screen() != ui.ScreenLoad || len(rows) != 1 {
		t.Fatalf("stage=%s expected one SAV: screen=%s rows=%v", stage, app.Screen(), rows)
	}
	err := app.HeadlessActivate(rows[0].Text)
	missionHandoffStage(t, stage, map[string]any{"screen": app.Screen().String(), "message": app.HeadlessMessage(), "error": fmt.Sprint(err)})
	if err != nil || app.Screen() == ui.ScreenLoad {
		t.Fatalf("stage=%s cold LOAD refused: %v; %s", stage, err, app.HeadlessMessage())
	}
	return f, app
}

func missionHandoffTown(t *testing.T) (*FrontEnd, *ui.App, sav.CampaignProjection) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	prepareAcceptedCampaignMission(t, f, 131)
	if f.Town.Chapter() != 130 || !slices.Contains(f.Town.Available(), 131) || !f.TownScreen().(*townScreen).AtTownSquare() {
		t.Fatal("stage=accepted-offer: expected settled chapter 130 with accepted 131")
	}
	before, _, err := f.Snapshot(false)
	if err != nil || before.Mission != 0 || before.WorldMapReturn != nil {
		t.Fatalf("stage=town-snapshot: %v", err)
	}
	missionHandoffStage(t, "accepted-offer", map[string]any{"main": f.Town.Chapter(), "selected": f.Town.selectedMission(), "available": f.Town.Available()})
	store := SaveStore{Dir: missionHandoffDirectory(t, "town")}
	prepared, err := f.SaveDialogSeams(store, OriginalStore{}).Prepare(ui.SaveRequest{
		Directory: store.Dir, Name: "Accepted side town", Format: ui.SaveSAV, OnMap: false,
	})
	if err != nil {
		t.Fatalf("stage=town-save-prepare: %v", err)
	}
	paths, err := prepared.Commit(false)
	if err != nil || len(paths) != 1 {
		t.Fatalf("stage=town-save-commit: %v %v", paths, err)
	}
	after, _, err := f.Snapshot(false)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("stage=town-save: SAVE changed source state", err)
	}
	_, projection := missionHandoffDecode(t, "town-saved-bytes", paths[0])
	if projection.Main.Mission != 130 || projection.SelectedMission != 130 {
		t.Fatal("stage=town-saved-bytes: prerequisite differs from the intended 130/130 case")
	}
	g, app := missionHandoffLoad(t, "town-cold-load", paths[0])
	if app.Screen() != ui.ScreenTown || g.Town.progress == nil || g.Town.Chapter() != 130 || g.Town.selectedMission() != 130 || !slices.Contains(g.Town.Available(), 131) || !g.TownScreen().(*townScreen).AtTownSquare() {
		t.Fatal("stage=town-cold-load: missing restored, settled 130/130 with accepted 131")
	}
	return g, app, projection
}

func missionHandoffSampleNow(t *testing.T, f *FrontEnd) missionHandoffSample {
	t.Helper()
	s, _, err := f.Snapshot(true)
	if err != nil || !s.CampaignState || f.live == nil {
		t.Fatal("stage=sample: missing current campaign or live world", err)
	}
	return missionHandoffSample{s.World, f.live.world.Hash(), s.Mission, s.Campaign}
}

func missionHandoffMissionSave(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	missionHandoffStage(t, "mission-opened", map[string]any{"mission": f.liveMission, "main": f.Town.Chapter(), "selected": f.Town.selectedMission()})
	store := SaveStore{Dir: missionHandoffDirectory(t, "mission")}
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	openMissionGameMenu(t, app)
	before := missionHandoffSampleNow(t, f)
	if err := app.HeadlessGameMenuAction("save"); err != nil || app.Screen() != ui.ScreenSave {
		t.Fatalf("stage=mission-save-dialog: %v; screen=%s", err, app.Screen())
	}
	if err := app.HeadlessSaveEdit(store.Dir, "Accepted side mission", ui.SaveSAV); err != nil {
		t.Fatalf("stage=mission-save-edit: %v", err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatalf("stage=mission-save-commit: %v; %s", err, app.HeadlessMessage())
	}
	after := missionHandoffSampleNow(t, f)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("stage=mission-save: SAVE changed World or campaign")
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 || !IsOriginal(entries[0].Name) {
		t.Fatalf("stage=mission-save-list: %v %v; %s", entries, err, app.HeadlessMessage())
	}
	path := filepath.Join(store.Dir, entries[0].Name)
	raw, projection := missionHandoffDecode(t, "mission-saved-bytes", path)
	file, err := sav.Open(raw)
	if err != nil || file.Head.Mission != 131 || projection.Main.Mission != 130 {
		t.Fatal("stage=mission-saved-bytes: wrong save point", err)
	}
	// Let the real loader report the mismatch before the expected final fields.
	proof := missionHandoffProof{SHA256: fmt.Sprintf("%x", sha256.Sum256(raw)), Samples: []missionHandoffSample{after}}
	f.LiveAdvance(1)
	proof.Samples = append(proof.Samples, missionHandoffSampleNow(t, f))
	encoded, err := json.Marshal(proof)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".proof.json", encoded, 0600); err != nil {
		t.Fatal(err)
	}
	runSpellWitnessChild(t, path, "AGAINROM_MISSION_HANDOFF_INPUT")
	if projection.SelectedMission != 131 || before.Mission != 131 || before.Campaign.SelectedMission != 131 {
		t.Fatal("stage=final-fields: accepted mission did not become campaign selection")
	}
}

func missionHandoffChild(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := os.ReadFile(path + ".proof.json")
	if err != nil {
		t.Fatal(err)
	}
	var proof missionHandoffProof
	if err := json.Unmarshal(encoded, &proof); err != nil || len(proof.Samples) != 2 || proof.SHA256 != fmt.Sprintf("%x", sha256.Sum256(raw)) {
		t.Fatal("stage=mission-cold-input: changed or invalid proof", err)
	}
	f, app := missionHandoffLoad(t, "mission-cold-load", path)
	if app.Screen() != ui.ScreenMap {
		t.Fatal("stage=mission-cold-load: did not enter the map")
	}
	for i, want := range proof.Samples {
		got := missionHandoffSampleNow(t, f)
		encoded, err := json.Marshal(got)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(missionHandoffDirectory(t, "evidence"), fmt.Sprintf("cold-sample-%d.json", i)), encoded, 0600); err != nil {
			t.Fatal(err)
		}
		missionHandoffStage(t, "mission-continuation", map[string]any{"tick": i, "worldHash": got.Hash, "sourceWorldHash": want.Hash, "rawWorldEqual": bytes.Equal(got.World, want.World), "main": got.Campaign.Main.Mission, "selected": got.Campaign.SelectedMission})
		if got.Mission != want.Mission || !reflect.DeepEqual(got.Campaign, want.Campaign) {
			t.Fatalf("stage=mission-continuation tick=%d mission=%d/%d hash=%x/%x worldEqual=%t campaignEqual=%t", i, got.Mission, want.Mission, got.Hash, want.Hash, bytes.Equal(got.World, want.World), reflect.DeepEqual(got.Campaign, want.Campaign))
		}
		if i == 0 {
			f.LiveAdvance(1)
		}
	}
}

func TestReleaseMissionHandoffDirectSAV(t *testing.T) {
	if path := os.Getenv("AGAINROM_MISSION_HANDOFF_INPUT"); path != "" {
		missionHandoffChild(t, path)
		return
	}
	f, app, _ := missionHandoffTown(t)
	if err := app.OpenMission(f.MissionOpenerWith(131, f.Carried)); err != nil {
		t.Fatalf("stage=public-direct-open: %v", err)
	}
	missionHandoffMissionSave(t, f, app)
}

func TestReleaseMissionHandoffWorldMapSAV(t *testing.T) {
	if path := os.Getenv("AGAINROM_MISSION_HANDOFF_INPUT"); path != "" {
		missionHandoffChild(t, path)
		return
	}
	f, app, _ := missionHandoffTown(t)
	if err := app.HeadlessActivate("GATES"); err != nil {
		t.Fatalf("stage=world-map-gates: %v", err)
	}
	if err := app.HeadlessActivate("walk out to mission 131"); err != nil || app.Screen() != ui.ScreenMap {
		t.Fatalf("stage=world-map-entry: %v; screen=%s message=%s", err, app.Screen(), app.HeadlessMessage())
	}
	missionHandoffMissionSave(t, f, app)
}

func TestReleaseMissionHandoffActivation(t *testing.T) {
	f, _, _ := missionHandoffTown(t)
	var before Snapshot
	snapshotTown(f.Town, &before)
	previousWorld, previousMission := f.live, f.liveMission
	for _, mission := range []int{-1, 120} {
		if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpenerWith(mission, f.Carried)(); err == nil {
			t.Fatalf("stage=failed-open: mission %d unexpectedly accepted", mission)
		}
		var after Snapshot
		snapshotTown(f.Town, &after)
		if !reflect.DeepEqual(before, after) || f.live != previousWorld || f.liveMission != previousMission {
			t.Fatal("stage=failed-open: rejection changed town or live driver")
		}
	}
	var activate func()
	opener := f.missionOpenerMode(131, f.Carried, nil, nil, nil, &activate, nil, f.Difficulty, f.Town)
	if _, _, _, _, _, _, _, _, _, _, err := opener(); err != nil {
		t.Fatalf("stage=deferred-prepare: %v", err)
	}
	var prepared Snapshot
	snapshotTown(f.Town, &prepared)
	if activate == nil || !reflect.DeepEqual(before, prepared) || f.live != previousWorld || f.liveMission != previousMission {
		t.Fatal("stage=deferred-prepare: uncommitted candidate changed town or live driver")
	}
	activate()
	missionHandoffStage(t, "deferred-commit", map[string]any{"mission": f.liveMission, "main": f.Town.Chapter(), "selected": f.Town.selectedMission()})
	if f.liveMission != 131 || f.Town.Chapter() != 130 || f.Town.selectedMission() != 131 {
		t.Fatal("stage=deferred-commit: accepted mission did not synchronize selection")
	}
}

func TestReleaseMissionHandoffMismatchGuard(t *testing.T) {
	f, _, projection := missionHandoffTown(t)
	var before Snapshot
	snapshotTown(f.Town, &before)
	bad := before
	bad.Mission = 131
	for _, source := range []string{"snapshot", "SAV"} {
		var err error
		if source == "snapshot" {
			_, _, err = f.Restore(bad)
		} else {
			_, _, err = f.RestoreOriginal(originalSaveWithCampaign(t, 131, projection))
		}
		if err == nil || !strings.Contains(err.Error(), "active mission 131 does not match selected mission 130") {
			t.Fatalf("stage=mismatch-guard source=%s: %v", source, err)
		}
		var after Snapshot
		snapshotTown(f.Town, &after)
		if !reflect.DeepEqual(before, after) || f.live != nil || f.liveMission != 0 {
			t.Fatal("stage=mismatch-guard: rejected LOAD mutated town or installed a mission")
		}
		missionHandoffStage(t, "mismatch-guard", map[string]string{"source": source, "error": err.Error()})
	}
}
