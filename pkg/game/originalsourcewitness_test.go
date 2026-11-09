package game

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// originalSourceWitnessFile holds, as numbers, what each source an original
// file states yields from the synthetic inputs below: the groups in order,
// the application state fields, the selection ids, the refusal texts and the
// hash of the SAV bytes a resave writes. Each line is a row number, a step
// number and the row's numbers.
const originalSourceWitnessFile = "testdata/originalsourcewitness.golden"

// The rows and steps of the witness. A town row records its groups (steps
// from 100) and the bytes of the file a resave of it writes (step 1); a
// mission row its read application (step 2), first frame (step 3) and
// captures (steps 4 to 6); a projection row the leaves it wrote (step 7), the
// document's bytes (step 8) and its refusal (step 9).
const (
	sourceRowTownLoaded = iota + 1
	sourceRowTownCurrent
	sourceRowTownSecondGraph
	sourceRowTownSecondCurrent
	sourceRowTownEmpty
	sourceRowTownNoSource
	sourceRowMissionStore
	sourceRowMissionStoreWide
	sourceRowMissionStoreWhole
	sourceRowMissionSecondDocument
	sourceRowMissionNoStore
	sourceRowMissionRefusedIndex
	sourceRowMissionRefusedMissing
	sourceRowMissionDocument
	sourceRowProjectCurrent
	sourceRowProjectAbsent
	sourceRowProjectLocal
	sourceRowProjectRefused
	sourceRowDocumentRefused
)

const (
	sourceStepSave = iota + 1
	sourceStepRead
	sourceStepFrame
	sourceStepCapture
	sourceStepCaptureNil
	sourceStepCaptureLocal
	sourceStepProjected
	sourceStepDocument
	sourceStepRefusal
)

type originalSourceTrace struct{ partyCrossingTrace }

// numbers records v under a row and step, as the party crossings do.
func (w *originalSourceTrace) numbers(row, step int, v any) { w.line(row, step, v) }

// refusal records an error's text as its FNV hash, zero for none.
func (w *originalSourceTrace) refusal(row, step int, err error) {
	var text uint64
	if err != nil {
		nums := []string{err.Error()}
		text = partyCrossingHash(nums)
	}
	fmt.Fprintf(&w.b, "%d %d %d\n", row, step, text)
}

// bytes records the SHA-256 of b as two numbers and its length.
func (w *originalSourceTrace) bytes(row, step int, b []byte) {
	sum := sha256.Sum256(b)
	fmt.Fprintf(&w.b, "%d %d %d %d %d\n", row, step, len(b), binary.BigEndian.Uint64(sum[:8]), binary.BigEndian.Uint64(sum[8:16]))
}

// groups records each group in order: its keys, a hash of its fields and of
// each member's party ID, in order.
func (w *originalSourceTrace) groups(row int, groups []cityLiveGroup) {
	fmt.Fprintf(&w.b, "%d 99 %d\n", row, len(groups))
	for i, g := range groups {
		var nums []string
		partyCrossingNumbers(&nums, reflect.ValueOf(g.Payload))
		fmt.Fprintf(&w.b, "%d %d %d %d %d %d %d", row, 100+i, g.Payload.F1c, g.Payload.F40, g.Payload.F44, partyCrossingHash(nums), len(g.Members))
		for _, member := range g.Members {
			fmt.Fprintf(&w.b, " %d", partyCrossingHash([]string{member}))
		}
		w.b.WriteByte('\n')
	}
}

// missionOpen restores b on a fresh second-game front and opens its mission,
// returning the refusal of either step.
func originalSourceMissionOpen(t *testing.T, b []byte) (*FrontEnd, error) {
	t.Helper()
	f := secondMissionFrontOf(t, secondMissionContent{})
	open, _, err := f.RestoreOriginal(b)
	if err != nil {
		return f, err
	}
	_, _, _, _, _, _, _, _, _, _, err = open()
	return f, err
}

