package video

import (
	"bytes"
	"encoding/binary"
	"image"
	"io"
	"testing"
)

func TestIndependentWireFixture(t *testing.T) {
	// Literal ARV2, 2x1, two frames, no audio; not produced by
	// WriteHeader/WriteFrame.
	raw := []byte{'A', 'R', 'V', '2', 2, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255, 255, 255, 255, 255}
	r := bytes.NewReader(raw)
	i, err := ReadHeader(r)
	if err != nil || i != (Info{Width: 2, Height: 1, Frames: 2}) {
		t.Fatalf("header %+v %v", i, err)
	}
	a, audioA, err := ReadFrame(r, i)
	if err != nil || audioA != nil || !bytes.Equal(a.Pix, raw[24:32]) {
		t.Fatalf("first frame %v %v", err, audioA)
	}
	b, audioB, err := ReadFrame(r, i)
	if err != nil || audioB != nil || !bytes.Equal(b.Pix, raw[32:]) {
		t.Fatalf("second frame %v %v", err, audioB)
	}
	var out bytes.Buffer
	if err := WriteHeader(&out, i); err != nil {
		t.Fatal(err)
	}
	if err := WriteFrame(&out, i, a, nil); err != nil {
		t.Fatal(err)
	}
	if err := WriteFrame(&out, i, b, nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), raw) {
		t.Fatal("writer differs from independent wire bytes")
	}
}

func TestIndependentWireFixtureWithAudio(t *testing.T) {
	// Literal ARV2, 1x1, two frames, mono 22050 Hz: frame 0 carries two
	// interleaved samples (4 bytes), frame 1 carries none.
	raw := []byte{'A', 'R', 'V', '2', 1, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0}
	raw = append(raw, 0x22, 0x56, 0, 0, 1, 16, 0, 0) // AudioRate=22050, channels=1, depth=16
	raw = append(raw, 10, 20, 30, 40)                // frame 0 pixel (RGBA)
	raw = append(raw, 4, 0, 0, 0)                    // frame 0 audio length = 4
	raw = append(raw, 1, 2, 3, 4)                    // frame 0 audio bytes
	raw = append(raw, 50, 60, 70, 80)                // frame 1 pixel
	raw = append(raw, 0, 0, 0, 0)                    // frame 1 audio length = 0
	info := Info{Width: 1, Height: 1, Frames: 2, AudioRate: 22050, AudioChannels: 1, AudioBitDepth: 16}
	r := bytes.NewReader(raw)
	i, err := ReadHeader(r)
	if err != nil || i != info {
		t.Fatalf("header %+v %v", i, err)
	}
	a, audioA, err := ReadFrame(r, i)
	if err != nil || !bytes.Equal(a.Pix, []byte{10, 20, 30, 40}) || !bytes.Equal(audioA, []byte{1, 2, 3, 4}) {
		t.Fatalf("first frame pix=%v audio=%v err=%v", a, audioA, err)
	}
	b, audioB, err := ReadFrame(r, i)
	if err != nil || !bytes.Equal(b.Pix, []byte{50, 60, 70, 80}) || audioB != nil {
		t.Fatalf("second frame pix=%v audio=%v err=%v", b, audioB, err)
	}
	var out bytes.Buffer
	if err := WriteHeader(&out, i); err != nil {
		t.Fatal(err)
	}
	if err := WriteFrame(&out, i, a, []byte{1, 2, 3, 4}); err != nil {
		t.Fatal(err)
	}
	if err := WriteFrame(&out, i, b, nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), raw) {
		t.Fatal("writer differs from independent wire bytes")
	}
}

