package town

import (
	"encoding/json"
	"image"
	"image/color"
	"strings"
	"testing"
	"time"
)

// Room pages run over a host that answers art, time, draws, sounds and named
// values. Every test picture is a solid colour whose red channel names its
// entry and green its frame plus one.

type fakePageHost struct {
	art    *Art
	now    time.Time
	draws  []int
	values map[string]int
	log    []string
}

func (h *fakePageHost) Art() *Art      { return h.art }
func (h *fakePageHost) Now() time.Time { return h.now }
func (h *fakePageHost) Draw(source string, n int) int {
	if len(h.draws) == 0 {
		return 0
	}
	v := h.draws[0]
	h.draws = h.draws[1:]
	return v % n
}
func (h *fakePageHost) Reseed(source string) { h.log = append(h.log, "reseed "+source) }
func (h *fakePageHost) PlaySound(source, key string, loop bool) Voice {
	h.log = append(h.log, "play "+source+":"+key)
	return &fakeVoice{key: key, playing: true}
}
func (h *fakePageHost) StopSound(v Voice) { h.log = append(h.log, "stop "+v.(*fakeVoice).key) }
func (h *fakePageHost) Value(name string) int {
	if v, ok := h.values[name]; ok {
		return v
	}
	return -1
}

func (h *fakePageHost) plays() int {
	n := 0
	for _, l := range h.log {
		if strings.HasPrefix(l, "play ") {
			n++
		}
	}
	return n
}

func pageFrames(entry uint8, n int) []image.Image {
	out := make([]image.Image, n)
	for i := range out {
		out[i] = solid(2, 2, color.RGBA{R: entry, G: uint8(i + 1), A: 255})
	}
	return out
}

var pageVocabulary = Vocabulary{
	Hooks:      testVocabulary.Hooks,
	Conditions: testVocabulary.Conditions,
	Values:     []string{"chosen", "class"},
	Events:     []string{"pick", "train", "yes", "change"},
}

// sceneTown is baseTown whose room carries scene.
func sceneTown(scene obj) obj {
	d := baseTown()
	d["rooms"].(list)[0].(obj)["scene"] = scene
	return d
}

func decodeScene(t *testing.T, scene obj) (*Description, error) {
	t.Helper()
	data, err := json.Marshal(sceneTown(scene))
	if err != nil {
		t.Fatal(err)
	}
	return Decode(data, pageVocabulary)
}

func newTestPage(t *testing.T, scene obj, art map[string][]image.Image, host *fakePageHost) *Page {
	t.Helper()
	d, err := decodeScene(t, scene)
	if err != nil {
		t.Fatal(err)
	}
	host.art = &Art{Frames: art}
	if host.now.IsZero() {
		host.now = time.Unix(100, 0)
	}
	p := NewPage(d, "inn", host, nil)
	if p == nil {
		t.Fatal("no page for a room with a scene")
	}
	return p
}

func series(name string, n int) obj {
	return obj{"name": name, "format": "series", "key": name + "%d.bmp", "count": n}
}

func layer(group, actor string) obj {
	return obj{"group": group, "actor": actor, "mode": "copy", "at": list{0, 0}}
}

func bounceScene() obj {
	return obj{
		"art":    list{series("d", 4)},
		"actors": list{obj{"name": "d", "program": "bounce", "art": "d", "frames": 4, "on": list{obj{"event": "train", "do": "arm"}}}},
		"steps":  list{obj{"run": list{"advance d"}}},
		"layers": list{layer("g", "d")},
	}
}

