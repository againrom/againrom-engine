package alch

import (
	"encoding/binary"
	"errors"
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
