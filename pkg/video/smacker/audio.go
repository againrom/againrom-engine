// Ported from libsmacker 1.2.0 (Greg Kennedy, LGPL-2.1-or-later): raw and
// DPCM audio-chunk decode (smk_get_audio_data in smacker.c). See doc.go for
// the package-wide attribution and THIRD_PARTY_NOTICES.md for the notice
// entry.
package smacker

import (
	"encoding/binary"
	"fmt"
)

// errBinkAudio marks a track whose compress field selected the Bink
// (perceptual) codec. libsmacker itself does not implement this path either
// (see its own warning in smk_open_generic); no file in the shipped corpus
// sets it (DIV-1258).
var errBinkAudio = fmt.Errorf("smacker: audio: Bink-compressed track unsupported")

// decodeAudioChunk ports smk_render_audio. p is the audio record payload
// after the caller has already consumed its own 4-byte declared-size prefix.
func decodeAudioChunk(t audioTrack, p []byte) ([]byte, error) {
	switch t.compress {
	case 0:
		out := make([]byte, len(p))
		copy(out, p)
		return out, nil
	case 2:
		return nil, errBinkAudio
	}
	return decodeDPCM(t, p)
}

// decodeDPCM ports the compress==1 branch of smk_render_audio: a bitstream
// of Huffman-coded deltas, one or two 8-bit trees per channel depending on
// bit depth, decoded as a running sum against the previous same-channel
// sample. The first 4 bytes of p are the declared unpacked buffer size.
func decodeDPCM(t audioTrack, p []byte) ([]byte, error) {
	if len(p) < 4 {
		return nil, fmt.Errorf("smacker: audio: need 4 bytes for unpacked size")
	}
	size := binary.LittleEndian.Uint32(p)
	if size > maxAudioBuffer {
		return nil, fmt.Errorf("smacker: audio: unpacked size %d exceeds bound", size)
	}
	out := make([]byte, size)
	r := newBitReader(p[4:])

	bit, err := r.readBit()
	if err != nil {
		return nil, fmt.Errorf("smacker: audio: initial bit: %w", err)
	}
	if bit == 0 {
		return nil, fmt.Errorf("smacker: audio: initial bit unset")
	}
	// The next two bits restate channel count and bit depth; libsmacker only
	// warns on a mismatch against the header-declared values and keeps
	// decoding by the header's own numbers, so this port does the same.
	if _, err := r.readBit(); err != nil {
		return nil, fmt.Errorf("smacker: audio: channel bit: %w", err)
	}
	if _, err := r.readBit(); err != nil {
		return nil, fmt.Errorf("smacker: audio: depth bit: %w", err)
	}

	tree0, err := buildHuff8(r)
	if err != nil {
		return nil, fmt.Errorf("smacker: audio: tree 0: %w", err)
	}
	var tree1, tree2, tree3 *huff8Tree
	if t.BitDepth == 16 {
		if tree1, err = buildHuff8(r); err != nil {
			return nil, fmt.Errorf("smacker: audio: tree 1: %w", err)
		}
	}
	if t.Channels == 2 {
		if tree2, err = buildHuff8(r); err != nil {
			return nil, fmt.Errorf("smacker: audio: tree 2: %w", err)
		}
		if t.BitDepth == 16 {
			if tree3, err = buildHuff8(r); err != nil {
				return nil, fmt.Errorf("smacker: audio: tree 3: %w", err)
			}
		}
	}

	// Initial per-channel sample. The bitstream carries the RIGHT channel
	// first (when stereo) then the LEFT/mono channel, and for 16-bit depth
	// each carries its HIGH byte before its LOW byte — the reverse order
	// from the delta path below. This asymmetry is the original codec's own
	// layout, not a transcription choice.
	if t.Channels == 2 {
		a, err := r.readByte()
		if err != nil {
			return nil, fmt.Errorf("smacker: audio: initial right sample: %w", err)
		}
		if t.BitDepth == 16 {
			b, err := r.readByte()
			if err != nil {
				return nil, fmt.Errorf("smacker: audio: initial right sample low byte: %w", err)
			}
			if len(out) < 4 {
				return nil, fmt.Errorf("smacker: audio: buffer too small for stereo header")
			}
			binary.LittleEndian.PutUint16(out[2:], uint16(b)|uint16(a)<<8)
		} else {
			if len(out) < 2 {
				return nil, fmt.Errorf("smacker: audio: buffer too small for stereo header")
			}
			out[1] = byte(a)
		}
	}
	a, err := r.readByte()
	if err != nil {
		return nil, fmt.Errorf("smacker: audio: initial sample: %w", err)
	}
	if t.BitDepth == 16 {
		b, err := r.readByte()
		if err != nil {
			return nil, fmt.Errorf("smacker: audio: initial sample low byte: %w", err)
		}
		if len(out) < 2 {
			return nil, fmt.Errorf("smacker: audio: buffer too small for header")
		}
		binary.LittleEndian.PutUint16(out, uint16(b)|uint16(a)<<8)
	} else {
		if len(out) < 1 {
			return nil, fmt.Errorf("smacker: audio: buffer too small for header")
		}
		out[0] = byte(a)
	}

	if t.BitDepth == 8 {
		j, k := 1, 1
		if t.Channels == 2 {
			j, k = 2, 2
		}
		for k < int(size) {
			v, err := tree0.lookup(r)
			if err != nil {
				return nil, fmt.Errorf("smacker: audio: delta lookup: %w", err)
			}
			if j >= len(out) {
				return nil, fmt.Errorf("smacker: audio: delta index out of range")
			}
			out[j] = byte(int8(v) + int8(out[j-t.Channels]))
			j++
			k++
			if t.Channels == 2 {
				v, err := tree2.lookup(r)
				if err != nil {
					return nil, fmt.Errorf("smacker: audio: right delta lookup: %w", err)
				}
				if j >= len(out) {
					return nil, fmt.Errorf("smacker: audio: right delta index out of range")
				}
				out[j] = byte(int8(v) + int8(out[j-2]))
				j++
				k++
			}
		}
		return out, nil
	}

	j, k := 1, 2
	if t.Channels == 2 {
		j, k = 2, 4
	}
	sampleAt := func(i int) int16 { return int16(binary.LittleEndian.Uint16(out[2*i:])) }
	putSample := func(i int, v int16) { binary.LittleEndian.PutUint16(out[2*i:], uint16(v)) }
	for k < int(size) {
		lo, err := tree0.lookup(r)
		if err != nil {
			return nil, fmt.Errorf("smacker: audio: delta low lookup: %w", err)
		}
		hi, err := tree1.lookup(r)
		if err != nil {
			return nil, fmt.Errorf("smacker: audio: delta high lookup: %w", err)
		}
		if 2*(j+1) > len(out) {
			return nil, fmt.Errorf("smacker: audio: delta index out of range")
		}
		delta := int16(uint16(lo) | uint16(hi)<<8)
		putSample(j, delta+sampleAt(j-t.Channels))
		j++
		k += 2
		if t.Channels == 2 {
			lo, err := tree2.lookup(r)
			if err != nil {
				return nil, fmt.Errorf("smacker: audio: right delta low lookup: %w", err)
			}
			hi, err := tree3.lookup(r)
			if err != nil {
				return nil, fmt.Errorf("smacker: audio: right delta high lookup: %w", err)
			}
			if 2*(j+1) > len(out) {
				return nil, fmt.Errorf("smacker: audio: right delta index out of range")
			}
			delta := int16(uint16(lo) | uint16(hi)<<8)
			putSample(j, delta+sampleAt(j-2))
			j++
			k += 2
		}
	}
	return out, nil
}
