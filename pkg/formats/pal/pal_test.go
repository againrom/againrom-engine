package pal_test

import (
	"strings"
	"testing"

	"againrom/pkg/formats/pal"
)

// stream builds a synthetic palette file: the two magic bytes, filler up to the
// table offset, the table itself, and however many trailing bytes were asked
// for. NOTHING here comes from a game install — the entries are generated, and
// the filler is a value the decoder must never read.
func stream(entries func(i int) (b, g, r, x byte), trailing int) []byte {
	out := make([]byte, pal.MinSize+trailing)
	for i := range out {
		out[i] = 0xa5 // must be invisible everywhere but inside the window
	}
	out[0], out[1] = 'B', 'M'
	for i := 0; i < pal.EntryCount; i++ {
		b, g, r, x := entries(i)
		e := out[pal.TableOffset+i*pal.EntrySize:]
		e[0], e[1], e[2], e[3] = b, g, r, x
	}
	return out
}

// ramp is the generator the exactness cases use: three channels that disagree
// with each other at every index, so a decoder that swapped two of them, or read
// the reserved byte as one, cannot pass.
func ramp(i int) (b, g, r, x byte) {
	return byte(i), byte(255 - i), byte(i * 7), byte(i * 13)
}

func TestDecodeReadsTheWindowExactly(t *testing.T) {
	got, err := pal.Decode(stream(ramp, 0))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	for i := 0; i < pal.EntryCount; i++ {
		b, g, r, _ := ramp(i)
		want := pal.Color{R: r, G: g, B: b}
		if got[i] != want {
			t.Fatalf("entry %d = %+v, want %+v", i, got[i], want)
		}
	}
}

// Entry 0 is decoded like every other one and given no meaning: it is the
// transparent key by the sheet's convention, not by anything a table says.
func TestDecodeCarriesEntryZeroAndTheExtremes(t *testing.T) {
	edge := func(i int) (b, g, r, x byte) {
		switch i {
		case 0:
			return 0x11, 0x22, 0x33, 0xff
		case 255:
			return 0xff, 0xff, 0xff, 0x00
		}
		return 0, 0, 0, 0
	}
	got, err := pal.Decode(stream(edge, 0))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if want := (pal.Color{R: 0x33, G: 0x22, B: 0x11}); got[0] != want {
		t.Fatalf("entry 0 = %+v, want %+v", got[0], want)
	}
	if want := (pal.Color{R: 0xff, G: 0xff, B: 0xff}); got[255] != want {
		t.Fatalf("entry 255 = %+v, want %+v", got[255], want)
	}
}

// A longer stream is read exactly as one of MinSize is: the pixel data after the
// window is never opened, so the shipped 17462- and 26678-byte shapes and the
// bare minimum decode to the same table.
func TestDecodeIgnoresEverythingOutsideTheWindow(t *testing.T) {
	bare, err := pal.Decode(stream(ramp, 0))
	if err != nil {
		t.Fatalf("Decode(bare): %v", err)
	}
	for _, trailing := range []int{1, 1024, 17462 - pal.MinSize} {
		long, err := pal.Decode(stream(ramp, trailing))
		if err != nil {
			t.Fatalf("Decode(+%d): %v", trailing, err)
		}
		if long != bare {
			t.Fatalf("+%d trailing bytes changed the table", trailing)
		}
	}
}

func TestDecodeRefusesAShortStream(t *testing.T) {
	full := stream(ramp, 0)
	// Every length below the minimum, the empty stream included — the loop is
	// the whole domain of this refusal and not three samples of it.
	for n := 0; n < pal.MinSize; n++ {
		_, err := pal.Decode(full[:n])
		if err == nil {
			t.Fatalf("%d byte(s): no error", n)
		}
		if !strings.Contains(err.Error(), "want at least") {
			t.Fatalf("%d byte(s): %v does not name the length", n, err)
		}
	}
}

func TestDecodeRefusesEveryPrefixButTheMagic(t *testing.T) {
	full := stream(ramp, 0)
	accepted := 0
	for hi := 0; hi < 256; hi++ {
		for lo := 0; lo < 256; lo++ {
			s := append([]byte(nil), full...)
			s[0], s[1] = byte(hi), byte(lo)
			_, err := pal.Decode(s)
			if err == nil {
				accepted++
				if hi != 'B' || lo != 'M' {
					t.Fatalf("prefix %#02x %#02x accepted", hi, lo)
				}
				continue
			}
			if !strings.Contains(err.Error(), "want") {
				t.Fatalf("prefix %#02x %#02x: %v does not name what it wanted", hi, lo, err)
			}
		}
	}
	if accepted != 1 {
		t.Fatalf("%d of 65536 prefixes accepted, want 1", accepted)
	}
}

// The one prefix that is not the magic and matters: the shared-owner palette,
// which is sixteen 1024-byte tables read whole with no seek and begins 00 00.
// Handed that shape, a reader written for the common case would answer 1024
// bytes of the eleventh table; this one refuses.
func TestDecodeRefusesTheOtherPaletteShape(t *testing.T) {
	shared := make([]byte, 16*1024)
	for i := range shared {
		shared[i] = byte(i)
	}
	shared[0], shared[1] = 0x00, 0x00
	if _, err := pal.Decode(shared); err == nil {
		t.Fatal("the 16-table shape decoded as a BMP colour table")
	}
}

// Totality: no stream panics the decoder. The cases are generated rather than
// listed, because the failure this guards against is an index computed from a
// length nobody bounded.
func TestDecodeIsTotal(t *testing.T) {
	for n := 0; n <= pal.MinSize+7; n++ {
		for _, fill := range []byte{0x00, 0x42, 0xff} {
			s := make([]byte, n)
			for i := range s {
				s[i] = fill
			}
			if n >= 2 && fill == 0x42 {
				s[0], s[1] = 'B', 'M'
			}
			// The only assertion is that this returns.
			_, _ = pal.Decode(s)
		}
	}
}

// The constants are a stated contract, not incidental: 256 entries of 4 bytes is
// the 0x400 the engine reads, and the offset is the 0x36 it seeks to.
func TestGeometryConstants(t *testing.T) {
	if pal.TableOffset != 0x36 || pal.TableSize != 0x400 || pal.MinSize != 0x436 {
		t.Fatalf("offset %#x, table %#x, min %#x", pal.TableOffset, pal.TableSize, pal.MinSize)
	}
	if pal.EntryCount*pal.EntrySize != pal.TableSize {
		t.Fatalf("%d x %d != %d", pal.EntryCount, pal.EntrySize, pal.TableSize)
	}
}
