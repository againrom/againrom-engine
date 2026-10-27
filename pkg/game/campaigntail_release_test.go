package game

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// campaignTailTownApp puts an App on f's open town screen and wires the
// ordinary save seams to a new directory.
func campaignTailTownApp(t *testing.T, f *FrontEnd) (*ui.App, SaveStore) {
	t.Helper()
	f.SetDeterministicFrames(true)
	app := f.App("campaign tail")
	t.Cleanup(app.StopAudio)
	app.Layout(640, 480)
	app.SetTown(f.TownScreen())
	app.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town", Label: "town"}} }, func(string) (ui.MapOpener, bool, error) { return nil, true, nil })
	if err := app.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("town"); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("open the town screen", err, app.Screen())
	}
	store := SaveStore{Dir: t.TempDir()}
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	return app, store
}

// campaignTailWant is the tail a SAV's campaign record must hold.
type campaignTailWant struct {
	main, selected   uint32
	autoGet, last    uint32
	first            bool
	time, counter    uint32
	markers          []uint32
	hired            []int
	mapName, chooser string
}

func campaignTailCheck(t *testing.T, name string, raw []byte, selector int, want campaignTailWant) {
	t.Helper()
	c := autoGetCampaign(t, raw)
	var markers []uint32
	for _, m := range c.Markers {
		markers = append(markers, m.Value)
	}
	var hired []int
	for i, h := range c.MercenaryHired {
		if h {
			hired = append(hired, i+1)
		}
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	label, err := OriginalSaveLabel(raw, selector)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s: main %d selected %d AutoGetMission %#x LastMission %d FirstMapPoint %v time %d counter %d markers %v hired %v map %q chooser %q",
		name, c.Main.Mission, c.SelectedMission, c.AutoGetMission, c.LastMission, c.FirstMapPoint, c.MissionTime, c.ScoreEvents, markers, hired, doc.Head.MapName, label)
	if c.Main.Mission != want.main || c.SelectedMission != want.selected || c.AutoGetMission != want.autoGet ||
		c.LastMission != want.last || c.FirstMapPoint != want.first || c.MissionTime != want.time || c.ScoreEvents != want.counter ||
		!reflect.DeepEqual(markers, want.markers) || !reflect.DeepEqual(hired, want.hired) {
		t.Fatalf("%s: tail differs from %+v", name, want)
	}
	if want.chooser != "" && (label != want.chooser || doc.Head.MapName != want.mapName) {
		t.Fatalf("%s: chooser %q map %q, want %q and %q", name, label, doc.Head.MapName, want.chooser, want.mapName)
	}
}

