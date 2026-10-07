package ui

import (
	"bytes"
	"testing"

	"againrom/pkg/render/terrain"
)

// The cliff fixture's relief grid at three minutes of the day, written as
// literals rather than recomputed. cliffAlts raises a 2x2 block by 127, so the
// angle reaches every row: at minute 0 the sun is at -0.78539815, at minute 360
// (noon) it is at zero, and at minute 720 it is at +0.78539815.
//
// THE FIRST AND LAST ARE THE SAME ANGLE MIRRORED and the grids still differ,
// which is the whole of AC-10: the lateral shear's direction follows the angle's
// SIGN, so a mirrored sun is not the same light.
var (
	cliffAtDawn = []uint8{64, 64, 64, 64, 64, 64, 46, 56, 38, 46, 45, 38}
	cliffAtNoon = []uint8{46, 65, 65, 46, 65, 65, 46, 38, 38, 46, 38, 38}
	cliffAtDusk = []uint8{46, 46, 64, 46, 56, 64, 46, 56, 38, 46, 45, 38}
)

func sameLevels(a, b []uint8) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestViewerWithNoClockDrawsWhatItAlwaysDrew(t *testing.T) {
	v, err := NewViewer("m", cliffGrid(), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	if v.Sun() != terrain.DefaultDaytime {
		t.Errorf("a fresh viewer's sun is %+v, want DefaultDaytime %+v", v.Sun(), terrain.DefaultDaytime)
	}
	if !v.TimeFlow() {
		t.Error("the cycle's switch is off on a fresh viewer; the compiled default is on")
	}
	want := terrain.LevelGrid(cliffAlts, cliffW, cliffH, terrain.DefaultDaytime)
	if !sameLevels(v.levels, want) {
		t.Errorf("levels = %v, want the pre-cycle grid %v", v.levels, want)
	}
}

func TestRelightHappensOnlyOnTheCadence(t *testing.T) {
	v, err := NewViewer("m", cliffGrid(), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}

	// Sub-tick 16 is the start of minute 1 — a whole tick, but not a relight
	// minute. Nothing may move.
	seed := v.Sun()
	v.SetLightClock(16)
	if v.Sun() != seed {
		t.Fatalf("the sun moved at sub-tick 16, which no relight fires on: %+v", v.Sun())
	}

	// Sub-tick 320 is minute 20, the second relight of the day.
	v.SetLightClock(320)
	if want := terrain.SunAt(20, true); v.Sun() != want {
		t.Fatalf("at minute 20 the sun is %+v, want %+v", v.Sun(), want)
	}

	// And over a run: 4 relights in sub-ticks 0..1279 (minutes 0, 20, 40, 60).
	v2, err := NewViewer("m", cliffGrid(), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	moves, prev := 0, v2.Sun()
	for s := uint64(0); s < 1280; s++ {
		v2.SetLightClock(s)
		if v2.Sun() != prev {
			moves++
			prev = v2.Sun()
		}
	}
	if moves != 4 {
		t.Errorf("the sun moved %d times over 1280 sub-ticks, want 4", moves)
	}
}

func TestRestoreScheduledLightClockMatchesWarmCache(t *testing.T) {
	warm, err := NewViewer("warm", cliffGrid(), litTileset())
	if err != nil {
		t.Fatal(err)
	}
	var next uint64
	for _, tick := range []uint64{0, 1, 15, 16, 319, 320, 321, 9343, 11951, 15551, 23041} {
		for ; next <= tick; next++ {
			warm.SetLightClock(next)
		}
		cold, err := NewViewer("cold", cliffGrid(), litTileset())
		if err != nil {
			t.Fatal(err)
		}
		cold.RestoreScheduledLightClock(tick)
		if cold.Sun() != warm.Sun() || !sameLevels(cold.levels, warm.levels) {
			t.Fatalf("tick%d: cold sun/grid %+v/%v, warm %+v/%v", tick,
				cold.Sun(), cold.levels, warm.Sun(), warm.levels)
		}
		if cold.lightClock != tick {
			t.Fatalf("retained clock%d, want%d", cold.lightClock, tick)
		}
	}
}

func TestRestoredLightUsesCurrentClockOnlyForAnExplicitForce(t *testing.T) {
	v, err := NewViewer("restored", cliffGrid(), litTileset())
	if err != nil {
		t.Fatal(err)
	}
	v.RestoreScheduledLightClock(9343)
	if theta := v.Sun().Theta; !(theta > 0.4799 && theta < 0.4801) {
		t.Fatalf("restored theta%g, want last scheduled minute580", theta)
	}
	// Restoring must not replace the actual clock with its rounded cadence.
	v.SetTimeFlow(true)
	forced := v.Sun()
	if !(forced.Theta > 0.4864 && forced.Theta < 0.4866) {
		t.Fatalf("forced theta%g, want current minute583", forced.Theta)
	}
	v.SetLightClock(9344)
	if v.Sun() != forced {
		t.Fatal("ordinary non-relight tick changed the explicitly forced cache")
	}
	v.SetLightClock(9600)
	if theta := v.Sun().Theta; !(theta > 0.5235 && theta < 0.5237) {
		t.Fatalf("next scheduled theta%g, want minute600", theta)
	}
}

func TestRelightIsIdempotent(t *testing.T) {
	v, err := NewViewer("m", cliffGrid(), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.SetLightClock(320)
	sun, levels := v.Sun(), append([]uint8(nil), v.levels...)
	v.SetLightClock(320)
	if v.Sun() != sun || !sameLevels(v.levels, levels) {
		t.Error("a second push of the same clock changed the sun or the grid")
	}
}

func TestSwitchForcesARelight(t *testing.T) {
	for _, tc := range []struct {
		name    string
		relight uint64 // a relight instant, to get a cycle-on sun drawn
		then    uint64 // a sub-tick the cadence does not fire on
		moves   bool
	}{
		{"the day band, where the sun is computed", 0, 16, true},
		{"the dawn band, whose literal is the other sign", 19200, 19216, true},
		{"the dusk band, whose angle literal equals the cycle-off arm's but whose shroud pair does not", 11520, 11536, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, err := NewViewer("m", cliffGrid(), litTileset())
			if err != nil {
				t.Fatalf("NewViewer: %v", err)
			}
			if !terrain.RelightDue(tc.relight) {
				t.Fatalf("fixture: sub-tick %d is not a relight instant", tc.relight)
			}
			if terrain.RelightDue(tc.then) {
				t.Fatalf("fixture: sub-tick %d IS a relight instant", tc.then)
			}
			v.SetLightClock(tc.relight)
			v.SetLightClock(tc.then)
			before := v.Sun()

			v.ToggleTimeFlow()
			if v.TimeFlow() {
				t.Fatal("ToggleTimeFlow left the switch on")
			}
			if moved := v.Sun() != before; moved != tc.moves {
				t.Errorf("the switch moved the sun: %v, want %v (from %+v to %+v)",
					moved, tc.moves, before, v.Sun())
			}
			// Whether it moved or not, the sun is now the cycle-off arm's,
			// applied at the sub-tick the force fired on.
			if want := terrain.SunAt(terrain.ClockMinute(tc.then), false); v.Sun() != want {
				t.Errorf("with the cycle off the sun is %+v, want %+v", v.Sun(), want)
			}
		})
	}
}

func TestStepLightClockWalksADayAndReturns(t *testing.T) {
	v, err := NewViewer("m", cliffGrid(), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.SetLightClock(0)
	sun, levels := v.Sun(), append([]uint8(nil), v.levels...)

	v.StepLightClock()
	if v.Sun() == sun {
		t.Fatalf("one hourly step left the sun at %+v", v.Sun())
	}
	if want := terrain.SunAt(60, true); v.Sun() != want {
		t.Errorf("after one step the sun is %+v, want minute 60's %+v", v.Sun(), want)
	}
	for i := 1; i < 24; i++ {
		v.StepLightClock()
	}
	if v.Sun() != sun {
		t.Errorf("after 24 hourly steps the sun is %+v, want the opening %+v", v.Sun(), sun)
	}
	if !sameLevels(v.levels, levels) {
		t.Errorf("after 24 hourly steps the grid is %v, want the opening %v", v.levels, levels)
	}
}

func TestOffsetDoesNotMoveTheCadence(t *testing.T) {
	const hour = terrain.SubTicksPerMinute * 60
	for s := uint64(0); s < 2000; s++ {
		for k := uint64(1); k <= 24; k++ {
			if terrain.RelightDue(s+k*hour) != terrain.RelightDue(s) {
				t.Fatalf("an offset of %d hours moved the cadence at sub-tick %d", k, s)
			}
		}
	}
}

func TestRelightRebuildsTheReliefGrid(t *testing.T) {
	v, err := NewViewer("m", cliffGrid(), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	for _, tc := range []struct {
		minute uint64
		want   []uint8
	}{
		{0, cliffAtDawn},
		{360, cliffAtNoon},
		{720, cliffAtDusk},
	} {
		v.SetLightClock(tc.minute * terrain.SubTicksPerMinute)
		if !sameLevels(v.levels, tc.want) {
			t.Errorf("at minute %d the grid is %v, want %v", tc.minute, v.levels, tc.want)
		}
	}
	if sameLevels(cliffAtDawn, cliffAtDusk) {
		t.Fatal("derivation: the mirrored suns give the same grid, so this fixture does not discriminate")
	}

	flat, err := NewViewer("m", altGrid(3, 4, make([]uint8, 12)...), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	for _, tc := range []struct {
		minute uint64
		want   uint8
	}{
		{0, 46}, {360, 46}, {720, 46}, // day, day, dusk1 at its own phase 0: ambient 14, range 32
		{960, 64}, {1199, 64}, // night, constant: ambient 32, range 8
	} {
		flat.SetLightClock(tc.minute * terrain.SubTicksPerMinute)
		for i, got := range flat.levels {
			if got != tc.want {
				t.Fatalf("flat ground at minute %d, vertex %d = %d, want %d", tc.minute, i, got, tc.want)
			}
		}
	}
}

func TestRelightKeepsTheConstructorsValidityVerdict(t *testing.T) {
	for _, tc := range []struct {
		name string
		g    terrain.Grid
	}{
		{"one entry long", altGrid(2, 2, 1, 2, 3, 4, 5)},
		{"a whole extra row", altGrid(2, 2, 1, 2, 3, 4, 5, 6)},
		{"one entry short", altGrid(2, 2, 1, 2, 3)},
		{"no altitude layer at all", grid(2, 2)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, err := NewViewer("m", tc.g, litTileset())
			if err != nil {
				t.Fatalf("NewViewer: %v", err)
			}
			if v.levels != nil || v.Lit() {
				t.Fatal("fixture drift: this altitude length must already be invalid")
			}
			for s := uint64(0); s < 700; s++ {
				v.SetLightClock(s)
			}
			v.ToggleTimeFlow()
			v.StepLightClock()
			if v.levels != nil {
				t.Errorf("a relight built a grid the constructor refused to: %v", v.levels)
			}
			if v.Lit() {
				t.Error("a relight made an unlit viewer lit")
			}
			// The cache still moves where no grid does: the sun is view state
			// and the grid is what it is applied to.
			want := terrain.SunAt(terrain.ClockMinute(699+terrain.SubTicksPerMinute*60), false)
			if v.Sun() != want {
				t.Errorf("the sun is %+v, want %+v", v.Sun(), want)
			}
		})
	}
}

func TestSpritesTakeTheirLightFromTheSameCache(t *testing.T) {
	v, err := NewViewer("m", cliffGrid(), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	for _, minute := range []uint64{0, 20, 360, 720, 960, 1200, 1420} {
		v.SetLightClock(minute * terrain.SubTicksPerMinute)
		if got, want := v.spriteRow(), terrain.SpriteRow(v.Sun()); got != want {
			t.Fatalf("at minute %d spriteRow = %d, want the cache's %d", minute, got, want)
		}
		if want := terrain.SpriteRow(terrain.SunAt(minute, true)); v.spriteRow() != want {
			t.Errorf("at minute %d spriteRow = %d, want the schedule's %d", minute, v.spriteRow(), want)
		}
		if want := terrain.SunAt(minute, true).SkyTint; v.Sun().SkyTint != want {
			t.Errorf("at minute %d the sky tint is %v, want the schedule's %v", minute, v.Sun().SkyTint, want)
		}
	}
	// Under the unshaded diagnostic the row is still the diagnostic's own, and
	// the sun does not reach it.
	v.SetUnshaded(true)
	if got := v.spriteRow(); got != spriteUnlit {
		t.Errorf("unshaded spriteRow = %d, want %d", got, spriteUnlit)
	}
}

func TestTheSpriteWireIsTheCacheAndNotAConstant(t *testing.T) {
	v, err := NewViewer("m", cliffGrid(), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	// Ambient 0x28 is ramp row 10 where DefaultDaytime's 0x0e is row 3, and the
	// tint is non-zero where every band's is (0,0,0).
	v.sun = terrain.Light{Ambient: 0x28, Range: 0x20, SkyTint: [3]uint8{9, 0, 5}}

	if got, want := v.spriteRow(), terrain.SpriteRow(v.sun); got != want {
		t.Errorf("spriteRow = %d, want the cache's %d", got, want)
	}
	if got := v.spriteRow(); got == terrain.SpriteRow(terrain.DefaultDaytime) {
		t.Errorf("spriteRow = %d, the constant's own row: the wire is not the cache", got)
	}

	f := spriteLightFrame(4, 4)
	got := v.spritePixels(f)
	if want := f.RGBALit(v.sun.SkyTint, terrain.SpriteRow(v.sun)); !bytes.Equal(got.Pix, want.Pix) {
		t.Error("spritePixels did not shade by the cache's tint and row")
	}
	if fixed := f.RGBALit(terrain.DefaultDaytime.SkyTint, terrain.SpriteRow(terrain.DefaultDaytime)); bytes.Equal(got.Pix, fixed.Pix) {
		t.Error("spritePixels' pixels are the constant's: the tint is not the cache's")
	}
}

// BenchmarkRelight is SC-3's measurement: what one relight costs over a map the
// size of the largest the corpus ships, 256x256 vertices.
//
// It is EVIDENCE INFRASTRUCTURE and belongs to the verification stage rather
// than to a task, which is why it arrives untrailered with verification.md. The
// figure it produces is only meaningful beside the period it happens at: one
// relight per 20 in-game minutes, which at the cadence a map opens on is 320
// sub-ticks of 62 ms -- a little under 20 s of real time.
func BenchmarkRelight(b *testing.B) {
	const w, h = 256, 256
	alts := make([]uint8, w*h)
	for i := range alts {
		alts[i] = uint8((i*7 + i/w*13) % 128)
	}
	g := terrain.Grid{Width: w, Height: h, Tiles: make([]uint16, w*h), Altitudes: alts}
	v, err := NewViewer("m", g, litTileset())
	if err != nil {
		b.Fatalf("NewViewer: %v", err)
	}
	if v.levels == nil {
		b.Fatal("fixture must be lit")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.lightClock = uint64(i) * terrain.SubTicksPerMinute * terrain.RelightPeriodMinutes
		v.relight()
	}
}
