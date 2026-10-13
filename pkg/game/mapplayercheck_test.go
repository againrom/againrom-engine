package game

import (
	"encoding/binary"

	"againrom/internal/synth"
	"strings"
	"testing"
)

func TestMapReportFindsPlayerReferencesInBothNodeArrays(t *testing.T) {
	o := cleanOptions()
	o.Type7Payload = scriptPayloadWithDrop(
		[]scriptNode{{label: "action", opcode: 0xffff, id: 51, values: []uint32{99}, tags: []uint32{3}}},
		[]scriptNode{{label: "condition", opcode: 0xffff, id: 52, values: []uint32{99}, tags: []uint32{3}}},
	)
	m := built(t, o, nil, nil)
	rows := ofRule(findings(t, checkInspector(), m, "fixture.alm"), "script-reference")
	seen := map[string]bool{}
	for _, row := range rows {
		if strings.Contains(row.Text, "player 99") {
			if !row.Error || row.Index != 0 {
				t.Fatalf("player reference has wrong location or severity: %+v", row)
			}
			seen[row.Kind] = true
		}
	}
	if !seen["Action"] || !seen["Condition"] {
		t.Fatalf("missing player reference findings: %+v", rows)
	}
}

func TestMapReportUsesStructureOwnerLowWordAndWholeUnitWord(t *testing.T) {
	for _, owner := range []uint32{0x10001, 0x10000, 0x10002} {
		o := cleanOptions()
		o.Objects = []synth.ALMObject{{X: 14*256 + 128, Y: 12*256 + 128, Kind: 1, Field0E: owner, Field12: 7}}
		m := built(t, o, nil, func(b []byte, payload func(uint32) int) {
			binary.LittleEndian.PutUint32(b[payload(6)+0x14:], owner)
		})
		rows := ofRule(findings(t, checkInspector(), m, "fixture.alm"), "owner-slot")
		counts := map[string]int{}
		for _, row := range rows {
			counts[row.Kind]++
		}
		wantStructure := 1
		if owner == 0x10001 {
			wantStructure = 0
		}
		if counts["Unit"] != 1 || counts["Structure"] != wantStructure {
			t.Fatalf("owner %#x: findings %+v", owner, rows)
		}
	}
}
