package town

import (
	"encoding/json"
	"image"
	"strings"
	"testing"
)

// seatTown is baseTown whose room is a page with page art, a scene with one
// keyed slot, a three-by-two seat grid and two buttons.
func seatTown() obj {
	d := baseTown()
	room := d["rooms"].(list)[0].(obj)
	room["art"] = list{
		obj{"name": "back", "format": "picture", "key": "back.bmp"},
		obj{"name": "actor", "format": "sprites", "key": "actor.16a"},
		obj{"name": "off", "format": "picture", "key": "off.bmp"},
		obj{"name": "on", "format": "picture", "key": "on.bmp"},
	}
	room["scene"] = obj{"sounds": obj{"slots": list{obj{"name": "pick", "source": "s", "key": "pick.wav"}}}}
	room["seats"] = obj{"first": list{10, 50, 20, 60}, "step": list{10, 0}, "row-step": list{0, -10},
		"columns": 3, "rows": 2, "back": "back", "actor": "actor", "select": "pick", "frame-ms": 125, "compare": "greater"}
	room["buttons"] = list{
		obj{"name": "talk", "rect": list{40, 0, 50, 10}, "off": "off", "on": "on", "caption": 7},
		obj{"name": "exit", "rect": list{40, 10, 50, 20}, "off": "off"},
	}
	return d
}

// The seat grid steps right within a row and by the row step between rows;
// a point finds the first populated seat holding it.
func TestSeatGridGeometry(t *testing.T) {
	d := mustDecode(t, seatTown())
	s := d.Rooms[0].Seats
	for i, want := range []image.Rectangle{
		image.Rect(10, 50, 20, 60), image.Rect(20, 50, 30, 60), image.Rect(30, 50, 40, 60),
		image.Rect(10, 40, 20, 50), image.Rect(20, 40, 30, 50), image.Rect(30, 40, 40, 50),
	} {
		if got := s.Seat(i); got != want {
			t.Errorf("seat %d = %v, want %v", i, got, want)
		}
	}
	if !s.Seat(6).Empty() || !s.Seat(-1).Empty() {
		t.Error("a seat outside the grid has a rectangle")
	}
	if got := s.SeatAt(image.Pt(25, 45), 6); got != 4 {
		t.Errorf("seat at (25,45) = %d, want 4", got)
	}
	if got := s.SeatAt(image.Pt(25, 45), 4); got != -1 {
		t.Errorf("an unpopulated seat answered %d", got)
	}
	b := d.Rooms[0].Buttons
	if ButtonAt(b, image.Pt(45, 15)) != 1 || ButtonAt(b, image.Pt(5, 5)) != -1 || *b[0].Caption != 7 || b[1].Caption != nil {
		t.Error("buttons do not answer their rectangles and captions")
	}
}

