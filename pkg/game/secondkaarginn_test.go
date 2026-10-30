package game

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	"againrom/pkg/town"
)

// The Kaarg inn page runs over a synthetic host: a clock, a queue of raw
// draws and a log of draws, sound and loop requests. Every taverner series
// frame is a 4x4 solid picture whose red channel names its series and green
// its frame plus one.

type kaargInnHost struct {
	art   *town.Art
	now   time.Time
	draws []int
	log   []string
}

func (h *kaargInnHost) Art() *town.Art { return h.art }
func (h *kaargInnHost) Now() time.Time { return h.now }
func (h *kaargInnHost) Draw(source string, n int) int {
	h.log = append(h.log, "draw "+source)
	if len(h.draws) == 0 {
		return 0
	}
	v := h.draws[0]
	h.draws = h.draws[1:]
	return v % n
}
func (h *kaargInnHost) Seed() int64           { return 1 }
func (h *kaargInnHost) Condition(string) bool { return false }
func (h *kaargInnHost) Value(string) int      { return -1 }
func (h *kaargInnHost) PlaySound(r town.SoundRequest) town.Voice {
	h.log = append(h.log, "play "+r.Key)
	return nil
}
func (h *kaargInnHost) StopSound(town.Voice)      {}
func (h *kaargInnHost) StartLoop(key string) bool { h.log = append(h.log, "loop "+key); return true }
func (h *kaargInnHost) StopLoop(key string)       { h.log = append(h.log, "stoploop "+key) }
func (h *kaargInnHost) Leave(string)              {}
func (h *kaargInnHost) Hook(string, string)       {}
func (h *kaargInnHost) Dialogue(string)           {}

// plays answers the sound requests logged since the last call, in order, and
// clears the log.
func (h *kaargInnHost) plays() []string {
	var out []string
	for _, l := range h.log {
		if k, ok := strings.CutPrefix(l, "play "); ok {
			out = append(out, k)
		}
	}
	h.log = nil
	return out
}

var kaargSeries = []struct {
	name string
	n    int
}{{"a1", 15}, {"a2", 25}, {"a3", 3}, {"a4", 7}, {"a5", 29}}

func kaargInnArt() *town.Art {
	a := &town.Art{Frames: map[string][]image.Image{}}
	for i, s := range kaargSeries {
		for f := 0; f < s.n; f++ {
			pic := image.NewRGBA(image.Rect(0, 0, 4, 4))
			for p := 0; p < len(pic.Pix); p += 4 {
				copy(pic.Pix[p:], []byte{uint8(i + 1), uint8(f + 1), 0, 255})
			}
			a.Frames[s.name] = append(a.Frames[s.name], pic)
		}
	}
	return a
}

// kaargInn is the shipped town 2 inn scene shown and active over h.
func kaargInn(t *testing.T, h *kaargInnHost, proc *town.Process) *town.Scene {
	t.Helper()
	d := townDescriptions["rom2-town2"]
	if d == nil || d.Scene("inn") == nil {
		t.Fatal("no town 2 inn scene")
	}
	if h.art == nil {
		h.art = kaargInnArt()
	}
	if h.now.IsZero() {
		h.now = time.Unix(100, 0)
	}
	s := town.NewScene(d, "inn", h, proc)
	s.SetActive(true, true)
	return s
}

// kaargShown paints the scene and answers the taverner series and frame it
// drew and the screen point of the frame's corner.
func kaargShown(t *testing.T, s *town.Scene) (string, int, image.Point) {
	t.Helper()
	a := s.Actor("taverner").(*town.Alternating)
	dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
	s.Paint(dst, "back")
	for y := 0; y < 480; y++ {
		for x := 160; x < 480; x++ {
			if c := dst.RGBAAt(x, y); c.A == 255 && c.B == 0 && c.R >= 1 && c.R <= 5 {
				if int(c.G)-1 != a.ShownFrame {
					t.Fatalf("drew frame %d, published frame %d", c.G-1, a.ShownFrame)
				}
				return kaargSeries[c.R-1].name, a.ShownFrame, image.Pt(x, y)
			}
		}
	}
	return "", 0, image.Point{}
}

// kman answers the Kman requests among sounds.
func kman(sounds []string) []string {
	var out []string
	for _, k := range sounds {
		if strings.Contains(k, "/kman") {
			out = append(out, k)
		}
	}
	return out
}

