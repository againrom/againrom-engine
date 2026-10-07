package refraction

import (
	"image"
	"image/draw"
	"math"
	"sort"
)

type Map struct {
	Size    int
	Offsets []image.Point
}

type Sample struct {
	Destination image.Point
	Source      image.Point
}

// Picture7Map builds the coordinate table established by ANIM-098.
func Picture7Map() *Map {
	m := &Map{Size: 19, Offsets: make([]image.Point, 19*19)}
	var factors [19]float64
	factors[0] = 1
	for k := 1; k < len(factors); k++ {
		factors[k] = float64(k) / (20 * math.Sin(math.Atan(float64(k)/4)))
	}
	for i := 0; i < m.Size; i++ {
		for j := 0; j < m.Size; j++ {
			p := image.Pt(i, j)
			k := int(math.Sqrt(float64(i*i+j*j)) + 0.5)
			if k < m.Size {
				p = image.Pt(int(float64(i)*factors[k]+0.5), int(float64(j)*factors[k]+0.5))
			}
			m.Offsets[i*m.Size+j] = p
		}
	}
	return m
}

func (m *Map) Valid() bool {
	return m != nil && m.Size > 0 && m.Size <= 256 && len(m.Offsets) == m.Size*m.Size
}

// Resolve follows ordered in-place copies, retaining each final pixel's
// original source coordinate. Both points must meet clip (ANIM-097/099).
func Resolve(m *Map, center image.Point, clip image.Rectangle) []Sample {
	if !m.Valid() || clip.Empty() {
		return nil
	}
	sources := make(map[image.Point]image.Point)
	for i := m.Size - 1; i >= 0; i-- {
		for j := m.Size - 1; j >= 0; j-- {
			p := m.Offsets[i*m.Size+j]
			for _, sign := range [4]image.Point{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
				d := center.Add(image.Pt(i*sign.X, j*sign.Y))
				s := center.Add(image.Pt(p.X*sign.X, p.Y*sign.Y))
				if !d.In(clip) || !s.In(clip) {
					continue
				}
				original := s
				if prior, ok := sources[s]; ok {
					original = prior
				}
				sources[d] = original
			}
		}
	}
	out := make([]Sample, 0, len(sources))
	for d, s := range sources {
		out = append(out, Sample{Destination: d, Source: s})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Destination.Y < out[j].Destination.Y ||
			out[i].Destination.Y == out[j].Destination.Y && out[i].Destination.X < out[j].Destination.X
	})
	return out
}

func SourceBounds(samples []Sample) image.Rectangle {
	var bounds image.Rectangle
	for _, sample := range samples {
		bounds = bounds.Union(image.Rectangle{Min: sample.Source, Max: sample.Source.Add(image.Pt(1, 1))})
	}
	return bounds
}

func Apply(dst *image.RGBA, m *Map, center image.Point, clip image.Rectangle) {
	if dst == nil {
		return
	}
	samples := Resolve(m, center, clip.Intersect(dst.Rect))
	region := SourceBounds(samples)
	if region.Empty() {
		return
	}
	snapshot := image.NewRGBA(region)
	draw.Draw(snapshot, region, dst, region.Min, draw.Src)
	for _, sample := range samples {
		dst.SetRGBA(sample.Destination.X, sample.Destination.Y, snapshot.RGBAAt(sample.Source.X, sample.Source.Y))
	}
}
