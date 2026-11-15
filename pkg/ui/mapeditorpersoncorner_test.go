package ui

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"againrom/internal/witness"
	"againrom/pkg/words"
)

// cornerHost is the card host with the unit's facing pad and a figure of a
// person's size.
type cornerHost struct {
	*cardHost
	choices []InspectionChoice
	pad     []image.Point
}

func (h *cornerHost) facingField(value int64) InspectionField {
	return InspectionField{Key: "facing", Label: "Facing", Section: "identity", Value: value, Choices: h.choices, ChoicePad: h.pad}
}

func (h *cornerHost) Fields(kind RecordKind, index int) []InspectionField {
	fields := h.cardHost.Fields(kind, index)
	if kind == KindUnit {
		fields = append(fields, h.facingField(0))
	}
	return fields
}

// UnitSettings hands the facing back as the draft set it, as the game's
// host does.
func (h *cornerHost) UnitSettings(index int, values []InspectionFieldValue) (MapEntity, []InspectionField, *InspectionUnitPreview, error) {
	entity, fields, _, err := h.cardHost.UnitSettings(index, values)
	facing := int64(0)
	for _, v := range values {
		if v.Key == "facing" {
			facing = v.Value
		}
	}
	return entity, append(fields, h.facingField(facing)), &InspectionUnitPreview{Doll: image.NewRGBA(image.Rect(0, 0, 64, 112))}, err
}

func cornerEditor(t *testing.T, lang string) *MapEditor {
	t.Helper()
	e := wideEditor(t, lang, 0, "plain", map[string]int64{"Unit/identity": 300})
	h := &cornerHost{cardHost: &cardHost{wideHost: e.doc.Edits.(*wideHost)}, pad: []image.Point{{0, 0}, {0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}}}
	w := words.For(words.Language(lang))
	for n := range 9 {
		key := fmt.Sprintf("editor.unit.facing.%d", n)
		if n == 0 {
			key = "editor.unit.facing.random"
		}
		h.choices = append(h.choices, InspectionChoice{Value: int64(n), Label: w.Text(key)})
	}
	e.open = func(string) (*InspectionDocument, error) {
		d := wideDocument(h.wideHost)
		d.Edits = h
		return d, nil
	}
	if err := e.Open("views"); err != nil {
		t.Fatal(err)
	}
	return e
}

// The plain view stands the facing pad in its lower left corner beside the
// card: the pad's left edge on the room beside the card, its bottom on the
// card's, clear of the card and of the figure above it, at 640x480 where
// the band scrolls and at 1920x1080 where it stands; each of its nine
// buttons sets the facing.
func TestEditorPlainViewFacingPadInTheLowerLeftCorner(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, size := range []image.Point{{640, 480}, {1920, 1080}} {
			t.Run(fmt.Sprintf("%s/%v", lang, size), func(t *testing.T) {
				e := cornerEditor(t, lang)
				e.props.plain = true
				e.Layout(size.X, size.Y)
				e.Select(viewsUnit, false)
				e.objectSpec()
				if !e.PlainView() || !e.nativeUnitWindow() {
					t.Fatalf("no plain person window: plain %v person %v", e.PlainView(), e.nativeUnitWindow())
				}
				facing := cardFieldIndex(t, e, "facing")
				pad := func() (popupSpec, popupLayout, popupRowRect) {
					e.ScrollDetails(-1000)
					for range 100 {
						s, _, _ := e.objectSpec()
						l := layoutPopup(s, e.font, e.bodyFont())
						for _, rr := range l.Rows {
							if rr.Index == e.fieldRow(facing) {
								return s, l, rr
							}
						}
						e.ScrollDetails(1)
					}
					t.Fatal("the facing pad never shows")
					return popupSpec{}, popupLayout{}, popupRowRect{}
				}
				s, l, rr := pad()
				if dir := os.Getenv("AGAINROM_PERSON_FRAMES"); dir != "" {
					witness.PNG(t, filepath.Join(dir, fmt.Sprintf("person-corner-%s-%dx%d.png", lang, size.X, size.Y)), e.ChromeImage())
				}
				row := s.Rows[rr.Index]
				if !row.placed() || !row.RestLeft || row.Kind != popupDirection || len(rr.Buttons) != 9 {
					t.Fatalf("facing is no pad in the corner: placed %v kind %d buttons %d", row.placed(), row.Kind, len(rr.Buttons))
				}
				if rr.Rect.Min.X != l.CardRest.Min.X || rr.Rect.Max.Y != l.Card.Max.Y || rr.Rect.Max.X > l.Card.Min.X-popupPad || !rr.Rect.In(l.Rect) {
					t.Fatalf("pad %v is not in the lower left corner: rest %v card %v window %v", rr.Rect, l.CardRest, l.Card, l.Rect)
				}
				fig := e.unitFigureSize()
				figure := image.Rect(l.CardRest.Max.X-fig.X, l.CardRest.Min.Y, l.CardRest.Max.X, l.CardRest.Min.Y+fig.Y)
				if fig.Y <= 0 || rr.Rect.Overlaps(figure) {
					t.Fatalf("pad %v meets the figure %v", rr.Rect, figure)
				}
				for _, b := range rr.Buttons {
					if !b.In(rr.Control) {
						t.Fatalf("button %v leaves the pad %v", b, rr.Control)
					}
				}
				for n := range 9 {
					_, l, rr = pad()
					if part, _ := l.hit(center(rr.Buttons[n])); part != popupRowButton {
						t.Fatalf("button %d at %v is no row button: part %d", n, rr.Buttons[n], part)
					}
					modesClick(e, center(rr.Buttons[n]))
					facing = cardFieldIndex(t, e, "facing")
					if e.obj.buf[facing] != strconv.Itoa(n) {
						t.Fatalf("button %d chose %q", n, e.obj.buf[facing])
					}
				}
			})
		}
	}
}
