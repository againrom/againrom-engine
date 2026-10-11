package town

import (
	"image"
	"strings"
	"testing"
	"time"
)

// A stepper sent toward its last frame on hover starts toward its rest end,
// so entry requests no sound; hover walks it up with one sound, and leaving
// walks it back with the other.
func TestStepperTowardTheLastOnHoverStartsTowardItsRest(t *testing.T) {
	d := baseTown()
	d["art"] = append(d["art"].(list), obj{"name": "door", "format": "sprites", "key": "door.16a", "count": 4})
	d["sounds"].(obj)["slots"] = append(d["sounds"].(obj)["slots"].(list), obj{"name": "g", "source": "s"})
	d["actors"] = append(d["actors"].(list), obj{"name": "door", "program": "stepper", "art": "door", "rest": "first",
		"toward-last-on-hover": "gate", "first-sound": "shut", "last-sound": "open", "slot": "g"})
	d["step"] = obj{"admitted": list{"advance door"}}
	l := testLoader()
	l.sizes["door.16a"] = image.Pt(4, 400)
	v, h := newTestView(t, d, l)
	v.SetActive(true, true)
	door := v.Actor("door").(*Stepper)
	step := func() { h.tick(51 * time.Millisecond); v.Advance() }
	v.Advance()
	step()
	if door.Frame != 0 || h.plays() != 0 {
		t.Fatalf("unhovered: frame %d, %d sounds", door.Frame, h.plays())
	}
	v.Pointer(image.Pt(30, 3))
	var frames []int
	for i := 0; i < 4; i++ {
		step()
		frames = append(frames, door.Frame)
	}
	if got := h.take(); !equal(frames, []int{1, 2, 3, 3}) || strings.Count(got, "play s:open") != 1 || strings.Contains(got, "shut") {
		t.Fatalf("hovered: frames %v, log %q", frames, got)
	}
	v.Pointer(image.Pt(3, 3))
	step()
	if got := h.take(); door.Frame != 2 || !strings.Contains(got, "play s:shut") {
		t.Fatalf("away: frame %d, log %q", door.Frame, got)
	}
}

