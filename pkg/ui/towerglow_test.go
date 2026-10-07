package ui

// Structure light masks (owner report 2, hotfix DIV-1313 amended): which
// currently placed structures qualify as a light source, the footprint they
// dilate their radius from, and the pulse's own bounded, periodic math. All
// exercised as plain Go, with no window and no GPU — this package's own
// standing discipline (composeMinimap's opening note).

import (
	"image"
	"math"
	"testing"

	"againrom/pkg/render/terrain"
)

// towerClass, steadyGlowClass and plainClass mirror structureBundle's own
// shape (structures_test.go): a hand-built class with just enough to be
// drawable. towerClass carries the shipped values of IDs 12/13 "Tower 1" and
// "Tower 2"; steadyGlowClass carries ID 15 "Well 2", the shipped class that
// glows with no pulse at all.
func towerClass() *terrain.StructureClass {
	return &terrain.StructureClass{TileWidth: 1, TileHeight: 1, FullHeight: 1,
		Frames: []*terrain.StaticFrame{structureFrame()}, LightRadius: 1, LightPulse: 8}
}

func steadyGlowClass() *terrain.StructureClass {
	return &terrain.StructureClass{TileWidth: 1, TileHeight: 1, FullHeight: 1,
		Frames: []*terrain.StaticFrame{structureFrame()}, LightRadius: 1, LightPulse: 0}
}

func plainClass() *terrain.StructureClass {
	return &terrain.StructureClass{TileWidth: 1, TileHeight: 1, FullHeight: 1,
		Frames: []*terrain.StaticFrame{structureFrame()}}
}

// wideTowerClass is a 2x2 footprint carrying the same shipped LightRadius 1
// as towerClass — both shipped tower classes (IDs 12/13) really are 2x2 — the
// one class this file exercises the coordinator's own edge-dilation
// correction against: a 1x1 class cannot tell "radius from the anchor cell"
// apart from "radius from the footprint's edge", since the two readings
// coincide on it.
func wideTowerClass() *terrain.StructureClass {
	return &terrain.StructureClass{TileWidth: 2, TileHeight: 2, FullHeight: 1,
		Frames: []*terrain.StaticFrame{structureFrame()}, LightRadius: 1, LightPulse: 8}
}

// nightSun is the SPRITE plane's own precondition, and every test below that
// reads v.structureLighting sets it. SpriteRow is clamp(Ambient>>2) and
// ShadeScale(4*row+32) runs the other way, so a HIGHER ambient byte selects a
// DARKER ramp row: 0x28 gives row 10 and a sprite ambient of 0.75, under the
// mask's own 1.2 trough, so the sprite plane engages at every pulse phase.
// Under the viewer's own default sun that ambient is 1.625 and the sprite
// plane deliberately stays empty for most of the pulse — that asymmetry is
// TestStructureLightSpriteMaskStaysOffWhenItWouldDarken's subject, and it is
// why a test about the mask's SHAPE cannot also be a test about daylight.
// The terrain plane needs no such setting: its peak (3) clears the terrain
// ambient (1.5625) at every phase.
func nightSun() terrain.Light { return terrain.Light{Ambient: 0x28} }

