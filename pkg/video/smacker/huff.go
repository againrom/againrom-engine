// Ported from libsmacker 1.2.0 (Greg Kennedy, LGPL-2.1-or-later): the two
// Huffman tree shapes (smk_huff8/smk_huff16 build and lookup in smacker.c).
// See doc.go for the package-wide attribution and THIRD_PARTY_NOTICES.md for
// the notice entry.
package smacker

import "fmt"

// huff8Branch/huff8LeafMask port SMK_HUFF8_BRANCH/SMK_HUFF8_LEAF_MASK: a
// tree-in-array encoding where a branch node stores the index of its right
// child (the left child is always the very next array slot) with the high
// bit set to distinguish it from a leaf value.
const (
	huff8Branch    = 0x8000
	huff8LeafMask  = 0x7FFF
	huff8TreeLimit = 511 // N leaves + N-1 branches, N=256 for an 8-bit alphabet
)

// huff8Tree ports struct smk_huff8_t.
type huff8Tree struct {
	tree [huff8TreeLimit]uint16
	size int
}

// buildHuff8 ports smk_huff8_build. Smacker huff trees begin with a set bit
// (tree present) and end with an unset bit; very small or audio-only files
// may carry no tree at all.
func buildHuff8(r *bitReader) (*huff8Tree, error) {
	bit, err := r.readBit()
	if err != nil {
		return nil, fmt.Errorf("smacker: huff8: initial bit: %w", err)
	}
	t := &huff8Tree{}
	if bit != 0 {
		if err := t.buildRec(r); err != nil {
			return nil, err
		}
	}
	bit, err = r.readBit()
	if err != nil {
		return nil, fmt.Errorf("smacker: huff8: final bit: %w", err)
	}
	if bit != 0 {
		return nil, fmt.Errorf("smacker: huff8: final bit set, tree malformed")
	}
	return t, nil
}

// buildRec ports _smk_huff8_build_rec.
func (t *huff8Tree) buildRec(r *bitReader) error {
	if t.size >= huff8TreeLimit {
		return fmt.Errorf("smacker: huff8: tree exceeds %d nodes", huff8TreeLimit)
	}
	bit, err := r.readBit()
	if err != nil {
		return fmt.Errorf("smacker: huff8: branch bit: %w", err)
	}
	if bit != 0 {
		index := t.size
		t.size++
		if err := t.buildRec(r); err != nil {
			return err
		}
		t.tree[index] = huff8Branch | uint16(t.size)
		return t.buildRec(r)
	}
	v, err := r.readByte()
	if err != nil {
		return fmt.Errorf("smacker: huff8: leaf byte: %w", err)
	}
	t.tree[t.size] = uint16(v)
	t.size++
	return nil
}

// lookup ports smk_huff8_lookup.
func (t *huff8Tree) lookup(r *bitReader) (int, error) {
	index := 0
	for t.tree[index]&huff8Branch != 0 {
		bit, err := r.readBit()
		if err != nil {
			return 0, fmt.Errorf("smacker: huff8: lookup bit: %w", err)
		}
		if bit != 0 {
			index = int(t.tree[index] & huff8LeafMask)
			if index >= huff8TreeLimit {
				return 0, fmt.Errorf("smacker: huff8: branch target out of range")
			}
		} else {
			index++
		}
	}
	return int(t.tree[index]), nil
}

// huff16Branch/huff16Cache/huff16LeafMask port SMK_HUFF16_BRANCH,
// SMK_HUFF16_CACHE and SMK_HUFF16_LEAF_MASK: the same tree-in-array shape as
// huff8Tree, but a leaf may instead name one of the last three distinct
// decoded values (a most-recently-used cache), which the video and audio
// bigtrees rely on for repeated colour pairs and motion vectors.
const (
	huff16Branch   = 0x80000000
	huff16Cache    = 0x40000000
	huff16LeafMask = 0x3FFFFFFF
)

// huff16Tree ports struct smk_huff16_t.
type huff16Tree struct {
	tree  []uint32
	size  int
	cache [3]uint16
}

