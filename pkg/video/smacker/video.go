// Ported from libsmacker 1.2.0 (Greg Kennedy, LGPL-2.1-or-later): palette
// rendering and per-block video decode (smk_render_palette / smk_render_video
// in smacker.c). See doc.go for the package-wide attribution and
// THIRD_PARTY_NOTICES.md for the notice entry.
package smacker

import "fmt"

// palette6to8 ports the palmap table in smk_render_palette: Smacker palette
// entries are 6-bit; this expands each to the 8-bit range an RGBA frame
// needs. It is the original library's own table, not a choice made here.
var palette6to8 = [64]byte{
	0x00, 0x04, 0x08, 0x0C, 0x10, 0x14, 0x18, 0x1C,
	0x20, 0x24, 0x28, 0x2C, 0x30, 0x34, 0x38, 0x3C,
	0x41, 0x45, 0x49, 0x4D, 0x51, 0x55, 0x59, 0x5D,
	0x61, 0x65, 0x69, 0x6D, 0x71, 0x75, 0x79, 0x7D,
	0x82, 0x86, 0x8A, 0x8E, 0x92, 0x96, 0x9A, 0x9E,
	0xA2, 0xA6, 0xAA, 0xAE, 0xB2, 0xB6, 0xBA, 0xBE,
	0xC3, 0xC7, 0xCB, 0xCF, 0xD3, 0xD7, 0xDB, 0xDF,
	0xE3, 0xE7, 0xEB, 0xEF, 0xF3, 0xF7, 0xFB, 0xFF,
}

// renderPalette ports smk_render_palette. p is the palette record payload
// (after the leading byte-count byte already consumed by the caller);
// palette is mutated in place, one 256x3 RGB table.
func renderPalette(palette *[256][3]byte, p []byte) error {
	var old [256][3]byte
	old = *palette
	i := 0
	for i < 256 && len(p) > 0 {
		switch {
		case p[0]&0x80 != 0:
			// Skip block: preserve count entries from the previous palette.
			count := int(p[0]&0x7F) + 1
			p = p[1:]
			if i+count > 256 {
				return fmt.Errorf("smacker: palette skip overflow at %d+%d", i, count)
			}
			i += count
		case p[0]&0x40 != 0:
			// Colour-shift block: copy count entries of the previous palette
			// starting at src into the next entries of the new palette.
			if len(p) < 2 {
				return fmt.Errorf("smacker: palette shift: truncated")
			}
			count := int(p[0]&0x3F) + 1
			src := int(p[1])
			p = p[2:]
			if i+count > 256 || src+count > 256 || (src < i && src+count > i) {
				return fmt.Errorf("smacker: palette shift overflow count=%d src=%d dst=%d", count, src, i)
			}
			for k := 0; k < count; k++ {
				palette[i+k] = old[src+k]
			}
			i += count
		default:
			// Direct colour: three 6-bit indices, in R,G,B order.
			if len(p) < 3 {
				return fmt.Errorf("smacker: palette direct: truncated")
			}
			for c := 0; c < 3; c++ {
				if p[c] > 0x3F {
					return fmt.Errorf("smacker: palette index %#x exceeds 0x3F", p[c])
				}
				palette[i][c] = palette6to8[p[c]]
			}
			p = p[3:]
			i++
		}
	}
	if i < 256 {
		return fmt.Errorf("smacker: palette record filled only %d/256 entries", i)
	}
	return nil
}

// sizetable ports the block-run-length table used by the TYPE tree's low six
// bits: most codes are literal run counts 1..59, the last five widen to
// large power-of-two runs.
var sizetable = [64]uint16{
	1, 2, 3, 4, 5, 6, 7, 8,
	9, 10, 11, 12, 13, 14, 15, 16,
	17, 18, 19, 20, 21, 22, 23, 24,
	25, 26, 27, 28, 29, 30, 31, 32,
	33, 34, 35, 36, 37, 38, 39, 40,
	41, 42, 43, 44, 45, 46, 47, 48,
	49, 50, 51, 52, 53, 54, 55, 56,
	57, 58, 59, 128, 256, 512, 1024, 2048,
}