// sourceWitnessExport is the SAV bytes a mission or town snapshot writes.
func sourceWitnessExport(t *testing.T, f *FrontEnd, withMission bool) []byte {
	t.Helper()
	s, label, err := f.Snapshot(withMission)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// originalSourceWitness runs every source of the table over synthetic files.
func originalSourceWitness(t *testing.T) string {
	var w originalSourceTrace

	// Town arm: an original first-game town, the provenance reduced form.
	raw, base := cityProjectionSource(t)
	loaded := cityProjectionLoad(t, raw, base)
	w.groups(sourceRowTownLoaded, loaded.Town.cityGroups)
	current := cityProjectionSave(t, loaded)
	w.bytes(sourceRowTownLoaded, sourceStepSave, current)

	// Town arm: a current SAV of that town, bound through its document.
	cold := cityProjectionLoad(t, current, base)
	w.groups(sourceRowTownCurrent, cold.Town.cityGroups)
	w.bytes(sourceRowTownCurrent, sourceStepSave, cityProjectionSave(t, cold))

	// Town arm: a second-game town file, the actor graph as input.
	graph := coldIdentityTown(t, druidsTownFile())
	w.groups(sourceRowTownSecondGraph, graph.Town.cityGroups)
	second := exportTown(t, graph)
	w.bytes(sourceRowTownSecondGraph, sourceStepSave, second)

	// Town arm: an engine SAV of the second game's town.
	secondCold := coldIdentityTown(t, second)
	w.groups(sourceRowTownSecondCurrent, secondCold.Town.cityGroups)
	w.bytes(sourceRowTownSecondCurrent, sourceStepSave, exportTown(t, secondCold))

	// Town arm: a source that states no groups for this file.
	party := loaded.Carried
	w.groups(sourceRowTownEmpty, cityGroupsFromLoaded(party, nil))
	w.groups(sourceRowTownEmpty, cityGroupsFromLoaded(party, &SnapshotOriginalCity{Unavailable: "absent"}))
	w.groups(sourceRowTownEmpty, cityGroupsFromGraph(sav.SavedActorGraph{}, party[:1], nil))
	w.groups(sourceRowTownEmpty, cityGroupsFromCurrent(sav.CityData{}, nil))

	// Town arm: a current SAV whose provenance cannot be read holds no group
	// source at all, so the groups are the party's.
	src, err := decodeOriginalSource(current, mapload.ModContext{}, cold.Campaign.Value())
	if err != nil {
		t.Fatal(err)
	}
	if src.campaign.actions == nil {
		t.Fatal("the current SAV holds no current action leaf")
	}
	unreadable, err := sav.Open(secondFixtureTown("town", secondFixtureHero(), 44))
	if err != nil {
		t.Fatal(err)
	}
	src.sf = unreadable
	var session CampaignSession
	if _, err := session.restoreOriginalTown(src, cold.originalInstall(), cold.Campaign.Value()); err != nil {
		w.refusal(sourceRowTownNoSource, sourceStepRefusal, err)
	} else {
		w.groups(sourceRowTownNoSource, session.Town.cityGroups)
	}

	missionFile := func(store []byte) []byte { return secondMissionFile(secondMissionFixtureHero(), store) }

	// Mission arm: a file's state store, read at the LOAD and adopted on the
	// first frame; both speed rows of the existing first-frame witness.
	sourceWitnessStoreMission(t, &w, sourceRowMissionStore, missionFile(secondMissionStore(4, 0, -1)))
	sourceWitnessStoreMission(t, &w, sourceRowMissionStoreWide, missionFile(secondMissionStore(8, 1, 6)))

	// Mission arm: a file whose store holds every directory reads as the same
	// source, and the engine's own SAV of that mission, whose document is
	// complete, reads nothing from the store at the LOAD.
	whole, _, _ := secondMissionLoadFile(t, missionFile(secondMissionWholeStore(4, 0, -1)))
	w.numbers(sourceRowMissionStoreWhole, sourceStepRead, whole.live.mission.state.fileApplication != nil)
	sourceWitnessMissionState(t, &w, sourceRowMissionStoreWhole, whole)
	engineSAV := sourceWitnessExport(t, whole, true)
	w.bytes(sourceRowMissionStoreWhole, sourceStepSave, engineSAV)
	engine, _, _ := secondMissionLoadFile(t, engineSAV)
	w.numbers(sourceRowMissionSecondDocument, sourceStepRead, engine.live.mission.state.fileApplication != nil)
	sourceWitnessMissionState(t, &w, sourceRowMissionSecondDocument, engine)
	w.bytes(sourceRowMissionSecondDocument, sourceStepSave, sourceWitnessExport(t, engine, true))

	// Mission arm: a file with no world half states no store. The route's
	// answer, then the read moment on that file over a World that exists.
	noWorld := labelledContainer(sav.SecondGame.Magic, "mission", secondSavedWorld(10, 44, ptrTo(secondMissionFixtureHero()), false, 0),
		append(secondMissionStore(4, 0, -1), secondSavedTail(0, [2]uint32{10, placeMission})...))
	f, err := originalSourceMissionOpen(t, noWorld)
	w.refusal(sourceRowMissionNoStore, sourceStepRefusal, err)
	if err == nil {
		w.numbers(sourceRowMissionNoStore, sourceStepRead, f.live.mission.state.fileApplication != nil)
	}
	worldless, err := sav.Open(noWorld)
	if err != nil {
		t.Fatal(err)
	}
	readNothing, err := readFileApplication(worldless, whole.live.world)
	w.refusal(sourceRowMissionNoStore, sourceStepRefusal, err)
	w.numbers(sourceRowMissionNoStore, sourceStepRead, readNothing != nil)

	// Mission arm: the file's store is refused at the read moment.
	_, err = originalSourceMissionOpen(t, missionFile(secondMissionStore(4, 0, 99)))
	w.refusal(sourceRowMissionRefusedIndex, sourceStepRefusal, err)
	var missing []synth.RegNode
	for _, node := range secondMissionStoreNodes(4, 0, -1) {
		if node.Name != "View" {
			missing = append(missing, node)
		}
	}
	_, err = originalSourceMissionOpen(t, missionFile(synth.Reg(0x11, missing)))
	w.refusal(sourceRowMissionRefusedMissing, sourceStepRefusal, err)

	// First frame, capture and resave from a decoded document.
	open, app := itemMutationOpen(t, 1, false)
	fresh, _ := itemMutationCheckpoint(t, open, app, 1)
	document := fresh.live.mission.state.savedDocument
	ids, err := applicationActorIDs(document.Document, document.Actors, fresh.live.world)
	if err != nil {
		t.Fatal(err)
	}
	w.numbers(sourceRowMissionDocument, sourceStepRead, ids)
	sourceWitnessMissionState(t, &w, sourceRowMissionDocument, fresh)
	w.bytes(sourceRowMissionDocument, sourceStepSave, sourceWitnessExport(t, fresh, true))

	// Projection back into a document, each application state it reads.
	_, snapshot, world := applicationSnapshotFixture(t)
	project := func(row int, s Snapshot) {
		doc, err := sav.CloneDocumentData(*snapshot.SavedDocument.Document)
		if err != nil {
			t.Fatal(err)
		}
		if err := projectApplicationState(&doc, s, world); err != nil {
			w.refusal(row, sourceStepRefusal, err)
			return
		}
		state, err := readOriginalApplicationState(&doc)
		if err != nil {
			t.Fatal(err)
		}
		w.numbers(row, sourceStepProjected, state)
		encoded, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		w.bytes(row, sourceStepDocument, encoded)
	}
	selected := snapshot
	view := selected.ApplicationState.View
	view.Selection = []uint32{uint32(selected.SavedDocument.Actors[0].EntityID), uint32(selected.SavedDocument.Actors[1].EntityID)}
	view.ViewX, view.ViewY = 12, 13
	selected.ApplicationState = cloneApplicationState(selected.ApplicationState)
	selected.ApplicationState.View = view
	selected.QuickSpells = [4]uint32{23, 1, 0, 18}
	project(sourceRowProjectCurrent, selected)

	absent := snapshot
	absent.ApplicationState = nil
	absent.CameraSet, absent.CameraX, absent.CameraY, absent.CameraZoom = true, 12.75, 13.25, 2
	project(sourceRowProjectAbsent, absent)

	local := snapshot
	localView := localOnlyApplicationView(ui.SaveApplicationState{ShowHealth: true, FlyingHP: true, TimeFlow: true})
	local.ApplicationState = &SnapshotApplicationState{LocalOnly: true, Version: 1, View: localView, Baseline: localView,
		Original: OriginalStateData{Wimpy: 0, Formation: 2}}
	local.CameraSet, local.CameraX, local.CameraY, local.CameraZoom = true, 12.75, 13.25, 2
	project(sourceRowProjectLocal, local)

	refused := snapshot
	refusedApp := cloneApplicationState(snapshot.ApplicationState)
	refusedApp.View.Selection = []uint32{999999}
	refused.ApplicationState = refusedApp
	project(sourceRowProjectRefused, refused)
	bare := snapshot
	bare.SavedDocument = nil
	w.refusal(sourceRowProjectRefused, sourceStepRefusal, projectApplicationState(&sav.DocumentData{}, bare, world))
	w.refusal(sourceRowProjectRefused, sourceStepRefusal, projectApplicationState(nil, snapshot, world))

	// The document's refusal at the decode, before any game state changes.
	bad, err := sav.CloneDocumentData(*snapshot.SavedDocument.Document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := patchStateLeaf(bad.State.RootKind, bad.State.DirectoryRecords, bad.State.ValueRecords, "/SpellBook/Pressed", func(v *sav.CityStateValueData) error {
		v.Int32 = 1 << 20
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	w.refusal(sourceRowDocumentRefused, sourceStepRefusal, validateOriginalApplicationSource(&bad))
	w.refusal(sourceRowDocumentRefused, sourceStepRefusal, validateOriginalApplicationSource(nil))
	return w.b.String()
}

// sourceWitnessStoreMission loads a state-store mission through its one
// route and records its read, first frame and captures.
func sourceWitnessStoreMission(t *testing.T, w *originalSourceTrace, row int, b []byte) {
	t.Helper()
	f, _, _ := secondMissionLoadFile(t, b)
	file := f.live.mission.state.fileApplication
	if file == nil {
		t.Fatal("the mission file kept no state store application")
	}
	w.numbers(row, sourceStepRead, *file)
	sourceWitnessMissionState(t, w, row, f)
	w.bytes(row, sourceStepSave, sourceWitnessExport(t, f, true))
}

// sourceWitnessMissionState records the first frame the LOAD restored and
// the captures of it.
func sourceWitnessMissionState(t *testing.T, w *originalSourceTrace, row int, f *FrontEnd) {
	t.Helper()
	mw := f.live
	if mw == nil || mw.applicationState == nil || mw.fog == nil {
		t.Fatal("the LOAD restored no first frame")
	}
	w.numbers(row, sourceStepFrame, *mw.applicationState)
	w.bytes(row, sourceStepFrame, mw.fog.explored)
	saved := mw.applicationState
	capture := func(step int) {
		captured, err := f.captureApplicationState(mw)
		if err != nil {
			w.refusal(row, sourceStepRefusal, err)
			return
		}
		w.numbers(row, step, *captured)
	}
	capture(sourceStepCapture)
	mw.applicationState = nil
	capture(sourceStepCaptureNil)
	view := localOnlyApplicationView(ui.SaveApplicationState{ShowHealth: true, FlyingHP: false, TimeFlow: true})
	mw.applicationState = &SnapshotApplicationState{LocalOnly: true, Version: applicationStateVersion, View: view, Baseline: view,
		Original: OriginalStateData{Wimpy: 1, Formation: 5}}
	capture(sourceStepCaptureLocal)
	mw.applicationState = saved
}

// Each town fact picks its row in the table's order: a current file with a
// readable provenance binds through its document, the second game's file
// through its graph, an original through its provenance, and none of them
// leaves no source.
func TestTownSourceTableChoosesOneRowPerFile(t *testing.T) {
	bare := &originalSource{campaign: &originalCampaignDecode{}}
	second := &originalSource{campaign: &originalCampaignDecode{fileSession: &currentSessionData{}}}
	refused := errors.New("no provenance")
	for _, c := range []struct {
		name string
		in   originalTownInputs
		want string
	}{
		{"current with provenance", originalTownInputs{src: bare, hasCurrent: true}, "current SAV document"},
		{"current without provenance", originalTownInputs{src: bare, hasCurrent: true, cityErr: refused}, ""},
		{"current with provenance over a file session", originalTownInputs{src: second, hasCurrent: true}, "current SAV document"},
		{"current without provenance over a file session", originalTownInputs{src: second, hasCurrent: true, cityErr: refused}, "file session and actor graph"},
		{"no current over a file session", originalTownInputs{src: second}, "file session and actor graph"},
		{"an original with provenance", originalTownInputs{src: bare}, "city provenance"},
		{"an original without provenance", originalTownInputs{src: bare, cityErr: refused}, "city provenance"},
	} {
		got := ""
		if row := originalTownSource(c.in); row != nil {
			got = row.name
		}
		if got != c.want {
			t.Errorf("%s: row %q, want %q", c.name, got, c.want)
		}
	}
	rows := originalFileSources()
	names := map[string]bool{}
	for _, row := range rows {
		if row.name == "" || names[row.name] {
			t.Errorf("row %q is unnamed or repeated", row.name)
		}
		names[row.name] = true
		if (row.townHolds == nil) != (row.townGroups == nil) || (row.missionHolds == nil) != (row.readApplication == nil) || (row.frameHolds == nil) != (row.frame == nil) || (row.captureHolds == nil) != (row.captureApplication == nil) {
			t.Errorf("row %q holds a test without its reader", row.name)
		}
	}
	if len(rows) != 7 {
		t.Errorf("%d rows, want the three town sources, the state store, the document, capture and casters", len(rows))
	}
}

// TestOriginalSourceWitness pins every source of an original file's state
// over synthetic inputs, so a move of the source table shows here.
func TestOriginalSourceWitness(t *testing.T) {
	got := originalSourceWitness(t)
	if os.Getenv("AGAINROM_WRITE_ORIGINAL_SOURCE_WITNESS") != "" {
		if err := os.WriteFile(filepath.FromSlash(originalSourceWitnessFile), []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(filepath.FromSlash(originalSourceWitnessFile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.ReplaceAll(string(want), "\r", "") != got {
		t.Fatalf("the sources changed from %s:\n%s", originalSourceWitnessFile, partyCrossingFirstDiff(got, string(want)))
	}
}
