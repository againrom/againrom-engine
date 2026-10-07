package audio

import (
	"encoding/binary"
	"testing"
)

// buildWAV assembles a minimal RIFF/WAVE stream — a fmt chunk and a data
// chunk in the given order, around the given channel count, bit depth,
// source rate and raw sample bytes. It is the whole fixture layer this file's
// tests use (tasks.md T1, "a twenty-line helper"): no game file is read here
// and none is committed anywhere in this repository.
func buildWAV(channels, bitsPerSample, rate int, data []byte, dataFirst bool) []byte {
	fmtChunk := make([]byte, 16)
	binary.LittleEndian.PutUint16(fmtChunk[0:], riffFormatPCM)
	binary.LittleEndian.PutUint16(fmtChunk[2:], uint16(channels))
	binary.LittleEndian.PutUint32(fmtChunk[4:], uint32(rate))
	blockAlign := channels * bitsPerSample / 8
	binary.LittleEndian.PutUint32(fmtChunk[8:], uint32(rate*blockAlign)) // byte rate, read by nothing here
	binary.LittleEndian.PutUint16(fmtChunk[12:], uint16(blockAlign))
	binary.LittleEndian.PutUint16(fmtChunk[14:], uint16(bitsPerSample))

	chunk := func(id string, body []byte) []byte {
		out := make([]byte, 0, 8+len(body)+1)
		out = append(out, id...)
		var size [4]byte
		binary.LittleEndian.PutUint32(size[:], uint32(len(body)))
		out = append(out, size[:]...)
		out = append(out, body...)
		if len(body)%2 == 1 {
			out = append(out, 0) // the RIFF pad byte
		}
		return out
	}

	var chunks []byte
	if dataFirst {
		chunks = append(chunk("data", data), chunk("fmt ", fmtChunk)...)
	} else {
		chunks = append(chunk("fmt ", fmtChunk), chunk("data", data)...)
	}

	out := make([]byte, 0, 12+len(chunks))
	out = append(out, "RIFF"...)
	var riffSize [4]byte
	binary.LittleEndian.PutUint32(riffSize[:], uint32(4+len(chunks)))
	out = append(out, riffSize[:]...)
	out = append(out, "WAVE"...)
	out = append(out, chunks...)
	return out
}

// le16 packs int16 values as little-endian bytes: the data chunk body for a
// 16-bit stream.
func le16(vals ...int16) []byte {
	out := make([]byte, 2*len(vals))
	for i, v := range vals {
		binary.LittleEndian.PutUint16(out[2*i:], uint16(v))
	}
	return out
}

func TestDecodeWAV16BitMono(t *testing.T) {
	b := buildWAV(1, 16, DeviceRate, le16(100, -200, 300), false)
	s, err := DecodeWAV(b, DeviceRate)
	if err != nil {
		t.Fatalf("DecodeWAV: %v", err)
	}
	if s.Rate != DeviceRate {
		t.Fatalf("Rate = %d, want %d", s.Rate, DeviceRate)
	}
	want := []int16{100, -200, 300}
	if !equalPCM(s.PCM, want) {
		t.Fatalf("PCM = %v, want %v", s.PCM, want)
	}
}

func TestDecodeWAV8BitMono(t *testing.T) {
	// 128 is 8-bit silence -> 0; 255 -> (255-128)<<8 = 32512; 0 -> (0-128)<<8 = -32768.
	b := buildWAV(1, 8, DeviceRate, []byte{128, 255, 0}, false)
	s, err := DecodeWAV(b, DeviceRate)
	if err != nil {
		t.Fatalf("DecodeWAV: %v", err)
	}
	want := []int16{0, 32512, -32768}
	if !equalPCM(s.PCM, want) {
		t.Fatalf("PCM = %v, want %v", s.PCM, want)
	}
}

func TestDecodeWAV16BitStereoDownmix(t *testing.T) {
	// Frame 0: L=100, R=300 -> 200. Frame 1: L=-200, R=-400 -> -300.
	b := buildWAV(2, 16, DeviceRate, le16(100, 300, -200, -400), false)
	s, err := DecodeWAV(b, DeviceRate)
	if err != nil {
		t.Fatalf("DecodeWAV: %v", err)
	}
	want := []int16{200, -300}
	if !equalPCM(s.PCM, want) {
		t.Fatalf("PCM = %v, want %v", s.PCM, want)
	}
}

// The chunk walk names no order: a stream carrying data before fmt decodes
// exactly as one carrying them the other way round, which is what the RIFF
// container actually allows and what DecodeWAV's own comment claims.
func TestDecodeWAVAcceptsEitherChunkOrder(t *testing.T) {
	forward := buildWAV(1, 16, DeviceRate, le16(1, 2, 3), false)
	reversed := buildWAV(1, 16, DeviceRate, le16(1, 2, 3), true)
	sf, err := DecodeWAV(forward, DeviceRate)
	if err != nil {
		t.Fatalf("DecodeWAV(fmt-first): %v", err)
	}
	sr, err := DecodeWAV(reversed, DeviceRate)
	if err != nil {
		t.Fatalf("DecodeWAV(data-first): %v", err)
	}
	if !equalPCM(sf.PCM, sr.PCM) {
		t.Fatalf("chunk order changed the decode: %v vs %v", sf.PCM, sr.PCM)
	}
}