// The first paint draws r = ((rand*21)>>15)+1 and starts the series it maps
// to at its screen point: 1 a1, 2 a2, 5 a5, 3, 4 and 6 a3, 7..21 a4; a1
// requests Kman3 and a2 Kman2 (R2-ENGINE-277, R2-ENGINE-687).
func TestKaargInnTavernerChoosesItsSeriesFromTheDraw(t *testing.T) {
	for _, c := range []struct {
		raw   int
		want  string
		at    image.Point
		sound string
	}{
		{0, "a1", image.Pt(232, 88), "town_kaarg/inn/kman3.wav"},
		{1560, "a1", image.Pt(232, 88), "town_kaarg/inn/kman3.wav"},
		{1561, "a2", image.Pt(232, 88), "town_kaarg/inn/kman2.wav"},
		{3120, "a2", image.Pt(232, 88), "town_kaarg/inn/kman2.wav"},
		{3121, "a3", image.Pt(272, 112), ""},
		{4682, "a3", image.Pt(272, 112), ""},
		{6242, "a5", image.Pt(248, 84), ""},
		{7801, "a5", image.Pt(248, 84), ""},
		{7802, "a3", image.Pt(272, 112), ""},
		{9362, "a3", image.Pt(272, 112), ""},
		{9363, "a4", image.Pt(360, 136), ""},
		{32767, "a4", image.Pt(360, 136), ""},
	} {
		h := &kaargInnHost{draws: []int{c.raw}}
		s := kaargInn(t, h, nil)
		if got := h.plays(); fmt.Sprint(got) != "[town_kaarg/inn/kin1.wav]" {
			t.Fatalf("entry requested %v, want Kin1 once", got)
		}
		s.Advance()
		name, frame, at := kaargShown(t, s)
		var want []string
		if c.sound != "" {
			want = []string{c.sound}
		}
		got := kman(h.plays())
		if name != c.want || frame != 0 || at != c.at || fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("raw %d shows %s frame %d at %v requesting %v; want %s frame 0 at %v requesting %v",
				c.raw, name, frame, at, got, c.want, c.at, want)
		}
	}
}

// Each series steps one file on the paint more than 100 ms after the last
// step; a paint draws before it advances, so each file shows for one step;
// the paint after a series ends shows the next draw's series with no wait;
// a5 walks its table (R2-ENGINE-277).
func TestKaargInnTavernerStepsEachSeriesOnce(t *testing.T) {
	h := &kaargInnHost{draws: []int{0, 0, 0, 0, 0, 6242}}
	s := kaargInn(t, h, nil)
	s.Advance()
	if name, f, _ := kaargShown(t, s); name != "a1" || f != 0 {
		t.Fatalf("first paint shows %s frame %d", name, f)
	}
	painted := map[string][]int{}
	var order []string
	for i := 0; i < 15+29; i++ {
		h.now = h.now.Add(100 * time.Millisecond)
		s.Advance()
		name, f, _ := kaargShown(t, s)
		h.now = h.now.Add(time.Millisecond)
		s.Advance()
		if n2, f2, _ := kaargShown(t, s); n2 != name || f2 != f {
			t.Fatalf("the step paint drew %s %d, the paint 1 ms before %s %d", n2, f2, name, f)
		}
		if len(order) == 0 || order[len(order)-1] != name {
			order = append(order, name)
		}
		painted[name] = append(painted[name], f)
	}
	if fmt.Sprint(order) != "[a1 a5]" || len(painted["a1"]) != 15 || len(painted["a5"]) != 29 {
		t.Fatalf("series %v painted %v", order, painted)
	}
	for name, frames := range painted {
		for i, f := range frames {
			if f != i {
				t.Fatalf("%s painted %v, want each file once in order", name, frames)
			}
		}
	}
}

// Kvox6..8 by R(3) and Kdish1..4 by R(4) each run on their own clock of
// 2000+R(2000) ms, drawn after each request; the first paint finds both
// clocks past (R2-ENGINE-687, DIV-4271). Entry requests Kin1 and loops
// Kvox5; a pause stops the loop.
func TestKaargInnVoiceAndDishClocks(t *testing.T) {
	h := &kaargInnHost{draws: []int{
		0,            // taverner: a1
		32766, 16384, // voice: Kvox8, then 3000 ms
		32767, 0, // dish: Kdish1, then 2000 ms
		16384, 32767, // dish at 2001 ms: Kdish3, then 2000 ms
		16384, 0, // voice at 3001 ms: Kvox7, then 2000 ms
	}}
	s := kaargInn(t, h, nil)
	if got := strings.Join(h.log, ","); got != "play town_kaarg/inn/kin1.wav,loop town_kaarg/inn/kvox5.wav" {
		t.Fatalf("entry logged %q, want Kin1 then the Kvox5 loop", got)
	}
	h.log = nil
	start := h.now
	s.Advance()
	if got := fmt.Sprint(h.plays()); got != "[town_kaarg/inn/kman3.wav town_kaarg/inn/kvox8.wav town_kaarg/inn/kdish1.wav]" {
		t.Fatalf("first paint requested %s", got)
	}
	for _, c := range []struct {
		at   time.Duration
		want string
	}{
		{2000 * time.Millisecond, "[]"},
		{2001 * time.Millisecond, "[town_kaarg/inn/kdish3.wav]"},
		{3000 * time.Millisecond, "[]"},
		{3001 * time.Millisecond, "[town_kaarg/inn/kvox7.wav]"},
		{4001 * time.Millisecond, "[]"},
	} {
		h.now = start.Add(c.at)
		s.Advance()
		if got := fmt.Sprint(h.plays()); got != c.want {
			t.Errorf("at %v requested %s, want %s", c.at, got, c.want)
		}
	}
	if len(h.draws) != 0 {
		t.Errorf("%d draws left over", len(h.draws))
	}
	s.SetActive(false, true)
	if got := strings.Join(h.log, ","); got != "stoploop town_kaarg/inn/kvox5.wav" {
		t.Errorf("pause logged %q", got)
	}
}

