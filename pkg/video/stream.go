// Package video carries presentation frames across the optional decoder process
// boundary. This protocol is authored by Againrom, not a game file format.
package video

import (
	"encoding/binary"
	"fmt"
	"image"
	"io"
)

const (
	MaxDimension = 2048
	MaxFrames    = 100000
	// MaxAudioChunk bounds one frame's audio payload, on both the write and the
	// read side.
	MaxAudioChunk = 1 << 20
)

// Info fixes every allocation before frame data is accepted. Frames are RGBA,
// tightly packed and top-down. The decoder paces delivery; no simulation clock
// or installed struct crosses this boundary.
type Info struct {
	Width, Height, Frames        uint32
	AudioRate                    uint32
	AudioChannels, AudioBitDepth uint8
}

func (i Info) Validate() error {
	if i.Width == 0 || i.Height == 0 || i.Width > MaxDimension || i.Height > MaxDimension || i.Frames == 0 || i.Frames > MaxFrames {
		return fmt.Errorf("video: invalid stream dimensions/count %dx%d/%d", i.Width, i.Height, i.Frames)
	}
	if i.AudioRate == 0 {
		if i.AudioChannels != 0 || i.AudioBitDepth != 0 {
			return fmt.Errorf("video: audio channel/depth set with no audio rate")
		}
		return nil
	}
	if i.AudioChannels != 1 && i.AudioChannels != 2 {
		return fmt.Errorf("video: invalid audio channel count %d", i.AudioChannels)
	}
	if i.AudioBitDepth != 16 {
		return fmt.Errorf("video: unsupported audio bit depth %d", i.AudioBitDepth)
	}
	return nil
}

// HasAudio reports whether every frame of this stream carries a (possibly
// empty) length-prefixed audio segment. See ReadFrame/WriteFrame.
func (i Info) HasAudio() bool { return i.AudioRate != 0 }

// audioFrameBytes is the byte size of one interleaved sample frame: 2 bytes
// per channel, 16-bit only (Info.Validate already refuses any other depth).
func (i Info) audioFrameBytes() uint32 { return uint32(i.AudioChannels) * 2 }

func WriteHeader(w io.Writer, info Info) error {
	if err := info.Validate(); err != nil {
		return err
	}
	var b [24]byte
	copy(b[:4], "ARV2")
	binary.LittleEndian.PutUint32(b[4:8], info.Width)
	binary.LittleEndian.PutUint32(b[8:12], info.Height)
	binary.LittleEndian.PutUint32(b[12:16], info.Frames)
	binary.LittleEndian.PutUint32(b[16:20], info.AudioRate)
	b[20] = info.AudioChannels
	b[21] = info.AudioBitDepth
	// b[22:24] stay zero: reserved, checked on read.
	return writeAll(w, b[:])
}

func ReadHeader(r io.Reader) (Info, error) {
	var b [24]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return Info{}, fmt.Errorf("video: header: %w", err)
	}
	if string(b[:4]) != "ARV2" {
		return Info{}, fmt.Errorf("video: invalid stream signature")
	}
	if b[22] != 0 || b[23] != 0 {
		return Info{}, fmt.Errorf("video: non-zero reserved header bytes")
	}
	i := Info{
		Width: binary.LittleEndian.Uint32(b[4:8]), Height: binary.LittleEndian.Uint32(b[8:12]), Frames: binary.LittleEndian.Uint32(b[12:16]),
		AudioRate: binary.LittleEndian.Uint32(b[16:20]), AudioChannels: b[20], AudioBitDepth: b[21],
	}
	return i, i.Validate()
}

// ReadFrame reads one video frame and, when info declares a stream-wide
// audio track (HasAudio), that same frame's own length-prefixed PCM chunk --
// possibly empty for a frame the decoder produced no audio for, never absent
// when the stream carries audio at all. A video-only stream returns a nil
// chunk and reads no audio bytes.
func ReadFrame(r io.Reader, info Info) (*image.RGBA, []byte, error) {
	if err := info.Validate(); err != nil {
		return nil, nil, err
	}
	frame := image.NewRGBA(image.Rect(0, 0, int(info.Width), int(info.Height)))
	if _, err := io.ReadFull(r, frame.Pix); err != nil {
		return nil, nil, fmt.Errorf("video: frame: %w", err)
	}
	if !info.HasAudio() {
		return frame, nil, nil
	}
	var lb [4]byte
	if _, err := io.ReadFull(r, lb[:]); err != nil {
		return nil, nil, fmt.Errorf("video: frame audio length: %w", err)
	}
	n := binary.LittleEndian.Uint32(lb[:])
	if n > MaxAudioChunk {
		return nil, nil, fmt.Errorf("video: frame audio length %d exceeds bound", n)
	}
	if n%info.audioFrameBytes() != 0 {
		return nil, nil, fmt.Errorf("video: frame audio length %d is not a whole sample frame", n)
	}
	if n == 0 {
		return frame, nil, nil
	}
	chunk := make([]byte, n)
	if _, err := io.ReadFull(r, chunk); err != nil {
		return nil, nil, fmt.Errorf("video: frame audio: %w", err)
	}
	return frame, chunk, nil
}

// WriteFrame refuses stride or bounds mismatches instead of emitting an
// ambiguous byte stream. Helper frames have exactly these fixed dimensions.
// audioChunk must be empty for a stream with no audio track (HasAudio false)
// and, whenever supplied, a whole number of interleaved sample frames.
func WriteFrame(w io.Writer, info Info, frame *image.RGBA, audioChunk []byte) error {
	if err := info.Validate(); err != nil {
		return err
	}
	if frame == nil || frame.Rect != image.Rect(0, 0, int(info.Width), int(info.Height)) || frame.Stride != int(info.Width)*4 || len(frame.Pix) != int(info.Width*info.Height)*4 {
		return fmt.Errorf("video: frame does not match stream dimensions")
	}
	if !info.HasAudio() {
		if len(audioChunk) != 0 {
			return fmt.Errorf("video: audio chunk on a video-only stream")
		}
		return writeAll(w, frame.Pix)
	}
	if len(audioChunk) > MaxAudioChunk {
		return fmt.Errorf("video: frame audio length %d exceeds bound", len(audioChunk))
	}
	if uint32(len(audioChunk))%info.audioFrameBytes() != 0 {
		return fmt.Errorf("video: frame audio length %d is not a whole sample frame", len(audioChunk))
	}
	if err := writeAll(w, frame.Pix); err != nil {
		return err
	}
	var lb [4]byte
	binary.LittleEndian.PutUint32(lb[:], uint32(len(audioChunk)))
	if err := writeAll(w, lb[:]); err != nil {
		return err
	}
	if len(audioChunk) == 0 {
		return nil
	}
	return writeAll(w, audioChunk)
}

func writeAll(w io.Writer, b []byte) error {
	n, err := w.Write(b)
	if err == nil && n != len(b) {
		err = io.ErrShortWrite
	}
	return err
}
