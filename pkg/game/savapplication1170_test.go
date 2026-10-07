package game

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func applicationDomainFixture1170() *SnapshotApplicationState {
	view := ui.SaveApplicationState{ViewX: 12, ViewY: 13, Zoom: 1, InventoryOpen: true, SpellBookOpen: true,
		DollOpen: true, WornOpen: true, MinimapOpen: true, ShowHealth: true, FlyingHP: true, TimeFlow: true,
		PressedSpell: 23, PeriodUS: 31000}
	return &SnapshotApplicationState{Version: 1, View: view, Baseline: view, WimpyBaseline: 0,
		Original: OriginalStateData{Wimpy: 7, ShowHP: -7, FlyingHP: 9, Formation: 31, Speed: 99,
			ShowTimeFlow: -3, InventoryOpen: 6, BookOpen: 5, Pressed: 5, ViewX: 12, ViewY: 13}}
}

func TestApplication1170RawDomainsOnlyChangeWhenTheirProducerChanges(t *testing.T) {
	app := applicationDomainFixture1170()
	want := app.Original
	got, err := applicationCurrentRaw(app)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("unchanged raw options = %+v, want %+v: %v", got, want, err)
	}
	app.View.ShowHealth, app.View.FlyingHP, app.View.TimeFlow = false, false, false
	app.View.InventoryOpen, app.View.SpellBookOpen = false, false
	app.View.PressedSpell, app.View.PeriodUS = 1, 83000
	app.View.ViewX, app.View.ViewY = 21, 22
	want.ShowHP, want.FlyingHP, want.ShowTimeFlow = 0, 0, 0
	want.InventoryOpen, want.BookOpen, want.Pressed, want.Speed = 0, 0, 0, 2
	want.ViewX, want.ViewY = 21, 22
	got, err = applicationCurrentRaw(app)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("changed raw options = %+v, want %+v: %v", got, want, err)
	}
	if app.Original.Formation != 31 || app.Original.Wimpy != 7 {
		t.Fatal("unrelated native defaults overwrote source-only domains")
	}
}

func applicationSnapshotFixture(t *testing.T) (*FrontEnd, Snapshot, *sim.World) {
	t.Helper()
	f := poolFixtureFront(t, 91, 92)
	f.Campaign = resolved(saveCampaign(), nil)
	runtimeA, runtimeB := uint32(1234007), uint32(2345009)
	actors := []*poolFixtureActor{
		{mapID: 91, cell: 0x100f, hp: 10, maxHP: 100, mana: 10, maxMana: 100, profile: literalProfile1107(), runtime: &runtimeA},
		{mapID: 92, cell: 0x1211, hp: 20, maxHP: 100, mana: 10, maxMana: 100, profile: literalProfile1107(), runtime: &runtimeB},
	}
	raw := completeDocumentTail1115(t, f, clockFixture1112(9343, 584, actors...))
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.SetStoreIntArray("Fog", "Data", []int32{1600}); err != nil {
		t.Fatal(err)
	}
	raw = file.Marshal()
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("fixture restore: %v", err)
	}
	if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
		t.Fatal(err)
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	var world sim.World
	if err := world.UnmarshalBinary(s.World); err != nil {
		t.Fatal(err)
	}
	return f, s, &world
}

func appLeafInts1170(t *testing.T, doc *sav.DocumentData, path string) []uint32 {
	t.Helper()
	value, err := readOriginalStateLeaf(doc, path, 6)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]uint32, len(value.Bytes)/4)
	for i := range out {
		out[i] = binary.LittleEndian.Uint32(value.Bytes[4*i:])
	}
	return out
}

