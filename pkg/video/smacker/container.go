// Ported from libsmacker 1.2.0 (Greg Kennedy, LGPL-2.1-or-later): container
// header, frame index and Huffman-tree-chunk parsing (smk_open_generic in
// smacker.c). See doc.go for the package-wide attribution and
// THIRD_PARTY_NOTICES.md for the notice entry.
package smacker

import (
	"encoding/binary"
	"fmt"
)

// Bounds on allocations derived from untrusted header fields. libsmacker
// itself trusts these values outright (smk_malloc exits the process on
// failure rather than refusing bad input); this port never allocates a size
// it has not first checked against either a fixed sanity bound or the number
// of bytes actually remaining in the caller-supplied buffer.
//
// maxDimension mirrors pkg/video.MaxDimension: this package has no import
// path back to pkg/video (pkg/video imports this package, so the reverse
// would cycle), so the bound is restated here rather than shared.
const (
	maxDimension     = 2048
	maxFrames        = 100000
	maxHuffTreeNodes = 1 << 20 // ~4MB per bigtree array, far above any shipped file
	maxAudioBuffer   = 64 << 20
)

// audioTrack holds one of the seven possible header-declared audio streams.
type audioTrack struct {
	Exists   bool
	Channels int // 1 or 2
	BitDepth int // 8 or 16
	Rate     int
	// compress: 0 raw PCM, 1 Smacker DPCM, 2 Bink (unsupported, matches upstream).
	compress int
}

// container holds everything parsed from the file header: dimensions, frame
// pacing, the per-frame chunk index and the four video bigtrees. It does not
// yet hold any decoded frame; that is decoder.go's job.
type container struct {
	version byte // '2' or '4'
	width   int
	height  int
	frames  int // displayed frame count, excludes the ring frame
	ring    bool

	// frameRateRaw is the raw signed header field; Decoder.IntervalUnits derives
	// the pacing unit count from it.
	frameRateRaw int32
	yDouble      bool
	yInterlace   bool

	audio [7]audioTrack

	// chunkSize/keyframe/frameType are indexed 0..frames+ring-1. chunkOffset
	// gives each chunk's byte offset into data, computed once during Open so
	// Next never has to re-scan preceding chunks.
	chunkSize   []uint32
	chunkOffset []int
	keyframe    []bool
	frameType   []byte

	tree [4]*huff16Tree // order: MMAP, MCLR, FULL, TYPE

	data []byte // the caller's full input; chunk bytes alias into it
}

// Tree slot indices, matching SMK_TREE_MMAP/MCLR/FULL/TYPE.
const (
	treeMMAP = 0
	treeMCLR = 1
	treeFULL = 2
	treeTYPE = 3
)

