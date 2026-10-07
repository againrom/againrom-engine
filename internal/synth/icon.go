package synth

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
)

// ---------------------------------------------------------------------------
// Windows icon resources
// ---------------------------------------------------------------------------

// IconEntry is one image of a synthetic icon group: the bytes of its icon
// resource and the size and depth its group entry declares.
type IconEntry struct {
	ID            uint16 // the icon resource's numeric name
	Width, Height int    // 256 is written as the byte 0
	Depth         int
	Data          []byte
}

// IconGroup encodes the group directory of the entries, in order.
func IconGroup(entries []IconEntry) []byte {
	out := make([]byte, 6+14*len(entries))
	binary.LittleEndian.PutUint16(out[2:], 1)
	binary.LittleEndian.PutUint16(out[4:], uint16(len(entries)))
	for i, e := range entries {
		at := 6 + 14*i
		out[at], out[at+1] = byte(e.Width%256), byte(e.Height%256)
		if e.Depth < 8 {
			out[at+2] = byte(1 << e.Depth)
		}
		binary.LittleEndian.PutUint16(out[at+4:], 1)
		binary.LittleEndian.PutUint16(out[at+6:], uint16(e.Depth))
		binary.LittleEndian.PutUint32(out[at+8:], uint32(len(e.Data)))
		binary.LittleEndian.PutUint16(out[at+12:], e.ID)
	}
	return out
}

// IconResources returns the icon resources of the entries, all in one language.
func IconResources(entries []IconEntry, language uint16) []PEResource {
	out := make([]PEResource, len(entries))
	for i, e := range entries {
		out[i] = PEResource{Type: 3, ID: e.ID, Language: language, Data: e.Data}
	}
	return out
}

// IconPNG encodes img as an icon image stored as a whole PNG file.
func IconPNG(img *image.NRGBA) []byte {
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		panic("synth: " + err.Error())
	}
	return out.Bytes()
}

// IconDIB encodes img as one bitmap image of an icon resource at depth 1, 4, 8,
// 24 or 32 bits per pixel: a BITMAPINFOHEADER of twice the image's height, a
// colour table of 1<<depth entries below 24 bits, the colour bitmap and the AND
// mask, both bottom-up with rows padded to four bytes.
//
// A pixel whose alpha is 0 is a masked pixel. Below 32 bits its colour is not
// black: it is the highest table index, which an unused slot fills with
// 0x33 0x22 0x11, or that colour itself at 24 bits, so a reader that used the
// colour under the mask would show it. At 32 bits the mask is left clear and
// the alpha byte carries the transparency, as an icon with an alpha channel
// does. The padding bits that end each mask row are set. Below 32 bits every
// alpha must be 0 or 255; the table takes the distinct opaque colours in the
// order the image first shows them.
func IconDIB(img *image.NRGBA, depth int) []byte {
	return iconDIB(img, depth, depth == 32)
}

// IconDIBMasked32 encodes img at 32 bits per pixel with the fourth byte of every
// pixel zero and the transparency carried by the AND mask alone, the shape an
// icon written before alpha channels existed has at that depth.
func IconDIBMasked32(img *image.NRGBA) []byte { return iconDIB(img, 32, false) }

func iconDIB(img *image.NRGBA, depth int, alphaChannel bool) []byte {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	index := map[color.NRGBA]int{}
	var table []color.NRGBA
	if depth < 24 {
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				c := img.NRGBAAt(x, y)
				if c.A == 0 {
					continue
				}
				if c.A != 255 {
					panic("synth: partial alpha needs 32 bits per pixel")
				}
				if _, seen := index[c]; !seen {
					index[c] = len(table)
					table = append(table, c)
				}
			}
		}
		if len(table) > 1<<depth {
			panic("synth: more colours than the depth holds")
		}
	}
	xorStride := (w*depth + 31) / 32 * 4
	andStride := (w + 31) / 32 * 4
	tableLen := 0
	if depth < 24 {
		tableLen = 4 << depth
	}
	out := make([]byte, 40+tableLen+h*(xorStride+andStride))
	le := binary.LittleEndian
	le.PutUint32(out[0:], 40)
	le.PutUint32(out[4:], uint32(w))
	le.PutUint32(out[8:], uint32(2*h))
	le.PutUint16(out[12:], 1)
	le.PutUint16(out[14:], uint16(depth))
	le.PutUint32(out[20:], uint32(h*xorStride))
	for i, c := range table {
		copy(out[40+4*i:], []byte{c.B, c.G, c.R, 0})
	}
	if depth < 24 && len(table) < 1<<depth {
		copy(out[40+4*(1<<depth-1):], []byte{0x11, 0x22, 0x33, 0})
	}
	xorAt := 40 + tableLen
	andAt := xorAt + h*xorStride
	for y := 0; y < h; y++ {
		file := h - 1 - y
		xr := out[xorAt+file*xorStride:]
		ar := out[andAt+file*andStride:]
		for x := 0; x < andStride*8; x++ {
			if x >= w {
				ar[x/8] |= 0x80 >> uint(x%8)
			}
		}
		for x := 0; x < w; x++ {
			c := img.NRGBAAt(b.Min.X+x, b.Min.Y+y)
			masked := c.A == 0
			if masked && !alphaChannel {
				ar[x/8] |= 0x80 >> uint(x%8)
			}
			switch {
			case depth < 24:
				i := 1<<depth - 1
				if !masked {
					i = index[c]
				}
				switch depth {
				case 1:
					xr[x/8] |= byte(i) << (7 - uint(x%8))
				case 4:
					xr[x/2] |= byte(i) << (4 * (1 - uint(x%2)))
				default:
					xr[x] = byte(i)
				}
			case depth == 24:
				if masked {
					c = color.NRGBA{R: 0x33, G: 0x22, B: 0x11}
				}
				copy(xr[3*x:], []byte{c.B, c.G, c.R})
			default:
				if masked && !alphaChannel {
					c = color.NRGBA{R: 0x33, G: 0x22, B: 0x11}
				}
				a := c.A
				if !alphaChannel {
					a = 0
				}
				copy(xr[4*x:], []byte{c.B, c.G, c.R, a})
			}
		}
	}
	return out
}
