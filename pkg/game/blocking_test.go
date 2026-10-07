package game

// What blocks what.
//
// EVERY FIXTURE IS SYNTHETIC: an in-memory .res archive holding a synthetic
// data.bin, and an .alm assembled from the format contract. No game install is
// read and no window is opened.

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/vfs"
)

// The fixture map: a square whose impassable border ring leaves an open
// interior, split in two by one blocked column.
//
// The side is chosen so the ring — eight cells deep on every edge — leaves an
// interior at all: a map of twice that or fewer on an axis is border throughout
// and has no open cell for a structure to stand on.
const (
	blockSide   = 24 // map extent, both axes
	blockOpenLo = 8  // first interior column/row
	blockOpenHi = 16 // one past the last

	// blockCut is the column the fixture closes with a nonzero overlay byte,
	// splitting the interior into a left region and a right one that are not
	// 8-connected: the two are two columns apart, so no diagonal step joins them.
	blockCut = 12

	// The two placement keys the fixture table answers. A structure key selects
	// its entry by the key's LOW BYTE, and the collection is one-based, so a key
	// IS its entry's index here.
	blockBridgeKey = 1 // a 1x8 strip that attaches every cell and closes none
	blockHutKey    = 2 // a 2x2 square that attaches every cell and closes all
)

// blockHutAt is the hut's anchor: the top-left of the 2x2 rectangle, on open
// interior ground and clear of the cut column, so the four cells it closes are
// four the terrain arms leave open.
var blockHutAt = image.Point{X: 9, Y: 9}

// blockRow builds one buildings entry. The four positions are the footprint
// contract's: the two extents, the BLOCKING set — a set bit closes its cell —
// and the ATTACH set, which names the cells of the rectangle the structure
// occupies at all.
//
// The array is written wider than the four positions read, as a shipped row is,
// so a reader that took a shorter row as "no parameters" would be caught.
func blockRow(name string, w, h, blocking, attach int32) synth.DataBinRow {
	p := make([]int32, 38)
	p[0], p[1] = w, h
	p[4], p[5] = blocking, attach
	return synth.DataBinRow{Name: name, Params: p}
}

// blockTableBytes is the fixture definition table: the two buildings entries the
// cases place, and nothing in the two collections a unit placement searches.
//
// The units collection is left EMPTY on purpose. A placement that resolves no
// entry keeps the provisional health and the constructor's rate, which is what
// gives the crossing case a unit that can actually walk; a row of zeros would
// resolve and hand it a maximum health of zero and a rate of zero, and the case
// would then fail for a reason that has nothing to do with the plane.
func blockTableBytes() []byte {
	var d synth.DataBin
	d.Rows[synth.DataBinBuildings] = []synth.DataBinRow{
		blockRow("bridge", 1, 8, 0, 0xFF),
		blockRow("hut", 2, 2, 0xF, 0xF),
	}
	return d.Bytes()
}

// blockTable loads that table the way the front-end loads its own: through
// LoadTable, out of a container filesystem over a synthetic world archive and
// the scenario archive beside it that the load also reads.
func blockTable(t *testing.T) *mapload.Table {
	t.Helper()
	dir := t.TempDir()
	for name, data := range map[string][]byte{
		WorldArchive:    synth.Archive([]synth.File{{Path: "data/data.bin", Data: blockTableBytes()}}),
		ScenarioArchive: synth.Archive([]synth.File{{Path: "npc.reg", Data: synth.NPCReg(nil)}}),
	} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	fsys, err := vfs.Open([]string{dir + "/" + WorldArchive, dir + "/" + ScenarioArchive}, nil)
	if err != nil {
		t.Fatalf("vfs.Open: %v", err)
	}
	tbl, err := LoadTable(fsys)
	if err != nil {
		t.Fatalf("LoadTable: %v", err)
	}
	return tbl
}

// blockMap assembles the fixture map with the given placements: the open
// interior, the cut column, and whatever type-4 and type-6 records a case needs.
func blockMap(t *testing.T, objects []synth.ALMObject, units []synth.ALMUnit) *alm.Map {
	t.Helper()
	overlay := make([]uint8, blockSide*blockSide)
	for y := 0; y < blockSide; y++ {
		overlay[y*blockSide+blockCut] = 1
	}
	m, err := alm.Open(synth.ALM(synth.ALMOptions{
		Width: blockSide, Height: blockSide, Overlay: overlay, Objects: objects, Units: units,
	}))
	if err != nil {
		t.Fatalf("alm.Open: %v", err)
	}
	return m
}