// Leaving mid-series and entering again restarts the series at its first
// file with no start sound and no draw; a5 resumes at its stored step
// (R2-ENGINE-685).
func TestKaargInnTavernerStateOutlivesEntry(t *testing.T) {
	for _, c := range []struct {
		raw, steps int
		want       string
		frame      int
	}{{6242, 5, "a5", 5}, {1561, 4, "a2", 0}} {
		proc := &town.Process{}
		h := &kaargInnHost{draws: []int{c.raw}}
		s := kaargInn(t, h, proc)
		s.Advance()
		for i := 0; i < c.steps; i++ {
			h.now = h.now.Add(101 * time.Millisecond)
			s.Advance()
		}
		s.SetActive(false, false)
		h.log = nil
		again := kaargInn(t, h, proc)
		again.Advance()
		name, f, _ := kaargShown(t, again)
		log := strings.Join(h.log, ",")
		if name != c.want || f != c.frame || log != "play town_kaarg/inn/kin1.wav,loop town_kaarg/inn/kvox5.wav" {
			t.Errorf("re-entry after %d steps of %s shows %s frame %d logging %q; want frame %d, Kin1 and the loop only",
				c.steps, c.want, name, f, log, c.frame)
		}
	}
}

// The selected seat steps (cursor+1) mod frame count on the paint more than
// 125 ms after its last step, on the scene's kept seat clock; entry selects
// seat 0 and restarts every cursor (R2-ENGINE-686).
func TestKaargInnSeatStepsOnItsSceneClock(t *testing.T) {
	f, app, screen := secondCampaignFixtureFiles(t, false)
	c := innAt(2, 30, map[int]int32{532: 1, 552: 2})
	c.room = secondTownInn
	f.Town.second = c
	app.SetTown(screen)
	d, ok := screen.composedTown()
	if !ok || d.Town != "rom2-town2" {
		t.Fatal("no composed town 2")
	}
	h := &kaargInnHost{now: time.Unix(100, 0)}
	f.TownAnimationNow = func() time.Time { return h.now }
	proc := &town.Process{}
	s := kaargInn(t, h, proc)
	seats := d.Room("inn").Seats
	pic := image.NewRGBA(image.Rect(0, 0, 1, 1))
	frames := []image.Image{pic, pic, pic}
	screen.enterSeats()
	if list, _ := c.innSeats(); len(list) < 2 || screen.composed.seat != 0 {
		t.Fatalf("entry selected seat %d of %d", screen.composed.seat, len(list))
	}
	paint := func(d time.Duration) int {
		h.now = h.now.Add(d)
		s.Advance()
		screen.advanceSeats(s, seats, frames)
		return screen.seatFrame(0)
	}
	if got := paint(0); got != 0 || s.Clock("kaarg-seat") != h.now {
		t.Fatalf("first paint stepped to %d or left the seat clock unstamped", got)
	}
	if got := paint(125 * time.Millisecond); got != 0 {
		t.Fatal("a seat step at exactly 125 ms")
	}
	var got []int
	for i := 0; i < 3; i++ {
		got = append(got, paint(126*time.Millisecond))
	}
	if fmt.Sprint(got) != "[1 2 0]" || screen.seatFrame(1) != 0 {
		t.Fatalf("seat 0 stepped %v and seat 1 shows %d", got, screen.seatFrame(1))
	}
	paint(126 * time.Millisecond)
	stamp := s.Clock("kaarg-seat")
	s.SetActive(false, false)
	screen.enterSeats()
	again := kaargInn(t, h, proc)
	if again.Clock("kaarg-seat") != stamp || screen.seatFrame(0) != 0 || screen.composed.seat != 0 {
		t.Fatalf("re-entry: seat clock %v (kept %v), seat 0 frame %d, selected %d",
			again.Clock("kaarg-seat"), stamp, screen.seatFrame(0), screen.composed.seat)
	}
}
