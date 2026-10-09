package game

import (
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
)

// Counts are literal results of the independent exact-record seat reader,
// limited to unique selectors and living members. ActorGraph/SourceBinding and
// the importer under test do not supply expected counts. These cases preserve
// the authored ALM binding; they do not establish full saved GroupAI fidelity.
func TestReleaseActorRegistry1111AuthoredGroupsSurviveOriginalAndNativeLoad(t *testing.T) {
	for _, tc := range []struct {
		path, sha string
		counts    map[uint32]int32
	}{
		{"2026-08-02/game0003.sav", "8902ab4b04068af4a8d37107e6c2d5a9a9cc30bbe67a022901583a01da7ee03d", map[uint32]int32{4: 3, 3: 3, 5: 1}},
		{"2026-08-02/game0001.sav", "c5010cfbbe1f59018452813e7eda192dc5f3ce9834c9246bad23287e168899dd", map[uint32]int32{1: 3, 3: 4, 7: 3, 8: 2}},
		{"2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c", map[uint32]int32{1: 2, 4: 3, 7: 1, 3: 3, 5: 2}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			f := releaseFront(t)
			path, raw := groundCorpusFile(t, tc.path, tc.sha)
			ms, _, err := loadOriginalMission(f, raw)
			if err != nil {
				t.Fatal(err)
			}
			registryGroupPhase1111(t, ms.World, tc.counts)
			for _, fromMap := range []bool{false, true} {
				f := releaseFront(t)
				f.SetDeterministicFrames(true)
				app := f.App("source groups")
				if fromMap {
					if err := app.OpenMission(f.MissionOpener(int(ms.Number))); err != nil {
						t.Fatal(err)
					}
				}
				store := SaveStore{Dir: t.TempDir()}
				save, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(path)}, nil)
				app.SetSaveSeams(save, list, load)
				groundAppLoad(t, app, list, filepath.Base(path))
				// Save before the first script phase, then test both independent
				// continuations. game0021 may be at Victory, so use the production
				// callback rather than pretending its modal permits menu keys.
				fresh, _ := holdingsNativeFresh(t, f, app, store, save)
				for _, w := range []*sim.World{f.live.world, fresh.live.world} {
					registryGroupPhase1111(t, w, tc.counts)
					registryGroupOrder1111(t, w, 3)
				}
			}
		})
	}
}

func registryGroupPhase1111(t *testing.T, w *sim.World, want map[uint32]int32) {
	t.Helper()
	for group, count := range want {
		var got int32
		for _, e := range w.Entities() {
			if e.Alive() && e.Group == group {
				got++
			}
		}
		if got != count {
			t.Fatalf("authored group%d at LOAD: got%d want%d", group, got, count)
		}
	}
	checks := w.Script().Checks()
	for range 20 {
		trace := sim.StepTraced(w, nil)
		if !trace.Pass {
			continue
		}
		seen := map[uint32]bool{}
		for _, run := range trace.Checks {
			if run.Op != sim.ScriptCheckGroupCount || int(run.Check) >= len(checks) {
				continue
			}
			group := checks[run.Check].Group
			if count, ok := want[group]; ok {
				seen[group] = true
				if !run.Wrote || run.Value != count {
					t.Fatalf("first script phase group%d wrote%t value%d want%d", group, run.Wrote, run.Value, count)
				}
			}
		}
		if len(seen) != len(want) {
			t.Fatalf("first phase missed authored checks: %v", seen)
		}
		t.Logf("first script phase tick%d: %d literal group counts retained", trace.Tick, len(seen))
		return
	}
	t.Fatal("no script phase")
}

func registryGroupOrder1111(t *testing.T, imported *sim.World, group uint32) {
	t.Helper()
	// Isolate the public GroupOrder dispatcher from natural incoming AI debt.
	// Identity/owner/authored-group come from the actual imported bindings;
	// positions and the order are a synthetic control, not a ROM1 witness.
	var selected, other sim.Entity
	for _, e := range imported.Entities() {
		if !e.Alive() || e.SourceBinding.Class == 0 || e.Group == 0 {
			continue
		}
		if e.Group == group {
			selected = e
		} else {
			other = e
		}
	}
	if selected.Group != group || other.Group == 0 {
		t.Fatal("missing literal GroupOrder control bindings")
	}
	s, err := sim.NewScript(nil, []sim.ScriptInstant{{Op: sim.ScriptInstantGroupOrder, HasGroup: true, Group: group, Args: [10]int32{4, 20, 20}}},
		[]sim.ScriptTrigger{{Instants: [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}, Once: true}})
	if err != nil {
		t.Fatal(err)
	}
	w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: selected.ID, Owner: selected.Owner, Group: selected.Group, X: 10, Y: 10, HP: 10, MaxHP: 10},
			{ID: other.ID, Owner: other.Owner, Group: other.Group, X: 12, Y: 10, HP: 10, MaxHP: 10}}, s)
	if err != nil {
		t.Fatal(err)
	}
	for range 20 {
		if sim.StepTraced(w, nil).Pass {
			break
		}
	}
	for _, e := range w.Entities() {
		if e.ID == selected.ID && (!e.HasTarget || e.TargetX != 20 || e.TargetY != 20) || e.ID == other.ID && e.HasTarget {
			t.Fatalf("GroupOrder targeted wrong imported binding: %+v", e)
		}
	}
}