func TestHeaderAndFrameRefusals(t *testing.T) {
	for _, i := range []Info{{Width: 0, Height: 1, Frames: 1}, {Width: 1, Height: 0, Frames: 1}, {Width: 1, Height: 1, Frames: 0},
		{Width: 2049, Height: 1, Frames: 1}, {Width: 1, Height: 2049, Frames: 1}, {Width: 1, Height: 1, Frames: MaxFrames + 1},
		{Width: 0xffffffff, Height: 0xffffffff, Frames: 1},
		// Asymmetric audio fields: a rate with no channel/depth, or the reverse.
		{Width: 1, Height: 1, Frames: 1, AudioChannels: 1}, {Width: 1, Height: 1, Frames: 1, AudioBitDepth: 16},
		{Width: 1, Height: 1, Frames: 1, AudioRate: 22050}, {Width: 1, Height: 1, Frames: 1, AudioRate: 22050, AudioChannels: 3, AudioBitDepth: 16},
		{Width: 1, Height: 1, Frames: 1, AudioRate: 22050, AudioChannels: 1, AudioBitDepth: 8},
	} {
		var b [24]byte
		copy(b[:], "ARV2")
		binary.LittleEndian.PutUint32(b[4:], i.Width)
		binary.LittleEndian.PutUint32(b[8:], i.Height)
		binary.LittleEndian.PutUint32(b[12:], i.Frames)
		binary.LittleEndian.PutUint32(b[16:], i.AudioRate)
		b[20], b[21] = i.AudioChannels, i.AudioBitDepth
		if _, err := ReadHeader(bytes.NewReader(b[:])); err == nil {
			t.Fatalf("accepted %+v", i)
		}
		if err := WriteHeader(io.Discard, i); err == nil {
			t.Fatalf("wrote %+v", i)
		}
	}
	if _, err := ReadHeader(bytes.NewReader([]byte("ARV1\x01\x00\x00\x00\x01\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"))); err == nil {
		t.Fatal("accepted other version")
	}
	if _, err := ReadHeader(bytes.NewReader(nil)); err == nil {
		t.Fatal("accepted missing header")
	}
	// Non-zero reserved bytes are refused even with an otherwise valid header.
	var reserved [24]byte
	copy(reserved[:], "ARV2")
	binary.LittleEndian.PutUint32(reserved[4:], 1)
	binary.LittleEndian.PutUint32(reserved[8:], 1)
	binary.LittleEndian.PutUint32(reserved[12:], 1)
	reserved[22] = 1
	if _, err := ReadHeader(bytes.NewReader(reserved[:])); err == nil {
		t.Fatal("accepted non-zero reserved header bytes")
	}
	if _, _, err := ReadFrame(bytes.NewReader([]byte{1, 2, 3}), Info{Width: 1, Height: 1, Frames: 1}); err == nil {
		t.Fatal("accepted partial pixel")
	}
	if err := WriteFrame(io.Discard, Info{Width: 1, Height: 1, Frames: 1}, image.NewRGBA(image.Rect(0, 0, 2, 1)), nil); err == nil {
		t.Fatal("accepted dimension mismatch")
	}
	audioInfo := Info{Width: 1, Height: 1, Frames: 1, AudioRate: 22050, AudioChannels: 1, AudioBitDepth: 16}
	frame := image.NewRGBA(image.Rect(0, 0, 1, 1))
	if err := WriteFrame(io.Discard, Info{Width: 1, Height: 1, Frames: 1}, frame, []byte{1, 2}); err == nil {
		t.Fatal("accepted audio chunk on a video-only stream")
	}
	if err := WriteFrame(io.Discard, audioInfo, frame, []byte{1}); err == nil {
		t.Fatal("accepted a partial sample frame")
	}
	pixelPlusOversizedLength := append(append([]byte{}, frame.Pix...), 0xff, 0xff, 0xff, 0x7f)
	if _, _, err := ReadFrame(bytes.NewReader(pixelPlusOversizedLength), audioInfo); err == nil {
		t.Fatal("accepted an audio length beyond the bound")
	}
	pixelPlusOddLength := append(append([]byte{}, frame.Pix...), 1, 0, 0, 0)
	if _, _, err := ReadFrame(bytes.NewReader(pixelPlusOddLength), audioInfo); err == nil {
		t.Fatal("accepted an audio length that is not a whole sample frame")
	}
}
