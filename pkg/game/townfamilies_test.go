package game

import (
	"fmt"
	"image"
	"image/color"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// familyFrames builds one 3x3 solid frame per index whose colour names the
// family, sheet and frame, so a pixel identifies what was drawn.
func familyFrames(count int, base, sheet byte) []image.Image {
	out := make([]image.Image, count)
	for i := range out {
		pic := image.NewRGBA(image.Rect(0, 0, 3, 3))
		for y := 0; y < 3; y++ {
			for x := 0; x < 3; x++ {
				pic.SetRGBA(x, y, color.RGBA{R: base + byte(i), G: 10 + 20*sheet, B: base, A: 255})
			}
		}
		out[i] = pic
	}
	return out
}

const (
	familyHorseBase   = 10
	familyBabaBase    = 60
	familyDervishBase = 110
)

func familyColour(base byte, sheet, frame int) color.RGBA {
	return color.RGBA{R: base + byte(frame), G: 10 + 20*byte(sheet), B: base, A: 255}
}

func installTownFamilyArt(a *town.Art, horse, baba, dervish bool) {
	for p := 0; p < 5; p++ {
		for v := 0; v < 3; v++ {
			if horse {
				a.Frames[fmt.Sprintf("horse/%d/%d", p, v)] = familyFrames(15, familyHorseBase, byte(v))
			}
		}
	}
	for p := 0; p < 4; p++ {
		for v := 0; v < 2; v++ {
			if baba {
				a.Frames[fmt.Sprintf("baba/%d/%d", p, v)] = familyFrames(31+v, familyBabaBase, byte(v))
			}
		}
	}
	for p := 0; p < 4; p++ {
		if dervish {
			a.Frames[fmt.Sprintf("dervish/%d", p)] = familyFrames(30, familyDervishBase, 0)
		}
	}
}

// familyDraws is the scripted generator. Once the script is spent it answers
// the value that makes every later arm long and quiet.
type familyDraws struct {
	script []int
}

const rawQuiet = 0x7ffe // delay 6999 ms, last sheet

func (d *familyDraws) next() int {
	if len(d.script) == 0 {
		return rawQuiet
	}
	v := d.script[0]
	d.script = d.script[1:]
	return v
}

type townFamilyRig struct {
	f      *FrontEnd
	app    *ui.App
	s      *townScreen
	now    *time.Time
	voices *exteriorRecorder
	draws  *familyDraws
	rolls  *int
}

func newTownFamilyRig(t *testing.T, horse, baba, dervish bool) *townFamilyRig {
	t.Helper()
	f := shellFrontEnd()
	art := exteriorTestArt(t)
	installTownFamilyArt(art, horse, baba, dervish)
	f.TownSquareArt = resolved(art, nil)
	f.Town.announceMission(f.Town.currentMain())
	now := time.Unix(500, 0)
	rolls := 0
	f.TownAnimationNow = func() time.Time { return now }
	f.TownAnimationRandom = func(int) int { rolls++; return 0 }
	f.TownAmbientRandom = func(int) int { rolls += 1000; return 0 }
	r := &exteriorRecorder{}
	f.SoundPlayer = r
	f.SoundBank = &SoundBank{named: map[string]soundCacheEntry{}, cache: map[int]soundCacheEntry{}}
	for i, p := range []string{"town/horse1.wav", "town/horse2.wav"} {
		f.SoundBank.named[p] = soundCacheEntry{sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{int16(111 + i)}}, ok: true}
	}
	app, s := exteriorApp(t, f)
	return &townFamilyRig{f: f, app: app, s: s, now: &now, voices: r, draws: &familyDraws{}, rolls: &rolls}
}

// enter starts a fresh view whose generator answers the script and checks the
// entry drew exactly those values.
func (r *townFamilyRig) enter(t *testing.T, script ...int) {
	t.Helper()
	r.draws.script = script
	r.s.squareView().SetRawDraw("wildlife", r.draws.next)
	r.s.resetTownExterior()
	r.s.squareView().SetClockLast(time.Time{})
	r.s.TownSquareActive(true)
	if len(r.draws.script) != 0 {
		t.Fatalf("entry left %d scripted draws", len(r.draws.script))
	}
	exteriorPaint(t, r.app, r.now, 0)
}

func (r *townFamilyRig) paint(t *testing.T, dt time.Duration) *image.RGBA {
	t.Helper()
	return exteriorPaint(t, r.app, r.now, dt)
}

func (r *townFamilyRig) startsOf(id int16) int { return sampleStarts(r.voices, id) }

func (r *townFamilyRig) endVoices() {
	for _, v := range r.voices.voices {
		v.playing = false
	}
}

