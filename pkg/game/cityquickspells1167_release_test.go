package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// This oracle locates the physical record, independently of the production
// ID map, CityData projection and originalQuickSpells reader.
func cityQuickRaw1167(t *testing.T, raw []byte, want [4]int32) (*sav.File, int) {
	t.Helper()
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	n := int(binary.LittleEndian.Uint32(f.Store[16:20]))
	for i := 0; i < n; i++ {
		r := f.Store[24+32*i : 24+32*(i+1)]
		if string(bytes.TrimRight(r[16:32], "\x00")) != "Shortcuts" {
			continue
		}
		if binary.LittleEndian.Uint32(r[8:12]) != 16 || binary.LittleEndian.Uint32(r[12:16]) != 6 {
			t.Fatal("raw shortcut store must have kind6 and 16 bytes")
		}
		at := 24 + 32*n + 4 + int(binary.LittleEndian.Uint32(r[4:8]))
		var got [4]int32
		for j := range got {
			got[j] = int32(binary.LittleEndian.Uint32(f.Store[at+4*j:]))
		}
		if got != want {
			t.Fatalf("raw F5-F8 signed cells %v, want %v", got, want)
		}
		return f, at
	}
	t.Fatal("missing raw Shortcuts record")
	return nil, 0
}

func cityQuickKey1167(t *testing.T, app *ui.App, key string) {
	t.Helper()
	if err := app.HeadlessKey(key); err != nil {
		t.Fatal(key, err)
	}
}

// Bind through the installed App/Viewer attached by the production mission
// opener to this same FrontEnd's slots. The mage/book/mana are controlled
// inputs. The town party stays unchanged, and progress changes only by the
// selection a fresh mission activation commits: this is a paused input probe,
// not a claimed played mission or a campaign-arrival witness.
func cityQuickProbe1167(t *testing.T, f *FrontEnd, act func(*ui.App, sim.EntityID)) {
	t.Helper()
	before, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
	hero.Skill[1] = 100
	party := []mapload.PartyMember{{ID: "probe-mage", PlayerCharacter: true, StartingHero: true, Mage: true, Class: 0x18,
		Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero,
		KnownSpells: 1<<1 | 1<<16 | 1<<18 | 1<<23,
		Saved: &mapload.Saved{Cell: mapload.Cell{X: 29, Y: 50}, HP: 100, MaxHP: 100, Mana: 1000, MaxMana: 1000,
			HealthRegenPeriod: 100, ManaRegenPeriod: 50}}}
	f.SetDeterministicFrames(true)
	app := f.App("1167 current city bindings")
	app.Layout(1024, 768)
	mission := f.Town.Chapter()
	if err := app.OpenMission(f.MissionOpenerWith(mission, party)); err != nil {
		t.Fatal(err)
	}
	cityQuickKey1167(t, app, "0")
	id := f.live.mission.ids[0]
	inspectionCentre(f.live, 29, 50)
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.HeadlessSpellPoint(16); err != nil {
		cityQuickKey1167(t, app, "book")
	}
	act(app, id)
	after, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	if before.CampaignState {
		if after.Campaign.SelectedMission != uint32(mission) || after.Campaign.FirstMapPoint || !after.CampaignMarkerSelected {
			t.Fatalf("activation committed mission %d, first map point %v, marker %v; want %d, false, true",
				after.Campaign.SelectedMission, after.Campaign.FirstMapPoint, after.CampaignMarkerSelected, mission)
		}
		before.Campaign.SelectedMission, before.Campaign.FirstMapPoint = after.Campaign.SelectedMission, after.Campaign.FirstMapPoint
		before.CampaignMarkerSelected = after.CampaignMarkerSelected
	}
	before.QuickSpells = after.QuickSpells
	if !reflect.DeepEqual(before, after) {
		a, b := reflect.ValueOf(&before).Elem(), reflect.ValueOf(&after).Elem()
		for i := 0; i < a.NumField(); i++ {
			x, y := installShareRead(a.Field(i)), installShareRead(b.Field(i))
			if !reflect.DeepEqual(x, y) {
				t.Logf("probe changed field %s: %v => %v", a.Type().Field(i).Name, x, y)
			}
		}
		t.Fatal("input probe changed town state beyond current shortcuts and the committed selection")
	}
}

