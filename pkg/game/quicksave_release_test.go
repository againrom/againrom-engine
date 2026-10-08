package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseQuickSaveCurrentStateAndLoad(t *testing.T) {
	if path := os.Getenv("AGAINROM_QUICK_SAVE_INPUT"); path != "" {
		missionAutosaveCold(t, path)
		return
	}
	if path := os.Getenv("AGAINROM_QUICK_LOAD_INPUT"); path != "" {
		quickLoadCold(t, path)
		return
	}
	if path := os.Getenv("AGAINROM_QUICK_TOWN_INPUT"); path != "" {
		timedTownCold(t, path)
		return
	}
	for _, width := range []int{640, 1024} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			dir := quickProofDirectory(t)
			store := SaveStore{Dir: filepath.Join(dir, "saves")}
			f.Options = OptionsStore{Path: filepath.Join(dir, "options.txt")}
			app := f.App("quick current state")
			app.Layout(width, width*3/4)
			now := time.Unix(100, 0)
			f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return now })
			if err := app.OpenMission(f.DirectNewGame(10)); err != nil {
				t.Fatal(err)
			}
			start := missionAutosaveRow(t, f, store, 10)
			startRaw, err := store.Read(start.Name)
			if err != nil {
				t.Fatal(err)
			}
			manual := filepath.Join(store.Dir, "manual.sav")
			if err := os.WriteFile(manual, startRaw, 0600); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessKey("0"); err != nil {
				t.Fatal(err)
			}
			var emitted [5][]byte
			var latest missionAutosaveProof
			var lastHash uint64
			for i := range 5 {
				for attempts := 0; app.HeadlessNoticeOpen() && attempts < 32; attempts++ {
					before := readQuickSaveSlots(store.Dir)
					if err := app.HeadlessKey("f4"); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(before, readQuickSaveSlots(store.Dir)) {
						t.Fatal("notice admitted F4")
					}
					if err := app.HeadlessActivate("notice"); err != nil {
						t.Fatal(err)
					}
				}
				if app.HeadlessNoticeOpen() {
					t.Fatal("notice did not settle")
				}
				first := missionAutosaveSampleNow(t, f)
				if i > 0 && first.State.Hash == lastHash {
					t.Fatal("changed-source loss control did not change")
				}
				lastHash = first.State.Hash
				if err := app.HeadlessKey("f4"); err != nil || app.Screen() != ui.ScreenMap {
					t.Fatal("actual F4", err)
				}
				if !reflect.DeepEqual(first, missionAutosaveSampleNow(t, f)) {
					t.Fatal("F4 changed current source state")
				}
				slot := i % 3
				path := filepath.Join(store.Dir, string(quickSaveBase(slot))+".sav")
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal("missing-write loss control", err)
				}
				owner, err := os.ReadFile(strings.TrimSuffix(path, ".sav") + quickOwnerExtension)
				if err != nil {
					t.Fatal(err)
				}
				if seq, err := validateQuickSavePair(slot, raw, owner); err != nil || seq != uint64(i+1) {
					t.Fatal("actual sequence", seq, err)
				}
				if i > 0 && bytes.Equal(raw, emitted[i-1]) {
					t.Fatal("F4 retained early bytes")
				}
				emitted[i] = bytes.Clone(raw)
				proof := missionAutosaveProof{SHA256: fmt.Sprintf("%x", sha256.Sum256(raw)), Label: fmt.Sprintf("quicksave %d", slot+1), Samples: []missionAutosaveSample{first}}
				for _, entity := range f.live.world.Entities() {
					if entity.Owner == sim.SelfSlot && entity.Alive() {
						proof.Actor, proof.X, proof.Y = uint32(entity.ID), int(entity.X)+1, int(entity.Y)
						break
					}
				}
				if proof.Actor == 0 {
					t.Fatal("no next-action actor")
				}
				missionAutosaveNext(f, proof)
				proof.Samples = append(proof.Samples, missionAutosaveSampleNow(t, f))
				if bytes.Equal(first.State.World, proof.Samples[1].State.World) {
					t.Fatal("next action did not change world")
				}
				archived := filepath.Join(dir, fmt.Sprintf("sample-%d.sav", i+1))
				quickWriteProof(t, archived, raw, proof)
				runSpellWitnessChild(t, archived, "AGAINROM_QUICK_SAVE_INPUT")
				latest = proof
			}
			for slot, index := range []int{3, 4, 2} {
				if got, err := store.Read(string(quickSaveBase(slot)) + ".sav"); err != nil || !bytes.Equal(got, emitted[index]) {
					t.Fatal("oldest replacement retained wrong state", slot, err)
				}
			}
			for attempts := 0; app.HeadlessNoticeOpen() && attempts < 32; attempts++ {
				if err := app.HeadlessActivate("notice"); err != nil {
					t.Fatal(err)
				}
			}
			if err := app.HeadlessKey("f9"); err != nil || app.Screen() != ui.ScreenMap {
				t.Fatal("actual F9", err)
			}
			quickAssertSample(t, missionAutosaveSampleNow(t, f), latest.Samples[0])
			missionAutosaveNext(f, latest)
			quickAssertSample(t, missionAutosaveSampleNow(t, f), latest.Samples[1])
			quickWriteProof(t, filepath.Join(store.Dir, "quick-save-2.sav"), emitted[4], latest)
			runSpellWitnessChild(t, filepath.Join(store.Dir, "quick-save-2.sav"), "AGAINROM_QUICK_LOAD_INPUT")
			for _, path := range []string{manual, filepath.Join(store.Dir, start.Name)} {
				if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, startRaw) {
					t.Fatal("manual/start changed", err)
				}
			}
			quickInstalledDelete(t, f, app, store)
			t.Logf("five distinct live-source F4 SAVs; oldest/newest slots [4,5,3]; six fresh processes; F9 and ordinary cold LOAD/next action match at width %d", width)
		})
	}
	t.Run("town", quickTownWitness)
}

