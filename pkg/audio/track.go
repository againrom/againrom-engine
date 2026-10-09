package audio

import (
	"encoding/binary"
	"fmt"

	"againrom/pkg/formats/wav"
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

// DecodeTrackWAV decodes an uncompressed mono or stereo PCM WAV through
// wav.Parse into interleaved 16-bit stereo at rate. Mono is duplicated; stereo
// keeps left and right distinct.
func DecodeTrackWAV(b []byte, rate int) (Track, error) {
	if rate <= 0 {
		return Track{}, fmt.Errorf("audio: target rate %d, want a positive rate", rate)
	}
	p, err := wav.Parse(b)
	if err != nil {
		return Track{}, err
	}
	frames := p.Frames()
	if p.Channels == 2 && p.BitsPerSample == 16 && p.Rate == rate {
		// The shipped MUSIC.RES population takes this identity path. Retaining
		// the data slice avoids a second roughly track-sized allocation.
		return Track{Rate: rate, StereoPCM: p.Data[:frames*4]}, nil
	}
	outFrames := int64(frames) * int64(rate) / int64(p.Rate)
	out := make([]byte, int(outFrames)*4)
	for i := int64(0); i < outFrames; i++ {
		from := int(i * int64(p.Rate) / int64(rate))
		binary.LittleEndian.PutUint16(out[4*i:], uint16(p.Sample(from, 0)))
		binary.LittleEndian.PutUint16(out[4*i+2:], uint16(p.Sample(from, 1)))
	}
	return Track{Rate: rate, StereoPCM: out}, nil
}
