package alch

import (
	"encoding/binary"
	"errors"
	"reflect"
	"testing"
)

// pageFile is a network character file in the hero format page's write
// order: the AlCh tag, 8 bytes, the 32-byte name, 32 bytes, flags with bit 0
// set, 4 bytes and two 28-byte blocks, where such a file ends.
func pageFile(name string) []byte {
	f := binary.LittleEndian.AppendUint32(nil, 0x68436c41)
	f = append(f, make([]byte, 8)...)
	n := make([]byte, 32)
	copy(n, name)
	f = append(f, n...)
	f = append(f, make([]byte, 32)...)
	f = binary.LittleEndian.AppendUint32(f, 1)
	f = append(f, make([]byte, 4)...)
	return append(f, make([]byte, 56)...)
}

func TestReadHeaderDecodesAPageFile(t *testing.T) {
	b := pageFile("Gwendolyn")
	if len(b) != 0x8c {
		t.Fatalf("page file is %d bytes", len(b))
	}
	h, err := ReadHeader(b)
	if err != nil || string(h.Name) != "Gwendolyn" {
		t.Fatalf("header %q, %v", h.Name, err)
	}
	full := pageFile("Thirty-two letters fill the box!")
	if h, err := ReadHeader(full); err != nil || len(h.Name) != 32 {
		t.Fatalf("full name %q, %v", h.Name, err)
	}
}

// A short file of a name and four statistics is the page file with the
// statistics as the first four dwords of the first 28-byte block at +0x54.
func TestWriteShortFileMatchesThePageFile(t *testing.T) {
	want := pageFile("Gwendolyn")
	for i, v := range []uint32{18, 22, 15, 25} {
		binary.LittleEndian.PutUint32(want[0x54+4*i:], v)
	}
	got, err := Write(Character{Name: []byte("Gwendolyn"), Flags: FlagShort, Stats: [4]uint32{18, 22, 15, 25}})
	if err != nil || string(got) != string(want) {
		t.Fatalf("write %x, %v\nwant  %x", got, err, want)
	}
}

// Every field survives a write and a read, the bytes after the header too;
// a name with no room for its NUL is refused.
func TestWriteThenReadEqualsTheInput(t *testing.T) {
	in := Character{Name: []byte("Bors"), Flags: 6, Store74: 0x01020304, Stats: [4]uint32{1, 2, 3, 4},
		Long: []byte{9, 8, 7}}
	for i := range in.Store08 {
		in.Store08[i] = byte(0x10 + i)
	}
	for i := range in.Store30 {
		in.Store30[i] = byte(0x30 + i)
	}
	for i := range in.Store90 {
		in.Store90[i] = byte(0x90 + i)
	}
	for i := range in.StoreAC {
		in.StoreAC[i] = byte(0xac + i)
	}
	b, err := Write(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Read(b)
	if err != nil || !reflect.DeepEqual(out, in) {
		t.Fatalf("read %+v, %v\nwant %+v", out, err, in)
	}
	if _, err := Write(Character{Name: []byte("Thirty-two letters fill the box!")}); !errors.Is(err, ErrName) {
		t.Errorf("full name: %v", err)
	}
	if _, err := Write(Character{Name: []byte("A\x00B")}); !errors.Is(err, ErrName) {
		t.Errorf("NUL name: %v", err)
	}
}

func TestReadHeaderRefusesBadTagAndShortFile(t *testing.T) {
	tag := pageFile("Gwendolyn")
	tag[3] = 'i'
	if _, err := ReadHeader(tag); !errors.Is(err, ErrTag) {
		t.Errorf("tag: %v", err)
	}
	if _, err := ReadHeader(pageFile("Gwendolyn")[:0x8b]); !errors.Is(err, ErrSize) {
		t.Errorf("short: %v", err)
	}
	if _, err := ReadHeader(nil); !errors.Is(err, ErrTag) {
		t.Errorf("empty: %v", err)
	}
}
