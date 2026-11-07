package game

import (
	"image"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// secondBridge is one bridge record of a second-game map: its record index,
// its anchor cell and the extension's extents.
type secondBridge struct {
	index         int
	col, row      int
	width, height int
}

// secondMapBridges is every record of m that carries an author-sized
// rectangle.
func secondMapBridges(m *alm.Map) []secondBridge {
	var out []secondBridge
	for i, o := range m.Objects {
		if w, h, ok := o.VariableExtent(); ok {
			out = append(out, secondBridge{index: i, col: int(o.X >> 8), row: int(o.Y >> 8), width: w, height: h})
		}
	}
	return out
}

// secondBridgeEnds is the cell beyond each short end of the deck, in the
// middle of its width.
func secondBridgeEnds(b secondBridge) (image.Point, image.Point) {
	if b.height > b.width {
		return image.Pt(b.col+b.width/2, b.row-1), image.Pt(b.col+b.width/2, b.row+b.height)
	}
	return image.Pt(b.col-1, b.row+b.height/2), image.Pt(b.col+b.width, b.row+b.height/2)
}

// secondGroundRoute reports whether a ground mover walks from one cell to the
// other on block, 8-connected, with the cells of closed shut.
func secondGroundRoute(block []byte, width, height int, from, to image.Point, closed image.Rectangle) bool {
	shut := func(p image.Point) bool {
		return p.X < 0 || p.Y < 0 || p.X >= width || p.Y >= height ||
			block[p.Y*width+p.X]&1 != 0 || p.In(closed)
	}
	if shut(from) || shut(to) {
		return false
	}
	seen := map[image.Point]bool{from: true}
	queue := []image.Point{from}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if p == to {
			return true
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				n := p.Add(image.Pt(dx, dy))
				if !seen[n] && !shut(n) {
					seen[n] = true
					queue = append(queue, n)
				}
			}
		}
	}
	return false
}

// checkSecondBridges: the live world holds each bridge with its author-sized
// rectangle, every deck cell open, the viewer draws a frame on every deck
// cell, and a ground route crosses each bridge that is the only way across.
// It returns how many bridges carry a route.
func checkSecondBridges(t *testing.T, stage string, f *FrontEnd, bridges []secondBridge) int {
	t.Helper()
	m := f.live.mission.state.Map
	w := f.live.world
	block := w.CurrentPolicy().Terrain.Block
	routes := 0
	for _, b := range bridges {
		var st sim.Structure
		found := false
		for _, s := range w.Structures() {
			if int(s.Col) == b.col && int(s.Row) == b.row && s.Kind == 33 {
				st, found = s, true
			}
		}
		if !found || int(st.Width) != b.width || int(st.Height) != b.height || st.Attach != ^uint32(0) || st.Blocking != 0 {
			t.Fatalf("%s: bridge at (%d,%d) is %+v (found %v), want %dx%d, all attached, none blocking",
				stage, b.col, b.row, st, found, b.width, b.height)
		}
		deck := image.Rect(b.col, b.row, b.col+b.width, b.row+b.height)
		for y := deck.Min.Y; y < deck.Max.Y; y++ {
			for x := deck.Min.X; x < deck.Max.X; x++ {
				if cell := block[y*m.Width+x]; cell&1 != 0 {
					t.Fatalf("%s: bridge deck (%d,%d) is closed with %#x", stage, x, y, cell)
				}
			}
		}
		if entries, ruined := f.live.view.StructureRuinFrames(uint32(st.ID)); entries != b.width*b.height || ruined != 0 {
			t.Fatalf("%s: bridge at (%d,%d) draws %d entries (%d ruined), want %d", stage, b.col, b.row, entries, ruined, b.width*b.height)
		}
		from, to := secondBridgeEnds(b)
		if secondGroundRoute(block, m.Width, m.Height, from, to, deck) {
			continue
		}
		if secondGroundRoute(block, m.Width, m.Height, from, to, image.Rectangle{}) {
			routes++
		}
	}
	t.Logf("%s: %d of %d bridges carry the only ground route across", stage, routes, len(bridges))
	return routes
}

// secondBridgeUnitRoutes orders the party's first member to the far end of
// each bridge and counts the bridges whose stored route steps on the deck.
// The order advances the world one tick per bridge.
func secondBridgeUnitRoutes(t *testing.T, stage string, f *FrontEnd, bridges []secondBridge) int {
	t.Helper()
	w := f.live.world
	hero := f.live.mission.ids[0]
	crossed := 0
	for _, b := range bridges {
		deck := image.Rect(b.col, b.row, b.col+b.width, b.row+b.height)
		_, to := secondBridgeEnds(b)
		sim.Step(w, []sim.Command{sim.MoveTo(hero, sim.CellPoint{X: int32(to.X), Y: int32(to.Y)})})
		route := w.Route(hero)
		on := 0
		for _, c := range route {
			if image.Pt(int(c[0]), int(c[1])).In(deck) {
				on++
			}
		}
		t.Logf("%s: route to (%d,%d) has %d cells, %d on the deck at (%d,%d)", stage, to.X, to.Y, len(route), on, b.col, b.row)
		if on > 0 {
			crossed++
		}
	}
	return crossed
}

