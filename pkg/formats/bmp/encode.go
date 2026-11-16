package bmp

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"

	"againrom/pkg/formats/pal"
)

// EncodePaletted is the one bitmap writer: img as an uncompressed 8-bit
// Windows bitmap, the shape DecodePaletted reads. The colour table holds
// img's own entries and the rows are stored bottom-up, each padded to four
// bytes. A pixel naming no entry of the table is refused.
func EncodePaletted(img *image.Paletted) ([]byte, error) {
	r := img.Bounds()
	w, h, n := r.Dx(), r.Dy(), len(img.Palette)
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("bmp: %dx%d picture", w, h)
	}
	if n < 1 || n > maxPalette {
		return nil, fmt.Errorf("bmp: %d palette entries, want 1 to %d", n, maxPalette)
	}
	stride := (w + 3) &^ 3
	off := HeaderLen + n*pal.EntrySize
	out := make([]byte, off+stride*h)
	le := binary.LittleEndian
	out[0], out[1] = 'B', 'M'
	le.PutUint32(out[2:], uint32(len(out)))
	le.PutUint32(out[10:], uint32(off))
	le.PutUint32(out[14:], infoHeaderLen)
	le.PutUint32(out[18:], uint32(w))
	le.PutUint32(out[22:], uint32(h))
	le.PutUint16(out[26:], 1)
	le.PutUint16(out[28:], IndexedBitsPerPixel)
	le.PutUint32(out[34:], uint32(stride*h))
	le.PutUint32(out[46:], uint32(n))
	for i, c := range img.Palette {
		v := color.NRGBAModel.Convert(c).(color.NRGBA)
		copy(out[HeaderLen+i*pal.EntrySize:], []byte{v.B, v.G, v.R, 0})
	}
	for y := range h {
		row := out[off+(h-1-y)*stride:]
		for x := range w {
			k := img.Pix[y*img.Stride+x]
			if int(k) >= n {
				return nil, fmt.Errorf("bmp: pixel (%d,%d) index %d outside the %d-entry palette", x, y, k, n)
			}
			row[x] = k
		}
	}
	return out, nil
}
