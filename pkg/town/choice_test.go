package town

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"time"
)

// A chooser draws its state on the paint it rests, with no wait between
// series: values come from the host as raw draws of 32768, chosen through
// base 1 plus r*21/32768, the taverner's ((rand*21)>>15)+1 (R2-ENGINE-277).
func choiceScene() obj {
	return obj{
		"random": list{obj{"name": "s", "source": "host"}},
		"art":    list{series("a", 3), series("b", 2), series("c", 3)},
		"sounds": obj{"slots": list{obj{"name": "x", "source": "inn", "key": "x.wav"}}},
		"clocks": list{obj{"name": "t", "period-ms": 100, "compare": "greater", "process": true, "unstamped": "stamp"}},
		"actors": list{obj{"name": "a", "program": "alternating", "clock": "t", "keep": true,
			"choice": obj{"draw": "s", "raw": 32768, "form": "fraction", "n": 21, "base": 1},
			"states": list{
				obj{"name": "one", "art": "a", "complete": 3, "picks": list{1}, "motion": "forward", "start-sounds": list{"x"}},
				obj{"name": "two", "art": "b", "complete": 2, "picks": list{3, 4, 6}, "motion": "forward", "at": list{5, 5}},
				obj{"name": "tab", "art": "c", "complete": 3, "picks": list{5}, "motion": "table", "table": list{0, 0, 2, 1}},
			}}},
		"steps":  list{obj{"run": list{"arm a", "publish a", "advance a"}}},
		"layers": list{layer("g", "a")},
	}
}

func choiceArt() map[string][]image.Image {
	return map[string][]image.Image{"a": pageFrames(1, 3), "b": pageFrames(2, 2), "c": pageFrames(3, 3)}
}

// shown is the state and frame a chooser publishes.
func shown(a *Alternating) [2]int { return [2]int{a.ShownState, a.ShownFrame} }

func TestPageChoiceMapsRawDrawsToStates(t *testing.T) {
	d, err := decodeScene(t, choiceScene())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ r, want int }{
		{0, 1}, {1560, 1}, {1561, 2}, {6241, 4}, {6242, 5}, {7801, 5}, {7802, 6}, {32767, 21},
	} {
		h := &fakeHost{draws: []int{c.r}}
		h.art, h.now = &Art{Frames: choiceArt()}, time.Unix(100, 0)
		p := NewScene(d, "inn", h, nil)
		if got := p.amount(*d.Scene("inn").Actors[0].Choice); got != c.want {
			t.Errorf("raw %d chooses %d, want %d", c.r, got, c.want)
		}
	}
}

func TestPageChoiceRunsEachSeriesOnceInStepsOfMoreThanItsPeriod(t *testing.T) {
	h := &fakeHost{draws: []int{0, 0, 6242, 9363, 7802}}
	p := newTestPage(t, choiceScene(), choiceArt(), h)
	p.Enter()
	a := p.Actor("a").(*Alternating)
	p.Advance()
	if shown(a) != [2]int{1, 0} || h.take() != "play inn:x.wav" {
		t.Fatalf("first paint shows %v, want series one frame 0 and its start sound", shown(a))
	}
	h.tick(100 * time.Millisecond)
	p.Advance()
	if a.Index[0] != 0 {
		t.Fatal("a step at exactly the period")
	}
	var frames []int
	for i := 0; i < 3; i++ {
		h.tick(101 * time.Millisecond)
		p.Advance()
		frames = append(frames, a.ShownFrame)
	}
	if a.State != 0 || a.Cached[0] != 2 || a.Index[0] != 0 || frames[0] != 0 || frames[1] != 1 || frames[2] != 2 {
		t.Fatalf("series one painted %v and ended %+v; want 0, 1, 2 and rest on its last frame", frames, *a)
	}
	h.tick(time.Millisecond)
	h.finish()
	p.Advance()
	if shown(a) != [2]int{1, 2} || !strings.HasSuffix(h.take(), "play inn:x.wav") {
		t.Fatalf("the restarted series shows %v, want its cached last frame and its start sound", shown(a))
	}
	for i := 0; i < 3; i++ {
		h.tick(101 * time.Millisecond)
		p.Advance()
	}
	if a.State != 0 {
		t.Fatalf("series one again did not end: %+v", *a)
	}
	frames = nil
	for i := 0; i < 4; i++ {
		h.tick(101 * time.Millisecond)
		p.Advance()
		frames = append(frames, a.ShownFrame)
	}
	if fmt.Sprint(frames) != "[0 0 2 1]" || a.State != 0 || a.Step != 4 {
		t.Fatalf("the table state painted %v and ended %+v, want 0, 0, 2, 1", frames, *a)
	}
	h.tick(time.Millisecond)
	p.Advance()
	if a.State != 0 || shown(a)[0] != 0 {
		t.Fatalf("value 7 chose %+v; no state holds it, so the actor rests and draws again", *a)
	}
	h.tick(time.Millisecond)
	p.Advance()
	if shown(a) != [2]int{2, 0} || a.Step != 0 || h.plays() != 0 {
		t.Fatalf("value 6 shows %v with %d sounds, want state two silent", shown(a), h.plays())
	}
	dst := image.NewRGBA(image.Rect(0, 0, 10, 10))
	p.Paint(dst, "g")
	if r, g, _, _ := dst.At(5, 5).RGBA(); r>>8 != 2 || g>>8 != 1 {
		t.Fatalf("state two drew %d,%d at its own point", r>>8, g>>8)
	}
}

