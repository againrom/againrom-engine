// Package wav decodes RIFF/WAVE PCM and Microsoft ADPCM for every caller.
// Resampling and channel mixing are the caller's.
package wav

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// FormatPCM is the linear PCM format tag.
const FormatPCM = 1

// PCM holds mono or stereo samples at Rate. PCM input aliases Data; ADPCM
// allocates signed 16-bit Data. FormatTag identifies the source encoding,
// including on an error after its fmt tag was read; other error fields are zero.
type PCM struct {
	FormatTag     uint16
	Channels      int
	BitsPerSample int
	Rate          int
	Data          []byte
}

// Parse walks the RIFF chunks once, in either order, and rejects malformed
// input. Tag 1 accepts 8/16-bit PCM; tag 2 accepts 4-bit Microsoft ADPCM.
// RIFF chunk padding and ADPCM block padding are excluded from samples.
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
		format  []byte
		fact    []byte
	)
	for pos := 12; pos+8 <= len(b); {
		id := string(b[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(b[pos+4 : pos+8]))
		pos += 8
		if size < 0 || size > len(b)-pos {
			return PCM{FormatTag: p.FormatTag}, fmt.Errorf("wav: %s chunk claims %d byte(s), only %d remain", id, size, len(b)-pos)
		}
		body := b[pos : pos+size]
		switch id {
		case "fmt ":
			if err := p.readFormat(body); err != nil {
				return PCM{FormatTag: p.FormatTag}, err
			}
			format = body
			haveFmt = true
		case "data":
			p.Data = body
		case "fact":
			fact = body
		}
		pos += size + size%2
	}
	if !haveFmt {
		return PCM{}, errors.New("wav: no fmt chunk")
	}
	if p.Data == nil {
		return PCM{FormatTag: p.FormatTag}, errors.New("wav: no data chunk")
	}
	if p.FormatTag == 2 {
		if err := p.decodeADPCM(format, fact); err != nil {
			return PCM{FormatTag: p.FormatTag}, err
		}
	}
	return p, nil
}

func (p *PCM) readFormat(body []byte) error {
	if len(body) < 16 {
		return fmt.Errorf("wav: fmt chunk is %d byte(s), want at least 16", len(body))
	}
	p.FormatTag = binary.LittleEndian.Uint16(body[0:2])
	if p.FormatTag != FormatPCM && p.FormatTag != 2 {
		return fmt.Errorf("wav: format tag %d, want 1 (PCM) or 2 (Microsoft ADPCM)", p.FormatTag)
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
	if p.FormatTag == 2 {
		if p.BitsPerSample != 4 {
			return fmt.Errorf("wav: ADPCM has %d bit(s) per sample, want 4", p.BitsPerSample)
		}
		return nil
	}
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
