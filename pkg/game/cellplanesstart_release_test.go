package game

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// cellPlaneMissionPair starts mission n twice from its shipped map, with its
// Buildings constructed as mission entry does: once holding the installed
// terrain's cell planes, and once with the Table naming no installed base,
// the World a mission started before the planes existed. ok is false when the
// edition ships no map n.
func cellPlaneMissionPair(t *testing.T, f *FrontEnd, n int) (planes, native *Mission, m *alm.Map, ok bool) {
	t.Helper()
	addr, found := MissionMap(n)
	if !found {
		return nil, nil, nil, false
	}
	raw, err := f.Archives.Containers.ReadFile(addr)
	if err != nil {
		return nil, nil, nil, false
	}
	open := func(table *mapload.Table) (*Mission, *alm.Map) {
		m, err := tableFiles(f.Table).openMap(raw)
		if err != nil {
			t.Fatalf("mission %d: %v", n, err)
		}
		ms, err := startMissionFromWith(m, addr, n, table, openDifficulty, nil, nil)
		if err == nil {
			err = constructCurrentBuildings(ms, f.Archives.Containers)
		}
		if err != nil {
			t.Fatalf("mission %d: %v", n, err)
		}
		return ms, m
	}
	planes, m = open(f.Table)
	bare := *f.Table
	bare.DocumentCellPlanes = true
	native, _ = open(&bare)
	return planes, native, m, true
}

// nativeProjection is the projection a planes-free World's SAVE made of its
// terrain: the installed constructor planes under the installed cost table,
// with the native grid, cost and height folded over each in-bounds cell. It
// is written here independently of the engine's plane builder.
func nativeProjection(t *testing.T, f *FrontEnd, m *alm.Map, w *sim.World) *sim.SavedCellPlanes {
	t.Helper()
	costs := (*mapload.Table)(nil).TerrainCostTable()
	if f.Table.TerrainCosts != nil {
		costs = *f.Table.TerrainCosts
	}
	planes, err := mapload.OriginalTerrainPlanes(m, costs)
	if err != nil {
		t.Fatal(err)
	}
	p := w.CurrentPolicy().Terrain
	width := int(w.Bounds().Width)
	for at, grid := range p.Block {
		key := uint16(at/width)<<8 | uint16(at%width)
		stat := grid&3 | (grid&8)>>1
		if grid&2 != 0 || (at < len(m.Overlay) && m.Overlay[at] != 0) {
			stat |= 4
		}
		planes.Static[key] = planes.Static[key]&^7 | stat
		planes.Dynamic[key] = planes.Dynamic[key]&^7 | planes.Static[key]&7
		planes.Cost[key], planes.Height[key], planes.CostKnown[key] = p.Cost[at], p.Height[at], 1
	}
	return planes
}

// cellPlaneCauses names the start writer of each cell a mission's World holds
// beyond its terrain: a living on-map actor's footprint (MOVE-085), a
// Building's rectangle (TERR-240), a Sack's cell (SAV-SACKENTRY-590) and a
// cell effect's or a cell tail's cell.
func cellPlaneCauses(w *sim.World) map[uint16]string {
	causes := map[uint16]string{}
	for _, s := range w.Structures() {
		for dy := int32(0); dy < int32(s.Height); dy++ {
			for dx := int32(0); dx < int32(s.Width); dx++ {
				if x, y := s.Col+dx, s.Row+dy; x >= 0 && y >= 0 && x < 256 && y < 256 {
					causes[uint16(y)<<8|uint16(x)] = "building"
				}
			}
		}
	}
	for _, s := range w.Sacks() {
		causes[uint16(s.Y)<<8|uint16(s.X)] = "sack"
	}
	for _, c := range w.CellEffects() {
		causes[uint16(c.Y)<<8|uint16(c.X)] = "cell effect"
	}
	for _, c := range w.CellTails() {
		causes[uint16(c.Y)<<8|uint16(c.X)] = "cell tail"
	}
	for _, e := range w.Entities() {
		if !e.Alive() || e.OffMap {
			continue
		}
		side := int32(e.TokenSize)
		if side == 0 {
			side = 1
		}
		for dy := int32(0); dy < side; dy++ {
			for dx := int32(0); dx < side; dx++ {
				if x, y := e.X+dx, e.Y+dy; x >= 0 && y >= 0 && x < 256 && y < 256 {
					causes[uint16(y)<<8|uint16(x)] = "actor"
				}
			}
		}
	}
	return causes
}

