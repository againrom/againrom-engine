package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func secondMissionSaveDirectory(t *testing.T) string {
	t.Helper()
	root := os.Getenv("AGAINROM_SECOND_SAVE_OUTPUT")
	if root == "" {
		t.Fatal("AGAINROM_SECOND_SAVE_OUTPUT is required for ROM2 save witnesses")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("save witness output must be absolute")
	}
	dir := filepath.Join(root, filepath.Base(os.Getenv("AGAINROM_ASSETS")), strings.ReplaceAll(t.Name(), "/", "_"))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	attempt, err := os.MkdirTemp(dir, "attempt-")
	if err != nil {
		t.Fatal(err)
	}
	return attempt
}

func secondMissionMove(t *testing.T, app *ui.App, id sim.EntityID, x, y int) {
	t.Helper()
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	px, py := secondMissionCellPoint(t, app, x, y)
	if err := app.HeadlessPointer("press", px, py); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessPointer("release", px, py); err != nil {
		t.Fatal(err)
	}
}

func secondMissionCellPoint(t *testing.T, app *ui.App, col, row int) (int, int) {
	t.Helper()
	for y := 0; y < 768; y += 4 {
		for x := 0; x < 1024; x += 4 {
			cx, cy, err := app.HeadlessDropCell(x, y)
			if err == nil && cx == col && cy == row {
				return x, y
			}
		}
	}
	t.Fatalf("cell %d,%d outside ordinary current view", col, row)
	return 0, 0
}

func secondMissionSettled(t *testing.T) (*FrontEnd, *ui.App) {
	return secondMissionSettledAt(t, 10)
}

func secondMissionSettledAt(t *testing.T, mission int) (*FrontEnd, *ui.App) {
	t.Helper()
	f := secondGameFront(t)
	app := f.App("first mission save")
	app.Layout(1024, 768)
	enterSecondCampaignMission(t, app)
	if mission == 20 {
		enterSecondCampaignNextMission(t, f, app, false)
	}
	for range 20 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	dismissSecondMissionDialogue(t, app, f.live)
	id := f.live.mission.ids[0]
	e, _ := f.live.world.Entity(id)
	secondMissionMove(t, app, id, int(e.X)-1, int(e.Y))
	for range 8 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	moved, _ := f.live.world.Entity(id)
	if moved.X == e.X && moved.Y == e.Y && moved.Transit == 0 {
		t.Fatal("ordinary move changed no actor action state")
	}
	if f.live.world.Outcome() != sim.OutcomeUndecided || app.HeadlessNoticeOpen() {
		t.Fatal("ordinary move did not reach settled nonterminal state")
	}
	return f, app
}