func TestPageBounceRunsOutAndBackOncePerArm(t *testing.T) {
	h := &fakePageHost{}
	p := newTestPage(t, bounceScene(), map[string][]image.Image{"d": pageFrames(9, 4)}, h)
	p.Enter()
	b := p.Actor("d").(*Bounce)
	p.Advance()
	if b.Shown() || b.Frame != 0 {
		t.Fatalf("an unarmed bounce moved: %+v", *b)
	}
	if !p.Event("train") || b.Shown() {
		t.Fatal("arming did not take, or showed before its first step")
	}
	for step, want := range []int{1, 2, 3, 2, 1, 0} {
		p.Advance()
		if b.Frame != want || b.Shown() != (want != 0) {
			t.Fatalf("step %d = frame %d shown %v, want %d", step, b.Frame, b.Shown(), want)
		}
		dst := image.NewRGBA(image.Rect(0, 0, 4, 4))
		p.Paint(dst, "g")
		if got := dst.RGBAAt(0, 0).G; b.Shown() && got != uint8(want+1) || !b.Shown() && got != 0 {
			t.Fatalf("step %d painted frame marker %d", step, got)
		}
	}
}

func TestPageCycleStampsThenCountsOnItsWaitAndOutlivesEntry(t *testing.T) {
	scene := obj{
		"actors": list{obj{"name": "c", "program": "cycle", "frames": 3,
			"wait": obj{"base-ms": 500, "compare": "at-least"}, "order": list{list{0, 1, 2}, list{2, 1, 0}}}},
		"steps": list{obj{"run": list{"advance c"}}},
	}
	h := &fakePageHost{}
	p := newTestPage(t, scene, map[string][]image.Image{}, h)
	c := p.Actor("c").(*Cycle)
	p.Advance()
	if !c.Stamped || c.Index != 0 {
		t.Fatalf("first step = %+v, want a stamp only", *c)
	}
	for _, tc := range []struct {
		after time.Duration
		index int
	}{{499 * time.Millisecond, 0}, {time.Millisecond, 1}, {time.Hour, 2}, {500 * time.Millisecond, 0}} {
		h.now = h.now.Add(tc.after)
		p.Advance()
		if c.Index != tc.index {
			t.Fatalf("after +%v index %d, want %d", tc.after, c.Index, tc.index)
		}
	}
	h.now = h.now.Add(500 * time.Millisecond)
	p.Advance()
	if c.Slot(0) != 1 || c.Slot(1) != 1 || c.Slot(-1) != 1 {
		t.Fatalf("slot of count 1 = %d %d %d", c.Slot(0), c.Slot(1), c.Slot(-1))
	}
	h.now = h.now.Add(500 * time.Millisecond)
	p.Advance()
	if c.Slot(1) != 0 {
		t.Fatalf("variant 1 maps count 2 to %d, want 0", c.Slot(1))
	}
	p.Reset()
	p.Enter()
	if c.Index != 2 || !c.Stamped {
		t.Fatal("a page entry reset the cycle")
	}
}

func TestPageSelectorSelectsReleasesStopsAndRaises(t *testing.T) {
	scene := bounceScene()
	scene["art"] = append(scene["art"].(list), series("m0", 6), series("m1", 6))
	scene["actors"] = append(scene["actors"].(list), obj{"name": "racks", "program": "selector", "frames": 6,
		"loop-at": 4, "loop-to": 1, "stop-at": 5, "release": 4, "selected": "chosen", "raise": "picked",
		"members": list{obj{"name": "r0", "art": "m0"}, obj{"name": "r1", "art": "m1"}},
		"on":      list{obj{"event": "pick", "do": "select"}}})
	scene["actors"].(list)[0].(obj)["on"] = list{obj{"event": "picked", "do": "arm"}}
	scene["steps"] = list{obj{"run": list{"advance racks", "advance d"}}}
	h := &fakePageHost{values: map[string]int{"chosen": 0}}
	art := map[string][]image.Image{"d": pageFrames(9, 4), "m0": pageFrames(1, 6), "m1": pageFrames(2, 6)}
	p := newTestPage(t, scene, art, h)
	s := p.Actor("racks").(*Selector)
	if p.Event("pick", 1) {
		t.Fatal("a page never entered answered a selection")
	}
	p.Enter()
	if s.Selected != 0 || !s.Enabled[0] || s.Enabled[1] {
		t.Fatalf("entry = %+v, want the host's member", *s)
	}
	for _, want := range []int{1, 2, 3, 1, 2} {
		p.Advance()
		if s.Index[0] != want {
			t.Fatalf("selected index %d, want %d", s.Index[0], want)
		}
	}
	if p.Event("pick", 0) {
		t.Fatal("selecting the selected member changed state")
	}
	if !p.Event("pick", 1) || s.Index[0] != 4 || s.Index[1] != 0 || !s.Enabled[1] {
		t.Fatalf("selection = %+v, want release and a fresh start", *s)
	}
	if p.Actor("d").(*Bounce).Step != 1 {
		t.Fatal("a selection did not raise its event")
	}
	p.Advance()
	if s.Enabled[0] || s.Index[1] != 1 {
		t.Fatalf("released member = %+v, want stopped at its stop frame", *s)
	}
	art["m1"][2] = nil
	dst := image.NewRGBA(image.Rect(0, 0, 4, 4))
	scene2 := p.scene.Layers
	p.scene.Layers = append(scene2, LayerSpec{Group: "r", Actor: "r1", Mode: "copy"})
	p.Paint(dst, "r")
	if got := dst.RGBAAt(0, 0); got.R != 2 || got.G != 1 {
		t.Fatalf("an incomplete member painted %v, want its first frame", got)
	}
}