func equal(a, b []int) bool {
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

// A dialogue click opens its key formatted with the named value; a latch
// with no slot stops its slots and requests nothing.
func TestDialogueClickAndASilentLatch(t *testing.T) {
	d := baseTown()
	d["sounds"].(obj)["slots"] = append(d["sounds"].(obj)["slots"].(list), obj{"name": "b", "source": "s"})
	d["hotspots"].(list)[0].(obj)["click"] = list{obj{"dialogue": "talk%d", "value": "stage"}}
	d["hotspots"].(list)[0].(obj)["hover"] = list{obj{"latch": "near", "slot": "b", "sound": "near.wav"}}
	d["hotspots"].(list)[1].(obj)["click"] = list{obj{"dialogue": "plain"}}
	d["hotspots"].(list)[1].(obj)["hover"] = list{obj{"latch": "far", "stop": list{"b"}}}
	v, h := newTestView(t, d, testLoader())
	h.values = map[string]int{"stage": 30}
	v.SetActive(true, true)
	h.take()
	v.Click(v.Hotspot("door"))
	v.Click(v.Hotspot("gate"))
	if got := h.take(); got != "dialogue talk30,dialogue plain" {
		t.Fatalf("clicks: %q", got)
	}
	v.Pointer(image.Pt(3, 3))
	v.Pointer(image.Pt(30, 3))
	if got := h.take(); got != "play s:near.wav,stop near.wav" || !v.Latch("far") {
		t.Fatalf("latches: %q", got)
	}
}

// A family member indexed by sheet starts on its picked sheet, keeps its wait
// from the start so the scheduler restarts it while it runs, moves while its
// frame lies in a range, and idles on its idle sheet's first frame.
func TestFamilyMemberBySheetRestartsWhileRunningAndIdles(t *testing.T) {
	d := baseTown()
	d["art"] = append(d["art"].(list), obj{"name": "p", "format": "series", "key": "p%d%%02d.bmp",
		"index": list{obj{"count": 2, "base": 1}}, "counts": list{2, 4}, "firsts": list{0, 1}})
	d["actors"] = append(d["actors"].(list), obj{"name": "life", "program": "families",
		"members": list{obj{"name": "girl", "art": "p", "mode": "episode", "sheets": 2, "positions": list{list{1, 1}},
			"stamp": "start", "idle-sheet": 0,
			"frame-positions": list{obj{"from": 1, "to": 2, "at": list{20, 5}}},
			"later":           obj{"draw": "host", "base-ms": 120, "compare": "greater"},
			"sheet-pick":      obj{"draw": "host", "n": 2}}},
		"entry":      list{obj{"wait": "girl", "after": obj{"draw": "host", "base-ms": 40, "compare": "greater"}}},
		"step-order": list{"girl"}, "paint-order": list{"girl"}})
	d["step"] = obj{"admitted": list{"advance life"}, "after": list{"schedule life"}}
	d["layers"] = append(d["layers"].(list), obj{"actor": "girl", "mode": "copy"})
	l := testLoader()
	for _, k := range []string{"p100.bmp", "p101.bmp", "p201.bmp", "p202.bmp", "p203.bmp", "p204.bmp"} {
		l.sizes[k] = image.Pt(2, 2)
	}
	v, h := newTestView(t, d, l)
	h.draws = []int{1, 1}
	v.SetActive(true, true)
	life := v.Actor("life").(*Families)
	shown := func() string {
		sheet, frame, at, ok := life.Shown(v, "girl")
		if !ok {
			return "none"
		}
		return sheet + "@" + string(rune('0'+frame)) + " " + at.String()
	}
	if got := shown(); got != "p/0@0 (1,1)" {
		t.Fatalf("idle before any start: %s", got)
	}
	v.Advance()
	h.tick(51 * time.Millisecond)
	v.Advance()
	g := life.Member("girl")
	if !g.Active || g.Sheet != 1 || g.Current != 0 || shown() != "p/1@0 (1,1)" {
		t.Fatalf("start: %+v %s", *g, shown())
	}
	start := g.Clock
	var trail []string
	for i := 0; i < 3; i++ {
		h.tick(51 * time.Millisecond)
		v.Advance()
		trail = append(trail, shown())
	}
	if g.Clock != start.Add(153*time.Millisecond) || strings.Join(trail, ",") != "p/1@1 (20,5),p/1@2 (20,5),p/1@0 (1,1)" {
		t.Fatalf("run: %v, clock moved %v", trail, g.Clock.Sub(start))
	}
	g.Wait = time.Hour
	for i := 0; i < 4; i++ {
		h.tick(51 * time.Millisecond)
		v.Advance()
	}
	if g.Active {
		t.Fatalf("the episode outran its sheet: %+v", *g)
	}
	if got := shown(); got != "p/0@0 (1,1)" {
		t.Fatalf("idle after the episode: %s", got)
	}
	dst := image.NewRGBA(image.Rect(0, 0, 40, 30))
	v.Paint(dst, "")
	if got := dst.RGBAAt(1, 1); got != keyColour("p100.bmp") {
		t.Fatalf("idle paint %v, want the idle sheet's first frame", got)
	}
}

// The new kinds' parameters are refused when the composer cannot run them.
func TestDecodeRefusesBadDialoguesSteppersAndMembers(t *testing.T) {
	member := func(extra obj) func(obj) {
		return func(d obj) {
			d["art"] = append(d["art"].(list), obj{"name": "p", "format": "series", "key": "p%02d.bmp", "count": 2})
			m := obj{"name": "girl", "art": "p", "mode": "episode", "positions": list{list{0, 0}},
				"later": obj{"draw": "host", "base-ms": 1, "compare": "greater"}}
			for k, v := range extra {
				m[k] = v
			}
			d["actors"] = append(d["actors"].(list), obj{"name": "life", "program": "families", "members": list{m}})
		}
	}
	for _, c := range []struct {
		name, want string
		mutate     func(obj)
	}{
		{"dialogue value without a number", `opens dialogue "talk" with value "stage"`, func(d obj) {
			d["hotspots"].(list)[0].(obj)["click"] = list{obj{"dialogue": "talk", "value": "stage"}}
		}},
		{"dialogue number without a value", `opens dialogue "talk%d"`, func(d obj) {
			d["hotspots"].(list)[0].(obj)["click"] = list{obj{"dialogue": "talk%d"}}
		}},
		{"dialogue unknown value", `unknown value "stag"`, func(d obj) {
			d["hotspots"].(list)[0].(obj)["click"] = list{obj{"dialogue": "talk%d", "value": "stag"}}
		}},
		{"latch sound without a slot", `latches sound "x.wav" on no slot`, func(d obj) {
			d["hotspots"].(list)[0].(obj)["hover"] = list{obj{"latch": "l", "sound": "x.wav"}}
		}},
		{"stepper both ends", "moves toward both ends", func(d obj) {
			d["actors"] = append(d["actors"].(list), obj{"name": "door", "program": "stepper", "art": "sprite", "rest": "first",
				"toward-first-on-hover": "gate", "toward-last-on-hover": "door", "slot": "a"})
		}},
		{"unknown stamp", `unknown stamp "paint"`, member(obj{"stamp": "paint"})},
		{"idle sheet beyond its sheets", "idles on sheet 1 of 0", member(obj{"idle-sheet": 1})},
		{"backward frame range", "moves over frames 3 to 2", member(obj{"frame-positions": list{obj{"from": 3, "to": 2, "at": list{0, 0}}}})},
		{"member without a position", `member "girl" has no position`, member(obj{"positions": list{}})},
		{"sheets without a pick", "sheet-pick when it has sheets", member(obj{"sheets": 2})},
	} {
		t.Run(c.name, func(t *testing.T) {
			d := baseTown()
			c.mutate(d)
			_, err := decodeJSON(t, d)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %v, want %q", err, c.want)
			}
		})
	}
}
