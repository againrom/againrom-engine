package sav

import (
	"encoding/binary"
	"fmt"
)

// The transport is a run/literal code over 16-BIT WORDS, not bytes. The repeated
// unit being a word is the trap the whole codec turns on: a byte-wise reading
// keeps the length and corrupts the content, so it is not a variant to fall back
// to when something does not fit.
const (
	// opRun is the high bit: an opcode at or above it is a run, and the low
	// seven bits are its count.
	opRun = 0x80
	// maxRun is the widest run one opcode can express.
	maxRun = 0x7f
	// maxLiteral is the widest literal the SHIPPED ENCODER emits. 0x7f is a
	// legal literal of 127 words and the decoder accepts it; the encoder
	// stops one short of it because that is what the original does, and
	// reproducing the original's output byte for byte is the only test of
	// this encoder available without a running original.
	maxLiteral = 126
)

// Decompress expands a blob — the bytes from the container's 0x10 to its blobEnd
// — into the decoded stream.
//
// The blob's leading u32 is the OUTPUT WORD COUNT. The opcode loop that follows
// is bounded by the SOURCE span and never by the output count, and must then have
// emitted exactly that many words: a decode that ends short, ends long, or reads
// past its input is an error naming which, because each is a different way for a
// caller's slice to be wrong.
func Decompress(blob []byte) ([]byte, error) {
	if len(blob) < 4 {
		return nil, fmt.Errorf("sav: blob is %d bytes, too short for its word count", len(blob))
	}
	words := binary.LittleEndian.Uint32(blob)
	if int64(words)*2 > int64(len(blob))*int64(maxRun) {
		return nil, fmt.Errorf("sav: blob declares %d words, more than %d bytes of opcodes can emit",
			words, len(blob))
	}
	out := make([]byte, 0, int(words)*2)
	for p := 4; p < len(blob); {
		n := blob[p]
		p++
		if n < opRun {
			end := p + 2*int(n)
			if end > len(blob) {
				return nil, fmt.Errorf("sav: literal of %d words at %d overruns the blob (%d bytes)",
					n, p-1, len(blob))
			}
			out = append(out, blob[p:end]...)
			p = end
			continue
		}
		if p+2 > len(blob) {
			return nil, fmt.Errorf("sav: run at %d overruns the blob (%d bytes)", p-1, len(blob))
		}
		w := blob[p : p+2]
		p += 2
		for i := 0; i < int(n&maxRun); i++ {
			out = append(out, w...)
		}
	}
	if len(out) != int(words)*2 {
		return nil, fmt.Errorf("sav: blob emitted %d words, declared %d", len(out)/2, words)
	}
	return out, nil
}

// Compress is Decompress run backwards, and it reproduces the ORIGINAL
// ENCODER's output byte for byte on every file of the shipped corpus.
//
// The rule is greedy and has no tie to break: a run wherever this word equals
// the next, a literal otherwise, each capped at what one opcode can carry. An
// odd-length input is padded to an even byte length first, because the unit is a
// word and half a word cannot be emitted.
func Compress(body []byte) []byte {
	src := body
	if len(src)%2 != 0 {
		src = append(append(make([]byte, 0, len(body)+1), body...), 0)
	}
	words := len(src) / 2
	out := make([]byte, 4, len(src)+len(src)/64+8)
	binary.LittleEndian.PutUint32(out, uint32(words))
	at := func(i int) uint16 { return binary.LittleEndian.Uint16(src[2*i:]) }
	for i := 0; i < words; {
		if i+1 < words && at(i) == at(i+1) {
			n := 2
			for i+n < words && at(i+n) == at(i) && n < maxRun {
				n++
			}
			out = append(out, byte(opRun|n), src[2*i], src[2*i+1])
			i += n
			continue
		}
		j := i
		for j < words && !(j+1 < words && at(j) == at(j+1)) && j-i < maxLiteral {
			j++
		}
		out = append(out, byte(j-i))
		out = append(out, src[2*i:2*j]...)
		i = j
	}
	return out
}
