package game

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Human geometry and expected deltas are MAGIC-263's.

func launchHuman(offsets []int) *terrain.UnitClass {
	return &terrain.UnitClass{CenterX: 64, CenterY: 78, TileSize: 1,
		Selection:   image.Rect(48, 48, 80, 90),
		ShootOffset: offsets}
}

var (
	launchMageOffsets  = []int{57, 75, 45, 66, 44, 53, 52, 43, 68, 42, 80, 50, 81, 62, 73, 73}
	launchStaffOffsets = []int{52, 79, 36, 64, 37, 46, 54, 33, 75, 34, 91, 46, 91, 65, 75, 79}
	launchBowOffsets   = []int{61, 90, 36, 80, 28, 58, 40, 39, 65, 30, 85, 40, 96, 60, 87, 81}
)

var launchFacings = [8]uint8{0, 32, 64, 96, 128, 160, 192, 224}

var (
	launchMageDeltas = [8]image.Point{{32, -288}, {128, -224}, {136, -128}, {72, -40},
		{-56, -24}, {-152, -96}, {-160, -200}, {-96, -280}}
	launchStaffDeltas = [8]image.Point{{88, -352}, {216, -256}, {216, -104}, {88, 8},
		{-96, 8}, {-224, -112}, {-216, -256}, {-80, -360}}
)

func TestCastLaunchHumanClassesInEightDirections(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		offsets []int
		want    [8]image.Point
	}{
		{"mage", launchMageOffsets, launchMageDeltas},
		{"mage_st", launchStaffOffsets, launchStaffDeltas},
		{"archer", launchBowOffsets, [8]image.Point{{8, -384}, {168, -304}, {256, -144}, {184, 24},
			{-24, 96}, {-224, 16}, {-288, -160}, {-192, -312}}},
		{"sword (empty array)", nil, [8]image.Point{{-48, -57}, {-48, -57}, {-48, -57}, {-48, -57},
			{-48, -57}, {-48, -57}, {-48, -57}, {-48, -57}}},
	} {
		c := launchHuman(tc.offsets)
		for d, facing := range launchFacings {
			if got := castLaunch(c, facing, 34); got != tc.want[d] {
				t.Errorf("%s direction %d: launch %v, want %v", tc.name, d, got, tc.want[d])
			}
			if got := castLaunch(c, facing+16, 34); got != tc.want[d] {
				t.Errorf("%s direction %d odd facing: launch %v, want %v", tc.name, d, got, tc.want[d])
			}
		}
	}
}

func TestCastLaunchTeleportTakesTheSelectionFallback(t *testing.T) {
	t.Parallel()
	c := launchHuman(launchStaffOffsets)
	for _, facing := range launchFacings {
		if got := castLaunch(c, facing, teleportPicture); got != image.Pt(-48, -57) {
			t.Errorf("Teleport at facing %d: launch %v, want (-48,-57)", facing, got)
		}
	}

	const staffClass = 24
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 1, X: 3, Y: 3, HP: 10, MaxHP: 10, Class: staffClass}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	mw := spWorld(t)
	mw.world = w
	mw.units = &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{staffClass: c}}
	mw.projectiles = &terrain.EffectSet{Sheets: map[int]*terrain.EffectSheet{
		teleportPicture: {Frames: spFrames(4, 8), Phases: 4, RotationPhases: 1, CenterX: 4, CenterY: 4}}}
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: 26, FromX: 3, FromY: 3, ToX: 9, ToY: 5}})
	records := flightRecords(mw)
	if len(records) != 2 {
		t.Fatalf("Teleport built %d records, want 2", len(records))
	}
	for i, want := range []image.Point{{3*256 + 128 - 48, 3*256 + 128 - 57}, {9*256 + 128 - 48, 5*256 + 128 - 57}} {
		if got := image.Pt(int(records[i].X), int(records[i].Y)); got != want {
			t.Errorf("Teleport record %d stands at %v, want %v", i, got, want)
		}
	}
}

