package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// findMotion1134 locates a carried saved motion by its sim EntityID, the same
// identity space the review's own MarshalBinary error text names ("entity
// record 56"), rather than the archive index sameMoverRouteForAudit keys by.
func findMotion1134(motions []sim.SavedActorMotion, entity sim.EntityID) (sim.SavedActorMotion, bool) {
	for _, m := range motions {
		if m.Entity == entity {
			return m, true
		}
	}
	return sim.SavedActorMotion{}, false
}

func TestReleaseOriginalMoverRouteRestoresOnLoad1134(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-24/game0021.sav",
		"7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := source.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	want := archiveActorsForAudit(graph)
	witness, ok := want[96]
	if !ok || len(witness.StaticRoute) != 3 || len(witness.DynamicRoute) != 4 {
		t.Fatalf("fixture does not carry the known 3/4-element route witness: %+v", witness)
	}

	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1134 mover route")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0021.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{Dir: dir}, nil))
	_, list, _ := agsSaveSeams(f, store, OriginalStore{Dir: dir}, nil)
	groundAppLoad(t, app, list, "game0021.sav")

	// Independently reconstructed live comparison: walks the file's own
	// decode against the live world state in parallel, on
	// TestMoverRouteCorpusAudit1134's own shape, rather than calling
	// exportOriginalMoverRoutes for both directions.
	live := archiveMotionsForAudit(f.live.world)
	if msg, ok := sameMoverRouteForAudit(want, live); !ok {
		t.Fatalf("file vs live mover/route: %s", msg)
	}
	liveWitness := live[96]
	if len(liveWitness.StaticRoute) != 3 || len(liveWitness.DynamicRoute) != 4 {
		t.Fatalf("LOAD did not restore the known route witness: %+v", liveWitness)
	}

	// Native export: write the live App state into a fresh decode of this
	// file's own bytes and require the complete body to reproduce the source
	// file byte for byte (exportOriginalMoverRoutes only ever patches inside
	// each actor's own mover/route spans,
	// TestSetActorMoverRouteLeavesUnrelatedBytesAlone).
	target, err := sav.Open(payload)
	if err != nil {
		t.Fatalf("sav.Open (export target): %v", err)
	}
	if err := exportOriginalMoverRoutes(target, f.live.world); err != nil {
		t.Fatalf("exportOriginalMoverRoutes: %v", err)
	}
	written, err := sav.Open(target.Marshal())
	if err != nil {
		t.Fatalf("sav.Open (re-decode written): %v", err)
	}
	if len(written.Body) != len(source.Body) {
		t.Fatalf("native export changed the body length: %d -> %d", len(source.Body), len(written.Body))
	}
	for i := range source.Body {
		if source.Body[i] != written.Body[i] {
			t.Fatalf("native export of the mover/route state does not reproduce the source file at byte %d", i)
		}
	}
	t.Logf("actors=%d: archive 96's 3-element static / 4-element dynamic route restored; LOAD and native export both reproduce the source file exactly", len(want))
}

func TestReleaseOriginalMoverRouteDomainBlockedCellIsNotRefusedAcrossSaveReload1134(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-14/game0014.sav",
		"4c37a3d122ea849ce989f054c82483e32be9450886e7e3b60b641936a0bf0afc")
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1134 closed-cell continuation")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0014.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game0014.sav")

	check := func() {
		motions, _, _, present := f.live.world.SavedActorMotions()
		if !present {
			t.Fatal("saved motions absent")
		}
		m, ok := findMotion1134(motions, 56)
		if !ok {
			t.Fatal("entity 56 (the review's own repro) is missing")
		}
		if len(m.DynamicRoute) != 1 || m.DynamicRoute[0] != 0x1140 {
			t.Fatalf("carried dynamic route should stay untouched: %v", m.DynamicRoute)
		}
		if !m.Current || m.Issue != "" {
			t.Fatalf("a route naming a closed cell is admitted, not refused: current=%v active=%v issue=%q",
				m.Current, m.Active, m.Issue)
		}
		if len(f.live.world.Route(56)) != 0 {
			t.Fatal("a route 51 cells from its unit was handed to native movement")
		}
	}
	check()
	hash := f.live.world.Hash()

	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatalf("Save after loading a route across a closed cell: %v", err)
	}
	entries, err := listAGS(store)
	if err != nil || len(entries) != 1 {
		t.Fatalf("ordinary SAVE: %+v %v", entries, err)
	}
	groundAppLoad(t, app, list, entries[0].Name)
	check()
	if f.live.world.Hash() != hash {
		t.Fatal("native SAVE/LOAD changed the motion's state")
	}
	t.Log("entity 56's route across a closed cell was kept across LOAD, native SAVE and reload; Hash unchanged")
}

func TestReleaseOriginalMoverRouteAnchorGuardSurvivesSaveReload1134(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-02/game9999.sav",
		"5822c37e8fa531e0b6d9b2348977c31f78d33f5035dd5c780e8b73459840b78e")
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1134 unanchored continuation")
	app.Layout(1024, 768)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game9999.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game9999.sav")

	entityPosition := func() (int32, int32, bool) {
		for _, e := range f.live.world.Entities() {
			if e.ID == 56 {
				return e.X, e.Y, e.HasTarget
			}
		}
		t.Fatal("entity 56 (the review's own repro) is missing")
		return 0, 0, false
	}
	check := func() {
		motions, _, _, present := f.live.world.SavedActorMotions()
		if !present {
			t.Fatal("saved motions absent")
		}
		m, ok := findMotion1134(motions, 56)
		if !ok {
			t.Fatal("entity 56 (the review's own repro) is missing")
		}
		if len(m.DynamicRoute) != 4 || m.DynamicRoute[0] != 0x293a {
			t.Fatalf("carried dynamic route should stay untouched: %v", m.DynamicRoute)
		}
		if !m.Current || m.Active || m.Issue != "" {
			t.Fatalf("an unanchored route should be withheld with no Issue, not refused or continued: current=%v active=%v issue=%q",
				m.Current, m.Active, m.Issue)
		}
	}
	check()
	x, y, hasTarget := entityPosition()
	if hasTarget {
		t.Fatalf("entity 56 should carry no native target: x=%d y=%d", x, y)
	}

	for range 200 {
		sim.Step(f.live.world, nil)
	}
	if nx, ny, hasTarget := entityPosition(); nx != x || ny != y || hasTarget {
		t.Fatalf("F-2's own verify: entity 56 should not move over 200 ticks: x=%d y=%d hasTarget=%v", nx, ny, hasTarget)
	}
	check()
	// hash is captured here, AFTER the 200-tick verify, so it is compared
	// against the exact state that gets written to the .ags below -- not
	// against the pre-tick state, which 200 ordinary ticks legitimately
	// changes all over the rest of the world regardless of entity 56.
	hash := f.live.world.Hash()

	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatalf("Save after loading an unanchored saved route: %v", err)
	}
	entries, err := listAGS(store)
	if err != nil || len(entries) != 1 {
		t.Fatalf("ordinary SAVE: %+v %v", entries, err)
	}
	groundAppLoad(t, app, list, entries[0].Name)
	check()
	if f.live.world.Hash() != hash {
		t.Fatal("native SAVE/LOAD changed the withheld motion's state")
	}
	t.Log("entity 56's unanchored route stayed withheld and the unit stood across 200 ticks, LOAD, native SAVE and reload; Hash unchanged")
}