func secondMissionNamedSave(t *testing.T, f *FrontEnd, app *ui.App, out, name string) []byte {
	t.Helper()
	f.ConfigureSaveSeams(app, SaveStore{Dir: out}, OriginalStore{}, nil)
	if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenSave {
		t.Fatal("ordinary named SAVE dialog", err, app.Screen())
	}
	if err := app.HeadlessSaveEdit(out, name, ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	hash := f.live.world.Hash()
	before, _, captureErr := f.Snapshot(true)
	if captureErr != nil {
		t.Fatal(captureErr)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		_, fileErr := os.Stat(filepath.Join(out, name+".sav"))
		after, _, captureErr := f.Snapshot(true)
		entries, dirErr := os.ReadDir(out)
		t.Fatalf("ordinary SAVE refusal: %v file=%v world_unchanged=%v state_unchanged=%v capture=%v dir=%v entries=%v", err, fileErr, hash == f.live.world.Hash(), reflect.DeepEqual(before, after), captureErr, dirErr, entries)
	}
	if app.Screen() != ui.ScreenGameMenu || app.HeadlessMessage() != f.Words.SaveAcknowledgement {
		t.Fatal("ordinary SAVE acknowledgement", app.Screen(), app.HeadlessMessage())
	}
	raw, err := os.ReadFile(filepath.Join(out, name+".sav"))
	if err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("return"); err != nil {
		t.Fatal(err)
	}
	if hash != f.live.world.Hash() {
		t.Fatal("SAVE changed live World")
	}
	return raw
}

func secondMissionCold(t *testing.T, out, name string) (*FrontEnd, *ui.App) {
	return secondMissionColdAt(t, out, name, 10)
}

func secondMissionColdAt(t *testing.T, out, name string, mission int) (*FrontEnd, *ui.App) {
	t.Helper()
	cold := secondGameFront(t)
	app := cold.App("cold first mission")
	app.Layout(1024, 768)
	cold.ConfigureSaveSeams(app, SaveStore{Dir: out}, OriginalStore{}, nil)
	_, list, _ := cold.SaveSeams(SaveStore{Dir: out}, OriginalStore{}, nil)
	groundAppLoad(t, app, list, localOriginalSaveToken(name))
	if !cold.live.mission.resumed || cold.liveMission != mission {
		t.Fatal("LOAD reran fresh mission entry")
	}
	return cold, app
}

type secondPoseSample struct {
	ID                     uint32
	Name, Art              []byte
	Geometry               string
	Cell, Step             image.Point
	Frame                  [32]byte
	Mirror                 bool
	WalkDistance, WalkTick int
	Previous               *image.Point
	FinePosition           bool
	FineX, FineY           uint8
	Transit, TransitSpan   int
	DamageJolt             image.Point
}

type secondSaveSample struct {
	World               []byte
	Hash                uint64
	View                ui.SaveApplicationState
	Campaign            *currentSecondCampaign
	Gold                int
	Quick               [4]uint32
	Party               [][]byte
	Objectives          string
	Explored, Visible   []byte
	Pending             []sim.Command
	Ignored             []bool
	Notices             []int32
	HUD, Minimap        [32]byte
	Poses               []secondPoseSample
	ViewportPNG         []byte
	Animation           ui.AnimationState
	Sun                 string
	Viewport, Under     [32]byte
	PartyIDs            []sim.EntityID
	Won                 []int
	Offered             int
	Program             *currentScriptProgram
	Policy              *sim.CurrentROM2Policy
	Registers           [100]int32
	Latches             [1000]bool
	WinCount, LossCount uint32
	Actions             sim.ActionContinuations
}

func secondSaveSampleNow(t *testing.T, f *FrontEnd, app *ui.App) secondSaveSample {
	t.Helper()
	raw, err := f.live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	s := secondSaveSample{World: raw, Hash: f.live.world.Hash(), View: f.live.view.SaveApplication(), Campaign: captureSecondCampaign(f.Town.second), Gold: f.Town.gold, Quick: f.quickSpells, Objectives: f.live.secondGameObjectives(""), Explored: bytes.Clone(f.live.fog.explored), Visible: bytes.Clone(f.live.fog.visible), Pending: append([]sim.Command(nil), f.live.pending...), Ignored: append([]bool(nil), f.live.pendingIgnored...), Notices: append([]int32(nil), f.live.mission.pendingMessages...)}
	for _, drawn := range f.live.entityDraws() {
		pose := secondPoseSample{ID: drawn.ID, Name: []byte(drawn.Name), Cell: drawn.Cell, Step: drawn.Step, Mirror: drawn.Mirror, FinePosition: drawn.FinePosition, FineX: drawn.FineX, FineY: drawn.FineY, Transit: drawn.Transit, TransitSpan: drawn.TransitSpan, DamageJolt: drawn.DamageJolt}
		if drawn.Art != nil {
			pose.Art = []byte(drawn.Art.Name)
			pose.Geometry = fmt.Sprintf("%d/%d/%d/%d", drawn.Art.Width, drawn.Art.Height, drawn.Art.CenterX, drawn.Art.CenterY)
		}
		if drawn.Frame != nil {
			pose.Frame = sha256.Sum256(drawn.Frame.RGBA().Pix)
		}
		clock := f.live.walk[sim.EntityID(drawn.ID)]
		pose.WalkDistance, pose.WalkTick = clock.dist, clock.tick
		if cell, exists := f.live.prev[sim.EntityID(drawn.ID)]; exists {
			pose.Previous = &cell
		}
		s.Poses = append(s.Poses, pose)
	}
	s.PartyIDs = append([]sim.EntityID(nil), f.live.mission.ids...)
	for _, p := range mapload.CarryParty(f.live.mission.party, f.live.world, f.live.mission.ids) {
		var raw bytes.Buffer
		if err := gob.NewEncoder(&raw).Encode(p); err != nil {
			t.Fatal(err)
		}
		s.Party = append(s.Party, raw.Bytes())
	}
	for id, won := range f.Town.won {
		if won {
			s.Won = append(s.Won, id)
		}
	}
	sort.Ints(s.Won)
	s.Offered = f.Offered
	program := f.live.world.Script()
	s.Program = &currentScriptProgram{Dialect: program.Dialect(), Checks: program.Checks(), Instants: program.Instants(), Triggers: program.Triggers()}
	s.Policy = f.live.world.CurrentPolicy().ROM2
	s.Registers = f.live.world.ScriptRegisters()
	s.WinCount, s.LossCount = f.live.world.ScriptCounters()
	for i := range s.Latches {
		s.Latches[i] = f.live.world.ScriptLatched(int32(i))
	}
	s.Actions = f.live.world.Actions()
	viewport, under, err := f.live.view.HeadlessStatusBarFrame()
	if err != nil {
		t.Fatal(err)
	}
	s.Viewport, s.Under = sha256.Sum256(viewport.Pix), sha256.Sum256(under.Pix)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, viewport); err != nil {
		t.Fatal(err)
	}
	s.ViewportPNG = encoded.Bytes()
	s.Animation = f.live.view.SaveAnimation()
	s.Sun = fmt.Sprintf("%+v", f.live.view.Sun())
	hud, err := app.HeadlessBottomHUD()
	if err != nil {
		t.Fatal(err)
	}
	s.HUD = sha256.Sum256(hud.Pix)
	mini, err := app.HeadlessMinimap()
	if err != nil {
		t.Fatal(err)
	}
	s.Minimap = sha256.Sum256(mini.Pix)
	return s
}
func secondAssertSample(t *testing.T, want, got secondSaveSample) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Logf("other discriminators animation=%+v/%+v sun=%s/%s poses=%v registers=%v latches=%v counters=%d,%d/%d,%d actions=%v ignored=%v gold=%d/%d quick=%v/%v", want.Animation, got.Animation, want.Sun, got.Sun, reflect.DeepEqual(want.Poses, got.Poses), want.Registers == got.Registers, want.Latches == got.Latches, want.WinCount, want.LossCount, got.WinCount, got.LossCount, reflect.DeepEqual(want.Actions, got.Actions), reflect.DeepEqual(want.Ignored, got.Ignored), want.Gold, got.Gold, want.Quick, got.Quick)
		for i, pose := range want.Poses {
			if i < len(got.Poses) && !reflect.DeepEqual(pose, got.Poses[i]) {
				t.Logf("pose difference before=%+v after=%+v", pose, got.Poses[i])
			}
		}
		if want.Viewport != got.Viewport {
			out := secondMissionSaveDirectory(t)
			_ = os.WriteFile(filepath.Join(out, "CPUWant.png"), want.ViewportPNG, 0600)
			_ = os.WriteFile(filepath.Join(out, "CPUGot.png"), got.ViewportPNG, 0600)
			t.Logf("viewport mismatch evidence %s; animation=%+v/%+v sun=%s/%s", out, want.Animation, got.Animation, want.Sun, got.Sun)
		}
		t.Fatalf("cold continuation differs: World=%v hash=%x/%x View=%+v/%+v campaign=%v party=%v objectives=%v fog=%v/%v pending=%v notices=%v HUD=%v minimap=%v viewport=%v/%v policy=%v program=%v partyIDs=%v won=%v offered=%d/%d", bytes.Equal(want.World, got.World), want.Hash, got.Hash, want.View, got.View, reflect.DeepEqual(want.Campaign, got.Campaign), reflect.DeepEqual(want.Party, got.Party), want.Objectives == got.Objectives, bytes.Equal(want.Explored, got.Explored), bytes.Equal(want.Visible, got.Visible), reflect.DeepEqual(want.Pending, got.Pending), reflect.DeepEqual(want.Notices, got.Notices), want.HUD == got.HUD, want.Minimap == got.Minimap, want.Viewport == got.Viewport, want.Under == got.Under, reflect.DeepEqual(want.Policy, got.Policy), reflect.DeepEqual(want.Program, got.Program), reflect.DeepEqual(want.PartyIDs, got.PartyIDs), reflect.DeepEqual(want.Won, got.Won), want.Offered, got.Offered)
	}
}

