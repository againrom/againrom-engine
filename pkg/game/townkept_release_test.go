package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// townKeptFields compares the carried actor fields of two records; the Mover
// turn rate byte 10 is written from the Human's current basis and is compared
// apart.
func townKeptFields(a, b sav.TownActorState) map[string]bool {
	mover := func(m []byte) []byte {
		out := slices.Clone(m)
		if len(out) > 10 {
			out[10] = 0
		}
		return out
	}
	return map[string]bool{
		"U50": a.State == b.State, "U136": a.Done == b.Done, "U138": a.Deadline == b.Deadline, "U6C": a.Timer == b.Timer,
		"U154": slices.Equal(mover(a.Mover), mover(b.Mover)), "U15C": slices.Equal(a.StaticRoute, b.StaticRoute),
		"U178": slices.Equal(a.DynamicRoute, b.DynamicRoute), "U158": slices.Equal(a.Order, b.Order), "U158_90": slices.Equal(a.OrderRing, b.OrderRing),
	}
}

// townKeptDocuments is every town document of the install root and the owner
// corpus: no World section.
func townKeptDocuments(t *testing.T) []string {
	t.Helper()
	var paths []string
	if root := os.Getenv("AGAINROM_ASSETS"); root != "" {
		files, _ := filepath.Glob(filepath.Join(root, "*.sav"))
		paths = append(paths, files...)
	}
	if corpus := os.Getenv("AGAINROM_SAVE_CORPUS"); corpus != "" {
		files, _ := filepath.Glob(filepath.Join(corpus, "*", "*.sav"))
		paths = append(paths, files...)
	}
	var towns []string
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err == nil && doc.World == nil {
			towns = append(towns, path)
		}
	}
	sort.Strings(towns)
	return towns
}

// A town LOAD then SAVE keeps each carried actor field of every persistent
// member: the LOAD fills the carrier from the record, the SAVE writes it.
func TestReleaseTownLoadSaveKeepsCarriedActorFields(t *testing.T) {
	f := releaseFront(t)
	paths := townKeptDocuments(t)
	if len(paths) == 0 {
		t.Skip("no town document in the install root or the owner corpus")
	}
	same, total := map[string]int{}, map[string]int{}
	rate := 0
	loaded := 0
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		fresh := &FrontEnd{InstallResources: f.InstallResources}
		if _, town, err := fresh.RestoreOriginal(raw); err != nil || !town {
			t.Logf("%s: town LOAD refused (town=%t): %v", path, town, err)
			continue
		}
		snapshot, _, err := fresh.Snapshot(false)
		if err != nil {
			t.Fatalf("%s: snapshot: %v", path, err)
		}
		out, err := fresh.ExportOriginalSave(snapshot, "kept")
		if err != nil {
			t.Fatalf("%s: town SAVE: %v", path, err)
		}
		source, err := sav.Open(raw)
		if err != nil {
			t.Fatal(err)
		}
		written, err := sav.Open(out)
		if err != nil {
			t.Fatal(err)
		}
		before, err := originalCityCharacters(source, f.Table)
		if err != nil {
			t.Fatalf("%s: source party: %v", path, err)
		}
		after, err := originalCityCharacters(written, f.Table)
		if err != nil {
			t.Fatalf("%s: written party: %v", path, err)
		}
		if len(before) != len(after) {
			t.Fatalf("%s: %d persistent members loaded, %d written", path, len(before), len(after))
		}
		loaded++
		for i := range before {
			for name, equal := range townKeptFields(before[i].Town, after[i].Town) {
				total[name]++
				if equal {
					same[name]++
				} else {
					t.Errorf("%s: member %d field %s changed: %+v -> %+v", path, i, name, before[i].Town, after[i].Town)
				}
			}
			if len(before[i].Town.Mover) > 10 && len(after[i].Town.Mover) > 10 && before[i].Town.Mover[10] == after[i].Town.Mover[10] {
				rate++
			}
		}
	}
	if loaded == 0 {
		t.Fatal("no town document loaded")
	}
	names := make([]string, 0, len(total))
	for name := range total {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		t.Logf("field %s: %d of %d kept", name, same[name], total[name])
	}
	t.Logf("%d town documents of %d loaded; Mover byte 10 kept on %d of %d members", loaded, len(paths), rate, total["U154"])
}

