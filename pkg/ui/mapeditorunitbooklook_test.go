package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/text"
)

// A link placed on the card draws as a push button that reports the
// pointer over it.
func TestPlacedLinkRowDrawsAsAHoverButton(t *testing.T) {
	f := panelFont()
	s := placedCardSpec()
	s.Rows = append(s.Rows, popupRow{Kind: popupLink, Label: "SPELLCASTER", Place: image.Rect(10, 80, 90, 96)})
	l := layoutPopup(s, f, f)
	var rect image.Rectangle
	for _, rr := range l.Rows {
		if s.Rows[rr.Index].Kind == popupLink {
			rect = rr.Rect
		}
	}
	if rect.Empty() {
		t.Fatal("the placed link has no rectangle")
	}
	hover := func(p image.Point, ok bool) (found, hot bool) {
		widgetRecord = func(c widgetCall) {
			if b, is := c.state.(pushButton); is && c.kind == widgetPushButton && c.rect == rect {
				found, hot = true, b.Hover
			}
		}
		defer func() { widgetRecord = nil }()
		drawPopup(image.NewRGBA(s.Bounds), f, f, nil, s, l, editorPopupInk, p, ok)
		return
	}
	if found, hot := hover(image.Point{}, false); !found || hot {
		t.Fatalf("at rest: button %v, hover %v", found, hot)
	}
	if found, hot := hover(rect.Min.Add(image.Pt(2, 2)), true); !found || !hot {
		t.Fatalf("pointer over: button %v, hover %v", found, hot)
	}
}

// A placed link draws only its button: the card's own caption under it is not
// drawn, while a placed link that stands over nothing leaves the caption.
func TestPlacedLinkRowLeavesNoCaptionUnderTheButton(t *testing.T) {
	f := panelFont()
	subject := cardColumnsSubjects()["fighter"]
	subject.DetailLevel, subject.DetailSet = 7, true
	subject.OriginalPanel = OriginalPanelActor{Known: true, XPValue: 9}
	var at image.Rectangle
	for _, ln := range CharacterPanelReport(CompactPanelLayout(nil), f, subject) {
		if ln.Right && ln.RightField == PanelFieldSpellcaster {
			at = image.Rect(ln.At.X+ln.RightX, ln.At.Y, ln.At.X+ln.RightX+8, ln.At.Y+8)
		}
		if ln.Field == PanelFieldSpellcaster {
			at = image.Rect(ln.At.X+ln.Indent, ln.At.Y, ln.At.X+ln.Indent+8, ln.At.Y+8)
		}
	}
	if at.Empty() {
		t.Fatal("the card states no SPELLCASTER caption")
	}
	glyphs := func(over []PanelField) int {
		s := placedCardSpec()
		s.Card = &subject
		s.Rows = append(s.Rows, popupRow{Kind: popupLink, Label: "SPELLCASTER", Over: over, Place: image.Rectangle{Min: at.Min.Mul(2), Max: at.Max.Mul(2)}})
		l := layoutPopup(s, f, f)
		text.ResetCapture()
		text.SetCapture(false)
		defer text.StopCapture()
		drawPopup(image.NewRGBA(s.Bounds), f, f, nil, s, l, editorPopupInk, image.Point{}, false)
		return len(text.Captured())
	}
	bare, kept := glyphs([]PanelField{PanelFieldSpellcaster}), glyphs(nil)
	if bare >= kept {
		t.Fatalf("with the link over the caption %d glyphs drew, without %d: the caption still draws", bare, kept)
	}
}

// An off cell of the spell bar is drawn at half brightness, an on cell as is.
func TestOffBookCellDrawsDimmed(t *testing.T) {
	green := color.RGBA{0, 200, 0, 255}
	art := &BottomHUDArt{Book: solidPic(480, 85, green), UnknownSpell: solidPic(36, 36, color.RGBA{9, 9, 9, 255})}
	bar, cols, ok := bookBarRect(image.Pt(640, 480))
	if !ok {
		t.Fatal("no bar")
	}
	entries := []SpellEntry{{ID: 3, Dim: true}, {ID: 4}}
	cells := bookCellRects(bar, cols)
	img := composeOriginalSpellBar(art, entries, 0, bar, 0)
	at := func(slot int) color.RGBA {
		c := cells[slot].Sub(bar.Min).Min
		return img.RGBAAt(c.X+1, c.Y+1)
	}
	if got, want := at(0), (color.RGBA{0, 100, 0, 255}); got != want {
		t.Errorf("off cell drew %v, want %v", got, want)
	}
	if got := at(1); got != green {
		t.Errorf("on cell drew %v, want %v", got, green)
	}
}

// A unit's own spell records leave the map labels; a cell record keeps its.
func TestUnitBookRecordsDrawNoMapLabel(t *testing.T) {
	e := mapLabelEditor(t)
	e.view.hideAreas = true
	e.doc.Records = []InspectionRecord{
		{Kind: KindEnchantment, Spatial: true, Cell: image.Pt(20, 20), Marker: "4: Sacrifice", UnitBook: true},
		{Kind: KindEnchantment, Spatial: true, Cell: image.Pt(20, 20), Marker: "9: Acid", UnitBook: true},
		{Kind: KindEnchantment, Spatial: true, Cell: image.Pt(20, 20), Marker: "Trap spell"},
	}
	mapLabelAnchor(e, true, e.mapArea().Min.Add(image.Pt(180, 160)))
	labels := e.mapLabelsOf(labelRecord)
	if len(labels) != 1 || labels[0].text != "Trap spell" {
		t.Fatalf("record labels %+v, want only the trap's", labels)
	}
}
