package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

type failTimedPublish struct {
	osNamedSaveFiles
	failAt, calls int
}

func (f *failTimedPublish) Publish(a, b string) (savePublishResult, error) {
	f.calls++
	if f.calls == f.failAt {
		return savePublishResult{}, errors.New("injected timed publication failure")
	}
	return f.osNamedSaveFiles.Publish(a, b)
}

func (f *failTimedPublish) Replace(a, b string) error {
	f.calls++
	if f.calls == f.failAt {
		return errors.New("injected timed replacement failure")
	}
	return f.osNamedSaveFiles.Replace(a, b)
}

func TestTimedAutosaveClockExpiry(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	store := SaveStore{Dir: t.TempDir()}
	now := time.Unix(100, 0)
	app := f.App("timed autosave")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return now })
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	hash := f.live.world.Hash()
	now = now.Add(5*time.Minute - time.Nanosecond)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(store.Dir, "timed-autosave-1.sav")); !os.IsNotExist(err) {
		t.Fatal("early save", err)
	}
	if f.live.world.Hash() != hash {
		t.Fatal("clock changed the paused simulation hash")
	}
	now = now.Add(time.Nanosecond)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(store.Dir, "timed-autosave-1.sav")); err != nil {
		t.Fatal("expired wall clock did not publish timed SAV", err)
	}
	if f.live.world.Hash() != hash {
		t.Fatal("saving changed the simulation hash")
	}
}

