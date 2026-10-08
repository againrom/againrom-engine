//go:build sessioncorpusaudit

package game

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
)

// censusMissionFront loads a mission SAV the way a cold LOAD does and opens its
// mission, so the front end holds the loaded Document. A name "native:N" is
// the engine's own SAVE of a freshly started mission N, loaded back; any other
// name is a corpus file.
func censusMissionFront(t *testing.T, relative string) *FrontEnd {
	t.Helper()
	var raw []byte
	if n, ok := strings.CutPrefix(relative, "native:"); ok {
		mission, err := strconv.Atoi(n)
		if err != nil {
			t.Fatal(err)
		}
		raw = censusNativeSave(t, mission)
	} else {
		raw = censusCorpusFile(t, relative)
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	open, _, err := f.RestoreOriginal(raw)
	if err != nil || open == nil {
		t.Skipf("%s does not load as a mission here: open=%v err=%v", relative, open != nil, err)
	}
	app := f.App("census")
	app.Layout(1024, 768)
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	// Play on, so the live World no longer equals the loaded Document.
	if n, _ := strconv.Atoi(os.Getenv("AGAINROM_CENSUS_TICKS")); n > 0 {
		for i := 0; i < n; i++ {
			f.live.tick()
		}
	}
	return f
}

func censusCorpusFile(t *testing.T, relative string) []byte {
	t.Helper()
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Skip("no AGAINROM_SAVE_CORPUS: the writer census needs owner saves")
	}
	raw, err := os.ReadFile(filepath.Join(corpus, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// censusNativeSave starts mission n with the starting mage, plays a little and
// writes the engine's own mission SAVE.
func censusNativeSave(t *testing.T, mission int) []byte {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
	app := f.App("census-native")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(mission, party)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 120; i++ {
		f.live.tick()
	}
	raw, err := censusMissionSave(t, f)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// censusMissionPrepare perturbs a copy of the Document the mission was loaded
// with and swaps it in for one export.
func censusMissionPrepare(t *testing.T, f *FrontEnd, export censusExporter) func() *censusRun {
	return func() *censusRun {
		state := f.live.mission.state.savedDocument
		clone, err := sav.CloneDocumentData(*state.Document)
		if err != nil {
			t.Fatal(err)
		}
		prev := state.Document
		state.Document = &clone
		return &censusRun{leaves: censusLeaves(&clone), export: func() ([]byte, error) { return export(t, f) },
			done: func() { state.Document = prev }}
	}
}

type censusExporter func(t *testing.T, f *FrontEnd) ([]byte, error)

func censusMissionSave(t *testing.T, f *FrontEnd) ([]byte, error) {
	snap, label, err := f.Snapshot(true)
	if err != nil {
		return nil, err
	}
	return f.ExportCurrentSave(snap, label)
}

func censusMissionAutosave(t *testing.T, f *FrontEnd) ([]byte, error) {
	snap, _, err := f.Snapshot(true)
	if err != nil {
		return nil, err
	}
	view, captured := f.detachedExporter(snap)
	raw, _, err := view.playerMissionSave(captured, "census autosave")
	return raw, err
}

func TestReleaseWriterCensusMissionDocument(t *testing.T) {
	name := os.Getenv("AGAINROM_CENSUS_SAVE")
	if name == "" {
		t.Skip("no AGAINROM_CENSUS_SAVE")
	}
	f := censusMissionFront(t, name)
	tag := strings.NewReplacer("/", "_", ".sav", "", ":", "-").Replace(name)
	for _, kind := range []struct {
		name   string
		export censusExporter
	}{{"mission-save", censusMissionSave}, {"autosave", censusMissionAutosave}} {
		if want := os.Getenv("AGAINROM_CENSUS_KIND"); want != "" && want != kind.name {
			continue
		}
		env := &censusEnv{t: t, prepare: censusMissionPrepare(t, f, kind.export)}
		results := env.table(kind.name + "-" + tag)
		count := map[string]int{}
		for _, r := range results {
			count[r.source]++
		}
		t.Logf("%s: %d patterns %v", kind.name, len(results), count)
	}
}

func TestReleaseWriterCensusLiveMission(t *testing.T) {
	name := os.Getenv("AGAINROM_CENSUS_SAVE")
	if name == "" {
		t.Skip("no AGAINROM_CENSUS_SAVE")
	}
	f := censusMissionFront(t, name)
	match := os.Getenv("AGAINROM_CENSUS_MATCH")
	tag := strings.NewReplacer("/", "_", ".sav", "", ":", "-").Replace(name)
	results := liveCensus(t, f.live.world, func() ([]byte, error) { return censusMissionSave(t, f) }, func(p string) bool {
		return censusMatch(p, match)
	})
	writeLiveCensus(t, "live-"+tag, results)
	t.Logf("%d live regions", len(results))
}

// censusTownPrepare reloads the town SAV, takes a town Snapshot and exposes
// the Snapshot's carriers of loaded state to perturbation.
func censusTownPrepare(t *testing.T, f *FrontEnd, raw []byte) func() *censusRun {
	return func() *censusRun {
		if _, city, err := f.RestoreOriginal(raw); err != nil || !city {
			t.Fatalf("restore town: city=%v err=%v", city, err)
		}
		snap, label, err := f.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		var leaves []censusLeaf
		var writes []func()
		emit := func(pattern, instance string, v reflect.Value, topo bool) {
			leaves = append(leaves, censusLeaf{pattern, instance, v, topo})
		}
		root := reflect.ValueOf(&snap).Elem()
		for _, name := range []string{"OriginalCity", "CityObjects", "Campaign", "DocPayload", "Documents", "CampaignRecords", "OfferLabels", "LastMap"} {
			censusWalk(emit, root.FieldByName(name), name, "", &writes)
		}
		return &censusRun{leaves: leaves, export: func() ([]byte, error) {
			for _, write := range writes {
				write()
			}
			return f.ExportCurrentSave(snap, label)
		}, done: func() {}}
	}
}

func TestReleaseWriterCensusTownDocument(t *testing.T) {
	name := os.Getenv("AGAINROM_CENSUS_TOWN")
	if name == "" {
		t.Skip("no AGAINROM_CENSUS_TOWN")
	}
	raw := censusCorpusFile(t, name)
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	env := &censusEnv{t: t, prepare: censusTownPrepare(t, f, raw), foreign: true}
	tag := strings.NewReplacer("/", "_", ".sav", "", ":", "-").Replace(name)
	results := env.table("town-save-" + tag)
	count := map[string]int{}
	for _, r := range results {
		count[r.source]++
	}
	t.Logf("town: %d input patterns %v", len(results), count)
}

func TestReleaseWriterCensusCapturedState(t *testing.T) {
	name := os.Getenv("AGAINROM_CENSUS_SAVE")
	if name == "" {
		t.Skip("no explicit capture census source")
	}
	f := censusMissionFront(t, name)
	env := censusEnv{t: t, foreign: true, prepare: func() *censusRun {
		snap, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		var leaves []censusLeaf
		var writes []func()
		emit := func(p, i string, v reflect.Value, topo bool) { leaves = append(leaves, censusLeaf{p, i, v, topo}) }
		root := reflect.ValueOf(&snap).Elem()
		for _, field := range []string{"Party", "CurrentRoster", "Difficulty", "Fame", "QuickSpells", "Open", "Gold", "Won", "Available", "Taken", "Offered", "MercenaryPool", "MercenaryEnabled", "MercenaryHired", "HeroGrantState", "ConsumedHeroGrants", "ApplicationState", "CameraSet", "CameraX", "CameraY", "CameraZoom", "Campaign", "Residue", "CurrentPartyIDs", "WorldSelectedOnce", "CityObjects"} {
			censusWalk(emit, root.FieldByName(field), "Snapshot."+field, "", &writes)
		}
		return &censusRun{leaves: leaves, export: func() ([]byte, error) {
			for _, write := range writes {
				write()
			}
			return f.ExportCurrentSave(snap, label)
		}, done: func() {}}
	}}
	tag := strings.NewReplacer("/", "_", ".sav", "", ":", "-").Replace(name)
	rows := env.table("capture-" + tag)
	t.Logf("%d captured input patterns; maps include mutable value copies; nil/empty populations remain unwitnessed", len(rows))
}
