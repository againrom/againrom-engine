// Package wav reads one RIFF/WAVE stream of uncompressed PCM: the public
// container, decoded once for every caller. It walks the chunk list, takes the
// fmt and data chunks in either order, skips every other chunk, and checks each
// declared size against the bytes that remain, so a lying size is an error and
// never a slice panic. Resampling and channel mixing are the caller's.
package wav

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// FormatPCM is the one fmt-chunk format tag accepted: linear PCM.
const FormatPCM = 1

// PCM is a decoded stream: its fmt chunk's channel count (1 or 2), sample width
// (8 or 16 bits) and rate, and the data chunk's bytes, aliased from the input.
type PCM struct {
	Channels      int
	BitsPerSample int
	Rate          int
	Data          []byte
}

// Parse reads the stream. A stream that is not RIFF/WAVE, a missing or
// malformed fmt chunk, a missing data chunk, a format other than PCM, a channel
// count other than 1 or 2 and a width other than 8 or 16 bits are refused. RIFF
// pads an odd-sized chunk to an even offset; the pad byte is skipped.
func Parse(b []byte) (PCM, error) {
	if len(b) < 12 {
		return PCM{}, fmt.Errorf("wav: %d byte(s), too short for a RIFF header", len(b))
	}
	if string(b[0:4]) != "RIFF" {
		return PCM{}, fmt.Errorf("wav: stream begins %q, want RIFF", b[0:4])
	}
	if string(b[8:12]) != "WAVE" {
		return PCM{}, fmt.Errorf("wav: RIFF form is %q, want WAVE", b[8:12])
	}
	var (
		p       PCM
		haveFmt bool
	)
	for pos := 12; pos+8 <= len(b); {
		id := string(b[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(b[pos+4 : pos+8]))
		pos += 8
		if size < 0 || size > len(b)-pos {
			return PCM{}, fmt.Errorf("wav: %s chunk claims %d byte(s), only %d remain", id, size, len(b)-pos)
		}
		body := b[pos : pos+size]
		switch id {
		case "fmt ":
			if err := p.readFormat(body); err != nil {
				return PCM{}, err
			}
			haveFmt = true
		case "data":
			p.Data = body
		}
		pos += size + size%2
	}
	if !haveFmt {
		return PCM{}, errors.New("wav: no fmt chunk")
	}
	if p.Data == nil {
		return PCM{}, errors.New("wav: no data chunk")
	}
	return p, nil
}

func (p *PCM) readFormat(body []byte) error {
	if len(body) < 16 {
		return fmt.Errorf("wav: fmt chunk is %d byte(s), want at least 16", len(body))
	}
	if format := binary.LittleEndian.Uint16(body[0:2]); format != FormatPCM {
		return fmt.Errorf("wav: format tag %d, want %d (PCM)", format, FormatPCM)
	}
	p.Channels = int(binary.LittleEndian.Uint16(body[2:4]))
	if p.Channels != 1 && p.Channels != 2 {
		return fmt.Errorf("wav: %d channel(s), want 1 or 2", p.Channels)
	}
	p.Rate = int(binary.LittleEndian.Uint32(body[4:8]))
	if p.Rate <= 0 {
		return fmt.Errorf("wav: fmt chunk declares a sample rate of %d", p.Rate)
	}
	p.BitsPerSample = int(binary.LittleEndian.Uint16(body[14:16]))
	if p.BitsPerSample != 8 && p.BitsPerSample != 16 {
		return fmt.Errorf("wav: %d bit(s) per sample, want 8 or 16", p.BitsPerSample)
	}
	return nil
}

// Frames is the number of whole frames in Data; a partial tail is not counted.
func (p PCM) Frames() int {
	return len(p.Data) / (p.Channels * p.BitsPerSample / 8)
}

// Sample is channel ch of frame i in the signed 16-bit range. A 16-bit sample
// is read as stored; an 8-bit sample is unsigned with 128 as silence and is
// shifted into the same range. A mono stream answers channel 0 for either
// channel.
func (p PCM) Sample(i, ch int) int16 {
	if p.Channels == 1 {
		ch = 0
	}
	width := p.BitsPerSample / 8
	off := (i*p.Channels + ch) * width
	if p.BitsPerSample == 16 {
		return int16(binary.LittleEndian.Uint16(p.Data[off:]))
	}
	return int16((int32(p.Data[off]) - 128) << 8)
}