// The first SAVE of a new campaign, inside mission 10, holds selected 10,
// AutoGetMission 20, LastMission 0, the first-MapPoint word 0 (the party
// stands on mission 10's map object), time 0, no marker and no hire flag
// (SAV-1332). The control is a town file written straight after character
// generation: the engine's first town holds main 30, whose record declares no
// AutoGetMission, and its party stands on MapPoint 0.
func TestReleaseNewCampaignMissionTenTail(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	app := f.App("new campaign tail")
	t.Cleanup(app.StopAudio)
	party := f.ChargenParty(ui.ChargenResult{Name: "Tail Hero", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	if err := app.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("mission SAVE %q: %v", name, err)
	}
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	campaignTailCheck(t, "mission 10", raw, f.textSelector(), campaignTailWant{main: 10, selected: 10, autoGet: 20})

	town := releaseFront(t)
	reachabilityWalkArrive(t, town, "Tail Hero")
	snapshot, label, err := town.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	control, err := town.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	campaignTailCheck(t, "first town", control, town.textSelector(), campaignTailWant{main: 30, selected: 30, autoGet: 0xFFFFFFFF, first: true})
}

// A town SAV writes its markers in the order the missions were registered
// (SAV-1333); a LOAD keeps that order and a later registration is appended.
// The two orders are each other's control.
func TestReleaseTownSAVMarkersKeepAppendOrder(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	for _, order := range [][]int{{60, 50}, {50, 60}} {
		f := kitOwnerWalkToChapter(t, 40)
		screen := f.TownScreen().(*townScreen)
		screen.resetWorldMarkers()
		for _, m := range order {
			screen.markWorldSelected(m)
		}
		raw := currentTownSave(t, f)
		want := []uint32{uint32(order[0]), uint32(order[1])}
		if got := campaignMarkerValues(t, raw); !reflect.DeepEqual(got, want) {
			t.Fatalf("order %v: SAV markers %v, want %v", order, got, want)
		}
		g, _, _ := cityGroupsColdApp(t, raw)
		g.TownScreen().(*townScreen).markWorldSelected(70)
		again := currentTownSave(t, g)
		if got := campaignMarkerValues(t, again); !reflect.DeepEqual(got, append(want, 70)) {
			t.Fatalf("order %v: SAV after LOAD and one more registration holds markers %v", order, got)
		}
	}
}

func campaignMarkerValues(t *testing.T, raw []byte) []uint32 {
	t.Helper()
	var out []uint32
	for _, m := range autoGetCampaign(t, raw).Markers {
		out = append(out, m.Value)
	}
	return out
}

// A hire sets its type's flag in the town SAV and a return clears it
// (SAV-1334, MERC-HIRE-003); the working pool cell keeps its value while the
// type is hired (MERC-POOL-012).
func TestReleaseTownSAVHireFlagSetAndCleared(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := kitOwnerWalkToChapter(t, 120)
	f.Town.gold = 3_000_000
	before := autoGetCampaign(t, currentTownSave(t, f))
	cityGroupsToggle(t, f, 6)
	hired := autoGetCampaign(t, currentTownSave(t, f))
	cityGroupsToggle(t, f, 6)
	returned := autoGetCampaign(t, currentTownSave(t, f))
	if before.MercenaryHired[5] || !hired.MercenaryHired[5] || returned.MercenaryHired[5] {
		t.Fatalf("type 6 hire flag before %v, hired %v, returned %v; want false, true, false",
			before.MercenaryHired[5], hired.MercenaryHired[5], returned.MercenaryHired[5])
	}
	if hired.MercenaryWorking[5] != before.MercenaryWorking[5] || returned.MercenaryWorking[5] != before.MercenaryWorking[5] {
		t.Fatalf("type 6 working pool %d, %d, %d; want it unchanged", before.MercenaryWorking[5], hired.MercenaryWorking[5], returned.MercenaryWorking[5])
	}
}

// A loaded original town keeps its AutoGetMission (SAV-1330) and its
// first-MapPoint word (SAV-1331): the next SAVE writes each byte for byte.
// 0, -1 and a mission number are each other's control.
func TestReleaseLoadedTownWritesItsTailBack(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	_, source := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
	for _, tc := range []struct {
		autoGet uint32
		first   uint32
	}{{0, 1}, {0xFFFFFFFF, 1}, {40, 1}, {0xFFFFFFFF, 0}, {0, 0}} {
		file, err := sav.Open(source)
		if err != nil {
			t.Fatal(err)
		}
		if err := file.SetCampaignScalar(2, tc.autoGet); err != nil {
			t.Fatal(err)
		}
		if err := file.SetCampaignScalar(4, tc.first); err != nil {
			t.Fatal(err)
		}
		f, _, _ := cityGroupsColdApp(t, file.Marshal())
		c := autoGetCampaign(t, currentTownSave(t, f))
		if c.AutoGetMission != tc.autoGet || c.FirstMapPoint != (tc.first != 0) {
			t.Fatalf("loaded AutoGetMission %#x and first-MapPoint %d wrote %#x and %v", tc.autoGet, tc.first, c.AutoGetMission, c.FirstMapPoint)
		}
	}
}

// The owner kit: on each install, the first town SAV of a new campaign, before
// any mission is played (main 30), and a town SAV after main 30, each written by the ordinary F2 SAVE
// with a numbered chooser label. AGAINROM_CAMPAIGN_TAIL_KIT names the output
// directory; the files go to its EN or RU subdirectory.
func TestReleaseCampaignTailKit(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	type kitFile struct {
		file, label string
		build       func(t *testing.T) *FrontEnd
		want        campaignTailWant
	}
	newTown := func(t *testing.T) *FrontEnd {
		f := releaseFront(t)
		reachabilityWalkArrive(t, f, "Kit Hero")
		return f
	}
	after30 := func(t *testing.T) *FrontEnd { return kitOwnerWalkToChapter(t, 40) }
	probe := releaseFront(t)
	edition := kitOwnerEdition(probe)
	first, second := "9771", "9772"
	if edition == "RU" {
		first, second = "9773", "9774"
	}
	files := []kitFile{
		{"game" + first + ".sav", first + " new town tail", newTown,
			campaignTailWant{main: 30, selected: 30, autoGet: 0xFFFFFFFF, first: true}},
		{"game" + second + ".sav", second + " town after 30", after30,
			campaignTailWant{main: 40, selected: 40, autoGet: 0xFFFFFFFF, first: true}},
	}
	out := ""
	if dir := os.Getenv("AGAINROM_CAMPAIGN_TAIL_KIT"); dir != "" {
		out = filepath.Join(dir, edition)
		if err := os.MkdirAll(out, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, k := range files {
		f := k.build(t)
		app, store := campaignTailTownApp(t, f)
		raw := cityRosterF2Save(t, app, store, k.label)
		want := k.want
		want.markers = campaignMarkerValues(t, raw)
		want.chooser = k.label
		if strings.Contains(want.chooser, "Generated") {
			t.Fatal("kit label names a generated mission")
		}
		campaignTailCheck(t, k.file, raw, f.textSelector(), want)
		if out != "" {
			if err := os.WriteFile(filepath.Join(out, k.file), raw, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
}