func cityQuickCast1167(t *testing.T, f *FrontEnd, app *ui.App, id sim.EntityID) {
	t.Helper()
	cityQuickKey1167(t, app, "book") // Close the book, then invoke existing F8.
	cityQuickKey1167(t, app, "f8")
	if _, current, armed := f.live.view.QuickSpellState(); current != 16 || !armed {
		t.Fatalf("loaded F8 did not arm spell16: current%d armed%v", current, armed)
	}
	x, y, err := app.HeadlessEntityPoint(uint32(id))
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	if got := f.live.pending; len(got) != 1 || got[0].Kind != sim.KindCast || got[0].Y != 16 || got[0].X != int32(id) {
		t.Fatalf("loaded F8 targeted %+v", got)
	}
	before, _ := f.live.entity(id)
	for tick := 0; tick < 200; tick++ {
		f.live.tick()
		after, _ := f.live.entity(id)
		if after.Mana < before.Mana {
			return
		}
	}
	t.Fatal("loaded F8 targeted cast spent no mana")
}

func TestReleaseCityQuickSpellsCurrentWriters1167(t *testing.T) {
	for _, origin := range []string{"imported", "generated"} {
		t.Run(origin, func(t *testing.T) {
			f := releaseFront(t)
			root := os.Getenv("AGAINROM_ASSETS")
			store := SaveStore{Dir: t.TempDir()}
			var privateSource string
			if origin == "imported" {
				path, source := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
				privateSource = filepath.Join(t.TempDir(), "source.sav")
				if err := WriteConvertedSave(privateSource, source, root); err != nil {
					t.Fatal(err)
				}
				_, _, load := agsSaveSeams(f, store, OriginalStore{Dir: filepath.Dir(privateSource)}, nil)
				if _, town, err := load("source.sav"); err != nil || !town {
					t.Fatal("private source LOAD", town, err)
				}
				t.Cleanup(func() {
					after, err := os.ReadFile(path)
					if err != nil || !bytes.Equal(source, after) {
						t.Fatal("preserved original source changed")
					}
				})
			} else {
				reachabilityWalkArrive(t, f, "Current shortcuts")
				f.addChapterCompanions(f.Town.Chapter())
			}
			// Prepare the current chapter's ordinary document grants before
			// measuring the key-only change; mission entry collects them too.
			f.Town.CollectDocuments(f.Town.Chapter())
			before, _, err := f.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			export := f.ExportNativeCitySave
			if origin == "imported" {
				export = f.ExportOriginalSave
			}
			control, err := export(before, "control")
			if err != nil {
				t.Fatal("unchanged town is not SAV-capable", err)
			}
			cityQuickProbe1167(t, f, func(app *ui.App, _ sim.EntityID) {
				for _, binding := range [][2]uint32{{5, 23}, {6, 18}, {8, 16}} {
					x, y, err := app.HeadlessSpellPoint(binding[1])
					if err != nil {
						t.Fatal(err)
					}
					if err := app.HeadlessPointer("hover", x, y); err != nil {
						t.Fatal(err)
					}
					cityQuickKey1167(t, app, fmt.Sprintf("ctrl-f%d", binding[0]))
				}
			})
			firstIDs, firstCells := [4]uint32{23, 18, 0, 16}, [4]int32{5, 23, -1, 7}
			if f.quickSpells != firstIDs {
				t.Fatal("real binding input", f.quickSpells)
			}
			// The ordinary town save callback reads current slots, regardless of
			// the retained paused driver. No world/offer state is normalized.
			save, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
			name, err := save(false)
			if err != nil || !IsOriginal(name) {
				t.Fatal("current bindings fell back instead of saving SAV", name, err)
			}
			raw, err := store.Read(name)
			if err != nil {
				t.Fatal(err)
			}
			actual, at := cityQuickRaw1167(t, raw, firstCells)
			base, baseAt := cityQuickRaw1167(t, control, [4]int32{-1, -1, -1, -1})
			comparison := bytes.Clone(actual.Store)
			copy(comparison[at:at+16], base.Store[baseAt:baseAt+16])
			if !bytes.Equal(base.Body, actual.Body) || !bytes.Equal(base.TailRest, actual.TailRest) || !bytes.Equal(base.Store, comparison) {
				t.Fatal("current shortcuts changed unrelated source records")
			}
			current, _, err := f.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			ags, err := EncodeSave(current, "quick\xffcity")
			if err != nil {
				t.Fatal(err)
			}
			converted, label, err := releaseFront(t).ConvertCitySave(ags, "sav", nil)
			if err != nil || label != "quick\xffcity" {
				t.Fatal("current AGS conversion", label, err)
			}
			cityQuickRaw1167(t, converted, firstCells)
			if privateSource != "" {
				if err := os.Remove(privateSource); err != nil {
					t.Fatal(err)
				}
			}
			fresh := releaseFront(t)
			save2, _, load2 := agsSaveSeams(fresh, store, OriginalStore{}, nil)
			if _, town, err := load2(localOriginalSaveToken(name)); err != nil || !town || fresh.quickSpells != firstIDs {
				t.Fatal("source-free fresh SAV LOAD", fresh.quickSpells, town, err)
			}
			cityQuickProbe1167(t, fresh, func(app *ui.App, id sim.EntityID) {
				cityQuickKey1167(t, app, "f5")
				cityQuickKey1167(t, app, "ctrl-f7") // Move23, clearing F5 to -1.
				if fresh.quickSpells != ([4]uint32{0, 18, 23, 16}) {
					t.Fatal("duplicate move did not clear the old key", fresh.quickSpells)
				}
				cityQuickCast1167(t, fresh, app, id)
			})
			name2, err := save2(false)
			if err != nil || !IsOriginal(name2) {
				t.Fatal("changed second town SAVE", name2, err)
			}
			raw2, err := store.Read(name2)
			if err != nil {
				t.Fatal(err)
			}
			cityQuickRaw1167(t, raw2, [4]int32{-1, 23, 5, 7})
			third := releaseFront(t)
			_, _, load3 := agsSaveSeams(third, store, OriginalStore{}, nil)
			if _, town, err := load3(localOriginalSaveToken(name2)); err != nil || !town || third.quickSpells != ([4]uint32{0, 18, 23, 16}) {
				t.Fatal("second SAV LOAD lost cleared/current bindings", third.quickSpells, town, err)
			}
			cityQuickProbe1167(t, third, func(app *ui.App, id sim.EntityID) { cityQuickCast1167(t, third, app, id) })
			// Empty is an explicit current update, distinct from nil/preserve.
			third.quickSpells = [4]uint32{}
			empty, _, err := third.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			emptyAGS, err := EncodeSave(empty, "cleared")
			if err != nil {
				t.Fatal(err)
			}
			emptySAV, _, err := releaseFront(t).ConvertCitySave(emptyAGS, "sav", nil)
			if err != nil {
				t.Fatal(err)
			}
			cityQuickRaw1167(t, emptySAV, [4]int32{-1, -1, -1, -1})
			if dir := os.Getenv("AGAINROM_1167_ARTIFACTS"); dir != "" {
				for suffix, b := range map[string][]byte{"current.ags": ags, "first.sav": raw, "second.sav": raw2} {
					if err := WriteConvertedSave(filepath.Join(dir, origin+"-"+suffix), b, root); err != nil {
						t.Fatal(err)
					}
				}
			}
			t.Log("App Ctrl-F5/F6/F8 => IDs[23 18 0 16], raw kind6 cells[5 23 -1 7]; ordinary town SAVE, private source removed, fresh LOAD; F5/Ctrl-F7 clears F5, second SAVE cells[-1 23 5 7], cold LOAD/F8 targeting spends mana; converter current and all-unbound words; unrelated city records retained")
		})
	}
}
