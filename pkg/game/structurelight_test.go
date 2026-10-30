package game

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// lightClass is one REG-125 row and the stamp byte TERR-LIGHT-287 executes
// for it at ticks 0, 1, ... over one period.
type lightClass struct {
	name          string
	kind          uint16
	w, h          int
	radius, pulse int
	bytes         []uint8
}

var towerBytes = []uint8{28, 29, 30, 31, 32, 33, 34, 35, 36, 35, 34, 33, 32, 31, 30, 29}

var shippedLightClasses = []lightClass{
	{"Tower 1", 12, 2, 2, 1, 8, towerBytes},
	{"Tower 2", 13, 2, 2, 1, 8, towerBytes},
	{"Well 2", 15, 2, 2, 1, 0, []uint8{32}},
	{"Well 3", 16, 3, 3, 1, 4, []uint8{30, 31, 32, 33, 34, 33, 32, 31}},
	{"Campfire", 53, 1, 1, 1, 6, []uint8{29, 30, 31, 32, 33, 34, 35, 34, 33, 32, 31, 30}},
}

const unlitKind = 20

func structureLightWorld(t *testing.T, structures []sim.Structure) *mapWorld {
	t.Helper()
	return structureLightWorldWith(t, structures, nil)
}

func structureLightWorldWith(t *testing.T, structures []sim.Structure, edit func(*terrain.StructureSet)) *mapWorld {
	t.Helper()
	w, err := sim.NewStructuredWorld(1, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, sim.Terrain{},
		nil, nil, sim.Relations{}, nil, nil,
		[]sim.SpellRule{{ID: 1, School: 1, MaxRange: 7, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}},
		sim.GhostTemplate{}, structures)
	if err != nil {
		t.Fatalf("NewStructuredWorld: %v", err)
	}
	set := &terrain.StructureSet{}
	for _, c := range shippedLightClasses {
		set.Classes[c.kind] = &terrain.StructureClass{ID: int32(c.kind), TileWidth: c.w, TileHeight: c.h, FullHeight: c.h,
			LightRadius: c.radius, LightPulse: c.pulse}
	}
	set.Classes[unlitKind] = &terrain.StructureClass{ID: unlitKind, TileWidth: 2, TileHeight: 2, FullHeight: 2}
	if edit != nil {
		edit(set)
	}
	v, err := ui.NewViewerWithStatics("structure light", terrain.Grid{Width: 16, Height: 16, Tiles: make([]uint16, 256)},
		&terrain.Tileset{}, nil, false, false, false, set, false)
	if err != nil {
		t.Fatalf("NewViewerWithStatics: %v", err)
	}
	return &mapWorld{world: w, view: v, swing: map[sim.EntityID]int{},
		phase: map[sim.EntityID]sim.AttackPhase{}, castRun: map[sim.EntityID]castRun{}, projectiles: sbSheets()}
}

func setLightTick(t *testing.T, mw *mapWorld, tick int) {
	t.Helper()
	if err := mw.view.RestoreAnimation(ui.AnimationState{Count: uint32(tick)}); err != nil {
		t.Fatalf("RestoreAnimation(%d): %v", tick, err)
	}
}

func litStructure(id sim.StructureID, kind uint16, col, row int32, w, h int) sim.Structure {
	return sim.Structure{ID: id, Kind: kind, Col: col, Row: row, Width: uint8(w), Height: uint8(h), Field42: 300, MaxHealth: 300}
}

// TERR-LIGHT-286, TERR-LIGHT-287: each shipped light class stamps the
// (TileWidth+1)x(TileHeight+1) corner vertices of its rectangle (R = 1) with
// the triangle wave of the viewer's tick counter, two full periods.
func TestEachLightClassStampsItsCornerVerticesOnTheTickWave(t *testing.T) {
	t.Parallel()
	for _, c := range shippedLightClasses {
		s := litStructure(1, c.kind, 3, 4, c.w, c.h)
		if c.kind == 53 {
			s.Field42, s.MaxHealth = 0, 0
		}
		mw := structureLightWorld(t, []sim.Structure{s})
		before := mw.world.Hash()
		period := len(c.bytes)
		for tick := 0; tick < 2*period; tick++ {
			setLightTick(t, mw, tick)
			stamps := mw.objectLightStamps(nil)
			grid := lightGrid(stamps)
			want := (c.w + 1) * (c.h + 1)
			if len(stamps) != want || len(grid) != want {
				t.Fatalf("%s tick %d: %d stamps on %d vertices, want %d", c.name, tick, len(stamps), len(grid), want)
			}
			for y := 4; y <= 4+c.h; y++ {
				for x := 3; x <= 3+c.w; x++ {
					if got, ok := grid[image.Pt(x, y)]; !ok || got != c.bytes[tick%period] {
						t.Fatalf("%s tick %d: vertex (%d,%d) = %d/%v, want %d", c.name, tick, x, y, got, ok, c.bytes[tick%period])
					}
				}
			}
			for _, st := range stamps {
				if !st.Point {
					t.Fatalf("%s tick %d: stamp %+v is not skipped with Dynamic lighting off", c.name, tick, st)
				}
			}
		}
		if mw.world.Hash() != before {
			t.Fatalf("%s: building the structure stamps changed the world hash", c.name)
		}
	}
}

