package audio

import (
	"image"
	"testing"
)

func TestPositionalTermsPublishedControls(t *testing.T) {
	view := ViewGeometry{Origin: image.Pt(8, 8), Span: image.Pt(15, 15)}
	for _, test := range []struct {
		name string
		fine image.Point
		want SpatialTerms
	}{
		{"centre", image.Pt(3968, 3968), SpatialTerms{0, 0}},
		{"one-horizontal-cell", image.Pt(4224, 3968), SpatialTerms{-13, 266}},
		{"eight-horizontal-cells", image.Pt(6016, 3968), SpatialTerms{-171, 2133}},
		{"eight-vertical-cells", image.Pt(3968, 6016), SpatialTerms{-171, 0}},
		{"eight-left-cells", image.Pt(1920, 3968), SpatialTerms{-171, -2133}},
		{"eighty-horizontal-cells", image.Pt(24448, 3968), SpatialTerms{-10000, 10000}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := PositionalTerms(test.fine, view)
			t.Logf("fine=%v origin=%v span=%v terms=%+v valid=%v", test.fine, view.Origin, view.Span, got, ok)
			if !ok || got != test.want {
				t.Fatalf("ordinary positional terms=%+v/%v, want %+v/true", got, ok, test.want)
			}
		})
	}
}

func TestPositionalTermsUsesFineWorldAndView(t *testing.T) {
	view := ViewGeometry{Origin: image.Pt(8, 8), Span: image.Pt(15, 15)}
	got, ok := PositionalTerms(image.Pt(3840, 3840), view)
	if !ok || got != (SpatialTerms{-9, -133}) {
		t.Fatal("fine source was rounded to a cell or odd view centre was rounded", got, ok)
	}
	view.Origin = image.Pt(18, 28)
	translated, ok := PositionalTerms(image.Pt(6400, 8960), view)
	if !ok || translated != got {
		t.Fatal("world and view translation changed their relative spatial terms", translated, got, ok)
	}
}

func TestAmbientTermsPublishedControls(t *testing.T) {
	view := ViewGeometry{Origin: image.Pt(8, 8), Span: image.Pt(15, 15)}
	for _, test := range []struct {
		name       string
		cells      []image.Point
		population bool
		want       SpatialTerms
	}{
		{"centre", []image.Point{image.Pt(15, 15)}, false, SpatialTerms{0, 0}},
		{"seven-cells", []image.Point{image.Pt(22, 15)}, false, SpatialTerms{0, 933}},
		{"eight-cells", []image.Point{image.Pt(23, 15)}, false, SpatialTerms{-172, 1047}},
		{"two-seven-sixteen-cells", []image.Point{image.Pt(22, 15), image.Pt(31, 15)}, false, SpatialTerms{0, 1402}},
		{"bird-two-seven-sixteen-cells", []image.Point{image.Pt(22, 15), image.Pt(31, 15)}, true, SpatialTerms{-1934, 1402}},
		{"vertical-eight-cells", []image.Point{image.Pt(15, 23)}, false, SpatialTerms{-172, 0}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := AmbientTerms(test.cells, view, test.population)
			t.Logf("cells=%v origin=%v span=%v population=%v terms=%+v valid=%v", test.cells, view.Origin, view.Span, test.population, got, ok)
			if !ok || got != test.want {
				t.Fatalf("ordinary ambient terms=%+v/%v, want %+v/true", got, ok, test.want)
			}
		})
	}
}

