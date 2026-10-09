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

	mw := spWorld(t)
	set := &terrain.EffectSet{Sheets: map[int]*terrain.EffectSheet{
		teleportPicture: {Frames: spFrames(4, 8), Phases: 4, RotationPhases: 1, CenterX: 4, CenterY: 4}}}
	mw.projectiles = set
	mw.spawnCast(image.Pt(3, 3), image.Pt(9, 5), 26, 0, 1, 0, 0, c)
	if len(mw.bolts) != 2 {
		t.Fatalf("Teleport spawned %d objects, want 2", len(mw.bolts))
	}
	for i, want := range []image.Point{{3*256 - 48, 3*256 - 57}, {9*256 - 48, 5*256 - 57}} {
		b := mw.bolts[i]
		if got := castShotPoint(b.from, b.to, 1, b.life, b.launch); got != want {
			t.Errorf("Teleport object %d stands at %v, want %v", i, got, want)
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
		want := image.Pt(5*ui.ShotScale, 5*ui.ShotScale).Add(castLaunch(launchHuman(launchStaffOffsets), facing, 34))
		if len(mw.bolts) != 2 {
			t.Fatalf("direction %d: %d objects, want 2", d, len(mw.bolts))
		}
		if _, _, points := mw.pathFigure(mw.bolts[0]); len(points) == 0 || points[0] != want {
			t.Errorf("direction %d: Lightning starts at %v, want %v", d, points, want)
		}
		if got := castShotPoint(mw.bolts[1].from, mw.bolts[1].to, 0, mw.bolts[1].life, mw.bolts[1].launch); got != want {
			t.Errorf("direction %d: Fire Arrow starts at %v, want %v", d, got, want)
		}
	}
}

func TestACastObjectInFlightKeepsItsLaunchAcrossTheVisualSnapshot(t *testing.T) {
	t.Parallel()
	mw := spWorld(t)
	mw.bolts = []spellBolt{{from: image.Pt(2, 2), to: image.Pt(9, 2), picture: 10, life: 9, age: 3,
		facing: 64, launch: image.Pt(216, -104)}}
	var r SnapshotResidue
	mw.actionVisuals(&r)
	restored := spWorld(t)
	restored.restoreActionVisuals(r.SpellBolts, r.HealBursts)
	if len(restored.bolts) != 1 || restored.bolts[0] != mw.bolts[0] {
		t.Fatalf("restored %+v, want %+v", restored.bolts, mw.bolts)
	}
	old := r.SpellBolts[0]
	old.Launch = image.Point{}
	restored.restoreActionVisuals([]SnapshotSpellBolt{old}, nil)
	if restored.bolts[0].launch != (image.Point{}) {
		t.Fatalf("an envelope without Launch restored %v", restored.bolts[0].launch)
	}
}
