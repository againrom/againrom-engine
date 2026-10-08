package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type generatedMissionProof struct {
	Mission     int
	Offset      int
	InputSHA256 [32]byte
	Samples     []map[string]json.RawMessage
	Arrival     map[string]json.RawMessage
}

func generatedMissionSample(t *testing.T, f *FrontEnd) map[string]json.RawMessage {
	t.Helper()
	w := f.live.world
	out := map[string]json.RawMessage{}
	put := func(name string, value any) {
		b, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		out[name] = b
	}
	actors := w.Entities()
	runtimes := map[sim.EntityID]sim.EntityID{}
	for _, e := range actors {
		runtimes[e.ID] = e.ID
	}
	remap := func(id sim.EntityID, structure bool) (sim.EntityID, error) {
		if structure {
			return id, nil
		}
		if runtime, ok := runtimes[id]; ok {
			return runtime, nil
		}
		return id, nil
	}
	put("population", len(actors))
	put("hash/world", w.Hash())
	for _, e := range actors {
		prefix := fmt.Sprintf("actor/%d/", e.ID)
		e.ID = runtimes[e.ID]
		if e.HasAttackTarget {
			e.AttackTarget, _ = remap(e.AttackTarget, e.AttackTargetKind == sim.AttackTargetStructure)
		}
		if e.HasEscortTarget {
			e.EscortTarget, _ = remap(e.EscortTarget, false)
		}
		if e.HasKillCredit {
			e.KillCreditSource, _ = remap(e.KillCreditSource, false)
		}
		v := reflect.ValueOf(e)
		for i := range v.NumField() {
			if v.Field(i).CanInterface() {
				put(prefix+v.Type().Field(i).Name, v.Field(i).Interface())
			}
		}
	}
	actions := w.Actions()
	if err := actions.RemapActors(remap); err != nil {
		t.Fatal(err)
	}
	slices.SortFunc(actions.Actors, func(a, b sim.ActorContinuation) int { return int(a.Entity) - int(b.Entity) })
	slices.SortFunc(actions.EffectCasters, func(a, b sim.AttachedEffectCaster) int {
		if a.Target != b.Target {
			return int(a.Target) - int(b.Target)
		}
		return int(a.Spell) - int(b.Spell)
	})
	put("actions", actions)
	effects := w.ActiveEffects()
	for i := range effects {
		e := &effects[i]
		e.Target, _ = remap(e.Target, false)
		if e.HasCaster {
			e.Caster, _ = remap(e.Caster, false)
		}
	}
	slices.SortFunc(effects, func(a, b sim.ActiveEffect) int {
		if a.Target != b.Target {
			return int(a.Target) - int(b.Target)
		}
		return int(a.Spell) - int(b.Spell)
	})
	put("effects", effects)
	put("items", w.SavedObjects())
	for _, d := range w.SavedDiaries() {
		if !d.Owner.Player {
			d.Owner.Actor = runtimes[d.Owner.Actor]
		}
		put(fmt.Sprintf("diary/%t/%d", d.Owner.Player, d.Owner.Actor), d)
	}
	groups, _, present := w.SavedGroups()
	put("groups/present", present)
	for _, g := range groups {
		if len(g.Members) == 0 {
			g.Members = nil
		}
		for i := range g.Members {
			if g.Members[i].Bound {
				g.Members[i].Entity = runtimes[g.Members[i].Entity]
			}
		}
		put(fmt.Sprintf("group/%d", g.ID), g)
	}
	players, _ := w.SavedGroupPlayers()
	put("players", players)
	formations, _ := w.SavedPlayerFormations()
	put("formations", formations)
	for _, p := range players {
		put(fmt.Sprintf("purse/%d", p.Slot), w.Purse(p.Slot))
	}
	clock, hasClock := w.SessionClock()
	put("clock", clock)
	put("hasClock", hasClock)
	put("tick", w.Tick())
	put("rng", w.RandomState())
	put("registers", w.ScriptRegisters())
	put("outcome", w.Outcome())
	put("relations", w.Relations())
	planes, planesPresent := w.SavedCellPlanes()
	put("planes/present", planesPresent)
	if planesPresent {
		for name, plane := range map[string][65536]byte{"Cost": planes.Cost, "Static": planes.Static, "Dynamic": planes.Dynamic, "Height": planes.Height, "CostKnown": planes.CostKnown} {
			var cells []byte
			for y := 0; y < int(f.live.mission.state.Map.Height); y++ {
				for x := 0; x < int(f.live.mission.state.Map.Width); x++ {
					cells = append(cells, plane[y*256+x])
				}
			}
			put("planes/"+name, cells)
		}
		put("planes/Costs", planes.Costs)
	}
	put("structures", w.Structures())
	put("bodies", currentBodies(w))
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	campaign := s.Campaign
	if !s.CampaignState {
		campaign, err = nativeCampaignProjectionForChapter(f, s, f.generatedCampaignChapter(s))
		if err != nil {
			t.Fatal(err)
		}
		campaign.SelectedMission = uint32(s.Mission)
		// SAV-1094/REG-SCN-063: mirror currentMissionDocument's own
		// AutoGetMission derivation, or this pre-save sample predicts a shape
		// SAVE no longer writes.
		campaign.AutoGetMission = autoGetMissionValue(f.Campaign.Value(), int(s.Mission))
	}
	// SAVE writes the mission shape even when town keeps its home point.
	campaign.FirstMapPoint = false
	put("campaign", campaign)
	put("documents", s.Documents)
	put("fame", s.Fame)
	put("chapter", f.Town.Chapter())
	put("mission", f.liveMission)
	put("offered", f.Offered)
	type member struct {
		ID, Name                  string
		Temporary, Player, Leader bool
		Mercenary                 uint8
		Companion                 int
		Actor                     sim.EntityID
	}
	var roster []member
	for i, p := range f.live.mission.party {
		roster = append(roster, member{p.ID, p.Name, p.Temporary, p.PlayerCharacter, p.StartingHero, p.MercenaryType, p.CompanionNPC, runtimes[f.live.mission.ids[i]]})
		if p.StartingHero {
			picture := f.live.inspectionUnitPicture(uint32(f.live.mission.ids[i]))
			if picture == nil {
				t.Fatal("current hero inspection figure is missing")
			}
			put("hero/picture", sha256.Sum256(picture.Pix))
			put("hero/picture-bounds", picture.Bounds().String())
		}
	}
	put("roster", roster)
	for _, building := range []TownBuilding{TownTavern, TownShop, TownSchool} {
		offers := f.Town.Offers(building)
		if !f.Town.Open() {
			offers = nil
		}
		for i := range offers {
			offers[i].Index = i
		}
		put(fmt.Sprintf("offers/%d", building), offers)
	}
	return out
}

