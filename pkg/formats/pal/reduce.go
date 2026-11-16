package pal

import (
	"cmp"
	"fmt"
	"image"
	"image/color"
	"slices"
)

// Reduce is the one colour reduction: pic's colours brought to at most n
// table entries (1 to EntryCount) by median cut, each pixel the entry of the
// box its colour falls in, with no dithering. A picture of n colours or
// fewer keeps every colour exactly. Colours are taken in sorted order, so
// the same picture always gives the same table. Alpha is dropped.
func Reduce(pic image.Image, n int) (*image.Paletted, error) {
	if n < 1 || n > EntryCount {
		return nil, fmt.Errorf("pal: %d colours, want 1 to %d", n, EntryCount)
	}
	r := pic.Bounds()
	if r.Empty() {
		return nil, fmt.Errorf("pal: empty picture")
	}
	key := func(x, y int) uint32 {
		c := color.NRGBAModel.Convert(pic.At(x, y)).(color.NRGBA)
		return uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B)
	}
	counts := map[uint32]int{}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			counts[key(x, y)]++
		}
	}
	type bucket struct {
		key   uint32
		count int
	}
	all := make([]bucket, 0, len(counts))
	for k, c := range counts {
		all = append(all, bucket{k, c})
	}
	slices.SortFunc(all, func(a, b bucket) int { return cmp.Compare(a.key, b.key) })
	channel := func(k uint32, ch int) int { return int(k>>(16-8*ch)) & 0xff }
	// widest is the channel of b whose values spread furthest, and the spread.
	widest := func(b []bucket) (int, int) {
		best, spread := 0, -1
		for ch := range 3 {
			lo, hi := 255, 0
			for _, e := range b {
				v := channel(e.key, ch)
				lo, hi = min(lo, v), max(hi, v)
			}
			if hi-lo > spread {
				best, spread = ch, hi-lo
			}
		}
		return best, spread
	}
	boxes := [][]bucket{all}
	for len(boxes) < n {
		at, ch, spread := -1, 0, 0
		for i, b := range boxes {
			if len(b) < 2 {
				continue
			}
			if c, s := widest(b); s > spread || at < 0 {
				at, ch, spread = i, c, s
			}
		}
		if at < 0 {
			break
		}
		b := boxes[at]
		slices.SortFunc(b, func(p, q bucket) int {
			return cmp.Or(cmp.Compare(channel(p.key, ch), channel(q.key, ch)), cmp.Compare(p.key, q.key))
		})
		total := 0
		for _, e := range b {
			total += e.count
		}
		cut, sum := 1, b[0].count
		for cut < len(b)-1 && 2*sum < total {
			sum += b[cut].count
			cut++
		}
		boxes[at] = b[:cut]
		boxes = append(boxes, b[cut:])
	}
	palette := make(color.Palette, len(boxes))
	index := map[uint32]uint8{}
	for i, b := range boxes {
		var sum [3]int
		total := 0
		for _, e := range b {
			for ch := range 3 {
				sum[ch] += channel(e.key, ch) * e.count
			}
			total += e.count
			index[e.key] = uint8(i)
		}
		mean := func(ch int) uint8 { return uint8((sum[ch] + total/2) / total) }
		palette[i] = color.RGBA{R: mean(0), G: mean(1), B: mean(2), A: 0xff}
	}
	out := image.NewPaletted(image.Rect(0, 0, r.Dx(), r.Dy()), palette)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			out.Pix[(y-r.Min.Y)*out.Stride+x-r.Min.X] = index[key(x, y)]
		}
	}
	return out, nil
}
