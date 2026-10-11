package town

import (
	"image"
	"strings"
	"testing"
	"time"
)

// hoverTown is the base square with one family member on three sheets that
// takes sheet 2 under the door's hover, starts only while idle, and is
// placed by the sheet it draws; a highlight shows under the door's hover or
// while the member runs sheet 2.
func hoverTown() (obj, fakeLoader) {
	d := baseTown()
	d["art"] = append(d["art"].(list),
		obj{"name": "p", "format": "series", "key": "p%d%%02d.bmp", "index": list{obj{"count": 3, "base": 1}},
			"counts": list{2, 2, 3}, "first": 1},
		obj{"name": "hili", "format": "picture", "key": "hili.bmp", "size": list{2, 2}})
	d["hotspots"].(list)[0].(obj)["hover"] = list{obj{"arm": "girl", "changed": true}}
	d["actors"] = append(d["actors"].(list), obj{"name": "life", "program": "families",
		"members": list{obj{"name": "girl", "art": "p", "mode": "episode", "sheets": 3, "stamp": "start", "idle-sheet": 0,
			"start-when":      "idle",
			"hover-sheet":     obj{"hotspot": "door", "sheet": 2},
			"sheet-positions": list{list{1, 1}, list{1, 1}, list{5, 2}},
			"later":           obj{"draw": "host", "base-ms": 120, "compare": "greater"},
			"sheet-pick":      obj{"draw": "host", "n": 2}}},
		"entry":      list{obj{"wait": "girl", "after": obj{"draw": "host", "base-ms": 40, "compare": "greater"}}},
		"step-order": list{"girl"}, "paint-order": list{"girl"}})
	d["step"] = obj{"admitted": list{"advance life"}, "after": list{"schedule life"}}
	d["layers"] = append(d["layers"].(list),
		obj{"art": "hili", "mode": "copy", "at": list{30, 20},
			"when": obj{"any": list{obj{"hover": "door"}, obj{"running": "girl", "sheet": 2}}}},
		obj{"actor": "girl", "mode": "copy"})
	l := testLoader()
	for _, k := range []string{"p101.bmp", "p102.bmp", "p201.bmp", "p202.bmp", "p301.bmp", "p302.bmp", "p303.bmp", "hili.bmp"} {
		l.sizes[k] = image.Pt(2, 2)
	}
	return d, l
}

// A hover change starts the member on its hover sheet at once, at its first
// frame; hover that does not change starts nothing, and a member already on
// that sheet is not restarted. While hovered an idle member shows the hover
// sheet at that sheet's position. The highlight shows under hover and while
// the member runs the hover sheet.
func TestHoverSheetStartsAMemberAtOnce(t *testing.T) {
	d, l := hoverTown()
	v, h := newTestView(t, d, l)
	v.SetActive(true, true)
	v.Advance()
	life := v.Actor("life").(*Families)
	g := life.Member("girl")
	shown := func() string {
		sheet, frame, at, ok := life.Shown(v, "girl")
		if !ok {
			return "none"
		}
		return sheet + "@" + string(rune('0'+frame)) + " " + at.String()
	}
	if got := shown(); got != "p/0@0 (1,1)" {
		t.Fatalf("idle: %s", got)
	}
	hili := func() bool {
		dst := image.NewRGBA(image.Rect(0, 0, 40, 30))
		v.Paint(dst, "")
		return dst.RGBAAt(31, 21) == keyColour("hili.bmp")
	}
	if hili() {
		t.Fatal("highlight without hover or sheet 2")
	}
	h.tick(10 * time.Millisecond)
	v.Pointer(image.Pt(3, 3))
	if !g.Active || g.Sheet != 2 || g.Current != 0 || !g.Clock.Equal(h.now) || shown() != "p/2@0 (5,2)" || !hili() {
		t.Fatalf("hover start: %+v %s", *g, shown())
	}
	h.tick(51 * time.Millisecond)
	v.Advance()
	v.Pointer(image.Pt(4, 4))
	if g.Current != 1 {
		t.Fatalf("an unchanged hover or a running sheet restarted the member: %+v", *g)
	}
	v.Pointer(image.Pt(30, 3))
	if !hili() {
		t.Fatal("no highlight while the member runs sheet 2")
	}
	h.tick(51 * time.Millisecond)
	v.Advance()
	h.tick(51 * time.Millisecond)
	v.Advance()
	if g.Active || hili() {
		t.Fatalf("the episode did not end after its sheet: %+v", *g)
	}
	v.Pointer(image.Pt(3, 3))
	if !g.Active || g.Current != 0 {
		t.Fatalf("a changed hover did not restart the idle member: %+v", *g)
	}
	v.Pointer(image.Pt(30, 3))
	g.Active = false
	v.Pointer(image.Pt(3, 3))
	g.Active = false
	if got := shown(); got != "p/2@0 (5,2)" {
		t.Fatalf("idle under hover: %s", got)
	}
}