func generatedMissionArrival(t *testing.T, f *FrontEnd) map[string]json.RawMessage {
	t.Helper()
	// Controlled victory prerequisite: dispatch the same completion boundary
	// as the victory notice, without claiming to have fought the whole map.
	m := f.live.mission
	if next, line := f.FinishMissionWithRoster(20, m.party, f.live.world, m.ids, m.state.Start.Roster); next != 0 {
		t.Fatal("mission20 did not reach town", next, line)
	}
	if !f.Town.Open() || f.Town.Chapter() != 30 || f.Offered != 30 {
		t.Fatalf("mission20 arrival open=%t chapter=%d offered=%d", f.Town.Open(), f.Town.Chapter(), f.Offered)
	}
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	p := s.Campaign
	if !s.CampaignState {
		p, err = nativeCampaignProjectionForChapter(f, s, 30)
		if err != nil {
			t.Fatal(err)
		}
	}
	out := map[string]json.RawMessage{"campaign": mustGeneratedJSON(t, p), "documents": mustGeneratedJSON(t, f.Town.Documents()), "gold": mustGeneratedJSON(t, f.Town.Gold())}
	var names []string
	for _, member := range f.Carried {
		names = append(names, member.ID+"/"+member.Name)
	}
	out["roster"] = mustGeneratedJSON(t, names)
	for _, building := range []TownBuilding{TownTavern, TownShop, TownSchool} {
		out[fmt.Sprint(building)] = mustGeneratedJSON(t, f.Town.Offers(building))
	}
	return out
}

