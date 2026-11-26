package a2c

import (
	"encoding/binary"
	"errors"
	"math/bits"
	"reflect"
	"testing"
)

// scrambled is a section body stored as the format page states it: byte i
// XORed with bits 16 to 23 of a register that starts as key | key << 16,
// shifts left one bit per byte and has key ORed back in after every 16th
// byte.
func scrambled(plain []byte, key uint16) []byte {
	out := make([]byte, len(plain))
	reg := uint32(key)<<16 | uint32(key)
	for i := range plain {
		out[i] = plain[i] ^ uint8(reg>>16&0xff)
		reg = reg << 1
		if i%16 == 15 {
			reg = reg | uint32(key)
		}
	}
	return out
}

// pageFile is a file with section A as the format page lays it out, followed
// by an empty B header: the magic, A's header, A's 52 scrambled bytes.
func pageFile(id0, id1 uint32, name string, key uint16) []byte {
	a := make([]byte, 52)
	binary.LittleEndian.PutUint32(a[0:], id0)
	binary.LittleEndian.PutUint32(a[4:], id1)
	copy(a[0x0c:0x2b], name)
	a[0x2c], a[0x2f] = 0x40, 4
	var sum uint32
	for _, c := range a {
		sum = sum*2 + uint32(c)
	}
	f := binary.LittleEndian.AppendUint32(nil, 0x04507989)
	f = binary.LittleEndian.AppendUint32(f, 0xAAAAAAAA)
	f = binary.LittleEndian.AppendUint32(f, 52)
	f = binary.LittleEndian.AppendUint16(f, 0)
	f = binary.LittleEndian.AppendUint16(f, key)
	f = binary.LittleEndian.AppendUint32(f, sum)
	f = append(f, scrambled(a, key)...)
	f = binary.LittleEndian.AppendUint32(f, 0x55555555)
	return append(f, make([]byte, 12)...)
}

func pageSum(plain []byte) uint32 {
	var sum uint32
	for _, c := range plain {
		sum = sum*2 + uint32(c)
	}
	return sum
}

// pageSection is a section as the format page lays it out, under the key
// the low 15 bits of its plain checksum give.
func pageSection(tag uint32, plain []byte) []byte {
	sum := pageSum(plain)
	key := uint16(sum & 0x7fff)
	f := binary.LittleEndian.AppendUint32(nil, tag)
	f = binary.LittleEndian.AppendUint32(f, uint32(len(plain)))
	f = binary.LittleEndian.AppendUint16(f, 0)
	f = binary.LittleEndian.AppendUint16(f, key)
	f = binary.LittleEndian.AppendUint32(f, sum)
	return append(f, scrambled(plain, key)...)
}

// newFile is a new character's file from the claims: the magic; A with the
// pair, the name and the unit byte; B's 2560 zero bytes as 20 repeat packets
// of 127 and one of 20; C's 72 zero bytes; E with money and the four
// statistics, each stored field the reader's chain solved by hand and a zero
// pad before each field whose key bit is set, in the reader's bit order.
func newFile(id1 uint32, name string, unit byte, stats [4]uint8) []byte {
	a := make([]byte, 0x34)
	binary.LittleEndian.PutUint32(a, 1)
	binary.LittleEndian.PutUint32(a[4:], id1)
	binary.LittleEndian.PutUint32(a[8:], 0x529c0291)
	copy(a[0x0c:], name)
	a[0x2c] = unit
	f := binary.LittleEndian.AppendUint32(nil, 0x04507989)
	f = append(f, pageSection(0xAAAAAAAA, a)...)
	b := binary.LittleEndian.AppendUint32(nil, 2560)
	for range 20 {
		b = append(b, 0xff, 0)
	}
	f = append(f, pageSection(0x55555555, append(b, 0x80|20, 0))...)
	f = append(f, pageSection(0x40A40A40, make([]byte, 72))...)

	plain := make([]byte, 0x34)
	binary.LittleEndian.PutUint32(plain[0x10:], 1000)
	copy(plain[0x14:], stats[:])
	sum := pageSum(plain)
	key := uint16(sum & 0x7fff)
	v5, v6, v7, v8 := stats[0], stats[1], stats[2], stats[3]
	var zero uint32
	stored := []uint32{0x1529251, zero - 0x13141516, zero - 0xabcdef, 0x17ff12aa, 1000 + 0x21524542,
		uint32(v5 - 1000*0x11&0xff), uint32(v6 - v5*3), uint32(v7 - v5 - v6*5), uint32(v8 - v5*7 - v7*9),
		0x10121974, 0, 0xdadedade, 0, 0, 0, 0}
	pads := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 14, 13, 9, 10, 11, 12, 13}
	f = binary.LittleEndian.AppendUint32(f, 0x41392521)
	f = binary.LittleEndian.AppendUint32(f, uint32(0x34+bits.OnesCount16(key)))
	f = binary.LittleEndian.AppendUint16(f, 0)
	f = binary.LittleEndian.AppendUint16(f, key)
	f = binary.LittleEndian.AppendUint32(f, sum)
	for i, s := range stored {
		if key&(1<<pads[i]) != 0 {
			f = append(f, 0)
		}
		if i >= 5 && i <= 8 {
			f = append(f, uint8(s))
		} else {
			f = binary.LittleEndian.AppendUint32(f, s)
		}
	}
	return f
}

func newCharacter(id1 uint32, name string, unit byte, stats [4]uint8) Character {
	return Character{
		A:       A{ID0: 1, ID1: id1, ListKey: ListKeyAny, Name: []byte(name), Unit: unit},
		Kills:   make([]byte, KillsSize),
		Hotkeys: make([]byte, HotkeysSize),
		E:       E{Money: 1000, Base: stats},
	}
}

