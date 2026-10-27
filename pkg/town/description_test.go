package town

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBaseTownDecodes(t *testing.T) {
	mustDecode(t, baseTown())
}

// Each refusal is one mutation of a valid description and the words its error
// must carry.
func TestDecodeRefusesWhatTheComposerCannotRun(t *testing.T) {
	families := func(entry list, members list, paint list) func(obj) {
		return func(d obj) {
			art := d["art"].(list)
			d["art"] = append(art, obj{"name": "w", "format": "sprites", "key": "w%d.16a", "count": 2,
				"index": list{obj{"count": 2, "base": 1}}})
			d["random"] = append(d["random"].(list), obj{"name": "g", "source": "lcg",
				"multiplier": 3, "increment": 1, "shift": 0, "mask": 32767})
			d["actors"] = append(d["actors"].(list), obj{"name": "life", "program": "families",
				"members": members, "entry": entry, "step-order": list{}, "paint-order": paint})
		}
	}
	member := func(name, mode string) obj {
		m := obj{"name": name, "art": "w", "mode": mode, "positions": list{list{0, 0}, list{1, 1}}}
		if mode == "episode" {
			m["later"] = obj{"draw": "g", "base-ms": 1, "span-ms": 2, "compare": "greater"}
			m["sheet-pick"] = obj{"draw": "g", "n": 1}
		}
		return m
	}
	pick := func(n int) obj { return obj{"draw": "g", "form": "scaled", "n": n} }
	cases := []struct {
		name   string
		mutate func(obj)
		raw    func([]byte) []byte
		want   string
	}{
		{name: "unknown hook in a step", want: `unknown hook "enter-hok"`, mutate: func(d obj) {
			d["rooms"].(list)[0].(obj)["enter"].(list)[1].(obj)["hook"] = "enter-hok"
		}},
		{name: "unknown hook in a click", want: `unknown hook "closd"`, mutate: func(d obj) {
			d["hotspots"].(list)[1].(obj)["click"].(list)[0].(obj)["else"].(list)[0].(obj)["hook"] = "closd"
		}},
		{name: "unknown condition in an if", want: `unknown condition "opn"`, mutate: func(d obj) {
			d["hotspots"].(list)[1].(obj)["click"].(list)[0].(obj)["if"] = "opn"
		}},
		{name: "unknown save condition", want: `unknown condition "admited"`, mutate: func(d obj) {
			d["save"].(obj)["admitted-when"] = "admited"
		}},
		{name: "room button one-line index past the shop", want: "room-buttons one-line index 4 outside", mutate: func(d obj) {
			d["room-buttons"] = obj{"font": "name", "ink": list{1, 2, 3}, "hover-ink": list{4, 5, 6}, "one-line": list{1, 4}}
		}},
		{name: "trailing brace", want: "trailing data", raw: func(b []byte) []byte { return append(b, '}') }},
		{name: "trailing bracket", want: "trailing data", raw: func(b []byte) []byte { return append(b, ' ', ']') }},
		{name: "trailing value", want: "trailing data", raw: func(b []byte) []byte { return append(b, []byte("{}")...) }},
		{name: "duplicate mask byte", want: "mask byte 1 is mapped twice", mutate: func(d obj) {
			m := d["mask"].(obj)
			m["bytes"] = append(m["bytes"].(list), obj{"byte": 1, "hotspot": "gate"})
		}},
		{name: "negative tip slot", want: "negative tip slot -1", mutate: func(d obj) {
			d["hotspots"].(list)[0].(obj)["tip"] = -1
		}},
		{name: "zero clock period", want: "clock period 0 ms", mutate: func(d obj) {
			d["clock"].(obj)["period-ms"] = 0
		}},
		{name: "unknown field", want: "unknown field", mutate: func(d obj) { d["colour"] = 1 }},
		{name: "unknown program", want: `unknown program "spiral"`, mutate: func(d obj) {
			d["actors"].(list)[0].(obj)["program"] = "spiral"
		}},
		{name: "unknown art key", want: `unknown art "nothing"`, mutate: func(d obj) {
			d["layers"].(list)[0].(obj)["art"] = "nothing"
		}},
		{name: "paint-order names a loop member", want: `paint-order names loop member "d"`,
			mutate: families(list{obj{"position": "b", "pick": pick(2)}, obj{"position": "d", "pick": pick(2)}},
				list{member("b", "episode"), member("d", "loop")}, list{"b", "d"})},
		{name: "unequal picks that can only be equal", want: `can only draw the value "b" holds`,
			mutate: families(list{obj{"position": "b", "pick": pick(1)}, obj{"position": "d", "unequal": "b", "pick": pick(1)}},
				list{member("b", "episode"), member("d", "loop")}, list{"b"})},
		{name: "unequal to a member drawn later", want: "no earlier entry positions",
			mutate: families(list{obj{"position": "d", "unequal": "b", "pick": pick(2)}, obj{"position": "b", "pick": pick(2)}},
				list{member("b", "episode"), member("d", "loop")}, list{"b"})},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := baseTown()
			if c.mutate != nil {
				c.mutate(d)
			}
			data, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			if c.raw != nil {
				data = c.raw(data)
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

// A families pair whose picks can differ still decodes: the refusal is only
// for a redraw that can never end.
func TestDecodeAcceptsUnequalPicksThatCanDiffer(t *testing.T) {
	d := baseTown()
	d["art"] = append(d["art"].(list), obj{"name": "w", "format": "sprites", "key": "w%d.16a", "count": 2,
		"index": list{obj{"count": 2, "base": 1}}})
	d["random"] = append(d["random"].(list), obj{"name": "g", "source": "lcg", "multiplier": 3, "increment": 1, "mask": 32767})
	d["actors"] = append(d["actors"].(list), obj{"name": "life", "program": "families",
		"members": list{obj{"name": "b", "art": "w", "mode": "loop", "positions": list{list{0, 0}}},
			obj{"name": "d", "art": "w", "mode": "loop", "positions": list{list{0, 0}, list{1, 1}}}},
		"entry": list{obj{"position": "b", "pick": obj{"draw": "g", "n": 1}},
			obj{"position": "d", "unequal": "b", "pick": obj{"draw": "g", "n": 2}}}})
	mustDecode(t, d)
}

// A refusal of a name the format does not take lists the values it does.
func TestDecodeRefusalsListTheAcceptedValues(t *testing.T) {
	for _, c := range []struct {
		name   string
		mutate func(obj)
		want   string
	}{
		{"clock compare", func(d obj) { d["clock"].(obj)["compare"] = "x" },
			`clock compare "x"; accepted: greater, at-least`},
		{"art format", func(d obj) { d["art"].(list)[0].(obj)["format"] = "" },
			`has unknown format ""; accepted: picture, series, sprites, frame, mask`},
		{"layer mode", func(d obj) { d["layers"].(list)[0].(obj)["mode"] = "" },
			`has unknown mode ""; accepted: copy, over`},
		{"square hook", func(d obj) { d["square"].(obj)["enter"].(list)[1].(obj)["hook"] = "zzz" },
			`square names unknown hook "zzz"; accepted: closed, enter-hook, leave-hook`},
		{"room page", func(d obj) { d["rooms"].(list)[0].(obj)["page"] = "zzz" },
			`names unknown page "zzz"; accepted: inn`},
		{"actor program", func(d obj) { d["actors"].(list)[0].(obj)["program"] = "zzz" },
			`has unknown program "zzz"; accepted: episode, pendulum,`},
	} {
		d := baseTown()
		c.mutate(d)
		if _, err := decodeJSON(t, d); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
}
