package a2c

import (
	"encoding/binary"
	"errors"
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
