package smacker

import (
	"testing"
	"time"
)

// The header interval maps to units of 10 microseconds: a non-negative value
// is milliseconds times 100 modulo 2^32, a negative one its magnitude. Zero
// stays zero (no default), and nothing is floored or capped.
func TestIntervalUnitsFromHeader(t *testing.T) {
	for _, c := range []struct {
		raw  int32
		want uint32
	}{
		{0, 0},
		{-1, 1},
		{1, 100},
		{100, 10000},
		{-6666, 6666},
		{-4000, 4000},
		{-100000, 100000},
		{2147483647, 4294967196},
		{42949673, 4},
		{-2147483648, 2147483648},
	} {
		if got := intervalUnits(c.raw); got != c.want {
			t.Errorf("intervalUnits(%d) = %d, want %d", c.raw, got, c.want)
		}
	}
}

func TestIntervalDurationHasNoDefault(t *testing.T) {
	d := &Decoder{c: &container{frameRateRaw: 0}}
	if d.Interval() != 0 {
		t.Fatalf("zero header interval = %v, want 0", d.Interval())
	}
	d.c.frameRateRaw = -6666
	if d.Interval() != 66660*time.Microsecond {
		t.Fatalf("interval = %v, want 66.66ms", d.Interval())
	}
}

func TestAudioDecodableRefusesBinkAndAbsent(t *testing.T) {
	d := &Decoder{c: &container{}}
	d.c.audio[0] = audioTrack{Exists: true, Channels: 2, BitDepth: 16, Rate: 22050, compress: 1}
	d.c.audio[1] = audioTrack{Exists: true, Channels: 2, BitDepth: 16, Rate: 22050, compress: 2}
	d.c.audio[2] = audioTrack{Exists: true, Channels: 1, BitDepth: 16, Rate: 22050, compress: 0}
	for i, want := range []bool{true, false, true, false} {
		if got := d.AudioDecodable(i); got != want {
			t.Errorf("AudioDecodable(%d) = %v, want %v", i, got, want)
		}
	}
	if d.AudioDecodable(-1) || d.AudioDecodable(7) {
		t.Error("out-of-range track reported decodable")
	}
}