// checkSecondCrypt: the map's Sclep, a structure without maximum health,
// draws its intact art.
func checkSecondCrypt(t *testing.T, stage string, f *FrontEnd) {
	t.Helper()
	for _, s := range f.live.world.Structures() {
		if s.Kind != 133 {
			continue
		}
		if s.MaxHealth != 0 {
			t.Fatalf("%s: Sclep 1 holds maximum health %d, want 0", stage, s.MaxHealth)
		}
		if entries, ruined := f.live.view.StructureRuinFrames(uint32(s.ID)); entries == 0 || ruined != 0 {
			t.Fatalf("%s: Sclep 1 draws %d entries, %d ruined, want intact", stage, entries, ruined)
		}
		return
	}
	t.Fatalf("%s: no Sclep 1 on the map", stage)
}

// checkSecondBridgeSAV: the SAV holds each bridge Building with its
// rectangle and masks, and no deck cell of its block records is closed.
func checkSecondBridgeSAV(t *testing.T, raw []byte, bridges []secondBridge) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	buildings, present, err := file.Buildings()
	if err != nil || !present {
		t.Fatal("SAV Buildings", present, err)
	}
	for _, b := range bridges {
		found := false
		for _, x := range buildings {
			if int(x.Col) == b.col && int(x.Row) == b.row && x.Kind == 33 {
				found = true
				if int(x.Width) != b.width || int(x.Height) != b.height || x.Attach != ^uint32(0) || x.Blocking != 0 {
					t.Fatalf("SAV bridge at (%d,%d) is %dx%d attach %#x blocking %#x, want %dx%d", b.col, b.row,
						x.Width, x.Height, x.Attach, x.Blocking, b.width, b.height)
				}
			}
		}
		if !found {
			t.Fatalf("SAV holds no bridge Building at (%d,%d)", b.col, b.row)
		}
	}
	if file.World == nil {
		t.Fatal("mission SAV has no world")
	}
	for _, blk := range file.World.Blocks {
		for _, b := range bridges {
			if p := image.Pt(blk.Col(), blk.Row()); p.In(image.Rect(b.col, b.row, b.col+b.width, b.row+b.height)) && blk.Static&1 != 0 {
				t.Fatalf("SAV bridge deck (%d,%d) is closed with static %#x", p.X, p.Y, blk.Static)
			}
		}
	}
}

// TestReleaseSecondGameMissionBridgesSurviveSAVAndLoad builds the second
// game's mission 42 from the install, whose three bridges are records of
// kind 0x21 (0x1000021 where the install flags the extension), and checks
// each bridge, its deck and a route across it, then the SAV, then a LOAD of
// that SAV in a new front end.
func TestReleaseSecondGameMissionBridgesSurviveSAVAndLoad(t *testing.T) {
	f := secondGameFront(t)
	f.Options = OptionsStore{}
	app := f.App("second-game bridges")
	app.Layout(1024, 768)
	c := newSecondCampaign()
	c.bank[769] = 1
	c.available = append(c.available, partyPlace{kind: placeMission, id: 42})
	if f.Town == nil {
		f.Town = NewTown(Campaign{})
	}
	f.Town.second = c
	f.Carried = MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table)
	screen := f.TownScreen().(*townScreen)
	app.SetTown(screen)
	destinationTownCampaign{screen}.leave()
	if err := app.OpenMission(f.MissionOpener(42)); err != nil || f.live == nil {
		t.Fatalf("mission 42: %v %s", err, app.HeadlessMessage())
	}
	bridges := secondMapBridges(f.live.mission.state.Map)
	if len(bridges) != 3 {
		t.Fatalf("mission 42 holds %d bridge records, want 3", len(bridges))
	}
	for _, b := range bridges {
		kind := f.live.mission.state.Map.Objects[b.index].Kind
		if uint16(kind) != 0x21 {
			t.Fatalf("bridge record %d has kind %#x, want low word 0x21", b.index, kind)
		}
		t.Logf("bridge record %d: kind %#x, %dx%d at (%d,%d)", b.index, kind, b.width, b.height, b.col, b.row)
	}
	if routes := checkSecondBridges(t, "fresh", f, bridges); routes == 0 {
		t.Fatal("fresh: no bridge carries the only ground route across its water")
	}
	checkSecondCrypt(t, "fresh", f)
	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	checkSecondBridgeSAV(t, raw, bridges)
	if secondBridgeUnitRoutes(t, "fresh", f, bridges) == 0 {
		t.Fatal("fresh: no unit route steps on a bridge deck")
	}

	cold := secondGameFront(t)
	cold.Options = OptionsStore{}
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("LOAD town=%t: %v", town, err)
	}
	if err := cold.App("second-game bridges LOAD").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if routes := checkSecondBridges(t, "LOAD", cold, bridges); routes == 0 {
		t.Fatal("LOAD: no bridge carries the only ground route across its water")
	}
	checkSecondCrypt(t, "LOAD", cold)
	if secondBridgeUnitRoutes(t, "LOAD", cold, bridges) == 0 {
		t.Fatal("LOAD: no unit route steps on a bridge deck")
	}
}