func quickProofDirectory(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("AGAINROM_QUICK_SAVE_OUTPUT")
	if dir == "" {
		return t.TempDir()
	}
	if !filepath.IsAbs(dir) {
		t.Fatal("quick output must be absolute")
	}
	dir = filepath.Join(dir, filepath.Base(os.Getenv("AGAINROM_ASSETS")), strings.ReplaceAll(t.Name(), "/", "_"))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func quickWriteProof(t *testing.T, path string, raw []byte, proof missionAutosaveProof) {
	t.Helper()
	encoded, err := json.MarshalIndent(proof, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".proof.json", encoded, 0600); err != nil {
		t.Fatal(err)
	}
}

func quickAssertSample(t *testing.T, got, want missionAutosaveSample) {
	t.Helper()
	if got.State.Hash != want.State.Hash || !bytes.Equal(got.State.World, want.State.World) || got.State.Mission != want.State.Mission ||
		want.CampaignKnown && !reflect.DeepEqual(got.State.Campaign, want.State.Campaign) || got.Main != want.Main || got.Selected != want.Selected || got.Gold != want.Gold ||
		!reflect.DeepEqual(got.Won, want.Won) || !reflect.DeepEqual(got.View, want.View) {
		t.Fatalf("current source/restored state differ: World=%v hash=%x/%x view=%v", bytes.Equal(got.State.World, want.State.World), got.State.Hash, want.State.Hash, reflect.DeepEqual(got.View, want.View))
	}
}

func quickLoadCold(t *testing.T, path string) {
	t.Helper()
	encoded, err := os.ReadFile(path + ".proof.json")
	if err != nil {
		t.Fatal(err)
	}
	var proof missionAutosaveProof
	if err := json.Unmarshal(encoded, &proof); err != nil || len(proof.Samples) != 2 {
		t.Fatal("invalid proof", err)
	}
	store := SaveStore{Dir: t.TempDir()}
	for slot := range 3 {
		base := string(quickSaveBase(slot))
		for _, ext := range []string{".sav", quickOwnerExtension} {
			raw, err := os.ReadFile(filepath.Join(filepath.Dir(path), base+ext))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(store.Dir, base+ext), raw, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	raw, err := newestQuickSave(readQuickSaveSlots(store.Dir))
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(raw)) != proof.SHA256 {
		t.Fatal("cold newest ownership", err)
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("cold quickload")
	app.Layout(1024, 768)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if err := app.OpenMission(f.DirectNewGame(10)); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("f9"); err != nil || f.live == nil || !f.live.mission.resumed {
		t.Fatal("cold App F9", err)
	}
	quickAssertSample(t, missionAutosaveSampleNow(t, f), proof.Samples[0])
	missionAutosaveNext(f, proof)
	quickAssertSample(t, missionAutosaveSampleNow(t, f), proof.Samples[1])
	t.Logf("QUICK-F9-COLD-PASS pid=%d SHA256=%s", os.Getpid(), proof.SHA256)
}

func quickInstalledDelete(t *testing.T, f *FrontEnd, app *ui.App, store SaveStore) {
	t.Helper()
	for attempts := 0; app.HeadlessNoticeOpen() && attempts < 32; attempts++ {
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessKey("f3"); err != nil {
		t.Fatal(err)
	}
	rows := app.HeadlessRows()
	row := -1
	for i, item := range rows {
		if item.Text == "quicksave 2 - mission 10" {
			row = i
		}
	}
	if row < 0 {
		t.Fatal("quick row absent from ordinary chooser", rows)
	}
	if err := app.HeadlessKey("home"); err != nil {
		t.Fatal(err)
	}
	for range row {
		if err := app.HeadlessKey("down"); err != nil {
			t.Fatal(err)
		}
	}
	for _, key := range []string{"delete", "enter", "escape"} {
		if err := app.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() == ui.ScreenGameMenu {
		if err := app.HeadlessGameMenuAction("return"); err != nil {
			t.Fatal(err)
		}
	}
	for _, ext := range []string{".sav", quickOwnerExtension} {
		if _, err := os.Stat(filepath.Join(store.Dir, "quick-save-2") + ext); !os.IsNotExist(err) {
			t.Fatal("ordinary delete retained quick pair", err)
		}
	}
	if err := app.HeadlessKey("f4"); err != nil {
		t.Fatal(err)
	}
	if slots := readQuickSaveSlots(store.Dir); slots[1].sequence != 5 {
		t.Fatal("deleted slot did not recover with highest surviving sequence plus one", slots[1].sequence)
	}
}

func quickTownWitness(t *testing.T) {
	f := releaseFront(t)
	f.Carried = f.NextParty()
	f.Town.Open()
	f.arriveInTown()
	snapshot, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, _, err := f.playerMissionSave(snapshot, "town seed")
	if err != nil {
		t.Fatal(err)
	}
	dir := quickProofDirectory(t)
	if err := os.WriteFile(filepath.Join(dir, "manual.sav"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	g := releaseFront(t)
	app := g.App("quick town")
	app.Layout(640, 480)
	g.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	for _, key := range []string{"load", "enter"} {
		if err := app.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenTown {
		t.Fatal("town seed failed")
	}
	g.TownScreen().(*townScreen).CloseTip()
	before := g.Town.Gold()
	if err := app.HeadlessKey("f4"); err != nil {
		t.Fatal(err)
	}
	older, err := os.ReadFile(filepath.Join(dir, "quick-save-1.sav"))
	if err != nil {
		t.Fatal(err)
	}
	g.Town.gold += 77
	first := timedTownCapture(t, g)
	if err := app.HeadlessKey("f4"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "quick-save-2.sav")
	saved, err := os.ReadFile(path)
	if err != nil || bytes.Equal(saved, raw) || g.Town.Gold() != before+77 {
		t.Fatal("town current-state write", err)
	}
	g.Town.gold += 13
	if err := app.HeadlessKey("f9"); err != nil {
		t.Fatal(err)
	}
	g.TownScreen().(*townScreen).CloseTip()
	if got := timedTownCapture(t, g); !reflect.DeepEqual(got, first) {
		currentValueDiagnostics(t, "town", reflect.ValueOf(first), reflect.ValueOf(got))
		t.Fatal("town F9 lost current state")
	}
	if got, err := os.ReadFile(filepath.Join(dir, "quick-save-1.sav")); err != nil || !bytes.Equal(got, older) {
		t.Fatal("town newest replaced older bytes", err)
	}
	proof := timedTownProof{fmt.Sprintf("%x", sha256.Sum256(saved)), []timedTownSample{first, timedTownNext(t, g, app)}}
	var encoded bytes.Buffer
	if err := gob.NewEncoder(&encoded).Encode(proof); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".town-proof", encoded.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	runSpellWitnessChild(t, path, "AGAINROM_QUICK_TOWN_INPUT")
}
