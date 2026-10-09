package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// This is the owner's unmodified M10 save plus explicitly named diagnostic
// mutations. It is not evidence that an original runtime cleared/changed a
// trigger, and re-emitting its envelope is not a source-free SAV writer.
func TestReleaseOriginalCellTriggers1106AppOverlayAndNativeEntry(t *testing.T) {
	_ = releaseFront(t)
	_, payload := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	for _, variant := range []string{"natural", "saved-clear", "changed-source", "relocation-insert"} {
		t.Run(variant, func(t *testing.T) {
			source, err := sav.Open(payload)
			if err != nil {
				t.Fatal(err)
			}
			if source.World.CellRecCount != 184 || !bytes.Equal(source.Body[44159:44165], []byte{13, 1, 19, 61, 21, 63}) ||
				!bytes.Equal(source.Body[45563:45569], []byte{13, 1, 23, 65, 22, 64}) {
				t.Fatal("named source records changed")
			}
			wantCast := sim.ScriptCast{FromX: 19, FromY: 61, Spell: 13, Power: 1, AtUnit: true}
			switch variant {
			case "saved-clear":
				source.Body[44159] = 0
			case "changed-source":
				copy(source.Body[44159:44165], []byte{13, 2, 17, 59, 231, 232})
				wantCast.FromX, wantCast.FromY, wantCast.Power = 17, 59, 2
			case "relocation-insert":
				source.Body[44113], source.Body[44114] = 255, 254
				copy(source.Body[44159:44165], []byte{26, 8, 9, 10, 251, 252})
			}
			// Independent raw 54-byte walk, with the two ALM construction tails
			// stated literally. This oracle does not call the new projection or
			// importer and also checks every zero tail in the real saved table.
			expected := map[uint16][6]byte{0x3f15: {13, 1, 19, 61, 21, 63}, 0x4016: {13, 1, 23, 65, 22, 64}}
			for i := 0; i < 184; i++ {
				off := 43033 + 54*i
				var tail [6]byte
				copy(tail[:], source.Body[off+46:off+52])
				expected[binary.LittleEndian.Uint16(source.Body[off:off+2])] = tail
			}
			var want []sim.CellTail
			for key, tail := range expected {
				want = append(want, sim.CellTail{X: int32(key & 255), Y: int32(key >> 8), Bytes: tail})
			}
			sort.Slice(want, func(i, j int) bool { return want[i].Y*256+want[i].X < want[j].Y*256+want[j].X })
			changed := source.Marshal()
			if variant == "natural" {
				changed = payload
			}
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "game9999.sav"), changed, 0600); err != nil {
				t.Fatal(err)
			}
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			app := f.App("1106 original overlay")
			app.Layout(1024, 768)
			store := SaveStore{Dir: t.TempDir()}
			app.SetSaveSeams(nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: dir}, nil))
			_, list, _ := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: dir}, nil)
			groundAppLoad(t, app, list, "game9999.sav")
			if !reflect.DeepEqual(f.live.world.CellTails(), want) || f.live.world.Tick() != rawSavedSubTick1112(t, payload) || len(f.live.world.ScriptCasts()) != 0 {
				t.Fatalf("LOAD did not install exact union: tails=%d want=%d tick=%d", len(f.live.world.CellTails()), len(want), f.live.world.Tick())
			}
			ms, report, err := ResumeOriginalSave(f.Archives.Containers, changed, f.Table, mapload.DifficultyNormal, nil, f.Bodies)
			if err != nil || ms == nil || !report.CellTriggersApplied || report.CellTriggerRecords != 184 || !reflect.DeepEqual(ms.World.CellTails(), want) || ms.World.Tick() != rawSavedSubTick1112(t, payload) || len(ms.World.ScriptCasts()) != 0 {
				t.Fatalf("diagnostic overlay: %v %+v", err, report)
			}
			// Explicit controlled endpoint, after both untouched LOAD/union
			// checks: isolate cell-entry casting from competing mission notices.
			// This remains a pointer/native-cast witness, not incoming original
			// mission chronology. Keep the saved clocks and all four variants.
			quiet, err := sim.NewScript(nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			controlled, err := sim.NewControlledScriptWorld(f.live.world, quiet)
			if err != nil {
				t.Fatal(err)
			}
			mapload.BindSourceDerive(controlled)
			*f.live.world = *controlled
			t.Log("controlled endpoint: competing mission program disabled after exact original LOAD checks; saved clocks retained")
			id := f.live.mission.ids[0]
			wantCast.Target = id
			// Controlled setup only. Entry itself is the ordinary App pointer
			// move from this non-trigger adjacent cell, never a LOAD attachment.
			if err := f.live.world.HeadlessPlace(id, 21, 64); err != nil {
				t.Fatal(err)
			}
			f.live.push()
			fresh, freshApp := cellAppSaveFresh1106(t, f, app, store, "before-entry")
			for _, pair := range []struct {
				f   *FrontEnd
				app *ui.App
			}{{f, app}, {fresh, freshApp}} {
				pair.f.live.view.Camera().CenterOn(22*32, 64*32)
				if err := pair.app.HeadlessSelectEntity(uint32(id)); err != nil {
					t.Fatal(err)
				}
				entryPointer1084(t, pair.app, 21, 63)
			}
			entered := false
			for tick := 0; tick < 100; tick++ {
				if err := entryStep1084(app); err != nil {
					t.Fatal(err)
				}
				if err := entryStep1084(freshApp); err != nil {
					t.Fatal(err)
				}
				if f.live.world.Hash() != fresh.live.world.Hash() || !reflect.DeepEqual(f.live.world.ScriptCasts(), fresh.live.world.ScriptCasts()) {
					t.Fatalf("fresh App continuation differs at %d: S%d/%d pending%v/%v", tick, f.live.world.Tick(), fresh.live.world.Tick(), f.live.pending, fresh.live.pending)
				}
				for _, cast := range f.live.world.ScriptCasts() {
					if cast.Target != id {
						continue
					}
					if variant == "saved-clear" {
						t.Fatalf("saved clear rearmed cast %+v", cast)
					}
					if cast != wantCast {
						t.Fatalf("cast %+v want %+v", cast, wantCast)
					}
					entered = true
				}
				e, _ := f.live.entity(id)
				if variant == "saved-clear" && e.X == 21 && e.Y == 63 && len(f.live.world.ScriptCasts()) == 0 {
					entered = true
				}
				if entered {
					break
				}
			}
			if !entered {
				e, _ := f.live.entity(id)
				t.Fatalf("pointer failed to enter saved cell: S%d outcome%v stopped%t pos%d,%d target%t/%d,%d", f.live.world.Tick(), f.live.world.Outcome(), f.live.stopped, e.X, e.Y, e.HasTarget, e.TargetX, e.TargetY)
			}
			if variant == "saved-clear" {
				for range 10 {
					if err := entryStep1084(app); err != nil {
						t.Fatal(err)
					}
					if len(f.live.world.ScriptCasts()) != 0 {
						t.Fatal("cleared standing cell cast")
					}
				}
				t.Logf("saved clear: ordinary pointer entry (21,63), no cell cast; six bytes and AGS preserved")
				return
			}
			pending, _ := cellAppSaveFresh1106(t, f, app, SaveStore{Dir: t.TempDir()}, "pending-entry")
			a, b := sim.StepReported(f.live.world, nil), sim.StepReported(pending.live.world, nil)
			wantEvent := sim.ScriptCastEvent{Spell: 13, FromX: wantCast.FromX, FromY: wantCast.FromY, ToX: 21, ToY: 63}
			if !reflect.DeepEqual(a, b) || f.live.world.Hash() != pending.live.world.Hash() || len(a.ScriptCasts) != 1 || !reflect.DeepEqual(a.ScriptCasts[0], wantEvent) {
				t.Fatalf("pending native cast continuation: %+v / %+v", a.ScriptCasts, b.ScriptCasts)
			}
			t.Logf("%s: records184 union%d; pointer cast %+v; ordinary AGS before/pending fresh LOAD hash/events equal", variant, len(want), wantCast)
		})
	}
}

func cellAppSaveFresh1106(t *testing.T, f *FrontEnd, app *ui.App, store SaveStore, stage string) (*FrontEnd, *ui.App) {
	t.Helper()
	app.SetSaveSeams(nativeContinuationSeams1170(t, f, store, OriginalStore{}, nil))
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := listAGS(store)
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".ags" {
		t.Fatalf("%s ordinary SAVE %+v %v", stage, entries, err)
	}
	if app.Screen() == ui.ScreenGameMenu {
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
	}
	fresh := releaseFront(t)
	fresh.SetDeterministicFrames(true)
	freshApp := fresh.App("1106 fresh " + stage)
	freshApp.Layout(1024, 768)
	save, list, load := nativeContinuationSeams1170(t, fresh, store, OriginalStore{}, nil)
	freshApp.SetSaveSeams(save, list, load)
	groundAppLoad(t, freshApp, list, entries[0].Name)
	a, _ := f.live.world.MarshalBinary()
	b, _ := fresh.live.world.MarshalBinary()
	if !bytes.Equal(a, b) || f.live.world.Hash() != fresh.live.world.Hash() {
		t.Fatalf("%s ordinary SAVE/fresh LOAD changed world", stage)
	}
	return fresh, freshApp
}
