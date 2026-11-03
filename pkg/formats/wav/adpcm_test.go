package wav_test

import (
	"encoding/binary"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/wav"
)

var coefficients = [][2]int16{{256, 0}, {512, -256}, {0, 0}, {192, 64}, {240, 0}, {460, -208}, {392, -232}}

func adpcmFmt(channels, align, samples int, coefs [][2]int16) []byte {
	b := fmtBody(2, channels, 22050, 4)
	binary.LittleEndian.PutUint16(b[12:], uint16(align))
	b = binary.LittleEndian.AppendUint16(b, uint16(4+4*len(coefs)))
	b = binary.LittleEndian.AppendUint16(b, uint16(samples))
	b = binary.LittleEndian.AppendUint16(b, uint16(len(coefs)))
	for _, c := range coefs {
		b = binary.LittleEndian.AppendUint16(b, uint16(c[0]))
		b = binary.LittleEndian.AppendUint16(b, uint16(c[1]))
	}
	return b
}

func adpcmBlock(predictors []byte, delta, sample1, sample2 []int16, codes ...byte) []byte {
	b := append([]byte(nil), predictors...)
	for _, field := range [][]int16{delta, sample1, sample2} {
		for _, value := range field {
			b = binary.LittleEndian.AppendUint16(b, uint16(value))
		}
	}
	return append(b, codes...)
}

