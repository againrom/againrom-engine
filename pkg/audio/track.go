package audio

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Track is decoded, device-rate stereo PCM. StereoPCM is interleaved signed
// 16-bit little-endian data: left, right, left, right. Unlike Sample, Track
// preserves the two source channels. Music is streamed through a retained
// player and has no positional down-mix.
type Track struct {
	Rate      int
	StereoPCM []byte
}

// Frames reports the number of complete stereo frames in the track.
func (t Track) Frames() int { return len(t.StereoPCM) / 4 }

// DecodeTrackWAV decodes an uncompressed mono or stereo PCM RIFF/WAVE into
// interleaved 16-bit stereo at rate. Mono is duplicated. Stereo keeps left and
// right distinct. Unknown chunks are skipped and malformed sizes are refused.
func DecodeTrackWAV(b []byte, rate int) (Track, error) {
	if rate <= 0 {
		return Track{}, fmt.Errorf("audio: target rate %d, want a positive rate", rate)
	}
	if len(b) < 12 {
		return Track{}, fmt.Errorf("audio: %d byte(s), too short for a RIFF header", len(b))
	}
	if string(b[0:4]) != "RIFF" {
		return Track{}, fmt.Errorf("audio: stream begins %q, want RIFF", b[0:4])
	}
	if string(b[8:12]) != "WAVE" {
		return Track{}, fmt.Errorf("audio: RIFF form is %q, want WAVE", b[8:12])
	}

	var (
		haveFmt                 bool
		channels, bitsPerSample int
		sourceRate              int
		data                    []byte
	)
	for pos := 12; pos+8 <= len(b); {
		id := string(b[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(b[pos+4 : pos+8]))
		pos += 8
		if size < 0 || pos+size > len(b) {
			return Track{}, fmt.Errorf("audio: %s chunk claims %d byte(s), only %d remain", id, size, len(b)-pos)
		}
		body := b[pos : pos+size]
		switch id {
		case "fmt ":
			if len(body) < 16 {
				return Track{}, fmt.Errorf("audio: fmt chunk is %d byte(s), want at least 16", len(body))
			}
			format := binary.LittleEndian.Uint16(body[0:2])
			if format != riffFormatPCM {
				return Track{}, fmt.Errorf("audio: format tag %d, want %d (PCM)", format, riffFormatPCM)
			}
			channels = int(binary.LittleEndian.Uint16(body[2:4]))
			if channels != 1 && channels != 2 {
				return Track{}, fmt.Errorf("audio: %d channel(s), want 1 or 2", channels)
			}
			sourceRate = int(binary.LittleEndian.Uint32(body[4:8]))
			if sourceRate <= 0 {
				return Track{}, fmt.Errorf("audio: fmt chunk declares a sample rate of %d", sourceRate)
			}
			bitsPerSample = int(binary.LittleEndian.Uint16(body[14:16]))
			if bitsPerSample != 8 && bitsPerSample != 16 {
				return Track{}, fmt.Errorf("audio: %d bit(s) per sample, want 8 or 16", bitsPerSample)
			}
			haveFmt = true
		case "data":
			data = body
		}
		pos += size
		if size%2 == 1 {
			pos++
		}
	}
	if !haveFmt {
		return Track{}, errors.New("audio: no fmt chunk")
	}
	if data == nil {
		return Track{}, errors.New("audio: no data chunk")
	}

	bytesPerSample := bitsPerSample / 8
	frameSize := channels * bytesPerSample
	frames := len(data) / frameSize
	if channels == 2 && bitsPerSample == 16 && sourceRate == rate {
		// The shipped MUSIC.RES population takes this identity path. Retaining
		// the data slice avoids a second roughly track-sized allocation; b is
		// already caller-owned and the 44-byte WAVE header is the only extra
		// residency it keeps with the current track.
		return Track{Rate: rate, StereoPCM: data[:frames*4]}, nil
	}

	source := make([]int16, frames*2)
	for i := 0; i < frames; i++ {
		base := i * frameSize
		left := int16(widen(data, base, bitsPerSample))
		right := left
		if channels == 2 {
			right = int16(widen(data, base+bytesPerSample, bitsPerSample))
		}
		source[2*i], source[2*i+1] = left, right
	}

	outFrames := int64(frames) * int64(rate) / int64(sourceRate)
	out := make([]byte, int(outFrames)*4)
	for i := int64(0); i < outFrames; i++ {
		from := i * int64(sourceRate) / int64(rate)
		binary.LittleEndian.PutUint16(out[4*i:], uint16(source[2*from]))
		binary.LittleEndian.PutUint16(out[4*i+2:], uint16(source[2*from+1]))
	}
	return Track{Rate: rate, StereoPCM: out}, nil
}
