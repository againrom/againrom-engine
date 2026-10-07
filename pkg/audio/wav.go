package audio

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Sample is decoded mono PCM at a stated rate: the playable form of one slot
// (spec Terms). PCM holds one signed 16-bit value per frame regardless of the
// bit depth or channel count the source file carried — DecodeWAV is the only
// place that width and channel count are visible at all, so everything past
// it in this package, and everything in pkg/ui and pkg/game beyond it,
// handles one shape.
type Sample struct {
	Rate int
	PCM  []int16
}

// riffFormatPCM is the one fmt-chunk format tag this decoder accepts: linear
// PCM.
const riffFormatPCM = 1

// DecodeWAV reads one RIFF/WAVE stream holding uncompressed PCM: one or two
// channels, 8 or 16 bits per sample, any source rate. Two channels are
// averaged down to one, an 8-bit unsigned sample is widened to the signed
// 16-bit range every other path in this package uses, and the result is
// resampled to rate — the caller's device rate, so a decoded Sample always
// plays at the rate the caller opened its device with (AC-13).
//
// The chunk list is WALKED rather than assumed to be fmt-then-data: a real WAV
// file may carry a LIST, a fact or a JUNK chunk before either one (padding
// chunks a great many encoders write), and this decoder skips anything it
// does not name rather than refusing a file that carries one. Every chunk's
// declared size is checked against the bytes actually remaining before that
// chunk's body is read, so a truncated or lying size is an error here and
// never a slice panic (spec AC-12). RIFF pads an odd-sized chunk body to the
// next even offset; that pad byte is skipped and is never read as data.
//
// Two chunks are required — fmt (the format) and data (the samples) — in
// either order, and either one missing or absent is an error. A stream that
// is not RIFF, or whose form type is not WAVE, is refused before any chunk is
// read. A non-positive target rate is refused too: nothing downstream of this
// function can construct one (DeviceRate is a positive constant), so a caller
// that manages to hand one in gets a named error here instead of the
// divide-by-zero it would otherwise buy in resample.
func DecodeWAV(b []byte, rate int) (Sample, error) {
	if rate <= 0 {
		return Sample{}, fmt.Errorf("audio: target rate %d, want a positive rate", rate)
	}
	if len(b) < 12 {
		return Sample{}, fmt.Errorf("audio: %d byte(s), too short for a RIFF header", len(b))
	}
	if string(b[0:4]) != "RIFF" {
		return Sample{}, fmt.Errorf("audio: stream begins %q, want RIFF", b[0:4])
	}
	if string(b[8:12]) != "WAVE" {
		return Sample{}, fmt.Errorf("audio: RIFF form is %q, want WAVE", b[8:12])
	}

	var (
		haveFmt                 bool
		channels, bitsPerSample int
		sourceRate              int
		data                    []byte
	)

	pos := 12
	for pos+8 <= len(b) {
		id := string(b[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(b[pos+4 : pos+8]))
		pos += 8
		if size < 0 || pos+size > len(b) {
			return Sample{}, fmt.Errorf("audio: %s chunk claims %d byte(s), only %d remain", id, size, len(b)-pos)
		}
		body := b[pos : pos+size]
		switch id {
		case "fmt ":
			if len(body) < 16 {
				return Sample{}, fmt.Errorf("audio: fmt chunk is %d byte(s), want at least 16", len(body))
			}
			format := binary.LittleEndian.Uint16(body[0:2])
			if format != riffFormatPCM {
				return Sample{}, fmt.Errorf("audio: format tag %d, want %d (PCM)", format, riffFormatPCM)
			}
			channels = int(binary.LittleEndian.Uint16(body[2:4]))
			if channels != 1 && channels != 2 {
				return Sample{}, fmt.Errorf("audio: %d channel(s), want 1 or 2", channels)
			}
			sourceRate = int(binary.LittleEndian.Uint32(body[4:8]))
			if sourceRate <= 0 {
				return Sample{}, fmt.Errorf("audio: fmt chunk declares a sample rate of %d", sourceRate)
			}
			bitsPerSample = int(binary.LittleEndian.Uint16(body[14:16]))
			if bitsPerSample != 8 && bitsPerSample != 16 {
				return Sample{}, fmt.Errorf("audio: %d bit(s) per sample, want 8 or 16", bitsPerSample)
			}
			haveFmt = true
		case "data":
			data = body
		}
		pos += size
		if size%2 == 1 {
			pos++ // the RIFF pad byte: not part of any chunk's data
		}
	}
	if !haveFmt {
		return Sample{}, errors.New("audio: no fmt chunk")
	}
	if data == nil {
		return Sample{}, errors.New("audio: no data chunk")
	}

	pcm := downmix(data, channels, bitsPerSample)
	return Sample{Rate: rate, PCM: resample(pcm, sourceRate, rate)}, nil
}

// downmix turns the data chunk's raw bytes into mono signed-16 PCM, one value
// per frame. A data chunk whose declared length does not land on a whole
// number of frames — nothing here assumes it will — simply leaves the partial
// tail undecoded rather than reading past it: frames is a floor division, so
// every byte offset this loop touches was already inside data.
//
// Averaging two channels and widening an 8-bit unsigned sample to the signed
// 16-bit range COMMUTE: the WAV format's own zero point is 128 for 8-bit and 0
// for 16-bit, and subtracting 128 before or after averaging a stereo pair
// yields the same value either way. This widens each channel into the signed
// range first, so the mono and stereo paths share one averaging step instead
// of the bit depth needing two separate ones.
func downmix(data []byte, channels, bitsPerSample int) []int16 {
	bytesPerSample := bitsPerSample / 8
	frameSize := bytesPerSample * channels
	frames := len(data) / frameSize
	pcm := make([]int16, frames)
	for i := 0; i < frames; i++ {
		base := i * frameSize
		left := widen(data, base, bitsPerSample)
		if channels == 1 {
			pcm[i] = int16(left)
			continue
		}
		right := widen(data, base+bytesPerSample, bitsPerSample)
		pcm[i] = int16((left + right) / 2)
	}
	return pcm
}

// widen reads one channel's sample at byte offset off into the signed 16-bit
// range: a 16-bit sample is read as-is (WAV's own signed representation), and
// an 8-bit sample is WAV's own UNSIGNED representation, with 128 as silence,
// shifted up to fill the same range a 16-bit silence (0) sits in the middle
// of.
func widen(data []byte, off, bitsPerSample int) int32 {
	if bitsPerSample == 16 {
		return int32(int16(binary.LittleEndian.Uint16(data[off:])))
	}
	return (int32(data[off]) - 128) << 8
}

// resample is nearest-neighbour: output frame i is input frame i*from/to,
// computed in int64 so a sample long enough to overflow a 32-bit product still
// resamples correctly (plan T1). See doc.go for why nearest neighbour and not
// an interpolating resampler — it is exact integer arithmetic, which is what
// keeps AC-13 an exact frame count rather than a tolerance.
//
// For every i in the output it produces, i*from/to is provably < len(in): the
// output length n is floor(len(in)*to/from), so for i <= n-1, i*from/to (real
// division) is strictly less than len(in), and the floor of a value strictly
// less than an integer is at most that integer minus one. No bounds check is
// needed for that reason, and none is written.
func resample(in []int16, from, to int) []int16 {
	if from == to {
		return in
	}
	n := int64(len(in)) * int64(to) / int64(from)
	out := make([]int16, n)
	for i := range out {
		out[i] = in[int64(i)*int64(from)/int64(to)]
	}
	return out
}