type secondSaveProof struct {
	Samples []secondSaveSample
	Actor   sim.EntityID
	X, Y    int
	Ticks   int
	Mission int
}

func secondMissionColdProof(t *testing.T, path string) {
	raw, err := os.ReadFile(path + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var p secondSaveProof
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	mission := p.Mission
	if mission == 0 {
		mission = 10
	}
	f, app := secondMissionColdAt(t, filepath.Dir(path), filepath.Base(path), mission)
	secondAssertSample(t, p.Samples[0], secondSaveSampleNow(t, f, app))
	secondMissionMove(t, app, p.Actor, p.X, p.Y)
	for range p.Ticks {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	secondAssertSample(t, p.Samples[1], secondSaveSampleNow(t, f, app))
	t.Log("fresh-process ordinary App LOAD, first presentation, identical ordinary pointer command and ticks")
}

func TestReleaseSecondMissionNamedSaveContinuation(t *testing.T) {
	secondMissionNamedContinuation(t, 10)
}

func TestReleaseSecondMissionTwentyNamedSaveContinuation(t *testing.T) {
	secondMissionNamedContinuation(t, 20)
}

func secondMissionNamedContinuation(t *testing.T, mission int) {
	secondGameRoot(t)
	if path := os.Getenv("AGAINROM_SECOND_SAVE_INPUT"); path != "" {
		secondMissionColdProof(t, path)
		return
	}
	out := secondMissionSaveDirectory(t)
	f, app := secondMissionSettledAt(t, mission)
	before := secondSaveSampleNow(t, f, app)
	capture, err := json.Marshal(before)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "CaptureBefore.json"), capture, 0600); err != nil {
		t.Fatal(err)
	}
	state, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	stateRaw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "SnapshotBefore.json"), stateRaw, 0600); err != nil {
		t.Fatal(err)
	}
	frame, _, err := f.live.view.HeadlessStatusBarFrame()
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(out, "CPUBefore.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, frame); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	raw := secondMissionNamedSave(t, f, app, out, "First mission")
	cold, a := secondMissionColdAt(t, out, "First mission.sav", mission)
	secondAssertSample(t, before, secondSaveSampleNow(t, cold, a))
	assertCurrentWorldEqual(t, f.live.world, cold.live.world, "cold LOAD")
	id := f.live.mission.ids[0]
	e, _ := f.live.world.Entity(id)
	x, y := int(e.X), int(e.Y)+1
	secondMissionMove(t, app, id, x, y)
	secondMissionMove(t, a, id, x, y)
	for range 24 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		assertCurrentWorldEqual(t, f.live.world, cold.live.world, "post-load ordinary action")
	}
	after := secondSaveSampleNow(t, f, app)
	secondAssertSample(t, after, secondSaveSampleNow(t, cold, a))
	if bytes.Equal(before.World, after.World) {
		t.Fatal("next ordinary action did not change World")
	}
	proof := secondSaveProof{Samples: []secondSaveSample{before, after}, Actor: id, X: x, Y: y, Ticks: 24, Mission: mission}
	encoded, err := json.Marshal(proof)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(out, "First mission.sav")
	if err := os.WriteFile(path+".json", encoded, 0600); err != nil {
		t.Fatal(err)
	}
	runSpellWitnessChild(t, path, "AGAINROM_SECOND_SAVE_INPUT")
	secondMissionNamedSave(t, cold, a, out, "Continued mission")
	again, b := secondMissionColdAt(t, out, "Continued mission.sav", mission)
	secondAssertSample(t, after, secondSaveSampleNow(t, again, b))
	t.Logf("ordinary changed M%d -> named %d-byte SAV -> cold App first frame -> 24 identical post-action ticks -> resave/cold LOAD", mission, len(raw))
}

