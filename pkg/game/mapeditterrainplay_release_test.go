package game

import (
	"image"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// guestFrontWorld opens the guest mission of the mod in dir and returns the
// front end with its mission running.
func guestFrontWorld(t *testing.T, dir string) *FrontEnd {
	t.Helper()
	f := guestFront(t, dir, guestModID)
	app := f.App("painted guest mission")
	party := f.ChargenParty(ui.ChargenResult{Name: "Guest", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}})
	if err := app.OpenMission(f.MissionOpenerWith(guestMissionNo, party)); err != nil {
		t.Fatal(err)
	}
	return f
}

// pointerStroke drags the pointer over cells in the window, one press and one
// release: the stroke the author makes with the brush.
func pointerStroke(e *ui.MapEditor, cells []image.Point) {
	now := time.Unix(1, 0)
	for _, c := range cells {
		p := e.CellPoint(c)
		e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, now)
	}
	p := e.CellPoint(cells[len(cells)-1])
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, now)
}

// brushWindow puts the window in a brush mode with the radius at zero and the
// list on the row for class, and returns that row.
func brushWindow(t *testing.T, e *ui.MapEditor, mode string, row int) {
	t.Helper()
	if err := e.Key(mode); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		_ = e.Key("radiusdown")
	}
	for i := 0; i < row; i++ {
		if err := e.Key("down"); err != nil {
			t.Fatal(err)
		}
	}
}

func paletteRow(t *testing.T, class uint8) int {
	t.Helper()
	for i, p := range mapedit.Palette {
		if p.Class == class {
			return i
		}
	}
	t.Fatalf("no palette entry for class %d", class)
	return 0
}

// walkRoute is a straight stretch of ground: cell (i, j) lies i cells along dir
// and j cells across it from start.
type walkRoute struct{ start, dir, perp image.Point }

func (r walkRoute) at(i, j int) image.Point {
	return r.start.Add(image.Pt(r.dir.X*i+r.perp.X*j, r.dir.Y*i+r.perp.Y*j))
}

// across is how far a cell lies from the walking line.
func (r walkRoute) across(c image.Point) int {
	d := c.Sub(r.start)
	n := d.X*r.perp.X + d.Y*r.perp.Y
	if n < 0 {
		return -n
	}
	return n
}

// openRoute finds the open ground nearest to the party leader's start, by
// walking distance, where a 10-cell line and the 11-cell row across its middle
// are open in the unedited map, clear of the leader's own start: the route
// starts at that cell. Scattered obstacles elsewhere are left to the pathfinder.
func openRoute(t *testing.T, m *alm.Map, start image.Point) walkRoute {
	t.Helper()
	block := mapload.Passability(m)
	open := func(c image.Point) bool {
		return c.X >= 0 && c.Y >= 0 && c.X < m.Width && c.Y < m.Height && block[c.Y*m.Width+c.X] == 0
	}
	seen := map[image.Point]bool{start: true}
	queue := []image.Point{start}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for _, dir := range []image.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
			r := walkRoute{start: c, dir: dir, perp: image.Pt(-dir.Y, dir.X)}
			ok := true
			free := func(i, j int) bool {
				near := r.at(i, j).Sub(start)
				return open(r.at(i, j)) && (abs(near.X) > 2 || abs(near.Y) > 2)
			}
			for i := 0; i <= 9 && ok; i++ {
				ok = free(i, 0)
			}
			for j := -5; j <= 5 && ok; j++ {
				ok = free(4, j)
			}
			if ok {
				return r
			}
		}
		for _, d := range []image.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
			if n := c.Add(d); open(n) && !seen[n] {
				seen[n] = true
				queue = append(queue, n)
			}
		}
	}
	t.Fatalf("no open stretch is reachable from the party start %v of the guest map (%d reachable cells)", start, len(seen))
	return walkRoute{}
}

// walk orders the leader to the goal and returns the cells it stands on, in
// order, until it arrives or 1500 ticks pass.
func walk(f *FrontEnd, id sim.EntityID, goal image.Point) []image.Point {
	w := f.live.world
	sim.Step(w, []sim.Command{sim.MoveTo(id, sim.CellPoint{X: int32(goal.X), Y: int32(goal.Y)})})
	var trail []image.Point
	for tick := 0; tick < 1500; tick++ {
		for _, e := range w.Entities() {
			if e.ID == id {
				c := image.Pt(int(e.X), int(e.Y))
				if len(trail) == 0 || trail[len(trail)-1] != c {
					trail = append(trail, c)
				}
			}
		}
		if len(trail) > 0 && trail[len(trail)-1] == goal {
			break
		}
		sim.Step(w, nil)
	}
	return trail
}