// Entry rebinds every series at its first frame; the chosen state, the table
// step and the process clock outlive it, and no start sound repeats
// (R2-ENGINE-685).
func TestPageKeptChoiceOutlivesEntry(t *testing.T) {
	d, err := decodeScene(t, choiceScene())
	if err != nil {
		t.Fatal(err)
	}
	h := &fakeHost{draws: []int{6242}}
	h.art, h.now = &Art{Frames: choiceArt()}, time.Unix(100, 0)
	proc := &Process{}
	p := NewScene(d, "inn", h, proc)
	p.Enter()
	p.Advance()
	for i := 0; i < 2; i++ {
		h.tick(101 * time.Millisecond)
		p.Advance()
	}
	a := p.Actor("a").(*Alternating)
	stamp := p.Clock("t")
	if a.State != 3 || a.Step != 2 {
		t.Fatalf("before leaving %+v, want the table state at step 2", *a)
	}
	again := NewScene(d, "inn", h, proc)
	again.Enter()
	b := again.Actor("a").(*Alternating)
	if b.State != 3 || b.Step != 2 || b.Index[0] != 0 || b.Cached[0] != 0 || again.Clock("t") != stamp {
		t.Fatalf("after entry %+v clock %v, want state 3 step 2 with every series at 0 and clock %v", *b, again.Clock("t"), stamp)
	}
	h.take()
	again.Advance()
	if shown(b) != [2]int{3, 2} || h.plays() != 0 {
		t.Fatalf("the kept table state shows %v with %d sounds, want table entry 2 and no sound", shown(b), h.plays())
	}
}

// A clock with a wait draws its period after each admission; one with no
// stamp is admitted as long past; a group with a pick runs one entry
// (R2-ENGINE-687).
func TestSceneClockDrawsItsWaitAndAGroupPicksOneEntry(t *testing.T) {
	scene := obj{
		"random": list{obj{"name": "s", "source": "host"}},
		"sounds": obj{"slots": list{
			obj{"name": "a", "source": "inn", "key": "a.wav"},
			obj{"name": "b", "source": "inn", "key": "b.wav"},
			obj{"name": "c", "source": "inn", "key": "c.wav"},
		}},
		"clocks": list{obj{"name": "v", "compare": "greater", "process": true, "unstamped": "admit",
			"wait": obj{"draw": "s", "raw": 32768, "form": "scaled", "n": 2000, "base": 2000}}},
		"steps": list{obj{"clock": "v", "pick": obj{"draw": "s", "raw": 32768, "form": "scaled", "n": 3},
			"run": list{"sound a", "sound b", "sound c"}}},
	}
	h := &fakeHost{draws: []int{32766, 16384, 16384, 0, 32767}}
	p := newTestPage(t, scene, nil, h)
	p.Enter()
	p.Advance()
	if got := h.take(); got != "play inn:c.wav" || p.Clock("v") != h.now {
		t.Fatalf("the unstamped clock's first paint played %q", got)
	}
	h.tick(3000 * time.Millisecond)
	p.Advance()
	if h.plays() != 0 {
		t.Fatal("played at exactly the drawn 3000 ms")
	}
	h.tick(time.Millisecond)
	p.Advance()
	if got := h.take(); got != "play inn:b.wav" {
		t.Fatalf("after 3001 ms played %q, want b", got)
	}
	h.tick(2001 * time.Millisecond)
	p.Advance()
	if got := h.take(); got != "play inn:a.wav" {
		t.Fatalf("after the drawn 2000 ms played %q, want a", got)
	}
}

func TestDecodeRefusesAChooserOrClockItCannotRun(t *testing.T) {
	actor := func(d obj) obj { return d["actors"].(list)[0].(obj) }
	state := func(d obj, i int) obj { return actor(d)["states"].(list)[i].(obj) }
	cases := []struct {
		name   string
		mutate func(obj)
		want   string
	}{
		{"choice with a delay", func(d obj) { actor(d)["delay"] = obj{"draw": "s", "n": 2} }, "a choice takes no delay and no modulus"},
		{"state without picks", func(d obj) { delete(state(d, 0), "picks") }, `state "one" needs picks`},
		{"picks without a choice", func(d obj) {
			delete(actor(d), "choice")
			actor(d)["delay"] = obj{"draw": "s", "n": 2}
		}, `state "one" has picks and its actor no choice`},
		{"table motion with no table", func(d obj) { delete(state(d, 2), "table") }, `state "tab" has motion "table" and 0 table entries`},
		{"table frame outside", func(d obj) { state(d, 2)["table"] = list{0, 3} }, "table frame 3 outside its 3 frames"},
		{"fraction without raw", func(d obj) { delete(actor(d)["choice"].(obj), "raw") }, "a fraction pick needs raw and n"},
		{"unknown unstamped", func(d obj) { d["clocks"].(list)[0].(obj)["unstamped"] = "skip" }, `unstamped "skip"`},
		{"unstamped clock of the scene", func(d obj) { delete(d["clocks"].(list)[0].(obj), "process") }, "but no process clock"},
		{"period and wait", func(d obj) {
			d["clocks"].(list)[0].(obj)["wait"] = obj{"draw": "s", "n": 2}
		}, "has both a period and a wait"},
		{"group pick outside its run", func(d obj) {
			d["steps"] = list{obj{"clock": "t", "pick": obj{"draw": "s", "n": 3}, "run": list{"arm a", "publish a"}}}
		}, "picks entries 0 to 2 of 2"},
	}
	if _, err := decodeScene(t, choiceScene()); err != nil {
		t.Fatalf("the base scene is refused: %v", err)
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			scene := choiceScene()
			c.mutate(scene)
			_, err := decodeScene(t, scene)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %v does not name %q", err, c.want)
			}
		})
	}
}