func TestReleaseSecondMissionSaveEntryPoints(t *testing.T) {
	secondMissionSaveEntries(t, 10)
}

func TestReleaseSecondMissionTwentySaveEntryPoints(t *testing.T) {
	secondMissionSaveEntries(t, 20)
}

func secondMissionSaveEntries(t *testing.T, mission int) {
	secondGameRoot(t)
	out := secondMissionSaveDirectory(t)
	f := secondGameFront(t)
	app := f.App("automatic first mission")
	app.Layout(1024, 768)
	f.Options = OptionsStore{Path: filepath.Join(out, "options.txt")}
	now := time.Unix(100, 0)
	f.ConfigureSaveSeams(app, SaveStore{Dir: out}, OriginalStore{}, func() time.Time { return now })
	defer app.FlushBackground()
	enterSecondCampaignMission(t, app)
	if mission == 20 {
		enterSecondCampaignNextMission(t, f, app, false)
	}
	row := missionAutosaveRow(t, f, SaveStore{Dir: out}, mission)
	cold, a := secondMissionColdAt(t, out, row.Name, mission)
	secondAssertSample(t, secondSaveSampleNow(t, f, app), secondSaveSampleNow(t, cold, a))
	for range 20 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	dismissSecondMissionDialogue(t, app, f.live)
	id := f.live.mission.ids[0]
	e, _ := f.live.world.Entity(id)
	secondMissionMove(t, app, id, int(e.X)-1, int(e.Y))
	for range 8 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	captured := secondSaveSampleNow(t, f, app)
	if err := app.HeadlessKey("f4"); err != nil {
		t.Fatal(err)
	}
	quick := string(quickSaveBase(0)) + ".sav"
	cold, a = secondMissionColdAt(t, out, quick, mission)
	secondAssertSample(t, captured, secondSaveSampleNow(t, cold, a))
	for range 4 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessKey("f9"); err != nil {
		t.Fatal(err)
	}
	secondAssertSample(t, captured, secondSaveSampleNow(t, f, app))
	save, _, _ := f.SaveSeams(SaveStore{Dir: out}, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil {
		t.Fatal(err)
	}
	cold, a = secondMissionColdAt(t, out, name, mission)
	secondAssertSample(t, captured, secondSaveSampleNow(t, cold, a))
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := f.ExportCurrentSave(snapshot, "same producer")
	if err != nil {
		t.Fatal(err)
	}
	for _, export := range []func(Snapshot, string) ([]byte, error){f.ExportCurrentWorldSave, f.ExportNativeCitySave, f.ExportOriginalSave} {
		raw, err := export(snapshot, "same producer")
		if err != nil || !bytes.Equal(raw, reference) {
			t.Fatal("compatibility API differs from sole producer", err)
		}
	}
	view, detached := f.detachedExporter(snapshot)
	old := *f.Town.second
	oldWorld, err := f.live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	oldParty := mapload.CloneParty(f.live.mission.party)
	oldView := f.live.view.SaveApplication()
	oldAnimation := f.live.view.SaveAnimation()
	oldWalk, oldPrevious := f.live.walk, f.live.prev
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	for range 16 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessGameMenuAction("return"); err != nil {
		t.Fatal(err)
	}
	if f.live.view.SaveAnimation() != oldAnimation || f.live.world.Hash() != captured.Hash {
		t.Fatal("held menu wall span advanced World or presentation phase")
	}
	f.Town.second.bank[991] = 4321
	f.Town.second.available = nil
	bank, _ := f.live.world.ROM2ScenarioState()
	bank[1000] ^= 77
	f.live.world.SetROM2ScenarioState(bank)
	f.live.mission.party[0].Name += " changed later"
	changedView := oldView
	changedView.ViewX += 0.25
	if err := f.live.view.RestoreCamera(changedView.ViewX, changedView.ViewY, changedView.Zoom); err != nil {
		t.Fatal(err)
	}
	if err := f.live.view.RestoreAnimation(ui.AnimationState{Count: oldAnimation.Count + 77, RemainderUS: oldAnimation.RemainderUS + 1}); err != nil {
		t.Fatal(err)
	}
	f.live.walk = map[sim.EntityID]walkClock{id: {dist: 991, tick: 12}}
	f.live.prev = map[sim.EntityID]image.Point{id: image.Pt(3, 4)}
	detachedRaw, err := view.ExportCurrentSave(detached, "same producer")
	if err != nil || !bytes.Equal(detachedRaw, reference) {
		t.Fatal("detached autosave read later live campaign", err)
	}
	*f.Town.second = old
	if err := f.live.world.UnmarshalBinary(oldWorld); err != nil {
		t.Fatal(err)
	}
	f.live.mission.party = oldParty
	f.live.walk, f.live.prev = oldWalk, oldPrevious
	if err := f.live.view.RestoreAnimation(oldAnimation); err != nil {
		t.Fatal(err)
	}
	if err := f.live.view.RestoreCamera(oldView.ViewX, oldView.ViewY, oldView.Zoom); err != nil {
		t.Fatal(err)
	}
	var timedBefore *secondSaveSample
	f.ConfigureSaveSeams(app, SaveStore{Dir: out}, OriginalStore{}, func() time.Time {
		if now.After(time.Unix(100, 0)) {
			s := secondSaveSampleNow(t, f, app)
			timedBefore = &s
		}
		return now
	})
	now = now.Add(5 * time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	app.FlushBackground()
	timedName := string(timedSaveBase(0)) + ".sav"
	if timedBefore == nil {
		t.Fatal("no pre-publication sample")
	}
	cold, a = secondMissionColdAt(t, out, timedName, mission)
	secondAssertSample(t, *timedBefore, secondSaveSampleNow(t, cold, a))
	t.Log("mission-start autosave tick0; F4/cold LOAD; F9; SaveSeams; three compat exporters; detached campaign mutation; timed autosave/cold LOAD")
}

func secondChangeSave(t *testing.T, raw []byte, change func(*currentActionData)) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	change(a)
	leaf, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(leaf)
	doc.Label = []byte(fmt.Sprintf("%s %x", t.Name(), digest[:8]))
	if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
		t.Fatal(err)
	}
	result, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestReleaseSecondMissionSaveLossControls(t *testing.T) {
	secondMissionSaveLosses(t, 10)
}

func TestReleaseSecondMissionTwentySaveLossControls(t *testing.T) {
	secondMissionSaveLosses(t, 20)
}

func secondMissionSaveLosses(t *testing.T, mission int) {
	secondGameRoot(t)
	out := secondMissionSaveDirectory(t)
	f, app := secondMissionSettledAt(t, mission)
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, "loss controls")
	if err != nil {
		t.Fatal(err)
	}
	for _, loss := range []string{"dialect", "bank absent", "campaign absent", "game absent", "game unknown", "current destination", "available overflow", "fog absent", "fog extent", "fog plane", "activity duplicate", "unavailable", "foreign kind", "unsupported pair"} {
		t.Run(loss, func(t *testing.T) {
			changed := secondChangeSave(t, raw, func(a *currentActionData) {
				switch loss {
				case "dialect":
					a.Program.Dialect = sim.ScriptROM1
				case "bank absent":
					a.Policy.ROM2 = nil
				case "campaign absent":
					a.Session.Second = nil
				case "game absent":
					a.Session.Game = ""
				case "game unknown":
					a.Session.Game = "other"
				case "current destination":
					a.Session.Second.Current.ID = 30
				case "unavailable":
					a.Session.Second.Available = nil
				case "foreign kind":
					a.Session.Second.Current.Kind = 2
				case "unsupported pair":
					a.Session.Second.Current.ID = 30
					a.Session.Second.Available[0].ID = 30
				case "available overflow":
					a.Session.Second.Available = append(a.Session.Second.Available, currentSecondLocation{1, 20})
				case "fog absent":
					a.Fog = nil
				case "fog extent":
					a.Fog.Cols = 257
				case "fog plane":
					a.Fog.Visible = a.Fog.Visible[:len(a.Fog.Visible)-1]
				case "activity duplicate":
					a.Policy.ROM2.Groups = append(a.Policy.ROM2.Groups, a.Policy.ROM2.Groups[len(a.Policy.ROM2.Groups)-1])
				}
			})
			before := secondSaveSampleNow(t, f, app)
			oldTown, oldLive := f.Town, f.live
			if _, _, err := f.RestoreOriginal(changed); err == nil {
				t.Fatal("invalid continuation admitted")
			}
			if oldTown != f.Town || oldLive != f.live {
				t.Fatal("refusal adopted candidate")
			}
			secondAssertSample(t, before, secondSaveSampleNow(t, f, app))
		})
	}
	for _, loss := range []string{"header-current", "truncated"} {
		t.Run(loss, func(t *testing.T) {
			changed := raw[:len(raw)/2]
			if loss == "header-current" {
				changed = secondChangeSave(t, raw, func(a *currentActionData) {
					other := 20
					if mission == 20 {
						other = 10
					}
					a.Session.Second.Current.ID = other
					a.Session.Second.Available[0].ID = other
				})
			}
			before := secondSaveSampleNow(t, f, app)
			oldTown, oldLive := f.Town, f.live
			if _, _, err := f.RestoreOriginal(changed); err == nil {
				t.Fatal("malformed/header mismatch admitted")
			}
			if f.Town != oldTown || f.live != oldLive {
				t.Fatal("refusal adopted candidate")
			}
			secondAssertSample(t, before, secondSaveSampleNow(t, f, app))
		})
	}
	secondMissionPresentationControls(t, f, app, out, raw, mission)
	secondMissionTerminalRefusals(t, f, app, out)
	for _, loss := range []string{"bank", "Forced", "campaign bank"} {
		t.Run(loss, func(t *testing.T) {
			changed := secondChangeSave(t, raw, func(a *currentActionData) {
				switch loss {
				case "bank":
					a.Policy.ROM2.Scenario[1000] ^= 1
				case "Forced":
					if len(a.Policy.ROM2.Groups) == 0 {
						t.Fatal("no activity discriminator")
					}
					a.Policy.ROM2.Groups[0].Forced = !a.Policy.ROM2.Groups[0].Forced
				case "campaign bank":
					a.Session.Second.Bank[991] ^= 1
				}
			})
			path := filepath.Join(out, loss+".sav")
			if err := os.WriteFile(path, changed, 0600); err != nil {
				t.Fatal(err)
			}
			cold, _ := secondMissionColdAt(t, out, filepath.Base(path), mission)
			if loss == "campaign bank" {
				if reflect.DeepEqual(cold.Town.second, f.Town.second) {
					t.Fatal("campaign bank loss invisible")
				}
			} else if cold.live.world.Hash() == f.live.world.Hash() {
				t.Fatal("simulation loss invisible")
			}
		})
	}
	empty := secondChangeSave(t, raw, func(a *currentActionData) { a.Program = &currentScriptProgram{Dialect: sim.ScriptROM2} })
	if err := os.WriteFile(filepath.Join(out, "empty.sav"), empty, 0600); err != nil {
		t.Fatal(err)
	}
	cold, _ := secondMissionColdAt(t, out, "empty.sav", mission)
	if !cold.live.world.Script().Empty() || cold.live.world.Script().Dialect() != sim.ScriptROM2 {
		t.Fatal("explicit empty dialect was reset")
	}
	absent := secondChangeSave(t, raw, func(a *currentActionData) { a.Program = nil })
	if _, _, err := f.RestoreOriginal(absent); err == nil {
		t.Fatal("ROM2 no-program accepted without required dialect")
	}
	firstRoot := os.Getenv("AGAINROM_FIRST_ASSETS")
	if firstRoot == "" {
		t.Fatal("AGAINROM_FIRST_ASSETS is required for ROM2 save loss controls")
	}
	first, err := NewFrontEnd(firstRoot)
	if err != nil {
		t.Fatal(err)
	}
	cleanupFrontAudio(t, first)
	first.SetDeterministicFrames(true)
	first.Options = OptionsStore{}
	a := first.App("first game identity")
	a.Layout(1024, 768)
	if err := a.OpenMission(first.DirectNewGame(10)); err != nil {
		t.Fatal(err)
	}
	firstSnapshot, _, err := first.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	firstRaw, err := first.ExportCurrentSave(firstSnapshot, "first game")
	if err != nil {
		t.Fatal(err)
	}
	beforeFirst := first.live.world.Hash()
	beforeSecond := f.live.world.Hash()
	oldFirst, oldSecond := first.live, f.live
	if _, _, err := first.RestoreOriginal(raw); err == nil {
		t.Fatal("ROM2 SAV entered ROM1 base")
	}
	if _, _, err := f.RestoreOriginal(firstRaw); err == nil {
		t.Fatal("ROM1 SAV entered ROM2 base")
	}
	if first.live != oldFirst || f.live != oldSecond || first.live.world.Hash() != beforeFirst || f.live.world.Hash() != beforeSecond {
		t.Fatal("wrong-game refusal changed active mission")
	}
	historical := secondChangeSave(t, firstRaw, func(a *currentActionData) { a.Session.Game = ""; a.Program.Dialect = sim.ScriptROM1 })
	if _, _, err := first.RestoreOriginal(historical); err != nil {
		t.Fatal("historical absent identity no longer defaults ROM1", err)
	}
	t.Log("independent dialect/bank/Forced/campaign/game controls, atomic cross-game refusal and historical absent-ROM1 marker")
}

