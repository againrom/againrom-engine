package game

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func openWorldEffectsTestSave(t *testing.T, f *FrontEnd, source []byte, name string) (*ui.App, SaveStore, string) {
	t.Helper()
	f.SetDeterministicFrames(true)
	app := f.App("Retained world effects")
	app.Layout(1024, 768)
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, source, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, name)
	return app, store, path
}

// The oracle's source is the archive byte reader1152, not World/Document or
// the new binding decoder. Root identity finds its remapped current DTO row.
func light1162Differences(w *sim.World, state *SnapshotSAVDocument, source spell1152Graph, tick int) []string {
	var diff []string
	add := func(ok bool, name string) {
		if !ok {
			diff = append(diff, name)
		}
	}
	root := source.nodes[source.roots[0]]
	effects := w.SavedSpellEffects()
	alive := tick <= 18
	add(len(effects) == map[bool]int{true: 1, false: 0}[alive], "live root population")
	if alive && len(effects) == 1 {
		add(effects[0].AE4C == uint16(18-tick), "live countdown")
		add(slices.Equal(effects[0].AE48[:], root.raw["AE48"]), "live payload bytes")
	}
	add(state != nil && state.WorldEffects != nil && state.Document != nil, "current Document bindings")
	if state == nil || state.Document == nil {
		return diff
	}
	add(len(state.Document.World.Effects) == map[bool]int{true: 1, false: 0}[alive], "Document root population")
	var current *sav.DocumentRecordData
	for i := range state.Document.Objects {
		r := &state.Document.Objects[i]
		identity, _ := savedStructureValue(r, "Identity")
		if r.Class == "AreaEffect" && identity == root.values["Identity"] {
			current = r
		}
	}
	add((current != nil) == alive, "Document identity/retirement")
	if current != nil {
		timer, _ := savedStructureValue(current, "AE4C")
		add(timer == uint32(18-tick), "Document countdown")
	}
	count := 0
	for _, c := range w.SavedCellRecords() {
		if c.SpellEffects[4] == root.values["Identity"] {
			count++
		}
	}
	add(count == map[bool]int{true: 57, false: 0}[alive], "live57 cells/cleanup")
	count = 0
	for _, c := range state.Document.World.Cells {
		if c.Layers[4] == root.values["Identity"] {
			count++
		}
		if !alive {
			n := 0
			for _, key := range c.Layers {
				if key != 0 {
					n++
				}
			}
			if c.LayerCount != uint8(n) {
				diff = append(diff, "Document occupied-layer count")
				break
			}
		}
	}
	add(count == map[bool]int{true: 57, false: 0}[alive], "Document57 cells/cleanup")
	return diff
}

