package game

import (
	"image"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type sackFeetRig struct {
	front *FrontEnd
	app   *ui.App
	live  *mapWorld
	hero  sim.EntityID
}

func newSackFeetRig(t *testing.T, mission int) sackFeetRig {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("inventory drop at feet")
	if err := app.OpenMission(f.NewGameOpener(mission, ui.ChargenResult{Name: "Sack drop", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	r := sackFeetRig{front: f, app: app, live: f.live, hero: f.live.mission.ids[0]}
	for range 32 {
		if err := stepConsumableApp(app); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessSelectEntity(uint32(r.hero)); err != nil {
		t.Fatal(err)
	}
	r.live.refreshInventorySelection()
	for slot := sim.EquipSlots; sackFeetCarried(r.live.world, r.hero, true) < 2 && slot >= 1; slot-- {
		worn, _ := r.live.world.EquippedItems(r.hero)
		if worn[slot-1].Empty() {
			continue
		}
		x, y, err := app.HeadlessDollSlotPoint(slot)
		if err != nil {
			continue
		}
		for _, edge := range []string{"press", "release"} {
			if err := app.HeadlessPointer(edge, x, y); err != nil {
				t.Fatal(err)
			}
		}
		if err := stepConsumableApp(app); err != nil {
			t.Fatal(err)
		}
	}
	if n := sackFeetCarried(r.live.world, r.hero, true); n < 2 {
		t.Fatalf("normal doll taps produced %d carried item units, want at least two", n)
	}
	return r
}

func sackFeetCarried(w *sim.World, hero sim.EntityID, ordinary bool) int {
	pack, _ := w.CarriedStacks(hero)
	n := 0
	for _, item := range pack {
		if !ordinary || data.ItemCode(item.Code) != data.QuestDocumentCode {
			n += int(item.Count)
		}
	}
	return n
}

func sackFeetClone(t *testing.T, w *sim.World) *sim.World {
	t.Helper()
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	copy := new(sim.World)
	if err := copy.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	mapload.BindSourceDerive(copy)
	return copy
}

func sackFeetOccupied(w *sim.World, at sim.CellPoint, except sim.EntityID) bool {
	for _, e := range w.Entities() {
		if e.ID == except || e.OffMap || !e.Alive() {
			continue
		}
		side := int32(e.TokenSize)
		if side < 1 {
			side = 1
		}
		if at.X >= e.X && at.X < e.X+side && at.Y >= e.Y && at.Y < e.Y+side {
			return true
		}
	}
	return false
}

func sackFeetStructure(w *sim.World, at sim.CellPoint) bool {
	for _, row := range w.StructureOccupancy() {
		if int32(row.Cell&255) == at.X && int32(row.Cell>>8) == at.Y {
			return true
		}
	}
	return false
}

func (r sackFeetRig) standing(t *testing.T) sim.CellPoint {
	t.Helper()
	e, ok := r.live.entity(r.hero)
	if !ok || e.OffMap || !e.Alive() || e.HasTarget || e.Transit != 0 || len(r.live.world.Route(r.hero)) != 0 {
		t.Fatalf("dropper is not standing without a walk: %+v route=%v", e, r.live.world.Route(r.hero))
	}
	return sim.CellPoint{X: e.X, Y: e.Y}
}

func (r sackFeetRig) centre(t *testing.T) sim.CellPoint {
	t.Helper()
	at := r.standing(t)
	r.live.push()
	inspectionCentre(r.live, int(at.X), int(at.Y))
	if err := r.app.HeadlessSelectEntity(uint32(r.hero)); err != nil {
		t.Fatal(err)
	}
	r.live.refreshInventorySelection()
	return at
}

func (r sackFeetRig) placeNear(t *testing.T, target sim.CellPoint) bool {
	t.Helper()
	b := r.live.world.Bounds()
	grid := r.live.world.CurrentPolicy().Terrain.Block
	for dy := int32(-2); dy <= 2; dy++ {
		for dx := int32(-2); dx <= 2; dx++ {
			at := sim.CellPoint{X: target.X + dx, Y: target.Y + dy}
			if dx == 0 && dy == 0 || at.X < 1 || at.Y < 1 || at.X >= b.Width-1 || at.Y >= b.Height-1 ||
				grid[int(at.Y*b.Width+at.X)]&1 != 0 || sackFeetOccupied(r.live.world, at, r.hero) || sackFeetStructure(r.live.world, at) {
				continue
			}
			copy := sackFeetClone(t, r.live.world)
			if copy.HeadlessPlace(r.hero, at.X, at.Y) != nil {
				continue
			}
			e, _ := copy.Entity(r.hero)
			if e.X-target.X < -2 || e.X-target.X > 2 || e.Y-target.Y < -2 || e.Y-target.Y > 2 || e.X == target.X && e.Y == target.Y {
				continue
			}
			if err := r.live.world.HeadlessPlace(r.hero, at.X, at.Y); err != nil {
				t.Fatal(err)
			}
			r.live.tick()
			r.centre(t)
			return true
		}
	}
	return false
}

func (r sackFeetRig) blockedTarget(t *testing.T) sim.CellPoint {
	t.Helper()
	b := r.live.world.Bounds()
	grid := r.live.world.CurrentPolicy().Terrain.Block
	planes, saved := r.live.world.SavedCellPlanes()
	for y := int32(24); y < b.Height-24; y++ {
		for x := int32(16); x < b.Width-16; x++ {
			at := sim.CellPoint{X: x, Y: y}
			if grid[int(y*b.Width+x)]&1 == 0 || saved && planes.Dynamic[y*256+x]&1 == 0 ||
				sackFeetStructure(r.live.world, at) || sackFeetOccupied(r.live.world, at, r.hero) {
				continue
			}
			if r.placeNear(t, at) {
				t.Logf("installed ground-blocked target %v", at)
				return at
			}
		}
	}
	t.Fatal("no installed ground-blocked cell with an adjacent standing position")
	return sim.CellPoint{}
}

func (r sackFeetRig) groundPixel(t *testing.T, at sim.CellPoint) image.Point {
	t.Helper()
	for y := 130; y < 550; y += 2 {
		for x := 350; x < 700; x += 2 {
			cx, cy, err := r.app.HeadlessDropCell(x, y)
			if err == nil && int32(cx) == at.X && int32(cy) == at.Y {
				return image.Pt(x, y)
			}
		}
	}
	t.Fatalf("cell %v has no visible map pixel", at)
	return image.Point{}
}

func (r sackFeetRig) drag(t *testing.T, pixel image.Point, requested, landing sim.CellPoint) {
	t.Helper()
	origin := r.standing(t)
	pack, _ := r.live.world.CarriedStacks(r.hero)
	index := -1
	for i, item := range pack {
		if item.Count != 0 && data.ItemCode(item.Code) != data.QuestDocumentCode {
			index = i
			break
		}
	}
	if index < 0 {
		t.Fatal("no carried item remains to drag")
	}
	code := pack[index].Code
	beforeHeld := sackFeetCarried(r.live.world, r.hero, false)
	beforeTotal := thrownHeld(r.live.world, r.hero)
	beforeSacks := len(r.live.world.Sacks())
	beforeSack := groundAt(r.live.world.Sacks(), landing.X, landing.Y)
	beforeItems := 0
	var beforeIdentity sim.SavedObjectID
	if beforeSack != nil {
		beforeItems, beforeIdentity = len(beforeSack.Items), beforeSack.ObjectID
	}
	sx, sy, err := r.app.HeadlessPackCellPoint(index)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []struct {
		name string
		at   image.Point
	}{{"press", image.Pt(sx, sy)}, {"move", pixel}, {"release", pixel}} {
		if err := r.app.HeadlessPointer(edge.name, edge.at.X, edge.at.Y); err != nil {
			t.Fatal(err)
		}
	}
	cx, cy, err := r.app.HeadlessDropCell(pixel.X, pixel.Y)
	if err != nil || int32(cx) != requested.X || int32(cy) != requested.Y {
		t.Fatalf("pointer pixel %v resolved to %d,%d (%v), want %v", pixel, cx, cy, err, requested)
	}
	r.live.frameAdvance(func() {
		if len(r.live.pending) != 1 || r.live.pending[0].Kind != sim.KindDropCarried || r.live.pending[0].Entity != r.hero || r.live.pending[0].Spell != uint16(index) {
			t.Fatalf("inventory gesture queued %+v, want one carried drop and no walk", r.live.pending)
		}
		r.live.tick()
	})
	e, _ := r.live.entity(r.hero)
	if e.X != origin.X || e.Y != origin.Y || e.HasTarget || e.Transit != 0 || len(r.live.world.Route(r.hero)) != 0 {
		t.Fatalf("drop walked or waited for movement: origin=%v actor=%+v route=%v", origin, e, r.live.world.Route(r.hero))
	}
	if n := sackFeetCarried(r.live.world, r.hero, false); n != beforeHeld-1 {
		t.Fatalf("inventory holds %d item units after drag, want %d", n, beforeHeld-1)
	}
	if n := thrownHeld(r.live.world, r.hero); n != beforeTotal {
		t.Fatalf("inventory, worn and ground hold %d item units, want %d", n, beforeTotal)
	}
	sack := groundAt(r.live.world.Sacks(), landing.X, landing.Y)
	if sack == nil || len(sack.Items) != beforeItems+1 || sack.Items[len(sack.Items)-1] != code {
		t.Fatalf("drag to %v left sacks %v, want item %#x in sack at %v (previous units %d)", requested, thrownSackItems(r.live.world), code, landing, beforeItems)
	}
	wantSacks := beforeSacks + 1
	if beforeSack != nil {
		wantSacks = beforeSacks
		if sack.ObjectID != beforeIdentity {
			t.Fatalf("existing feet sack identity changed from %d to %d", beforeIdentity, sack.ObjectID)
		}
	}
	if n := len(r.live.world.Sacks()); n != wantSacks {
		t.Fatalf("ground has %d sacks, want %d", n, wantSacks)
	}
	t.Logf("pointer %v resolved %v; item %#x landed %v without walking; carried %d->%d, sack units %d->%d", pixel, requested, code, landing, beforeHeld, beforeHeld-1, beforeItems, beforeItems+1)
}

func (r sackFeetRig) saveContinuation(t *testing.T) {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := r.front.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil {
		t.Fatal("SAVE after inventory drop", err)
	}
	cold, _ := loadSAVWindow(t, store, name)
	for tick := 0; tick <= 16; tick++ {
		if got, want := cold.live.world.Hash(), r.live.world.Hash(); got != want {
			t.Fatalf("SAVE/cold LOAD tick %d changed World hash: got %016x want %016x", tick, got, want)
		}
		if tick < 16 {
			r.live.tick()
			cold.live.tick()
		}
	}
	t.Logf("SAVE %s and cold LOAD preserve World hash through 16 matching ticks", name)
}

func (r sackFeetRig) fallback(t *testing.T, target sim.CellPoint, pixel image.Point) {
	t.Helper()
	own := r.standing(t)
	r.drag(t, pixel, target, own)
	r.drag(t, pixel, target, own)
	r.saveContinuation(t)
}

func TestReleaseInventoryDropOutsideWindowLandsAtFeet(t *testing.T) {
	r := newSackFeetRig(t, 20)
	own := r.centre(t)
	target := sim.CellPoint{X: own.X + 5, Y: own.Y}
	r.fallback(t, target, r.groundPixel(t, target))
}

func TestReleaseInventoryDropGroundBlockedLandsAtFeet(t *testing.T) {
	r := newSackFeetRig(t, 20)
	target := r.blockedTarget(t)
	r.fallback(t, target, r.groundPixel(t, target))
}

func TestReleaseInventoryDropOccupiedLandsAtFeet(t *testing.T) {
	r := newSackFeetRig(t, 10)
	hero, _ := r.live.entity(r.hero)
	for _, e := range r.live.world.Entities() {
		if e.ID == r.hero || e.OffMap || !e.Alive() || e.HasTarget || e.HasAttackTarget || e.Transit != 0 || r.live.world.Relations().Hostile(hero.Owner, e.Owner) {
			continue
		}
		target := sim.CellPoint{X: e.X, Y: e.Y}
		if r.placeNear(t, target) {
			r.fallback(t, target, r.groundPixel(t, target))
			return
		}
	}
	t.Fatal("no idle nonhostile installed unit with an adjacent standing position")
}

func TestReleaseInventoryDropStructureLandsAtFeet(t *testing.T) {
	r := newSackFeetRig(t, 10)
	for _, row := range r.live.world.StructureOccupancy() {
		target := sim.CellPoint{X: int32(row.Cell & 255), Y: int32(row.Cell >> 8)}
		if target.X < 16 || target.Y < 24 {
			continue
		}
		if r.placeNear(t, target) {
			r.fallback(t, target, r.groundPixel(t, target))
			return
		}
	}
	t.Fatal("no installed structure cell with an adjacent standing position")
}

func TestReleaseInventoryDropOffMapLandsAtFeet(t *testing.T) {
	r := newSackFeetRig(t, 20)
	r.centre(t)
	r.live.view.Camera().ClearClampBounds()
	r.live.view.Camera().CenterOn(0, 0)
	target := sim.CellPoint{X: 1 << 20, Y: 1 << 20}
	for y := 80; y < 550; y += 2 {
		for x := 24; x < 80; x += 2 {
			cx, cy, err := r.app.HeadlessDropCell(x, y)
			if err == nil && int32(cx) == target.X && int32(cy) == target.Y {
				r.fallback(t, target, image.Pt(x, y))
				return
			}
		}
	}
	t.Fatal("no undrawn or border map pixel at the legal camera edge")
}

func TestReleaseInventoryDropUnreachableLandsAtFeet(t *testing.T) {
	r := newSackFeetRig(t, 20)
	target := r.blockedTarget(t)
	copy := sackFeetClone(t, r.live.world)
	sim.Step(copy, []sim.Command{sim.MoveTo(r.hero, target)})
	for range 64 {
		sim.Step(copy, nil)
	}
	e, _ := copy.Entity(r.hero)
	if e.X == target.X && e.Y == target.Y || len(copy.Route(r.hero)) != 0 {
		t.Fatalf("blocked installed target %v has a Move route: actor=%+v route=%v", target, e, copy.Route(r.hero))
	}
	t.Logf("independent Move control cannot reach %v after 65 ticks; actor %d,%d route=%v", target, e.X, e.Y, copy.Route(r.hero))
	r.fallback(t, target, r.groundPixel(t, target))
}

func TestReleaseInventoryDropInsideWindowKeepsAim(t *testing.T) {
	r := newSackFeetRig(t, 20)
	grid := r.live.world.CurrentPolicy().Terrain.Block
	b := r.live.world.Bounds()
	for y := int32(24); y < b.Height-24; y++ {
		for x := int32(16); x < b.Width-16; x++ {
			copy := sackFeetClone(t, r.live.world)
			if copy.HeadlessPlace(r.hero, x, y) != nil {
				continue
			}
			e, _ := copy.Entity(r.hero)
			own := sim.CellPoint{X: e.X, Y: e.Y}
			first, corner := sim.CellPoint{X: e.X + 1, Y: e.Y + 1}, sim.CellPoint{X: e.X + 2, Y: e.Y + 2}
			free := true
			for _, at := range []sim.CellPoint{own, first, corner} {
				free = free && grid[int(at.Y*b.Width+at.X)]&1 == 0 && !sackFeetOccupied(r.live.world, at, r.hero) && !sackFeetStructure(r.live.world, at)
			}
			if !free {
				continue
			}
			if err := r.live.world.HeadlessPlace(r.hero, x, y); err != nil {
				t.Fatal(err)
			}
			r.live.tick()
			r.centre(t)
			r.drag(t, r.groundPixel(t, first), first, first)
			r.drag(t, r.groundPixel(t, corner), corner, corner)
			r.saveContinuation(t)
			return
		}
	}
	t.Fatal("no installed open 1,1 and 2,2 drop controls")
}