// A town SAVE right after a mission ends writes the mission's run end and
// the cull's two stores for each kept member, and the constructor values for
// a member hired in the town.
func TestReleaseTownSAVAfterMissionWritesKeptAndHiredFields(t *testing.T) {
	f := loadedMission20(t, nil)
	// A move gives the hero a run end the mission sets, so the written U138
	// is the mission's value and not a constructor 0.
	hero := equipmentReturnHero(t, f)
	start := f.live.world.Tick()
	h, _ := f.live.entity(hero)
	f.live.pending = append(f.live.pending, sim.MoveTo(hero, sim.CellPoint{X: h.X + 2, Y: h.Y}))
	for f.live.world.Tick() < start+100 {
		f.live.tick()
	}
	if h, _ = f.live.entity(hero); currentActionRunEnd(h) == 0 {
		t.Fatalf("the move left no run end: %+v", h.ActionClock)
	}
	ends := map[string]uint32{}
	for i, p := range f.live.mission.party {
		if e, ok := f.live.entity(f.live.mission.ids[i]); ok && e.Alive() {
			ends[p.ID] = currentActionRunEnd(e)
		}
	}
	if _, _, err := f.LiveCompleteCampaign(); err != nil {
		t.Fatalf("finish mission 20: %v", err)
	}
	hireInTown(t, f, 14)
	snapshot, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	data := writtenTownData(t, f)
	var actors []*sav.CityUnitData
	for _, o := range data.Objects {
		if o.Unit != nil && (o.Class == "Human" || o.Class == "Unit") {
			actors = append(actors, o.Unit)
		}
	}
	if len(actors) != len(snapshot.Party) {
		t.Fatalf("written town has %d actors for a party of %d", len(actors), len(snapshot.Party))
	}
	kept, hired := 0, 0
	for i, p := range snapshot.Party {
		u := actors[i]
		state, done, deadline := binary.LittleEndian.Uint32(u.Scalar1[4:8]), u.Scalar2[41], binary.LittleEndian.Uint32(u.Scalar2[42:46])
		if end, ok := ends[p.ID]; ok {
			kept++
			if p.Kept == nil || state != 0 || done != 1 || deadline != end || u.Raw154[5] == 0 {
				t.Fatalf("kept %s writes U50/U136/U138/mask %d/%d/%d/%#x, want 0/1/%d/nonzero (carrier %t)", p.ID, state, done, deadline, u.Raw154[5], end, p.Kept != nil)
			}
			continue
		}
		hired++
		if p.Kept != nil || state != 0x0b || done != 1 || deadline != 0 {
			t.Fatalf("town-built %s writes U50/U136/U138 %#x/%d/%d, want 0xb/1/0", p.ID, state, done, deadline)
		}
	}
	if kept == 0 || hired == 0 {
		t.Fatalf("written town holds %d kept and %d town-built members", kept, hired)
	}
	t.Logf("%d kept members wrote the mission's run end, %d town-built members wrote constructor values", kept, hired)
}