// cellPlaneDifferences counts, by plane, cause and the two bytes, every cell
// where got and want differ; a cell no start writer explains is returned in
// unexplained.
func cellPlaneDifferences(got, want *sim.SavedCellPlanes, causes map[uint16]string) (counts map[string]int, unexplained []string) {
	counts = map[string]int{}
	for key := 0; key < 65536; key++ {
		k := uint16(key)
		for _, pl := range []struct {
			name string
			a, b byte
		}{
			{"Static", got.Static[k], want.Static[k]},
			{"Dynamic", got.Dynamic[k], want.Dynamic[k]},
			{"Cost", got.Cost[k], want.Cost[k]},
			{"CostKnown", got.CostKnown[k], want.CostKnown[k]},
			{"Height", got.Height[k], want.Height[k]},
		} {
			if pl.a == pl.b {
				continue
			}
			cause, ok := causes[k]
			if !ok {
				unexplained = append(unexplained, fmt.Sprintf("%s %04x %#x/%#x", pl.name, k, pl.a, pl.b))
				continue
			}
			counts[fmt.Sprintf("%s/%s(%#x>%#x)", pl.name, cause, pl.a, pl.b)]++
		}
	}
	return counts, unexplained
}

func sortedCounts(counts map[string]int) string {
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, fmt.Sprintf("%s:%d", k, counts[k]))
	}
	return strings.Join(out, " ")
}

// TestReleaseEveryMissionStartHoldsTheInstalledCellPlanes starts every
// mission the install ships. Its World holds the cell planes from mission
// start, with the installed cost table, and they equal the projection a
// planes-free World's SAVE made of its terrain (the installed constructor
// planes with the native grid, cost and height folded in) on every cell but
// those a start writer changes: a living actor's footprint and a Building's
// rectangle. Loss control: a start that skipped the actors' cell entry would
// hold no actor-caused Dynamic difference, and a mission with a living actor
// must show one.
func TestReleaseEveryMissionStartHoldsTheInstalledCellPlanes(t *testing.T) {
	f := releaseFront(t)
	opened, total := 0, map[string]int{}
	for n := 1; n <= 150; n++ {
		ms, nat, m, ok := cellPlaneMissionPair(t, f, n)
		if !ok {
			continue
		}
		opened++
		got, present := ms.World.SavedCellPlanes()
		if !present {
			t.Fatalf("mission %d starts without cell planes", n)
		}
		if _, held := nat.World.SavedCellPlanes(); held {
			t.Fatalf("mission %d: the planes-free control holds cell planes", n)
		}
		want := nativeProjection(t, f, m, nat.World)
		if got.Costs != want.Costs {
			t.Fatalf("mission %d cost table %v, installed %v", n, got.Costs, want.Costs)
		}
		counts, unexplained := cellPlaneDifferences(got, want, cellPlaneCauses(ms.World))
		for _, d := range unexplained {
			t.Errorf("mission %d: %s (carrier/projection) has no start writer", n, d)
		}
		living := false
		for _, e := range ms.World.Entities() {
			if e.Alive() && !e.OffMap {
				living = true
			}
		}
		occupied := 0
		for k, v := range counts {
			total[k[:strings.IndexByte(k, '(')]] += v
			if strings.HasPrefix(k, "Dynamic/actor(") {
				occupied += v
			}
		}
		if living && occupied == 0 {
			t.Errorf("mission %d: living actors occupy no cell slot", n)
		}
		t.Logf("mission %d: %s", n, sortedCounts(counts))
	}
	if opened < 20 {
		t.Fatalf("opened only %d missions", opened)
	}
	t.Logf("%d missions; differences by plane and writer: %s", opened, sortedCounts(total))
}

