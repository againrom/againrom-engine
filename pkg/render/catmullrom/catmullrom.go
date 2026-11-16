// Package catmullrom is the final-frame scaler the owner chose for the window
// (engine/docs/HOTFIXES.md): the separable 4x4 cubic convolution of the
// Mitchell-Netravali family, by default Catmull-Rom (B = 0, C = 1/2). It holds
// the Kage source the UI compiles, which takes B and C as data, and a pure-Go
// reference resampler with the same arithmetic.
//
// Destination pixel d on one axis, with scale s and origin o, samples source
// position p = (d + 0.5 - o) / s. With t = p - 0.5, i = floor(t) and
// f = t - i, the taps are i-1, i, i+1 and i+2, each clamped to the source's
// own bounds, weighted by Weights(f). The stored premultiplied values are
// interpolated directly; the result is clamped to [0,1] per channel and then
// r, g and b to at most alpha.
package catmullrom

import (
	"image"
	"math"
)

// Kernel is one member of the Mitchell-Netravali family by its B and C.
type Kernel struct{ B, C float64 }

// The two kernels the window offers: Catmull-Rom is the default picture;
// Mitchell is the softer B = C = 1/3 of the published recommendation.
var (
	CatmullRom = Kernel{B: 0, C: 0.5}
	Mitchell   = Kernel{B: 1.0 / 3, C: 1.0 / 3}
)

// At is the kernel's value at distance x from the tap, 0 <= x < 2.
func (k Kernel) At(x float64) float64 {
	b, c := k.B, k.C
	x2 := x * x
	x3 := x2 * x
	if x < 1 {
		return ((12-9*b-6*c)*x3 + (-18+12*b+6*c)*x2 + (6 - 2*b)) / 6
	}
	return ((-b-6*c)*x3 + (6*b+30*c)*x2 + (-12*b-48*c)*x + (8*b + 24*c)) / 6
}

// Weights answers the four tap weights for fraction f in [0,1): the kernel
// evaluated at 1+f, f, 1-f and 2-f.
func (k Kernel) Weights(f float64) [4]float64 {
	if k == CatmullRom {
		// The default picture keeps the closed form it was pinned with.
		f2 := f * f
		f3 := f2 * f
		return [4]float64{(-f3 + 2*f2 - f) / 2, (3*f3 - 5*f2 + 2) / 2, (-3*f3 + 4*f2 + f) / 2, (f3 - f2) / 2}
	}
	return [4]float64{k.At(1 + f), k.At(f), k.At(1 - f), k.At(2 - f)}
}

// Weights is Catmull-Rom's four tap weights for fraction f.
func Weights(f float64) [4]float64 { return CatmullRom.Weights(f) }

// Taps answers the four source indices, clamped to [0, n), and their weights
// for source position p in pixels, where pixel k spans [k, k+1).
func (k Kernel) Taps(p float64, n int) (idx [4]int, w [4]float64) {
	t := p - 0.5
	i := math.Floor(t)
	w = k.Weights(t - i)
	for j := range idx {
		idx[j] = min(max(int(i)+j-1, 0), n-1)
	}
	return idx, w
}

// Taps is Catmull-Rom's taps.
func Taps(p float64, n int) (idx [4]int, w [4]float64) { return CatmullRom.Taps(p, n) }

// Covered answers the destination pixel range [lo, hi) whose centres fall in
// [o, o + n*s): the pixels a draw of an n-pixel source at scale s and origin
// o fills.
func Covered(n int, s, o float64) (lo, hi int) {
	return int(math.Ceil(o - 0.5)), int(math.Ceil(o + float64(n)*s - 0.5))
}

// Resample draws src into dst at uniform scale s with its top-left corner at
// (ox, oy) in dst's coordinates, replacing every covered pixel of dst.Rect
// with the filtered result. Pixels outside the placed rectangle are left
// unchanged.
func Resample(dst, src *image.RGBA, s, ox, oy float64) {
	CatmullRom.Resample(dst, src, s, ox, oy)
}