// Each refusal is one mutation of a valid description and the words its
// error must carry: page controls, page kinds, and numbers the composer
// would otherwise run silently wrong.
func TestDecodeRefusesBadControlsAndNumbers(t *testing.T) {
	room := func(d obj) obj { return d["rooms"].(list)[0].(obj) }
	flock := func(d obj) obj {
		d["art"] = append(d["art"].(list), obj{"name": "birds", "format": "sprites", "key": "b%d.16a", "count": 4,
			"index": list{obj{"count": 2, "base": 1}}})
		a := obj{"name": "birds", "program": "flock", "art": "birds", "groups": 1, "group-size": 2, "frames": 4, "slot": "a",
			"wait":  obj{"draw": "host", "base-ms": 10, "span-ms": 5, "compare": "greater"},
			"group": obj{"draw": "host", "n": 1}, "count": obj{"draw": "host", "n": 2, "base": 1}}
		d["actors"] = append(d["actors"].(list), a)
		return a
	}
	cases := []struct {
		name   string
		mutate func(obj)
		want   string
	}{
		{"unknown page kind", func(d obj) { room(d)["page"] = "inm" }, `unknown page "inm"`},
		{"seat art unknown", func(d obj) { room(d)["seats"].(obj)["back"] = "bak" }, `unknown page art "bak"`},
		{"seat sound not keyed", func(d obj) { room(d)["seats"].(obj)["select"] = "nothing" }, `no keyed slot`},
		{"seats without columns", func(d obj) { room(d)["seats"].(obj)["columns"] = 0 }, "seats need columns"},
		{"seats without a period", func(d obj) { room(d)["seats"].(obj)["frame-ms"] = 0 }, "frame period"},
		{"seat compare typo", func(d obj) { room(d)["seats"].(obj)["compare"] = "greatr" }, "frame period"},
		{"empty pane", func(d obj) { room(d)["pane"] = obj{"rect": list{480, 238, 480, 480}} }, "pane rectangle"},
		{"empty button", func(d obj) { room(d)["buttons"].(list)[1].(obj)["rect"] = list{1, 1, 1, 9} }, "is empty"},
		{"repeated button", func(d obj) { room(d)["buttons"].(list)[1].(obj)["name"] = "talk" }, "repeated"},
		{"button art unknown", func(d obj) { room(d)["buttons"].(list)[1].(obj)["on"] = "of" }, `unknown page art "of"`},
		{"wait compare typo", func(d obj) { flock(d)["wait"].(obj)["compare"] = "at-lest" }, `wait compare "at-lest"`},
		{"wait form typo", func(d obj) { flock(d)["wait"].(obj)["form"] = "scald" }, `wait form "scald"`},
		{"pick form typo", func(d obj) { flock(d)["group"].(obj)["form"] = "scald" }, `pick form "scald"`},
		{"negative pick base", func(d obj) { flock(d)["count"].(obj)["base"] = -1 }, "negative pick"},
		{"negative wait", func(d obj) { flock(d)["wait"].(obj)["base-ms"] = -5 }, "negative wait"},
		{"wait from an unknown source", func(d obj) { flock(d)["wait"].(obj)["draw"] = "hst" }, `unknown source "hst"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := seatTown()
			c.mutate(d)
			data, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			_, err = Decode(data, testVocabulary)
			if err == nil {
				t.Fatalf("decoded; want an error naming %q", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %q does not name %q", err, c.want)
			}
		})
	}
}

// A selector that never loops back or stops, and frame cues beyond their
// art, are refused.
func TestDecodeRefusesRunawayIndices(t *testing.T) {
	selector := obj{"name": "racks", "program": "selector", "frames": 4, "loop-at": 3, "loop-to": 1, "stop-at": 4,
		"release": 2, "members": list{obj{"name": "rack", "art": "rack"}}}
	scene := func(mutate func(a obj)) obj {
		a := obj{}
		for k, v := range selector {
			a[k] = v
		}
		mutate(a)
		return obj{"art": list{obj{"name": "rack", "format": "series", "key": "r%d.bmp", "count": 4}}, "actors": list{a}}
	}
	if _, err := decodeScene(t, scene(func(obj) {})); err != nil {
		t.Fatal("the control selector does not decode:", err)
	}
	for name, mutate := range map[string]func(obj){
		"no stop":      func(a obj) { a["stop-at"] = 0 },
		"no loop":      func(a obj) { a["loop-at"], a["loop-to"] = 0, 0 },
		"loop forward": func(a obj) { a["loop-to"] = 3 },
	} {
		if _, err := decodeScene(t, scene(mutate)); err == nil || !strings.Contains(err.Error(), "must loop back and stop") {
			t.Errorf("%s: %v", name, err)
		}
	}
	tender := obj{
		"random": list{obj{"name": "r", "source": "host"}},
		"art":    list{obj{"name": "drink", "format": "series", "key": "d%d.bmp", "count": 4}},
		"sounds": obj{"slots": list{obj{"name": "glug", "source": "s", "key": "g.wav"}}},
		"clocks": list{obj{"name": "c", "period-ms": 80, "compare": "greater"}},
		"actors": list{obj{"name": "tender", "program": "alternating", "clock": "c",
			"delay": obj{"draw": "r", "raw": 100, "divide": 2, "base": 10},
			"states": list{obj{"name": "drink", "art": "drink", "complete": 4, "when": "odd", "motion": "forward",
				"frame-sounds": list{obj{"frame": 4, "slot": "glug"}}}}}},
	}
	if _, err := decodeScene(t, tender); err == nil || !strings.Contains(err.Error(), "cues frame 4 beyond its 4 frames") {
		t.Errorf("a state cue beyond its frames: %v", err)
	}
}
