package ui

import (
	"fmt"
	"image"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/text"
	"againrom/pkg/words"
)

// namedHost is cardHost giving the unit the name field and the equipment
// group word the game's host gives.
type namedHost struct {
	*cardHost
	lang string
	name InspectionField
	// texts are the text values the host was asked to write, by key.
	texts map[string]string
}

func (h *namedHost) SetFields(kind RecordKind, index int, values []InspectionFieldValue) (*InspectionDocument, error) {
	for _, v := range values {
		if v.Text {
			h.texts[v.Key] = v.TextValue
		}
	}
	return h.cardHost.SetFields(kind, index, values)
}

func (h *namedHost) Fields(kind RecordKind, index int) []InspectionField {
	out := h.cardHost.Fields(kind, index)
	if kind != KindUnit {
		return out
	}
	for i := range out {
		if out[i].Group == "Equipment" {
			out[i].Group = words.For(h.lang).Text("editor.unit.group.equipment")
		}
	}
	return append(out, h.name)
}

var namedField = InspectionField{Key: "name", Label: "Name", Section: "identity", Text: true, TextValue: "Gildarius"}

func (h *namedHost) UnitSettings(index int, values []InspectionFieldValue) (MapEntity, []InspectionField, *InspectionUnitPreview, error) {
	entity, fields, preview, err := h.cardHost.UnitSettings(index, values)
	if h.name.TextValue == "" {
		entity.Name = ""
	}
	return entity, append(fields, h.name), preview, err
}

// unitWindowEditor is cardEditor's unit of class, a person (300) or a
// creature (301), over namedHost, its window in the normal mode when plain.
func unitWindowEditor(t *testing.T, lang string, class int64, plain bool) *MapEditor {
	t.Helper()
	e, _ := namedUnitEditor(t, lang, class, plain, namedField)
	return e
}

// namedUnitEditor is unitWindowEditor with the unit's name field as given.
func namedUnitEditor(t *testing.T, lang string, class int64, plain bool, name InspectionField) (*MapEditor, *namedHost) {
	t.Helper()
	e, h := cardEditor(t, lang, class, false)
	named := &namedHost{cardHost: h, lang: lang, name: name, texts: map[string]string{}}
	e.open = func(string) (*InspectionDocument, error) {
		d := wideDocument(h.wideHost)
		d.Edits = named
		return d, nil
	}
	if err := e.Open("views"); err != nil {
		t.Fatal(err)
	}
	e.props.plain = plain
	e.Layout(1920, 1080)
	e.Select(viewsUnit, false)
	return e, named
}

// unitWindowCases are both languages, a person and a creature, both modes.
func unitWindowCases(t *testing.T, run func(t *testing.T, e *MapEditor, lang string, plain bool)) {
	for _, lang := range []string{"en", "ru"} {
		for _, class := range []int64{300, 301} {
			for _, plain := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/class=%d/plain=%v", lang, class, plain), func(t *testing.T) {
					run(t, unitWindowEditor(t, lang, class, plain), lang, plain)
				})
			}
		}
	}
}

// The unit window's card stands under no heading; the name is edited in an
// input on the card, and no row of its own edits it; the equipment heading
// stands once in the detailed mode and not in the normal one, and no
// heading repeats.
func TestEditorUnitWindowCardEditsTheNameUnderNoHeading(t *testing.T) {
	unitWindowCases(t, func(t *testing.T, e *MapEditor, lang string, plain bool) {
		s, _, ok := e.objectSpec()
		if !ok || s.Card == nil {
			t.Fatal("no unit window with a card")
		}
		stats := map[string]string{"en": "Stats", "ru": "Параметры"}[lang]
		equipment, headings := 0, map[string]int{}
		for _, row := range s.Rows {
			if row.Kind != popupHeading {
				continue
			}
			if row.Label == e.editorHostText(stats) {
				t.Errorf("the window heads its card %q", stats)
			}
			if row.Label == e.word("editor.person.equipment") {
				equipment++
			}
			if headings[row.Label]++; headings[row.Label] == 2 {
				t.Errorf("heading %q stands twice", row.Label)
			}
		}
		if want := map[bool]int{false: 1, true: 0}[plain]; equipment != want {
			t.Errorf("%d equipment headings, want %d", equipment, want)
		}
		name := slices.IndexFunc(e.obj.fields, func(f InspectionField) bool { return f.Key == "name" })
		rows := 0
		for row, i := range e.obj.rowField {
			if i != name {
				continue
			}
			rows++
			if !s.Rows[row].placed() {
				t.Errorf("row %d %q edits the name off the card", row, s.Rows[row].Label)
			}
		}
		if name < 0 || rows != 1 {
			t.Errorf("%d rows edit the name, want its one input on the card", rows)
		}
	})
}

