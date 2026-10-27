package town

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	"againrom/pkg/screenart"
)

func newTestView(t *testing.T, d obj, loader fakeLoader) (*Scene, *fakeHost) {
	t.Helper()
	desc := mustDecode(t, d)
	art, err := LoadArt(desc.SquareScene(), screenart.NewLibrary(loader))
	if err != nil {
		t.Fatal(err)
	}
	h := &fakeHost{art: art, now: time.Unix(100, 0), conditions: map[string]bool{}}
	return NewScene(desc, desc.Square.Name, h, nil), h
}

func TestLoadArtRefusesARequiredEntryAndSkipsAnOptionalOne(t *testing.T) {
	desc := mustDecode(t, baseTown())
	l := testLoader()
	delete(l.sizes, "sprite.16a")
	art, err := LoadArt(desc.SquareScene(), screenart.NewLibrary(l))
	if err != nil {
		t.Fatal(err)
	}
	if art.Pictures("sprite") != nil || len(art.Problems) != 1 {
		t.Fatalf("optional sprite: frames %v problems %v", art.Pictures("sprite"), art.Problems)
	}
	delete(l.sizes, "base.bmp")
	if _, err := LoadArt(desc.SquareScene(), screenart.NewLibrary(l)); err == nil || !strings.Contains(err.Error(), "base.bmp") {
		t.Fatalf("required base: %v", err)
	}
}

func TestMaskHotspotsClickAndRooms(t *testing.T) {
	v, h := newTestView(t, baseTown(), testLoader())
	if got := v.HotspotAt(image.Pt(3, 3)); got == nil || got.Name != "door" {
		t.Fatalf("left half: %v", got)
	}
	if got := v.HotspotAt(image.Pt(30, 3)); got == nil || got.Name != "gate" {
		t.Fatalf("right half: %v", got)
	}
	if v.HotspotAt(image.Pt(50, 3)) != nil {
		t.Fatal("outside the mask answered a hotspot")
	}
	if menu := v.Click(v.Hotspot("gate")); menu != "" || h.take() != "hook closed@" {
		t.Fatalf("closed gate: menu %q", menu)
	}
	h.conditions["open"] = true
	if menu := v.Click(v.Hotspot("gate")); menu != "m" {
		t.Fatalf("open gate: menu %q", menu)
	}
	v.Click(v.Hotspot("door"))
	if got := h.take(); got != "hook enter-hook@inn" {
		t.Fatalf("door: %q", got)
	}
	v.LeaveRoom("inn")
	if got := h.take(); got != "hook leave-hook@inn,hook leave-hook@square" {
		t.Fatalf("leave: %q", got)
	}
	if v.SaveAdmitted() {
		t.Fatal("save admitted without its condition")
	}
}

func TestEpisodeArmsOnHoverAndRunsOnTheClock(t *testing.T) {
	v, h := newTestView(t, baseTown(), testLoader())
	v.SetActive(true, true)
	ep := v.Actor("spin").(*Episode)
	v.Advance() // stamps the clock
	v.Pointer(image.Pt(3, 3))
	if !ep.Enabled {
		t.Fatal("hover did not arm")
	}
	h.tick(50 * time.Millisecond)
	v.Advance()
	if ep.Frame != 0 {
		t.Fatalf("a step at exactly the period ran: frame %d", ep.Frame)
	}
	h.tick(time.Millisecond)
	v.Advance()
	if ep.Frame != 1 || !strings.Contains(h.take(), "play s:spin.wav") {
		t.Fatalf("first step: frame %d", ep.Frame)
	}
	for i := 0; i < 2; i++ {
		h.tick(51 * time.Millisecond)
		v.Advance()
	}
	if ep.Frame != 0 || ep.Enabled {
		t.Fatalf("after the last frame: frame %d enabled %v", ep.Frame, ep.Enabled)
	}
	dst := image.NewRGBA(image.Rect(0, 0, 40, 30))
	v.Paint(dst, "")
	if dst.RGBAAt(2, 3).R != 9 || dst.RGBAAt(2, 3).G != 1 {
		t.Fatalf("episode frame 0 not painted at its layer point: %v", dst.RGBAAt(2, 3))
	}
}

func TestPauseSilencesAndResumeRestartsTheLoop(t *testing.T) {
	d := baseTown()
	d["sounds"].(obj)["loop"] = "crowd.wav"
	v, h := newTestView(t, d, testLoader())
	v.SetActive(true, true)
	if got := h.take(); got != "loop crowd.wav" {
		t.Fatalf("activation: %q", got)
	}
	v.SetActive(false, true)
	if got := h.take(); got != "stoploop crowd.wav" || v.Hover() != nil {
		t.Fatalf("pause: %q", got)
	}
	v.SetActive(true, false)
	if got := h.take(); got != "leave" || v.Present() {
		t.Fatalf("off the square: %q", got)
	}
}

