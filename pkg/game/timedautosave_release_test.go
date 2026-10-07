package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseTimedAutosaveCurrentStateAndOptions(t *testing.T) {
	if path := os.Getenv("AGAINROM_TIMED_TOWN_INPUT"); path != "" {
		timedTownCold(t, path)
		return
	}
	if path := os.Getenv("AGAINROM_TIMED_AUTOSAVE_INPUT"); path != "" {
		missionAutosaveCold(t, path)
		return
	}
	for _, width := range []int{640, 1024} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			dir := t.TempDir()
			if root := os.Getenv("AGAINROM_TIMED_AUTOSAVE_OUTPUT"); root != "" {
				if !filepath.IsAbs(root) {
					t.Fatal("timed output must be absolute")
				}
				dir = filepath.Join(root, strings.ReplaceAll(t.Name(), "/", "_"))
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			store := SaveStore{Dir: filepath.Join(dir, "saves")}
			f.Options = OptionsStore{Path: filepath.Join(dir, "options.txt")}
			now := time.Unix(100, 0)
			app := f.App("timed current state")
			app.Layout(width, width*3/4)
			path := filepath.Join(store.Dir, "timed-autosave-1.sav")
			var sourceBefore *missionAutosaveSample
			f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time {
				if !now.Before(time.Unix(100, 0).Add(6*time.Minute)) && app.Screen() == ui.ScreenMap && f.live != nil && !f.live.view.NoticeOpen() {
					if _, err := os.Stat(path); os.IsNotExist(err) {
						sample := missionAutosaveSampleNow(t, f)
						sourceBefore = &sample
					}
				}
				return now
			})
			if err := app.OpenMission(f.DirectNewGame(10)); err != nil {
				t.Fatal(err)
			}
			start := missionAutosaveRow(t, f, store, 10)
			startRaw, err := store.Read(start.Name)
			if err != nil {
				t.Fatal(err)
			}
			key := func(name string) {
				t.Helper()
				if err := app.HeadlessKey(name); err != nil {
					t.Fatal(err)
				}
			}
			action := func(name string) {
				t.Helper()
				if err := app.HeadlessGameMenuAction(name); err != nil {
					t.Fatal(err)
				}
			}
			key("0")
			key("escape")
			action("game-options")
			rows := app.HeadlessRows()
			font := f.Font.Value()
			for _, row := range rows[len(rows)-4 : len(rows)-2] {
				if font == nil || font.Advance(row.Text) > 180 || strings.Contains(row.Text, "?") {
					t.Fatal("installed timed caption does not fit", row.Text)
				}
			}
			pix, err := app.HeadlessGameOptionsFrame()
			if err != nil {
				t.Fatal("installed options frame", err)
			}
			var picture bytes.Buffer
			if err := png.Encode(&picture, pix); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "options.png"), picture.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
			action("timed-autosave")
			action("autosave-minutes")
			action("options-cancel")
			prefs, err := f.Options.timedAutosave()
			if err != nil || !prefs.Enabled || prefs.Minutes != 5 {
				t.Fatal("installed Cancel persisted", prefs, err)
			}
			action("game-options")
			action("autosave-minutes")
			action("page-return")
			prefs, err = f.Options.timedAutosave()
			if err != nil || !prefs.Enabled || prefs.Minutes != 6 {
				t.Fatal("installed OK persistence", prefs, err)
			}
			now = now.Add(6 * time.Minute)
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("options modal admitted expiry", err)
			}
			for range 16 {
				f.live.tick()
			}
			action("return")
			for attempts := 0; f.live.view.NoticeOpen() && attempts < 32; attempts++ {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatal("dialogue admitted timed write", err)
				}
				if err := app.HeadlessActivate("notice"); err != nil {
					t.Fatal("dialogue resume", err)
				}
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal("missing-write loss control", err)
			}
			file, err := sav.Open(raw)
			if err != nil || file.Head.CounterA != 16 || bytes.Equal(raw, startRaw) {
				t.Fatal("timed save retained early state", err)
			}
			if sourceBefore == nil {
				t.Fatal("no independent pre-publication source sample")
			}
			first := *sourceBefore
			afterSave := missionAutosaveSampleNow(t, f)
			if first.State.Hash != afterSave.State.Hash || !bytes.Equal(first.State.World, afterSave.State.World) || !reflect.DeepEqual(first.View, afterSave.View) {
				t.Fatal("publication changed the source state")
			}
			proof := missionAutosaveProof{SHA256: fmt.Sprintf("%x", sha256.Sum256(raw)), Label: "timed autosave 1", Samples: []missionAutosaveSample{first}}
			for _, entity := range f.live.world.Entities() {
				if entity.Owner == sim.SelfSlot && entity.Alive() {
					proof.Actor, proof.X, proof.Y = uint32(entity.ID), int(entity.X)+1, int(entity.Y)
					break
				}
			}
			if proof.Actor == 0 {
				t.Fatal("no actor")
			}
			missionAutosaveNext(f, proof)
			proof.Samples = append(proof.Samples, missionAutosaveSampleNow(t, f))
			if bytes.Equal(first.State.World, proof.Samples[1].State.World) || f.live.world.Tick() != 32 {
				t.Fatal("late-state control did not advance")
			}
			late, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			lateRaw, _, err := f.playerMissionSave(late, "late-state control")
			if err != nil || bytes.Equal(raw, lateRaw) {
				t.Fatal("late-state output indistinguishable", err)
			}
			if after, err := os.ReadFile(path); err != nil || !bytes.Equal(after, raw) {
				t.Fatal("live action changed saved bytes", err)
			}
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
			runSpellWitnessChild(t, path, "AGAINROM_TIMED_AUTOSAVE_INPUT")
			timedMapRecovery(t, f, app, store, &now)
			if after, err := store.Read(start.Name); err != nil || !bytes.Equal(after, startRaw) {
				t.Fatal("timed save changed mission-start file", err)
			}
			t.Logf("%dx%d: current tick16 + next-action tick32 cold LOAD, SAV %s", width, width*3/4, proof.SHA256)
		})
	}
	t.Run("town", timedTownWitness)
}

