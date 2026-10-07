// Package backdrop remaps packed dialogue background pixels.
package backdrop

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
)

type Layout uint8

const (
	RGB565 Layout = iota
	RGB555
)

type Mode uint8

const (
	Full Mode = iota
	Reduced
)

type Lookup struct {
	Layout Layout
	Mode   Mode
	Level  uint8
	words  []uint16
}

func New(layout Layout, mode Mode) (*Lookup, error) {
	return NewLevel(layout, mode, 3)
}

func NewLevel(layout Layout, mode Mode, level uint8) (*Lookup, error) {
	if layout > RGB555 || mode > Reduced || level > 16 {
		return nil, fmt.Errorf("unsupported backdrop layout/mode %d/%d", layout, mode)
	}
	n := 65536
	if mode == Reduced {
		n >>= 3
	}
	l := &Lookup{Layout: layout, Mode: mode, Level: level, words: make([]uint16, n)}
	keep := 16 - int(level)
	greenBits := 6
	if layout == RGB555 {
		greenBits = 5
	}
	for i := range l.words {
		p := i
		if mode == Reduced {
			p = i*8 + 4
		}
		b := (p & 31) * keep / 16
		g := ((p >> 5) & ((1 << greenBits) - 1)) * keep / 16
		r := ((p >> (5 + greenBits)) & 31) * keep / 16
		l.words[i] = uint16(r<<(5+greenBits) | g<<5 | b)
	}
	return l, nil
}

func (l *Lookup) Word(p uint16) uint16 {
	if l.Layout == RGB555 {
		p &= 0x7fff
	}
	if l.Mode == Reduced {
		return l.words[p>>3]
	}
	return l.words[p]
}

// Remap intersects half-open rectangles and leaves pitch padding untouched.
func (l *Lookup) Remap(pix []byte, width, height, pitch int, request, clip image.Rectangle) error {
	if width < 0 || height < 0 || pitch < 0 || width > pitch/2 || height > len(pix)/max(1, pitch) {
		return fmt.Errorf("invalid packed backdrop surface")
	}
	r := request.Intersect(clip).Intersect(image.Rect(0, 0, width, height))
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			i := y*pitch + 2*x
			binary.LittleEndian.PutUint16(pix[i:i+2], l.Word(binary.LittleEndian.Uint16(pix[i:i+2])))
		}
	}
	return nil
}

// Color truncates RGBA channels into packed bits and expands by floor(255*c/max).
func (l *Lookup) Color(c color.RGBA) color.RGBA {
	g, shift, mask := int(c.G)>>2, 11, 63
	if l.Layout == RGB555 {
		g, shift, mask = int(c.G)>>3, 10, 31
	}
	p := l.Word(uint16((int(c.R)>>3)<<shift | g<<5 | int(c.B)>>3))
	return color.RGBA{uint8(int(p>>shift) * 255 / 31), uint8(int(p>>5&uint16(mask)) * 255 / mask), uint8(int(p&31) * 255 / 31), c.A}
}

func (l *Lookup) Apply(dst *image.RGBA, request, clip image.Rectangle, shows uint64) {
	if dst == nil {
		return
	}
	r := request.Intersect(clip).Intersect(dst.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			c := dst.RGBAAt(x, y)
			for i := uint64(0); i < shows; i++ {
				next := l.Color(c)
				if next == c {
					break
				}
				c = next
			}
			dst.SetRGBA(x, y, c)
		}
	}
}

// Texture maps every packed input to its expanded output for GPU submission.
func (l *Lookup) Texture() *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, 256, 256))
	shift, mask := 11, 63
	if l.Layout == RGB555 {
		shift, mask = 10, 31
	}
	for p := 0; p < 65536; p++ {
		w := l.Word(uint16(p))
		dst.SetRGBA(p&255, p>>8, color.RGBA{uint8(int(w>>shift) * 255 / 31), uint8(int(w>>5&uint16(mask)) * 255 / mask), uint8(int(w&31) * 255 / 31), 255})
	}
	return dst
}
