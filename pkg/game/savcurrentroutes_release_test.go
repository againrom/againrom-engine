package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseCapturedFieldsThroughProductionSaveRoutes(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	dir := t.TempDir()
	if output := os.Getenv("AGAINROM_CURRENT_ROUTES_OUT"); output != "" {
		dir = filepath.Join(output, filepath.Base(f.Archives.Root))
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	store := SaveStore{Dir: filepath.Join(dir, "saves")}
	f.Options = OptionsStore{Path: filepath.Join(dir, "options.txt")}
	app := f.App("current field routes")
	app.Layout(1024, 768)
	now := time.Unix(100, 0)
	var captures []writerSourceReference
	f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return now }, func(route string, captured Snapshot) {
		name := fmt.Sprintf("source-%02d-%s.json", len(captures)+1, route)
		captures = append(captures, writeWriterSourceCapture(t, dir, name, observeWriterSources(f, captured, route)))
	})
	defer func() {
		raw, err := json.MarshalIndent(struct {
			Revision, KnowledgePin string
			Captures               []writerSourceReference
		}{os.Getenv("AGAINROM_WITNESS_REVISION"), os.Getenv("AGAINROM_WITNESS_KNOWLEDGE"), captures}, "", "  ")
		if err != nil {
			t.Error(err)
			return
		}
		if err := os.WriteFile(filepath.Join(dir, "source-captures.json"), raw, 0600); err != nil {
			t.Error(err)
		}
	}()
	party := skillCaseParty(f, skillCases()[0])
	initialBody := uint16(party[0].Hero.Body)
	derived, _, _ := mapload.PartySpawnWithTable(party[0], f.Table)
	if !party[0].StartingHero || derived.Body != int32(initialBody) {
		t.Fatal("route fixture needs one starting hero without a Body modifier")
	}
	if err := app.OpenMission(f.MissionOpenerWith(90, party)); err != nil {
		t.Fatal(err)
	}
	app.FlushBackground()
	entry := missionAutosaveRow(t, f, store, 90)
	entryRaw, err := store.Read(entry.Name)
	if err != nil {
		t.Fatal(err)
	}
	write := func(name string, raw []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name+".sav"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("entry", entryRaw)
	routePanelBytes(t, entryRaw, true, true)
	initialID := f.live.mission.ids[0]
	// Reopen an ordinary retained Document before changing captured fields.
	open, _, err := f.RestoreOriginal(entryRaw)
	if err != nil || open == nil {
		t.Fatal(err)
	}
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	initial := routeBoundHero(t, f, entryRaw)
	if f.live.mission.ids[0] != initialID || initial.Stats[0] != initialBody {
		t.Fatalf("initial entry differs from captured hero%d: Body%d, current%d", initialID, initial.Stats[0], initialBody)
	}
	baseBody := initialBody
	f.live.mission.party[0].Hero.Body += 3
	f.live.mission.party[0].Name = "Captured current hero"
	f.live.recomputeRaisedSkills()
	f.Difficulty = mapload.DifficultyHard
	f.quickSpells = [4]uint32{1, 16, 89, 0}
	f.fame = SnapshotFame{Known: true, Time: 0x1020304, Events: 0x5060708}
	if view := f.live.view.SaveApplication(); !view.DollOpen || !view.WornOpen {
		t.Fatal("route fixture must start with visible panels")
	}
	if err := app.HeadlessKey("doll"); err != nil {
		t.Fatal(err)
	}
	panels := f.live.view.SaveApplication()
	panels.WornOpen = false
	if err := f.live.view.RestoreSaveApplication(panels); err != nil {
		t.Fatal(err)
	}
	assert := func(route string, raw []byte) {
		t.Helper()
		file, err := sav.Open(raw)
		if err != nil {
			t.Fatal(err)
		}
		hero := routeBoundHero(t, f, raw)
		if file.Head.Difficulty != 3 || hero.Stats[0] != baseBody+3 {
			if os.Getenv("AGAINROM_CURRENT_ROUTES_DIAGNOSTIC") == "dump" && file.Head.Difficulty == 3 {
				t.Errorf("%s stale current Difficulty/Body %d/%d, expected 3/%d", route, file.Head.Difficulty, hero.Stats[0], baseBody+3)
			} else {
				t.Fatalf("%s stale current Difficulty/Body %d/%d, expected 3/%d", route, file.Head.Difficulty, hero.Stats[0], baseBody+3)
			}
		}
		characters, err := file.Party()
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, character := range characters {
			if character.Hero {
				found = character.Name == "Captured current hero"
			}
		}
		if !found {
			t.Fatalf("%s stale captured name", route)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		if file.World != nil {
			shown := route == "town-next-entry"
			routePanelBytes(t, raw, shown, shown)
		}
		for _, record := range doc.State.ValueRecords {
			if record.Path == "/Character/Name" && string(record.Value.Bytes) != "Captured current hero\x00" {
				t.Fatalf("%s stale current application name: %q", route, record.Value.Bytes)
			}
		}
		cityQuickRaw1167(t, raw, [4]int32{0, 7, -1, -1})
		assertFameOriginalCounters(t, raw, 0x1020304, 0x5060708)
		if bytes.Equal(entryRaw, raw) {
			t.Fatalf("%s copied entry bytes", route)
		}
		write(route, raw)
	}
	manual := cityRosterF2Save(t, app, store, "manual")
	assert("f2", manual)
	if err := app.HeadlessGameMenuAction("return"); err != nil {
		t.Fatal(err)
	}
	for attempts := 0; app.HeadlessNoticeOpen() && attempts < 32; attempts++ {
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessKey("f4"); err != nil {
		t.Fatal(err)
	}
	app.FlushBackground()
	quick, err := store.Read("quick-save-1.sav")
	if err != nil {
		t.Fatal("F4 output", app.Screen(), app.HeadlessNoticeOpen(), err)
	}
	assert("f4", quick)
	now = now.Add(6 * time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	app.FlushBackground()
	timed, err := store.Read("timed-autosave-1.sav")
	if err != nil {
		t.Fatal(err)
	}
	assert("timed", timed)
	if err := app.HeadlessKey("f9"); err != nil {
		t.Fatal(err)
	}
	if f.live.mission.party[0].Name != "Captured current hero" || f.quickSpells != ([4]uint32{1, 16, 89, 0}) {
		t.Fatal("F9 did not restore captured fields")
	}
	f.LiveAdvance(2)
	if panels := f.live.view.SaveApplication(); panels.DollOpen || panels.WornOpen {
		t.Fatalf("F4/F9 and next ticks lost hidden panels: %+v", panels)
	}
	if got := savHero(t, skillExport(t, f, true, "mission SAVE")).stats[0]; got != baseBody+3 {
		t.Fatal("F9 next ticks changed Body", got)
	}
	ms := f.live.mission
	if refusal := f.CampaignSession.carryMissionHome(f.townInstall(), 90, ms.party, f.live.world, ms.ids, nil); refusal != "" {
		t.Fatal(refusal)
	}
	townSeed := skillExport(t, f, false, "town SAVE")
	if err := os.WriteFile(filepath.Join(store.Dir, "town-seed.sav"), townSeed, 0600); err != nil {
		t.Fatal(err)
	}
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("@first"); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("town LOAD", err, app.Screen())
	}
	f.Carried[0].Name = "Captured current hero"
	f.Difficulty = mapload.DifficultyHard
	f.quickSpells = [4]uint32{1, 16, 89, 0}
	f.fame = SnapshotFame{Known: true, Time: 0x1020304, Events: 0x5060708}
	assert("town-f2", cityRosterF2Save(t, app, store, "town-current"))
	if err := app.OpenMission(f.MissionOpener(90)); err != nil {
		t.Fatal(err)
	}
	app.FlushBackground()
	entryNext := missionAutosaveRow(t, f, store, 90)
	entryNextRaw, err := store.Read(entryNext.Name)
	if err != nil {
		t.Fatal(err)
	}
	assert("town-next-entry", entryNextRaw)
	counts := map[string]int{}
	for _, capture := range captures {
		counts[capture.Route]++
	}
	if counts["entry"] != 2 || counts["f2"] != 2 || counts["f4"] != 1 || counts["timed"] != 1 {
		t.Fatal("actual pre-producer capture boundary population", counts)
	}
	t.Log("actual entry observer, F2, F4, timed Poll and town F2 wrote current captured fields; F9 and next ticks retained them")
}

func routePanelBytes(t *testing.T, raw []byte, doll, worn bool) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	data, present, err := sav.NativeActions(doc.State)
	if err != nil || !present {
		t.Fatal("panel state supplement", present, err)
	}
	var fields struct {
		Session struct {
			View struct {
				Panels *struct{ DollOpen, WornOpen bool }
			}
		}
	}
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	panels := fields.Session.View.Panels
	if panels == nil || panels.DollOpen != doll || panels.WornOpen != worn {
		t.Fatalf("raw panel state: %+v, expected %t/%t", panels, doll, worn)
	}
}

func routeBoundHero(t *testing.T, f *FrontEnd, raw []byte) sav.Character {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	party, err := file.Party()
	if err != nil {
		t.Fatal(err)
	}
	var hero sav.Character
	n := 0
	for _, member := range party {
		if member.Hero {
			hero, n = member, n+1
		}
	}
	if n != 1 {
		t.Fatal("route requires exactly one starting hero", n)
	}
	if file.World == nil {
		return hero
	}
	s, _, err := f.Snapshot(true)
	if err != nil || len(s.CurrentPartyIDs) != len(s.Party) || len(s.Party) == 0 || !s.Party[0].StartingHero {
		t.Fatal("captured route party lacks exact binding", err)
	}
	if s.SavedDocument == nil {
		var world sim.World
		if err := world.UnmarshalBinary(s.World); err != nil {
			t.Fatal(err)
		}
		s.SavedDocument, err = f.materializeCurrentWorld(s, &world)
		if err != nil {
			t.Fatal(err)
		}
	}
	n = 0
	for _, binding := range s.SavedDocument.Actors {
		if binding.EntityID == s.CurrentPartyIDs[0] && !binding.Retired {
			key, _ := savedStructureValue(&s.SavedDocument.Document.Objects[binding.ObjectIndex-1], "Identity")
			if key != hero.Key {
				t.Fatalf("captured hero%d object%d key%d != raw starting hero%d", binding.EntityID, binding.ObjectIndex, key, hero.Key)
			}
			n++
		}
	}
	if n != 1 {
		t.Fatal("captured starting hero binding count", n)
	}
	return hero
}

func TestReleaseSaveRoutesWriteIdenticalStateExceptName(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	dir := t.TempDir()
	if output := os.Getenv("AGAINROM_CURRENT_ROUTES_OUT"); output != "" {
		if !filepath.IsAbs(output) {
			t.Fatal("route equivalence output must be absolute")
		}
		dir = filepath.Join(output, filepath.Base(f.Archives.Root), "same-state")
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	store := SaveStore{Dir: filepath.Join(dir, "saves")}
	f.Options = OptionsStore{Path: filepath.Join(dir, "options.txt")}
	app := f.App("same state save routes")
	app.Layout(1024, 768)
	defer app.FlushBackground()
	now := time.Unix(100, 0)
	var baseline Snapshot
	var captures []writerSourceReference
	counts, sameState := map[string]int{}, map[string]bool{}
	f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return now }, func(route string, captured Snapshot) {
		if route == "entry" {
			return
		}
		counts[route]++
		if route != "f2" && route != "f4" && route != "timed" {
			t.Errorf("unexpected producer capture %q", route)
			return
		}
		if counts[route] != 1 {
			t.Errorf("%s captured %d times", route, counts[route])
			return
		}
		sameState[route] = reflect.DeepEqual(baseline, captured)
		captures = append(captures, writeWriterSourceCapture(t, dir, "source-"+route+".json", observeWriterSources(f, captured, route)))
		if !sameState[route] {
			t.Errorf("%s changed the complete captured Snapshot before its SAV producer; World equal=%t", route, bytes.Equal(baseline.World, captured.World))
		}
	})
	if err := app.OpenMission(f.MissionOpenerWith(90, skillCaseParty(f, skillCases()[0]))); err != nil {
		t.Fatal(err)
	}
	app.FlushBackground()
	for attempts := 0; app.HeadlessNoticeOpen() && attempts < 32; attempts++ {
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenMap || app.HeadlessNoticeOpen() {
		t.Fatal("same-state mission is not ready", app.Screen())
	}
	f.quickSpells = [4]uint32{1, 16, 89, 0}
	f.fame = SnapshotFame{Known: true, Time: 0x1020304, Events: 0x5060708}
	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessPointer("hover", 512, 384); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	var err error
	baseline, _, err = f.Snapshot(true)
	if err != nil || len(baseline.World) == 0 || !f.live.view.SaveApplication().PlayerPaused {
		t.Fatal("same-state baseline needs the actual paused mission", err)
	}
	captures = append(captures, writeWriterSourceCapture(t, dir, "source-baseline.json", observeWriterSources(f, baseline, "baseline")))

	manual := cityRosterF2Save(t, app, store, "same-state")
	if err := app.HeadlessGameMenuAction("return"); err != nil {
		t.Fatal(err)
	}
	for attempts := 0; app.HeadlessNoticeOpen() && attempts < 32; attempts++ {
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenMap || app.HeadlessNoticeOpen() {
		t.Fatal("F2 did not return to the paused mission", app.Screen())
	}
	if err := app.HeadlessPointer("hover", 512, 384); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("f4"); err != nil {
		t.Fatal(err)
	}
	app.FlushBackground()
	quick, err := store.Read("quick-save-1.sav")
	if err != nil {
		t.Fatal("actual F4 output", err)
	}
	now = now.Add(6 * time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	app.FlushBackground()
	timed, err := store.Read("timed-autosave-1.sav")
	if err != nil {
		t.Fatal("actual timer Poll output", err)
	}
	for _, route := range []string{"f2", "f4", "timed"} {
		if counts[route] != 1 {
			t.Errorf("%s actual producer capture count=%d, expected 1", route, counts[route])
		}
	}

	nameStart := func(raw []byte, name string) int {
		t.Helper()
		if len(raw) < 16 || string(raw[:4]) != "Asg&" {
			t.Fatal("actual route output lacks the raw SAV container header")
		}
		start := uint64(binary.LittleEndian.Uint32(raw[4:8]))
		blobBytes := uint64(binary.LittleEndian.Uint32(raw[12:16]))
		if start < 20 || start+256 > uint64(len(raw)) || blobBytes != start-16 {
			t.Fatal("actual route output has invalid raw NAME bounds", start, blobBytes, len(raw))
		}
		at := int(start)
		if len(name) >= 256 || !bytes.Equal(raw[at:at+len(name)], []byte(name)) || raw[at+len(name)] != 0 {
			t.Fatalf("actual route NAME differs from %q", name)
		}
		return at
	}
	outputs := []struct {
		Route, File, Name string
		Raw               []byte
	}{
		{"f2", "same-state.sav", "same-state", manual},
		{"f4", "quick-save-1.sav", "quicksave 1", quick},
		{"timed", "timed-autosave-1.sav", "timed autosave 1", timed},
	}
	type comparison struct {
		Route, File                string
		NameStart, NameEnd         int
		SameState, EqualExceptName bool
		FirstDifference            int
	}
	var comparisons []comparison
	baseStart := nameStart(manual, outputs[0].Name)
	for _, output := range outputs {
		start := nameStart(output.Raw, output.Name)
		width := max(len(outputs[0].Name), len(output.Name)) + 1
		first := -1
		for at := 0; at < min(len(manual), len(output.Raw)); at++ {
			if start == baseStart && at >= start && at < start+width {
				continue
			}
			if manual[at] != output.Raw[at] {
				first = at
				break
			}
		}
		if first < 0 && len(manual) != len(output.Raw) {
			first = min(len(manual), len(output.Raw))
		}
		equal := first < 0 && start == baseStart
		comparisons = append(comparisons, comparison{output.Route, output.File, start, start + width, sameState[output.Route], equal, first})
		if !equal {
			t.Errorf("F2/%s SAV differs outside the raw NAME text plus NUL: offset=%d, lengths=%d/%d, NAME starts=%d/%d", output.Route, first, len(manual), len(output.Raw), baseStart, start)
		}
	}
	result, err := json.MarshalIndent(struct {
		Revision, KnowledgePin string
		Captures               []writerSourceReference
		Comparisons            []comparison
	}{os.Getenv("AGAINROM_WITNESS_REVISION"), os.Getenv("AGAINROM_WITNESS_KNOWLEDGE"), captures, comparisons}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "equivalence.json"), result, 0600); err != nil {
		t.Fatal(err)
	}
}