func leaderOf(t *testing.T, f *FrontEnd) (sim.EntityID, image.Point) {
	t.Helper()
	id := mapload.PartyEntity(f.live.mission.state.Map, 0)
	for _, e := range f.live.world.Entities() {
		if e.ID == id {
			return id, image.Pt(int(e.X), int(e.Y))
		}
	}
	t.Fatal("the party leader is not in the world")
	return 0, image.Point{}
}

func tileClassAt(m *alm.Map, c image.Point) uint8 {
	return mapload.TileClass(m.Tiles[c.Y*m.Width+c.X])
}

// A band of mountain painted across the party's way is walked around, and
// grass painted over it is walked across.
func TestReleaseMapEditorPaintedTerrainChangesWhereAPartyWalks(t *testing.T) {
	x := installedInspector(t)
	plain := guestMods(t, guestModID)
	saveEditedGuestMap(t, x, plain, func(*ui.MapEditor) {})
	f := guestFrontWorld(t, plain)
	leader, partyStart := leaderOf(t, f)
	plainMap := f.live.mission.state.Map
	route := openRoute(t, plainMap, partyStart)
	approach := func(f *FrontEnd, id sim.EntityID) {
		t.Helper()
		if trail := walk(f, id, route.start); len(trail) == 0 || trail[len(trail)-1] != route.start {
			t.Fatalf("the leader never reaches the stretch's start %v from %v: %v", route.start, partyStart, trail)
		}
	}
	goal := route.at(8, 0)
	var band []image.Point
	bandCell := map[image.Point]bool{}
	for j := -3; j <= 3; j++ {
		band = append(band, route.at(4, j))
		bandCell[route.at(4, j)] = true
	}
	crosses := func(trail []image.Point) bool {
		for _, c := range trail {
			if bandCell[c] {
				return true
			}
		}
		return false
	}
	approach(f, leader)
	plainTrail := walk(f, leader, goal)
	if len(plainTrail) == 0 || plainTrail[len(plainTrail)-1] != goal || !crosses(plainTrail) {
		t.Fatalf("loss control: the unedited walk to %v does not run through the band %v: %v", goal, band, plainTrail)
	}

	mountainDir := guestMods(t, guestModID)
	saveEditedGuestMap(t, x, mountainDir, func(e *ui.MapEditor) {
		e.ShowCell(route.at(4, 0))
		brushWindow(t, e, "terrain", paletteRow(t, mapedit.ClassMountain))
		pointerStroke(e, band)
		if !e.Dirty() {
			t.Fatal("the stroke left the document clean")
		}
	})
	fm := guestFrontWorld(t, mountainDir)
	mountainMap := fm.live.mission.state.Map
	blocked := mapload.Passability(mountainMap)
	for _, c := range band {
		if got := tileClassAt(mountainMap, c); got != mapedit.ClassMountain {
			t.Fatalf("band cell %v has class %d after the stroke, want Mountain", c, got)
		}
		if blocked[c.Y*mountainMap.Width+c.X]&1 == 0 {
			t.Fatalf("band cell %v is open ground after the stroke", c)
		}
	}
	if !reflect.DeepEqual(plainMap.Altitudes, mountainMap.Altitudes) || !reflect.DeepEqual(plainMap.Overlay, mountainMap.Overlay) || !reflect.DeepEqual(plainMap.Units, mountainMap.Units) {
		t.Fatal("a terrain stroke changed another plane or the units")
	}
	// The stroke rewrites the band and, ring by ring, the cells the classes
	// it inserts reach: every rewritten cell joins the band through
	// rewritten cells.
	w := plainMap.Width
	rewritten := map[image.Point]bool{}
	for i := range plainMap.Tiles {
		if plainMap.Tiles[i] != mountainMap.Tiles[i] {
			rewritten[image.Pt(i%w, i/w)] = true
		}
	}
	changed := len(rewritten)
	reached := map[image.Point]bool{}
	queue := append([]image.Point(nil), band...)
	for _, c := range band {
		reached[c] = true
	}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if n := c.Add(image.Pt(dx, dy)); rewritten[n] && !reached[n] {
					reached[n] = true
					queue = append(queue, n)
				}
			}
		}
	}
	for c := range rewritten {
		if !reached[c] {
			t.Fatalf("tile %v changed away from the stroke", c)
		}
	}
	t.Logf("a 7-cell mountain band rewrote %d tile words", changed)
	id, from := leaderOf(t, fm)
	if from != partyStart {
		t.Fatalf("the leader starts at %v on the painted map, %v on the plain one", from, partyStart)
	}
	approach(fm, id)
	trail := walk(fm, id, goal)
	if len(trail) == 0 || trail[len(trail)-1] != goal {
		t.Fatalf("the leader never reaches %v around the painted mountains: %v", goal, trail)
	}
	around := false
	for _, c := range trail {
		if bandCell[c] {
			t.Fatalf("the leader stands on painted mountain %v", c)
		}
		around = around || route.across(c) >= 4
	}
	if !around || len(trail) <= len(plainTrail) {
		t.Fatalf("the leader did not walk around the band: %d cells against the straight %d: %v", len(trail), len(plainTrail), trail)
	}

	// Paint grass back over the band in the same window session.
	grassDir := guestMods(t, guestModID)
	saveEditedGuestMap(t, x, grassDir, func(e *ui.MapEditor) {
		e.ShowCell(route.at(4, 0))
		brushWindow(t, e, "terrain", paletteRow(t, mapedit.ClassMountain))
		pointerStroke(e, band)
		back := paletteRow(t, mapedit.ClassMountain) - paletteRow(t, mapedit.ClassGrass)
		for i := 0; i < back; i++ {
			_ = e.Key("up")
		}
		pointerStroke(e, band)
	})
	fg := guestFrontWorld(t, grassDir)
	gm := fg.live.mission.state.Map
	for _, c := range band {
		if got := tileClassAt(gm, c); got != mapedit.ClassGrass {
			t.Fatalf("band cell %v has class %d after the grass stroke", c, got)
		}
	}
	gid, _ := leaderOf(t, fg)
	approach(fg, gid)
	gtrail := walk(fg, gid, goal)
	if len(gtrail) == 0 || gtrail[len(gtrail)-1] != goal || !crosses(gtrail) {
		t.Fatalf("the leader does not cross the grass band: %v", gtrail)
	}
}