func TestPagePriorityWaitsRunsAStateAndEnds(t *testing.T) {
	scene := obj{
		"random": list{obj{"name": "s", "source": "host"}},
		"art":    list{obj{"name": "base", "format": "picture", "key": "base.bmp"}, series("idle", 2), series("yes", 1)},
		"clocks": list{obj{"name": "idle", "compare": "at-least"}},
		"actors": list{obj{"name": "q", "program": "priority", "art": "base", "clock": "idle",
			"delay": obj{"draw": "s", "n": 3, "times": 1000, "base": 1000},
			"states": list{obj{"name": "idle", "art": "idle", "complete": 2, "steps": 3},
				obj{"name": "yes", "art": "yes", "complete": 1, "steps": 2}},
			"on": list{obj{"event": "yes", "do": "yes"}}}},
		"steps":  list{obj{"run": list{"arm q", "advance q"}}},
		"layers": list{layer("g", "q")},
	}
	h := &fakePageHost{draws: []int{1, 1}}
	art := map[string][]image.Image{"base": pageFrames(5, 1), "idle": pageFrames(6, 2), "yes": pageFrames(7, 1)}
	p := newTestPage(t, scene, art, h)
	p.Enter()
	q := p.Actor("q").(*Priority)
	h.now = h.now.Add(1999 * time.Millisecond)
	p.Advance()
	if q.Running() >= 0 {
		t.Fatal("the first state rose before its drawn wait")
	}
	h.now = h.now.Add(time.Millisecond)
	p.Advance()
	if q.Running() != 0 || q.Index != 1 {
		t.Fatalf("at its wait = running %d index %d", q.Running(), q.Index)
	}
	dst := image.NewRGBA(image.Rect(0, 0, 4, 4))
	p.Paint(dst, "g")
	if got := dst.RGBAAt(0, 0); got.R != 6 || got.G != 1 {
		t.Fatalf("index 1 painted %v, want the state's first frame", got)
	}
	p.Event("yes")
	p.Advance()
	p.Advance()
	if q.Running() != 1 || q.Index != 0 || !p.Clock("idle").Equal(h.now) {
		t.Fatalf("after the first state = running %d index %d", q.Running(), q.Index)
	}
}