// TERR-LIGHT-286: R moves the block by R-1 and does not widen it.
func TestLightRadiusMovesTheStampBlock(t *testing.T) {
	t.Parallel()
	mw := structureLightWorldWith(t, []sim.Structure{litStructure(1, 12, 3, 4, 2, 2)},
		func(set *terrain.StructureSet) { set.Classes[12].LightRadius = 3 })
	grid := lightGrid(mw.objectLightStamps(nil))
	if len(grid) != 9 {
		t.Fatalf("R=3 stamped %d vertices, want 9", len(grid))
	}
	for y := 6; y <= 8; y++ {
		for x := 5; x <= 7; x++ {
			if grid[image.Pt(x, y)] != 28 {
				t.Fatalf("R=3 vertex (%d,%d) = %d, want 28; grid %v", x, y, grid[image.Pt(x, y)], grid)
			}
		}
	}
}

// TERR-LIGHT-285: LightRadius 0 and a health word that is not positive stamp
// nothing; a Campfire's maximum of 0 gives health word 1000 (TERR-STRUCT-210).
func TestStructureLightGates(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		s      sim.Structure
		stamps int
	}{
		{"LightRadius 0", litStructure(1, unlitKind, 3, 4, 2, 2), 0},
		{"destroyed Tower", sim.Structure{ID: 1, Kind: 12, Col: 3, Row: 4, Width: 2, Height: 2, Field42: 0, MaxHealth: 300}, 0},
		{"health -5", sim.Structure{ID: 1, Kind: 12, Col: 3, Row: 4, Width: 2, Height: 2, Field42: 0xfffb, MaxHealth: 300}, 0},
		{"health 1", sim.Structure{ID: 1, Kind: 12, Col: 3, Row: 4, Width: 2, Height: 2, Field42: 1, MaxHealth: 300}, 9},
		{"Campfire, maximum 0", sim.Structure{ID: 1, Kind: 53, Col: 3, Row: 4, Width: 1, Height: 1}, 4},
	} {
		mw := structureLightWorld(t, []sim.Structure{tc.s})
		if got := len(mw.objectLightStamps(nil)); got != tc.stamps {
			t.Errorf("%s: %d stamps, want %d", tc.name, got, tc.stamps)
		}
	}
}

// TERR-LIGHT-287: every structure reads one counter, so a class is in phase.
func TestLightStructuresOfAClassPulseInPhase(t *testing.T) {
	t.Parallel()
	mw := structureLightWorld(t, []sim.Structure{
		litStructure(1, 12, 1, 1, 2, 2), litStructure(2, 12, 9, 9, 2, 2), litStructure(3, 13, 1, 9, 2, 2)})
	for tick := 0; tick < 16; tick++ {
		setLightTick(t, mw, tick)
		stamps := mw.objectLightStamps(nil)
		if len(stamps) != 27 {
			t.Fatalf("tick %d: %d stamps, want 27", tick, len(stamps))
		}
		for _, s := range stamps {
			if s.Level != towerBytes[tick] {
				t.Fatalf("tick %d: stamp %+v, want every Tower at %d", tick, s, towerBytes[tick])
			}
		}
	}
}

// TERR-LIGHT-288: the structure walk runs before the projectile walk, so a
// projectile stamp at a vertex a structure also stamps stands.
func TestAProjectileStampOverAStructureStampStands(t *testing.T) {
	t.Parallel()
	mw := structureLightWorld(t, []sim.Structure{litStructure(1, 12, 1, 0, 2, 2)})
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: 1, FromX: 1, FromY: 1, ToX: 10, ToY: 1}})
	records := flightRecords(mw)
	if len(records) != 1 {
		t.Fatalf("the cast built %d records, want 1", len(records))
	}
	cell := image.Pt(floorDiv(int(records[0].X), ui.ShotScale), floorDiv(int(records[0].Y), ui.ShotScale))
	arrow := map[image.Point]bool{}
	for _, c := range []image.Point{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		arrow[cell.Add(c)] = true
	}
	grid := lightGrid(mw.objectLightStamps(nil))
	shared := 0
	for y := 0; y <= 2; y++ {
		for x := 1; x <= 3; x++ {
			v := image.Pt(x, y)
			want := uint8(28)
			if arrow[v] {
				want, shared = 16, shared+1
			}
			if grid[v] != want {
				t.Errorf("vertex %v = %d, want %d (arrow cell %v)", v, grid[v], want, cell)
			}
		}
	}
	if shared == 0 {
		t.Fatalf("the arrow at cell %v shares no vertex with the Tower", cell)
	}
}
