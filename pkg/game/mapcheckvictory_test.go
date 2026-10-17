package game

import (
	"encoding/binary"
	"slices"
	"testing"

	"againrom/pkg/ui"
)

// A map is won only by a trigger that runs instant 4; the victory rule names a
// map whose script holds none as endless, and only such a map.
func TestMapRuleVictoryNamesAMapNothingWins(t *testing.T) {
	x := checkInspector()
	drop := scriptNode{label: "Drop location", opcode: dropActionOpcode, id: 99, values: []uint32{16, 16}, tags: []uint32{5, 6}}
	win := scriptNode{label: "Mission complete", opcode: scriptOpWin, id: winActionID}
	withTrigger := func(actions []scriptNode, acts uint32) []byte {
		b := scriptPayload(actions, nil)
		b = binary.LittleEndian.AppendUint32(b[:len(b)-4], 1)
		return append(b, winTrigger("T", acts)...)
	}
	for _, c := range []struct {
		name   string
		script []byte
		want   bool
	}{
		{"an empty script", make([]byte, 12), true},
		{"only a drop cell", scriptPayload([]scriptNode{drop}, nil), true},
		{"an instant 4 no trigger runs", scriptPayload([]scriptNode{drop, win}, nil), true},
		{"a trigger that runs another action", withTrigger([]scriptNode{drop, win}, 99), true},
		{"a trigger that runs instant 4", withTrigger([]scriptNode{drop, win}, winActionID), false},
	} {
		o := cleanOptions()
		o.Type7Payload = c.script
		rows := ofRule(findings(t, x, built(t, o, nil, nil), "x"), ui.RuleVictory)
		if got := len(rows) == 1 && !rows[0].Error && rows[0].Kind == ""; got != c.want || len(rows) > 1 {
			t.Errorf("%s: victory findings %+v, want one: %v", c.name, rows, c.want)
		}
	}
	// Loss control: without the row nothing names the endless map.
	x.rules = slices.DeleteFunc(slices.Clone(mapRules), func(r mapRule) bool { return r.ID == ui.RuleVictory })
	o := cleanOptions()
	o.Type7Payload = scriptPayload([]scriptNode{drop}, nil)
	if rows := ofRule(findings(t, x, built(t, o, nil, nil), "x"), ui.RuleVictory); len(rows) != 0 {
		t.Fatalf("a table without the victory row reports %+v", rows)
	}
}

// The start point is one rule: a map with an empty script gets one drop-cell
// finding, never a second rule's.
func TestMapRuleStartPointIsTheDropCellRule(t *testing.T) {
	x := checkInspector()
	o := cleanOptions()
	o.Type7Payload = make([]byte, 12)
	rows := findings(t, x, built(t, o, nil, nil), "x")
	if got := ofRule(rows, ui.RuleDropCell); len(got) != 1 || got[0].Error {
		t.Fatalf("drop-cell findings %+v", got)
	}
	for _, r := range rows {
		if r.Rule != ui.RuleDropCell && r.Rule != ui.RuleVictory {
			t.Errorf("a map with an empty script reports %+v", r)
		}
	}
	// Loss control: without the row nothing names the missing start point.
	x.rules = slices.DeleteFunc(slices.Clone(mapRules), func(r mapRule) bool { return r.ID == ui.RuleDropCell })
	if got := ofRule(findings(t, x, built(t, o, nil, nil), "x"), ui.RuleDropCell); len(got) != 0 {
		t.Fatalf("a table without the drop-cell row reports %+v", got)
	}
}
