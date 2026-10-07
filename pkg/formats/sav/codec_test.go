package sav

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// blob assembles a compressed blob from a word count and raw opcode bytes, so a
// test states the stream it means rather than a hexdump of it.
func blob(words int, ops ...byte) []byte {
	b := make([]byte, 4, 4+len(ops))
	binary.LittleEndian.PutUint32(b, uint32(words))
	return append(b, ops...)
}

func TestDecompressLiteralAndRun(t *testing.T) {
	for _, c := range []struct {
		name string
		in   []byte
		want []byte
	}{
		{"two literal words", blob(2, 0x02, 0xaa, 0xbb, 0xcc, 0xdd),
			[]byte{0xaa, 0xbb, 0xcc, 0xdd}},
		{"a run of three", blob(3, 0x83, 0x11, 0x22),
			[]byte{0x11, 0x22, 0x11, 0x22, 0x11, 0x22}},
		{"a run whose two bytes differ is still ONE unit",
			blob(2, 0x82, 0x34, 0x12), []byte{0x34, 0x12, 0x34, 0x12}},
		{"literal then run", blob(3, 0x01, 0x01, 0x02, 0x82, 0x09, 0x08),
			[]byte{0x01, 0x02, 0x09, 0x08, 0x09, 0x08}},
		// The four opcode values the SHIPPED ENCODER never emits are all
		// legal and the decoder must take them, because "never observed"
		// is a fact about an encoder and not about a decoder.
		{"opcode 0x00 is a literal of no words, costing one byte",
			blob(1, 0x00, 0x01, 0xff, 0xee), []byte{0xff, 0xee}},
		{"opcode 0x80 is a run of no words, costing three",
			blob(1, 0x80, 0x77, 0x77, 0x01, 0x12, 0x34), []byte{0x12, 0x34}},
		{"opcode 0x81 is a run of one word", blob(1, 0x81, 0x5a, 0xa5),
			[]byte{0x5a, 0xa5}},
		{"opcode 0x7f is a literal of 127 words", blob(127, append([]byte{0x7f},
			make([]byte, 254)...)...), make([]byte, 254)},
		{"an empty stream", blob(0), nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := Decompress(c.in)
			if err != nil {
				t.Fatalf("Decompress: %v", err)
			}
			if !bytes.Equal(got, c.want) {
				t.Fatalf("got % x, want % x", got, c.want)
			}
		})
	}
}

func TestDecompressRefuses(t *testing.T) {
	for _, c := range []struct {
		name string
		in   []byte
	}{
		{"a blob too short to hold its own count", []byte{1, 2, 3}},
		{"a literal that overruns the blob", blob(4, 0x04, 0xaa, 0xbb)},
		{"a run whose word overruns the blob", blob(4, 0x84, 0xaa)},
		{"a stream that emits fewer words than declared", blob(9, 0x82, 0x01, 0x02)},
		{"a stream that emits more words than declared", blob(1, 0x82, 0x01, 0x02)},
		{"a count no span of opcodes could emit", blob(1 << 20)},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := Decompress(c.in); err == nil {
				t.Fatal("want an error, got none")
			}
		})
	}
}

// TestCompressRoundTrip is the property the whole package rests on: whatever the
// encoder emits, the decoder reads back as what went in.
func TestCompressRoundTrip(t *testing.T) {
	for _, c := range []struct {
		name string
		in   []byte
	}{
		{"nothing", nil},
		{"one word", []byte{1, 2}},
		{"all one word repeated far past a run's reach",
			bytes.Repeat([]byte{0xde, 0xad}, 400)},
		{"no two adjacent words alike", counting(600)},
		{"runs and literals alternating", mixed()},
		{"an ODD byte length, which the word unit cannot express", []byte{1, 2, 3}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := Decompress(Compress(c.in))
			if err != nil {
				t.Fatalf("Decompress: %v", err)
			}
			want := c.in
			if len(want)%2 != 0 {
				want = append(append([]byte(nil), want...), 0)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("round trip lost bytes: got %d, want %d", len(got), len(want))
			}
		})
	}
}

// TestCompressCapsItsOpcodes checks the two ceilings by construction rather than
// by reading them off the constants: a run longer than one opcode can carry must
// come out as more than one opcode, and so must a literal.
func TestCompressCapsItsOpcodes(t *testing.T) {
	long := bytes.Repeat([]byte{0x11, 0x22}, maxRun+5)
	out := Compress(long)
	if out[4] != byte(opRun|maxRun) {
		t.Fatalf("first opcode is %#02x, want the widest run %#02x", out[4], opRun|maxRun)
	}
	lit := counting(maxLiteral + 10)
	out = Compress(lit)
	if out[4] != maxLiteral {
		t.Fatalf("first opcode is %#02x, want the widest literal %#02x", out[4], maxLiteral)
	}
}

// TestCompressPrefersARunAtEveryRepeat is the encoder's whole rule, stated as a
// test: a literal is what happens where a repeat does not.
func TestCompressPrefersARunAtEveryRepeat(t *testing.T) {
	in := []byte{1, 0, 2, 0, 2, 0, 3, 0}
	out := Compress(in)
	want := []byte{0x01, 1, 0, opRun | 2, 2, 0, 0x01, 3, 0}
	if !bytes.Equal(out[4:], want) {
		t.Fatalf("got % x, want % x", out[4:], want)
	}
}

func counting(words int) []byte {
	b := make([]byte, 2*words)
	for i := 0; i < words; i++ {
		binary.LittleEndian.PutUint16(b[2*i:], uint16(i+1))
	}
	return b
}

func mixed() []byte {
	var b []byte
	for i := 0; i < 40; i++ {
		b = append(b, counting(3)...)
		b = append(b, bytes.Repeat([]byte{byte(i), 0x7f}, 5)...)
	}
	return b
}
