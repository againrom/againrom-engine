package audio

import (
	"bytes"
	"testing"
)

func onAxisSample(vals ...int16) Sample {
	return Sample{Rate: DeviceRate, PCM: vals}
}

// AC-10: mute yields no play, and master volume zero yields no play.
func TestStereoMuteAndZeroMasterYieldNoPlay(t *testing.T) {
	s := onAxisSample(1000, -1000, 2000)
	p := Placement{Left: GainUnit, Right: GainUnit}

	if buf := Stereo(s, p, Settings{Master: MasterUnit, Muted: true}); buf != nil {
		t.Fatalf("muted: got a %d-byte buffer, want nil", len(buf))
	}
	if buf := Stereo(s, p, Settings{Master: 0, Muted: false}); buf != nil {
		t.Fatalf("master zero: got a %d-byte buffer, want nil", len(buf))
	}
	// A Placement beyond the falloff radius, built directly with both sides at
	// zero (as Place itself never returns, since it reports false instead —
	// see placement_test.go), plays nothing either, for the same reason.
	if buf := Stereo(s, Placement{}, DefaultSettings); buf != nil {
		t.Fatalf("zero Placement: got a %d-byte buffer, want nil", len(buf))
	}
}

// AC-10: halving the master volume halves the amplitude of the synthesised
// buffer.
func TestStereoHalvingMasterHalvesAmplitude(t *testing.T) {
	s := onAxisSample(1000, -2000, 4000)
	p := Placement{Left: GainUnit, Right: GainUnit}

	full := Stereo(s, p, Settings{Master: MasterUnit, Muted: false})
	half := Stereo(s, p, Settings{Master: MasterUnit / 2, Muted: false})

	if full == nil || half == nil {
		t.Fatalf("got nil buffer(s): full=%v half=%v", full, half)
	}
	if len(full) != len(half) {
		t.Fatalf("len(full)=%d len(half)=%d, want equal", len(full), len(half))
	}

	fullSamples := decodeInterleaved16(full)
	halfSamples := decodeInterleaved16(half)
	for i := range fullSamples {
		want := fullSamples[i] / 2
		if halfSamples[i] != want {
			t.Fatalf("sample %d: full=%d half=%d, want half == full/2 == %d",
				i, fullSamples[i], halfSamples[i], want)
		}
	}

	// Full volume at full Placement gain reproduces the source exactly: the
	// identity of the whole scaling chain when neither factor attenuates.
	wantFull := []int16{1000, 1000, -2000, -2000, 4000, 4000}
	for i, v := range wantFull {
		if fullSamples[i] != v {
			t.Fatalf("full-volume sample %d = %d, want %d (source unchanged)", i, fullSamples[i], v)
		}
	}
}

func decodeInterleaved16(buf []byte) []int16 {
	out := make([]int16, len(buf)/2)
	for i := range out {
		out[i] = int16(uint16(buf[2*i]) | uint16(buf[2*i+1])<<8)
	}
	return out
}

// TestStereoClampsTheMaster is the reconciliation's own assertion (0126
// verification): the master arrives from a command-line integer with no range,
// so Stereo must be total over it. A negative master must be SILENCE and not
// an inverted copy at the same loudness, and a master past MasterUnit must be
// full volume and not clipping.
func TestStereoClampsTheMaster(t *testing.T) {
	s := Sample{Rate: DeviceRate, PCM: []int16{1000, -1000, 20000, -20000}}
	full := Placement{Left: GainUnit, Right: GainUnit}

	if got := Stereo(s, full, Settings{Master: -100}); got != nil {
		t.Errorf("a master of -100 produced %d bytes, want none", len(got))
	}
	if got := Stereo(s, full, Settings{Master: -1}); got != nil {
		t.Errorf("a master of -1 produced %d bytes, want none", len(got))
	}

	want := Stereo(s, full, Settings{Master: MasterUnit})
	if want == nil {
		t.Fatal("a full master produced no buffer")
	}
	for _, over := range []int{MasterUnit + 1, 500, 1 << 20} {
		got := Stereo(s, full, Settings{Master: over})
		if !bytes.Equal(got, want) {
			t.Errorf("a master of %d differs from a full master; want the clamp", over)
		}
	}
}

// TestStereoPutsTheLeftChannelInTheFirstWordOfEachFrame pins WHICH physical
// output word carries which channel. The gain computation is witnessed by
// the tests above; the channel identity was not, because each of them places
// the sound symmetrically and a symmetric placement is byte-identical under
// a swap of the two writes. Swapping them is an audible defect that passed
// every test in this module until this test was written (test audit).
//
// The wanted values are derived from the definitions rather than recomputed:
// scaleGain(g, MasterUnit) is g*MasterUnit/MasterUnit = g, and scaleSample(v,
// g) is v*g/GainUnit. So at full master, a Left of GainUnit passes a sample
// through unchanged and a Right of GainUnit/4 quarters it. Recomputing them
// with the package's own helpers would assert only that the code equals
// itself.
func TestStereoPutsTheLeftChannelInTheFirstWordOfEachFrame(t *testing.T) {
	s := onAxisSample(1000, -2000)
	p := Placement{Left: GainUnit, Right: GainUnit / 4}

	buf := Stereo(s, p, DefaultSettings)
	if buf == nil {
		t.Fatal("got a nil buffer for an audible placement")
	}
	if len(buf) != 4*len(s.PCM) {
		t.Fatalf("len(buf)=%d, want %d (two 16-bit words per PCM value)",
			len(buf), 4*len(s.PCM))
	}

	// decodeInterleaved16 walks 16-bit words in order, so the even indices are
	// each frame's first word and the odd ones its second.
	got := decodeInterleaved16(buf)
	want := []int16{1000, 250, -2000, -500}
	if len(got) != len(want) {
		t.Fatalf("decoded %d words, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("word %d = %d, want %d (frame %d, %s channel)",
				i, got[i], want[i], i/2, channelName(i))
		}
	}
}

// channelName names the half of a frame an index falls in, so a failure above
// reads as a channel rather than as a word number.
func channelName(word int) string {
	if word%2 == 0 {
		return "left"
	}
	return "right"
}