// An elapsed wait of a running "idle" member restamps its clock and draws its
// next wait without restarting it; an idle member started under hover takes
// the hover sheet and draws no sheet.
func TestStartWhenIdleConsumesTheWaitWhileRunning(t *testing.T) {
	d, l := hoverTown()
	v, h := newTestView(t, d, l)
	v.SetActive(true, true)
	life := v.Actor("life").(*Families)
	g := life.Member("girl")
	v.Advance()
	h.tick(41 * time.Millisecond)
	h.draws = []int{1}
	v.Advance()
	if !g.Active || g.Sheet != 1 || g.Current != 0 || g.Wait != 120*time.Millisecond || len(h.draws) != 0 {
		t.Fatalf("scheduled start: %+v", *g)
	}
	start := g.Clock
	g.Wait = 0
	h.draws = []int{5}
	h.tick(time.Millisecond)
	v.Advance()
	if !g.Active || g.Current != 0 || g.Sheet != 1 || !g.Clock.Equal(start.Add(time.Millisecond)) || g.Wait != 120*time.Millisecond || len(h.draws) != 1 {
		t.Fatalf("a running member restarted or kept its wait: %+v, draws left %v", *g, h.draws)
	}
	v.Pointer(image.Pt(3, 3))
	g.Active, g.Wait = false, 0
	h.tick(time.Millisecond)
	v.Advance()
	if !g.Active || g.Sheet != 2 || len(h.draws) != 1 {
		t.Fatalf("a hovered start: %+v, draws left %v", *g, h.draws)
	}
}