// MAGIC-261, MAGIC-264 on synthetic creature geometry.
func TestCastLaunchCreatureCasters(t *testing.T) {
	t.Parallel()
	sling := &terrain.UnitClass{CenterX: 64, CenterY: 64, TileSize: 1, Selection: image.Rect(40, 40, 88, 90),
		ShootOffset: []int{59, 77, 44, 67, 42, 53, 51, 42, 66, 39, 80, 45, 85, 59, 77, 72}}
	if got, want := castLaunch(sling, 0, 34), image.Pt(8*(66-64), 8*(39-64)); got != want {
		t.Errorf("ShootOffset creature facing north: %v, want %v", got, want)
	}
	if got, want := castLaunch(sling, 128, 34), image.Pt(8*(59-64), 8*(77-64)); got != want {
		t.Errorf("ShootOffset creature facing south: %v, want %v", got, want)
	}
	plain := &terrain.UnitClass{CenterX: 64, CenterY: 80, TileSize: 1, Selection: image.Rect(40, 40, 90, 100)}
	for _, facing := range launchFacings {
		if got := castLaunch(plain, facing, 10); got != image.Pt(25-64, 30-80) {
			t.Errorf("fallback creature at facing %d: %v, want (-39,-50)", facing, got)
		}
	}
	plain.TileSize = 2
	if got := castLaunch(plain, 0, 10); got != image.Pt(128-39, 128-50) {
		t.Errorf("two-tile fallback creature: %v, want (89,78)", got)
	}
	if got := castLaunch(nil, 0, 10); got != (image.Point{}) {
		t.Errorf("an unknown class launches at %v, want the cell centre", got)
	}
}

func TestACastObjectLeavesTheCastersStaffTip(t *testing.T) {
	t.Parallel()
	const staffClass = 24
	for d, facing := range launchFacings {
		caster := sim.Entity{ID: 1, X: 5, Y: 5, HP: 10, MaxHP: 10, Class: staffClass, Facing: facing}
		w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil,
			[]sim.Entity{caster}, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		mw := spWorld(t)
		mw.world = w
		mw.units = &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{staffClass: launchHuman(launchStaffOffsets)}}
		mw.observeCasts([]sim.CastEvent{
			{Caster: 1, Target: 2, Spell: spLightning, FromX: 5, FromY: 5, ToX: 11, ToY: 9, Facing: facing},
			{Caster: 1, Target: 2, Spell: 1, FromX: 5, FromY: 5, ToX: 11, ToY: 9, Facing: facing},
		})
		launch := castLaunch(launchHuman(launchStaffOffsets), facing, 34)
		want := image.Pt(5*ui.ShotScale, 5*ui.ShotScale).Add(launch)
		records := flightRecords(mw)
		if len(records) != 2 {
			t.Fatalf("direction %d: %d records, want 2", d, len(records))
		}
		display := ui.GroundPixel(want)
		paths := mw.recordPaths(records[0], 0)
		if len(paths) != 1 {
			t.Fatalf("direction %d: Lightning holds %d figures, want 1", d, len(paths))
		}
		if _, _, points := mw.pathFigure(paths[0]); len(points) == 0 || !withinPixel(points[0], display) {
			t.Errorf("direction %d: Lightning starts at %v, want %v", d, points, display)
		}
		// The record's absolute point is the cell centre plus the launch.
		start := want.Add(image.Pt(ui.ShotScale/2, ui.ShotScale/2))
		if trail := mw.shots.trail[records[1].ID]; len(trail) == 0 || trail[0] != start {
			t.Errorf("direction %d: Fire Arrow starts at %v, want %v", d, trail, start)
		}
	}
}

// TestALegacyVisualSnapshotBoltIsDropped: an object in flight is a World
// record, so a snapshot's legacy bolt list restores nothing.
func TestALegacyVisualSnapshotBoltIsDropped(t *testing.T) {
	t.Parallel()
	restored := spWorld(t)
	restored.restoreActionVisuals([]SnapshotSpellBolt{{From: image.Pt(2, 2), To: image.Pt(9, 2), Picture: 10, Life: 9, Age: 3}}, nil)
	if got := flightRecords(restored); len(got) != 0 || len(restored.flights) != 0 {
		t.Fatalf("a legacy bolt restored %+v", got)
	}
}

// withinPixel: the first stored point is the first sample, whose ordinate can
// differ from the launch point by evaluation residue before truncation
// (MAGIC-275).
func withinPixel(a, b image.Point) bool {
	d := a.Sub(b)
	return max(d.X, -d.X, d.Y, -d.Y) <= 1
}
