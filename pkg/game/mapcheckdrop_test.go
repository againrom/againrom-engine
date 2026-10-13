package game

import (
	"strings"
	"testing"

	"againrom/internal/synth"
)

// A map with no usable drop cell sends the party to a random cell of the
// fallback square; the drop-cell rule names it when that square leaves the map
// or lands on blocked cells.
func TestMapRuleNamesAMapWhoseFallbackDropLeavesIt(t *testing.T) {
	x := checkInspector()
	bare := func(side int, script []byte) synth.ALMOptions {
		o := cleanOptions()
		o.Width, o.Height, o.Type7Payload = side, side, script
		return o
	}
	none := make([]byte, 12)
	zero := scriptPayload([]scriptNode{{label: "drop", opcode: dropActionOpcode, id: 1, values: []uint32{0, 9}, tags: []uint32{5, 6}}}, nil)
	for _, c := range []struct {
		name   string
		side   int
		script []byte
		want   string
	}{
		{"no drop cell on a 32 map", 32, none, "5041 of its 5041 cells"},
		{"a drop cell with a zero coordinate", 32, zero, "5041 of its 5041 cells"},
		{"no drop cell on a 96 map", 96, none, "cells are outside the 96x96 map or blocked"},
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
	// Loss controls: a drop cell, and a map whose fallback square is open.
	for name, o := range map[string]synth.ALMOptions{
		"a usable drop cell":                 bare(32, scriptPayloadWithDrop(nil, nil)),
		"no drop cell on a map of 128 cells": bare(128, none),
	} {
		if rows := ofRule(findings(t, x, built(t, o, nil, nil), "x"), "drop-cell"); len(rows) != 0 {
			t.Errorf("%s: %+v", name, rows)
		}
	}
}
