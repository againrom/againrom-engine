package wav_test

import (
	"encoding/binary"
	"strings"
	"testing"

	"againrom/pkg/formats/wav"
)

type chunk struct {
	id   string
	body []byte
}

func fmtBody(format, channels, rate, bits int) []byte {
	b := make([]byte, 16)
	binary.LittleEndian.PutUint16(b[0:], uint16(format))
	binary.LittleEndian.PutUint16(b[2:], uint16(channels))
	binary.LittleEndian.PutUint32(b[4:], uint32(rate))
	binary.LittleEndian.PutUint16(b[14:], uint16(bits))
	return b
}

func riff(form string, chunks ...chunk) []byte {
	out := []byte("RIFF\x00\x00\x00\x00" + form)
	for _, c := range chunks {
		out = append(out, c.id...)
		out = binary.LittleEndian.AppendUint32(out, uint32(len(c.body)))
		out = append(out, c.body...)
		if len(c.body)%2 == 1 {
			out = append(out, 0xEE)
		}
	}
	binary.LittleEndian.PutUint32(out[4:], uint32(len(out)-8))
	return out
}

func TestParseWalksChunksInEitherOrderAndSkipsUnknownOnes(t *testing.T) {
	data := []byte{1, 0, 2, 0, 3, 0}
	for name, b := range map[string][]byte{
		"fmt first":  riff("WAVE", chunk{"fmt ", fmtBody(1, 1, 22050, 16)}, chunk{"LIST", []byte{9, 9, 9}}, chunk{"data", data}),
		"data first": riff("WAVE", chunk{"JUNK", []byte{7}}, chunk{"data", data}, chunk{"fmt ", fmtBody(1, 1, 22050, 16)}),
	} {
		p, err := wav.Parse(b)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if p.Channels != 1 || p.BitsPerSample != 16 || p.Rate != 22050 || string(p.Data) != string(data) {
			t.Fatalf("%s: %+v", name, p)
		}
		if p.Frames() != 3 || p.Sample(2, 0) != 3 || p.Sample(2, 1) != 3 {
			t.Fatalf("%s: frames %d, sample %d/%d", name, p.Frames(), p.Sample(2, 0), p.Sample(2, 1))
		}
	}
}

func TestSampleWidensEightBitAndSeparatesStereoChannels(t *testing.T) {
	p, err := wav.Parse(riff("WAVE", chunk{"fmt ", fmtBody(1, 2, 8000, 8)}, chunk{"data", []byte{128, 255, 0, 129, 7}}))
	if err != nil {
		t.Fatal(err)
	}
	if p.Frames() != 2 {
		t.Fatalf("frames %d, want 2 (the odd tail byte is not a frame)", p.Frames())
	}
	want := [][2]int16{{0, 127 << 8}, {-128 << 8, 1 << 8}}
	for i, w := range want {
		if l, r := p.Sample(i, 0), p.Sample(i, 1); l != w[0] || r != w[1] {
			t.Fatalf("frame %d = %d,%d, want %d,%d", i, l, r, w[0], w[1])
		}
	}
	p16, err := wav.Parse(riff("WAVE", chunk{"fmt ", fmtBody(1, 2, 8000, 16)}, chunk{"data", []byte{0x00, 0x80, 0xff, 0x7f}}))
	if err != nil {
		t.Fatal(err)
	}
	if p16.Sample(0, 0) != -32768 || p16.Sample(0, 1) != 32767 {
		t.Fatalf("16-bit stereo = %d,%d", p16.Sample(0, 0), p16.Sample(0, 1))
	}
}

func TestParseRefusesEveryMalformedShape(t *testing.T) {
	good := fmtBody(1, 1, 8000, 8)
	cases := map[string]struct {
		b    []byte
		want string
	}{
		"short":       {[]byte("RIFF"), "too short"},
		"not riff":    {append([]byte("RIFX"), riff("WAVE")[4:]...), "want RIFF"},
		"not wave":    {riff("AVI "), "want WAVE"},
		"no fmt":      {riff("WAVE", chunk{"data", []byte{1}}), "no fmt"},
		"no data":     {riff("WAVE", chunk{"fmt ", good}), "no data"},
		"short fmt":   {riff("WAVE", chunk{"fmt ", good[:14]}), "at least 16"},
		"not pcm":     {riff("WAVE", chunk{"fmt ", fmtBody(3, 1, 8000, 8)}), "format tag"},
		"channels":    {riff("WAVE", chunk{"fmt ", fmtBody(1, 3, 8000, 8)}), "channel"},
		"zero rate":   {riff("WAVE", chunk{"fmt ", fmtBody(1, 1, 0, 8)}), "sample rate"},
		"bits":        {riff("WAVE", chunk{"fmt ", fmtBody(1, 1, 8000, 24)}), "bit(s)"},
		"lying chunk": {append(riff("WAVE", chunk{"fmt ", good}), 'd', 'a', 't', 'a', 0xff, 0, 0, 0), "claims"},
	}
	for name, c := range cases {
		if _, err := wav.Parse(c.b); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error %v, want one naming %q", name, err, c.want)
		}
	}
}

func TestParseNeverPanicsOnTruncation(t *testing.T) {
	full := riff("WAVE", chunk{"fmt ", fmtBody(1, 2, 8000, 16)}, chunk{"data", make([]byte, 33)})
	for n := 0; n < len(full); n++ {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic at %d bytes: %v", n, r)
				}
			}()
			_, _ = wav.Parse(full[:n])
		}()
	}
}