// renderVideo ports smk_render_video. frame is the persistent width*height
// index plane (VOID blocks intentionally leave bytes untouched, carrying the
// previous frame's pixels forward — this is the codec's own inter-frame
// delta, not an omission). trees is the container's four bigtrees in
// MMAP/MCLR/FULL/TYPE order.
func renderVideo(frame []byte, width, height int, version byte, trees [4]*huff16Tree, p []byte) error {
	bs := newBitReader(p)
	for i := range trees {
		trees[i].cache = [3]uint16{}
	}
	row, col := 0, 0
	for row < height {
		unpack, err := trees[treeTYPE].lookup(bs)
		if err != nil {
			return fmt.Errorf("smacker: video: TYPE lookup: %w", err)
		}
		typ := unpack & 0x0003
		blocklen := (unpack & 0x00FC) >> 2
		typedata := byte((unpack & 0xFF00) >> 8)

		if typ == 1 && version == '4' {
			bit, err := bs.readBit()
			if err != nil {
				return fmt.Errorf("smacker: video: v4 type bit: %w", err)
			}
			if bit != 0 {
				typ = 4
			} else {
				bit, err := bs.readBit()
				if err != nil {
					return fmt.Errorf("smacker: video: v4 type bit 2: %w", err)
				}
				if bit != 0 {
					typ = 5
				}
			}
		}

		run := int(sizetable[blocklen])
		for j := 0; j < run && row < height; j++ {
			skip := row*width + col
			switch typ {
			case 0: // MCLR: two colours plus a 16-pixel selector bitmap
				mclr, err := trees[treeMCLR].lookup(bs)
				if err != nil {
					return fmt.Errorf("smacker: video: MCLR lookup: %w", err)
				}
				s1, s2 := byte(mclr>>8), byte(mclr)
				mmap, err := trees[treeMMAP].lookup(bs)
				if err != nil {
					return fmt.Errorf("smacker: video: MMAP lookup: %w", err)
				}
				bitpos := 1
				for k := 0; k < 4; k++ {
					for c := 0; c < 4; c++ {
						if mmap&bitpos != 0 {
							frame[skip+c] = s1
						} else {
							frame[skip+c] = s2
						}
						bitpos <<= 1
					}
					skip += width
				}
			case 1: // FULL: every pixel explicit, two per FULL-tree value
				for k := 0; k < 4; k++ {
					v, err := trees[treeFULL].lookup(bs)
					if err != nil {
						return fmt.Errorf("smacker: video: FULL lookup: %w", err)
					}
					frame[skip+3], frame[skip+2] = byte(v>>8), byte(v)
					v, err = trees[treeFULL].lookup(bs)
					if err != nil {
						return fmt.Errorf("smacker: video: FULL lookup: %w", err)
					}
					frame[skip+1], frame[skip] = byte(v>>8), byte(v)
					skip += width
				}
			case 2: // VOID: no change, previous frame's pixels stand
			case 3: // SOLID: one value fills all 16 pixels
				for k := 0; k < 4; k++ {
					frame[skip], frame[skip+1], frame[skip+2], frame[skip+3] = typedata, typedata, typedata, typedata
					skip += width
				}
			case 4: // v4 DOUBLE: one 2x2 colour quad, doubled to 4x4
				for k := 0; k < 2; k++ {
					v, err := trees[treeFULL].lookup(bs)
					if err != nil {
						return fmt.Errorf("smacker: video: v4 double FULL lookup: %w", err)
					}
					hi, lo := byte(v>>8), byte(v)
					for i := 0; i < 2; i++ {
						frame[skip+2], frame[skip+3] = hi, hi
						frame[skip], frame[skip+1] = lo, lo
						skip += width
					}
				}
			case 5: // v4 HALF: full horizontal detail, halved vertically
				for k := 0; k < 2; k++ {
					v, err := trees[treeFULL].lookup(bs)
					if err != nil {
						return fmt.Errorf("smacker: video: v4 half FULL lookup a: %w", err)
					}
					hi, lo := byte(v>>8), byte(v)
					frame[skip+3], frame[skip+2] = hi, lo
					frame[skip+width+3], frame[skip+width+2] = hi, lo
					v, err = trees[treeFULL].lookup(bs)
					if err != nil {
						return fmt.Errorf("smacker: video: v4 half FULL lookup b: %w", err)
					}
					hi, lo = byte(v>>8), byte(v)
					frame[skip+1], frame[skip] = hi, lo
					frame[skip+width+1], frame[skip+width] = hi, lo
					skip += 2 * width
				}
			}
			col += 4
			if col >= width {
				col = 0
				row += 4
			}
		}
	}
	return nil
}
