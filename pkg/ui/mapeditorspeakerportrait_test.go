package ui

import (
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"os"
	"strings"
	"testing"
)

// speakerPortraitInk is a colour no other part of the editor's frame uses, so
// a pixel of it is the portrait.
var speakerPortraitInk = color.RGBA{R: 0xff, B: 0xff, A: 0xff}

// speakerPortraitEdits is a message page of two cards: speaker 5, who has a
// face, and speaker 6, who has none.
type speakerPortraitEdits struct {
	*editorTextEdits
	face *image.RGBA
}

func (*speakerPortraitEdits) MessageParts(uint32) []InspectionMessagePart {
	return []InspectionMessagePart{{Source: 0, Number: 1, Body: "First", Speaker: 5}, {Source: 1, Number: 2, Body: "Second", Speaker: 6}}
}
func (s *speakerPortraitEdits) MessageFields(uint32, ...int) []InspectionField {
	return []InspectionField{{Key: "body", Text: true}, {Key: "speaker", Value: -1,
		Choices: []InspectionChoice{{Value: -1, Label: "No speaker"}, {Value: 5, Label: "5: Anna"}, {Value: 6, Label: "6: Bors"}},
		Icons:   map[int64]*image.RGBA{5: s.face}}}
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

// speakerPortraitEditor opens the two-card page; the face is a picture of
// speakerPortraitInk the dialogue pane's size.
func speakerPortraitEditor(t *testing.T) (*MapEditor, *image.RGBA) {
	t.Helper()
	e, edits := selectedTextEditor(t, panelFont(), image.Pt(1280, 800))
	face := image.NewRGBA(image.Rect(0, 0, 96, 96))
	for i := 0; i < len(face.Pix); i += 4 {
		copy(face.Pix[i:i+4], []byte{speakerPortraitInk.R, speakerPortraitInk.G, speakerPortraitInk.B, speakerPortraitInk.A})
	}
	e.doc.Edits = &speakerPortraitEdits{editorTextEdits: edits, face: face}
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

// A page card shows its chosen speaker's portrait inside the speaker field,
// after the name; a card whose speaker has no face shows none.
func TestMessageCardShowsTheChosenSpeakersPortrait(t *testing.T) {
	e, face := speakerPortraitEditor(t)
	s, _, ok := e.messageSpec()
	if !ok {
		t.Fatal("the message editor builds no popup")
	}
	first, second := s.Rows[e.fieldRow(1)], s.Rows[e.fieldRow(3)]
	if len(first.Icons) != 1 || first.Icons[0] != face {
		t.Fatalf("speaker 5's card carries %v, want its portrait", first.Icons)
	}
	if len(second.Icons) != 0 {
		t.Fatalf("speaker 6's card carries %v, want no portrait", second.Icons)
	}
	pic := e.ChromeImage()
	l := layoutPopup(s, e.font, e.bodyFont())
	for _, rr := range l.Rows {
		switch rr.Index {
		case e.fieldRow(1):
			if portraitPixels(pic, rr.Control) == 0 {
				t.Error("speaker 5's card draws no portrait in its speaker field")
			}
		case e.fieldRow(3):
			if n := portraitPixels(pic, rr.Control); n != 0 {
				t.Errorf("speaker 6's card draws %d portrait pixels", n)
			}
		}
	}
}

// The speaker list's rows: each speaker with a face carries its portrait, and
// every other row keeps an empty portrait cell, "No speaker" first. The open
// list is recorded in testdata/mapeditor_speaker_portraits.golden;
// MAPEDITOR_SPEAKERS_WRITE=1 rewrites it.
func TestSpeakerListDrawsEachSpeakersPortrait(t *testing.T) {
	e, face := speakerPortraitEditor(t)
	if _, _, ok := e.objectSpec(); !ok {
		t.Fatal("the message editor builds no popup")
	}
	e.objectRowPress(popupRowControl, e.fieldRow(1))
	if e.obj.open != 1 || e.obj.choice == nil {
		t.Fatal("the speaker list did not open")
	}
	var lines []string
	for k, row := range e.obj.choice.Rows() {
		if want := k == 1; (row.Icon == face) != want || !want && row.Icon != nil || !row.IconCell {
			t.Errorf("row %d %q: picture %v, cell %v", k, row.Text, row.Icon != nil, row.IconCell)
		}
		lines = append(lines, fmt.Sprintf("row %d %s picture=%v cell=%v", k, row.Text, row.Icon != nil, row.IconCell))
	}
	if rows := e.obj.choice.Rows(); len(rows) == 0 || rows[0].Text != "No speaker" {
		t.Fatal("No speaker is not the first row")
	}
	pic := e.ChromeImage()
	d, ok := e.openDropdown()
	if !ok {
		t.Fatal("the open list has no drop-down")
	}
	list := image.Rect(d.Rect.Min.X, d.Rect.Max.Y, d.Rect.Max.X, d.Rect.Max.Y+d.Rows*d.Rect.Dy()+4)
	n := portraitPixels(pic, list)
	if n == 0 {
		t.Error("the open list draws no portrait")
	}
	h := fnv.New32a()
	h.Write(pic.Pix)
	lines = append(lines, fmt.Sprintf("list portrait pixels %d", n), fmt.Sprintf("frame %08x", h.Sum32()))
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
