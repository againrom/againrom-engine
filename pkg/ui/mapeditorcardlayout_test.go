package ui

import (
	"fmt"
	"image"
	"testing"

	"againrom/pkg/render/text"
)

func TestEditorCardInputsKeepClearOfLabelsAndEdges(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, plain := range []bool{true, false} {
			for _, overhang := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/plain=%v/overhang=%v", lang, plain, overhang), func(t *testing.T) {
					e, _ := cardEditor(t, lang, 300, true)
					e.objectSpec()
					motion := []InspectionField{{Key: "sight", Label: "Sight", Section: "stats", Value: 5}, {Key: "speed", Label: "Speed", Section: "stats", Value: 16}}
					for _, f := range motion {
						e.obj.fields = append(e.obj.fields, f)
						e.obj.buf = append(e.obj.buf, fieldInput(f))
					}
					e.obj.unit.fields = append(e.obj.unit.fields, motion...)
					e.obj.unit.entity.Char.Sight, e.obj.unit.entity.Speed = 5, 16
					e.obj.unit.key = fmt.Sprint(e.obj.buf)
					e.cardFont.Selector = e.font.Selector
					if overhang {
						for i := range e.cardFont.Glyphs {
							g := &e.cardFont.Glyphs[i]
							g.Width = g.Advance + e.cardFont.Spacing + 3
							g.Pixels = make([]text.Pixel, g.Width*g.Height)
							g.Pixels[g.Width-1] = text.Pixel{Level: text.MaxLevel, Painted: true}
						}
					}
					if !plain {
						r, ok := e.ViewSwitchRect()
						viewsClick(t, e, r, ok, "detailed view")
					}
					s, _, ok := e.objectSpec()
					if !ok || s.Card == nil {
						t.Fatal("unit card missing")
					}
					l := layoutPopup(s, e.font, e.bodyFont())
					scale := max(1, s.CardScale)
					var labels, boxes []image.Rectangle
					y := map[PanelField]int{}
					label := func(at image.Point, value string) {
						w, h := s.CardFont.Measure(value)
						if w > 0 {
							labels = append(labels, image.Rectangle{Min: at.Mul(scale), Max: at.Add(image.Pt(w, h)).Mul(scale)}.Add(l.Card.Min))
						}
					}
					for _, r := range CharacterPanelReport(CompactPanelLayout(nil), s.CardFont, *s.Card) {
						label(r.At.Add(image.Pt(r.Indent, 0)), r.Label)
						y[r.Field] = r.At.Y
						if r.Right {
							label(r.At.Add(image.Pt(r.RightX, 0)), r.RightLabel)
							y[r.RightField] = r.At.Y
						}
					}
					if y[PanelFieldHealth] != y[PanelFieldReaction] || y[PanelFieldManaCard] != y[PanelFieldSpirit] ||
						y[PanelFieldHealth]-y[PanelFieldHealthHeading] != s.CardFont.Height()+1 ||
						y[PanelFieldManaCard]-y[PanelFieldManaCardHeading] != s.CardFont.Height()+1 {
						t.Fatal("health/mana no longer follow the game's heading/value rows")
					}
					for _, rr := range l.Rows {
						if rr.Control.Empty() {
							continue
						}
						if !s.Rows[rr.Index].placed() {
							if rr.Control.Overlaps(l.Card) {
								t.Error("ordinary input overlaps the card", rr.Control, l.Card)
							}
							continue
						}
						box := rr.Control
						if !box.In(l.Card.Inset(1)) {
							t.Error("input touches/leaves the card edge", box, l.Card)
						}
						for _, label := range labels {
							if box.Inset(-1).Overlaps(label) {
								t.Error("input touches/overlaps a drawn label", box, label)
							}
						}
						for _, other := range boxes {
							if box.Inset(-1).Overlaps(other) {
								t.Error("inputs touch/overlap", box, other)
							}
						}
						boxes = append(boxes, box)
					}
					if plain && len(boxes) != len(cardKeys)+len(motion) || !plain && len(boxes) != 0 {
						t.Fatal("unexpected card input count", len(boxes))
					}
				})
			}
		}
	}
}