func trainingScene() obj {
	return obj{
		"random":       list{obj{"name": "s", "source": "host"}},
		"art":          list{series("col", 4), series("atr", 3), series("am", 2), series("btr", 3), series("bm", 2)},
		"sounds":       obj{"slots": list{obj{"name": "rotate", "source": "school", "key": "r.wav", "untracked": true}}},
		"clocks":       list{obj{"name": "gate", "period-ms": 10, "compare": "greater", "rebase": "every"}},
		"enter-active": true,
		"actors": list{
			obj{"name": "col", "program": "target", "art": "col", "frames": 4, "endpoints": list{0, 3}, "value": "class",
				"wait": obj{"base-ms": 10, "compare": "greater"}},
			obj{"name": "tr", "program": "training", "clock": "gate", "column": "col", "slot": "rotate", "value": "class",
				"delay":     obj{"draw": "s", "raw": 100, "divide": 1},
				"delay-ms":  1000,
				"hold-pick": obj{"draw": "s", "raw": 100, "form": "scaled", "n": 2, "base": 2},
				"variants": list{
					obj{"name": "a", "transition": "atr", "transition-count": 3, "idle": "am", "idle-count": 2, "column-at": 1},
					obj{"name": "b", "transition": "btr", "transition-count": 3, "idle": "bm", "idle-count": 2, "column-at": 1},
				},
				"on": list{obj{"event": "change", "do": "change"}}},
		},
		"steps":  list{obj{"run": list{"advance tr"}}},
		"layers": list{layer("m", "a"), layer("m", "b"), layer("c", "col")},
	}
}

func trainingArt() map[string][]image.Image {
	return map[string][]image.Image{"col": pageFrames(1, 4), "atr": pageFrames(2, 3), "am": pageFrames(3, 2),
		"btr": pageFrames(4, 3), "bm": pageFrames(5, 2)}
}

func TestPageTrainingWaitsForTheTransitionAndWalksTheColumn(t *testing.T) {
	h := &fakePageHost{values: map[string]int{"class": 0}}
	p := newTestPage(t, trainingScene(), trainingArt(), h)
	p.Enter()
	tr, col := p.Actor("tr").(*Training), p.Actor("col").(*Target)
	if tr.Busy(p) || !tr.Sides[0].TransitionActive || !col.Ready || col.Frame != 0 {
		t.Fatalf("entry = %+v column %+v", *tr, *col)
	}
	if !p.Event("change", 0, 1) || !tr.Busy(p) || tr.WaitVariant != 1 || col.Goal != 3 || tr.ColumnStarted {
		t.Fatalf("change = %+v column %+v", *tr, *col)
	}
	h.now = h.now.Add(10 * time.Millisecond)
	p.Advance()
	if tr.Sides[1].TransitionIndex != 0 {
		t.Fatal("the gate admitted its period exactly")
	}
	h.now = h.now.Add(time.Millisecond)
	p.Advance()
	if tr.Sides[1].TransitionIndex != 1 || !tr.ColumnStarted || col.Frame != 1 || h.plays() != 1 {
		t.Fatalf("column start = %+v column %+v plays %d", *tr, *col, h.plays())
	}
	for i := 0; tr.Busy(p) && i < 10; i++ {
		h.now = h.now.Add(11 * time.Millisecond)
		p.Advance()
	}
	if tr.Busy(p) || col.Frame != 3 || h.plays() != 1 {
		t.Fatalf("settled = %+v column %+v plays %d", *tr, *col, h.plays())
	}
}