func TestReleaseWorldEffectsContinue1162(t *testing.T) {
	_, raw := groundCorpusFile(t, "2027-09-07/game0125.sav", "3a055c8dcef6f053721e1c1478c82552c199b92fb178e9f034fe7500077b4efd")
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	source, err := spell1152Expected(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(source.roots) != 1 || len(source.nodes) != 2 {
		t.Fatal("authentic Light graph population")
	}
	root := source.nodes[source.roots[0]]
	child := source.nodes[root.refs["AE44"]]
	if root.values["AE4C"] != 18 || root.values["T08"] != 1 || root.values["T0C"] != 12 || !slices.Equal(root.raw["AE48"], []byte{1, 4, 3, 0}) || child.values["E40"] != 0x00100001 || child.values["E3C"] != 19 || child.values["E3D"] != 1 {
		t.Fatal("authentic independent Light source fields changed")
	}
	worldEffects1162ExpiryCuts(t, raw, source)
	f := releaseFront(t)
	app, store, path := openWorldEffectsTestSave(t, f, raw, "light.sav")
	drivers := f.live.world.SavedWorldEffectDrivers()
	if drivers == nil || len(drivers.Areas) != 1 || drivers.Areas[0].Mode != 3 || drivers.Areas[0].Spell != 12 || drivers.Areas[0].Layer != 4 || len(drivers.Areas[0].Cells) != 57 {
		t.Fatal("Light mode/layer binding lost", drivers)
	}
	var subject sim.EntityID
	found := false
	for _, e := range f.live.world.ActiveEffects() {
		if e.Spell == 12 && e.Remaining == 12 {
			subject, found = e.Target, true
		}
	}
	if !found {
		t.Fatal("authentic actor Light12 attachment missing")
	}
	check := func(front *FrontEnd, tick int) {
		t.Helper()
		s, _, err := front.Snapshot(true)
		if err != nil {
			t.Fatal("Snapshot", tick, err)
		}
		if d := light1162Differences(front.live.world, s.SavedDocument, source, tick); len(d) != 0 {
			t.Fatal("Light independent current state", tick, d)
		}
		if tick <= 2 || tick >= 18 {
			want := uint16(12 - tick)
			if tick == 2 {
				want = 15
			}
			if tick >= 18 {
				// The root pulses when 1 becomes 0, refreshing to16 before
				// the actor phase consumes one tick. The next call cleans
				// the root while the independent attachment continues.
				want = uint16(33 - tick)
			}
			got := uint16(0)
			for _, e := range front.live.world.ActiveEffects() {
				if e.Target == subject && e.Spell == 12 {
					got = e.Remaining
				}
			}
			if got != want {
				for _, e := range front.live.world.Entities() {
					if e.ID == subject {
						t.Logf("subject %d at %d,%d ground %d alive %t actorBinding %v", subject, e.X, e.Y, e.Domain, e.Alive(), e.SourceBinding)
					}
				}
				t.Fatal("ordinary saved payload pulse/refresh", tick, got, want, front.live.world.SavedWorldEffectDrivers())
			}
		}
	}
	check(f, 0)
	// The authentic actor is outside the cloud at14,41. A controlled existing
	// headless relocation places it in the retained cloud for the pulse witness.
	if err := f.live.world.HeadlessPlace(subject, 19, 40); err != nil {
		t.Fatal(err)
	}
	f.live.tick()
	check(f, 1)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	fresh := spell1152MenuFresh(t, f, app, store, releaseFront)
	check(fresh, 1)
	// Permanent suppression, counter, payload and identity loss controls.
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"countdown", "root", "Token", "payload"} {
		bad, err := cloneSavedDocument(snapshot.SavedDocument)
		if err != nil {
			t.Fatal(err)
		}
		row := bad.WorldEffects.Areas[0]
		r := &bad.Document.Objects[row.ObjectIndex-1]
		switch name {
		case "countdown":
			savedObjectSetValue(r, "AE4C", 18)
		case "root":
			bad.Document.World.Effects = nil
		case "Token":
			savedObjectSetValue(r, "Identity", root.values["Identity"]+1)
		case "payload":
			savedObjectSetValue(&bad.Document.Objects[row.ChildIndex-1], "E40", 1)
		}
		if err := restoreSavedDocument(&Mission{World: f.live.world}, bad); err == nil {
			t.Fatal("native area loss admitted", name)
		}
	}
	for tick := 2; tick <= 19; tick++ {
		f.live.tick()
		fresh.live.tick()
		check(f, tick)
		check(fresh, tick)
		if f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatal("source-free continuation differs", tick)
		}
	}
	t.Log(fmt.Sprintf("Light raw source18/mode1/57cells: step17, changed menu SAVE, source-free LOAD, controlled actor moved to19,40; pulse16 refreshes actor%d to16 then actor phase leaves15, zero cut retained, next removal clears root+exclusive child+57 cells; driver/counter/identity/payload controls RED", subject))
}

func TestReleaseRetiredWorldEffectsColdLoad(t *testing.T) {
	for _, tc := range []struct {
		name, path, digest string
		graph              bool
	}{
		{"retired graph", "2026-08-15/game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b", true},
		{"retired area", "2027-09-07/game0125.sav", "3a055c8dcef6f053721e1c1478c82552c199b92fb178e9f034fe7500077b4efd", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, raw := groundCorpusFile(t, tc.path, tc.digest)
			front := releaseFront(t)
			app, _, _ := openWorldEffectsTestSave(t, front, raw, "retired.sav")
			if tc.graph && front.live.world.SavedSpellGraph() == nil || !tc.graph && front.live.world.SavedWorldEffectDrivers() == nil {
				t.Fatal("source has no effect carrier to retire")
			}
			retired := false
			for range 64 {
				graph := front.live.world.SavedSpellGraph()
				drivers := front.live.world.SavedWorldEffectDrivers()
				if tc.graph {
					retired = graph == nil && drivers == nil && len(front.live.world.SavedSpellEffects()) == 0
				} else {
					retired = drivers == nil && len(front.live.world.SavedSpellEffects()) == 0
				}
				if retired {
					break
				}
				front.LiveAdvance(1)
			}
			if !retired {
				t.Fatalf("ordinary ticks kept finished effects: graph %v, drivers %v", front.live.world.SavedSpellGraph(), front.live.world.SavedWorldEffectDrivers())
			}
			graph, drivers := front.live.world.SavedSpellGraph(), front.live.world.SavedWorldEffectDrivers()
			store, name, _ := menuSAVE(t, front, app, OriginalStore{})
			cold, _ := loadSAVWindow(t, store, name)
			if !reflect.DeepEqual(graph, cold.live.world.SavedSpellGraph()) || !reflect.DeepEqual(drivers, cold.live.world.SavedWorldEffectDrivers()) {
				t.Fatalf("cold SAV lost retired current effects: graph %v -> %v, drivers %v -> %v", graph, cold.live.world.SavedSpellGraph(), drivers, cold.live.world.SavedWorldEffectDrivers())
			}
		})
	}
}