// A route waits its first wait, picks a route and moves its cursor one frame
// per admitted step from -1, drawing frame frames*route+cursor; past its
// frames it ends with the cursor at zero, and a start while it runs changes
// the route and keeps the cursor.
func TestRouteKeepsItsCursorAcrossStarts(t *testing.T) {
	d := baseTown()
	d["art"] = append(d["art"].(list), obj{"name": "bug", "format": "sprites", "key": "bug.16a", "count": 9})
	d["actors"] = append(d["actors"].(list), obj{"name": "bug", "program": "route", "art": "bug", "frames": 3,
		"first": obj{"draw": "host", "base-ms": 40, "compare": "greater"},
		"wait":  obj{"draw": "host", "base-ms": 1000, "compare": "greater"},
		"route": obj{"draw": "host", "n": 3}})
	d["step"] = obj{"admitted": list{"advance bug"}, "after": list{"schedule bug"}}
	d["layers"] = append(d["layers"].(list), obj{"actor": "bug", "mode": "copy", "at": list{0, 0}})
	l := testLoader()
	l.sizes["bug.16a"] = image.Pt(2, 900)
	v, h := newTestView(t, d, l)
	v.SetActive(true, true)
	bug := v.Actor("bug").(*Route)
	if bug.Cursor != -1 || bug.Route != -1 || bug.Active {
		t.Fatalf("entry: %+v", *bug)
	}
	v.Advance()
	h.tick(41 * time.Millisecond)
	h.draws = []int{1}
	v.Advance()
	if !bug.Active || bug.Route != 1 || bug.Cursor != -1 {
		t.Fatalf("start: %+v", *bug)
	}
	if _, ok := bug.Shown(); ok {
		t.Fatal("drawn before its first step")
	}
	var frames []int
	for i := 0; i < 3; i++ {
		h.tick(51 * time.Millisecond)
		v.Advance()
		f, ok := bug.Shown()
		if !ok {
			f = -1
		}
		frames = append(frames, f)
	}
	if !equal(frames, []int{3, 4, 5}) {
		t.Fatalf("frames %v", frames)
	}
	h.tick(51 * time.Millisecond)
	v.Advance()
	if bug.Active || bug.Cursor != 0 || bug.Route != -1 {
		t.Fatalf("end: %+v", *bug)
	}
	bug.Wait = 0
	h.draws = []int{2}
	h.tick(time.Millisecond)
	v.Advance()
	h.tick(51 * time.Millisecond)
	v.Advance()
	if f, _ := bug.Shown(); f != 7 {
		t.Fatalf("a later start drew frame %d, want 7: route 2, the kept cursor 0 then 1", f)
	}
	bug.Wait = 0
	h.draws = []int{0}
	h.tick(time.Millisecond)
	v.Advance()
	if f, _ := bug.Shown(); !bug.Active || f != 1 {
		t.Fatalf("a start while running rewound or kept the route: frame %d %+v", f, *bug)
	}
	dst := image.NewRGBA(image.Rect(0, 0, 40, 30))
	v.Paint(dst, "")
	if got := dst.RGBAAt(0, 0); got.G != 2 {
		t.Fatalf("painted %v, want frame 1", got)
	}
}

// Under a modulus an alternating actor starts the state whose residues hold
// its wait's remainder; a held state with its own point draws there until
// its second step.
func TestPageAlternatingChoosesItsStateByResidue(t *testing.T) {
	scene := obj{
		"random": list{obj{"name": "s", "source": "host"}},
		"art":    list{series("a", 3), series("b", 3), series("c", 1)},
		"clocks": list{obj{"name": "t", "period-ms": 10, "compare": "greater"}},
		"actors": list{obj{"name": "x", "program": "alternating", "clock": "t", "modulus": 4,
			"delay": obj{"draw": "s", "n": 4, "base": 100},
			"states": list{
				obj{"name": "one", "art": "a", "complete": 3, "residues": list{0}, "motion": "forward"},
				obj{"name": "two", "art": "b", "complete": 3, "residues": list{1}, "motion": "forward"},
				obj{"name": "still", "art": "c", "complete": 1, "residues": list{2, 3}, "motion": "hold", "steps": 2, "at": list{7, 8}},
			}}},
		"steps":  list{obj{"run": list{"arm x", "publish x", "advance x"}}},
		"layers": list{layer("g", "x")},
	}
	for _, c := range []struct{ draw, state int }{{0, 1}, {1, 2}, {2, 3}, {3, 3}} {
		h := &fakeHost{draws: []int{c.draw, 0}}
		p := newTestPage(t, scene, map[string][]image.Image{"a": pageFrames(1, 3), "b": pageFrames(2, 3), "c": pageFrames(3, 1)}, h)
		p.Enter()
		a := p.Actor("x").(*Alternating)
		h.now = h.now.Add(time.Duration(101+c.draw) * time.Millisecond)
		p.Advance()
		if a.State != c.state {
			t.Fatalf("wait %d ms started state %d, want %d", 100+c.draw, a.State, c.state)
		}
		if c.state != 3 {
			continue
		}
		dst := image.NewRGBA(image.Rect(0, 0, 40, 30))
		p.Paint(dst, "g")
		if dst.RGBAAt(7, 8).A == 0 || dst.RGBAAt(0, 0).A != 0 {
			t.Fatal("the still state is not drawn at its own point")
		}
		h.now = h.now.Add(11 * time.Millisecond)
		p.Advance()
		if a.State != 0 {
			t.Fatalf("a held state did not end on its second step: %+v", *a)
		}
	}
}

