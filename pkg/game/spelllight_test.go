package game

import (
	"fmt"
	"image"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func lightGrid(stamps []ui.LightStamp) map[image.Point]uint8 {
	out := map[image.Point]uint8{}
	for _, s := range stamps {
		out[s.Vertex] = s.Level
	}
	return out
}

// MAGIC-270, MAGIC-272: u8(10*phase) on the corners of every figure cell.
func TestLightningAndPrismaticLightTheirPathPerPhase(t *testing.T) {
	t.Parallel()
	wantPhases := []uint8{4, 3, 2, 1, 0, 1, 2, 1, 0, 1, 2, 3, 4}
	for _, spell := range []int{spLightning, spPrismatic} {
		mw := spWorld(t)
		mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: uint16(spell),
			FromX: 2, FromY: 2, ToX: 9, ToY: 6}})
		for age := 0; age < 13; age++ {
			stamps := mw.objectLightStamps(nil)
			if len(stamps) == 0 {
				t.Fatalf("spell %d age %d stamps nothing", spell, age)
			}
			for _, s := range stamps {
				if s.Point || s.Level != 10*wantPhases[age] {
					t.Fatalf("spell %d age %d stamp %+v, want path level %d", spell, age, s, 10*wantPhases[age])
				}
			}
			want := map[image.Point]bool{}
			var points []image.Point
			for _, b := range mw.recordPaths(flightRecords(mw)[0], 0) {
				_, _, figure := mw.pathFigure(b)
				points = append(points, figure...)
			}
			for _, p := range points {
				c, _ := mw.displayLightCell(p)
				for _, v := range []image.Point{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
					want[c.Add(v)] = true
				}
			}
			got := lightGrid(stamps)
			if len(got) != len(want) {
				t.Fatalf("spell %d age %d stamped %d vertices, want %d", spell, age, len(got), len(want))
			}
			for v := range want {
				if _, ok := got[v]; !ok {
					t.Fatalf("spell %d age %d left path vertex %v unstamped", spell, age, v)
				}
			}
			flightStep(mw)
		}
		if got := mw.objectLightStamps(nil); len(got) != 0 {
			t.Fatalf("spell %d: an expired path still stamps %d vertices", spell, len(got))
		}
	}
}

// MAGIC-281: the direct route starts at actionphase -1, so its five calls
// keep phase 0 and then take 4,3,2,1.
func TestADirectLightningLightsFivePhases(t *testing.T) {
	t.Parallel()
	mw := spWorld(t)
	mw.observeScriptCasts([]sim.ScriptCastEvent{{Spell: spLightning, FromX: 2, FromY: 2, ToX: 7, ToY: 2}})
	for _, phase := range []uint8{0, 4, 3, 2, 1} {
		stamps := mw.objectLightStamps(nil)
		if len(stamps) == 0 || stamps[0].Level != 10*phase {
			t.Fatalf("direct Lightning stamps %v, want level %d", stamps, 10*phase)
		}
		flightStep(mw)
	}
	if len(flightRecords(mw)) != 0 {
		t.Fatal("the direct object outlived five calls")
	}
}

func TestPointLightFootprints(t *testing.T) {
	t.Parallel()
	for radius, want := range []int{4, 12, 32, 52} {
		stamps := pointLightStamps(nil, image.Pt(10, 10), radius, 7)
		if got := len(lightGrid(stamps)); got != want || len(stamps) != want {
			t.Errorf("radius %d stamps %d (%d distinct) vertices, want %d", radius, len(stamps), got, want)
		}
		for _, s := range stamps {
			if !s.Point || s.Level != 7 {
				t.Errorf("radius %d stamp %+v", radius, s)
			}
		}
	}
	got := lightGrid(pointLightStamps(nil, image.Pt(10, 10), 1, 0))
	for _, v := range []image.Point{{10, 10}, {11, 11}, {9, 10}, {12, 11}, {10, 9}, {11, 12}} {
		if _, ok := got[v]; !ok {
			t.Errorf("radius 1 left %v unstamped", v)
		}
	}
	for _, v := range []image.Point{{9, 9}, {12, 12}} {
		if _, ok := got[v]; ok {
			t.Errorf("radius 1 stamped the diagonal %v", v)
		}
	}
}

