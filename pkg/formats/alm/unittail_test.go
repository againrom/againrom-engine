package alm_test

import (
	"testing"

	"againrom/pkg/formats/alm"
)

// TestUnitTailDecode reads every override byte from the file offset the
// read map gives it (ALM-TAILMAP-079), and WithoutTail restores the absent
// values.
func TestUnitTailDecode(t *testing.T) {
	rec := type6RecordHealth(0x0180, 0x0280, 0x0007, 0x0011, 0x84, 0, 77)
	copy(rec[0x24:], le16(uint16(0xfffb))) // mana -5
	for i, b := range []byte{15, 41, 32, 20} {
		rec[0x2c+i] = b
	}
	rec[0x30], rec[0x35] = 9, 9 // read by no spawner instruction
	rec[0x31], rec[0x32] = 7, 240
	for i := 0; i < 5; i++ {
		rec[0x36+i] = byte(10 + i)
		rec[0x3b+i] = byte(90 + i)
	}
	p := baseSections(2, 2, 0, 0, 1)
	p[6] = rec
	u := openOK(t, buildMap(p)).Units[0]
	want := alm.UnitTail{Body: 15, Mind: 41, Spirit: 32, Reaction: 20, Mana: -5, HasMana: true,
		Defence: 7, Absorption: 240, Skill: [5]uint8{10, 11, 12, 13, 14},
		Protection: [5]uint8{90, 91, 92, 93, 94}, SkipGear: true}
	if u.Tail != want || u.CurrentHP != 77 || !u.HasCurrentHP {
		t.Fatalf("tail = %+v hp %d/%v, want %+v hp 77", u.Tail, u.CurrentHP, u.HasCurrentHP, want)
	}
	bare := u.WithoutTail()
	if bare.Tail != (alm.UnitTail{Mana: -1}) || bare.HasCurrentHP || bare.CurrentHP != -1 || bare.Flags != 0x04 {
		t.Fatalf("WithoutTail = %+v", bare)
	}
}
