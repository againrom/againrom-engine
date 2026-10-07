package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

func TestNativeCityRosterWritesOriginalArchetype(t *testing.T) {
	for _, tc := range []struct {
		figure string
		mage   bool
		word   uint16
	}{
		{"mfighter", false, 0x21},
		{"ffighter", false, 0x22},
		{"mmage", true, 0x23},
		{"fmage", true, 0x24},
	} {
		t.Run(tc.figure, func(t *testing.T) {
			m := mapload.PartyMember{StartingHero: true, FigureDir: tc.figure, Mage: tc.mage}
			u := nativeCityUnitData(0x20, 0x10, m, m, nil, [sav.UnitStatWords]uint16{}, [sav.CharacterSkillSlots]uint16{})
			if got := binary.LittleEndian.Uint16(u.Token[17:19]); got != tc.word {
				t.Fatalf("wire type = %#x, want %#x (HERO-CLASS-013 player-character mode)", got, tc.word)
			}
		})
	}
}

func TestNativeCityArchetypeKeepsHireAndItemMarkers(t *testing.T) {
	// Hired Humans keep their table class, including a mage hire. An Item's
	// marker has a separate meaning and does not inherit its owner's class.
	for _, class := range []int32{3, 10, 24} {
		m := mapload.PartyMember{MercenaryType: 14, Class: class, Mage: true, FigureDir: "fmage"}
		u := nativeCityUnitData(0x20, 0x10, m, m, nil, [sav.UnitStatWords]uint16{}, [sav.CharacterSkillSlots]uint16{})
		if got := binary.LittleEndian.Uint16(u.Token[17:19]); got != uint16(class) {
			t.Fatalf("hire class = %#x, want %#x", got, class)
		}
	}
	item := nativeCityItemToken(0x30, 0x20, 7, 600)
	if got := binary.LittleEndian.Uint16(item[17:19]); got != 0x21 {
		t.Fatalf("item marker changed: %#x", got)
	}
}
