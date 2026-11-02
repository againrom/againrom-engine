package ui

import (
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"os"
	"strings"
	"testing"
	"time"

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
		Icons:   map[int64]*image.RGBA{5: s.face}, IconTip: true}}
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
func (*speakerPortraitEdits) UnitSpeaker(int) (int64, bool) { return 0, false }

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

// The speaker list names each speaker in the host's words, "No speaker"
// first, and carries no small portrait. Resting the pointer on a speaker's
// row shows the editor's one tip: the name over the face at its full size; a
// row without a face shows none. The open lists and the tips are recorded in
// testdata/mapeditor_speaker_portraits.golden; MAPEDITOR_SPEAKERS_WRITE=1
// rewrites it.
func TestSpeakerListNamesEachSpeaker(t *testing.T) {
	var lines []string
	for _, lang := range []string{"en", "ru"} {
		e, face := speakerPortraitEditor(t, lang)
		if _, _, ok := e.objectSpec(); !ok {
			t.Fatal("the message editor builds no popup")
		}
		e.objectRowPress(popupRowControl, e.fieldRow(1))
		if e.obj.open != 1 || e.obj.choice == nil {
			t.Fatal("the speaker list did not open")
		}
		labels := e.doc.Edits.(*speakerPortraitEdits).labels
		rows := e.obj.choice.Rows()
		if len(rows) != len(labels) {
			t.Fatalf("%s: the list has %d rows", lang, len(rows))
		}
		for k, row := range rows {
			if row.Text != labels[k] || row.Icon != nil || row.IconCell {
				t.Errorf("%s row %d %q: picture %v, cell %v; want %q alone", lang, k, row.Text, row.Icon != nil, row.IconCell, labels[k])
			}
			lines = append(lines, fmt.Sprintf("%s row %d %s picture=%v cell=%v", lang, k, speakerLabels[lang][k], row.Icon != nil, row.IconCell))
		}
		d, ok := e.openDropdown()
		if !ok {
			t.Fatal("the open list has no drop-down")
		}
		list := d.listBox(e.bodyFont())
		if n := portraitPixels(e.ChromeImage(), list.Rect); n != 0 {
			t.Errorf("%s: the open list draws %d portrait pixels", lang, n)
		}
		top, _ := d.List.Visible()
		t0 := time.Unix(100, 0)
		rest := func(row int, at time.Duration) {
			p := center(list.Row(row - top))
			e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, t0.Add(at))
		}
		rest(1, 0)
		rest(1, editorTipDelay*time.Millisecond-time.Millisecond)
		if got := e.TipShown(); got != "" {
			t.Fatalf("%s: the tip %q shows before the delay", lang, got)
		}
		rest(1, editorTipDelay*time.Millisecond)
		if got := e.TipShown(); got != labels[1] || e.tip.picture != face {
			t.Fatalf("%s: the tip reads %q with picture %v, want %q with the face", lang, got, e.tip.picture != nil, labels[1])
		}
		pic := e.ChromeImage()
		tip, at, ok := tooltipPicture(tooltipTarget{kind: tooltipText, id: "editor", lines: []string{labels[1]}, font: e.bodyFont()}, e.pointer, pic.Bounds(), nil, e.editorTipStyle(face))
		if !ok {
			t.Fatal("the tip composes no box")
		}
		box := tip.Bounds().Add(at)
		n := portraitPixels(pic, box)
		if want := face.Bounds().Dx() * face.Bounds().Dy(); n != want {
			t.Errorf("%s: the tip draws %d face pixels, want the whole face's %d", lang, n, want)
		}
		h := fnv.New32a()
		h.Write(pic.Pix)
		lines = append(lines, fmt.Sprintf("%s tip %s box %v face pixels %d frame %08x", lang, speakerLabels[lang][1], box, n, h.Sum32()))
		for k, row := range []int{0, 2} {
			from := time.Duration(k+1) * time.Second
			rest(row, from)
			rest(row, from+editorTipDelay*time.Millisecond)
			if got := e.TipShown(); got != "" {
				t.Errorf("%s: row %d shows the tip %q", lang, row, got)
			}
		}
	}
	const path = "testdata/mapeditor_speaker_portraits.golden"
	got := strings.Join(lines, "\n") + "\n"
	if os.Getenv("MAPEDITOR_SPEAKERS_WRITE") != "" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.ReplaceAll(string(raw), "\r\n", "\n"); got != want {
		t.Errorf("the speaker list reads\n%s\nrecorded\n%s", got, want)
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