func TestDecodeWAVRefusesNonPCMFormat(t *testing.T) {
	b := buildWAV(1, 16, DeviceRate, le16(1, 2, 3), false)
	binary.LittleEndian.PutUint16(b[20:], 3) // IEEE float tag, in place of PCM's 1
	if _, err := DecodeWAV(b, DeviceRate); err == nil {
		t.Fatal("a non-PCM format tag decoded without error")
	}
}

func TestDecodeWAVRefusesMissingDataChunk(t *testing.T) {
	fmtChunk := make([]byte, 16)
	binary.LittleEndian.PutUint16(fmtChunk[0:], riffFormatPCM)
	binary.LittleEndian.PutUint16(fmtChunk[2:], 1)
	binary.LittleEndian.PutUint32(fmtChunk[4:], DeviceRate)
	binary.LittleEndian.PutUint16(fmtChunk[12:], 2)
	binary.LittleEndian.PutUint16(fmtChunk[14:], 16)

	out := append([]byte("RIFF"), make([]byte, 4)...)
	out = append(out, "WAVE"...)
	out = append(out, "fmt "...)
	var size [4]byte
	binary.LittleEndian.PutUint32(size[:], uint32(len(fmtChunk)))
	out = append(out, size[:]...)
	out = append(out, fmtChunk...)
	binary.LittleEndian.PutUint32(out[4:], uint32(len(out)-8))

	if _, err := DecodeWAV(out, DeviceRate); err == nil {
		t.Fatal("a stream with no data chunk decoded without error")
	}
}

// Refused rather than panicking (spec AC-12): every prefix of a well-formed
// stream, and a handful of hand-truncated chunk headers, must return an error
// and never panic. This is the totality half of AC-12, mirrored on the same
// idea pal_test.go's TestDecodeIsTotal pins for pkg/formats/pal.
func TestDecodeWAVRefusesTruncationWithoutPanicking(t *testing.T) {
	full := buildWAV(2, 16, DeviceRate, le16(1, 2, 3, 4, 5, 6), false)
	for n := 0; n <= len(full); n++ {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("DecodeWAV panicked at %d byte(s): %v", n, r)
				}
			}()
			_, _ = DecodeWAV(full[:n], DeviceRate)
		}()
	}
	// The one prefix that must succeed is the whole stream; every one short of
	// it must report an error, since every chunk it can be cut inside of is
	// required.
	if _, err := DecodeWAV(full, DeviceRate); err != nil {
		t.Fatalf("the untruncated stream itself failed to decode: %v", err)
	}
	for n := 0; n < len(full); n++ {
		if _, err := DecodeWAV(full[:n], DeviceRate); err == nil {
			t.Fatalf("%d of %d byte(s) decoded without error", n, len(full))
		}
	}
}

func TestDecodeWAVRefusesNonPositiveTargetRate(t *testing.T) {
	full := buildWAV(1, 16, DeviceRate, le16(1, 2, 3), false)
	for _, rate := range []int{0, -1, -22050} {
		if _, err := DecodeWAV(full, rate); err == nil {
			t.Fatalf("target rate %d decoded without error", rate)
		}
	}
}

// AC-13: a sample at twice the device rate decodes to half as many frames;
// one already at the device rate decodes to exactly its own frame count.
func TestDecodeWAVResamplesToDeviceRate(t *testing.T) {
	src := le16(0, 100, 200, 300, 400, 500, 600, 700) // 8 frames
	b := buildWAV(1, 16, 2*DeviceRate, src, false)
	s, err := DecodeWAV(b, DeviceRate)
	if err != nil {
		t.Fatalf("DecodeWAV: %v", err)
	}
	if len(s.PCM) != 4 {
		t.Fatalf("%d frame(s) at twice the device rate, want 4", len(s.PCM))
	}

	same := buildWAV(1, 16, DeviceRate, src, false)
	s2, err := DecodeWAV(same, DeviceRate)
	if err != nil {
		t.Fatalf("DecodeWAV: %v", err)
	}
	if len(s2.PCM) != 8 {
		t.Fatalf("%d frame(s) already at the device rate, want 8 (its own)", len(s2.PCM))
	}
}

// AC-13's identity case: resample at equal rates returns the SAME backing
// array rather than a copy. This is checked at resample's own level because
// DecodeWAV's public result offers no way to observe allocation.
func TestResampleIdentityReturnsSameBackingArray(t *testing.T) {
	in := []int16{1, 2, 3, 4, 5}
	out := resample(in, DeviceRate, DeviceRate)
	if len(out) != len(in) {
		t.Fatalf("len(out) = %d, want %d", len(out), len(in))
	}
	if &out[0] != &in[0] {
		t.Fatal("resample at equal rates allocated a new array instead of returning the input")
	}
}

func TestResampleHalvesFrameCountAtTwiceRate(t *testing.T) {
	in := make([]int16, 10)
	for i := range in {
		in[i] = int16(i * 10)
	}
	out := resample(in, 2*DeviceRate, DeviceRate)
	if len(out) != 5 {
		t.Fatalf("len(out) = %d, want 5", len(out))
	}
}

func equalPCM(got, want []int16) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