// openContainer ports the header half of smk_open_generic. data is the whole
// .smk file; it is not copied, so the caller must not mutate it while the
// resulting container (and any Decoder built from it) is in use.
func openContainer(data []byte) (*container, error) {
	rd := &reader{data: data}
	sig, err := rd.take(3)
	if err != nil {
		return nil, fmt.Errorf("smacker: signature: %w", err)
	}
	if string(sig) != "SMK" {
		return nil, fmt.Errorf("smacker: invalid signature %q", sig)
	}
	versionByte, err := rd.take(1)
	if err != nil {
		return nil, err
	}
	c := &container{data: data}
	c.version = versionByte[0]
	if c.version != '2' && c.version != '4' {
		if c.version < '4' {
			c.version = '2'
		} else {
			c.version = '4'
		}
	}

	width, err := rd.u32()
	if err != nil {
		return nil, err
	}
	height, err := rd.u32()
	if err != nil {
		return nil, err
	}
	frames, err := rd.u32()
	if err != nil {
		return nil, err
	}
	if width == 0 || height == 0 || width > maxDimension || height > maxDimension {
		return nil, fmt.Errorf("smacker: dimensions %dx%d outside bound", width, height)
	}
	if width%4 != 0 || height%4 != 0 {
		// Block decode below always advances by 4 columns/rows: the Smacker
		// container's own 4x4 block encoding, not a ROM1-specific property.
		// A non-aligned extent would read and write past the intended plane;
		// refuse it rather than guess. Every payload in both shipped video
		// archives (66/66, EN+RU) has block-aligned dimensions; see DIV-1258.
		return nil, fmt.Errorf("smacker: non-block-aligned extent %dx%d", width, height)
	}
	if frames == 0 || frames > maxFrames {
		return nil, fmt.Errorf("smacker: frame count %d outside bound", frames)
	}
	c.width, c.height, c.frames = int(width), int(height), int(frames)

	rateWord, err := rd.u32()
	if err != nil {
		return nil, err
	}
	c.frameRateRaw = int32(rateWord)

	flags, err := rd.u32()
	if err != nil {
		return nil, err
	}
	if flags&0x01 != 0 {
		c.ring = true
	}
	if flags&0x02 != 0 {
		c.yDouble = true
	}
	if flags&0x04 != 0 {
		c.yInterlace = true
	}

	// Per-track max buffer sizes: informational only in this port (Go slices
	// grow as needed rather than preallocating to a declared maximum), read
	// here purely to keep the header cursor aligned with the file layout.
	for i := 0; i < 7; i++ {
		if _, err := rd.u32(); err != nil {
			return nil, err
		}
	}

	treeSize, err := rd.u32()
	if err != nil {
		return nil, err
	}

	var videoTreeSize [4]uint32
	for i := range videoTreeSize {
		v, err := rd.u32()
		if err != nil {
			return nil, err
		}
		videoTreeSize[i] = v
	}

	for i := 0; i < 7; i++ {
		w, err := rd.u32()
		if err != nil {
			return nil, err
		}
		if w&0x40000000 == 0 {
			continue
		}
		t := audioTrack{Exists: true}
		if w&0x80000000 != 0 {
			t.compress = 1
		}
		if w&0x20000000 != 0 {
			t.BitDepth = 16
		} else {
			t.BitDepth = 8
		}
		if w&0x10000000 != 0 {
			t.Channels = 2
		} else {
			t.Channels = 1
		}
		if w&0x0c000000 != 0 {
			t.compress = 2 // Bink (perceptual): unsupported, matches upstream
		}
		t.Rate = int(w & 0x00FFFFFF)
		c.audio[i] = t
	}

	// Dummy field: reserved, always skipped.
	if _, err := rd.u32(); err != nil {
		return nil, err
	}

	total := c.frames
	if c.ring {
		total++
	}
	c.chunkSize = make([]uint32, total)
	c.keyframe = make([]bool, total)
	for i := 0; i < total; i++ {
		v, err := rd.u32()
		if err != nil {
			return nil, fmt.Errorf("smacker: chunk size %d: %w", i, err)
		}
		if v&0x01 != 0 {
			c.keyframe[i] = true
		}
		c.chunkSize[i] = v &^ 3
	}

	c.frameType = make([]byte, total)
	for i := 0; i < total; i++ {
		b, err := rd.take(1)
		if err != nil {
			return nil, fmt.Errorf("smacker: frame type %d: %w", i, err)
		}
		c.frameType[i] = b[0]
	}

	if int(treeSize) < 0 || treeSize > uint32(len(data)-rd.pos) {
		return nil, fmt.Errorf("smacker: hufftree chunk size %d exceeds remaining input", treeSize)
	}
	treeChunk, err := rd.take(int(treeSize))
	if err != nil {
		return nil, err
	}
	bits := newBitReader(treeChunk)
	for i := 0; i < 4; i++ {
		t, err := buildHuff16(bits, videoTreeSize[i])
		if err != nil {
			return nil, fmt.Errorf("smacker: video tree %d: %w", i, err)
		}
		c.tree[i] = t
	}

	// Pre-compute each chunk's offset into data so Next never rescans
	// preceding chunks. Sizes are validated against what actually remains.
	c.chunkOffset = make([]int, total)
	for i := 0; i < total; i++ {
		c.chunkOffset[i] = rd.pos
		if c.chunkSize[i] > uint32(len(data)-rd.pos) {
			return nil, fmt.Errorf("smacker: chunk %d size %d exceeds remaining input", i, c.chunkSize[i])
		}
		rd.pos += int(c.chunkSize[i])
	}

	return c, nil
}

// reader is a small cursor over the raw file bytes used only during header
// parsing; per-frame chunk bytes are addressed directly through
// container.chunkOffset/chunkSize instead.
type reader struct {
	data []byte
	pos  int
}

func (r *reader) take(n int) ([]byte, error) {
	if n < 0 || n > len(r.data)-r.pos {
		return nil, fmt.Errorf("smacker: unexpected end of header at offset %d, want %d byte(s)", r.pos, n)
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b, nil
}

func (r *reader) u32() (uint32, error) {
	b, err := r.take(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}