func timedMapRecovery(t *testing.T, f *FrontEnd, app *ui.App, store SaveStore, now *time.Time) {
	t.Helper()
	protected := filepath.Join(store.Dir, "timed-autosave-2.sav")
	foreign := []byte("protected foreign slot")
	if err := os.WriteFile(protected, foreign, 0600); err != nil {
		t.Fatal(err)
	}
	before := missionAutosaveSampleNow(t, f)
	sourceTick := f.live.world.Tick()
	*now = now.Add(6 * time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Dir, "timed-autosave-3.sav")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("protected slot stopped the installed healthy slot", err)
	}
	if after := missionAutosaveSampleNow(t, f); !reflect.DeepEqual(before, after) {
		t.Fatal("recovery write changed source")
	}
	proof := missionAutosaveProof{SHA256: fmt.Sprintf("%x", sha256.Sum256(raw)), Label: "timed autosave 3", Samples: []missionAutosaveSample{before}}
	for _, entity := range f.live.world.Entities() {
		if entity.Owner == sim.SelfSlot && entity.Alive() {
			proof.Actor, proof.X, proof.Y = uint32(entity.ID), int(entity.X)+1, int(entity.Y)
			break
		}
	}
	if proof.Actor == 0 {
		t.Fatal("no recovery next-action actor")
	}
	missionAutosaveNext(f, proof)
	proof.Samples = append(proof.Samples, missionAutosaveSampleNow(t, f))
	if bytes.Equal(before.State.World, proof.Samples[1].State.World) {
		t.Fatal("recovery next action did not change state")
	}
	encoded, err := json.MarshalIndent(proof, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	proofPath := filepath.Join(filepath.Dir(store.Dir), "recovery-source.sav")
	if err := os.WriteFile(proofPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(proofPath+".proof.json", encoded, 0600); err != nil {
		t.Fatal(err)
	}
	runSpellWitnessChild(t, proofPath, "AGAINROM_TIMED_AUTOSAVE_INPUT")
	if err := app.HeadlessKey("f3"); err != nil {
		t.Fatal(err)
	}
	want := -1
	for i, row := range app.HeadlessRows() {
		if row.Text == "timed autosave 3 - mission 10" {
			want = i
		}
	}
	if want < 0 {
		t.Fatal("recovery SAV absent from ordinary chooser")
	}
	if err := app.HeadlessKey("home"); err != nil {
		t.Fatal(err)
	}
	for range want {
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
	if app.Screen() != ui.ScreenMap {
		t.Fatal("delete did not return to map")
	}
	base := strings.TrimSuffix(path, ".sav")
	for _, ext := range []string{".sav", timedOwnerExtension} {
		if _, err := os.Stat(base + ext); !os.IsNotExist(err) {
			t.Fatal("ordinary chooser retained verified pair", ext, err)
		}
	}
	*now = now.Add(6 * time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(path)
	if err != nil {
		t.Fatal("ordinary delete stopped installed timed recovery", err)
	}
	owner, err := os.ReadFile(base + timedOwnerExtension)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validateTimedSavePair(2, raw, owner); err != nil {
		t.Fatal(err)
	}
	file, err := sav.Open(raw)
	if err != nil || file.Head.CounterA != uint32(sourceTick+16) {
		t.Fatal("post-delete save lost current state", err)
	}
	if got, err := os.ReadFile(protected); err != nil || !bytes.Equal(got, foreign) {
		t.Fatal("foreign slot changed", err)
	}
	if _, err := os.Stat(strings.TrimSuffix(protected, ".sav") + timedOwnerExtension); !os.IsNotExist(err) {
		t.Fatal("foreign slot was adopted", err)
	}
	t.Logf("recovery: protected slot2, slot3 source tick%d/next tick%d, ordinary Delete removes verified pair and timed write resumes", sourceTick, sourceTick+16)
}

type timedTownSample struct {
	Gold, Chapter, Offered int
	Won, Available         []int
	Party                  []mapload.PartyMember
}

type timedTownProof struct {
	SHA256  string
	Samples []timedTownSample
}

func timedTownCapture(t *testing.T, f *FrontEnd) timedTownSample {
	t.Helper()
	snapshot, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	return timedTownSample{snapshot.Gold, snapshot.MainMission, snapshot.Offered, snapshot.Won, snapshot.Available, snapshot.Party}
}

func timedTownNext(t *testing.T, f *FrontEnd, app *ui.App) timedTownSample {
	t.Helper()
	if err := app.HeadlessActivate("SHOP"); err != nil {
		t.Fatal(err)
	}
	for attempts := 0; f.townUI.room == roomTalk && attempts < 128; attempts++ {
		if err := app.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
	}
	if f.townUI.room != roomShop {
		t.Fatal("shop action did not settle")
	}
	f.townUI.CloseTip()
	before := timedTownCapture(t, f)
	worn, pack := currentTownMember(t, f, f.Carried[0].ID)
	if worn[0] == 0 {
		t.Fatal("next action lacks an equipped weapon")
	}
	x, y, err := app.HeadlessShopPoint("doll", 1)
	if err != nil {
		t.Fatal(err)
	}
	tx, ty, err := app.HeadlessShopPoint("pack", 0)
	if err != nil {
		t.Fatal(err)
	}
	portraitInputDrag(t, app, image.Pt(x, y), image.Pt(tx, ty))
	afterWorn, afterPack := currentTownMember(t, f, f.Carried[0].ID)
	count := func(v []uint16) int {
		n := 0
		for _, code := range v {
			if code == worn[0] {
				n++
			}
		}
		return n
	}
	after := timedTownCapture(t, f)
	if afterWorn[0] != 0 || count(afterPack) != count(pack)+1 || after.Gold != before.Gold || reflect.DeepEqual(before.Party, after.Party) {
		t.Fatal("next pointer action did not unequip without payment")
	}
	return after
}

func timedTownWitness(t *testing.T) {
	f := currentTown(t, nil, nil)
	raw := currentTownSave(t, f)
	dir := t.TempDir()
	if root := os.Getenv("AGAINROM_TIMED_AUTOSAVE_OUTPUT"); root != "" {
		if !filepath.IsAbs(root) {
			t.Fatal("explicit timed output required")
		}
		dir = filepath.Join(root, strings.ReplaceAll(t.Name(), "/", "_"))
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	manual := filepath.Join(dir, "town.sav")
	if err := os.WriteFile(manual, raw, 0600); err != nil {
		t.Fatal(err)
	}
	g := releaseFront(t)
	g.Options = OptionsStore{}
	app := g.App("timed town")
	app.Layout(640, 480)
	now := time.Unix(100, 0)
	g.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, func() time.Time { return now })
	for _, action := range []string{"load game", "@first"} {
		if err := app.HeadlessActivate(action); err != nil {
			t.Fatal(err)
		}
	}
	g.TownScreen().(*townScreen).CloseTip()
	if app.Screen() != ui.ScreenTown {
		t.Fatal("town LOAD refused")
	}
	oldGold := g.Town.Gold()
	g.Town.gold += 77
	source := timedTownCapture(t, g)
	now = now.Add(5 * time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	app.FlushBackground()
	path := filepath.Join(dir, "timed-autosave-1.sav")
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("town missing-write control", err)
	}
	if bytes.Equal(raw, saved) || source.Gold != oldGold+77 || !reflect.DeepEqual(source, timedTownCapture(t, g)) {
		t.Fatal("town writer lost current gold or changed source")
	}
	proof := timedTownProof{fmt.Sprintf("%x", sha256.Sum256(saved)), []timedTownSample{source, timedTownNext(t, g, app)}}
	var encoded bytes.Buffer
	if err := gob.NewEncoder(&encoded).Encode(proof); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".town-proof", encoded.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	runSpellWitnessChild(t, path, "AGAINROM_TIMED_TOWN_INPUT")
	if got, err := os.ReadFile(manual); err != nil || !bytes.Equal(got, raw) {
		t.Fatal("manual town save changed", err)
	}
	if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, saved) {
		t.Fatal("next town action changed timed bytes", err)
	}
	t.Logf("town gold %d -> %d, current source/cold LOAD and next pointer unequip match", oldGold, source.Gold)
}

func timedTownCold(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := os.ReadFile(path + ".town-proof")
	if err != nil {
		t.Fatal(err)
	}
	var proof timedTownProof
	if err := gob.NewDecoder(bytes.NewReader(encoded)).Decode(&proof); err != nil || len(proof.Samples) != 2 || proof.SHA256 != fmt.Sprintf("%x", sha256.Sum256(raw)) {
		t.Fatal("town proof/hash", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "timed-autosave-1.sav"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	f.Options = OptionsStore{}
	app := f.App("cold timed town")
	app.Layout(640, 480)
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	for _, action := range []string{"load game", "@first"} {
		if err := app.HeadlessActivate(action); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenTown {
		t.Fatal("cold town LOAD refused")
	}
	f.TownScreen().(*townScreen).CloseTip()
	if got := timedTownCapture(t, f); !reflect.DeepEqual(got, proof.Samples[0]) {
		t.Fatal("cold town differs from independent current source")
	}
	if got := timedTownNext(t, f, app); !reflect.DeepEqual(got, proof.Samples[1]) {
		t.Fatal("cold town next pointer action differs")
	}
	t.Logf("TIMED-TOWN-COLD-PASS pid=%d gold=%d", os.Getpid(), proof.Samples[0].Gold)
}