// townKeptRegen takes the actor as the next mission minted it, before its
// first tick, wounds it to half health and runs it alone on the mission's
// terrain from sub-tick 0 to until, so no combat, script, healer or order
// moves its pool or starts a run. It returns each health regeneration gain,
// in hundredths, with the sub-tick after the step that applied it.
func townKeptRegen(t *testing.T, f *FrontEnd, id sim.EntityID, until uint64) (ticks []uint64, gains []int64) {
	t.Helper()
	live := f.live
	if live.world.Tick() != 0 {
		t.Fatalf("the mission advanced to sub-tick %d before the measurement", live.world.Tick())
	}
	e, ok := live.entity(id)
	if !ok {
		t.Fatalf("no actor %d", id)
	}
	e.HP, e.HealthHundredths = e.MaxHP/2, 0
	m := live.mission.state.Map
	w, err := sim.NewStructuredWorld(1, live.world.Bounds(), sim.ModeCanonical,
		sim.Terrain{Block: mapload.PassabilityWith(m, f.Table), Cost: mapload.Cost(m), Height: mapload.Height(m)},
		[]sim.Entity{e}, nil, live.world.Relations(), nil, nil, mapload.SpellRules(f.Table), sim.GhostTemplate{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	mapload.BindSourceDerive(w)
	acc := func() int64 {
		e, _ := w.Entity(id)
		return int64(e.HP)*100 + int64(e.HealthHundredths)
	}
	prev := acc()
	for w.Tick() < until {
		before := w.Tick()
		sim.Step(w, nil)
		if w.Tick() == before {
			t.Fatalf("the measurement world did not advance from sub-tick %d", before)
		}
		if now := acc(); now != prev {
			if now < prev {
				t.Fatalf("actor %d lost health at sub-tick %d", id, w.Tick())
			}
			if e, _ := w.Entity(id); e.HP >= e.MaxHP {
				break // the cap cut this gain short
			}
			ticks, gains = append(ticks, w.Tick()), append(gains, now-prev)
			prev = now
		}
	}
	if after, _ := w.Entity(id); after.ActionClock != e.ActionClock {
		t.Fatalf("the actor started a run during the measurement: %+v", after.ActionClock)
	}
	return ticks, gains
}

// assertKeptRegen checks that every gain before stamp+80 is the rate-1 gain
// and every gain after it the rate-3 gain, with at least two of each.
func assertKeptRegen(t *testing.T, stamp uint32, ticks []uint64, gains []int64) {
	t.Helper()
	var one, three []int64
	for i, tick := range ticks {
		age := int64(int32(uint32(tick) - stamp))
		switch {
		case age < 79:
			one = append(one, gains[i])
		case age > 82:
			three = append(three, gains[i])
		}
	}
	if len(one) < 2 || len(three) < 2 {
		t.Fatalf("stamp %d: %d rate-1 and %d rate-3 gains (ticks %v gains %v)", stamp, len(one), len(three), ticks, gains)
	}
	for _, g := range one[1:] {
		if g != one[0] {
			t.Fatalf("stamp %d: rate-1 gains differ: %v (ticks %v gains %v)", stamp, one, ticks, gains)
		}
	}
	for _, g := range three[1:] {
		if g != three[0] {
			t.Fatalf("stamp %d: rate-3 gains differ: %v (ticks %v gains %v)", stamp, three, ticks, gains)
		}
	}
	if three[0] < 3*one[0] || three[0] > 3*one[0]+2 {
		t.Fatalf("stamp %d: gain %d before stamp+80, %d after; want rate 1 then 3 (ticks %v gains %v)", stamp, one[0], three[0], ticks, gains)
	}
	t.Logf("stamp %d: %d gains of %d at rate 1 until sub-tick %d, then %d gains of %d at rate 3", stamp, len(one), one[0], stamp+80, len(three), three[0])
}

// openKeptTownMission opens the town's main mission through the ordinary
// opener and returns the starting hero, whose run end must be the carried
// one.
func openKeptTownMission(t *testing.T, f *FrontEnd, app *ui.App) (sim.EntityID, uint32) {
	t.Helper()
	var stamp uint32
	found := false
	for _, p := range f.Carried {
		if p.StartingHero && p.Kept != nil {
			stamp, found = p.Kept.Deadline, true
		}
	}
	if !found {
		t.Fatal("the town's starting hero carries no kept state")
	}
	if err := app.OpenMission(f.MissionOpener(f.Town.Chapter())); err != nil {
		t.Fatalf("open mission %d: %v", f.Town.Chapter(), err)
	}
	hero := equipmentReturnHero(t, f)
	e, _ := f.live.entity(hero)
	if e.ActionClock != (sim.ActionClock{Known: true, End: stamp}) {
		t.Fatalf("the hero starts with action clock %+v, want known end %d", e.ActionClock, stamp)
	}
	return hero, stamp
}

// A hero kept in the owner's town document regenerates at rate 1 at the next
// mission start until the clock passes his stamp plus 80 sub-ticks, then at
// rate 3 (SAV-1355).
func TestReleaseLoadedTownKeptHeroRegeneratesFromItsStamp(t *testing.T) {
	f, _ := openOriginalSalesTown(t)
	app := f.App("kept regeneration")
	app.Layout(1024, 768)
	hero, stamp := openKeptTownMission(t, f, app)
	if stamp < 160 {
		t.Fatalf("the owner town's hero stamp %d leaves no window past sub-tick 81", stamp)
	}
	ticks, gains := townKeptRegen(t, f, hero, uint64(stamp)+80+5*64)
	assertKeptRegen(t, stamp, ticks, gains)
}

// A hero whose last action ended late in a mission keeps that run end into
// the next mission, reached from the town with and without a town SAVE and
// LOAD, and regenerates at rate 1 until the clock passes it by 80.
func TestReleaseMissionKeptHeroRegeneratesFromItsStamp(t *testing.T) {
	f := releaseFront(t)
	party := f.ChargenParty(ui.ChargenResult{Name: "Kept stamp", Choices: []int{1, 0, 0}, Stats: []int{31, 27, 24, 29}})
	app := f.App("kept stamp")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatalf("open mission 10: %v", err)
	}
	if _, _, err := f.LiveCompleteCampaign(); err != nil {
		t.Fatalf("finish mission 10: %v", err)
	}
	if err := app.OpenMission(f.MissionOpenerWith(20, f.NextParty())); err != nil {
		t.Fatalf("open mission 20: %v", err)
	}
	hero := equipmentReturnHero(t, f)
	live := f.live
	for live.world.Tick() < 300 {
		live.tick()
	}
	e, _ := live.entity(hero)
	live.pending = append(live.pending, sim.MoveTo(hero, sim.CellPoint{X: e.X + 2, Y: e.Y}))
	for live.world.Tick() < 400 {
		live.tick()
	}
	e, _ = live.entity(hero)
	if !e.ActionClock.Known || e.ActionClock.End < 300 {
		t.Fatalf("the move left action clock %+v", e.ActionClock)
	}
	stamp := e.ActionClock.End
	if _, _, err := f.LiveCompleteCampaign(); err != nil {
		t.Fatalf("finish mission 20: %v", err)
	}
	raw := currentTownSave(t, f)

	id, got := openKeptTownMission(t, f, app)
	if got != stamp {
		t.Fatalf("the town carried stamp %d, the mission left %d", got, stamp)
	}
	ticks, gains := townKeptRegen(t, f, id, uint64(stamp)+80+5*64)
	assertKeptRegen(t, stamp, ticks, gains)

	g := currentTownReload(t, raw)
	gapp := g.App("kept stamp reload")
	gapp.Layout(1024, 768)
	id, got = openKeptTownMission(t, g, gapp)
	if got != stamp {
		t.Fatalf("the town LOAD carried stamp %d, the mission left %d", got, stamp)
	}
	ticks, gains = townKeptRegen(t, g, id, uint64(stamp)+80+5*64)
	assertKeptRegen(t, stamp, ticks, gains)
}