// A hotspot with no tip answers none.
func TestNoTipHotspot(t *testing.T) {
	d := baseTown()
	d["hotspots"].(list)[1].(obj)["tip"] = 0
	d["hotspots"].(list)[1].(obj)["no-tip"] = true
	desc := mustDecode(t, d)
	if slot, ok := desc.Hotspots[0].TipSlot(); !ok || slot != 3 {
		t.Fatalf("door tip %d %v", slot, ok)
	}
	if _, ok := desc.Hotspots[1].TipSlot(); ok {
		t.Fatal("gate shows a tip")
	}
}

// The hover-sheet, route, residue and condition parameters are refused when
// the composer cannot run them.
func TestDecodeRefusesBadHoverSheetsRoutesAndResidues(t *testing.T) {
	member := func(path string, v any) func(obj) {
		return func(d obj) {
			m := d["actors"].(list)[1].(obj)["members"].(list)[0].(obj)
			if v == nil {
				delete(m, path)
				return
			}
			m[path] = v
		}
	}
	for _, c := range []struct {
		name, want string
		mutate     func(obj)
	}{
		{"hover sheet beyond the sheets", "hovers on sheet 3 of 3", member("hover-sheet", obj{"hotspot": "door", "sheet": 3})},
		{"hover sheet of no hotspot", `unknown hotspot "well"`, member("hover-sheet", obj{"hotspot": "well", "sheet": 1})},
		{"arm without a hover sheet", `arms "girl", which has no hover sheet`, member("hover-sheet", nil)},
		{"sheet positions short", "places 2 sheets of 3", member("sheet-positions", list{list{0, 0}, list{0, 0}})},
		{"unknown start-when", `unknown start-when "busy"`, member("start-when", "busy")},
		{"running without its sheet", "running member and its sheet only together", func(d obj) {
			d["layers"].(list)[2].(obj)["when"] = obj{"running": "girl"}
		}},
		{"running no member", `"spin", which is no family member`, func(d obj) {
			d["layers"].(list)[2].(obj)["when"] = obj{"running": "spin", "sheet": 1}
		}},
		{"alternatives and a condition", "mixes alternatives", func(d obj) {
			d["layers"].(list)[2].(obj)["when"] = obj{"hover": "door", "any": list{obj{"hover": "gate"}}}
		}},
		{"tip and no tip", "has tip slot 3 and no tip", func(d obj) { d["hotspots"].(list)[0].(obj)["no-tip"] = true }},
		{"route without its waits", "a route needs frames, first, wait and route", func(d obj) {
			d["actors"] = append(d["actors"].(list), obj{"name": "bug", "program": "route", "art": "sprite", "frames": 3})
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			d, _ := hoverTown()
			c.mutate(d)
			_, err := decodeJSON(t, d)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %v, want %q", err, c.want)
			}
		})
	}
	scene := obj{
		"random": list{obj{"name": "s", "source": "host"}},
		"art":    list{series("a", 3)},
		"clocks": list{obj{"name": "t", "period-ms": 10, "compare": "greater"}},
		"actors": list{obj{"name": "x", "program": "alternating", "clock": "t", "modulus": 4,
			"delay":  obj{"draw": "s", "n": 4, "base": 100},
			"states": list{obj{"name": "one", "art": "a", "complete": 3, "residues": list{4}, "motion": "forward"}}}},
	}
	if _, err := decodeScene(t, scene); err == nil || !strings.Contains(err.Error(), "residue 4 outside modulus 4") {
		t.Fatalf("residue outside the modulus: %v", err)
	}
	scene["actors"].(list)[0].(obj)["states"].(list)[0].(obj)["when"] = "odd"
	scene["actors"].(list)[0].(obj)["states"].(list)[0].(obj)["residues"] = list{1}
	if _, err := decodeScene(t, scene); err == nil || !strings.Contains(err.Error(), "needs residues and no parity") {
		t.Fatalf("parity under a modulus: %v", err)
	}
}