// blockBridge is the placement that spans the cut: anchored at its top row, one
// cell wide and eight tall, attaching every cell of that strip and closing none.
func blockBridge() synth.ALMObject {
	return synth.ALMObject{X: blockCut << 8, Y: blockOpenLo << 8, Kind: blockBridgeKey}
}

func blockHut() synth.ALMObject {
	return synth.ALMObject{X: uint32(blockHutAt.X) << 8, Y: uint32(blockHutAt.Y) << 8, Kind: blockHutKey}
}

// blockedPlane reports which cells of a plane are closed to a ground mover.
// Bit 0 is the ground bit; bit 1 is the air bit and no structure writes it.
func blockedGround(plane []byte, at image.Point) bool {
	return plane[at.Y*blockSide+at.X]&1 != 0
}

// TestLoadTableCarriesTheBuildingsCollection is AC-1.
//
// The collection is SUBSCRIPTED rather than searched, so a wrong collection id
// would resolve plausible rubbish rather than failing. The entry's own
// parameters are asserted and not merely the count, which is what makes that
// failure visible (plan R-1).
func TestLoadTableCarriesTheBuildingsCollection(t *testing.T) {
	tbl := blockTable(t)
	if tbl.Buildings == nil {
		t.Fatal("the loaded table carries no buildings collection, so no placed structure can reach a plane")
	}
	// Three slots: the reserved entry 0 and the two written rows.
	if got := tbl.Buildings.Len(); got != 3 {
		t.Fatalf("Buildings holds %d entries, want 3", got)
	}
	if got := tbl.Buildings.EntryName(blockHutKey); got != "hut" {
		t.Errorf("entry %d is named %q, want %q", blockHutKey, got, "hut")
	}
	p := tbl.Buildings.EntryParams(blockBridgeKey)
	if len(p) < 6 {
		t.Fatalf("entry %d carries %d parameters, too few for the footprint contract", blockBridgeKey, len(p))
	}
	if p[0] != 1 || p[1] != 8 || p[4] != 0 || p[5] != 0xFF {
		t.Errorf("entry %d reads (w=%d h=%d blocking=%#x attach=%#x), want (1, 8, 0x0, 0xff)",
			blockBridgeKey, p[0], p[1], p[4], p[5])
	}
}

func TestAPlacedStructureClosesWhatItSaysItCloses(t *testing.T) {
	m := blockMap(t, []synth.ALMObject{blockHut()}, nil)
	tbl := blockTable(t)

	plain := mustOpenMapWorld(t, m, nil, nil, worldFixtureViewer(t, m))
	withTable := mustOpenMapWorld(t, m, tbl, nil, worldFixtureViewer(t, m))

	for dy := 0; dy < 2; dy++ {
		for dx := 0; dx < 2; dx++ {
			at := blockHutAt.Add(image.Point{X: dx, Y: dy})
			if worldBlocks(t, plain.world, at) {
				t.Fatalf("the fixture is wrong: %v is already closed with no table, so closing it proves nothing", at)
			}
			if !worldBlocks(t, withTable.world, at) {
				t.Errorf("%v is open in the world the map screen builds; the hut's blocking set names it", at)
			}
		}
	}
	// One cell beyond the rectangle, so the case would fail a pass that closed
	// the whole map rather than the footprint.
	if beyond := blockHutAt.Add(image.Point{X: 2, Y: 0}); worldBlocks(t, withTable.world, beyond) {
		t.Errorf("%v is closed, and it lies outside the hut's 2x2 rectangle", beyond)
	}
}

func TestAPlacedStructureOpensWhatItSaysItOpens(t *testing.T) {
	m := blockMap(t, []synth.ALMObject{blockBridge()}, nil)
	tbl := blockTable(t)

	plain := mustOpenMapWorld(t, m, nil, nil, worldFixtureViewer(t, m))
	withTable := mustOpenMapWorld(t, m, tbl, nil, worldFixtureViewer(t, m))

	for y := blockOpenLo; y < blockOpenHi; y++ {
		at := image.Point{X: blockCut, Y: y}
		if !worldBlocks(t, plain.world, at) {
			t.Fatalf("the fixture is wrong: %v is already open with no table, so opening it proves nothing", at)
		}
		if worldBlocks(t, withTable.world, at) {
			t.Errorf("%v is still closed in the world the map screen builds; the bridge attaches it and closes none of it", at)
		}
	}
	// The cut column above the strip is untouched: the deck is eight cells tall
	// and the column is the map's full height, so a pass that opened the column
	// rather than the footprint fails here.
	if above := (image.Point{X: blockCut, Y: blockOpenLo - 1}); !worldBlocks(t, withTable.world, above) {
		t.Errorf("%v is open, and no footprint attaches it", above)
	}
}