// MAGIC-271: flight radii 0 and 1 at level 16; the explosion phase table.
func TestFireArrowFireBallAndExplosionLight(t *testing.T) {
	t.Parallel()
	mw := spWorld(t)
	mw.observeCasts([]sim.CastEvent{
		{Caster: 1, Target: 2, Spell: 1, FromX: 1, FromY: 1, ToX: 10, ToY: 1},
		{Caster: 3, Target: 4, Spell: spFireBall, FromX: 1, FromY: 8, ToX: 10, ToY: 8},
	})
	stamps := mw.objectLightStamps(nil)
	var arrow, ball []ui.LightStamp
	for _, s := range stamps {
		if s.Level != 16 || !s.Point {
			t.Fatalf("flight stamp %+v, want point level 16", s)
		}
		if s.Vertex.Y < 5 {
			arrow = append(arrow, s)
		} else {
			ball = append(ball, s)
		}
	}
	if len(arrow) != 4 || len(ball) != 12 {
		t.Fatalf("Fire Arrow stamped %d and Fire Ball %d vertices, want 4 and 12", len(arrow), len(ball))
	}
	b := flightRecords(mw)[0]
	cell := image.Pt(floorDiv(int(b.X), ui.ShotScale), floorDiv(int(b.Y), ui.ShotScale))
	if got := lightGrid(arrow); len(got) != 4 || got[cell] != 16 || got[cell.Add(image.Pt(1, 1))] != 16 {
		t.Errorf("Fire Arrow stamps %v, want the four corners of drawn cell %v", got, cell)
	}

	table := []struct {
		radius int
		level  uint8
	}{{1, 16}, {2, 8}, {3, 0}, {3, 0}, {3, 8}, {3, 16}, {3, 24}, {3, 32}, {3, 40}, {3, 46}, {3, 46}}
	counts := []int{4, 12, 32, 52}
	for phase, e := range table {
		got := pictureLightStamps(nil, fireBallBurstPicture, image.Pt(10, 10), phase)
		if len(got) != counts[e.radius] {
			t.Errorf("explosion phase %d stamps %d vertices, want radius %d's %d", phase, len(got), e.radius, counts[e.radius])
		}
		for _, s := range got {
			if s.Level != e.level || !s.Point {
				t.Errorf("explosion phase %d stamp %+v, want level %d", phase, s, e.level)
			}
		}
	}
	for _, phase := range []int{-1, 11, 21} {
		if got := pictureLightStamps(nil, fireBallBurstPicture, image.Pt(10, 10), phase); len(got) != 0 {
			t.Errorf("explosion phase %d stamps %d vertices, want none", phase, len(got))
		}
	}
	for _, picture := range []int{1, 7, 11, 14, 20, 30, 60} {
		if got := pictureLightStamps(nil, picture, image.Pt(10, 10), 0); len(got) != 0 {
			t.Errorf("picture %d stamps light", picture)
		}
	}
}

func TestObjectLightLeavesTheWorldUnchanged(t *testing.T) {
	t.Parallel()
	mw := spWorld(t)
	mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spLightning, FromX: 2, FromY: 2, ToX: 8, ToY: 5}})
	before := mw.world.Hash()
	mw.objectLightStamps(nil)
	if mw.world.Hash() != before {
		t.Fatal("building the light stamps changed the world hash")
	}
}

// A Fire Arrow and a Fire Ball record stamp their light at the cell the
// record stands in on every call (MAGIC-271).
func TestFlightLightStaysAtTheRecordsCell(t *testing.T) {
	t.Parallel()
	for _, spell := range []uint16{1, 2} {
		mw := sbWorld(t)
		mw.projectiles.Sheets[12] = &terrain.EffectSheet{Frames: spFrames(4, 8), Phases: 4, RotationPhases: 1}
		mw.observeCasts([]sim.CastEvent{{Caster: 1, Target: 2, Spell: spell, Weapon: true,
			FromX: 2, FromY: 2, ToX: 10, ToY: 2}})
		for call := 0; len(flightRecords(mw)) > 0; call++ {
			p := flightRecords(mw)[0]
			cell := image.Pt(floorDiv(int(p.X), ui.ShotScale), floorDiv(int(p.Y), ui.ShotScale))
			got := lightGrid(mw.objectLightStamps(nil))
			want := lightGrid(pictureLightStamps(nil, data.CastPicture(int(spell)), cell, -1))
			if len(want) == 0 || fmt.Sprint(got) != fmt.Sprint(want) {
				t.Errorf("spell %d call %d: record at %v, light %v, want %v", spell, call, cell, got, want)
			}
			flightStep(mw)
		}
	}
}