func steppers() obj {
	d := baseTown()
	d["art"] = append(d["art"].(list), obj{"name": "door", "format": "sprites", "key": "door.16a", "count": 4})
	d["sounds"].(obj)["slots"] = append(d["sounds"].(obj)["slots"].(list), obj{"name": "g", "source": "s"})
	d["actors"] = append(d["actors"].(list),
		obj{"name": "door", "program": "stepper", "art": "door", "rest": "last", "hold": "last", "hold-unless": "open",
			"toward-first-on-hover": "gate", "first-sound": "up", "last-sound": "down", "slot": "g"},
		obj{"name": "guard", "program": "driven", "art": "door", "rest": "last", "first-sound": "g1", "last-sound": "g2", "slot": "g"})
	d["step"] = obj{"admitted": list{"advance door", "advance guard"}}
	d["pointer"] = obj{"every": list{obj{"drive": "guard", "dir": 1}}}
	d["hotspots"].(list)[1].(obj)["hover"] = list{obj{"drive": "guard", "dir": -1, "unless": "open"}}
	return d
}

func TestStepperHoldsUntilItsConditionAndWalksTowardTheHover(t *testing.T) {
	l := testLoader()
	l.sizes["door.16a"] = image.Pt(4, 400)
	v, h := newTestView(t, steppers(), l)
	v.SetActive(true, true)
	door := v.Actor("door").(*Stepper)
	if door.Frame != 3 {
		t.Fatalf("rest frame %d, want the last", door.Frame)
	}
	v.Advance()
	v.Pointer(image.Pt(30, 3))
	step := func() { h.tick(51 * time.Millisecond); v.Advance() }
	step()
	if door.Frame != 3 {
		t.Fatalf("held door moved to %d", door.Frame)
	}
	h.conditions["open"] = true
	h.take()
	step()
	if door.Frame != 2 || !strings.Contains(h.take(), "play s:up") {
		t.Fatalf("open door toward the hover: frame %d", door.Frame)
	}
	step()
	step()
	step()
	if door.Frame != 0 {
		t.Fatalf("door clamps at its first frame: %d", door.Frame)
	}
}

func TestDrivenWalksInTheDrivenDirectionAndClamps(t *testing.T) {
	l := testLoader()
	l.sizes["door.16a"] = image.Pt(4, 400)
	v, h := newTestView(t, steppers(), l)
	v.SetActive(true, true)
	guard := v.Actor("guard").(*Driven)
	v.Advance()
	v.Pointer(image.Pt(30, 3)) // every: +1, then gate: -1 because the gate is shut
	if guard.Dir != -1 {
		t.Fatalf("dir %d", guard.Dir)
	}
	for i := 0; i < 5; i++ {
		h.tick(51 * time.Millisecond)
		v.Advance()
	}
	if guard.Frame != 0 || guard.Dir != 0 {
		t.Fatalf("guard frame %d dir %d after clamping", guard.Frame, guard.Dir)
	}
}

func TestPendulumSwingsUnderOneEnable(t *testing.T) {
	d := baseTown()
	d["actors"] = append(d["actors"].(list), obj{"name": "pair", "program": "pendulum",
		"members": list{obj{"name": "f", "art": "sprite"}, obj{"name": "m", "art": "sprite"}},
		"chance":  obj{"draw": "host", "n": 100, "above": 50}})
	d["step"] = obj{"admitted": list{"advance pair"}}
	d["hotspots"].(list)[0].(obj)["hover"] = list{obj{"arm": "pair"}}
	v, h := newTestView(t, d, testLoader())
	v.SetActive(true, true)
	p := v.Actor("pair").(*Pendulum)
	v.Advance()
	v.Pointer(image.Pt(3, 3))
	// f starts (99>50), m waits (0); then both run to the end and back.
	h.draws = []int{99, 0, 99, 99, 99, 99, 99, 99}
	step := func() { h.tick(51 * time.Millisecond); v.Advance() }
	step()
	if p.Member("f").Frame != 1 || p.Member("m").Frame != 0 {
		t.Fatalf("first swing f %+v m %+v", *p.Member("f"), *p.Member("m"))
	}
	for i := 0; i < 6 && p.Enabled; i++ {
		step()
	}
	if p.Enabled {
		t.Fatal("the pendulum never cleared its shared enable")
	}
}

// TestGeneratorsLastTheProcess pins the description's generators to the
// process: a second scene over the same Process continues the first one's
// sequence rather than seeding again, and a replacement raw source holds.
func TestGeneratorsLastTheProcess(t *testing.T) {
	d := baseTown()
	d["random"] = append(d["random"].(list), obj{"name": "g", "source": "lcg", "multiplier": 3, "increment": 1, "mask": 32767})
	desc := mustDecode(t, d)
	art, err := LoadArt(desc.SquareScene(), screenart.NewLibrary(testLoader()))
	if err != nil {
		t.Fatal(err)
	}
	scene := func(proc *Process) *Scene {
		return NewScene(desc, desc.Square.Name, &fakeHost{art: art, now: time.Unix(100, 0), conditions: map[string]bool{}}, proc)
	}
	draws := func(v *Scene) []int {
		var out []int
		for range 6 {
			out = append(out, v.Pick("g", "scaled", 1000))
		}
		return out
	}
	proc, control := &Process{}, &Process{}
	first := draws(scene(proc))
	draws(scene(control))
	if got, want := draws(scene(proc)), draws(scene(control)); fmt.Sprint(got) != fmt.Sprint(want) || fmt.Sprint(got) == fmt.Sprint(first) {
		t.Fatalf("a second scene drew %v, want the continuation %v, not the first draws %v", got, want, first)
	}
	v := scene(proc)
	v.SetRawDraw("g", func() int { return 500 })
	if got := v.Pick("g", "scaled", 1000); got != 500*1000/32767%1000 {
		t.Fatalf("the replacement raw source was dropped: %d", got)
	}
}
