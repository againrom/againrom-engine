package town

import (
	"image"
	"testing"
	"time"
)

// keptScene is one room with the chooser of choiceScene and the keeper of
// keeperScene, both kept: one keep flag and one process store carry the
// keeper's index and the chooser's state and table step.
func keptScene() obj {
	chooser, keeper := choiceScene(), keeperScene()
	scene := obj{
		"random": chooser["random"],
		"art":    append(append(list{}, chooser["art"].(list)...), keeper["art"].(list)...),
		"sounds": obj{"slots": append(append(list{}, chooser["sounds"].(obj)["slots"].(list)...),
			keeper["sounds"].(obj)["slots"].(list)...)},
		"clocks": append(append(list{}, chooser["clocks"].(list)...), keeper["clocks"].(list)...),
		"actors": append(append(list{}, chooser["actors"].(list)...), keeper["actors"].(list)...),
		"steps":  append(append(list{}, chooser["steps"].(list)...), keeper["steps"].(list)...),
		"layers": list{layer("g", "a"), layer("h", "q")},
	}
	for _, a := range scene["actors"].(list) {
		a.(obj)["keep"] = true
	}
	return scene
}

func TestPageKeepHoldsAnIndexAndAStateInOneProcessStore(t *testing.T) {
	d, err := decodeScene(t, keptScene())
	if err != nil {
		t.Fatal(err)
	}
	art := choiceArt()
	art["base"], art["a1"], art["a2"] = pageFrames(5, 1), pageFrames(6, 3), pageFrames(7, 3)
	// The first paint only stamps the process clocks. Draws of the second:
	// the chooser's 6242 (value 5, the table state, which then steps), the
	// keeper's wait 5000+1000*(7%5) and pick (32766*2/32767)%2; of the third,
	// the keeper's next wait.
	h := &fakeHost{}
	h.art, h.now = &Art{Frames: art}, time.Unix(100, 0)
	proc := &Process{}
	p := NewScene(d, "inn", h, proc)
	p.Enter()
	p.Advance()
	h.tick(7000 * time.Millisecond)
	h.draws = []int{6242, 7, 32766}
	p.Advance()
	h.draws = []int{0}
	p.Advance()
	a, q := p.Actor("a").(*Alternating), p.Actor("q").(*Priority)
	if a.State != 3 || a.Step != 1 || q.Index != 2 {
		t.Fatalf("before leaving the chooser is at state %d step %d and the keeper at index %d; want 3, 1 and 2",
			a.State, a.Step, q.Index)
	}
	p.SetActive(false, false)
	again := NewScene(d, "inn", h, proc)
	again.Enter()
	b, r := again.Actor("a").(*Alternating), again.Actor("q").(*Priority)
	if b.State != 3 || b.Step != 1 || b.Index[2] != 0 || b.Cached[2] != 0 || r.Index != 2 || r.Running() >= 0 {
		t.Fatalf("after entry the chooser is at state %d step %d (index %d, cached %d) and the keeper at index %d running %d;"+
			" want state 3 step 1 at element 0 and index 2 with none running", b.State, b.Step, b.Index[2], b.Cached[2], r.Index, r.Running())
	}
	fresh := NewScene(d, "inn", h, &Process{})
	fresh.Enter()
	if c, s := fresh.Actor("a").(*Alternating), fresh.Actor("q").(*Priority); c.State != 0 || c.Step != 0 || s.Index != 0 {
		t.Fatalf("another process starts at state %d step %d index %d, want all zero", c.State, c.Step, s.Index)
	}
	dst := image.NewRGBA(image.Rect(0, 0, 4, 4))
	h.take()
	again.Advance()
	again.Paint(dst, "g")
	if got := dst.RGBAAt(0, 0); got.R != 3 || got.G != 1 || h.plays() != 0 {
		t.Fatalf("the kept table state painted %v with %d sounds, want table entry 1 (frame 0) and no sound", got, h.plays())
	}
}
