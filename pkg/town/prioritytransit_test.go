package town

import (
	"image"
	"testing"
	"time"
)

// keeperScene is a priority actor with two idle states the wait picks
// between, a shared index that outlives entry, an end that lowers every
// raised state and a frame sound at index 1.
func keeperScene() obj {
	return obj{
		"random": list{obj{"name": "s", "source": "host"}},
		"art":    list{obj{"name": "base", "format": "picture", "key": "base.bmp"}, series("a1", 3), series("a2", 3)},
		"sounds": obj{"slots": list{obj{"name": "man", "source": "shop", "key": "man.wav"}}},
		"clocks": list{obj{"name": "idle", "compare": "at-least", "process": true}},
		"actors": list{obj{"name": "q", "program": "priority", "art": "base", "clock": "idle", "end": "all", "keep-index": true,
			"delay":     obj{"draw": "s", "raw": 32768, "form": "modulo", "n": 5, "times": 1000, "base": 5000},
			"idle-pick": obj{"draw": "s", "raw": 32768, "form": "scaled", "n": 2},
			"states": list{
				obj{"name": "a1", "art": "a1", "complete": 3, "steps": 4, "frame-sounds": list{obj{"frame": 1, "slot": "man"}}},
				obj{"name": "a2", "art": "a2", "complete": 3, "steps": 4, "frame-sounds": list{obj{"frame": 1, "slot": "man"}}}},
			"on": list{obj{"event": "yes", "do": "a2"}}}},
		"steps":  list{obj{"run": list{"cue q", "arm q", "advance q"}}},
		"layers": list{layer("g", "q")},
	}
}

func TestPagePriorityPicksKeepsItsIndexAndEndsAll(t *testing.T) {
	// Draws: a first stamp-only paint takes none; then the wait 5000+1000*(r%5)
	// and, when it starts, the pick (r*2/32767)%2.
	h := &fakeHost{draws: []int{7, 32766}}
	art := map[string][]image.Image{"base": pageFrames(5, 1), "a1": pageFrames(6, 3), "a2": pageFrames(7, 3)}
	p := newTestPage(t, keeperScene(), art, h)
	p.Enter()
	q := p.Actor("q").(*Priority)
	p.Advance() // stamps the process clock
	h.tick(7000 * time.Millisecond)
	p.Advance()
	if q.Running() != 1 || q.Index != 1 {
		t.Fatalf("a wait of 7000 ms with pick 1 = running %d index %d, want the second state at index 1", q.Running(), q.Index)
	}
	h.take()
	h.draws = []int{0}
	p.Advance()
	if q.Index != 2 || h.take() != "play shop:man.wav" {
		t.Fatalf("index 1's cue: index %d", q.Index)
	}
	// An entry lowers every state and keeps the index.
	p.Enter()
	if q.Running() >= 0 || q.Index != 2 {
		t.Fatalf("after entry = running %d index %d, want none at index 2", q.Running(), q.Index)
	}
	p.Event("yes")
	h.draws = []int{0}
	p.Advance()
	if q.Running() != 1 || q.Index != 3 {
		t.Fatalf("an event after entry = running %d index %d, want the inherited index advanced", q.Running(), q.Index)
	}
	dst := image.NewRGBA(image.Rect(0, 0, 4, 4))
	p.Paint(dst, "g")
	if got := dst.RGBAAt(0, 0); got.R != 7 || got.G != 3 {
		t.Fatalf("index 3 painted %v, want the second state's third frame", got)
	}
	q.Raised[0] = true // the first state runs, the second stays raised
	h.draws = []int{0}
	p.Advance()
	if q.Running() >= 0 || q.Index != 0 {
		t.Fatalf("end all = running %d index %d, want none at 0", q.Running(), q.Index)
	}
	if !p.Clock("idle").Equal(h.now) {
		t.Fatal("the end did not restamp the idle clock")
	}
}

func TestPagePriorityIdlePickLossControl(t *testing.T) {
	scene := keeperScene()
	delete(scene["actors"].(list)[0].(obj), "idle-pick")
	h := &fakeHost{draws: []int{7}}
	art := map[string][]image.Image{"base": pageFrames(5, 1), "a1": pageFrames(6, 3), "a2": pageFrames(7, 3)}
	p := newTestPage(t, scene, art, h)
	p.Enter()
	q := p.Actor("q").(*Priority)
	p.Advance()
	h.tick(6999 * time.Millisecond)
	p.Advance()
	if q.Running() >= 0 {
		t.Fatal("the state rose before its 7000 ms wait")
	}
	h.draws = []int{7}
	h.tick(time.Millisecond)
	p.Advance()
	if q.Running() != 0 {
		t.Fatalf("without a pick = running %d, want the first state", q.Running())
	}
}

func transitScene() obj {
	return obj{
		"art": list{series("lite", 2), series("burn", 2), series("cicle", 2), series("hot", 6)},
		"actors": list{obj{"name": "f", "program": "transit", "frames": 6, "value": "chosen", "toward": 3,
			"segments": list{"lite", "burn", "cicle"}, "selected-segments": list{"hot"},
			"loops": list{list{0, 1}, list{4, 5}},
			"on":    list{obj{"event": "pick", "do": "choose"}}}},
		"steps":  list{obj{"run": list{"advance f"}}},
		"layers": list{layer("g", "f")},
	}
}

func TestPageTransitLoopsAndRunsBetweenLoops(t *testing.T) {
	h := &fakeHost{values: map[string]int{"chosen": 0}}
	art := map[string][]image.Image{"lite": pageFrames(1, 2), "burn": pageFrames(2, 2), "cicle": pageFrames(3, 2), "hot": pageFrames(4, 6)}
	p := newTestPage(t, transitScene(), art, h)
	p.Enter()
	x := p.Actor("f").(*Transit)
	var cells []int
	step := func(n int) {
		for range n {
			p.Advance()
			cells = append(cells, x.Cursor)
		}
	}
	step(3)
	if p.Event("pick", 0) {
		t.Fatal("choosing the chosen value changed the transit")
	}
	p.Event("pick", 3)
	step(6)
	p.Event("pick", 1)
	step(6)
	want := []int{1, 0, 1, 2, 3, 4, 5, 4, 5, 4, 3, 2, 1, 0, 1}
	if len(cells) != len(want) {
		t.Fatalf("cells %v", cells)
	}
	for i := range want {
		if cells[i] != want[i] {
			t.Fatalf("cells %v, want %v", cells, want)
		}
	}
	p.Event("pick", 3)
	dst := image.NewRGBA(image.Rect(0, 0, 4, 4))
	p.Paint(dst, "g")
	if got := dst.RGBAAt(0, 0); got.R != 4 || got.G != 2 {
		t.Fatalf("cell 1 with the toward value painted %v, want the selected second cell", got)
	}
	p.Enter()
	if cell, selected := x.Shown(); cell != 0 || selected {
		t.Fatalf("entry = cell %d selected %v, want cell 0 unselected", cell, selected)
	}
}