func checkADPCM(t *testing.T, b []byte, channels int, want []int16) {
	t.Helper()
	p, err := wav.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if p.Channels != channels || p.BitsPerSample != 16 || p.Rate != 22050 || p.Frames() != len(want)/channels {
		t.Fatalf("decoded shape: channels=%d bits=%d rate=%d frames=%d", p.Channels, p.BitsPerSample, p.Rate, p.Frames())
	}
	got := make([]int16, 0, len(want))
	for i := 0; i < p.Frames(); i++ {
		for ch := 0; ch < channels; ch++ {
			got = append(got, p.Sample(i, ch))
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("samples %v, want %v", got, want)
	}
}

// TestParseADPCMMono pins the public format's equations: sample2 precedes sample1;
// predictor 0 repeats sample1, and signed codes 1,7,-8,-1 use deltas 16,16,38,114.
// Prediction plus code*delta gives 116,228,-76,-190; delta updates divide by 256.
func TestParseADPCMMono(t *testing.T) {
	f := adpcmFmt(1, 9, 6, coefficients)
	b := adpcmBlock([]byte{0}, []int16{16}, []int16{100}, []int16{90}, 0x17, 0x8f)
	for _, chunks := range [][]chunk{{{"fmt ", f}, {"data", b}}, {{"data", b}, {"JUNK", []byte{1}}, {"fmt ", f}}} {
		checkADPCM(t, riff("WAVE", chunks...), 1, []int16{90, 100, 116, 228, -76, -190})
	}
}

// TestParseADPCMStereo uses predictor 0 on the left and 1 (2*s1-s2) on the right.
// Codes 1,-1 give 116,-142 with deltas 16,32; codes 7,-8 then give 228,-408
// with deltas 16,28. Each byte holds left in its high nibble, right in its low.
func TestParseADPCMStereo(t *testing.T) {
	f := adpcmFmt(2, 16, 4, coefficients)
	b := adpcmBlock([]byte{0, 1}, []int16{16, 32}, []int16{100, -100}, []int16{90, -90}, 0x1f, 0x78)
	checkADPCM(t, riff("WAVE", chunk{"fmt ", f}, chunk{"data", b}), 2, []int16{90, -90, 100, -100, 116, -142, 228, -408})
}

// TestParseADPCMPartialFinalBlock restarts each block from its own header and
// decodes only available nibbles of the last block, without adding silence.
func TestParseADPCMPartialFinalBlock(t *testing.T) {
	for _, channels := range []int{1, 2} {
		p := make([]byte, channels)
		d, s1, s2 := []int16{16}, []int16{100}, []int16{90}
		want := []int16{90, 100, 116, 228, -76, -190, 90, 100, 116, 228}
		align, samples := 9, 6
		if channels == 2 {
			d, s1, s2 = []int16{16, 16}, []int16{100, 100}, []int16{90, 90}
			align, samples = 16, 4
			want = []int16{90, 90, 100, 100, 116, 212, -12, 174, 90, 90, 100, 100, 116, 212}
		}
		b := adpcmBlock(p, d, s1, s2, 0x17, 0x8f)
		b = append(b, adpcmBlock(p, d, s1, s2, 0x17)...)
		checkADPCM(t, riff("WAVE", chunk{"fmt ", adpcmFmt(channels, align, samples, coefficients)}, chunk{"data", b}), channels, want)
		fact := binary.LittleEndian.AppendUint32(nil, ^uint32(0))
		checkADPCM(t, riff("WAVE", chunk{"fmt ", adpcmFmt(channels, align, samples, coefficients)}, chunk{"fact", fact}, chunk{"data", b}), channels, want)
	}
}

// TestParseADPCMCustomCoefficientsAndClipping uses pair 7=(255,0): -255/256
// truncates toward zero. Pair 0 with delta 32767 clips code 7 to 32767 and
// code -8 to -32768; the clipped sample becomes the next prediction input.
func TestParseADPCMCustomCoefficientsAndClipping(t *testing.T) {
	coefs := append(append([][2]int16(nil), coefficients...), [2]int16{255, 0})
	b := adpcmBlock([]byte{7}, []int16{16}, []int16{-1}, []int16{0}, 0)
	checkADPCM(t, riff("WAVE", chunk{"fmt ", adpcmFmt(1, 8, 4, coefs)}, chunk{"data", b}), 1, []int16{0, -1, 0, 0})
	b = adpcmBlock([]byte{0}, []int16{32767}, []int16{32000}, []int16{0}, 0x78)
	checkADPCM(t, riff("WAVE", chunk{"fmt ", adpcmFmt(1, 8, 4, coefficients)}, chunk{"data", b}), 1, []int16{0, 32000, 32767, -32768})
}

// TestParseADPCMFactAndNibblePadding excludes the unused low nibble for an odd
// sample count, and fact limits the complete stream to its declared frame count.
func TestParseADPCMFactAndNibblePadding(t *testing.T) {
	b := adpcmBlock([]byte{0}, []int16{16}, []int16{100}, []int16{90}, 0x17)
	f := adpcmFmt(1, 8, 3, coefficients)
	checkADPCM(t, riff("WAVE", chunk{"fmt ", f}, chunk{"data", b}), 1, []int16{90, 100, 116})
	f = adpcmFmt(1, 8, 4, coefficients)
	checkADPCM(t, riff("WAVE", chunk{"fact", []byte{3, 0, 0, 0}}, chunk{"data", b}, chunk{"fmt ", f}), 1, []int16{90, 100, 116})
	checkADPCM(t, riff("WAVE", chunk{"fact", []byte{5, 0, 0, 0}}, chunk{"data", b}, chunk{"fmt ", f}), 1, []int16{90, 100, 116, 228})
}

func TestParseADPCMMalformed(t *testing.T) {
	f := adpcmFmt(1, 8, 4, coefficients)
	b := adpcmBlock([]byte{0}, []int16{16}, []int16{100}, []int16{90}, 0x17)
	edit := func(at int, value uint16) []byte {
		out := append([]byte(nil), f...)
		binary.LittleEndian.PutUint16(out[at:], value)
		return out
	}
	for name, tc := range map[string]struct{ format, data []byte }{
		"missing extension":               {f[:16], b},
		"short extension":                 {f[:21], b},
		"extension overruns fmt":          {edit(16, 100), b},
		"extension excludes coefficients": {edit(16, 4), b},
		"no coefficients":                 {edit(20, 0), b},
		"coefficient overrun":             {edit(20, 65535), b},
		"short block size":                {edit(12, 6), b},
		"zero block size":                 {edit(12, 0), b},
		"too many samples":                {edit(18, 5), b},
		"too few samples":                 {edit(18, 1), b},
		"wrong bit depth":                 {edit(14, 8), b},
		"wrong channels":                  {edit(2, 3), b},
		"bad predictor":                   {f, append([]byte{7}, b[1:]...)},
		"truncated block header":          {f, b[:6]},
		"truncated next block header":     {f, append(append([]byte(nil), b...), 0)},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := wav.Parse(riff("WAVE", chunk{"fmt ", tc.format}, chunk{"data", tc.data}))
			if err == nil || !strings.Contains(err.Error(), "wav:") {
				t.Fatalf("malformed ADPCM: %v", err)
			}
		})
	}
	for _, fact := range [][]byte{{}, {1}, {1, 0, 0}} {
		if _, err := wav.Parse(riff("WAVE", chunk{"fmt ", f}, chunk{"fact", fact}, chunk{"data", b})); err == nil {
			t.Fatal("malformed fact accepted")
		}
	}
	full := riff("WAVE", chunk{"fmt ", f}, chunk{"data", b})
	for n := 0; n < len(full)-1; n++ {
		if _, err := wav.Parse(full[:n]); err == nil {
			t.Fatalf("truncated RIFF accepted at %d", n)
		}
	}
}

func TestParseADPCMUnsignedDeltaAndArithmeticLimit(t *testing.T) {
	b := adpcmBlock([]byte{0}, []int16{-1}, []int16{0}, []int16{0}, 0x1f)
	checkADPCM(t, riff("WAVE", chunk{"fmt ", adpcmFmt(1, 8, 4, coefficients)}, chunk{"data", b}), 1, []int16{0, 0, 32767, -26112})
	codes := make([]byte, 40)
	for i := range codes {
		codes[i] = 0x88
	}
	b = adpcmBlock([]byte{0}, []int16{32767}, []int16{0}, []int16{0}, codes...)
	if _, err := wav.Parse(riff("WAVE", chunk{"fmt ", adpcmFmt(1, 47, 82, coefficients)}, chunk{"data", b})); err == nil || !strings.Contains(err.Error(), "arithmetic range") {
		t.Fatalf("unbounded delta was not refused: %v", err)
	}
}

func TestParseADPCMSourceTagOnFailure(t *testing.T) {
	f := adpcmFmt(1, 8, 4, coefficients)
	p, err := wav.Parse(riff("WAVE", chunk{"fmt ", f}, chunk{"data", []byte{0}}))
	if err == nil || p.FormatTag != 2 || p.Data != nil {
		t.Fatalf("error metadata: %+v, %v", p, err)
	}
}