// TestReleaseMissionStartPlanesAgainstTheRestartSlots compares the cell planes
// of a fresh start of each restart slot's mission (game9999.sav) with the
// planes the original's own start carries, as its LOAD builds them. Each
// difference is listed by plane and start writer; a terrain cell no writer
// explains fails.
func TestReleaseMissionStartPlanesAgainstTheRestartSlots(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("no AGAINROM_ASSETS: release integration needs a lawful install")
	}
	root := os.Getenv("AGAINROM_SAVE_CORPUS")
	if root == "" {
		root = filepath.Join(os.Getenv("AGAINROM_ASSETS"), "..", "saves")
	}
	paths, _ := filepath.Glob(filepath.Join(root, "*", "game9999.sav"))
	if len(paths) == 0 {
		t.Skip("no restart slot in the save corpus")
	}
	f := releaseFront(t)
	compared := 0
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		loaded, _, err := loadOriginalMission(f, raw)
		if err != nil {
			t.Logf("%s: not this edition's mission (%v)", path, err)
			continue
		}
		theirs, present := loaded.World.SavedCellPlanes()
		if !present {
			t.Fatalf("%s: the LOAD holds no cell planes", path)
		}
		ms, _, _, ok := cellPlaneMissionPair(t, f, loaded.Number)
		if !ok {
			t.Fatalf("%s: mission %d has no map", path, loaded.Number)
		}
		ours, _ := ms.World.SavedCellPlanes()
		causes := cellPlaneCauses(ms.World)
		for k, v := range cellPlaneCauses(loaded.World) {
			if _, held := causes[k]; !held {
				causes[k] = v
			}
		}
		counts, unexplained := cellPlaneDifferences(ours, theirs, causes)
		for _, d := range unexplained {
			t.Errorf("%s mission %d: %s (fresh/original) has no start writer", path, loaded.Number, d)
		}
		compared++
		t.Logf("%s mission %d: %s", path, loaded.Number, sortedCounts(counts))
	}
	if compared == 0 {
		t.Skip("no restart slot of this edition")
	}
}

// cellPlaneWalkers orders up to four living on-map actors to walk twelve
// cells across the map and returns the commands.
func cellPlaneWalkers(w *sim.World) []sim.Command {
	b := w.Bounds()
	var cmds []sim.Command
	for _, e := range w.Entities() {
		if len(cmds) == 4 {
			break
		}
		if !e.Alive() || e.OffMap {
			continue
		}
		x, y := e.X+12, e.Y+8
		if x >= b.Width-8 {
			x = e.X - 12
		}
		if y >= b.Height-8 {
			y = e.Y - 8
		}
		cmds = append(cmds, sim.MoveTo(e.ID, sim.CellPoint{X: x, Y: y}))
	}
	return cmds
}

type cellPlaneTrack map[sim.EntityID][]sim.CellPoint

// cellPlaneRun steps w for ticks after cmds and returns each actor's cells.
func cellPlaneRun(w *sim.World, cmds []sim.Command, ticks int) cellPlaneTrack {
	track := cellPlaneTrack{}
	sim.Step(w, cmds)
	for tick := 0; tick < ticks; tick++ {
		sim.Step(w, nil)
		for _, e := range w.Entities() {
			track[e.ID] = append(track[e.ID], sim.CellPoint{X: e.X, Y: e.Y})
		}
	}
	return track
}

// cellPlaneFirstDifference is the earliest tick where a and b place an actor
// differently and that actor, lowest ID first; at is -1 when none.
func cellPlaneFirstDifference(a, b cellPlaneTrack) (first sim.EntityID, at int) {
	at = -1
	for id, x := range a {
		y := b[id]
		for tick := range x {
			if tick >= len(y) || x[tick] != y[tick] {
				if at < 0 || tick < at || tick == at && id < first {
					first, at = id, tick
				}
				break
			}
		}
	}
	return first, at
}