// rawH and rawB are the smallest raw draws selecting position i (TOWN-004).
func rawH(i int) int { return (i*0x7fff + 4) / 5 }
func rawB(i int) int { return (i*0x7fff + 3) / 4 }

const (
	rawZero = 0
	rawMax  = 0x7fff
	// What the original's formulas make of these raw values:
	rawEntryDelayLong = 16383 // entry delay: (r*2000)/0x7fff = 999
	rawArmDelayMid    = 16383 // arm delay: (r*5000)/0x7fff = 2499
	rawHalf           = 16384 // (r*2)/0x7fff = 1 and (r*3)/0x7fff = 1
	rawSheet3         = 22000 // (r*3)/0x7fff = 2
)

// Entry draws the three positions, the dervish re-rolled off the baba's pick
// (TOWN-004), then the baba and horse entry delays (TOWN-491); all three
// families show frame 0 from entry at their table rectangles.
func TestTownFamiliesEnterDrawnAtFrameZeroOnTheChosenPositions(t *testing.T) {
	r := newTownFamilyRig(t, true, true, true)
	// horse 2, baba 1, dervish 1 (the baba's, re-rolled) then 3, baba delay
	// 2999 ms, horse delay 2000 ms.
	r.enter(t, rawH(2), rawB(1), rawB(1), rawB(3), rawEntryDelayLong, rawMax)
	w := r.s.sqWildlife()
	horse, baba, dervish := w.Member("horse"), w.Member("baba"), w.Member("dervish")
	if horse.Position != 2 || baba.Position != 1 || dervish.Position != 3 {
		t.Fatalf("positions horse%d baba%d dervish%d", horse.Position, baba.Position, dervish.Position)
	}
	if baba.Wait != 2999*time.Millisecond || horse.Wait != 2000*time.Millisecond {
		t.Fatalf("entry delays baba %v horse %v", baba.Wait, horse.Wait)
	}
	if baba.Sheet != 0 || horse.Sheet != 0 || baba.Current != -1 || horse.Current != -1 || baba.Active || horse.Active || !dervish.Active {
		t.Fatalf("entry state %+v", w.Members)
	}
	pix := r.paint(t, 0)
	for _, c := range []struct {
		name string
		p    image.Point
		want color.RGBA
	}{
		{"horse position 3, A1 frame 0", image.Pt(256, 344), familyColour(familyHorseBase, 0, 0)},
		{"baba position 2, A1 frame 0", image.Pt(308, 424), familyColour(familyBabaBase, 0, 0)},
		{"dervish position 4 frame 0", image.Pt(592, 388), familyColour(familyDervishBase, 0, 0)},
	} {
		if got := pix.RGBAAt(c.p.X, c.p.Y); got != c.want {
			t.Errorf("%s at %v = %v want %v", c.name, c.p, got, c.want)
		}
	}
}