// A unit with no name has the one name input on its card in both modes, no
// row under the card edits the name, and a name typed there is applied.
func TestEditorUnitWindowEditsAnEmptyNameOnTheCard(t *testing.T) {
	empty := namedField
	empty.TextValue = ""
	for _, lang := range []string{"en", "ru"} {
		for _, plain := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/plain=%v", lang, plain), func(t *testing.T) {
				e, h := namedUnitEditor(t, lang, 300, plain, empty)
				s, _, ok := e.objectSpec()
				if !ok || s.Card == nil {
					t.Fatal("no unit window with a card")
				}
				name := slices.IndexFunc(e.obj.fields, func(f InspectionField) bool { return f.Key == "name" })
				rows, at := 0, -1
				for row, i := range e.obj.rowField {
					if i == name {
						rows, at = rows+1, row
					}
				}
				if rows != 1 || !s.Rows[at].placed() {
					t.Fatalf("%d rows edit the name, want its one input on the card", rows)
				}
				r, ok := e.FindControl(ControlQuery{Role: ControlPopupRow, N: at})
				viewsClick(t, e, r, ok, "the card's name input")
				e.TextInput("Ann")
				e.applyObject()
				if got, ok := h.texts["name"]; !ok || got != "Ann" {
					t.Errorf("the host was asked to write the name %q (%v), want %q", got, ok, "Ann")
				}
			})
		}
	}
}

// At a window scale other than 1 the unit card's text reaches the window as
// glyphs the one text overlay draws at device resolution, scaled with the
// card: the overlay holds the card's glyphs at the card's scale, and the
// frame the window scaler resamples keeps no pixel of them. Both modes, both
// languages.
func TestEditorUnitCardTextIsDrawnByTheOverlayScaled(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, plain := range []bool{false, true} {
			e := unitWindowEditor(t, lang, 300, plain)
			e.textSmoothingEnabled = true
			e.deviceScale = 1.5
			e.drawPoints(ebiten.NewImage(1920, 1080))
			s, _, ok := e.objectSpec()
			if !ok || s.Card == nil || s.CardScale < 2 {
				t.Fatalf("%s plain %v: no scaled card", lang, plain)
			}
			card := layoutPopup(s, e.font, e.bodyFont()).Card
			glyphs := 0
			for _, c := range e.windowText {
				if _, k := shownCell(c, 0); image.Pt(c.X, c.Y).In(card) && k == s.CardScale {
					glyphs++
				}
			}
			if glyphs == 0 {
				t.Errorf("%s plain %v: the overlay draws no glyph of the card %v at its scale %d (%d calls)", lang, plain, card, s.CardScale, len(e.windowText))
			}
			// A card glyph cell whose every pixel still shows its ink in the
			// frame the scaler gets is baked; a cell an input covers is not.
			_, blit, calls, _ := e.chromePicture()
			baked := 0
			for _, c := range calls {
				if !image.Pt(c.X, c.Y).In(card) {
					continue
				}
				for n, p := range c.Glyph.Pixels {
					at, k := shownCell(c, n)
					ink := c.NativeColor(p.Level, n)
					shows := p.Painted && c.Under[n] != ink
					for i := range k * k {
						shows = shows && blit.RGBAAt(at.X+i%k, at.Y+i/k) == ink
					}
					if shows {
						baked++
					}
				}
			}
			if baked != 0 {
				t.Errorf("%s plain %v: %d glyph pixels of the card stay in the frame the window scaler resamples", lang, plain, baked)
			}
		}
	}
}

// shownCell is where cell n of c's glyph lands on the frame, and the side of
// the square it paints there.
func shownCell(c text.DrawCall, n int) (image.Point, int) {
	if c.Glyph == nil || c.Glyph.Width <= 0 {
		return image.Pt(c.X, c.Y), 0
	}
	w := c.Glyph.Width
	k := glyphBounds([]text.DrawCall{c}).Dx() / w
	return image.Pt(c.X+n%w*k, c.Y+n/w*k), k
}

// The normal mode is the detailed mode reduced: its rows are the detailed
// mode's rows, in the same order with the same words and values, less the
// equipment, the bag, the face and the sex, and the headings over them.
func TestEditorUnitWindowNormalModeIsTheDetailedModeReduced(t *testing.T) {
	type shown struct {
		kind         popupRowKind
		label, value string
		hold         popupHold
		key          string
	}
	rows := func(e *MapEditor, plain bool) []shown {
		e.props.plain = plain
		s, _, ok := e.objectSpec()
		if !ok {
			t.Fatal("no unit window")
		}
		var out []shown
		for i, row := range s.Rows {
			r := shown{kind: row.Kind, label: row.Label, value: row.Value, hold: row.Hold}
			if f, ok := e.obj.rowField[i]; ok {
				r.key = e.obj.fields[f].Key
			}
			out = append(out, r)
		}
		return out
	}
	for _, lang := range []string{"en", "ru"} {
		for _, class := range []int64{300, 301} {
			e := unitWindowEditor(t, lang, class, false)
			detailed, normal := rows(e, false), rows(e, true)
			name := fmt.Sprintf("%s class %d", lang, class)
			// left checks that the normal mode may leave out detailed row d.
			left := func(d shown) {
				section := ""
				if f := slices.IndexFunc(e.obj.fields, func(f InspectionField) bool { return f.Key == d.key }); f >= 0 {
					section = e.obj.fields[f].Section
				}
				if d.key != "" && d.key != "face" && d.key != "sex" && section != "equipment" && section != "inventory" {
					t.Errorf("%s: the normal mode leaves out the detailed row %q (%s)", name, d.label, d.key)
				}
			}
			at := 0
			for _, r := range normal {
				for at < len(detailed) && detailed[at] != r {
					left(detailed[at])
					at++
				}
				if at == len(detailed) {
					t.Fatalf("%s: the normal mode's row %q %q is not the detailed mode's, in its order", name, r.label, r.value)
				}
				at++
			}
			for _, d := range detailed[at:] {
				left(d)
			}
		}
	}
}