// buildHuff16 ports smk_huff16_build. allocSize is the "unpacked size" field
// from the container header for this tree; it fixes the number of leaves the
// tree must resolve to before the caller can trust the array is complete.
func buildHuff16(r *bitReader, allocSize uint32) (*huff16Tree, error) {
	bit, err := r.readBit()
	if err != nil {
		return nil, fmt.Errorf("smacker: huff16: initial bit: %w", err)
	}
	t := &huff16Tree{}
	if bit != 0 {
		low8, err := buildHuff8(r)
		if err != nil {
			return nil, fmt.Errorf("smacker: huff16: low tree: %w", err)
		}
		hi8, err := buildHuff8(r)
		if err != nil {
			return nil, fmt.Errorf("smacker: huff16: high tree: %w", err)
		}
		for i := 0; i < 3; i++ {
			lo, err := r.readByte()
			if err != nil {
				return nil, fmt.Errorf("smacker: huff16: cache %d low byte: %w", i, err)
			}
			hi, err := r.readByte()
			if err != nil {
				return nil, fmt.Errorf("smacker: huff16: cache %d high byte: %w", i, err)
			}
			t.cache[i] = uint16(lo) | uint16(hi)<<8
		}
		if allocSize < 12 || allocSize%4 != 0 {
			return nil, fmt.Errorf("smacker: huff16: illegal alloc size %d", allocSize)
		}
		limit := int((allocSize - 12) / 4)
		if limit > maxHuffTreeNodes {
			return nil, fmt.Errorf("smacker: huff16: tree of %d nodes exceeds bound", limit)
		}
		t.tree = make([]uint32, limit)
		if err := t.buildRec(r, low8, hi8, limit); err != nil {
			return nil, err
		}
		if t.size != limit {
			return nil, fmt.Errorf("smacker: huff16: incomplete tree, got %d want %d", t.size, limit)
		}
	} else {
		t.tree = []uint32{0}
	}
	bit, err = r.readBit()
	if err != nil {
		return nil, fmt.Errorf("smacker: huff16: final bit: %w", err)
	}
	if bit != 0 {
		return nil, fmt.Errorf("smacker: huff16: final bit set, tree malformed")
	}
	return t, nil
}

// buildRec ports _smk_huff16_build_rec, including the escape-code detection
// that lets a leaf point back into the recently-used cache instead of
// carrying its own low/high byte pair.
func (t *huff16Tree) buildRec(r *bitReader, low8, hi8 *huff8Tree, limit int) error {
	if t.size >= limit {
		return fmt.Errorf("smacker: huff16: tree exceeds allocation of %d nodes", limit)
	}
	bit, err := r.readBit()
	if err != nil {
		return fmt.Errorf("smacker: huff16: branch bit: %w", err)
	}
	if bit != 0 {
		index := t.size
		t.size++
		if err := t.buildRec(r, low8, hi8, limit); err != nil {
			return err
		}
		t.tree[index] = huff16Branch | uint32(t.size)
		return t.buildRec(r, low8, hi8, limit)
	}
	lo, err := low8.lookup(r)
	if err != nil {
		return fmt.Errorf("smacker: huff16: leaf low value: %w", err)
	}
	hi, err := hi8.lookup(r)
	if err != nil {
		return fmt.Errorf("smacker: huff16: leaf high value: %w", err)
	}
	v := uint32(lo) | uint32(hi)<<8
	switch v {
	case uint32(t.cache[0]):
		v = huff16Cache
	case uint32(t.cache[1]):
		v = huff16Cache | 1
	case uint32(t.cache[2]):
		v = huff16Cache | 2
	}
	t.tree[t.size] = v
	t.size++
	return nil
}

// lookup ports smk_huff16_lookup, including its cache-maintenance side
// effect: every decoded value (after cache substitution) becomes the new
// most-recently-used entry.
func (t *huff16Tree) lookup(r *bitReader) (int, error) {
	index := 0
	for t.tree[index]&huff16Branch != 0 {
		bit, err := r.readBit()
		if err != nil {
			return 0, fmt.Errorf("smacker: huff16: lookup bit: %w", err)
		}
		if bit != 0 {
			index = int(t.tree[index] & huff16LeafMask)
			if index >= len(t.tree) {
				return 0, fmt.Errorf("smacker: huff16: branch target out of range")
			}
		} else {
			index++
		}
	}
	value := t.tree[index]
	if value&huff16Cache != 0 {
		value = uint32(t.cache[value&huff16LeafMask])
	}
	if t.cache[0] != uint16(value) {
		t.cache[2] = t.cache[1]
		t.cache[1] = t.cache[0]
		t.cache[0] = uint16(value)
	}
	return int(value), nil
}
