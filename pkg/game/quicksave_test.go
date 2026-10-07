package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/pkg/ui"
)

func quickFixtureRaw(t *testing.T) []byte {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	app := f.App("quick slots")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, _, err := f.playerMissionSave(s, "quick state")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestQuickSaveOldestNewestAndStableTies(t *testing.T) {
	raw := quickFixtureRaw(t)
	for _, sequences := range [][3]uint64{{10, 1, 100}, {8, 8, 8}, {1, 2, 3}} {
		t.Run(fmt.Sprint(sequences), func(t *testing.T) {
			store := SaveStore{Dir: t.TempDir()}
			for slot, seq := range sequences {
				if err := writeQuickSave(store, nil, slot, seq, raw, nil); err != nil {
					t.Fatal(err)
				}
			}
			slots := readQuickSaveSlots(store.Dir)
			wantSlot, wantNext := 0, sequences[0]+1
			if sequences == [3]uint64{10, 1, 100} {
				wantSlot, wantNext = 1, 101
			}
			if sequences == [3]uint64{1, 2, 3} {
				wantNext = 4
			}
			if slot, next, err := quickSavePosition(slots); err != nil || slot != wantSlot || next != wantNext {
				t.Fatal("oldest", slot, next, err)
			}
			for slot := range slots {
				slots[slot].raw = []byte{byte(slot)}
			}
			wantNewest := 2
			if sequences[0] == sequences[1] {
				wantNewest = 0
			}
			if got, err := newestQuickSave(slots); err != nil || !bytes.Equal(got, []byte{byte(wantNewest)}) {
				t.Fatal("newest/tie", got, err)
			}
		})
	}
	slots := [3]quickSaveSlot{{sequence: 20}, {free: true}, {sequence: 5}}
	if slot, seq, err := quickSavePosition(slots); err != nil || slot != 1 || seq != 21 {
		t.Fatal("free slot first", slot, seq, err)
	}
	slots[0].sequence = math.MaxUint64
	if _, _, err := quickSavePosition(slots); err == nil {
		t.Fatal("sequence wrapped")
	}
	if _, err := newestQuickSave([3]quickSaveSlot{}); err == nil {
		t.Fatal("unverified filename fallback")
	}
}

func TestQuickSaveProtectedSlotsAndBoundedRecovery(t *testing.T) {
	raw := quickFixtureRaw(t)
	for _, kind := range []string{"foreign", "corrupt", "orphan", "mismatch"} {
		t.Run(kind, func(t *testing.T) {
			store := SaveStore{Dir: t.TempDir()}
			base := filepath.Join(store.Dir, string(quickSaveBase(1)))
			data := []byte("protected bytes")
			owner, _ := json.Marshal(quickSaveOwner{"againrom-quick-sav", 1, 500, fmt.Sprintf("%x", sha256.Sum256(data))})
			if kind != "orphan" {
				if err := os.WriteFile(base+".sav", data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "mismatch" {
				owner = []byte("foreign owner")
			}
			if kind != "foreign" {
				if err := os.WriteFile(base+quickOwnerExtension, owner, 0600); err != nil {
					t.Fatal(err)
				}
			}
			for seq := uint64(1); seq <= 7; seq++ {
				slot, next, err := quickSavePosition(readQuickSaveSlots(store.Dir))
				want := 0
				if seq%2 == 0 {
					want = 2
				}
				if err != nil || slot != want || next != seq {
					t.Fatal("healthy rotation", slot, next, seq, err)
				}
				if err := writeQuickSave(store, nil, slot, next, raw, nil); err != nil {
					t.Fatal(err)
				}
			}
			if kind != "orphan" {
				if got, err := os.ReadFile(base + ".sav"); err != nil || !bytes.Equal(got, data) {
					t.Fatal("protected SAV changed", err)
				}
			}
			if kind != "foreign" {
				if got, err := os.ReadFile(base + quickOwnerExtension); err != nil || !bytes.Equal(got, owner) {
					t.Fatal("protected owner changed", err)
				}
			}
			for _, slot := range []int{0, 2} {
				if err := os.WriteFile(filepath.Join(store.Dir, string(quickSaveBase(slot)))+quickOwnerExtension, []byte("invalid"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, _, err := quickSavePosition(readQuickSaveSlots(store.Dir)); err == nil {
				t.Fatal("all blocked accepted")
			}
		})
	}
}

type failQuickDelete struct{ osNamedSaveFiles }

func (f failQuickDelete) Remove(path string) error {
	if filepath.Ext(path) == quickOwnerExtension {
		return errors.New("injected quick companion deletion failure")
	}
	return f.osNamedSaveFiles.Remove(path)
}

func TestQuickSavePublicationDeleteAndCapturedBytes(t *testing.T) {
	raw := quickFixtureRaw(t)
	for _, kind := range []string{"new failure", "overwrite failure", "changed owner", "foreign owner", "delete failure", "delete success"} {
		t.Run(kind, func(t *testing.T) {
			store := SaveStore{Dir: t.TempDir()}
			base := filepath.Join(store.Dir, string(quickSaveBase(0)))
			manual := filepath.Join(store.Dir, "manual.sav")
			if err := os.WriteFile(manual, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if kind == "new failure" {
				if err := writeQuickSave(store, nil, 0, 1, raw, &failTimedPublish{failAt: 2}); err == nil {
					t.Fatal("no failure")
				}
				for _, ext := range []string{".sav", quickOwnerExtension} {
					if _, err := os.Stat(base + ext); !os.IsNotExist(err) {
						t.Fatal("partial new pair", err)
					}
				}
			} else {
				if err := writeQuickSave(store, nil, 0, 1, raw, nil); err != nil {
					t.Fatal(err)
				}
				owner, err := os.ReadFile(base + quickOwnerExtension)
				if err != nil {
					t.Fatal(err)
				}
				if kind == "overwrite failure" {
					if err := writeQuickSave(store, nil, 0, 2, raw, &failTimedPublish{failAt: 2}); err == nil {
						t.Fatal("no overwrite failure")
					}
					if got, err := os.ReadFile(base + quickOwnerExtension); err != nil || !bytes.Equal(got, owner) {
						t.Fatal("old owner lost", err)
					}
					if got, err := os.ReadFile(base + ".sav"); err != nil || !bytes.Equal(got, raw) {
						t.Fatal("old SAV lost", err)
					}
				} else {
					if kind == "foreign owner" {
						owner = []byte("foreign")
						if err := os.WriteFile(base+quickOwnerExtension, owner, 0600); err != nil {
							t.Fatal(err)
						}
					}
					var files namedSaveFileOps = osNamedSaveFiles{}
					if kind == "delete failure" {
						files = failQuickDelete{}
					}
					remove, err := prepareLoadDeleteFiles(store.Dir, localOriginalSaveToken(filepath.Base(base)+".sav"), nil, files)
					if err != nil {
						t.Fatal(err)
					}
					if kind == "changed owner" {
						owner = append(owner, ' ')
						if err := os.WriteFile(base+quickOwnerExtension, owner, 0600); err != nil {
							t.Fatal(err)
						}
					}
					err = remove()
					if kind == "changed owner" || kind == "delete failure" {
						if err == nil {
							t.Fatal("unsafe delete succeeded")
						}
						if got, err := os.ReadFile(base + ".sav"); err != nil || !bytes.Equal(got, raw) {
							t.Fatal("retained SAV lost", err)
						}
					} else if err != nil {
						t.Fatal(err)
					}
					if kind == "delete success" {
						if _, err := os.Stat(base + quickOwnerExtension); !os.IsNotExist(err) {
							t.Fatal("owned companion retained", err)
						}
					} else if got, err := os.ReadFile(base + quickOwnerExtension); err != nil || !bytes.Equal(got, owner) {
						t.Fatal("protected companion changed", err)
					}
				}
			}
			if got, err := os.ReadFile(manual); err != nil || !bytes.Equal(got, raw) {
				t.Fatal("manual changed", err)
			}
		})
	}
	store := SaveStore{Dir: t.TempDir()}
	if err := writeQuickSave(store, nil, 0, 1, raw, nil); err != nil {
		t.Fatal(err)
	}
	captured := readQuickSaveSlots(store.Dir)
	if err := os.WriteFile(filepath.Join(store.Dir, string(quickSaveBase(0)))+".sav", []byte("replaced path"), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := newestQuickSave(captured); err != nil || !bytes.Equal(got, raw) {
		t.Fatal("selection reread path", err)
	}
	if _, err := newestQuickSave(readQuickSaveSlots(store.Dir)); err == nil {
		t.Fatal("new selection admitted changed pair")
	}
}

func TestQuickLoadFailurePreservesDueDeadlineAndSession(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	store := SaveStore{Dir: t.TempDir()}
	now := time.Unix(100, 0)
	app := f.App("due quickload failure")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return now })
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	live, hash, tick := f.live, f.live.world.Hash(), f.live.world.Tick()
	now = now.Add(5 * time.Minute)
	if err := app.HeadlessKey("f9"); err != nil {
		t.Fatal(err)
	}
	if f.live != live || f.live.world.Hash() != hash || f.live.world.Tick() != tick || app.Screen() != ui.ScreenMap {
		t.Fatal("failed F9 changed session")
	}
	path := filepath.Join(store.Dir, "timed-autosave-1.sav")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("failed F9 admitted due poll", err)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("failed F9 reset due deadline", err)
	}
}

func TestQuickSaveAppFailureRecoveryAndManualDestination(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	store := SaveStore{Dir: t.TempDir()}
	app := f.App("quick blocked recovery")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	hash := f.live.world.Hash()
	for slot := range 3 {
		if err := os.WriteFile(filepath.Join(store.Dir, string(quickSaveBase(slot)))+".sav", []byte("foreign"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessKey("f4"); err != nil {
		t.Fatal(err)
	}
	visible := false
	for _, line := range f.live.view.MessageLines() {
		if strings.Contains(line.Text, "Quicksave failed: all three") {
			visible = true
		}
	}
	if !visible || f.live.world.Hash() != hash {
		t.Fatal("all-blocked error was not visible or changed state")
	}
	if err := os.Remove(filepath.Join(store.Dir, "quick-save-2.sav")); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("f4"); err != nil {
		t.Fatal(err)
	}
	if slots := readQuickSaveSlots(store.Dir); slots[1].sequence != 1 {
		t.Fatal("actual App retry did not find healthy slot")
	}
	prior, err := os.ReadFile(filepath.Join(store.Dir, "quick-save-2.sav"))
	if err != nil {
		t.Fatal(err)
	}
	other := t.TempDir()
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
	if app.Screen() == ui.ScreenGameMenu {
		if err := app.HeadlessGameMenuAction("return"); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessKey("f4"); err != nil {
		t.Fatal(err)
	}
	if slots := readQuickSaveSlots(other); slots[0].sequence != 1 {
		t.Fatal("quick did not follow committed manual destination")
	}
	if got, err := os.ReadFile(filepath.Join(store.Dir, "quick-save-2.sav")); err != nil || !bytes.Equal(got, prior) {
		t.Fatal("destination change touched prior slot", err)
	}
}
