package ui

import (
	"image"

	"againrom/pkg/render/refraction"
	"github.com/hajimehoshi/ebiten/v2"
)

type backgroundDeformationDraw struct {
	Center  image.Point
	Clip    image.Rectangle
	Mapping *refraction.Map
}

var defaultDeformationMap = refraction.Picture7Map()

func (v *Viewer) deformationPlacement(b SpellBolt) (effectScreenRect, bool) {
	mapping := b.Mapping
	if mapping == nil {
		mapping = defaultDeformationMap
	}
	if !mapping.Valid() {
		return effectScreenRect{}, false
	}
	point := b.groundPoint()
	px, py := point.X, point.Y+v.spellBoltLift(b)
	x, y := v.cam.WorldToScreen(float64(px), float64(py))
	center := image.Pt(int(x), int(y))
	radius := mapping.Size - 1
	r := image.Rect(center.X-radius, center.Y-radius, center.X+radius+1, center.Y+radius+1)
	if r.Intersect(image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH)).Empty() {
		return effectScreenRect{}, false
	}
	return effectScreenRect{screenRect: screenRect{X: float64(r.Min.X), Y: float64(r.Min.Y), W: float64(r.Dx()), H: float64(r.Dy())},
		Cell: b.Cell, Pass: b.Pass, Effect: b.Effect, Center: center, Mapping: mapping}, true
}

func (v *Viewer) drawBackgroundDeformation(target imageTarget, s effectScreenRect) {
	draw := backgroundDeformationDraw{Center: s.Center, Clip: image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH), Mapping: s.Mapping}
	if cpu, ok := target.(interface {
		DrawBackgroundDeformation(backgroundDeformationDraw)
	}); ok {
		cpu.DrawBackgroundDeformation(draw)
		return
	}
	if gpu, ok := target.(*ebiten.Image); ok {
		drawBackgroundDeformationGPU(gpu, draw)
	}
}

var copyDeformationSnapshot = func(snapshot, target *ebiten.Image, region image.Rectangle) {
	snapshot.DrawImage(target.SubImage(region).(*ebiten.Image), nil)
}

var submitDeformationPixels = func(target, snapshot *ebiten.Image, vertices []ebiten.Vertex, indices []uint16, op *ebiten.DrawTrianglesOptions) {
	target.DrawTriangles(vertices, indices, snapshot, op)
}

func drawBackgroundDeformationGPU(target *ebiten.Image, draw backgroundDeformationDraw) {
	samples := refraction.Resolve(draw.Mapping, draw.Center, draw.Clip.Intersect(target.Bounds()))
	region := refraction.SourceBounds(samples)
	if region.Empty() {
		return
	}
	snapshot := ebiten.NewImage(region.Dx(), region.Dy())
	defer snapshot.Dispose()
	copyDeformationSnapshot(snapshot, target, region)
	var op ebiten.DrawTrianglesOptions
	op.Filter, op.Blend = ebiten.FilterNearest, ebiten.BlendCopy
	for lo := 0; lo < len(samples); lo += 16383 {
		batch := samples[lo:min(lo+16383, len(samples))]
		vertices := make([]ebiten.Vertex, 4*len(batch))
		indices := make([]uint16, 6*len(batch))
		for i, sample := range batch {
			d, s := sample.Destination, sample.Source.Sub(region.Min)
			for k, corner := range [4]image.Point{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
				vertices[4*i+k] = ebiten.Vertex{DstX: float32(d.X + corner.X), DstY: float32(d.Y + corner.Y),
					SrcX: float32(s.X + corner.X), SrcY: float32(s.Y + corner.Y), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
			}
			for k, index := range [6]uint16{0, 1, 2, 1, 3, 2} {
				indices[6*i+k] = uint16(4*i) + index
			}
		}
		submitDeformationPixels(target, snapshot, vertices, indices, &op)
	}
}