func TestAmbientTermsRetainsZeroWeightSources(t *testing.T) {
	view := ViewGeometry{Origin: image.Pt(8, 8), Span: image.Pt(15, 15)}
	for _, test := range []struct {
		cells      []image.Point
		population bool
		want       SpatialTerms
	}{
		{[]image.Point{image.Pt(55, 15)}, false, SpatialTerms{-10000, 0}},
		{[]image.Point{image.Pt(22, 15), image.Pt(55, 15)}, false, SpatialTerms{0, 466}},
		{[]image.Point{image.Pt(22, 15), image.Pt(55, 15)}, true, SpatialTerms{-1934, 466}},
		{[]image.Point{image.Pt(55, 15), image.Pt(55, 15)}, true, SpatialTerms{-1934, 0}},
	} {
		got, ok := AmbientTerms(test.cells, view, test.population)
		t.Logf("cells=%v population=%v terms=%+v valid=%v", test.cells, test.population, got, ok)
		if !ok || got != test.want {
			t.Fatal("zero-weight source was refused, omitted from count or used as a pan divisor", got, ok, test.want)
		}
	}
}

func TestAmbientTermsQuantizationAndSignedAverage(t *testing.T) {
	view := ViewGeometry{Origin: image.Pt(20, 20), Span: image.Pt(15, 15)}
	got, ok := AmbientTerms([]image.Point{image.Pt(20, 27), image.Pt(11, 27)}, view, false)
	if !ok || got != (SpatialTerms{0, -1402}) {
		t.Fatal("negative weighted pan did not truncate toward zero", got, ok)
	}
	got, ok = AmbientTerms([]image.Point{image.Pt(33, 32)}, view, false)
	if !ok || got != (SpatialTerms{0, 800}) {
		t.Fatal("ambient radius was not quantized before the exponential", got, ok)
	}
}

func TestAmbientTermsPopulationCap(t *testing.T) {
	view := ViewGeometry{Span: image.Pt(16, 16)}
	cells := make([]image.Point, 31)
	for i := range cells {
		cells[i] = image.Pt(8, 8)
	}
	for _, count := range []int{1, 30, 31} {
		want := -1000
		if count == 1 {
			want = -1967
		}
		got, ok := AmbientTerms(cells[:count], view, true)
		if !ok || got != (SpatialTerms{want, 0}) {
			t.Fatal("population term changed count truncation or its cap", count, got, ok)
		}
	}
}

func TestSpatialTermsInvalidViewAndEmptySource(t *testing.T) {
	for _, span := range []image.Point{image.Pt(0, 15), image.Pt(15, 0), image.Pt(-1, 15), image.Pt(15, -1)} {
		view := ViewGeometry{Span: span}
		if got, ok := PositionalTerms(image.Point{}, view); ok || got != (SpatialTerms{}) {
			t.Fatal("invalid positional span accepted", span, got, ok)
		}
		if got, ok := AmbientTerms([]image.Point{image.Point{}}, view, false); ok || got != (SpatialTerms{}) {
			t.Fatal("invalid ambient span accepted", span, got, ok)
		}
	}
	if got, ok := AmbientTerms(nil, ViewGeometry{Span: image.Pt(15, 15)}, false); ok || got != (SpatialTerms{}) {
		t.Fatal("empty ambient population accepted", got, ok)
	}
}

func TestSpatialPlacementPortablePolicy(t *testing.T) {
	for _, test := range []struct {
		terms SpatialTerms
		want  Placement
	}{
		{SpatialTerms{0, 0}, Placement{10000, 10000}},
		{SpatialTerms{-2000, 0}, Placement{1000, 1000}},
		{SpatialTerms{-2000, 5000}, Placement{500, 1500}},
		{SpatialTerms{-2000, -5000}, Placement{1500, 500}},
		{SpatialTerms{-10000, 10000}, Placement{0, 0}},
		{SpatialTerms{0, 20000}, Placement{0, 10000}},
		{SpatialTerms{0, -20000}, Placement{10000, 0}},
		{SpatialTerms{100, 0}, Placement{10000, 10000}},
	} {
		before := test.terms
		got := SpatialPlacement(test.terms)
		t.Logf("portable terms=%+v placement=%+v", test.terms, got)
		if got != test.want || test.terms != before {
			t.Fatal("portable conversion changed signed terms or gain policy", got, test.want, test.terms, before)
		}
	}
}
