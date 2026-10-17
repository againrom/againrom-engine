package game

import (
	"strings"
	"testing"

	"againrom/internal/synth"
)

// A map with no usable drop cell sends the party to a random cell of the
// fallback square; the drop-cell rule names it when that square leaves the map
// or lands on blocked cells, and names a map with no drop cell at all as one
// with no start point.
func TestMapRuleNamesAMapWhoseFallbackDropLeavesIt(t *testing.T) {
	x := checkInspector()
	bare := func(side int, script []byte) synth.ALMOptions {
		o := cleanOptions()
		o.Width, o.Height, o.Type7Payload = side, side, script
		return o
	}
	none := make([]byte, 12)
	zero := scriptPayload([]scriptNode{{label: "drop", opcode: dropActionOpcode, id: 1, values: []uint32{0, 9}, tags: []uint32{5, 6}}}, nil)
	mixed := scriptPayload([]scriptNode{
		{label: "drop", opcode: dropActionOpcode, id: 1, values: []uint32{0, 9}, tags: []uint32{5, 6}},
		{label: "drop", opcode: dropActionOpcode, id: 2, values: []uint32{16, 16}, tags: []uint32{5, 6}},
	}, nil)
	valid := scriptPayload([]scriptNode{
		{label: "drop", opcode: dropActionOpcode, id: 1, values: []uint32{16, 16}, tags: []uint32{5, 6}},
		{label: "drop", opcode: dropActionOpcode, id: 2, values: []uint32{17, 16}, tags: []uint32{5, 6}},
	}, nil)
	for _, c := range []struct {
		name   string
		side   int
		script []byte
		want   string
	}{
		{"no drop cell on a 32 map", 32, none, "5041 of its 5041 cells"},
		{"a drop cell with a zero coordinate", 32, zero, "5041 of its 5041 cells"},
		{"a zero-coordinate cell among valid ones", 32, mixed, "1 of the map's 2 drop cells have a zero coordinate"},
		{"no drop cell on a 96 map", 96, none, "cells are outside the 96x96 map or blocked"},
		{"no drop cell on a map of 128 cells", 128, none, "the map has no start point: the party lands on a random cell of 30 to 100 on each axis"},
	} {
		rows := ofRule(findings(t, x, built(t, bare(c.side, c.script), nil, nil), "x"), "drop-cell")
		found := false
		for _, r := range rows {
			found = found || r.Error == false && strings.Contains(r.Text, c.want)
		}
		if !found {
			t.Errorf("%s: drop-cell findings %+v, want one holding %q", c.name, rows, c.want)
		}
	}
	// Loss controls: one drop cell and two.
	for name, o := range map[string]synth.ALMOptions{
		"a usable drop cell":    bare(32, scriptPayloadWithDrop(nil, nil)),
		"two usable drop cells": bare(32, valid),
	} {
		if rows := ofRule(findings(t, x, built(t, o, nil, nil), "x"), "drop-cell"); len(rows) != 0 {
			t.Errorf("%s: %+v", name, rows)
		}
	}
	// With no drop cell and an open fallback square no cell count is given.
	for _, r := range ofRule(findings(t, x, built(t, bare(128, none), nil, nil), "x"), "drop-cell") {
		if strings.Contains(r.Text, "outside") {
			t.Errorf("no drop cell on a map of 128: %+v", r)
		}
	}
	// A mixed table on an open fallback square gets no sentence.
	for _, r := range ofRule(findings(t, x, built(t, bare(128, mixed), nil, nil), "x"), "drop-cell") {
		if strings.Contains(r.Text, "have a zero coordinate") {
			t.Errorf("mixed table on a map of 128: %+v", r)
		}
	}
}
