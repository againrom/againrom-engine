package audio

import (
	"image"
	"math"
)

type ViewGeometry struct {
	Origin image.Point
	Span   image.Point
}

type SpatialTerms struct {
	Attenuation int
	Pan         int
}

func PositionalTerms(fine image.Point, view ViewGeometry) (SpatialTerms, bool) {
	if view.Span.X <= 0 || view.Span.Y <= 0 {
		return SpatialTerms{}, false
	}
	dx := float64(fine.X) - (float64(view.Origin.X)*256 + float64(view.Span.X)*128)
	dy := float64(fine.Y) - (float64(view.Origin.Y)*256 + float64(view.Span.Y)*128)
	pan := math.Trunc(dx * 4000 / (float64(view.Span.X) * 256))
	distance := math.Sqrt(dx*dx+dy*dy) / 256 / 8
	attenuation := math.Max(-10000, -(math.Exp(distance)-1)*100)
	return SpatialTerms{Attenuation: int(attenuation), Pan: int(max(-10000, min(10000, pan)))}, true
}

func AmbientTerms(cells []image.Point, view ViewGeometry, populationGain bool) (SpatialTerms, bool) {
	if len(cells) == 0 || view.Span.X <= 0 || view.Span.Y <= 0 {
		return SpatialTerms{}, false
	}
	listenerX := float64(view.Origin.X) + math.Floor(float64(view.Span.X)/2)
	listenerY := float64(view.Origin.Y) + math.Floor(float64(view.Span.Y)/2)
	maximum, panSum := 0, int64(0)
	for _, cell := range cells {
		dx, dy := float64(cell.X)-listenerX, float64(cell.Y)-listenerY
		rawPan := max(-2000, min(2000, math.Trunc(dx*2000/float64(view.Span.X))))
		radius := math.Trunc(math.Sqrt(dx*dx + dy*dy))
		q := math.Trunc(radius / 8)
		weight := int(math.Max(0, math.Trunc(10000-(math.Exp(q)-1)*100)))
		maximum = max(maximum, weight)
		panSum += int64(math.Trunc(rawPan * float64(weight) / 10000))
	}
	attenuation := maximum - 10000
	if populationGain {
		attenuation = int(min(math.Trunc(float64(len(cells))*1000/30), 1000)) - 2000
	}
	return SpatialTerms{Attenuation: attenuation, Pan: int(panSum / int64(len(cells)))}, true
}

func SpatialPlacement(term SpatialTerms) Placement {
	attenuation := max(-10000, min(0, term.Attenuation))
	pan := int64(max(-GainUnit, min(GainUnit, term.Pan)))
	gain := int64(math.Pow(10, float64(attenuation)/2000) * GainUnit)
	return Placement{
		Left:  clampGain(gain * (GainUnit - pan) / GainUnit),
		Right: clampGain(gain * (GainUnit + pan) / GainUnit),
	}
}
