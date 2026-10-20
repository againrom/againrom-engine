package ui

import (
	"image"
	"reflect"
	"testing"
)

func dollHoverMask() *SlotMask {
	mask := &SlotMask{W: 4, H: 2, Slot: make([]uint8, 8)}
	mask.Slot[1] = 3
	mask.Slot[6] = 5
	return mask
}

var dollHoverInfo = [12][]string{
	2: {"Steel Helm", "#Armour 4", "#Value 180"},
	4: {"Oak Staff", "#Damage 3-9"},
}

// detailedChargenWithDoll is detailedChargen with a preview that carries a doll
// slot map and the item information of its worn slots.
func detailedChargenWithDoll(t *testing.T, info [12][]string) *Chargen {
	t.Helper()
	stats := make([]ChargenStat, 4)
	for i := range stats {
		stats[i] = ChargenStat{Floor: 0, Ceiling: 50, Start: 10 + i}
	}
	figure := image.NewRGBA(image.Rect(0, 0, 4, 2))
	c := NewChargen(ChargenSetup{
		Name:      "Danath",
		PreCreate: &ChargenPreCreate{Art: &ChargenPresentation{Layout: testGenerator(), Font: messageFont()}},
		Choices:   []ChargenChoice{{Options: []string{"male", "female"}, Parent: -1}},
		Stats:     stats, Cost: triangular(50), Budget: 5000,
		Preview: func(ChargenResult) ChargenPreview {
			return ChargenPreview{Doll: figure, Worn: &ChargenDollHover{Mask: dollHoverMask(), Info: info}}
		},
	})
	c.Forward()
	if c.Stage() != DetailedStage {
		t.Fatal("fixture did not reach DetailedStage")
	}
	return c
}

func TestTooltipChargenDollShowsWornItemInformation(t *testing.T) {
	c := detailedChargenWithDoll(t, dollHoverInfo)
	a := NewApp("chargen", nil, nil, nil)
	a.Layout(640, 480)
	a.flow.chargen, a.flow.screen = c, ScreenChargen
	d := c.layout().Detail
	origin := d.Doll.Rect.Rectangle().Min.Add(d.Preview.FigureOffset.Pt())
	for _, tc := range []struct {
		at   image.Point
		want []string
	}{
		{origin.Add(image.Pt(1, 0)), dollHoverInfo[2]},
		{origin.Add(image.Pt(2, 1)), dollHoverInfo[4]},
	} {
		got := a.chargenTooltip(tc.at)
		if !reflect.DeepEqual(got.lines, tc.want) || got.key() == "" {
			t.Errorf("doll pixel %v hint = %q, want %q", tc.at, got.lines, tc.want)
		}
	}
	for _, p := range []image.Point{origin, origin.Add(image.Pt(3, 1)), origin.Add(image.Pt(40, 40)), image.Pt(1, 1)} {
		if got := a.chargenTooltip(p); reflect.DeepEqual(got.lines, dollHoverInfo[2]) || reflect.DeepEqual(got.lines, dollHoverInfo[4]) {
			t.Errorf("blank or outside pixel %v shows item information %q", p, got.lines)
		}
	}
}

// With no item information on the preview the same pixels name nothing: the
// information reaches the hover only through the preview's SlotInfo.
func TestTooltipChargenDollWithoutInformationNamesNothing(t *testing.T) {
	c := detailedChargenWithDoll(t, [12][]string{})
	a := NewApp("chargen", nil, nil, nil)
	a.Layout(640, 480)
	a.flow.chargen, a.flow.screen = c, ScreenChargen
	d := c.layout().Detail
	origin := d.Doll.Rect.Rectangle().Min.Add(d.Preview.FigureOffset.Pt())
	if got := a.chargenTooltip(origin.Add(image.Pt(1, 0))); got.key() != "" && len(got.lines) != 0 {
		t.Errorf("empty slot information still hints %q", got.lines)
	}
}

// townDollPixels are two pixels of the town pane's figure, clear of every
// corner control, with the slot each is marked as.
var townDollPixels = [2]image.Point{{70, 100}, {90, 140}}

func townHeroSurface(statistics bool, info [12][]string) TownSurfaceView {
	mask := &SlotMask{W: 160, H: 240, Slot: make([]uint8, 160*240)}
	mask.Slot[townDollPixels[0].Y*160+townDollPixels[0].X] = 3
	mask.Slot[townDollPixels[1].Y*160+townDollPixels[1].X] = 5
	return TownSurfaceView{Kind: TownSurfaceSchool, Font: messageFont(),
		Hero:         TownCharacterView{HasSubject: true, Statistics: statistics, Font: messageFont()},
		HeroSlotMask: mask, HeroSlotInfo: info}
}

func TestTownHeroDollRoutesMarkedPixelsToItemInformation(t *testing.T) {
	v := townHeroSurface(false, dollHoverInfo)
	at := shopFigureRect.Min
	if got, ok := TownHeroHoverLines(v, at.Add(townDollPixels[0])); !ok || !reflect.DeepEqual(got, dollHoverInfo[2]) {
		t.Fatalf("worn pixel = %q, %v; want slot 3 lines", got, ok)
	}
	if got, ok := TownHeroHoverLines(v, at.Add(townDollPixels[1])); !ok || !reflect.DeepEqual(got, dollHoverInfo[4]) {
		t.Fatalf("worn pixel = %q, %v; want slot 5 lines", got, ok)
	}
	for _, p := range []image.Point{at.Add(image.Pt(71, 100)), at.Add(image.Pt(-1, 0)), image.Pt(10, 10)} {
		if got, ok := TownHeroHoverLines(v, p); ok || got != nil {
			t.Errorf("blank or outside pixel %v = %q, %v", p, got, ok)
		}
	}
	if got, ok := TownHeroHoverLines(townHeroSurface(true, dollHoverInfo), at.Add(townDollPixels[0])); ok || got != nil {
		t.Errorf("statistics card shows doll information %q", got)
	}
	if got, ok := TownHeroHoverLines(townHeroSurface(false, [12][]string{}), at.Add(townDollPixels[0])); ok || got != nil {
		t.Errorf("slot without information shows %q", got)
	}
}

func TestTooltipTownHeroDollShowsWornItemInformation(t *testing.T) {
	s := &fakeSchoolAnimator{}
	s.view = townHeroSurface(false, dollHoverInfo)
	a := newTestApp(t, nil, nil)
	a.Layout(640, 480)
	a.SetTown(s)
	a.flow.showTown("")
	a.tooltipPoint = shopFigureRect.Min.Add(townDollPixels[0])
	if got := a.tooltipTarget(); !reflect.DeepEqual(got.lines, dollHoverInfo[2]) || got.key() == "" {
		t.Fatalf("school hero doll hint = %q, want %q", got.lines, dollHoverInfo[2])
	}
	s.view = townHeroSurface(false, [12][]string{})
	if got := a.tooltipTarget(); got.key() != "" {
		t.Fatalf("a doll without item information still hints %q", got.lines)
	}
}
