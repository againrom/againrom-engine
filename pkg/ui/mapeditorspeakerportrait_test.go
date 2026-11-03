package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/words"
)

// speakerPortraitInk is a colour no other part of the editor's frame uses, so
// a pixel of it is the portrait.
var speakerPortraitInk = color.RGBA{R: 0xff, B: 0xff, A: 0xff}

// speakerPortraitEdits is a message page of two cards: speaker 5, who has a
// face, and speaker 6, who has none.
type speakerPortraitEdits struct {
	*editorTextEdits
	face   *image.RGBA
	labels [3]string
}

// speakerLabels are the choice labels the host gives in each language: "No
// speaker", then speakers 5 and 6, as the host's "id: name".
var speakerLabels = map[string][3]string{
	"en": {"No speaker", "5: Anna", "6: Bors"},
	"ru": {"Без говорящего", "5: Анна", "6: Борис"},
}

func (*speakerPortraitEdits) MessageParts(uint32) []InspectionMessagePart {
	return []InspectionMessagePart{{Source: 0, Number: 1, Body: "First", Speaker: 5}, {Source: 1, Number: 2, Body: "Second", Speaker: 6}}
}
func (s *speakerPortraitEdits) MessageFields(uint32, ...int) []InspectionField {
	return []InspectionField{{Key: "body", Text: true}, {Key: "speaker", Value: -1,
		Choices: []InspectionChoice{{Value: -1, Label: s.labels[0]}, {Value: 5, Label: s.labels[1]}, {Value: 6, Label: s.labels[2]}},
		Icons:   map[int64]*image.RGBA{5: s.face}, IconTip: true, IconNotes: map[int64]string{5: "The face depends on the player's hero"}}}
}
func (*speakerPortraitEdits) SetMessage(uint32, string, int64) (*InspectionDocument, error) {
	return nil, nil
}
func (*speakerPortraitEdits) SetMessageParts(uint32, []InspectionMessagePart) (*InspectionDocument, error) {
	return nil, nil
}
func (*speakerPortraitEdits) CreateMessage(InspectionMessageOwner, string, int64) (*InspectionDocument, uint32, error) {
	return nil, 1, nil
}
func (*speakerPortraitEdits) CreateMessageParts(InspectionMessageOwner, []InspectionMessagePart) (*InspectionDocument, uint32, error) {
	return nil, 1, nil
}
func (*speakerPortraitEdits) PointMessageParts(InspectionMessageOwner, uint32, []InspectionMessagePart) (*InspectionDocument, error) {
	return nil, nil
}
func (*speakerPortraitEdits) UnitSpeaker(int) (int64, SpeakerMiss) {
	return 0, SpeakerMiss{Cause: SpeakerNoFit}
}

// speakerPortraitEditor opens the two-card page in lang at 1920x1080, the
// labels in the font's encoding as the host gives them; the face is a
// picture of speakerPortraitInk the dialogue pane's size.
func speakerPortraitEditor(t *testing.T, lang string) (*MapEditor, *image.RGBA) {
	t.Helper()
	font := wordsFont(lang)
	e, edits := selectedTextEditor(t, font, image.Pt(1920, 1080))
	e.SetWords(words.For(lang))
	face := image.NewRGBA(image.Rect(0, 0, 96, 96))
	for i := 0; i < len(face.Pix); i += 4 {
		copy(face.Pix[i:i+4], []byte{speakerPortraitInk.R, speakerPortraitInk.G, speakerPortraitInk.B, speakerPortraitInk.A})
	}
	s := &speakerPortraitEdits{editorTextEdits: edits, face: face}
	for k, label := range speakerLabels[lang] {
		s.labels[k], _ = textinput.EncodeText(label, font.Selector, textinput.Substitute)
	}
	e.doc.Edits = s
	e.openMessage(1)
	return e, face
}

// portraitPixels counts the frame's pixels of the portrait's colour inside r.
func portraitPixels(pic *image.RGBA, r image.Rectangle) int {
	n := 0
	r = r.Intersect(pic.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if pic.RGBAAt(x, y) == speakerPortraitInk {
				n++
			}
		}
	}
	return n
}

