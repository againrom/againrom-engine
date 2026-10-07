package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type missionAutosaveSample struct {
	State                missionHandoffSample
	View                 ui.SaveApplicationState
	CampaignKnown        bool
	Main, Selected, Gold int
	Won                  []int
}

type missionAutosaveProof struct {
	SHA256  string
	Label   string
	Actor   uint32
	X, Y    int
	Samples []missionAutosaveSample
}

func missionAutosaveSampleNow(t *testing.T, f *FrontEnd) missionAutosaveSample {
	t.Helper()
	snapshot, _, err := f.Snapshot(true)
	if err != nil || f.live == nil {
		t.Fatal("current started state", err)
	}
	return missionAutosaveSample{State: missionHandoffSample{snapshot.World, f.live.world.Hash(), snapshot.Mission, snapshot.Campaign},
		View: f.live.view.SaveApplication(), CampaignKnown: snapshot.CampaignState,
		Main: f.Town.Chapter(), Selected: f.Town.selectedMission(), Gold: f.Town.Gold(), Won: snapshot.Won}
}

func missionAutosaveStore(t *testing.T) SaveStore {
	t.Helper()
	dir := os.Getenv("AGAINROM_MISSION_AUTOSAVE_OUTPUT")
	if dir == "" {
		dir = t.TempDir()
	} else {
		if !filepath.IsAbs(dir) {
			t.Fatal("mission autosave output must be absolute")
		}
		dir = filepath.Join(dir, strings.ReplaceAll(t.Name(), "/", "_"))
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	return SaveStore{Dir: dir}
}

func missionAutosaveRow(t *testing.T, f *FrontEnd, store SaveStore, mission int) SaveEntry {
	t.Helper()
	store.Selector = f.textSelector()
	rows, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	label := fmt.Sprintf("autosave start mission %d - mission %d", mission, mission)
	for _, row := range rows {
		if row.Label == label {
			raw, err := store.Read(row.Name)
			if err != nil {
				t.Fatal(err)
			}
			file, err := sav.Open(raw)
			if err != nil || file.Head.Mission != uint32(mission) || file.Head.CounterA != 0 ||
				asciiLabel(file.Label, f.textSelector()) != fmt.Sprintf("autosave start mission %d", mission) {
				t.Fatal("invalid ordinary tick-zero mission autosave", err)
			}
			return row
		}
	}
	t.Fatalf("no automatic mission-%d SAV in %v; map messages %v", mission, rows, f.live.view.MessageLines())
	return SaveEntry{}
}

func missionAutosaveNext(f *FrontEnd, proof missionAutosaveProof) {
	f.live.enqueue(proof.Actor, proof.X, proof.Y)
	for range 16 {
		f.live.tick()
	}
}

func missionAutosaveWitness(t *testing.T, f *FrontEnd, app *ui.App, store SaveStore, mission int) {
	t.Helper()
	row := missionAutosaveRow(t, f, store, mission)
	raw, err := store.Read(row.Name)
	if err != nil {
		t.Fatal(err)
	}
	file, err := sav.Open(raw)
	label := fmt.Sprintf("autosave start mission %d", mission)
	if err != nil || file.Head.Mission != uint32(mission) || file.Head.CounterA != 0 ||
		asciiLabel(file.Label, f.textSelector()) != label || f.live.world.Tick() != 0 {
		t.Fatal("mission start did not write the accepted tick-zero state and exact label", err)
	}
	first := missionAutosaveSampleNow(t, f)
	proof := missionAutosaveProof{SHA256: fmt.Sprintf("%x", sha256.Sum256(raw)), Label: label, Samples: []missionAutosaveSample{first}}
	for _, entity := range f.live.world.Entities() {
		if entity.Owner == sim.SelfSlot && entity.Alive() {
			proof.Actor, proof.X, proof.Y = uint32(entity.ID), int(entity.X)+1, int(entity.Y)
			break
		}
	}
	if proof.Actor == 0 {
		t.Fatal("no local actor for the next action")
	}
	missionAutosaveNext(f, proof)
	proof.Samples = append(proof.Samples, missionAutosaveSampleNow(t, f))
	if bytes.Equal(first.State.World, proof.Samples[1].State.World) || f.live.world.Tick() != 16 {
		t.Fatal("late-state loss control did not advance the current world")
	}
	if after, err := store.Read(row.Name); err != nil || !bytes.Equal(raw, after) {
		t.Fatal("later play changed the tick-zero file", err)
	}
	late, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	lateRaw, _, err := f.playerMissionSave(late, "late-state control")
	if err != nil {
		t.Fatal(err)
	}
	lateFile, err := sav.Open(lateRaw)
	if err != nil || lateFile.Head.CounterA != 16 || bytes.Equal(raw, lateRaw) {
		t.Fatal("early and late producer output are indistinguishable", err)
	}
	path := filepath.Join(store.Dir, row.Name)
	encoded, err := json.MarshalIndent(proof, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".proof.json", encoded, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".late-control", lateRaw, 0600); err != nil {
		t.Fatal(err)
	}
	runSpellWitnessChild(t, path, "AGAINROM_MISSION_AUTOSAVE_INPUT")
	app.Layout(1280, 960)
	if after, err := store.Read(row.Name); err != nil || !bytes.Equal(raw, after) {
		t.Fatal("resize changed the mission-start file", err)
	}
	t.Logf("mission %d: %s label %q SHA256 %s; tick0 and next-action tick16 cold LOAD", mission, row.Name, label, proof.SHA256)
}

func missionAutosaveCold(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := os.ReadFile(path + ".proof.json")
	if err != nil {
		t.Fatal(err)
	}
	var proof missionAutosaveProof
	if err := json.Unmarshal(encoded, &proof); err != nil || len(proof.Samples) != 2 ||
		proof.SHA256 != fmt.Sprintf("%x", sha256.Sum256(raw)) {
		t.Fatal("invalid source-state proof", err)
	}
	store := SaveStore{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(store.Dir, filepath.Base(path)), raw, 0600); err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	app := f.App("mission start cold LOAD")
	app.Layout(1024, 768)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if err := app.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	rows := app.HeadlessRows()
	wantRow := proof.Label + fmt.Sprintf(" - mission %d", proof.Samples[0].State.Mission)
	if len(rows) != 1 || rows[0].Text != wantRow {
		t.Fatalf("ordinary chooser row = %v, want %q", rows, wantRow)
	}
	frame, _, err := app.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	var picture bytes.Buffer
	if err := png.Encode(&picture, frame); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".chooser.png", picture.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("@first"); err != nil || app.Screen() != ui.ScreenMap || !f.live.mission.resumed {
		t.Fatal("ordinary chooser cold LOAD", err, app.Screen())
	}
	for i, want := range proof.Samples {
		got := missionAutosaveSampleNow(t, f)
		if got.State.Hash != want.State.Hash || !bytes.Equal(got.State.World, want.State.World) ||
			got.State.Mission != want.State.Mission || want.CampaignKnown && !reflect.DeepEqual(got.State.Campaign, want.State.Campaign) ||
			got.Main != want.Main || got.Selected != want.Selected || got.Gold != want.Gold || !reflect.DeepEqual(got.Won, want.Won) ||
			!reflect.DeepEqual(got.View, want.View) {
			t.Fatalf("cold sample %d differs: World=%v hash=%x/%x campaign=%v view=%+v/%+v", i,
				bytes.Equal(got.State.World, want.State.World), got.State.Hash, want.State.Hash,
				reflect.DeepEqual(got.State.Campaign, want.State.Campaign), got.View, want.View)
		}
		if i == 0 {
			missionAutosaveNext(f, proof)
		}
	}
	if rows, err := store.List(); err != nil || len(rows) != 1 {
		t.Fatal("LOAD created another start save", rows, err)
	}
	if after, err := store.Read(filepath.Base(path)); err != nil || !bytes.Equal(raw, after) {
		t.Fatal("LOAD replaced its source file", err)
	}
}

func TestReleaseMissionStartAutosaveRoutes(t *testing.T) {
	if path := os.Getenv("AGAINROM_MISSION_AUTOSAVE_INPUT"); path != "" {
		missionAutosaveCold(t, path)
		return
	}
	for _, route := range []string{"new game", "direct mission", "direct base"} {
		t.Run(route, func(t *testing.T) {
			f := releaseFront(t)
			f.Options = OptionsStore{}
			f.SetDeterministicFrames(true)
			store := missionAutosaveStore(t)
			app := f.App("mission start " + route)
			app.Layout(1024, 768)
			f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
			switch route {
			case "new game":
				if err := app.HeadlessActivate("new game"); err != nil {
					t.Fatal(err)
				}
				if app.Screen() == ui.ScreenPicker {
					if err := app.HeadlessActivate("@first"); err != nil {
						t.Fatal(err)
					}
				}
				if err := headlessCreateCharacter(app, HeadlessCharacter{Name: "Autosave", Difficulty: "normal"}); err != nil {
					t.Fatal(err)
				}
			case "direct mission":
				if err := app.OpenMission(f.MissionOpener(10)); err != nil {
					t.Fatal(err)
				}
			case "direct base":
				if err := app.OpenMission(f.DirectNewGame(10)); err != nil {
					t.Fatal(err)
				}
			}
			missionAutosaveWitness(t, f, app, store, 10)
		})
	}
	t.Run("town worldmap and restart", func(t *testing.T) {
		f := releaseFront(t)
		f.Options = OptionsStore{}
		f.SetDeterministicFrames(true)
		store := missionAutosaveStore(t)
		source := os.Getenv("AGAINROM_SAVE_666")
		if source == "" {
			t.Skip("AGAINROM_SAVE_666 is not set")
		}
		original, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		app := f.App("mission start town route")
		app.Layout(640, 480)
		f.ConfigureSaveSeams(app, store, OriginalStore{Dir: filepath.Dir(source)}, nil)
		scenario := HeadlessScenario{Version: 7, Steps: []HeadlessStep{
			{Command: "load", Target: "666 - mission 20"},
			{Command: "activate", Target: "notice"},
			{Command: "wait_until", Until: &HeadlessUntil{Control: "SHOP"}, Ticks: 4000},
		}}
		if err := RunHeadlessScenario(f, app, scenario, io.Discard, io.Discard); err != nil {
			t.Fatal(err)
		}
		if rows, err := store.List(); err != nil || len(rows) != 0 {
			t.Fatal("source LOAD or homecoming wrote a mission-start save", rows, err)
		}
		screen := shopOrderEnter(t, f, app)
		beforeGold := f.Town.Gold()
		stock := shopOrderShelves(t, app, screen, "current-state purchase")
		_, _, _ = shopOrderBuy(t, app, screen, stock)
		if f.Town.Gold() >= beforeGold {
			t.Fatal("current-state loss control did not buy an item")
		}
		shopPointer(t, app)("button", 3, "press", "release")
		takeCampaignOffer(t, f, 30)
		gold := f.Town.Gold()
		app.Layout(1024, 768)
		if err := app.HeadlessActivate("GATES"); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessActivate("walk out to mission 30"); err != nil {
			t.Fatal(err)
		}
		row := missionAutosaveRow(t, f, store, 30)
		raw, err := store.Read(row.Name)
		if err != nil || bytes.Equal(original, raw) || f.live.world.Purse(sim.SelfSlot) != uint32(gold) {
			t.Fatal("entry copied source bytes or lost current city purse", err)
		}
		missionAutosaveWitness(t, f, app, store, 30)
		if err := app.SetModScreens([]ui.ModScreen{{Mod: "mission", Key: "restart", Kind: ui.ModActionRestart,
			Title: "Restart", MenuLabel: "Restart mission", Game: true}}); err != nil {
			t.Fatal(err)
		}
		openMissionGameMenu(t, app)
		if err := app.HeadlessActivate("Restart mission"); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessActivate("Restart"); err != nil {
			t.Fatal(err)
		}
		if f.liveMission != 30 || f.live.world.Tick() != 0 {
			t.Fatal("restart did not accept a fresh tick-zero mission")
		}
		if rows, err := store.List(); err != nil || len(rows) != 2 {
			t.Fatal("restart did not allocate a second ordinary slot", rows, err)
		}
		if after, err := store.Read(row.Name); err != nil || !bytes.Equal(raw, after) {
			t.Fatal("restart overwrote the earlier start file", err)
		}
		missionAutosaveWitness(t, f, app, store, 30)
	})
}

func TestReleaseMissionStartAutosaveSuccessor(t *testing.T) {
	if path := os.Getenv("AGAINROM_MISSION_AUTOSAVE_INPUT"); path != "" {
		missionAutosaveCold(t, path)
		return
	}
	_, raw := groundCorpusFile(t, autoGetTenSAV, autoGetTenHash)
	store := missionAutosaveStore(t)
	if err := os.WriteFile(filepath.Join(store.Dir, "game0004.sav"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	f, app := autoGetSession(t, store.Dir)
	app.Layout(1024, 768)
	var hero sim.EntityID
	for i, member := range f.live.mission.party {
		if member.StartingHero {
			hero = f.live.mission.ids[i]
		}
	}
	witch := mapload.ScriptUnits(f.live.mission.state.Map, f.live.mission.party)[21]
	f.live.enqueue(uint32(witch), 56, 21)
	for i := 0; i < 2000 && f.live.world.ScriptRegister(50) == 0; i++ {
		autoGetStep(t, f, app, "escort reaches the end")
	}
	if hero == 0 || f.live.world.ScriptRegister(50) == 0 {
		t.Fatal("source campaign cannot finish the escort")
	}
	f.live.enqueue(uint32(hero), 66, 16)
	autoGetVictory(t, f, app, 4000, "hero completes mission 10")
	missionAutosaveWitness(t, f, app, store, 20)
	if after, err := store.Read("game0004.sav"); err != nil || !bytes.Equal(raw, after) {
		t.Fatal("successor replaced the original source SAV", err)
	}
}