func TestApplication1170ProjectsCurrentFogAndExactMultipleSelection(t *testing.T) {
	_, s, world := applicationSnapshotFixture(t)
	app := s.ApplicationState
	if len(s.SavedDocument.Actors) != 2 {
		t.Fatal("fixture has no two-actor binding")
	}
	app.View.Selection = []uint32{uint32(s.SavedDocument.Actors[0].EntityID), uint32(s.SavedDocument.Actors[1].EntityID)}
	app.View.ViewX, app.View.ViewY = 12, 13
	s.QuickSpells = [4]uint32{23, 1, 0, 18}
	s.Residue.FogExplored = make([]byte, 1600)
	s.Residue.FogVisible = make([]byte, 1600)
	s.Residue.FogExplored[0], s.Residue.FogExplored[1], s.Residue.FogExplored[1599] = 1, 1, 1
	s.Residue.FogVisible[1] = 1
	// Expected runs and runtime IDs are literal fixture facts, independent of
	// both the native DTO order and the application projection under test.
	wantRuns, wantSelection := []uint32{2, 1597, 1}, []uint32{1234007, 2345009}
	doc, err := sav.CloneDocumentData(*s.SavedDocument.Document)
	if err != nil {
		t.Fatal(err)
	}
	frozen := doc
	if err := projectApplicationState(&doc, s, world); err != nil {
		t.Fatal(err)
	}
	if got := appLeafInts1170(t, &doc, "/Fog/Data"); !slices.Equal(got, wantRuns) {
		t.Fatalf("Fog/Data = %v", got)
	}
	if first, _ := readOriginalStateLeaf(&doc, "/Fog/FirstState", 2); first.Int32 != 0x8000 {
		t.Fatalf("Fog starts %x", first.Int32)
	}
	if got := appLeafInts1170(t, &doc, "/Objects/Selection"); !slices.Equal(got, wantSelection) {
		t.Fatalf("Selection = %v, want %v", got, wantSelection)
	}
	if got := appLeafInts1170(t, &doc, "/SpellBook/Shortcuts"); !slices.Equal(got, []uint32{5, 0, 0xffffffff, 23}) {
		t.Fatalf("shortcut indices = %v", got)
	}
	if slices.Equal(appLeafInts1170(t, &frozen, "/Fog/Data"), wantRuns) || slices.Equal(appLeafInts1170(t, &frozen, "/Objects/Selection"), wantSelection) {
		t.Fatal("freezing the historical Document unexpectedly passes the current-state oracle")
	}
	out, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	read, err := sav.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	fog, has, err := read.Fog()
	if err != nil || !has || !bytes.Equal(fog.Cells, s.Residue.FogExplored) {
		t.Fatalf("encoded Fog differs: %v", err)
	}
	// A single lost Fog projection must fail even while every other current
	// application leaf, including exact selection, remains correct.
	mutant, err := sav.CloneDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	for i := range mutant.State.ValueRecords {
		r := &mutant.State.ValueRecords[i]
		if r.Path != "/Fog/Data" && r.Path != "/Fog/FirstState" {
			continue
		}
		for _, old := range frozen.State.ValueRecords {
			if old.Path == r.Path {
				*r = old
			}
		}
	}
	wire, err := sav.EncodeDocumentData(mutant)
	if err != nil {
		t.Fatal("Fog omission must be structurally valid", err)
	}
	lost, err := sav.Open(wire)
	if err != nil {
		t.Fatal(err)
	}
	lostFog, _, err := lost.Fog()
	if err != nil || bytes.Equal(lostFog.Cells, s.Residue.FogExplored) || !slices.Equal(appLeafInts1170(t, &mutant, "/Objects/Selection"), wantSelection) {
		t.Fatal("single-cause Fog loss control failed", err)
	}
}

