package itemname

import (
	"reflect"
	"testing"
)

// leBin packs keys as bin's own little-endian u16 array.
func leBin(keys ...uint16) []byte {
	b := make([]byte, 2*len(keys))
	for i, k := range keys {
		b[2*i] = byte(k)
		b[2*i+1] = byte(k >> 8)
	}
	return b
}

func TestParsePairsEachKeyWithItsPositionalLine(t *testing.T) {
	bin := leBin(0x0101, 0x1382)
	txt := []byte("Sword\r\nChain Mail\r\n")
	got := Parse(bin, txt)
	want := map[uint16]string{0x0101: "Sword", 0x1382: "Chain Mail"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse(bin, txt) = %+v, want %+v", got, want)
	}
}

func TestParseAcceptsBareLF(t *testing.T) {
	got := Parse(leBin(1), []byte("Dagger\n"))
	if got[1] != "Dagger" {
		t.Errorf("Parse with bare LF: got[1] = %q, want %q", got[1], "Dagger")
	}
}

func TestParseDropsAnEmptyLine(t *testing.T) {
	got := Parse(leBin(1, 2), []byte("\r\nMace\r\n"))
	if _, ok := got[1]; ok {
		t.Errorf("Parse: key 1 has an empty line and must not appear, got %q", got[1])
	}
	if got[2] != "Mace" {
		t.Errorf("Parse: got[2] = %q, want %q", got[2], "Mace")
	}
}

func TestParseBoundsOnTheShorterOfTheTwoLengths(t *testing.T) {
	// Three keys, one line: only the first key is walked.
	got := Parse(leBin(1, 2, 3), []byte("Sword\r\n"))
	want := map[uint16]string{1: "Sword"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse with a short txt = %+v, want %+v", got, want)
	}
	// One key, three lines: only the first line is walked.
	got = Parse(leBin(9), []byte("Sword\r\nMace\r\nDagger\r\n"))
	want = map[uint16]string{9: "Sword"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse with a short bin = %+v, want %+v", got, want)
	}
}

func TestParseWithNoEntriesAnswersAnEmptyMap(t *testing.T) {
	if got := Parse(nil, nil); len(got) != 0 {
		t.Errorf("Parse(nil, nil) = %+v, want empty", got)
	}
}

// TestParseStoresTheLinesBytesUnchanged checks the package doc's rule
// directly: a stored name carries the shipped bytes with no code-page
// conversion, so a line built from bytes out of both of text.Convert's
// source blocks (0x80..0xAF, 0xE0..0xEF — pkg/render/text's own selector-1
// arithmetic) comes back byte for byte. Built from byte values rather than
// literal non-ASCII text (AGENTS.md rule 2).
func TestParseStoresTheLinesBytesUnchanged(t *testing.T) {
	txt := append([]byte{0x80, 0xAF, 0xE0, 0xEF}, '\r', '\n')
	got := Parse(leBin(5), txt)
	want := string([]byte{0x80, 0xAF, 0xE0, 0xEF})
	if got[5] != want {
		t.Errorf("Parse: got[5] = % x, want % x", []byte(got[5]), []byte(want))
	}
}

// TestParseLeavesAnUnmovedHighByteUnchanged checks a byte in
// 0xB0..0xDF — the block text.Convert's two moved ranges land ON, and so
// itself untouched by that converter — survives Parse exactly as any other
// byte does.
func TestParseLeavesAnUnmovedHighByteUnchanged(t *testing.T) {
	txt := append([]byte{0xB0, 0xDF}, '\r', '\n')
	got := Parse(leBin(7), txt)
	want := string([]byte{0xB0, 0xDF})
	if got[7] != want {
		t.Errorf("Parse: got[7] = % x, want % x", []byte(got[7]), []byte(want))
	}
}
