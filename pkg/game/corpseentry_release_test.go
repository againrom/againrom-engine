package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseCorpseEntrySAVLoad(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	dir := t.TempDir()
	if root := os.Getenv("AGAINROM_CORPSE_ENTRY_OUT"); root != "" {
		if !filepath.IsAbs(root) {
			t.Fatal("corpse entry output must be absolute")
		}
		dir = filepath.Join(root, filepath.Base(f.Archives.Root))
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	store := SaveStore{Dir: filepath.Join(dir, "saves")}
	f.Options = OptionsStore{Path: filepath.Join(dir, "options.txt")}
	app := f.App("corpse entry SAV")
	app.Layout(1024, 768)
	app.SetCutscenes(nil)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return time.Unix(100, 0) })
	party := f.ChargenParty(ui.ChargenResult{Name: "Corpse entry", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 30, 30}})
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	id := f.live.mission.ids[0]
	body, _ := f.live.entity(id)
	f.live.view.Camera().CenterOn(float64(body.X*32), float64(body.Y*32))
	heard := observeReleaseAudio(t, f)
	attach := func() {
		f.live.view.SetAudio(heard, heard)
		f.live.view.SetSpeechAudio(heard)
	}
	attach()
	key := func(k string) {
		t.Helper()
		if err := app.HeadlessKey(k); err != nil {
			t.Fatal(err)
		}
	}
	step := func() {
		t.Helper()
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	key("0")
	if len(heard.plays) != 0 {
		t.Fatal("live entry or startup strikes voiced", heard.plays)
	}
	if err := f.live.world.HeadlessDamage(id, body.HP); err != nil {
		t.Fatal(err)
	}
	f.live.push()
	step()
	if len(heard.plays) != 1 || !strings.HasSuffix(heard.plays[0].src, "/die.wav") {
		t.Fatal("ordinary live fall lost its die cue", heard.plays)
	}
	body, _ = f.live.entity(id)
	if body.Decay != sim.DecayFallen {
		t.Fatal("live fall did not produce independent stage 1", body)
	}
	key("escape")
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveEdit(store.Dir, "corpse-entry", ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if state, open := app.HeadlessSaveState(); open && state.Confirmation {
		if err := app.HeadlessSaveAction("overwrite"); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenGameMenu {
		t.Fatal("named SAVE did not finish", app.Screen(), app.HeadlessMessage())
	}
	path := filepath.Join(store.Dir, "corpse-entry.sav")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	record := generatedActorRecord(t, &doc, id)
	stage, err := savedStructureValue(record, "Stage")
	if err != nil || stage != 1 {
		t.Fatal("source Stage bytes", stage, err)
	}
	hp, err := savedStructureValue(record, "Health")
	if err != nil || int16(hp) != int16(body.HP) {
		t.Fatal("source Health bytes", hp, body.HP, err)
	}
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	savedTick := file.Head.CounterA
	savedHash := f.live.world.Hash()
	choose := func() {
		t.Helper()
		if app.Screen() == ui.ScreenGameMenu {
			if err := app.HeadlessGameMenuAction("load"); err != nil {
				t.Fatal(err)
			}
		} else {
			key("f3")
		}
		key("home")
		found := false
		for i, row := range app.HeadlessRows() {
			if row.Text == "corpse-entry - mission 20" {
				for range i {
					key("down")
				}
				found = true
				break
			}
		}
		if !found {
			t.Fatal("ordinary LOAD chooser has no saved subject", app.HeadlessRows())
		}
	}
	load := func() {
		t.Helper()
		choose()
		key("enter")
		if app.Screen() != ui.ScreenMap {
			t.Fatal("ordinary LOAD failed", app.Screen(), app.HeadlessMessage())
		}
		attach()
	}
	for cycle := 0; cycle < 2; cycle++ {
		before := len(heard.plays)
		load()
		cold, ok := f.live.entity(id)
		if !ok || cold.Decay != sim.DecayStage(stage) || cold.HP != int32(int16(hp)) || f.live.world.Tick() != uint64(savedTick) || f.live.world.Hash() != savedHash {
			t.Fatal("source bytes -> cold World lost stage/health/ID/tick/hash", cycle, cold, f.live.world.Tick(), savedTick)
		}
		beforeSound := f.live.world.Hash()
		snapshot, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		beforeRaw, err := f.ExportCurrentSave(snapshot, label)
		if err != nil {
			t.Fatal(err)
		}
		if f.live.stopped {
			step()
		} else {
			key("0")
		}
		if len(heard.plays) != before+1 || !strings.HasSuffix(heard.plays[before].src, "/die.wav") {
			t.Fatal("each successful LOAD must emit once", cycle, heard.plays[before:])
		}
		if f.live.world.Hash() != beforeSound {
			t.Fatal("sound playback changed World hash", f.live.world.Tick(), savedTick, f.live.stopped)
		}
		snapshot, label, err = f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		afterRaw, err := f.ExportCurrentSave(snapshot, label)
		if err != nil || !bytes.Equal(beforeRaw, afterRaw) {
			t.Fatal("sound playback changed SAV bytes", err)
		}
		for range 3 {
			f.live.push()
			step()
		}
		if len(heard.plays) != before+1 {
			t.Fatal("routine projections replayed entry", heard.plays[before:])
		}
		choose()
		key("escape")
		step()
		if len(heard.plays) != before+1 {
			t.Fatal("cancelled chooser replayed entry")
		}
		choose()
		if err := os.WriteFile(path, []byte("broken save"), 0600); err != nil {
			t.Fatal(err)
		}
		key("enter")
		if app.Screen() != ui.ScreenLoad || app.HeadlessMessage() == "" {
			t.Fatal("failed SAV LOAD lost the chooser", app.Screen(), app.HeadlessMessage())
		}
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		key("escape")
		step()
		if len(heard.plays) != before+1 || f.live.world.Hash() != beforeSound {
			t.Fatal("failed SAV LOAD replayed entry or changed World")
		}
		f.live.tick()
		if f.live.world.Tick() != uint64(savedTick)+1 || f.live.world.Hash() == beforeSound {
			t.Fatal("next production tick did not change state")
		}
		late, lateLabel, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		lateRaw, err := f.ExportCurrentSave(late, lateLabel)
		if err != nil || bytes.Equal(lateRaw, beforeRaw) {
			t.Fatal("changed-state SAV loss control", err)
		}
		if unchanged, err := os.ReadFile(path); err != nil || !bytes.Equal(unchanged, raw) {
			t.Fatal("playback/tick changed the saved file", err)
		}
	}
	var coldProof []map[string]any
	for _, coldStage := range []uint32{1, 2} {
		changed, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		actor := generatedActorRecord(t, &changed, id)
		if err := savedStructureSetValue(actor, "Stage", coldStage); err != nil {
			t.Fatal(err)
		}
		coldHP := int32(0)
		if err := savedStructureSetValue(actor, "Health", uint32(uint16(coldHP))); err != nil {
			t.Fatal(err)
		}
		// A body past stage 1 has served its dying countdown (HERO-DWELL-065).
		if coldStage != 1 {
			if err := savedStructureSetValue(actor, "U6C", 0); err != nil {
				t.Fatal(err)
			}
		}
		coldRaw, err := sav.EncodeDocumentData(changed)
		if err != nil {
			t.Fatal(err)
		}
		cold := releaseFront(t)
		cold.SetDeterministicFrames(true)
		open, town, err := cold.RestoreOriginal(coldRaw)
		if err != nil || town {
			t.Fatal("fresh source-stage LOAD", coldStage, town, err)
		}
		coldApp := cold.App("cold corpse entry")
		coldApp.Layout(1024, 768)
		coldApp.SetCutscenes(nil)
		if err := coldApp.OpenMission(open); err != nil {
			t.Fatal(err)
		}
		entity, ok := cold.live.entity(id)
		if !ok || uint32(entity.Decay) != coldStage || entity.HP != coldHP {
			t.Fatal("ordinary Stage/Health lost to stale continuation", coldStage, entity)
		}
		coldHeard := observeReleaseAudio(t, cold)
		if cold.live.stopped {
			err = coldApp.HeadlessStep()
		} else {
			err = coldApp.HeadlessKey("0")
		}
		if err != nil {
			t.Fatal(err)
		}
		want := 0
		if coldStage == 1 {
			want = 1
		}
		if len(coldHeard.plays) != want {
			t.Fatal("fresh source-stage entry sound loss control", coldStage, coldHeard.plays)
		}
		if coldStage == 1 {
			if cold.live.world.Hash() != savedHash {
				t.Fatal("fresh LOAD changed the source World")
			}
			cold.live.tick()
			if cold.live.world.Hash() != f.live.world.Hash() {
				t.Fatal("fresh LOAD next production tick diverged")
			}
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("source-stage%d.sav", coldStage)), coldRaw, 0600); err != nil {
			t.Fatal(err)
		}
		coldProof = append(coldProof, map[string]any{"source_stage": coldStage, "source_health": coldHP, "restored_stage": entity.Decay, "cue_count": len(coldHeard.plays), "save_sha256": fmt.Sprintf("%x", sha256.Sum256(coldRaw))})
	}
	var cues []string
	for _, play := range heard.plays {
		cues = append(cues, play.src)
	}
	proof := map[string]any{"save_sha256": fmt.Sprintf("%x", sha256.Sum256(raw)), "actor": id, "source_stage": stage, "source_health": int16(hp), "saved_tick": savedTick, "world_hash": fmt.Sprintf("%x", savedHash), "heard": cues, "successful_loads": 2, "fresh_source_controls": coldProof, "native_audibility": "unmeasured"}
	b, err := json.MarshalIndent(proof, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "proof.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
}