// A height stroke raises the saved height plane by the stated amount and the
// mission build carries it into the height plane the simulation reads.
func TestReleaseMapEditorPaintedHeightsReachTheHeightPlane(t *testing.T) {
	x := installedInspector(t)
	plain := guestMods(t, guestModID)
	saveEditedGuestMap(t, x, plain, func(*ui.MapEditor) {})
	f := guestFrontWorld(t, plain)
	_, start := leaderOf(t, f)
	plainMap := f.live.mission.state.Map
	centre := start.Add(image.Pt(0, 0))
	for centre.Y+3 < plainMap.Height && plainMap.Altitudes[centre.Y*plainMap.Width+centre.X] > 90 {
		centre.Y++
	}
	before := plainMap.Altitudes[centre.Y*plainMap.Width+centre.X]

	raised := guestMods(t, guestModID)
	saveEditedGuestMap(t, x, raised, func(e *ui.MapEditor) {
		e.ShowCell(centre)
		brushWindow(t, e, "height", 0)
		pointerStroke(e, []image.Point{centre})
	})
	fr := guestFrontWorld(t, raised)
	rm := fr.live.mission.state.Map
	after := rm.Altitudes[centre.Y*rm.Width+centre.X]
	if int(after) != int(before)+8 {
		t.Fatalf("height at %v is %d after a raise of 8 from %d", centre, after, before)
	}
	if got := mapload.Height(rm)[centre.Y*rm.Width+centre.X]; got != after {
		t.Fatalf("the simulation height plane holds %d at %v, the saved map %d", got, centre, after)
	}
	if !reflect.DeepEqual(plainMap.Tiles, rm.Tiles) || !reflect.DeepEqual(plainMap.Overlay, rm.Overlay) {
		t.Fatal("a height stroke changed the tiles or the overlay")
	}
	for i, h := range plainMap.Altitudes {
		if h != rm.Altitudes[i] && i != centre.Y*rm.Width+centre.X {
			t.Fatalf("a radius-0 height stroke changed height %d", i)
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