// A horse episode: the delay is strictly exceeded on a paint, the arm draws
// delay then sheet, the first step comes on a later hub, frames 1..14 follow
// one per hub, the horse sounds fire on their gates only when no voice of that
// sample plays, and nothing but cleanup stops a voice (TOWN-445, 491..495).
func TestTownFamiliesHorseEpisodeDelayStepsAndSounds(t *testing.T) {
	r := newTownFamilyRig(t, true, true, true)
	r.enter(t, rawH(2), rawB(1), rawB(3), rawEntryDelayLong, rawMax)
	frame := func() familyFrame { return r.s.sqExteriorFrame().Horse }

	r.paint(t, 2000*time.Millisecond)
	if r.s.sqWildlife().Member("horse").Active {
		t.Fatal("elapsed equal to the delay armed the horse")
	}
	r.draws.script = []int{rawArmDelayMid, rawSheet3}
	r.paint(t, time.Millisecond)
	h := r.s.sqWildlife().Member("horse")
	if !h.Active || h.Sheet != 2 || h.Current != 0 || h.Wait != 4499*time.Millisecond || len(r.draws.script) != 0 {
		t.Fatalf("arm state %+v script left %v", h, r.draws.script)
	}
	if f := frame(); !f.Visible || f.Sheet != 2 || f.Frame != 0 {
		t.Fatalf("arm frame %+v", f)
	}
	if r.startsOf(111) != 0 {
		t.Fatal("sound before frame 1")
	}

	// The next hub is the first step: A3 frame 1 requests Horse1 once.
	r.paint(t, 68*time.Millisecond)
	if frame().Frame != 1 || r.startsOf(111) != 1 || r.startsOf(112) != 0 {
		t.Fatalf("first step frame %d horse1 %d horse2 %d", frame().Frame, r.startsOf(111), r.startsOf(112))
	}
	// Paints inside the dwell change no frame and, while the sample plays,
	// request nothing.
	for i := 0; i < 3; i++ {
		r.paint(t, 10*time.Millisecond)
	}
	if frame().Frame != 1 || r.startsOf(111) != 1 {
		t.Fatalf("dwell changed frame %d or duplicated Horse1 (%d)", frame().Frame, r.startsOf(111))
	}
	// The sample has ended: the next paint at the same frame requests it again.
	r.voices.voices[0].playing = false
	r.paint(t, 10*time.Millisecond)
	if r.startsOf(111) != 2 {
		t.Fatalf("an ended sample was not requested again: %d", r.startsOf(111))
	}

	for want := 2; want <= 14; want++ {
		r.paint(t, 68*time.Millisecond)
		if got := frame().Frame; got != want {
			t.Fatalf("frame %d, want %d", got, want)
		}
	}
	if r.startsOf(112) != 1 {
		t.Fatalf("Horse2 at A3 frame 14: %d", r.startsOf(112))
	}
	horse2 := r.voices.voices[len(r.voices.voices)-1]
	r.paint(t, 68*time.Millisecond)
	if h := r.s.sqWildlife().Member("horse"); h.Active || h.Current != -1 {
		t.Fatalf("terminal step left %+v", h)
	}
	if f := frame(); !f.Visible || f.Frame != 0 || f.Sheet != 2 {
		t.Fatalf("idle frame %+v", f)
	}
	if !horse2.playing || horse2.stops != 0 {
		t.Fatal("the terminal step stopped a playing horse voice")
	}

	// The delay runs from the last step and is strict.
	r.paint(t, 4499*time.Millisecond)
	if r.s.sqWildlife().Member("horse").Active {
		t.Fatal("elapsed equal to the rolled delay armed the horse")
	}
	r.draws.script = []int{rawZero, rawZero}
	r.paint(t, time.Millisecond)
	if h := r.s.sqWildlife().Member("horse"); !h.Active || h.Sheet != 0 || h.Wait != 2000*time.Millisecond {
		t.Fatalf("second arm %+v", h)
	}

	// Leaving the square runs the town sound cleanup: the playing voice stops.
	r.s.TownSquareActive(false)
	if horse2.playing || horse2.stops != 1 {
		t.Fatalf("cleanup left Horse2 playing=%v stops=%d", horse2.playing, horse2.stops)
	}
}

// Sheet A2 requests Horse2 on frames 8 and 14, A1 only on 14.
func TestTownFamiliesHorse2GatesPerSheet(t *testing.T) {
	for _, c := range []struct {
		sheet, raw, want, first int
	}{{0, rawZero, 1, 14}, {1, rawHalf, 2, 8}} {
		r := newTownFamilyRig(t, true, true, true)
		r.enter(t, 0, 0, rawB(1), rawEntryDelayLong, rawZero)
		r.draws.script = []int{rawZero, c.raw}
		r.paint(t, 2001*time.Millisecond)
		if r.s.sqWildlife().Member("horse").Sheet != c.sheet || !r.s.sqWildlife().Member("horse").Active {
			t.Fatalf("sheet A%d armed as %+v", c.sheet+1, r.s.sqWildlife().Member("horse"))
		}
		seen := -1
		for f := 1; f <= 14; f++ {
			r.paint(t, 68*time.Millisecond)
			if seen < 0 && r.startsOf(112) > 0 {
				seen = f
			}
			r.endVoices()
		}
		if seen != c.first || r.startsOf(112) != c.want || r.startsOf(111) != 0 {
			t.Errorf("sheet A%d: first Horse2 at frame %d (want %d), Horse2 %d (want %d), Horse1 %d", c.sheet+1, seen, c.first, r.startsOf(112), c.want, r.startsOf(111))
		}
	}
}

// Baba steps through its own sheet length and returns to frame 0 at the end.
func TestTownFamiliesBabaSheetLengthsAndDelays(t *testing.T) {
	for _, c := range []struct {
		raw, sheet, frames int
	}{{rawZero, 0, 31}, {rawHalf, 1, 32}} {
		r := newTownFamilyRig(t, true, true, true)
		r.enter(t, 0, 0, rawB(1), rawZero, rawMax)
		r.paint(t, 2000*time.Millisecond)
		if r.s.sqWildlife().Member("baba").Active {
			t.Fatal("elapsed equal to the baba delay armed it")
		}
		r.draws.script = []int{rawZero, c.raw}
		r.paint(t, time.Millisecond)
		if b := r.s.sqWildlife().Member("baba"); !b.Active || b.Sheet != c.sheet || b.Wait != 2000*time.Millisecond {
			t.Fatalf("baba arm %+v", b)
		}
		steps := 0
		for r.s.sqWildlife().Member("baba").Active {
			r.paint(t, 68*time.Millisecond)
			steps++
			if steps > 40 {
				t.Fatal("episode never ended")
			}
		}
		if steps != c.frames || r.s.sqWildlife().Member("baba").Current != -1 {
			t.Errorf("sheet A%d: %d steps, want %d", c.sheet+1, steps, c.frames)
		}
	}
}