func TestTheBridgeJoinsTwoRegionsTheIngestLeavesDisjoint(t *testing.T) {
	m := blockMap(t, []synth.ALMObject{blockBridge()}, nil)
	tbl := blockTable(t)

	from := image.Point{X: blockOpenLo + 1, Y: 12}
	to := image.Point{X: blockOpenHi - 2, Y: 12}

	if reaches(t, mustOpenMapWorld(t, m, nil, nil, worldFixtureViewer(t, m)).world, from, to) {
		t.Fatal("the fixture is wrong: the two regions are already joined with no table")
	}
	if !reaches(t, mustOpenMapWorld(t, m, tbl, nil, worldFixtureViewer(t, m)).world, from, to) {
		t.Errorf("%v does not reach %v across the bridge in the world the map screen builds", from, to)
	}
}

// TestAMissionsWorldTakesTheSameSecondStage is AC-5's other half.
//
// The defect looked like the map screen's, because that is the screen it was
// seen on. It was one missing collection, and the mission path — a different
// builder, reached through a different front-end door — was broken identically.
func TestAMissionsWorldTakesTheSameSecondStage(t *testing.T) {
	m := blockMap(t, []synth.ALMObject{blockBridge(), blockHut()}, nil)
	tbl := blockTable(t)

	ms, _, err := mapload.StartMission(m, tbl, openDifficulty, MissionParty(nil, nil, nil))
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	plain, _, err := mapload.StartMission(m, nil, openDifficulty, MissionParty(nil, nil, nil))
	if err != nil {
		t.Fatalf("StartMission with no table: %v", err)
	}

	deck := image.Point{X: blockCut, Y: 12}
	if !worldBlocks(t, plain, deck) || worldBlocks(t, ms, deck) {
		t.Errorf("the deck at %v is closed=%v with no table and closed=%v with one; want true then false",
			deck, worldBlocks(t, plain, deck), worldBlocks(t, ms, deck))
	}
	if worldBlocks(t, plain, blockHutAt) || !worldBlocks(t, ms, blockHutAt) {
		t.Errorf("the hut cell %v is closed=%v with no table and closed=%v with one; want false then true",
			blockHutAt, worldBlocks(t, plain, blockHutAt), worldBlocks(t, ms, blockHutAt))
	}
}

// worldBlocks reports whether a world closes a cell to a ground mover.
//
// It asks the WORLD and not a plane derived beside it, and it asks through the
// world's own canonical byte form — the one traversal the digest is taken over —
// so what it reports is the grid a player's units route on rather than a second
// reading of the rule that built it.
func worldBlocks(t *testing.T, w *sim.World, at image.Point) bool {
	t.Helper()
	return worldGrid(t, w)[at.Y*blockSide+at.X]&1 != 0
}

// worldGrid pulls the block plane out of a world's canonical byte form.
//
// The form is a fixed header, then the grid, then the entity records. The header
// carries the grid's own length at a fixed offset, so this reads that length
// rather than recomputing it from the bounds — a grid the world sized
// differently from the map would be caught here instead of read past.
func worldGrid(t *testing.T, w *sim.World) []byte {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	const gridLenOffset, headerLen = 30, 34
	n := int(b[gridLenOffset]) | int(b[gridLenOffset+1])<<8 | int(b[gridLenOffset+2])<<16 | int(b[gridLenOffset+3])<<24
	if n != blockSide*blockSide {
		t.Fatalf("the world's grid is %d cells, want %d", n, blockSide*blockSide)
	}
	return b[headerLen : headerLen+n]
}

// reaches walks 8-connected reachability over a world's own plane.
//
// The neighbourhood is the full 3x3 WITH NO CORNER RULE, which is the shape the
// two regions were separated by two columns to survive: at four neighbours they
// would be disjoint for a second reason, and the case would pass without the
// bridge.
func reaches(t *testing.T, w *sim.World, from, to image.Point) bool {
	t.Helper()
	plane := worldGrid(t, w)
	blocked := func(p image.Point) bool {
		return p.X < 0 || p.Y < 0 || p.X >= blockSide || p.Y >= blockSide ||
			plane[p.Y*blockSide+p.X]&1 != 0
	}
	if blocked(from) || blocked(to) {
		t.Fatalf("the fixture is wrong: %v or %v is itself closed", from, to)
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
				n := p.Add(image.Point{X: dx, Y: dy})
				if (dx == 0 && dy == 0) || seen[n] || blocked(n) {
					continue
				}
				seen[n] = true
				queue = append(queue, n)
			}
		}
	}
	return false
}