// Resample is the package Resample with this kernel.
func (k Kernel) Resample(dst, src *image.RGBA, s, ox, oy float64) {
	sb := src.Rect
	sw, sh := sb.Dx(), sb.Dy()
	if sw <= 0 || sh <= 0 || s <= 0 {
		return
	}
	x0, x1 := Covered(sw, s, ox)
	y0, y1 := Covered(sh, s, oy)
	x0, x1 = max(x0, dst.Rect.Min.X), min(x1, dst.Rect.Max.X)
	y0, y1 = max(y0, dst.Rect.Min.Y), min(y1, dst.Rect.Max.Y)
	for y := y0; y < y1; y++ {
		ry, wy := k.Taps((float64(y)+0.5-oy)/s, sh)
		for x := x0; x < x1; x++ {
			rx, wx := k.Taps((float64(x)+0.5-ox)/s, sw)
			var acc [4]float64
			for ty := range 4 {
				var row [4]float64
				for tx := range 4 {
					o := src.PixOffset(sb.Min.X+rx[tx], sb.Min.Y+ry[ty])
					for c := range 4 {
						row[c] += wx[tx] * float64(src.Pix[o+c]) / 255
					}
				}
				for c := range 4 {
					acc[c] += wy[ty] * row[c]
				}
			}
			a := clamp01(acc[3])
			o := dst.PixOffset(x, y)
			for c := range 3 {
				dst.Pix[o+c] = byte(math.Round(min(clamp01(acc[c]), a) * 255))
			}
			dst.Pix[o+3] = byte(math.Round(a * 255))
		}
	}
}

func clamp01(v float64) float64 { return min(max(v, 0), 1) }

// Shader is the Kage source of the GPU path. It is drawn with
// DrawRectShader over the source's own size under the caller's Scale then
// Translate, and takes that scale and origin again as the uniform Placement
// (sx, sy, ox, oy): Ebitengine moves quad vertices off pixel centres, so the
// interpolated srcPos drifts from the op's own mapping, and each fragment
// derives its source position from its destination pixel instead. It
// fetches at texel centres clamped to the source region, never into
// neighbouring atlas content.
const Shader = `//kage:unit pixels

package main

var Placement vec4

// Kernel is the kernel's (B, C).
var Kernel vec2

func near(x float) float {
	b, c := Kernel.x, Kernel.y
	return ((12-9*b-6*c)*x*x*x + (-18+12*b+6*c)*x*x + (6 - 2*b)) / 6
}

func far(x float) float {
	b, c := Kernel.x, Kernel.y
	return ((-b-6*c)*x*x*x + (6*b+30*c)*x*x + (-12*b-48*c)*x + (8*b + 24*c)) / 6
}

func weights(f float) vec4 {
	return vec4(far(1+f), near(f), near(1-f), far(2-f))
}

func row(y float, x vec4, w vec4) vec4 {
	return w.x*imageSrc0UnsafeAt(vec2(x.x, y)) + w.y*imageSrc0UnsafeAt(vec2(x.y, y)) +
		w.z*imageSrc0UnsafeAt(vec2(x.z, y)) + w.w*imageSrc0UnsafeAt(vec2(x.w, y))
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	origin := imageSrc0Origin()
	lo := origin + 0.5
	hi := origin + imageSrc0Size() - 0.5
	t := (dstPos.xy-imageDstOrigin()-Placement.zw)/Placement.xy - 0.5
	i := floor(t)
	f := t - i
	c := origin + i + 0.5
	xs := clamp(vec4(c.x-1, c.x, c.x+1, c.x+2), vec4(lo.x), vec4(hi.x))
	ys := clamp(vec4(c.y-1, c.y, c.y+1, c.y+2), vec4(lo.y), vec4(hi.y))
	wx := weights(f.x)
	wy := weights(f.y)
	acc := wy.x*row(ys.x, xs, wx) + wy.y*row(ys.y, xs, wx) + wy.z*row(ys.z, xs, wx) + wy.w*row(ys.w, xs, wx)
	acc = clamp(acc*color, vec4(0), vec4(1))
	return vec4(min(acc.rgb, vec3(acc.a)), acc.a)
}
`