// TestStructureLightSourcesSelectsOnlyLightRadiusClasses is the shipped-data
// half of the selector: a structure class whose registry LightRadius is
// nonzero lights, and a class with LightRadius 0 — every other structure,
// including every non-"Tower 1"/"Tower 2" building named "tower" in
// structures.reg — does not, whatever its own name or footprint.
func TestStructureLightSourcesSelectsOnlyLightRadiusClasses(t *testing.T) {
	v, err := NewViewer("tg-select", grid(8, 8), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	tower, plain := towerClass(), plainClass()
	v.structuresFlat = []terrain.StructurePlacement{
		{StructureID: 1, Cell: image.Pt(1, 1), TopLeft: image.Pt(32, 32), Class: tower, Frame: tower.Frames[0]},
		{StructureID: 2, Cell: image.Pt(3, 3), TopLeft: image.Pt(96, 96), Class: plain, Frame: plain.Frames[0]},
	}

	sources := v.structureLightSources()
	if len(sources) != 1 {
		t.Fatalf("structureLightSources() = %d entries, want 1 (the LightRadius-0 class must not light): %+v", len(sources), sources)
	}
	if sources[0].pulse != 8 {
		t.Errorf("carried pulse = %d, want the class's own shipped 8", sources[0].pulse)
	}
	if len(sources[0].footprint) != 1 || sources[0].footprint[0] != image.Pt(1, 1) {
		t.Errorf("footprint = %v, want exactly [(1,1)]", sources[0].footprint)
	}
}

// TestStructureLightSourcesCarriesAZeroPulseClass is ID 15 "Well 2": a class
// the registry marks as a light source with LightPulse 0. It still lights —
// the selector is LightRadius — and it carries the zero that makes its mask
// steady rather than inheriting a neighbouring class's pulse.
func TestStructureLightSourcesCarriesAZeroPulseClass(t *testing.T) {
	v, err := NewViewer("tg-steady", grid(8, 8), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	steady := steadyGlowClass()
	v.structuresFlat = []terrain.StructurePlacement{
		{StructureID: 7, Cell: image.Pt(1, 1), TopLeft: image.Pt(32, 32), Class: steady, Frame: steady.Frames[0]},
	}

	sources := v.structureLightSources()
	if len(sources) != 1 {
		t.Fatalf("structureLightSources() = %d entries, want 1 (LightRadius selects, LightPulse does not): %+v", len(sources), sources)
	}
	if sources[0].pulse != 0 {
		t.Fatalf("carried pulse = %d, want 0", sources[0].pulse)
	}
	if got := towerGlowSwing(towerGlowElapsed(3, 500, 1000), sources[0].pulse); got != 0 {
		t.Errorf("swing for a zero-pulse class = %v, want exactly 0 (a steady mask)", got)
	}
}

// TestStructureLightWorksOutOfCurrentSight is the owner's own words as a
// test, and the opposite claim from this file's pre-correction version:
// "Сам объект... не должен анимироваться если он вне обзора... (освещение
// работает всегда!)" — the light itself is UNCONDITIONAL once the structure
// has been explored at least once (so it is placed at all), not gated back
// to FogVisible the way the removed ring was.
func TestStructureLightWorksOutOfCurrentSight(t *testing.T) {
	v, err := NewViewer("tg-sight", grid(8, 8), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.sun = nightSun()
	tower := towerClass()
	v.structuresFlat = []terrain.StructurePlacement{
		{StructureID: 1, Cell: image.Pt(2, 2), TopLeft: image.Pt(64, 64), Class: tower, Frame: tower.Frames[0]},
	}

	plane := make([]byte, 8*8)
	for i := range plane {
		plane[i] = FogExplored // seen once, not seen now
	}
	v.SetFog(plane, 8, 8)

	sources := v.structureLightSources()
	if len(sources) != 1 {
		t.Fatalf("structureLightSources() = %d entries over an EXPLORED-only (not currently visible) structure, want 1: lighting always works", len(sources))
	}
	v.refreshStructureLighting()
	if level, ok := v.structureLighting[image.Pt(2, 2)]; !ok || level <= 0 {
		t.Errorf("structureLighting[(2,2)] = %v/%v, want a positive level over an explored-but-not-visible structure", level, ok)
	}
}

// TestStructureLightDilatesAMultiCellFootprintFromItsEdge is the coordinator's
// own correction, made concrete: a 2x2 footprint with LightRadius 1 lights a
// 4x4 region — its own two-by-two cells plus one ring of cells all the way
// around — not a radius-1 disc about a single anchor cell, which on this
// footprint would leave the far corners unlit. Only a multi-cell class can
// tell the two readings apart (towerClass's own 1x1 cannot).
func TestStructureLightDilatesAMultiCellFootprintFromItsEdge(t *testing.T) {
	v, err := NewViewer("tg-dilate", grid(8, 8), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.sun = nightSun()
	wide := wideTowerClass()
	// Footprint anchored at (2,2): cells (2,2),(3,2),(2,3),(3,3) — one entry
	// per footprint cell, StructurePlacement's own "one image row of one
	// rectangle cell" shape (structures.go doc).
	v.structuresFlat = []terrain.StructurePlacement{
		{StructureID: 9, Cell: image.Pt(2, 2), TopLeft: image.Pt(64, 64), Class: wide, Frame: wide.Frames[0]},
		{StructureID: 9, Cell: image.Pt(3, 2), TopLeft: image.Pt(96, 64), Class: wide, Frame: wide.Frames[0]},
		{StructureID: 9, Cell: image.Pt(2, 3), TopLeft: image.Pt(64, 96), Class: wide, Frame: wide.Frames[0]},
		{StructureID: 9, Cell: image.Pt(3, 3), TopLeft: image.Pt(96, 96), Class: wide, Frame: wide.Frames[0]},
	}
	v.refreshStructureLighting()

	// THE MASK IS THE FOOTPRINT DILATED BY ONE CELL ORTHOGONALLY, NOT A 4x4
	// SQUARE. Each footprint cell stamps the cells within EUCLIDEAN distance
	// `radius` of it, so a diagonal at 1.41 falls outside a radius-1 stamp
	// (owner report R4: a square stamp reads wider than the radius it claims).
	lit := []image.Point{
		{2, 2}, {3, 2}, {2, 3}, {3, 3}, // the footprint itself
		{1, 2}, {1, 3}, {4, 2}, {4, 3}, // left and right fringe
		{2, 1}, {3, 1}, {2, 4}, {3, 4}, // top and bottom fringe
	}
	for _, p := range lit {
		if _, ok := v.structureLighting[p]; !ok {
			t.Errorf("structureLighting[%v] missing, want it lit", p)
		}
	}
	if len(v.structureLighting) != len(lit) {
		t.Errorf("structureLighting has %d cells, want exactly %d", len(v.structureLighting), len(lit))
	}
	// THE FOUR CORNERS OF THE OLD 4x4 ARE NOW UNLIT — this is the specific
	// regression the R4 report names, so it is asserted by name rather than
	// only by the count above.
	for _, p := range []image.Point{{1, 1}, {4, 4}, {1, 4}, {4, 1}} {
		if _, ok := v.structureLighting[p]; ok {
			t.Errorf("structureLighting[%v] present, want unlit (a diagonal is 1.41 cells from the footprint, outside radius 1)", p)
		}
	}
	for _, p := range []image.Point{{0, 2}, {5, 2}, {2, 0}, {2, 5}} {
		if _, ok := v.structureLighting[p]; ok {
			t.Errorf("structureLighting[%v] present, want unlit (two cells from the footprint)", p)
		}
	}
}

// TestStructureLightFallsOffWithDistance is the R4 report's own subject: the
// mask must be brightest at the source and dimmer at its edge, because a flat
// fill with a hard rim reads as a larger radius than it has.
func TestStructureLightFallsOffWithDistance(t *testing.T) {
	v, err := NewViewer("tg-falloff", grid(8, 8), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	// A two-cell radius so there is a middle ring to compare, not only an
	// edge: radius 1 has no interior sample between centre and rim.
	v.sun = nightSun()
	class := towerClass()
	class.LightRadius = 2
	v.structuresFlat = []terrain.StructurePlacement{
		{StructureID: 4, Cell: image.Pt(4, 4), TopLeft: image.Pt(128, 128), Class: class, Frame: class.Frames[0]},
	}
	v.refreshStructureLighting()

	// READ THE CELL-KEYED SPRITE PLANE, NOT THE VERTEX-KEYED TERRAIN ONE. The
	// terrain plane writes each lit cell's level to all four of its corners,
	// so the vertex one cell out from the source is still a corner OF the
	// source cell and takes the source's own value by max — the vertex plane
	// cannot resolve a one-cell step by construction, and reading it here
	// would test the corner expansion rather than the falloff.
	centre, ok := v.structureLighting[image.Pt(4, 4)]
	if !ok {
		t.Fatal("source cell is unlit")
	}
	mid, ok := v.structureLighting[image.Pt(5, 4)]
	if !ok {
		t.Fatal("cell one out from the source is unlit")
	}
	rim, ok := v.structureLighting[image.Pt(6, 4)]
	if !ok {
		t.Fatal("cell at the radius edge is unlit")
	}
	if !(centre > mid && mid > rim) {
		t.Errorf("mask does not fall off: centre %v, middle %v, rim %v; want strictly decreasing", centre, mid, rim)
	}
}

// TestStructureLightNeverDarkensACell is the guarantee the falloff must not
// break. cornerScales REPLACES a vertex's own shading with this plane's value
// and spellSpriteFactor DIVIDES by the sprite ambient, so a mask value under
// the ambient would darken the cell instead of leaving it alone. The falloff
// therefore attenuates the mask's excess over ambient, never its absolute
// level, and every written value must stay at or above that ambient.
func TestStructureLightNeverDarkensACell(t *testing.T) {
	v, err := NewViewer("tg-nodarken", grid(12, 12), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	class := towerClass()
	class.LightRadius = 4
	v.structuresFlat = []terrain.StructurePlacement{
		{StructureID: 4, Cell: image.Pt(6, 6), TopLeft: image.Pt(192, 192), Class: class, Frame: class.Frames[0]},
	}
	spriteAmbient := terrain.ShadeScale(4*v.spriteRow() + 32)
	terrainAmbient := terrain.ShadeScale(46)
	// Sweep the whole pulse, not one phase: the trough is where the mask is
	// closest to ambient and so where an underflow would first appear.
	for step := 0; step < 40; step++ {
		v.refreshStructureLighting()
		for cell, got := range v.structureLighting {
			if got < spriteAmbient {
				t.Fatalf("sprite mask at %v is %v, below the ambient %v: this DARKENS the cell", cell, got, spriteAmbient)
			}
		}
		for vertex, got := range v.structureTerrainLighting {
			if got < terrainAmbient {
				t.Fatalf("terrain mask at %v is %v, below the ambient %v: this DARKENS the vertex", vertex, got, terrainAmbient)
			}
		}
		v.anim.AdvanceOne()
	}
}

// TestStructureLightSpriteMaskStaysOffWhenItWouldDarken pins the guard the
// falloff work uncovered. Both consumers read this plane as the cell's ABSOLUTE
// level: spellSpriteFactor divides by the ambient and cornerScales replaces the
// vertex shading outright. So a mask value under the ambient dims rather than
// does nothing — and with peak 2 against a 1.625 sprite ambient that was most
// of every pulse. The plane must stay empty rather than write a dimming value.
func TestStructureLightSpriteMaskStaysOffWhenItWouldDarken(t *testing.T) {
	v, err := NewViewer("tg-nodim", grid(8, 8), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	class := towerClass()
	class.LightRadius = 1
	v.structuresFlat = []terrain.StructurePlacement{
		{StructureID: 4, Cell: image.Pt(4, 4), TopLeft: image.Pt(128, 128), Class: class, Frame: class.Frames[0]},
	}
	spriteAmbient := terrain.ShadeScale(4*v.spriteRow() + 32)
	for step := 0; step < 40; step++ {
		v.refreshStructureLighting()
		for cell, got := range v.structureLighting {
			if got <= spriteAmbient {
				t.Fatalf("sprite mask at %v wrote %v against ambient %v: it must stay absent rather than dim", cell, got, spriteAmbient)
			}
		}
		// Whatever the sprite plane does, spellSpriteFactor must never report
		// a factor under 1 for this structure's own cell.
		if got := v.spellSpriteFactor(image.Pt(4, 4)); got < 1 {
			t.Fatalf("spellSpriteFactor at the source = %v, under 1: the structure light DIMS its own cell", got)
		}
		v.anim.AdvanceOne()
	}
}

func TestStructureLightSourcesUnionsAStructuresOwnFootprint(t *testing.T) {
	v, err := NewViewer("tg-union", grid(8, 8), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	tower := towerClass()
	f := tower.Frames[0]
	v.structuresFlat = []terrain.StructurePlacement{
		{StructureID: 5, Cell: image.Pt(1, 1), TopLeft: image.Pt(32, 0), Class: tower, Frame: f},
		{StructureID: 5, Cell: image.Pt(1, 1), TopLeft: image.Pt(32, 32), Class: tower, Frame: f},
	}

	sources := v.structureLightSources()
	if len(sources) != 1 {
		t.Fatalf("structureLightSources() = %d entries, want 1: %+v", len(sources), sources)
	}
	if len(sources[0].footprint) != 1 {
		t.Fatalf("footprint = %v, want exactly one cell (deduplicated across the overhang entry)", sources[0].footprint)
	}
}

// TestStructureLightComposesWithSpellLightingByMax is DIV-1313's own
// composition rule, exercised directly: a cell already lit brighter by the
// spell plane keeps that value, and a cell the structure lights brighter than
// the spell plane takes the structure's value — the same MAX rule
// spellLightingCells uses for its own overlapping sources (pkg/game/world.go)
// applied here between the two viewer-side planes.
func TestStructureLightComposesWithSpellLightingByMax(t *testing.T) {
	v, err := NewViewer("tg-compose", grid(8, 8), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	// The structure's own sprite plane must be ENGAGED for a max between two
	// live values to be exercised at all; in daylight it is empty by design
	// and this test would only re-assert the spell plane's own value.
	v.sun = nightSun()
	tower := towerClass()
	v.structuresFlat = []terrain.StructurePlacement{
		{StructureID: 1, Cell: image.Pt(2, 2), TopLeft: image.Pt(64, 64), Class: tower, Frame: tower.Frames[0]},
	}
	// A brighter spell source on the structure's own cell: darkness's
	// terrain-ladder value (12) is numerically ABOVE any structure light can
	// reach (structureLightPeakTerrain is 3), so the max must still favour
	// the spell value here even though "brighter" and "higher" trade places
	// between the two ladders is exactly the trap DIV-1349's own hover fix
	// caught once already — this test spells out the ladder direction is
	// irrelevant to max, only magnitude is.
	v.SetSpellLighting([]SpellLightCell{{Cell: image.Pt(2, 2), Terrain: 12, Sprite: 1}})
	v.refreshStructureLighting()

	sprite := v.spellSpriteFactor(image.Pt(2, 2))
	normal := terrain.ShadeScale(4*v.spriteRow() + 32)
	// The structure's own sprite peak (2) is brighter than the spell cell's
	// own Sprite (1), so the composed factor must reflect the structure's
	// peak, not the spell's.
	if wantMin := structureLightPeakSprite * float32(structureLightFloor) / normal; sprite < wantMin {
		t.Errorf("spellSpriteFactor(2,2) = %v, want at least the structure's own floor level %v (max composition)", sprite, wantMin)
	}

	terrainVal, ok := v.spellTerrainScale(image.Pt(2, 2))
	if !ok {
		t.Fatal("spellTerrainScale(2,2) reports no override, want the spell plane's own 12 to survive composition")
	}
	if terrainVal != 12 {
		t.Errorf("spellTerrainScale(2,2) = %v, want the spell plane's own higher value 12 to win the max, not the structure's own lower peak", terrainVal)
	}
}

// TestStructureLightRespectsDisableLighting mirrors spellSpriteFactor's own
// gate (overlay.go): every lighting behaviour this hotfix batch adds must
// stay off under the diagnostic.
func TestStructureLightRespectsDisableLighting(t *testing.T) {
	v, err := NewViewer("tg-disable", grid(8, 8), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	tower := towerClass()
	v.structuresFlat = []terrain.StructurePlacement{
		{StructureID: 1, Cell: image.Pt(2, 2), TopLeft: image.Pt(64, 64), Class: tower, Frame: tower.Frames[0]},
	}
	v.SetGraphicsOptions(GraphicsOptions{DisableLighting: true})
	v.refreshStructureLighting()

	if v.structureLighting != nil || v.structureTerrainLighting != nil {
		t.Fatalf("refreshStructureLighting under DisableLighting left non-nil planes: sprite=%v terrain=%v", v.structureLighting, v.structureTerrainLighting)
	}
}

// TestTowerGlowSwingIsBoundedPeriodicAndPerClass is the pulse's own
// contract: a swing inside [-1,1], a cycle whose length is the class's own
// shipped LightPulse, and no swing at all when that value is zero — the three
// properties refreshStructureLighting assumes per structure without
// re-checking any of them.
func TestTowerGlowSwingIsBoundedPeriodicAndPerClass(t *testing.T) {
	const towerPulse, wellPulse = 8, 4
	towerCycle := float64(towerPulse) * towerGlowPulseTicksPerUnit

	for _, tc := range []struct {
		name    string
		elapsed float64
		pulse   int
	}{
		{"zero", 0, towerPulse},
		{"one full cycle", towerCycle, towerPulse},
		{"mid tick", 3.5, towerPulse},
		{"large count", 4093, towerPulse},
		{"faster class", 4093, wellPulse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := towerGlowSwing(tc.elapsed, tc.pulse)
			if got < -1 || got > 1 {
				t.Fatalf("swing %v is outside [-1,1]", got)
			}
		})
	}

	// NO PULSE MEANS NO SWING, at every instant, not merely at zero.
	for _, elapsed := range []float64{0, 0.5, 7, 4093.25} {
		if got := towerGlowSwing(elapsed, 0); got != 0 {
			t.Errorf("swing(%v, pulse 0) = %v, want exactly 0", elapsed, got)
		}
	}

	// PERIODIC ON THE CLASS'S OWN CYCLE: one whole cycle apart is the same
	// swing.
	a := towerGlowSwing(2, towerPulse)
	b := towerGlowSwing(2+towerCycle, towerPulse)
	if math.Abs(a-b) > 1e-9 {
		t.Errorf("swing(2) = %v, swing(2+%v) = %v, want equal (one full cycle apart)", a, towerCycle, b)
	}

	// THE SHIPPED ORDER SURVIVES: Well 3's LightPulse 4 reaches its own first
	// peak in half the ticks a tower's 8 needs, which is the one relation
	// this file claims to read out of the registry's numbers.
	towerPeak := towerCycle / 4
	wellPeak := float64(wellPulse) * towerGlowPulseTicksPerUnit / 4
	if math.Abs(towerGlowSwing(towerPeak, towerPulse)-1) > 1e-9 {
		t.Errorf("tower swing at its own quarter cycle = %v, want 1", towerGlowSwing(towerPeak, towerPulse))
	}
	if math.Abs(towerGlowSwing(wellPeak, wellPulse)-1) > 1e-9 {
		t.Errorf("well swing at its own quarter cycle = %v, want 1", towerGlowSwing(wellPeak, wellPulse))
	}
	if wellPeak*2 != towerPeak {
		t.Errorf("well peak %v and tower peak %v do not hold the shipped 4:8 ratio", wellPeak, towerPeak)
	}

	// MID-TICK PROGRESS MOVES THE SWING: half a tick elapsed lands strictly
	// between the tick's start and a whole tick later, on the rising quarter.
	start := towerGlowSwing(towerGlowElapsed(0, 0, 0), towerPulse)
	half := towerGlowSwing(towerGlowElapsed(0, 500, 1000), towerPulse)
	next := towerGlowSwing(towerGlowElapsed(1, 0, 0), towerPulse)
	if !(start < half && half < next) {
		t.Errorf("swing did not advance across a tick: start=%v half=%v next=%v", start, half, next)
	}
}

// TestStructureLightLevelStaysWithinFloorAndPeak is structureLightLevel's own
// contract: never below the floor fraction of its peak (so "работает всегда"
// never reads as fully dark) and never above the peak itself.
func TestStructureLightLevelStaysWithinFloorAndPeak(t *testing.T) {
	peak := structureLightPeakSprite
	floor := peak * structureLightFloor
	for _, swing := range []float64{-1, -0.5, 0, 0.5, 1} {
		got := structureLightLevel(peak, swing)
		if got < floor-1e-6 || got > peak+1e-6 {
			t.Errorf("structureLightLevel(peak, %v) = %v, want within [%v,%v]", swing, got, floor, peak)
		}
	}
	if got := structureLightLevel(peak, -1); math.Abs(float64(got-floor)) > 1e-6 {
		t.Errorf("structureLightLevel(peak, -1) = %v, want exactly the floor %v", got, floor)
	}
	if got := structureLightLevel(peak, 1); math.Abs(float64(got-peak)) > 1e-6 {
		t.Errorf("structureLightLevel(peak, 1) = %v, want exactly the peak %v", got, peak)
	}
}

// TestStructureLightNeverDimsAVertexAtTheConsumer is the other half of "a light
// only ever adds", and the half the plane itself cannot guarantee. The mask's
// falloff attenuates toward ShadeScale(46), the level FLAT ground holds under
// the cycle-off daytime sun — but relief moves a vertex either way and a swept
// LevelGrid measurement reaches 36 (scale 1.875) on the bright side, above that
// base. So the rim of a mask CAN sit under a sunlit slope's own shading, and
// nothing in refreshStructureLighting can see that: it writes one plane for the
// whole frame while cornerShading answers per tile.
//
// cornerScales therefore composes this plane by MAX, not by replacement. The
// second pass below is the discriminator: it forces a deliberately dim value
// into every vertex of the plane, which a replacing consumer would paint.
func TestStructureLightNeverDimsAVertexAtTheConsumer(t *testing.T) {
	v, err := NewViewer("tg-nodim-consumer", cliffGrid(), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	if !v.Lit() {
		t.Fatal("fixture must be lit: an unlit viewer has no relief shading to dim")
	}
	shading := make(map[image.Point][4]float32)
	for ty := 0; ty < cliffH; ty++ {
		for tx := 0; tx < cliffW; tx++ {
			shading[image.Pt(tx, ty)] = v.cornerShading(tx, ty)
		}
	}

	tower := towerClass()
	// BOTH LISTS. This fixture carries altitudes, so the viewer is in
	// ModeDisplaced and structureLists() reads structuresDisplaced —
	// structuresFlat alone would leave the plane empty and every assertion
	// below vacuous.
	places := []terrain.StructurePlacement{
		{StructureID: 1, Cell: image.Pt(1, 1), TopLeft: image.Pt(32, 32), Class: tower, Frame: tower.Frames[0]},
	}
	v.structuresFlat, v.structuresDisplaced = places, places
	v.refreshStructureLighting()

	check := func(pass string) {
		t.Helper()
		for ty := 0; ty < cliffH; ty++ {
			for tx := 0; tx < cliffW; tx++ {
				base := shading[image.Pt(tx, ty)]
				got := v.cornerScales(tx, ty)
				for i := range got {
					if got[i] < base[i] {
						t.Fatalf("%s: cell (%d,%d) corner %d = %v, under its own shading %v: the structure mask DIMMED it",
							pass, tx, ty, i, got[i], base[i])
					}
				}
			}
		}
	}
	check("real mask")

	// The mask must actually have reached something, or the pass above proves
	// nothing about composition.
	if len(v.structureTerrainLighting) == 0 {
		t.Fatal("the structure terrain plane is empty: this fixture cannot discriminate")
	}
	lifted := false
	for ty := 0; ty < cliffH && !lifted; ty++ {
		for tx := 0; tx < cliffW && !lifted; tx++ {
			base, got := shading[image.Pt(tx, ty)], v.cornerScales(tx, ty)
			for i := range got {
				if got[i] > base[i] {
					lifted = true
				}
			}
		}
	}
	if !lifted {
		t.Fatal("no corner was brightened by the mask: this fixture cannot discriminate")
	}

	// A value far under any shading this fixture holds, at every vertex the
	// mask already covers. A consumer that REPLACES paints all of them.
	for vertex := range v.structureTerrainLighting {
		v.structureTerrainLighting[vertex] = 0.01
	}
	check("forced dim value")
}
