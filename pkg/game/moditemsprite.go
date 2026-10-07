package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"sort"
)

// A mod's item picture is a PNG. The game reads item pictures as .16a sprites,
// so the PNG is converted once at start-up into the stream the game's own
// reader decodes: a 256-entry palette, one frame, and per pixel a palette index
// and a 4-bit level that sets how much of the pixel covers what is under it.

// maxModIconSide is the largest side of a mod item picture: the inventory and
// shop cells are 80 by 80.
const maxModIconSide = 80

// modIconSprite converts PNG bytes into a .16a stream. The error says why the
// picture was refused.
func modIconSprite(src []byte) ([]byte, error) {
	cfg, err := png.DecodeConfig(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("not a readable PNG: %v", err)
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > maxModIconSide || cfg.Height > maxModIconSide {
		return nil, fmt.Errorf("the picture is %dx%d; an item picture is 1 to %d pixels on each side", cfg.Width, cfg.Height, maxModIconSide)
	}
	img, err := png.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("not a readable PNG: %v", err)
	}
	return encodeSpriteA(img)
}

type iconPixel struct {
	r, g, b uint8
	level   uint8
	painted bool
}

// encodeSpriteA writes img as a one-frame .16a stream. A pixel with no alpha is
// left unpainted; every other pixel takes a palette entry of its colour and a
// level of its alpha in sixteenths. More than 256 colours are reduced by
// dropping the low bits of each channel until the palette fits.
func encodeSpriteA(img image.Image) ([]byte, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	pixels := make([]iconPixel, 0, w*h)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if c.A == 0 {
				pixels = append(pixels, iconPixel{})
				continue
			}
			level := (int(c.A)*16+127)/255 - 1
			level = max(0, min(15, level))
			pixels = append(pixels, iconPixel{r: c.R, g: c.G, b: c.B, level: uint8(level), painted: true})
		}
	}
	visible := false
	for _, p := range pixels {
		visible = visible || p.painted
	}
	if !visible {
		return nil, fmt.Errorf("the picture has no visible pixel")
	}
	drop := 0
	var palette []iconPixel
	index := map[[3]uint8]int{}
	for ; drop <= 8; drop++ {
		palette, index = nil, map[[3]uint8]int{}
		mask := uint8(0xff) << drop
		for _, p := range pixels {
			if !p.painted {
				continue
			}
			key := [3]uint8{p.r & mask, p.g & mask, p.b & mask}
			if _, ok := index[key]; !ok {
				index[key] = -1
				palette = append(palette, iconPixel{r: key[0], g: key[1], b: key[2]})
			}
		}
		if len(palette) <= 256 {
			break
		}
	}
	sort.Slice(palette, func(i, j int) bool {
		a, b := palette[i], palette[j]
		if a.r != b.r {
			return a.r < b.r
		}
		if a.g != b.g {
			return a.g < b.g
		}
		return a.b < b.b
	})
	mask := uint8(0xff) << drop
	for i, p := range palette {
		index[[3]uint8{p.r, p.g, p.b}] = i
	}

	var out bytes.Buffer
	pal := make([]byte, 1024)
	for i, p := range palette {
		pal[i*4], pal[i*4+1], pal[i*4+2] = p.b, p.g, p.r
	}
	out.Write(pal)

	var block bytes.Buffer
	put := func(v uint16) { _ = binary.Write(&block, binary.LittleEndian, v) }
	skip := 0
	flushSkip := func() {
		for skip > 0 {
			n := min(skip, 0x3fff)
			put(0x8000 | uint16(n))
			skip -= n
		}
	}
	for i := 0; i < len(pixels); {
		if !pixels[i].painted {
			skip++
			i++
			continue
		}
		flushSkip()
		j := i
		for j < len(pixels) && pixels[j].painted && j-i < 0x3fff {
			j++
		}
		put(uint16(j - i))
		for ; i < j; i++ {
			p := pixels[i]
			idx := index[[3]uint8{p.r & mask, p.g & mask, p.b & mask}]
			put(uint16(idx)<<1 | uint16(p.level)<<9)
		}
	}
	// A transparent tail is left to the decoder, which starts with every cell
	// unpainted.
	var head [12]byte
	binary.LittleEndian.PutUint32(head[0:], uint32(w))
	binary.LittleEndian.PutUint32(head[4:], uint32(h))
	binary.LittleEndian.PutUint32(head[8:], uint32(block.Len()))
	out.Write(head[:])
	out.Write(block.Bytes())
	var trailer [4]byte
	binary.LittleEndian.PutUint32(trailer[:], 1)
	out.Write(trailer[:])
	return out.Bytes(), nil
}
