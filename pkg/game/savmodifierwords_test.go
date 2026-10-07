package game

import (
	"encoding/binary"
	"fmt"
	"testing"

	"againrom/pkg/sim"
)

// Every modifier-backed effect kind keeps the speed rule: the word is its
// remaining part plus the live effect sum, and a zero word beside saved
// effects of the kind is an older save that never folded them.
func TestSavedModifierWordsFollowLiveEffectsPerKind(t *testing.T) {
	for _, m := range savedModifierEffects {
		for _, tc := range []struct {
			name            string
			word            int16
			children        []int16
			current, expect int32
		}{
			{"cast", 0, nil, 4, 4},
			{"unchanged", 4, []int16{4}, 4, 4},
			{"expired", 4, []int16{4}, 0, 0},
			{"equipment kept", 2, []int16{4}, 4, 2},
			{"older save without the folded word", 0, []int16{4}, 4, 4},
			{"older save expired", 0, []int16{4}, 0, 0},
		} {
			t.Run(fmt.Sprintf("key%d/%s", m.key, tc.name), func(t *testing.T) {
				doc, binding, _ := actorProjectionFixture(t, "Human")
				record := doc.Objects[binding.ObjectIndex-1]
				block, err := savedActorRaw(&record, "UD4", 64)
				if err != nil {
					t.Fatal(err)
				}
				binary.LittleEndian.PutUint16(block[m.offset:], uint16(int32(tc.word)*m.scale))
				var refs []uint16
				for i, magnitude := range tc.children {
					child := literalSavedEffectRecord(0xee0000 + uint32(i))
					for name, value := range map[string]uint32{"E0C": 24 + uint32(i), "E3C": m.key, "E3D": 1, "E40": uint32(uint16(magnitude)) | 100<<16} {
						savedObjectSetValue(&child, name, value)
					}
					doc.Objects = append(doc.Objects, child)
					refs = append(refs, uint16(len(doc.Objects)))
				}
				savedObjectSetRefs(&record, "Effects", refs, true)
				if err := projectSavedModifierWords(&doc, &record, map[sim.EffectKind]int32{m.kind: tc.current}); err != nil {
					t.Fatal(err)
				}
				block, _ = savedActorRaw(&record, "UD4", 64)
				if got := int32(int16(binary.LittleEndian.Uint16(block[m.offset:]))); got != tc.expect*m.scale {
					t.Fatalf("modifier word at +%#x is %d, want %d", m.offset, got, tc.expect*m.scale)
				}
			})
		}
	}
}
