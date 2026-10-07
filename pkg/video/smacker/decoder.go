// Ported from libsmacker 1.2.0 (Greg Kennedy, LGPL-2.1-or-later): the
// top-level open/enable/advance API (smk_open_generic, smk_enable_all,
// smk_next in smacker.c and smacker.h), restated with idiomatic Go error
// returns in place of the original's stderr diagnostics and boolean codes.
// See doc.go for the package-wide attribution and THIRD_PARTY_NOTICES.md for
// the notice entry.
package smacker

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

// Frame is one decoded unit: an update to the persistent palette-indexed
// video plane, plus whichever enabled audio tracks carried data this frame.
// Indices aliases decoder-owned memory and is valid only until the next call
// to Next; copy it if the caller needs to retain it past that point.
type Frame struct {
	Palette  [256][3]byte
	Indices  []byte // width*height, top-down, valid only when video is enabled
	Audio    [7][]byte
	Keyframe bool
	// NewPalette is set when this frame carried a palette record.
	NewPalette bool
}

// Decoder decodes one Smacker container's frames in order. Like the
// original library, video and each audio track are decoded only once
// explicitly enabled (EnableVideo/EnableAudio); everything is disabled at
// Open, matching smk_open_generic's zero-initialized enable switches. It is
// not safe for concurrent use.
type Decoder struct {
	c       *container
	palette [256][3]byte
	plane   []byte

	videoOn bool
	audioOn [7]bool

	next int // index of the next frame Next will decode
}

// Open parses a Smacker container's header, frame index and video Huffman
// trees. data is the whole file; Open does not copy it, and frames returned
// by Next alias into it, so the caller must not mutate data while the
// Decoder is in use.
func Open(data []byte) (*Decoder, error) {
	c, err := openContainer(data)
	if err != nil {
		return nil, err
	}
	return &Decoder{c: c, plane: make([]byte, c.width*c.height)}, nil
}

func (d *Decoder) Width() int  { return d.c.width }
func (d *Decoder) Height() int { return d.c.height }

// Frames reports the displayed frame count. It excludes the container's
// optional ring/loop frame, which this decoder parses but never delivers:
// every playback here is a single pass, never a loop.
func (d *Decoder) Frames() int { return d.c.frames }

// EnableVideo turns video block decode on or off. Off by default.
func (d *Decoder) EnableVideo(on bool) { d.videoOn = on }

// EnableAudio turns decode of one audio track (0..6) on or off. Off by
// default for every track.
func (d *Decoder) EnableAudio(track int, on bool) {
	if track >= 0 && track < 7 {
		d.audioOn[track] = on
	}
}

// AudioTrack reports the header-declared shape of audio track i (0..6).
// exists is false when the track is absent from the whole file.
func (d *Decoder) AudioTrack(i int) (channels, bitDepth, rate int, exists bool) {
	if i < 0 || i > 6 {
		return 0, 0, 0, false
	}
	t := d.c.audio[i]
	return t.Channels, t.BitDepth, t.Rate, t.Exists
}

// IntervalUnits reports the header's per-frame interval in units of 10
// microseconds, the decoder's own pacing unit: a non-negative header value is
// milliseconds per frame, so the unit count is that value times 100 modulo
// 2^32, and a negative one is its own magnitude in units. Zero stays zero; the
// installed decoder applies no default, floor or ceiling (VIDEO-073).
func (d *Decoder) IntervalUnits() uint32 { return intervalUnits(d.c.frameRateRaw) }

func intervalUnits(raw int32) uint32 {
	if raw < 0 {
		return uint32(-raw)
	}
	return uint32(raw) * 100
}

// Interval is IntervalUnits as a duration.
func (d *Decoder) Interval() time.Duration {
	return time.Duration(d.IntervalUnits()) * 10 * time.Microsecond
}

// AudioDecodable reports whether track i exists and uses a coding this
// decoder implements (raw or Smacker DPCM). A track that selects the Bink
// coding is not decodable and must not be enabled.
func (d *Decoder) AudioDecodable(i int) bool {
	if i < 0 || i > 6 {
		return false
	}
	t := d.c.audio[i]
	return t.Exists && t.compress != 2
}

// Next decodes the next frame in sequence, starting with frame 0 on the
// first call, and reports io.EOF once every displayed frame (Frames) has
// been returned. Only what EnableVideo/EnableAudio has turned on is
// actually decoded; a disabled component's bytes are still walked (so the
// frame's other components decode correctly) but never rendered.
func (d *Decoder) Next() (*Frame, error) {
	if d.next >= d.c.frames {
		return nil, io.EOF
	}
	i := d.next
	d.next++

	chunk := d.c.data[d.c.chunkOffset[i] : d.c.chunkOffset[i]+int(d.c.chunkSize[i])]
	frameType := d.c.frameType[i]
	p := chunk

	if frameType&0x01 != 0 {
		if len(p) < 1 {
			return nil, fmt.Errorf("smacker: frame %d: truncated palette record", i)
		}
		size := 4 * int(p[0])
		if size < 1 || size > len(p) {
			return nil, fmt.Errorf("smacker: frame %d: palette record size %d out of range", i, size)
		}
		if d.videoOn {
			if err := renderPalette(&d.palette, p[1:size]); err != nil {
				return nil, fmt.Errorf("smacker: frame %d: %w", i, err)
			}
		}
		p = p[size:]
	}

	var f Frame
	f.NewPalette = frameType&0x01 != 0
	for track := 0; track < 7; track++ {
		if frameType&(0x02<<uint(track)) == 0 {
			continue
		}
		if len(p) < 4 {
			return nil, fmt.Errorf("smacker: frame %d: truncated audio[%d] record", i, track)
		}
		size := int(binary.LittleEndian.Uint32(p))
		if size < 4 || size > len(p) {
			return nil, fmt.Errorf("smacker: frame %d: audio[%d] record size %d out of range", i, track, size)
		}
		if d.audioOn[track] {
			decoded, err := decodeAudioChunk(d.c.audio[track], p[4:size])
			if err != nil {
				return nil, fmt.Errorf("smacker: frame %d: audio[%d]: %w", i, track, err)
			}
			f.Audio[track] = decoded
		}
		p = p[size:]
	}

	if d.videoOn {
		if err := renderVideo(d.plane, d.c.width, d.c.height, d.c.version, d.c.tree, p); err != nil {
			return nil, fmt.Errorf("smacker: frame %d: %w", i, err)
		}
		f.Indices = d.plane
	}

	f.Palette = d.palette
	f.Keyframe = d.c.keyframe[i]
	return &f, nil
}