// A page card shows its chosen speaker's portrait in the empty column under
// its number, beside its text and not in the speaker field; a card whose
// speaker has no face shows none.
func TestMessageCardShowsTheChosenSpeakersPortrait(t *testing.T) {
	e, face := speakerPortraitEditor(t, "en")
	s, _, ok := e.messageSpec()
	if !ok {
		t.Fatal("the message editor builds no popup")
	}
	first, second := s.Rows[e.fieldRow(0)], s.Rows[e.fieldRow(2)]
	if first.Portrait != face {
		t.Fatalf("speaker 5's text row carries %v, want its portrait", first.Portrait)
	}
	if second.Portrait != nil {
		t.Fatalf("speaker 6's text row carries %v, want no portrait", second.Portrait)
	}
	for _, head := range []int{e.fieldRow(1), e.fieldRow(3)} {
		if len(s.Rows[head].Icons) != 0 {
			t.Fatalf("the speaker field %d carries the small face %v", head, s.Rows[head].Icons)
		}
	}
	pic := e.ChromeImage()
	l := layoutPopup(s, e.font, e.bodyFont())
	column := func(rr popupRowRect) image.Rectangle {
		return image.Rect(rr.Rect.Min.X, rr.Control.Min.Y, rr.Control.Min.X, rr.Control.Max.Y)
	}
	for _, rr := range l.Rows {
		switch rr.Index {
		case e.fieldRow(0):
			n := portraitPixels(pic, column(rr))
			if n == 0 {
				t.Error("speaker 5's card draws no portrait under its number")
			}
			if all := portraitPixels(pic, rr.Rect); all != n {
				t.Errorf("speaker 5's portrait leaves its column: %d of %d pixels outside", all-n, all)
			}
		case e.fieldRow(1):
			if n := portraitPixels(pic, rr.Rect); n != 0 {
				t.Errorf("speaker 5's speaker field draws %d portrait pixels", n)
			}
		case e.fieldRow(2):
			if n := portraitPixels(pic, rr.Rect); n != 0 {
				t.Errorf("speaker 6's card draws %d portrait pixels", n)
			}
		}
	}
}

// The portrait is the dialogue pane's cut: the default window of a picture,
// centred on a square, outside it transparent.
func TestSpeakerPortraitIsThePanesCut(t *testing.T) {
	face := image.NewRGBA(image.Rect(0, 0, 160, 240))
	for y := 0; y < 240; y++ {
		for x := 0; x < 160; x++ {
			face.SetRGBA(x, y, color.RGBA{R: uint8(x), G: uint8(y), A: 0xff})
		}
	}
	p := SpeakerPortrait(face, image.Rectangle{})
	if p.Bounds() != image.Rect(0, 0, noticeFaceH, noticeFaceH) {
		t.Fatal("portrait bounds", p.Bounds())
	}
	left := (noticeFaceH - noticeFaceW) / 2
	if got, want := p.RGBAAt(left, 0), (color.RGBA{R: noticeFaceDefX, G: noticeFaceDefY, A: 0xff}); got != want {
		t.Errorf("top-left of the cut %v, want the default window's corner %v", got, want)
	}
	if got := p.RGBAAt(left-1, 10); got.A != 0 {
		t.Errorf("left of the cut %v, want transparent", got)
	}
	if got := p.RGBAAt(left, noticeFaceDefH); got.A != 0 {
		t.Errorf("below the default window %v, want transparent", got)
	}
	w := SpeakerPortrait(face, NoticeFaceWindow(40, 10))
	if got, want := w.RGBAAt(left, 0), (color.RGBA{R: 40, G: 10, A: 0xff}); got != want {
		t.Errorf("a stated window's corner %v, want %v", got, want)
	}
	if SpeakerPortrait(nil, image.Rectangle{}) != nil {
		t.Error("no face made a portrait")
	}
}