// The dervish runs from entry and wraps at its 30 frames, one per hub.
func TestTownFamiliesDervishRevolutionOnePerHub(t *testing.T) {
	r := newTownFamilyRig(t, true, true, true)
	r.enter(t, 0, 0, rawB(1), rawMax, rawMax)
	for i := 1; i <= 31; i++ {
		r.paint(t, 68*time.Millisecond)
		if got := r.s.sqExteriorFrame().Dervish.Frame; got != i%30 {
			t.Fatalf("hub %d dervish frame %d", i, got)
		}
		r.paint(t, 10*time.Millisecond)
		if got := r.s.sqExteriorFrame().Dervish.Frame; got != i%30 {
			t.Fatalf("paint inside the hub interval moved the dervish to %d", got)
		}
	}
}

// A family whose sheet failed to load draws nothing and requests no sound;
// the others are unaffected.
func TestTownFamiliesAbsentFamilyDrawsNothingOthersContinue(t *testing.T) {
	for _, c := range []struct {
		name                 string
		horse, baba, dervish bool
	}{{"all present", true, true, true}, {"no horse", false, true, true}, {"no baba", true, false, true}, {"no dervish", true, true, false}} {
		r := newTownFamilyRig(t, c.horse, c.baba, c.dervish)
		r.enter(t, rawH(2), rawB(1), rawB(3), rawZero, rawZero)
		r.draws.script = []int{rawZero, rawSheet3, rawZero, rawSheet3}
		r.paint(t, 2001*time.Millisecond)
		for i := 0; i < 5; i++ {
			r.paint(t, 68*time.Millisecond)
			r.endVoices()
		}
		pix := r.paint(t, 0)
		fr := r.s.sqExteriorFrame()
		for _, tc := range []struct {
			name string
			p    image.Point
			want color.RGBA
			here bool
		}{
			{"horse", image.Pt(256, 344), familyColour(familyHorseBase, fr.Horse.Sheet, fr.Horse.Frame), c.horse},
			{"baba", image.Pt(308, 424), familyColour(familyBabaBase, fr.Baba.Sheet, fr.Baba.Frame), c.baba},
			{"dervish", image.Pt(592, 388), familyColour(familyDervishBase, 0, fr.Dervish.Frame), c.dervish},
		} {
			if drawn := pix.RGBAAt(tc.p.X, tc.p.Y) == tc.want; drawn != tc.here {
				t.Errorf("%s: %s drawn=%v want %v (pixel %v, expected sheet colour %v)", c.name, tc.name, drawn, tc.here, pix.RGBAAt(tc.p.X, tc.p.Y), tc.want)
			}
		}
		if got := len(r.voices.voices) > 0; got != c.horse {
			t.Errorf("%s: horse sounds requested=%v want %v", c.name, got, c.horse)
		}
	}
}

// The delay test runs on every paint and does not look at the active bit: a
// family whose clock is stale restarts at frame 0 on a freshly drawn sheet.
func TestTownFamiliesArmTestIgnoresAnActiveEpisode(t *testing.T) {
	r := newTownFamilyRig(t, true, true, true)
	r.enter(t, 0, 0, rawB(1), rawMax, rawMax)
	b := r.s.sqWildlife().Member("baba")
	b.Active, b.Current, b.Sheet = true, 7, 0
	b.Clock, b.Wait = r.now.Add(-5*time.Second), 2*time.Second
	r.draws.script = []int{rawZero, rawHalf}
	r.paint(t, 0)
	if !b.Active || b.Current != 0 || b.Sheet != 1 || !b.Clock.Equal(*r.now) {
		t.Fatalf("restart %+v", b)
	}
}

// seedHost is a still host with a fixed generator seed.
type seedHost struct {
	stillHost
	seed int64
}

func (h seedHost) Seed() int64 { return h.seed }

// crtRand is an independent CRT rand: state*214013+2531011, bits 16..30.
type crtRand struct{ state uint32 }

func (g *crtRand) next() int {
	g.state = g.state*214013 + 2531011
	return int(g.state>>16) & 0x7fff
}