func mustGeneratedJSON(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func requireGeneratedSample(t *testing.T, got, want map[string]json.RawMessage, step int) {
	t.Helper()
	keys := make([]string, 0, len(want))
	for key := range want {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	failures := 0
	for _, key := range keys {
		if !bytes.Equal(got[key], want[key]) {
			if len(want[key]) > 1200 || len(got[key]) > 1200 {
				t.Errorf("sample %d %s differs: sizes %d/%d hashes %x/%x", step, key, len(want[key]), len(got[key]), sha256.Sum256(want[key]), sha256.Sum256(got[key]))
			} else {
				t.Errorf("sample %d %s\nwant %s\ngot  %s", step, key, want[key], got[key])
			}
			failures++
			if failures == 8 {
				t.FailNow()
			}
		}
	}
	if failures > 0 {
		t.FailNow()
	}
	if len(got) != len(want) {
		t.Fatalf("sample %d population changed: fields %d/%d", step, len(got), len(want))
	}
}

func generatedMissionSave(t *testing.T, f *FrontEnd, app *ui.App) string {
	t.Helper()
	return generatedMissionSaveIn(t, f, app, t.TempDir())
}

// generatedMissionSaveIn saves into a directory that outlives the caller.
func generatedMissionSaveIn(t *testing.T, f *FrontEnd, app *ui.App, dir string) string {
	t.Helper()
	store := SaveStore{Dir: dir}
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	openMissionGameMenu(t, app)
	before := f.live.world.Hash()
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenSave {
		t.Fatal("ordinary SAVE did not open its dialog")
	}
	if err := app.HeadlessSaveEdit(store.Dir, "Generated mission", ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if f.live.world.Hash() != before {
		t.Fatal("ordinary SAVE advanced or changed the World")
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 || !IsOriginal(entries[0].Name) {
		t.Fatalf("ordinary SAVE: %v %v", entries, err)
	}
	raw, err := store.Read(entries[0].Name)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil || doc.World == nil || int(doc.Head.Mission) != f.liveMission {
		s, _, snapshotErr := f.Snapshot(true)
		_, producerErr := currentWorldDocument(s, false, f.Table)
		t.Fatalf("mission replaced by another save point: decode=%v snapshot=%v producer=%v", err, snapshotErr, producerErr)
	}
	if f.liveMission >= 30 && len(doc.Campaign.Documents) < 3 {
		t.Fatalf("mission %d SAV carries %d campaign documents, want at least 3", f.liveMission, len(doc.Campaign.Documents))
	}
	if doc.Campaign.Scalars[4] != 0 {
		t.Fatalf("mission SAV has town home-map-point scalar %d, want 0", doc.Campaign.Scalars[4])
	}
	if want := uint32(len(doc.Players) + 1); doc.Head.PlayerListField != want {
		t.Fatalf("mission SAV player-list dword=%d, want %d for %d players", doc.Head.PlayerListField, want, len(doc.Players))
	}
	actions, err := readCurrentActions(&doc)
	if err != nil || actions == nil {
		t.Fatal(err)
	}
	actorObjects := map[sim.EntityID]uint16{}
	for _, binding := range actions.Bindings {
		if !binding.Structure && !binding.Missing {
			actorObjects[binding.ID] = binding.Object
		}
	}
	for _, e := range f.live.world.Entities() {
		index := actorObjects[e.ID]
		if index == 0 || int(index) > len(doc.Objects) {
			t.Fatalf("saved actor %d lost its wire type", e.ID)
		}
		typeID, err := savedStructureValue(&doc.Objects[index-1], "T0E")
		if err != nil || typeID != uint32(uint16(e.TypeID)) {
			t.Fatal("drawable class replaced the SAV wire TypeID", e.ID, typeID, e.TypeID, err)
		}
	}
	switch app.Screen() {
	case ui.ScreenGameMenu:
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		if app.Screen() != ui.ScreenMap {
			t.Fatalf("generated SAV post-save Escape reached %s, want map", app.Screen())
		}
	case ui.ScreenMap:
		// Already at the map; do not reopen the menu.
	default:
		t.Fatalf("generated SAV write left screen %s, want game menu or map", app.Screen())
	}
	return filepath.Join(store.Dir, entries[0].Name)
}

func generatedMissionCold(t *testing.T, path string) {
	t.Helper()
	if second := generatedMissionColdPass(t, path, t.TempDir()); second != "" {
		runSpellWitnessChild(t, second, "AGAINROM_GENERATED_MISSION_INPUT")
	}
}

// generatedMissionColdPass loads one frozen SAV, compares its samples and, for
// a first SAV, writes the second SAV into dir and returns its path.
func generatedMissionColdPass(t *testing.T, path, dir string) (secondPath string) {
	t.Helper()
	b, err := os.ReadFile(path + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var proof generatedMissionProof
	if err = json.Unmarshal(b, &proof); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || sha256.Sum256(raw) != proof.InputSHA256 {
		t.Fatal("changed frozen SAV input", err)
	}
	f, app := loadSAVWindow(t, SaveStore{Dir: filepath.Dir(path)}, filepath.Base(path))
	for step, want := range proof.Samples {
		requireGeneratedSample(t, generatedMissionSample(t, f), want, proof.Offset+step)
		if step == 8 && proof.Offset == 0 {
			if dir == "" {
				t.Fatal("a second SAV was due but this stage has no directory for it")
			}
			second := generatedMissionSaveIn(t, f, app, dir)
			if proof.Mission == 111 {
				probe, _ := loadSAVWindow(t, SaveStore{Dir: filepath.Dir(second)}, filepath.Base(second))
				if f.live.world.Hash() != probe.live.world.Hash() {
					currentMenuWorldDiagnostics(t, f.live.world, probe.live.world)
				}
			}
			next := proof
			next.Offset, next.Samples = step, proof.Samples[step:]
			nextRaw, err := os.ReadFile(second)
			if err != nil || bytes.Equal(raw, nextRaw) {
				t.Fatal("second SAV did not change", err)
			}
			next.InputSHA256 = sha256.Sum256(nextRaw)
			data := mustGeneratedJSON(t, next)
			if err = os.WriteFile(second+".json", data, 0600); err != nil {
				t.Fatal(err)
			}
			emitSpellWitness(t, second, fmt.Sprintf("generated-m%d-second", proof.Mission), data)
			secondPath = second
		}
		if step+1 < len(proof.Samples) {
			f.live.tick()
		}
	}
	if proof.Arrival != nil {
		requireGeneratedSample(t, generatedMissionArrival(t, f), proof.Arrival, 17)
	}
	return secondPath
}

// A part runs in three processes: the writer saves a first SAV per mission, a
// fresh process loads them all and writes second SAVs, another loads those.
const generatedBatchVariable = "AGAINROM_GENERATED_MISSION_BATCH"

// generatedMissionBatch is one stage's SAVs; Next and Dir are empty for the last.
type generatedMissionBatch struct {
	Missions []int
	Paths    []string
	Next     string
	Dir      string
}

// missionDir gives each mission its own directory; SAVE expects one file there.
func missionDir(t *testing.T, dir string, mission int) string {
	t.Helper()
	dir = filepath.Join(dir, fmt.Sprint(mission))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeGeneratedBatch(t *testing.T, path string, batch generatedMissionBatch) {
	t.Helper()
	if err := os.WriteFile(path, mustGeneratedJSON(t, batch), 0600); err != nil {
		t.Fatal(err)
	}
}

func generatedMissionColdBatch(t *testing.T, batchPath string) {
	t.Helper()
	raw, err := os.ReadFile(batchPath)
	if err != nil {
		t.Fatal(err)
	}
	var batch generatedMissionBatch
	if err = json.Unmarshal(raw, &batch); err != nil || len(batch.Missions) != len(batch.Paths) || len(batch.Paths) == 0 {
		t.Fatal("generated mission batch is not a list of missions and SAVs", err)
	}
	var next generatedMissionBatch
	for i, mission := range batch.Missions {
		t.Run(fmt.Sprint(mission), func(t *testing.T) {
			var dir string
			if batch.Dir != "" {
				dir = missionDir(t, batch.Dir, mission)
			}
			if second := generatedMissionColdPass(t, batch.Paths[i], dir); second != "" {
				next.Missions = append(next.Missions, mission)
				next.Paths = append(next.Paths, second)
			}
		})
	}
	if !t.Failed() && batch.Next != "" {
		writeGeneratedBatch(t, batch.Next, next)
	}
}

// The 28 missions run as four release tests, so the release gate can give
// each its own process: mission i of the sorted union belongs to part i%4.
func TestReleaseGeneratedCampaignMissionsSAV(t *testing.T)      { generatedCampaignMissionsSAV(t, 0) }
func TestReleaseGeneratedCampaignMissionsSAVPart1(t *testing.T) { generatedCampaignMissionsSAV(t, 1) }
func TestReleaseGeneratedCampaignMissionsSAVPart2(t *testing.T) { generatedCampaignMissionsSAV(t, 2) }
func TestReleaseGeneratedCampaignMissionsSAVPart3(t *testing.T) { generatedCampaignMissionsSAV(t, 3) }

func generatedCampaignMissionsSAV(t *testing.T, part int) {
	if list := os.Getenv(generatedBatchVariable); list != "" {
		generatedMissionColdBatch(t, list)
		return
	}
	f := releaseFront(t)
	c := f.Campaign.Value()
	missions := append(append(slices.Clone(c.Main), c.Side...), c.Offered...)
	slices.Sort(missions)
	missions = slices.Compact(missions)
	if len(missions) != 28 {
		t.Fatalf("campaign mission union=%v", missions)
	}
	dir := t.TempDir()
	var first generatedMissionBatch
	for i, mission := range missions {
		if i%4 != part {
			continue
		}
		t.Run(fmt.Sprint(mission), func(t *testing.T) {
			first.Missions = append(first.Missions, mission)
			first.Paths = append(first.Paths, generatedMissionFirst(t, mission, nil, missionDir(t, dir, mission)))
		})
	}
	if len(first.Paths) != 7 {
		t.Fatalf("part %d wrote %d first SAVs, want 7", part, len(first.Paths))
	}
	firstList := filepath.Join(dir, "first.json")
	secondList := filepath.Join(dir, "second.json")
	first.Next, first.Dir = secondList, filepath.Join(dir, "second")
	writeGeneratedBatch(t, firstList, first)
	runSpellWitnessChild(t, firstList, generatedBatchVariable)
	raw, err := os.ReadFile(secondList)
	if err != nil {
		t.Fatal(err)
	}
	var second generatedMissionBatch
	if err = json.Unmarshal(raw, &second); err != nil || !slices.Equal(second.Missions, first.Missions) {
		t.Fatalf("second SAV batch %v does not cover missions %v: %v", second.Missions, first.Missions, err)
	}
	runSpellWitnessChild(t, secondList, generatedBatchVariable)
}

// generatedMissionRoute enters a generated mission, applies mutate to the
// live world when it is given, then runs ordinary SAVE, two cold LOADs and the
// continuation comparison.
// generatedMissionStart enters a generated mission and runs 17 ticks of one
// move order.
func generatedMissionStart(t *testing.T, mission int) (*FrontEnd, *ui.App) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	if mission < 30 {
		f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Generated mission", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
		if mission == 20 {
			f.Town.Won(10)
		}
	} else {
		prepareAcceptedCampaignMission(t, f, mission)
	}
	app := f.App("generated campaign mission")
	if err := app.OpenMission(f.MissionOpenerWith(mission, f.Carried)); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	mode, known := f.live.world.CommandFormationMode(sim.SelfSlot)
	if !known || mode != 2 {
		t.Fatal("binding installation reset the current formation", mode, known)
	}
	for _, e := range f.live.world.Entities() {
		if e.Owner == sim.SelfSlot && e.Humanoid && e.TypeID == 33 && !data.ComposesFigure(e.Class) {
			t.Fatal("valid hero wire type replaced the current drawable class")
		}
	}
	for _, e := range f.live.world.Entities() {
		if e.Owner == sim.SelfSlot {
			f.live.enqueue(uint32(e.ID), int(e.X+3), int(e.Y+2))
			break
		}
	}
	for range 17 {
		f.live.tick()
	}
	return f, app
}

func generatedMissionRoute(t *testing.T, mission int, mutate func(*testing.T, *FrontEnd)) {
	path := generatedMissionFirst(t, mission, mutate, t.TempDir())
	runSpellWitnessChild(t, path, "AGAINROM_GENERATED_MISSION_INPUT")
	t.Logf("mission %d: changed ordinary SAVE, two independent cold LOADs, 16 continuation ticks", mission)
}

func generatedMissionFirst(t *testing.T, mission int, mutate func(*testing.T, *FrontEnd), dir string) string {
	t.Helper()
	f, app := generatedMissionStart(t, mission)
	if mutate != nil {
		mutate(t, f)
	}
	proof := generatedMissionProof{Mission: mission}
	proof.Samples = append(proof.Samples, generatedMissionSample(t, f))
	path := generatedMissionSaveIn(t, f, app, dir)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	proof.InputSHA256 = sha256.Sum256(raw)
	for range 16 {
		f.live.tick()
		proof.Samples = append(proof.Samples, generatedMissionSample(t, f))
	}
	if mission == 20 {
		proof.Arrival = generatedMissionArrival(t, f)
	}
	data := mustGeneratedJSON(t, proof)
	if err = os.WriteFile(path+".json", data, 0600); err != nil {
		t.Fatal(err)
	}
	emitSpellWitness(t, path, fmt.Sprintf("generated-m%d-first", mission), data)
	return path
}
