package ui

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/witness"
)

func TestUnitWindowViewSwitchKeepsFirstColumn(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, class := range []int64{300, 301} {
			for _, startPlain := range []bool{false, true} {
				e := unitWindowEditor(t, lang, class, startPlain)
				var widths []int
				var corners []image.Point
				for _, plain := range []bool{startPlain, !startPlain} {
					if e.props.plain != plain {
						e.switchView()
					}
					s, _, _ := e.objectSpec()
					l := layoutPopup(s, e.font, e.bodyFont())
					corners = append(corners, l.Rect.Min)
					for _, row := range l.Rows {
						if s.Rows[row.Index].Column == 0 && !s.Rows[row.Index].placed() {
							widths = append(widths, row.Rect.Dx())
							break
						}
					}
					if dir := os.Getenv("AGAINROM_UNIT_REMARK_FRAMES"); dir != "" {
						if err := os.MkdirAll(dir, 0o755); err != nil {
							t.Fatal(err)
						}
						pic := e.ChromeImage()
						witness.PNG(t, filepath.Join(dir, fmt.Sprintf("unit-%s-%d-plain-%v.png", lang, class, plain)), pic)
					}
				}
				if len(widths) != 2 || widths[0] != widths[1] {
					t.Errorf("%s class %d first column widths %v", lang, class, widths)
				}
				if corners[0] != corners[1] {
					t.Errorf("%s class %d window corners %v", lang, class, corners)
				}
				e.switchView()
				s, _, _ := e.objectSpec()
				if corner := layoutPopup(s, e.font, e.bodyFont()).Rect.Min; corner != corners[0] {
					t.Errorf("%s class %d return corner %v, want %v", lang, class, corner, corners[0])
				}
			}
		}
	}
}

type unitDraftPlayHost struct {
	*namedHost
	played string
}

func (h *unitDraftPlayHost) SetFields(kind RecordKind, index int, values []InspectionFieldValue) (*InspectionDocument, error) {
	doc, err := h.namedHost.SetFields(kind, index, values)
	if doc != nil {
		doc.Edits = h
	}
	for _, v := range values {
		if v.Key == "name" {
			h.name.TextValue = v.TextValue
		}
	}
	return doc, err
}

func (h *unitDraftPlayHost) TestPlay() (string, error) {
	h.played = h.texts["name"]
	return "played", nil
}

func TestUnitCardDraftNameReachesTestPlay(t *testing.T) {
	e, h := namedUnitEditor(t, "en", 300, true, namedField)
	host := &unitDraftPlayHost{namedHost: h}
	e.doc.Edits = host
	e.ObjectFields()
	for i, f := range e.obj.fields {
		if f.Key == "name" {
			r, ok := e.FindControl(ControlQuery{Role: ControlField, N: i})
			viewsClick(t, e, r, ok, "name")
			_ = e.Key("selectall")
			e.TextInput("Edited name")
		}
	}
	if err := e.Key("testplay"); err != nil {
		t.Fatal(err)
	}
	if host.played != "Edited name" {
		t.Fatalf("F5 played name %q; want %q", host.played, "Edited name")
	}
}

func TestUnitWindowPortraitStandsBelowDoll(t *testing.T) {
	for _, class := range []int64{300, 301} {
		for _, plain := range []bool{false, true} {
			e := unitWindowEditor(t, "en", class, plain)
			e.objectSpec()
			portrait := image.NewRGBA(image.Rect(0, 0, 72, 96))
			f := InspectionField{Key: "speaker", Section: "doll", Picture: portrait, ReadOnly: true}
			e.obj.fields = append(e.obj.fields, f)
			e.obj.buf = append(e.obj.buf, "0")
			e.obj.unit.key = fmt.Sprint(e.obj.buf)
			e.obj.unit.fields = append(e.obj.unit.fields, f)
			s, _, _ := e.objectSpec()
			l := layoutPopup(s, e.font, e.bodyFont())
			found, figureBottom := false, 0
			for _, rr := range l.Rows {
				row := s.Rows[rr.Index]
				if row.Hold == holdFigure {
					figureBottom = rr.Rect.Max.Y
				}
				if row.Portrait == portrait {
					found = true
					if rr.Rect.Min.Y < figureBottom || row.Column != unitColumn("doll", e.unitSectionsShown(e.unitModeShown(), s.Card), s.Columns) {
						t.Errorf("class %d plain %v portrait %v before doll bottom %d", class, plain, rr.Rect, figureBottom)
					}
				}
			}
			if !found {
				t.Errorf("class %d plain %v dialogue portrait is absent", class, plain)
			}
		}
	}
}