// The generator is the CRT rand (AI-RAND-058) and the entry and arm delays
// stay inside their bounds over a seeded run.
func TestTownFamilyGeneratorIsTheCRTRandAndDelaysStayInBounds(t *testing.T) {
	v := town.NewView(rom1Town, seedHost{seed: 1}, nil)
	oracle := crtRand{state: 1}
	for i := 0; i < 5; i++ {
		raw := oracle.next()
		if got, want := v.Pick("wildlife", "scaled", 2000), raw*2000/0x7fff%2000; got != want {
			t.Fatalf("draw %d = %d want %d (raw %d)", i, got, want, raw)
		}
	}
	v = town.NewView(rom1Town, seedHost{seed: 1234}, nil)
	loEntry, hiEntry, loArm, hiArm := 1<<30, 0, 1<<30, 0
	for i := 0; i < 200000; i++ {
		e := 2000 + v.Pick("wildlife", "scaled", 2000)
		a := 2000 + v.Pick("wildlife", "scaled", 5000)
		loEntry, hiEntry = min(loEntry, e), max(hiEntry, e)
		loArm, hiArm = min(loArm, a), max(hiArm, a)
	}
	if loEntry < 2000 || hiEntry > 3999 || loArm < 2000 || hiArm > 6999 {
		t.Fatalf("entry %d..%d arm %d..%d", loEntry, hiEntry, loArm, hiArm)
	}
	if hiEntry < 3990 || loArm > 2010 || hiArm < 6990 {
		t.Fatalf("seeded run does not span the bounds: entry %d..%d arm %d..%d", loEntry, hiEntry, loArm, hiArm)
	}
	// The raw extremes: 0 and 0x7fff both give the minimum, 0x7ffe the maximum.
	for raw, want := range map[int]int{0: 2000, 0x7fff: 2000, 0x7ffe: 3999} {
		v.SetRawDraw("wildlife", func() int { return raw })
		if got := 2000 + v.Pick("wildlife", "scaled", 2000); got != want {
			t.Errorf("raw %#x: entry delay %d want %d", raw, got, want)
		}
	}
}

// The families draw from their own generator: the established exterior and
// ambient streams see exactly the draws they saw without the families.
func TestTownFamiliesLeaveEveryOtherGeneratorAlone(t *testing.T) {
	run := func(families bool) int {
		r := newTownFamilyRig(t, families, families, families)
		r.s.resetTownExterior()
		r.s.squareView().ResetDraw("wildlife")
		r.s.squareView().SetClockLast(time.Time{})
		r.s.TownSquareActive(true)
		exteriorPaint(t, r.app, r.now, 0)
		for i := 0; i < 400; i++ {
			exteriorPaint(t, r.app, r.now, 68*time.Millisecond)
		}
		return *r.rolls
	}
	with, without := run(true), run(false)
	if with != without || with == 0 {
		t.Fatalf("exterior draws with families %d, without %d", with, without)
	}
}

// Position rolls are scaled quotients, not r mod n (TOWN-505).
func TestTownFamilyPositionRollsAreTheScaledQuotients(t *testing.T) {
	roll := func(raw int) (horse, baba int) {
		v := town.NewView(rom1Town, stillHost{}, nil)
		v.SetRawDraw("wildlife", func() int { return raw })
		return v.Pick("wildlife", "scaled", 5), v.Pick("wildlife", "masked", 4)
	}
	for _, c := range []struct{ raw, horse, baba int }{
		{0, 0, 0},
		{0x7fff, 0, 0}, // 5 mod 5 and 4 and 3
		{0x7ffe, 4, 3}, // r mod 5 = 1 and r mod 4 = 2 under the old reading
		{rawH(1), 1, 0},
		{rawH(1) - 1, 0, 0},
		{rawB(1), 1, 1},
		{rawB(1) - 1, 1, 0},
		{rawB(3), 3, 3},
		{rawH(4), 4, 3},
		{rawH(4) - 1, 3, 3},
	} {
		if h, b := roll(c.raw); h != c.horse || b != c.baba {
			t.Errorf("raw %#x: horse %d baba %d want %d %d", c.raw, h, b, c.horse, c.baba)
		}
	}
	var horse [5]int
	var baba [4]int
	for raw := 0; raw <= 0x7fff; raw++ {
		h, b := roll(raw)
		horse[h]++
		baba[b]++
	}
	for i, n := range horse {
		if n == 0 {
			t.Errorf("horse position %d unreachable", i)
		}
	}
	for i, n := range baba {
		if n == 0 {
			t.Errorf("baba position %d unreachable", i)
		}
	}
}
