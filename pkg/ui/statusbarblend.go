package ui

import (
	"image"
	"image/color"
	"math"

	"againrom/pkg/render/terrain"
	"github.com/hajimehoshi/ebiten/v2"
)

func statusBarPixelRect(r screenRect, bounds image.Rectangle) image.Rectangle {
	return image.Rect(int(math.Ceil(r.X-0.5)), int(math.Ceil(r.Y-0.5)),
		int(math.Ceil(r.X+r.W-0.5)), int(math.Ceil(r.Y+r.H-0.5))).Intersect(bounds)
}

func blendStatusBarRect(dst *image.RGBA, r screenRect, c color.RGBA) {
	box := statusBarPixelRect(r, dst.Bounds())
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			dst.SetRGBA(x, y, terrain.StatusBarBlend(c, dst.RGBAAt(x, y)))
		}
	}
}

const statusBarHalfShader = `//kage:unit pixels
package main

var HalfRGB vec3

func Fragment(dst vec4, src vec2, colour vec4) vec4 {
	b := floor(imageSrc0At(src).rgb*255.0+0.5)
	p := floor(b/vec3(8.0,4.0,8.0))
	mixed := floor(p/2.0)+HalfRGB
	return vec4(floor(mixed*vec3(255.0/31.0,255.0/63.0,255.0/31.0))/255.0,1.0)
}
`

var statusBarHalfProgram *ebiten.Shader

var copyStatusBarSnapshot = func(snapshot, target *ebiten.Image, region image.Rectangle) {
	op := ebiten.DrawImageOptions{Blend: ebiten.BlendCopy}
	snapshot.DrawImage(target.SubImage(region).(*ebiten.Image), &op)
}

var submitStatusBarHalf = func(target *ebiten.Image, vertices []ebiten.Vertex, shader *ebiten.Shader, op *ebiten.DrawTrianglesShaderOptions) {
	target.DrawTrianglesShader(vertices, []uint16{0, 1, 2, 1, 2, 3}, shader, op)
}

// Each rectangle snapshots the current destination so overlapping fills compound.
func (v *Viewer) drawStatusBarHalfRect(target *ebiten.Image, r screenRect, c color.RGBA) {
	region := statusBarPixelRect(r, target.Bounds())
	if region.Empty() {
		return
	}
	if statusBarHalfProgram == nil {
		var err error
		statusBarHalfProgram, err = ebiten.NewShader([]byte(statusBarHalfShader))
		if err != nil {
			panic(err)
		}
	}
	if v.statusBarScratch == nil || v.statusBarScratch.Bounds().Dx() < region.Dx() || v.statusBarScratch.Bounds().Dy() < region.Dy() {
		size := region.Size()
		if v.statusBarScratch != nil {
			size.X = max(size.X, v.statusBarScratch.Bounds().Dx())
			size.Y = max(size.Y, v.statusBarScratch.Bounds().Dy())
			v.statusBarScratch.Dispose()
		}
		v.statusBarScratch = ebiten.NewImage(size.X, size.Y)
	}
	copyStatusBarSnapshot(v.statusBarScratch, target, region)
	x, y, w, h := float32(region.Min.X), float32(region.Min.Y), float32(region.Dx()), float32(region.Dy())
	vertices := []ebiten.Vertex{
		{DstX: x, DstY: y, SrcX: 0, SrcY: 0},
		{DstX: x + w, DstY: y, SrcX: w, SrcY: 0},
		{DstX: x, DstY: y + h, SrcX: 0, SrcY: h},
		{DstX: x + w, DstY: y + h, SrcX: w, SrcY: h},
	}
	op := ebiten.DrawTrianglesShaderOptions{Blend: ebiten.BlendCopy,
		Images:   [4]*ebiten.Image{v.statusBarScratch},
		Uniforms: map[string]any{"HalfRGB": []float32{float32(c.R / 8 / 2), float32(c.G / 4 / 2), float32(c.B / 8 / 2)}}}
	submitStatusBarHalf(target.SubImage(region).(*ebiten.Image), vertices, statusBarHalfProgram, &op)
}
