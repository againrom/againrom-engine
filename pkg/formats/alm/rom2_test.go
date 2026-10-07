package alm_test

import (
	"testing"

	"againrom/pkg/formats/alm"
)

var rom2Order = []uint32{0, 1, 2, 3, 5, 11, 4, 9, 8, 6, 7, 10, 12}

func rom2Meta(w, h, count5, count4, count6, c10 uint32, c11 [3]uint32, c12 uint32, name string) []byte {
	p := make([]byte, 660)
	copy(p[0x00:], le32(w))
	copy(p[0x04:], le32(h))
	copy(p[0x08:], lef32(testAngle))
	copy(p[0x1c:], le32(count5))
	copy(p[0x20:], le32(count4))
	copy(p[0x24:], le32(count6))
	copy(p[0x30:], le32(c10))
	for i, c := range c11 {
		copy(p[0x34+4*i:], le32(c))
	}
	copy(p[0x40:], le32(c12))
	copy(p[0x44:], []byte(name))
	copy(p[0x8c:], le32(0xaaaa0001))
	copy(p[0x90:], le32(0xaaaa0002))
	copy(p[0x94:], []byte("desc"))
	return p
}

func rom2Unit(x, y uint32, class int16, sub uint16, flags, serverID, owner uint32, hp int16, unitID uint16, group uint32) []byte {
	r := make([]byte, 48)
	copy(r[0x00:], le32(x))
	copy(r[0x04:], le32(y))
	copy(r[0x08:], le16(uint16(class)))
	copy(r[0x0a:], le16(sub))
	copy(r[0x0c:], le32(flags))
	copy(r[0x14:], le32(serverID))
	copy(r[0x18:], le32(owner))
	copy(r[0x24:], le16(uint16(hp)))
	copy(r[0x28:], le16(unitID))
	copy(r[0x2c:], le32(group))
	return r
}

func rom2File(version, tag uint32, mutate func(p map[uint32][]byte)) []byte {
	p := baseSections(4, 3, 0, 0, 2)
	p[0] = rom2Meta(4, 3, 0, 0, 2, 1, [3]uint32{1, 1, 1}, 2, "second")
	p[6] = concat(
		rom2Unit(5, 6, 9, 3, 0, 2000, 1, 120, 7, 41),
		rom2Unit(7, 8, 0, 0, 0x10, 901, 2, -1, 8, 42))
	p[10] = bytesN(16, 0x51)
	p[11] = bytesN(12+84+12, 0x52)
	p[12] = bytesN(28+2*28, 0x53)
	if mutate != nil {
		mutate(p)
	}
	blobs := make([][]byte, 0, len(rom2Order))
	for _, tid := range rom2Order {
		declared := uint32(len(p[tid]))
		if tid == 0 {
			declared = 644
		}
		blobs = append(blobs, concat(recordHdr(tag, 20, declared, tid, perMapConstBits), p[tid]))
	}
	return buildFile(fileHeader(almMagic, 20, 999, uint32(len(rom2Order)), version), blobs)
}

func TestSecondGameMapDecodes(t *testing.T) {
	for _, tag := range []uint32{5, 7} {
		m, err := alm.OpenROM2(rom2File(1300, tag, nil))
		if err != nil {
			t.Fatalf("tag %d: %v", tag, err)
		}
		if m.Width != 4 || m.Height != 3 || m.Name != "second" || m.Description != "desc" {
			t.Errorf("tag %d: header %dx%d %q %q", tag, m.Width, m.Height, m.Name, m.Description)
		}
		if m.Meta.Count10 != 1 || m.Meta.Count11 != [3]uint32{1, 1, 1} || m.Meta.Count12 != 2 ||
			m.Meta.Extra != [2]uint32{0xaaaa0001, 0xaaaa0002} {
			t.Errorf("tag %d: counts %+v", tag, m.Meta)
		}
		if len(m.Units) != 2 {
			t.Fatalf("tag %d: %d units", tag, len(m.Units))
		}
		u := m.Units[0]
		if u.X != 5 || u.Y != 6 || u.ClassID != 9 || u.ClassSubID != 3 || u.ServerID != 2000 || u.Owner != 1 ||
			u.CurrentHP != 120 || !u.HasCurrentHP || u.UnitID != 7 || u.GroupID != 41 {
			t.Errorf("tag %d: unit 0 = %+v", tag, u)
		}
		v := m.Units[1]
		if v.Flags != 0x10 || v.ServerID != 901 || v.HasCurrentHP {
			t.Errorf("tag %d: unit 1 = %+v", tag, v)
		}
		for i, n := range []int{16, 108, 84} {
			if len(m.Extension[i]) != n {
				t.Errorf("tag %d: extension %d is %d bytes, want %d", tag, 10+i, len(m.Extension[i]), n)
			}
		}
	}
}

func TestSecondGameVersionRange(t *testing.T) {
	for _, v := range []uint32{1300, 1600} {
		if _, err := alm.OpenROM2(rom2File(v, 5, nil)); err != nil {
			t.Errorf("version %d: %v", v, err)
		}
	}
	for _, v := range []uint32{1001, 1299, 1601} {
		if _, err := alm.OpenROM2(rom2File(v, 5, nil)); err == nil {
			t.Errorf("version %d accepted", v)
		}
	}
}

func TestSecondGameRefusesWhatItCannotWalk(t *testing.T) {
	for name, mutate := range map[string]func(p map[uint32][]byte){
		"a short unit record": func(p map[uint32][]byte) { p[6] = p[6][:80] },
		"an extension size off its count": func(p map[uint32][]byte) {
			p[12] = p[12][:len(p[12])-1]
		},
		"a type-0 shorter than its span": func(p map[uint32][]byte) { p[0] = p[0][:632] },
	} {
		if _, err := alm.OpenROM2(rom2File(1300, 5, mutate)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestFirstGameReaderRefusesSecondGameMap(t *testing.T) {
	openReject(t, rom2File(1300, 5, nil))
	openReject(t, rom2File(1300, 7, nil))
}