func TestApplication1170ProjectionRefusalsAreAtomic(t *testing.T) {
	_, source, world := applicationSnapshotFixture(t)
	for _, tc := range []struct {
		name   string
		mutate func(*Snapshot, *sav.DocumentData)
	}{
		{"unknown entity", func(s *Snapshot, _ *sav.DocumentData) { s.ApplicationState.View.Selection = []uint32{999} }},
		{"wrong exact binding", func(s *Snapshot, _ *sav.DocumentData) {
			s.SavedDocument.Actors[0].ObjectIndex, s.SavedDocument.Actors[1].ObjectIndex = s.SavedDocument.Actors[1].ObjectIndex, s.SavedDocument.Actors[0].ObjectIndex
		}},
		{"short Fog", func(s *Snapshot, _ *sav.DocumentData) { s.Residue.FogExplored = s.Residue.FogExplored[:1] }},
		{"wrong Fog extent", func(s *Snapshot, _ *sav.DocumentData) { s.Residue.FogCols, s.Residue.FogRows = 20, 80 }},
		{"non-bit Fog", func(s *Snapshot, _ *sav.DocumentData) { s.Residue.FogExplored[0] = 2 }},
		{"invalid zoom", func(s *Snapshot, _ *sav.DocumentData) { s.ApplicationState.View.Zoom = 0 }},
		{"invalid speed", func(s *Snapshot, _ *sav.DocumentData) { s.ApplicationState.View.PeriodUS = -1 }},
		{"missing leaf", func(_ *Snapshot, d *sav.DocumentData) {
			for i, r := range d.State.ValueRecords {
				if r.Path == "/View/Y" {
					d.State.ValueRecords = slices.Delete(d.State.ValueRecords, i, i+1)
					break
				}
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := source
			s.ApplicationState = cloneApplicationState(source.ApplicationState)
			var err error
			s.SavedDocument, err = cloneSavedDocument(source.SavedDocument)
			if err != nil {
				t.Fatal(err)
			}
			s.Residue.FogExplored = slices.Clone(source.Residue.FogExplored)
			doc, err := sav.CloneDocumentData(*source.SavedDocument.Document)
			if err != nil {
				t.Fatal(err)
			}
			tc.mutate(&s, &doc)
			before := append([]sav.CityStateRecordData(nil), doc.State.ValueRecords...)
			if err := projectApplicationState(&doc, s, world); err == nil {
				t.Fatal("invalid projection accepted")
			}
			if !reflect.DeepEqual(doc.State.ValueRecords, before) {
				t.Fatal("failed projection partially changed state")
			}
		})
	}
}

func TestApplication1170RejectsMalformedOriginalRunsBeforeAllocation(t *testing.T) {
	_, source, _ := applicationSnapshotFixture(t)
	for _, tc := range []struct {
		name  string
		runs  []uint32
		cells int
	}{
		{"negative", []uint32{0xffffffff}, 1600}, {"over-bound", []uint32{1 << 30}, 1600},
		{"empty", nil, 1600}, {"short", []uint32{1599}, 1600},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, _ := sav.CloneDocumentData(*source.SavedDocument.Document)
			for i := range doc.State.ValueRecords {
				if doc.State.ValueRecords[i].Path == "/Fog/Data" {
					data := make([]byte, 4*len(tc.runs))
					for j, run := range tc.runs {
						binary.LittleEndian.PutUint32(data[4*j:], run)
					}
					doc.State.ValueRecords[i].Value.Bytes = data
				}
			}
			if _, err := originalFogPlane(&doc, tc.cells); err == nil || !strings.Contains(err.Error(), "Fog") {
				t.Fatalf("bad Fog admitted: %v", err)
			}
		})
	}
}

func TestApplication1170NativeCheckpointPreservesApplication(t *testing.T) {
	f, _, _ := sackObjectsOpen1115(t, nil)
	f.live.tick()
	before, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := EncodeSave(before, "application checkpoint")
	if err != nil {
		t.Fatal(err)
	}
	saved, _, err := DecodeSave(raw)
	if err != nil {
		t.Fatal(err)
	}
	cold := cellStateFront(t)
	open, town, err := cold.Restore(saved)
	if err != nil || town {
		t.Fatal("application checkpoint restore", err)
	}
	if err := cold.App("application checkpoint").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	after, _, err := cold.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	a, b := reflect.ValueOf(before), reflect.ValueOf(after)
	for i := 0; i < a.NumField(); i++ {
		if !a.Field(i).CanInterface() {
			continue
		}
		if !reflect.DeepEqual(a.Field(i).Interface(), b.Field(i).Interface()) {
			t.Errorf("checkpoint %s changed: before %#v, after %#v", a.Type().Field(i).Name, a.Field(i).Interface(), b.Field(i).Interface())
		}
	}
}

