package ui

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

// previewProbe is a painting host that also pictures its objects: the picture
// of code n is one flat colour that n names.
type previewProbe struct {
	brushProbe
	asked []string
}

func (p *previewProbe) BrushObjects() []BrushObject {
	return []BrushObject{{Code: 0, Name: "No object"}, {Code: 3, Name: "Oak"}, {Code: 5, Name: "Stone"}}
}
func (p *previewProbe) NewRecordStep(InspectionNewRecord, string) InspectionNewStep {
	return InspectionNewStep{}
}
func (p *previewProbe) CheckNewRecord(InspectionNewRecord, string) error { return nil }
func (p *previewProbe) AddRecord(InspectionNewRecord) (*InspectionDocument, error) {
	return nil, nil
}
func (p *previewProbe) NewRecordPreview(kind, key string, value int64) *image.RGBA {
	p.asked = append(p.asked, key)
	if key != "object" || value <= 0 {
		return nil
	}
	pic := image.NewRGBA(image.Rect(0, 0, 40, 50))
	draw.Draw(pic, pic.Bounds(), image.NewUniform(color.RGBA{uint8(value * 40), 90, 200, 255}), image.Point{}, draw.Src)
	return pic
}

// The Objects brush shows the highlighted object's picture in the Aside box,
// a box wider than the Add window's, and the picture follows the highlight.
func TestMapEditorObjectsBrushShowsTheHighlightedObjectsPicture(t *testing.T) {
	e, p := brushEditor(t)
	probe := &previewProbe{brushProbe: *p}
	probe.doc = e.doc
	e.doc.Edits = probe
	if err := e.Key("objects"); err != nil {
		t.Fatal(err)
	}
	s, ok := e.listSpec()
	if !ok || s.ListAside == 0 {
		t.Fatalf("the Objects list keeps no picture column: %+v", s.ListAside)
	}
	box := layoutPopup(s, e.font, e.bodyFont()).Aside
	if box.Dx() <= previewW {
		t.Fatalf("the picture box is %d wide, not wider than the Add window's %d", box.Dx(), previewW)
	}
	centre := func() color.RGBA {
		pic := image.NewRGBA(image.Rect(0, 0, e.window.X, e.window.Y))
		s, _ := e.listSpec()
		e.drawBrushPreview(pic, s)
		return pic.RGBAAt((box.Min.X+box.Max.X)/2, (box.Min.Y+box.Max.Y)/2)
	}
	if got := centre(); got != (color.RGBA{}) {
		t.Fatalf("the clear row draws a picture %v", got)
	}
	_ = e.Key("down")
	first := centre()
	if first != (color.RGBA{120, 90, 200, 255}) {
		t.Fatalf("the Oak row draws %v", first)
	}
	_ = e.Key("down")
	if second := centre(); second != (color.RGBA{200, 90, 200, 255}) || second == first {
		t.Fatalf("the Stone row draws %v after Oak's %v", second, first)
	}
	e.Key("terrain")
	if s, _ := e.listSpec(); s.ListAside != 0 {
		t.Fatalf("the terrain list keeps a picture column of %d", s.ListAside)
	}
}
