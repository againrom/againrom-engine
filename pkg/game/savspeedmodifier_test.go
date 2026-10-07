package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/sim"
)

func TestSavedSpeedModifierFollowsLiveSpeedEffects(t *testing.T) {
	for _, tc := range []struct {
		name            string
		word            int16
		children        []int16
		current, expect int32
	}{
		{"cast", 0, nil, 4, 4},
		{"unchanged", 4, []int16{4}, 4, 4},
		{"expired", 4, []int16{4}, 0, 0},
		{"slow after haste", 4, []int16{4}, -3, -3},
		{"equipment kept", 2, []int16{4}, 4, 2},
		{"older save without the folded word", 0, []int16{4}, 4, 4},
		{"older save expired", 0, []int16{4}, 0, 0},
		{"two effects", 1, []int16{4, -3}, 2, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, binding, _ := actorProjectionFixture(t, "Human")
			record := doc.Objects[binding.ObjectIndex-1]
			block, err := savedActorRaw(&record, "UD4", 64)
			if err != nil {
				t.Fatal(err)
			}
			binary.LittleEndian.PutUint16(block[4:], uint16(tc.word))
			var refs []uint16
			for i, magnitude := range tc.children {
				child := literalSavedEffectRecord(0xdd0000 + uint32(i))
				for name, value := range map[string]uint32{"E0C": 24 + uint32(i), "E3C": 17, "E3D": 1, "E40": uint32(uint16(magnitude)) | 100<<16} {
					savedObjectSetValue(&child, name, value)
				}
				doc.Objects = append(doc.Objects, child)
				refs = append(refs, uint16(len(doc.Objects)))
			}
			savedObjectSetRefs(&record, "Effects", refs, true)
			if err := projectSavedModifierWords(&doc, &record, map[sim.EffectKind]int32{sim.EffectSpeed: tc.current}); err != nil {
				t.Fatal(err)
			}
			block, _ = savedActorRaw(&record, "UD4", 64)
			if got := int32(int16(binary.LittleEndian.Uint16(block[4:]))); got != tc.expect {
				t.Fatalf("speed modifier word %d, want %d", got, tc.expect)
			}
		})
	}
}