// A new character's file is the claims' layout byte for byte, and reads
// back as the character written.
func TestWriteNewCharacterMatchesTheClaimsFile(t *testing.T) {
	for _, tc := range []struct {
		id1   uint32
		name  string
		unit  byte
		stats [4]uint8
	}{{0, "Ranger", 0x40, [4]uint8{18, 22, 15, 25}}, {7, "Mage", 0xc0, [4]uint8{10, 14, 30, 26}}, {12, "X", 0, [4]uint8{1, 2, 3, 4}}} {
		in := newCharacter(tc.id1, tc.name, tc.unit, tc.stats)
		got, err := Write(in)
		if want := newFile(tc.id1, tc.name, tc.unit, tc.stats); err != nil || string(got) != string(want) {
			t.Fatalf("%s: write %x, %v\nwant  %x", tc.name, got, err, want)
		}
		if out, err := Read(got); err != nil || !reflect.DeepEqual(out, in) {
			t.Fatalf("%s: read %+v, %v", tc.name, out, err)
		}
	}
}

// Every section and every E field survives a write and a read, whatever key
// and pads the checksum gives; B's literal packets and long runs too.
func TestWriteThenReadEqualsTheInput(t *testing.T) {
	for seed := range 40 {
		in := Character{
			A:       A{ID0: uint32(seed), ID1: 0xfedcba98, ListKey: uint32(seed) * 0x01010101, Name: []byte("Ranger"), Unit: byte(seed)},
			Kills:   make([]byte, KillsSize),
			Hotkeys: []byte{1, 2, 3},
			Worn:    []byte{4, 5, byte(seed)},
			E: E{Player: [4]uint32{uint32(seed), 2, 3, 0xffffffff}, Money: uint32(seed) * 977, Base: [4]uint8{byte(seed), 2, 250, 4},
				Spells: 0x3ffffffe, Unit1C: 0x78, SkillXP: [5]uint32{uint32(seed), 6, 7, 8, 0x80000000}},
			Bag: []byte{byte(seed)},
		}
		in.A.Rest[6] = byte(seed)
		for i := range in.Kills {
			if i%300 < 200 {
				in.Kills[i] = byte(i * seed)
			}
		}
		b, err := Write(in)
		if err != nil {
			t.Fatal(err)
		}
		if out, err := Read(b); err != nil || !reflect.DeepEqual(out, in) {
			t.Fatalf("seed %d: read %+v, %v\nwant %+v", seed, out, err, in)
		}
	}
}

// Read refuses an unknown tag, a missing E, a broken E and a run past B's
// stated length; Write refuses a name with no room for its NUL.
func TestReadRefusesBrokenSections(t *testing.T) {
	good, err := Write(newCharacter(1, "Ranger", 0x40, [4]uint8{18, 22, 15, 25}))
	if err != nil {
		t.Fatal(err)
	}
	magic := binary.LittleEndian.AppendUint32(nil, 0x04507989)
	a := pageSection(0xAAAAAAAA, make([]byte, 0x34))
	broken := append([]byte(nil), good...)
	broken[len(broken)-1] ^= 1
	long := binary.LittleEndian.AppendUint32(nil, 4)
	long = append(long, 0x85, 0)
	for _, tc := range []struct {
		name string
		b    []byte
		want error
	}{
		{"unknown tag", append(append(append([]byte(nil), magic...), a...), pageSection(0x12345678, []byte{1})...), ErrTag},
		{"no E", append(append([]byte(nil), magic...), a...), ErrMissing},
		{"broken E", broken, ErrBody},
		{"run past B", append(append(append([]byte(nil), magic...), a...), pageSection(0x55555555, long)...), ErrBody},
	} {
		if _, err := Read(tc.b); !errors.Is(err, tc.want) {
			t.Errorf("%s: %v, want %v", tc.name, err, tc.want)
		}
	}
	if _, err := Write(Character{A: A{Name: []byte("Thirty-two letters fill the box!")}}); !errors.Is(err, ErrName) {
		t.Errorf("full name: %v", err)
	}
}

func TestReadIdentityDecodesAPageFile(t *testing.T) {
	id, err := ReadIdentity(pageFile(138639, 3133657, "Ranger", 0x5a3c))
	if err != nil {
		t.Fatal(err)
	}
	if id.ID0 != 138639 || id.ID1 != 3133657 || string(id.Name) != "Ranger" {
		t.Fatalf("identity %d %d %q", id.ID0, id.ID1, id.Name)
	}
}

func TestReadIdentityRefusesBadMagicChecksumAndSize(t *testing.T) {
	good := pageFile(1, 2, "Ranger", 0x7fff)
	magic := append([]byte(nil), good...)
	magic[0] ^= 1
	sum := append([]byte(nil), good...)
	sum[16] ^= 1
	body := append([]byte(nil), good...)
	body[20+51] ^= 0x10
	long := append([]byte(nil), good...)
	binary.LittleEndian.PutUint32(long[8:], uint32(len(long)))
	for _, tc := range []struct {
		name string
		b    []byte
		want error
	}{
		{"magic", magic, ErrMagic},
		{"stored checksum", sum, ErrChecksum},
		{"last body byte", body, ErrChecksum},
		{"length past the file", long, ErrSection},
		{"short header", good[:12], ErrSection},
		{"short body", good[:20+40], ErrSection},
		{"empty", nil, ErrMagic},
	} {
		if _, err := ReadIdentity(tc.b); !errors.Is(err, tc.want) {
			t.Errorf("%s: %v, want %v", tc.name, err, tc.want)
		}
	}
}