func TestPageTrainingIdlePlaysHoldsAndReturns(t *testing.T) {
	h := &fakePageHost{values: map[string]int{"class": 0}, draws: []int{0, 50}}
	scene := trainingScene()
	scene["actors"].(list)[1].(obj)["variants"].(list)[0].(obj)["idle-count"] = 3
	scene["art"].(list)[2].(obj)["count"] = 3
	art := trainingArt()
	art["am"] = pageFrames(3, 3)
	delete(art, "bm")
	p := newTestPage(t, scene, art, h)
	p.Enter()
	tr := p.Actor("tr").(*Training)
	for i := 0; tr.Sides[0].TransitionActive && i < 10; i++ {
		h.now = h.now.Add(11 * time.Millisecond)
		p.Advance()
	}
	h.now = h.now.Add(1000 * time.Millisecond)
	p.Advance()
	if tr.Sides[0].IdleActive {
		t.Fatal("idle armed at its wait exactly")
	}
	h.now = h.now.Add(time.Millisecond)
	p.Advance()
	side := &tr.Sides[0]
	if !side.IdleActive || side.IdleIndex != -1 || tr.Sides[1].IdleActive {
		t.Fatalf("idle arm = %+v, sibling without art %+v", *side, tr.Sides[1])
	}
	for _, want := range []int{0, 1, 2} {
		h.now = h.now.Add(11 * time.Millisecond)
		p.Advance()
		if side.IdleIndex != want || side.IdleCached != want {
			t.Fatalf("ascent = %+v, want %d", *side, want)
		}
	}
	h.now = h.now.Add(11 * time.Millisecond)
	p.Advance()
	if side.IdleDirection != -1 || tr.Static[0].HoldLimit != 3 {
		t.Fatalf("terminal = %+v hold %+v, want limit 2 + (50*2/99)%%2", *side, tr.Static[0])
	}
	for i := 0; i < 3; i++ {
		h.now = h.now.Add(11 * time.Millisecond)
		p.Advance()
	}
	if side.IdleIndex != 1 {
		t.Fatalf("after the hold = %+v, want the reverse begun", *side)
	}
	p.Reset()
	if !tr.Static[0].Initialized {
		t.Fatal("a page reset erased the idle timers")
	}
}

func TestPageLifecycleEntersRebasesAndSkipsWithoutArt(t *testing.T) {
	scene := obj{
		"art":    list{series("l", 3)},
		"clocks": list{obj{"name": "tick", "period-ms": 10, "compare": "at-least", "rebase": "after-first"}},
		"actors": list{obj{"name": "l", "program": "loop", "art": "l", "frames": 3, "loop": 2}},
		"steps":  list{obj{"clock": "tick", "when": "active", "run": list{"publish l", "advance l"}}},
		"layers": list{layer("g", "l")},
	}
	h := &fakePageHost{}
	p := newTestPage(t, scene, map[string][]image.Image{"l": pageFrames(4, 3)}, h)
	l := p.Actor("l").(*Loop)
	p.SetActive(true)
	if !p.Ready() || !p.Active() {
		t.Fatal("a page asked to run was not entered")
	}
	h.now = h.now.Add(9 * time.Millisecond)
	p.Advance()
	if l.Index != 0 {
		t.Fatal("the clock admitted less than its period")
	}
	h.now = h.now.Add(time.Millisecond)
	p.Advance()
	if l.Index != 1 {
		t.Fatalf("the clock did not admit its period: %+v", *l)
	}
	p.SetActive(false)
	h.now = h.now.Add(time.Hour)
	p.Advance()
	if l.Index != 1 {
		t.Fatal("a paused page advanced")
	}
	p.SetActive(true)
	h.now = h.now.Add(9 * time.Millisecond)
	p.Advance()
	if l.Index != 1 {
		t.Fatal("a resume caught up paused time")
	}
	h.now = h.now.Add(time.Millisecond)
	p.Advance()
	if l.Index != 0 {
		t.Fatalf("loop did not wrap at its loop length: %+v", *l)
	}
	h.art = nil
	h.now = h.now.Add(time.Hour)
	p.Advance()
	if l.Shown != 1 {
		t.Fatal("a page without art advanced")
	}
}

