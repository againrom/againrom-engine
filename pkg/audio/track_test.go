package audio

import (
	"encoding/binary"
	"testing"
)

func TestDecodeTrackWAVPreservesStereoChannels(t *testing.T) {
	wav := buildWAV(2, 16, DeviceRate, le16(100, -300, 200, -400), false)
	track, err := DecodeTrackWAV(wav, DeviceRate)
	if err != nil {
		t.Fatalf("DecodeTrackWAV: %v", err)
	}
	if track.Rate != DeviceRate || track.Frames() != 2 {
		t.Fatalf("track = rate %d, frames %d; want %d, 2", track.Rate, track.Frames(), DeviceRate)
	}
	got := make([]int16, len(track.StereoPCM)/2)
	for i := range got {
		got[i] = int16(binary.LittleEndian.Uint16(track.StereoPCM[2*i:]))
	}
	want := []int16{100, -300, 200, -400}
	if !equalPCM(got, want) {
		t.Fatalf("stereo words = %v, want %v", got, want)
	}
}

func TestDecodeTrackWAVDuplicatesMonoAndResamplesByFrame(t *testing.T) {
	wav := buildWAV(1, 16, 2*DeviceRate, le16(10, 20, 30, 40), false)
	track, err := DecodeTrackWAV(wav, DeviceRate)
	if err != nil {
		t.Fatalf("DecodeTrackWAV: %v", err)
	}
	got := make([]int16, len(track.StereoPCM)/2)
	for i := range got {
		got[i] = int16(binary.LittleEndian.Uint16(track.StereoPCM[2*i:]))
	}
	want := []int16{10, 10, 30, 30}
	if !equalPCM(got, want) {
		t.Fatalf("stereo words = %v, want %v", got, want)
	}
}

func TestDecodeTrackWAVMalformedIsAnError(t *testing.T) {
	full := buildWAV(2, 16, DeviceRate, le16(1, 2, 3, 4), false)
	for n := 0; n < len(full); n++ {
		if _, err := DecodeTrackWAV(full[:n], DeviceRate); err == nil {
			t.Fatalf("%d of %d byte(s) decoded without error", n, len(full))
		}
	}
	if _, err := DecodeTrackWAV(full, DeviceRate); err != nil {
		t.Fatalf("complete track: %v", err)
	}
}