func TestTimedAutosaveDeletedSlotRecovery(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	store := SaveStore{Dir: t.TempDir()}
	now := time.Unix(100, 0)
	app := f.App("delete timed slot")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return now })
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		now = now.Add(5 * time.Minute)
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessKey("f3"); err != nil {
		t.Fatal(err)
	}
	rows := app.HeadlessRows()
	want := -1
	for i, row := range rows {
		if row.Text == "timed autosave 2 - mission 10" {
			want = i
		}
	}
	if want < 0 {
		t.Fatal("missing slot in ordinary LOAD chooser", rows)
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
		t.Fatal("delete did not return to the live map", app.Screen())
	}
	base := filepath.Join(store.Dir, string(timedSaveBase(1)))
	for _, ext := range []string{".sav", timedOwnerExtension} {
		if _, err := os.Stat(base + ext); !os.IsNotExist(err) {
			t.Errorf("owned delete retained %s: %v", ext, err)
		}
	}
	for range 6 {
		now = now.Add(5 * time.Minute)
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	for slot := range 3 {
		base := filepath.Join(store.Dir, string(timedSaveBase(slot)))
		raw, err := os.ReadFile(base + ".sav")
		if err != nil {
			t.Fatal("rotation did not recover after player deletion", err)
		}
		owner, err := os.ReadFile(base + timedOwnerExtension)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := validateTimedSavePair(slot, raw, owner); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTimedAutosaveProtectedSlotRecovery(t *testing.T) {
	for _, kind := range []string{"foreign", "corrupt", "orphan", "mismatched"} {
		t.Run(kind, func(t *testing.T) {
			f := currentPoolFixtureFront(t, 91, 92)
			store := SaveStore{Dir: t.TempDir()}
			now := time.Unix(100, 0)
			app := f.App("protected slot")
			f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return now })
			if err := app.OpenMission(f.MissionOpener(10)); err != nil {
				t.Fatal(err)
			}
			base := filepath.Join(store.Dir, string(timedSaveBase(1)))
			raw := []byte("foreign bytes")
			owner, err := json.Marshal(timedSaveOwner{"againrom-timed-sav", 1, 1, fmt.Sprintf("%x", sha256.Sum256(raw))})
			if err != nil {
				t.Fatal(err)
			}
			if kind != "orphan" {
				if err := os.WriteFile(base+".sav", raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "mismatched" {
				owner = []byte("foreign companion")
			}
			if kind != "foreign" {
				if err := os.WriteFile(base+timedOwnerExtension, owner, 0600); err != nil {
					t.Fatal(err)
				}
			}
			for sequence := uint64(1); sequence <= 7; sequence++ {
				now = now.Add(5 * time.Minute)
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
				slot := 0
				if sequence%2 == 0 {
					slot = 2
				}
				path := filepath.Join(store.Dir, string(timedSaveBase(slot)))
				data, err := os.ReadFile(path + ".sav")
				if err != nil {
					t.Fatal("healthy slot did not receive current save", sequence, err)
				}
				marker, err := os.ReadFile(path + timedOwnerExtension)
				if err != nil {
					t.Fatal(err)
				}
				if got, err := validateTimedSavePair(slot, data, marker); err != nil || got != sequence {
					t.Fatal("rotation did not alternate slots 1 and 3", got, sequence, err)
				}
			}
			if kind != "orphan" {
				if got, err := os.ReadFile(base + ".sav"); err != nil || !bytes.Equal(got, raw) {
					t.Fatal("protected SAV changed", err)
				}
			}
			if kind != "foreign" {
				if got, err := os.ReadFile(base + timedOwnerExtension); err != nil || !bytes.Equal(got, owner) {
					t.Fatal("protected companion changed", err)
				}
			}
		})
	}
}

func TestTimedAutosaveDraftPersistenceAndDeferral(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	dir := t.TempDir()
	f.Options = OptionsStore{Path: filepath.Join(dir, "options.txt")}
	store := SaveStore{Dir: filepath.Join(dir, "saves")}
	now := time.Unix(100, 0)
	app := f.App("autosave options")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return now })
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
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
	key("escape")
	action("game-options")
	action("timed-autosave")
	action("autosave-minutes")
	action("options-cancel")
	v, err := f.Options.timedAutosave()
	if err != nil || !v.Enabled || v.Minutes != 5 {
		t.Fatal("Cancel persisted draft", v, err)
	}
	action("game-options")
	action("timed-autosave")
	action("autosave-minutes")
	action("page-return")
	v, err = f.Options.timedAutosave()
	if err != nil || v.Enabled || v.Minutes != 6 {
		t.Fatal("OK did not persist both settings", v, err)
	}
	now = now.Add(time.Hour)
	action("return")
	if _, err := os.Stat(filepath.Join(store.Dir, "timed-autosave-1.sav")); !os.IsNotExist(err) {
		t.Fatal("disabled timer wrote", err)
	}
	key("escape")
	action("game-options")
	action("timed-autosave")
	action("page-return")
	now = now.Add(6 * time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(store.Dir, "timed-autosave-1.sav")); !os.IsNotExist(err) {
		t.Fatal("menu did not defer timer", err)
	}
	for range 9 {
		f.live.tick()
	}
	action("return")
	raw, err := store.Read("timed-autosave-1.sav")
	if err != nil {
		t.Fatal("resume did not write", err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil || doc.Head.CounterA != 9 {
		t.Fatal("deferred save captured stale state", doc.Head, err)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(store.Dir, "timed-autosave-2.sav")); !os.IsNotExist(err) {
		t.Fatal("catch-up burst", err)
	}
	g := currentPoolFixtureFront(t, 91, 92)
	g.Options = f.Options
	fresh, err := g.Options.timedAutosave()
	if err != nil || !fresh.Enabled || fresh.Minutes != 6 {
		t.Fatal("fresh process preferences", fresh, err)
	}
}

func TestTimedAutosaveOwnedRotationAndRollback(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	app := f.App("owned slots")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, _, err := f.playerMissionSave(s, "timed autosave 1")
	if err != nil {
		t.Fatal(err)
	}
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "new", true: "overwrite"}[existing], func(t *testing.T) {
			store := SaveStore{Dir: t.TempDir()}
			manual := filepath.Join(store.Dir, "manual.sav")
			if err := os.WriteFile(manual, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if existing {
				if err := writeTimedSave(store, nil, 0, 1, raw, nil); err != nil {
					t.Fatal(err)
				}
			}
			files := &failTimedPublish{failAt: 2}
			if err := writeTimedSave(store, nil, 0, 2, raw, files); err == nil {
				t.Fatal("second publication did not fail")
			}
			path := filepath.Join(store.Dir, string(timedSaveBase(0)))
			if existing {
				owner, err := os.ReadFile(path + timedOwnerExtension)
				if err != nil {
					t.Fatal(err)
				}
				if seq, err := validateTimedSavePair(0, raw, owner); err != nil || seq != 1 {
					t.Fatal("overwrite did not retain old pair", seq, err)
				}
			} else {
				if _, err := os.Stat(path + ".sav"); !os.IsNotExist(err) {
					t.Fatal("partial new save", err)
				}
			}
			if got, err := os.ReadFile(manual); err != nil || !bytes.Equal(got, raw) {
				t.Fatal("manual file changed", err)
			}
			for seq := uint64(1); seq <= 7; seq++ {
				if existing && seq == 1 {
					continue
				}
				slot := int((seq - 1) % 3)
				if err := writeTimedSave(store, nil, slot, seq, raw, nil); err != nil {
					t.Fatal(err)
				}
			}
			if slot, seq, err := timedSavePosition(store.Dir); err != nil || slot != 1 || seq != 8 {
				t.Fatal("rotation", slot, seq, err)
			}
			if err := writeTimedSave(store, nil, 0, 7, raw, nil); err == nil {
				t.Fatal("stale rotation replaced a newer slot")
			}
			if rows, err := store.List(); err != nil || len(rows) != 4 {
				t.Fatal("not three separate slots plus manual", rows, err)
			}
			entries, err := os.ReadDir(store.Dir)
			if err != nil || len(entries) != 7 {
				t.Fatal("temporary or extra slot leaked", entries, err)
			}
		})
	}
	for _, kind := range []string{"unowned", "corrupt", "changed owner"} {
		t.Run(kind, func(t *testing.T) {
			store := SaveStore{Dir: t.TempDir()}
			path := filepath.Join(store.Dir, string(timedSaveBase(0)))
			if kind != "unowned" {
				if err := writeTimedSave(store, nil, 0, 1, raw, nil); err != nil {
					t.Fatal(err)
				}
			}
			data := raw
			if kind == "corrupt" {
				data = []byte("broken")
				owner, err := json.Marshal(timedSaveOwner{"againrom-timed-sav", 0, 1, fmt.Sprintf("%x", sha256.Sum256(data))})
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path+timedOwnerExtension, owner, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(path+".sav", data, 0600); err != nil {
				t.Fatal(err)
			}
			if kind == "changed owner" {
				if err := os.WriteFile(path+timedOwnerExtension, []byte("unowned"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := writeTimedSave(store, nil, 0, 2, raw, nil); err == nil {
				t.Fatal("replaced protected slot")
			}
			if got, err := os.ReadFile(path + ".sav"); err != nil || !bytes.Equal(data, got) {
				t.Fatal("protected SAV changed", err)
			}
		})
	}
}

func TestTimedAutosavePreferenceBoundsAndRawMap(t *testing.T) {
	dir := t.TempDir()
	options := OptionsStore{Path: filepath.Join(dir, "options.txt")}
	for _, text := range []string{"", "TimedAutosaveMinutes=0\n", "TimedAutosaveMinutes=-1\n", "TimedAutosaveMinutes=99999999999999999999\n"} {
		if err := os.WriteFile(options.Path, []byte(text+"Keep=untouched\n"), 0600); err != nil {
			t.Fatal(err)
		}
		v, err := options.timedAutosave()
		if err != nil || v != (ui.TimedAutosaveSettings{Enabled: true, Minutes: 5}) {
			t.Fatal("old preference defaults", v, err)
		}
	}
	before, err := os.ReadFile(options.Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, minutes := range []int{0, -1, ui.MaxAutosaveMinutes + 1} {
		if err := options.setTimedAutosave(ui.TimedAutosaveSettings{Enabled: true, Minutes: minutes}); err == nil {
			t.Fatal("accepted invalid interval", minutes)
		}
	}
	if after, err := os.ReadFile(options.Path); err != nil || !bytes.Equal(before, after) {
		t.Fatal("invalid interval changed preferences", err)
	}
	if err := options.setTimedAutosave(ui.TimedAutosaveSettings{Enabled: true, Minutes: ui.MaxAutosaveMinutes}); err != nil {
		t.Fatal(err)
	}
	if v, err := options.timedAutosave(); err != nil || v.Minutes != ui.MaxAutosaveMinutes {
		t.Fatal("largest complete minute overflowed", v, err)
	}
	if data, err := os.ReadFile(options.Path); err != nil || !strings.Contains(string(data), "Keep=untouched") {
		t.Fatal("unknown preference lost", err)
	}
	f := currentPoolFixtureFront(t, 91, 92)
	g := currentPoolFixtureFront(t, 91, 92)
	app := f.App("unowned map")
	store := SaveStore{Dir: t.TempDir()}
	now := time.Unix(100, 0)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return now })
	if err := app.OpenMission(g.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	now = now.Add(24 * time.Hour)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if rows, err := store.List(); err != nil || len(rows) != 0 {
		t.Fatal("raw foreign viewer wrote a SAV", rows, err)
	}
}

func TestTimedAutosaveFollowsManualSaveDestination(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	store := SaveStore{Dir: t.TempDir()}
	other := t.TempDir()
	now := time.Unix(100, 0)
	app := f.App("manual destination")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return now })
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	start, err := store.Read("game0000.sav")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveEdit(other, "manual", ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	manual, err := os.ReadFile(filepath.Join(other, "manual.sav"))
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(24 * time.Hour)
	if err := app.HeadlessGameMenuAction("return"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(other, "timed-autosave-1.sav")); err != nil {
		t.Fatal("timed save did not follow successful manual SAVE", err)
	}
	if _, err := os.Stat(filepath.Join(store.Dir, "timed-autosave-1.sav")); !os.IsNotExist(err) {
		t.Fatal("timed save used old destination", err)
	}
	if after, err := os.ReadFile(filepath.Join(other, "manual.sav")); err != nil || !bytes.Equal(after, manual) {
		t.Fatal("timed save changed manual", err)
	}
	if after, err := store.Read("game0000.sav"); err != nil || !bytes.Equal(after, start) {
		t.Fatal("timed save changed start SAV", err)
	}
}

func TestTimedAutosaveVisibleFailureRetryAndLoadReset(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	store := SaveStore{Dir: t.TempDir()}
	now := time.Unix(100, 0)
	app := f.App("retry and reset")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return now })
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Dir, "timed-autosave-1.sav")
	for slot := range 3 {
		if err := os.WriteFile(filepath.Join(store.Dir, string(timedSaveBase(slot))+".sav"), []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	now = now.Add(5 * time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fmt.Sprint(f.live.view.MessageLines()), "Timed autosave failed") {
		t.Fatal("write failure invisible", f.live.view.MessageLines())
	}
	for slot := range 3 {
		blocked := filepath.Join(store.Dir, string(timedSaveBase(slot))+".sav")
		if got, err := os.ReadFile(blocked); err != nil || string(got) != "keep" {
			t.Fatal("all-blocked failure changed protected bytes", err)
		}
		if err := os.Remove(blocked); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("unbounded immediate retry", err)
	}
	now = now.Add(time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("retry did not recover", err)
	}
	now = now.Add(4 * time.Minute)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("load"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("autosave start mission 10 - mission 10"); err != nil || app.Screen() != ui.ScreenMap {
		t.Fatal("accepted LOAD", err, app.Screen())
	}
	now = now.Add(time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(store.Dir, "timed-autosave-2.sav")); !os.IsNotExist(err) {
		t.Fatal("LOAD retained old deadline", err)
	}
	now = now.Add(4 * time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(store.Dir, "timed-autosave-2.sav")); err != nil {
		t.Fatal("LOAD did not start fresh deadline", err)
	}
	now = now.Add(4 * time.Minute)
	if err := app.OpenMission(f.DirectNewGame(10)); err != nil {
		t.Fatal("accepted new game", err)
	}
	now = now.Add(time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(store.Dir, "timed-autosave-3.sav")); !os.IsNotExist(err) {
		t.Fatal("NEW GAME retained old deadline", err)
	}
	now = now.Add(4 * time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(store.Dir, "timed-autosave-3.sav")); err != nil {
		t.Fatal("NEW GAME did not start fresh deadline", err)
	}
}