func TestApplicationLegacyCapturesCurrentViewWithoutRestoringHistoricalOptions(t *testing.T) {
	f, source, _ := applicationSnapshotFixture(t)
	source.ApplicationState = nil // Controlled predecessor without this DTO.
	for i := range source.SavedDocument.Document.State.ValueRecords {
		record := &source.SavedDocument.Document.State.ValueRecords[i]
		if record.Path == "/Objects/Selection" {
			record.Value.Bytes = make([]byte, 4)
			binary.LittleEndian.PutUint32(record.Value.Bytes, 1234007)
		}
	}
	raw, err := EncodeSave(source, "legacy application absence")
	if err != nil {
		t.Fatal(err)
	}
	saved, _, err := DecodeSave(raw)
	if err != nil {
		t.Fatal(err)
	}
	if saved.ApplicationState != nil {
		t.Fatal("legacy decode invented an application checkpoint")
	}
	open, town, err := f.Restore(saved)
	if err != nil || town {
		t.Fatal("legacy application restore", err)
	}
	if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
		t.Fatal(err)
	}
	view := f.live.view.SaveApplication()
	view.Selection = nil
	if err := f.live.view.RestoreSaveApplication(view); err != nil {
		t.Fatal(err)
	}
	current, _, err := f.Snapshot(true)
	if err != nil || current.ApplicationState == nil || !reflect.DeepEqual(current.ApplicationState.View, view) {
		t.Fatal("legacy capture did not use the current viewer", err)
	}
	if f.live.applicationState != nil {
		t.Fatal("capture mutated the live application checkpoint")
	}
	if err := projectApplicationState(current.SavedDocument.Document, current, f.live.world); err != nil {
		t.Fatal(err)
	}
	written, err := readOriginalApplicationState(current.SavedDocument.Document)
	if err != nil || len(written.Selection) != 0 {
		t.Fatal("historical selection replaced the empty current selection", written, err)
	}
}

func TestApplication1170OriginalReferenceAndFogRefusalsKeepActiveSession(t *testing.T) {
	f, before, _ := applicationSnapshotFixture(t)
	live := f.live
	valid, err := sav.EncodeDocumentData(*before.SavedDocument.Document)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, section, key string
		values             []int32
	}{
		{"unbound selection", "Objects", "Selection", []int32{1987654321}},
		{"repeated selection", "Objects", "Selection", []int32{1234007, 1234007}},
		{"short Fog", "Fog", "Data", []int32{1599}},
		{"over-bound Fog", "Fog", "Data", []int32{1 << 30}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := sav.Open(valid)
			if err != nil {
				t.Fatal(err)
			}
			if err := file.SetStoreIntArray(tc.section, tc.key, tc.values); err != nil {
				t.Fatal(err)
			}
			open, town, err := f.RestoreOriginal(file.Marshal())
			if err == nil && !town && open != nil {
				_, _, _, _, _, _, _, _, _, _, err = open()
			}
			if err == nil {
				t.Fatal("invalid application was admitted")
			}
			after, _, err := f.Snapshot(true)
			if err != nil || f.live != live || !reflect.DeepEqual(before, after) {
				t.Fatal("refused application changed the active session", err)
			}
		})
	}
}

func TestApplication1170BetweenSamplesExplorationHasNoVisibilityLeaf(t *testing.T) {
	f, _, _ := sackObjectsOpen1115(t, nil)
	f.live.fog.refresh(f.live.world, sim.SelfSlot)
	// This fixture's actor crosses a cell between scheduled fog samples.
	// Capture its sampled planes independently, then advance actual movement.
	wantExplored := bytes.Clone(f.live.fog.explored)
	wantVisible := bytes.Clone(f.live.fog.visible)
	f.live.tick()
	if !bytes.Equal(f.live.fog.explored, wantExplored) || !bytes.Equal(f.live.fog.visible, wantVisible) {
		t.Fatal("fixture crossed a scheduled visibility sample")
	}
	derived := make([]byte, len(wantExplored))
	for i, lit := range f.live.world.Sight(sim.SelfSlot) {
		if lit != 0 {
			derived[i] = wantExplored[i]
		}
	}
	if bytes.Equal(derived, wantVisible) {
		t.Fatal("fixture does not distinguish sampled visibility from cold derived sight")
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.CloneDocumentData(*s.SavedDocument.Document)
	if err != nil {
		t.Fatal(err)
	}
	if err := projectApplicationState(&doc, s, f.live.world); err != nil {
		t.Fatal(err)
	}
	got, err := originalFogPlane(&doc, len(wantExplored))
	if err != nil || !bytes.Equal(got, wantExplored) {
		t.Fatal("YA1 changed current exploration", err)
	}
	// TERR-FOG-145 gives original SAV no bit-14 storage. Replacing only the
	// sampled visibility cannot change any application leaf in this format.
	other, _ := sav.CloneDocumentData(*s.SavedDocument.Document)
	s.Residue.FogVisible = derived
	if err := projectApplicationState(&other, s, f.live.world); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(doc.State, other.State) {
		t.Fatal("application invented an original visibility leaf")
	}
}