// Each scene refusal is one mutation of a valid scene and the words its error
// must carry.
func TestDecodeRefusesScenesTheComposerCannotRun(t *testing.T) {
	actor := func(d obj, i int) obj { return d["actors"].(list)[i].(obj) }
	cases := []struct {
		name   string
		mutate func(obj)
		want   string
	}{
		{"unknown program", func(d obj) { actor(d, 1)["program"] = "spiral" }, `unknown program "spiral"`},
		{"phase the program lacks", func(d obj) { d["steps"] = list{obj{"run": list{"publish tr"}}} }, `a training has no phase "publish"`},
		{"group on a clock without a period", func(d obj) {
			d["clocks"] = append(d["clocks"].(list), obj{"name": "stamp", "compare": "greater"})
			d["steps"] = list{obj{"clock": "stamp", "run": list{"advance tr"}}}
		}, "which has no period"},
		{"zero period", func(d obj) { d["clocks"].(list)[0].(obj)["period-ms"] = 0 }, "period 0 ms"},
		{"training walks no target", func(d obj) { actor(d, 1)["column"] = "tr" }, "which is no target actor"},
		{"layer with art and actor", func(d obj) { d["layers"].(list)[0].(obj)["art"] = "col" }, "names neither or both of art and actor"},
		{"unknown event", func(d obj) { actor(d, 1)["on"] = list{obj{"event": "jump", "do": "change"}} }, `unknown event "jump"`},
		{"unknown value", func(d obj) { actor(d, 0)["value"] = "nope" }, `unknown value "nope"`},
		{"action the program lacks", func(d obj) { actor(d, 1)["on"] = list{obj{"event": "change", "do": "dance"}} }, `a training has no action "dance"`},
		{"endpoint outside the frames", func(d obj) { actor(d, 0)["endpoints"] = list{0, 4} }, "endpoint 4 is outside its 4 frames"},
		{"unknown slot", func(d obj) { actor(d, 1)["slot"] = "bell" }, `unknown slot "bell"`},
		{"draw from no host source", func(d obj) { actor(d, 1)["delay"].(obj)["draw"] = "dice" }, `unknown source "dice"`},
		{"unknown art", func(d obj) {
			actor(d, 1)["variants"].(list)[0].(obj)["idle"] = "zz"
		}, `unknown art "zz"`},
		{"reverse outside the idle", func(d obj) {
			actor(d, 1)["variants"].(list)[0].(obj)["reverse-from"] = 2
		}, "reverses from 2 of 2 frames"},
	}
	if _, err := decodeScene(t, trainingScene()); err != nil {
		t.Fatalf("the base scene is refused: %v", err)
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			scene := trainingScene()
			c.mutate(scene)
			_, err := decodeScene(t, scene)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %v does not name %q", err, c.want)
			}
		})
	}
}

func TestPageAlternatingStartsTheStateOfItsWaitsParity(t *testing.T) {
	scene := obj{
		"random": list{obj{"name": "s", "source": "host"}},
		"art":    list{series("o", 3), series("e", 3)},
		"sounds": obj{"slots": list{obj{"name": "x", "source": "inn", "key": "x.wav"}}},
		"clocks": list{obj{"name": "t", "period-ms": 10, "compare": "greater"}},
		"actors": list{obj{"name": "a", "program": "alternating", "clock": "t",
			"delay": obj{"draw": "s", "n": 2, "base": 100},
			"states": list{
				obj{"name": "odd", "art": "o", "complete": 3, "when": "odd", "motion": "forward", "start-sounds": list{"x"}},
				obj{"name": "even", "art": "e", "complete": 3, "when": "even", "motion": "ping-pong"},
			}}},
		"steps":  list{obj{"run": list{"arm a", "publish a", "cue a", "advance a"}}},
		"layers": list{layer("g", "a")},
	}
	h := &fakePageHost{draws: []int{1, 0}}
	p := newTestPage(t, scene, map[string][]image.Image{"o": pageFrames(1, 3), "e": pageFrames(2, 3)}, h)
	p.Enter()
	a := p.Actor("a").(*Alternating)
	h.now = h.now.Add(101 * time.Millisecond)
	p.Advance()
	if a.State != 0 {
		t.Fatal("the state started at its wait exactly")
	}
	h.now = h.now.Add(time.Millisecond)
	p.Advance()
	if a.State != 1 || a.Index[0] != 1 || h.plays() != 1 {
		t.Fatalf("an odd wait started %+v with %d sounds, want the odd state", *a, h.plays())
	}
	h.now = h.now.Add(11 * time.Millisecond)
	p.Advance()
	h.now = h.now.Add(11 * time.Millisecond)
	p.Advance()
	if a.State != 0 || a.Index[0] != 0 || a.Cached[0] != 2 || a.Delay != 100*time.Millisecond {
		t.Fatalf("forward end = %+v, want rest with the last frame cached and an even wait", *a)
	}
}