// cellPlaneTrackDifference names the earliest tick where a and b place an
// actor differently, the actor, its cells in a and b, and every actor that
// stands on its cell in a at that tick or the one before; "" when none.
func cellPlaneTrackDifference(a, b cellPlaneTrack) string {
	first, at := cellPlaneFirstDifference(a, b)
	if at < 0 {
		return ""
	}
	ids := make([]sim.EntityID, 0, len(a))
	for id := range a {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	x, y := a[first], b[first]
	cell := x[at]
	var near []string
	for _, id := range ids {
		for _, tick := range []int{max(at-1, 0), at} {
			if id != first && (a[id][tick] == cell || tick < len(b[id]) && b[id][tick] == cell) {
				near = append(near, fmt.Sprintf("%d@%d %v/%v", id, tick, a[id][tick], b[id][min(tick, len(b[id])-1)]))
			}
		}
	}
	return fmt.Sprintf("actor %d tick %d: from %v to %v / %v; on that cell %v", first, at, x[max(at-1, 0)], cell,
		y[min(at, len(y)-1)], near)
}

// openBuildingCells is every cell of a Building's rectangle whose blocking
// bit is clear: a doorway or a bridge deck (TERR-STRUCT-071).
func openBuildingCells(w *sim.World) map[uint16]bool {
	open := map[uint16]bool{}
	for _, s := range w.Structures() {
		for dy := int32(0); dy < int32(s.Height); dy++ {
			for dx := int32(0); dx < int32(s.Width); dx++ {
				bit := uint32(dy*int32(s.Width)+dx) & 31
				if x, y := s.Col+dx, s.Row+dy; s.Blocking&(1<<bit) == 0 && x >= 0 && y >= 0 && x < 256 && y < 256 {
					open[uint16(y)<<8|uint16(x)] = true
				}
			}
		}
	}
	return open
}

// TestReleaseMissionRoutesAndCostsMatchTheNativeReader starts every shipped
// mission with its cell planes and without them (the native grid, cost and
// height readers), orders up to four actors to walk and steps both Worlds
// 256 ticks. Height reads the same on every cell, and cost on every cell but
// an open Building cell, which the planes price at the cracked-ground cost
// (TERR-STRUCT-071). Where a walk differs, the first actor to differ crosses
// a cell whose bytes differ: an open Building cell, or for a ghost a cell
// with Static bit 2, which blocks a ghost on the planes as on a loaded World.
// Loss control: on the first mission with a walk, the planes World's Cost
// plane is rewritten along the native route; the planes World then walks
// differently from the native one, which a reader ignoring the planes would
// not.
func TestReleaseMissionRoutesAndCostsMatchTheNativeReader(t *testing.T) {
	f := releaseFront(t)
	opened, walks, changed, controlled := 0, 0, map[string]int{}, false
	for n := 1; n <= 150; n++ {
		ms, nat, _, ok := cellPlaneMissionPair(t, f, n)
		if !ok {
			continue
		}
		opened++
		planes, _ := ms.World.SavedCellPlanes()
		open := openBuildingCells(ms.World)
		p := nat.World.CurrentPolicy().Terrain
		width := int(nat.World.Bounds().Width)
		cracked := 0
		for at := range p.Cost {
			key := uint16(at/width)<<8 | uint16(at%width)
			if planes.Height[key] != p.Height[at] {
				t.Errorf("mission %d cell %04x: planes height %d, native %d", n, key, planes.Height[key], p.Height[at])
			}
			if planes.Cost[key] == p.Cost[at] {
				continue
			}
			if !open[key] || planes.Cost[key] != planes.Costs[5] {
				t.Errorf("mission %d cell %04x: planes cost %d, native %d", n, key, planes.Cost[key], p.Cost[at])
			}
			cracked++
		}
		cmds := cellPlaneWalkers(nat.World)
		if len(cmds) == 0 {
			continue
		}
		var control *sim.World
		if !controlled {
			control = cloneCellPlaneWorld(t, ms.World)
		}
		got := cellPlaneRun(ms.World, cmds, 256)
		want := cellPlaneRun(nat.World, cmds, 256)
		walks++
		if id, at := cellPlaneFirstDifference(want, got); at >= 0 {
			e, _ := nat.World.Entity(id)
			cause, kind := "", ""
			for _, track := range []cellPlaneTrack{want, got} {
				for _, c := range track[id][max(at-1, 0):] {
					key := uint16(c.Y)<<8 | uint16(c.X)
					switch {
					case cause == "" && open[key] && planes.Cost[key] != p.Cost[int(c.Y)*width+int(c.X)]:
						cause, kind = fmt.Sprintf("open Building cell %04x at the cracked-ground cost", key), "cracked cost"
					case cause == "" && e.Domain == sim.DomainGhost && planes.Static[key]&4 != 0:
						cause, kind = fmt.Sprintf("ghost blocked by Static bit 2 at %04x", key), "ghost Static bit 2"
					}
				}
			}
			d := cellPlaneTrackDifference(want, got)
			if cause == "" {
				t.Errorf("mission %d: planes and native routes differ at %s, crossing no cell whose bytes differ", n, d)
			} else {
				changed[kind]++
				t.Logf("mission %d: %s; %s", n, d, cause)
			}
		}
		t.Logf("mission %d: %d open Building cells at the cracked-ground cost", n, cracked)
		if control == nil {
			continue
		}
		var residue []sim.CurrentCellPlaneResidue
		for _, path := range want {
			for _, c := range path[1:] {
				key := uint16(c.Y)<<8 | uint16(c.X)
				cost := byte(1)
				if planes.Cost[key] < 64 {
					cost = 250
				}
				one := []sim.CurrentCellPlaneResidue{{Cell: key, Cost: &cost}}
				if control.RestoreCurrentCellCosts(nil, one...) == nil {
					residue = append(residue, one...)
				}
			}
		}
		if len(residue) == 0 {
			continue
		}
		controlled = true
		if cellPlaneTrackDifference(want, cellPlaneRun(control, cmds, 256)) == "" {
			t.Fatalf("mission %d: %d rewritten Cost cells left the walk unchanged", n, len(residue))
		}
		t.Logf("mission %d: loss control rewrote %d Cost cells and changed the walk", n, len(residue))
	}
	if opened < 20 || walks == 0 || !controlled {
		t.Fatalf("opened %d missions, %d walks, loss control %t", opened, walks, controlled)
	}
	t.Logf("%d missions, %d walks; changed walks by cause: %v", opened, walks, changed)
}

func cloneCellPlaneWorld(t *testing.T, w *sim.World) *sim.World {
	t.Helper()
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var out sim.World
	if err := out.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if out.Hash() != w.Hash() {
		t.Fatal("cloned World hash differs")
	}
	return &out
}

// cellPlaneSave SAVEs the mission f holds and returns the SAV.
func cellPlaneSave(t *testing.T, f *FrontEnd, label string) []byte {
	t.Helper()
	before := f.live.world.Hash()
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(label, err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil || f.live.world.Hash() != before {
		t.Fatal(label, err)
	}
	return raw
}

// cellPlaneLoad opens raw in a new front end as a cold LOAD; ok is false
// when raw is a town or another edition's mission.
func cellPlaneLoad(t *testing.T, raw []byte, label string) (*FrontEnd, bool) {
	t.Helper()
	f := releaseFront(t)
	f.SetTipsOff(true)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		return nil, false
	}
	a := f.App(label)
	a.SetCutscenes(nil)
	if err := a.OpenMission(open); err != nil {
		return nil, false
	}
	return f, true
}

// cellPlanesEqual fails when the two Worlds' cell planes differ in any byte.
func cellPlanesEqual(t *testing.T, want, got *sim.World, cut string) {
	t.Helper()
	a, okA := want.SavedCellPlanes()
	b, okB := got.SavedCellPlanes()
	if !okA || !okB {
		t.Fatalf("%s: cell planes held %t / %t", cut, okA, okB)
	}
	if *a != *b {
		counts, unexplained := cellPlaneDifferences(a, b, map[uint16]string{})
		t.Fatalf("%s: cell planes differ: %s %v", cut, sortedCounts(counts), unexplained[:min(len(unexplained), 8)])
	}
}

// cellPlaneTicks steps every World with the same walk for 256 ticks and fails
// at the first tick whose World hash or Cost plane differs from the first.
func cellPlaneTicks(t *testing.T, cut string, worlds ...*sim.World) {
	t.Helper()
	cmds := cellPlaneWalkers(worlds[0])
	for _, w := range worlds {
		sim.Step(w, cmds)
	}
	for tick := 0; tick < 256; tick++ {
		for _, w := range worlds {
			sim.Step(w, nil)
		}
		for i, w := range worlds[1:] {
			if w.Hash() != worlds[0].Hash() {
				t.Fatalf("%s: World %d hash differs at tick %d", cut, i+1, tick)
			}
		}
	}
	for i, w := range worlds[1:] {
		cellPlanesEqual(t, worlds[0], w, fmt.Sprintf("%s World %d after 256 ticks", cut, i+1))
	}
}

// TestReleaseCellPlanesSurviveSaveColdLoadAndTicks SAVEs three missions and
// LOADs each SAV cold in a separate World: a fresh mission, the mission of
// that SAV (an engine-SAV mission) and a mission from an original SAV of the
// corpus. Each cold World holds the continuous World's hash and cell planes,
// and both walk alike with equal hashes and Cost planes over the next 256
// ticks. Loss control: a SAV whose Block records hold the constructor's
// bytes for the cells the mission start occupied LOADs to a different World.
func TestReleaseCellPlanesSurviveSaveColdLoadAndTicks(t *testing.T) {
	f := releaseFront(t)
	f.SetTipsOff(true)
	a := f.App("cell planes fresh")
	a.SetCutscenes(nil)
	if err := a.OpenMission(f.MissionOpenerWith(10, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	fresh := cellPlaneSave(t, f, "Cell planes fresh")
	engine, ok := cellPlaneLoad(t, fresh, "cell planes engine SAV")
	if !ok {
		t.Fatal("fresh mission SAV did not LOAD")
	}
	assertCurrentWorldEqual(t, f.live.world, engine.live.world, "fresh SAVE and cold LOAD")
	cellPlanesEqual(t, f.live.world, engine.live.world, "fresh SAVE and cold LOAD")
	again, ok := cellPlaneLoad(t, cellPlaneSave(t, engine, "Cell planes engine"), "cell planes engine SAV again")
	if !ok {
		t.Fatal("engine-SAV mission SAV did not LOAD")
	}
	assertCurrentWorldEqual(t, engine.live.world, again.live.world, "engine-SAV SAVE and cold LOAD")
	cellPlanesEqual(t, engine.live.world, again.live.world, "engine-SAV SAVE and cold LOAD")

	doc, err := sav.DecodeDocumentData(fresh)
	if err != nil {
		t.Fatal(err)
	}
	base, err := readOriginalCellPlanes(f.live.mission.state.Map, f.Archives.Containers)
	if err != nil || base == nil {
		t.Fatal("installed constructor planes", err)
	}
	changed := 0
	for i, b := range doc.World.Blocks {
		if b.Dyn&0xc0 != 0 {
			doc.World.Blocks[i].Dyn, doc.World.Blocks[i].Static = base.Dynamic[b.Cell], base.Static[b.Cell]
			changed++
		}
	}
	if changed == 0 {
		t.Fatal("fresh SAV holds no occupied Block record")
	}
	crooked, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if control, ok := cellPlaneLoad(t, crooked, "cell planes loss control"); ok {
		if p, _ := control.live.world.SavedCellPlanes(); control.live.world.Hash() == f.live.world.Hash() {
			q, _ := f.live.world.SavedCellPlanes()
			t.Fatalf("constructor Block bytes on %d occupied cells LOAD to the same World (planes equal %t)", changed, *p == *q)
		}
	}

	cellPlaneTicks(t, "fresh and engine-SAV missions", f.live.world, engine.live.world, again.live.world)

	root := os.Getenv("AGAINROM_SAVE_CORPUS")
	if root == "" {
		root = filepath.Join(os.Getenv("AGAINROM_ASSETS"), "..", "saves")
	}
	paths, _ := filepath.Glob(filepath.Join(root, "*", "*.sav"))
	sort.Strings(paths)
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		original, ok := cellPlaneLoad(t, raw, "cell planes original SAV")
		if !ok {
			continue
		}
		cold, ok := cellPlaneLoad(t, cellPlaneSave(t, original, "Cell planes original"), "cell planes original resave")
		if !ok {
			t.Fatalf("%s: the resave did not LOAD", path)
		}
		assertCurrentWorldEqual(t, original.live.world, cold.live.world, path+" SAVE and cold LOAD")
		cellPlanesEqual(t, original.live.world, cold.live.world, path+" SAVE and cold LOAD")
		cellPlaneTicks(t, path, original.live.world, cold.live.world)
		t.Logf("%s mission %d: SAVE, cold LOAD and 256 ticks equal", path, original.live.mission.state.Number)
		return
	}
	t.Log("no original mission SAV of this edition in the corpus")
}