func TestReleaseSecondMissionNoticeSaveBoundary(t *testing.T) {
	secondMissionNoticeBoundary(t, 10)
}

func TestReleaseSecondMissionTwentyNoticeSaveBoundary(t *testing.T) {
	secondMissionNoticeBoundary(t, 20)
}

func secondMissionNoticeBoundary(t *testing.T, mission int) {
	secondGameRoot(t)
	out := secondMissionSaveDirectory(t)
	f, app := secondMissionSettledAt(t, mission)
	f.ConfigureSaveSeams(app, SaveStore{Dir: out}, OriginalStore{}, nil)
	if !f.live.openDialogue(1) {
		t.Fatal("installed dialogue unavailable")
	}
	for _, key := range []string{"f2", "f4"} {
		if err := app.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
		if app.Screen() != ui.ScreenMap {
			t.Fatal("open notice reached SAVE")
		}
	}
	save, _, _ := f.SaveSeams(SaveStore{Dir: out}, OriginalStore{}, nil)
	if _, err := save(true); err == nil {
		t.Fatal("compatibility seam discarded active notice")
	}
	dismissSecondMissionDialogue(t, app, f.live)
	f.live.observeScriptMessages([]int32{2, 3})
	secondMissionNamedSave(t, f, app, out, "Deferred notices")
	cold, a := secondMissionColdAt(t, out, "Deferred notices.sav", mission)
	if !reflect.DeepEqual(cold.live.mission.pendingMessages, []int32{2, 3}) {
		t.Fatal("ordered notice queue lost")
	}
	for _, event := range []int{2, 3} {
		f.live.settleSecondGameNotices()
		cold.live.settleSecondGameNotices()
		if !f.live.mission.open || !cold.live.mission.open || !bytes.Equal(f.live.mission.payload, cold.live.mission.payload) {
			t.Fatalf("notice%d was not replayed", event)
		}
		payload := bytes.Clone(f.live.mission.payload)
		for pages := 0; f.live.mission.open && bytes.Equal(f.live.mission.payload, payload); pages++ {
			if pages >= 64 {
				t.Fatal("notice exceeded page bound")
			}
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
			if f.live.mission.open != cold.live.mission.open || !bytes.Equal(f.live.mission.payload, cold.live.mission.payload) {
				t.Fatal("notice replay order differs")
			}
		}
	}
	if len(cold.live.mission.pendingMessages) != 0 {
		t.Fatal("notice replay retained queue")
	}
	t.Log("active notice blocks F2/F4/seam; legally saved pending [2,3] survives cold LOAD and ordered replay")
}
