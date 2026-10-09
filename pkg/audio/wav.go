package audio

import (
	"fmt"

	"againrom/pkg/formats/wav"
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

// DecodeWAV decodes one uncompressed PCM WAV through wav.Parse into mono
// signed-16 PCM at rate, the caller's device rate (AC-13). Two channels are
// averaged; a partial final frame is not decoded. A non-positive target rate is
// refused, so resample never divides by zero.
func DecodeWAV(b []byte, rate int) (Sample, error) {
	if rate <= 0 {
		return Sample{}, fmt.Errorf("audio: target rate %d, want a positive rate", rate)
	}
	p, err := wav.Parse(b)
	if err != nil {
		return Sample{}, err
	}
	pcm := make([]int16, p.Frames())
	for i := range pcm {
		if p.Channels == 1 {
			pcm[i] = p.Sample(i, 0)
			continue
		}
		pcm[i] = int16((int32(p.Sample(i, 0)) + int32(p.Sample(i, 1))) / 2)
	}
	return Sample{Rate: rate, PCM: resample(pcm, p.Rate, rate)}, nil
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
