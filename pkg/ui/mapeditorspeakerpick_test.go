package ui

import (
	"image"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/words"
)

// The cells of the speaker pick map: a unit record 5 shows, a unit no record
// shows, a unit record 7 shows, and open ground.
var (
	cellShownUnit   = image.Pt(10, 10)
	cellUnshownUnit = image.Pt(14, 10)
	cellOtherUnit   = image.Pt(18, 10)
	cellOpenGround  = image.Pt(40, 8)
)

// speakerPickEdits is a message page of two cards with no speaker; the
// speaker list offers no speaker, 5 and 7, and units 0 and 2 show speakers
// 5 and 7. writes records the parts each Apply writes.
type speakerPickEdits struct {
	*editorTextEdits
	writes [][]InspectionMessagePart
}

func (*speakerPickEdits) MessageParts(uint32) []InspectionMessagePart {
	return []InspectionMessagePart{{Source: 0, Number: 1, Body: "First", Speaker: -1}, {Source: 1, Number: 2, Body: "Second", Speaker: -1}}
}
func (*speakerPickEdits) MessageFields(uint32, ...int) []InspectionField {
	return []InspectionField{{Key: "body", Text: true}, {Key: "speaker", Value: -1,
		Choices: []InspectionChoice{{Value: -1, Label: "-"}, {Value: 5, Label: "5: Anna"}, {Value: 7, Label: "7: Wolf"}}}}
}
func (*speakerPickEdits) SetMessage(uint32, string, int64) (*InspectionDocument, error) {
	return nil, nil
}
func (s *speakerPickEdits) SetMessageParts(_ uint32, parts []InspectionMessagePart) (*InspectionDocument, error) {
	s.writes = append(s.writes, slices.Clone(parts))
	return nil, nil
}
func (*speakerPickEdits) CreateMessage(InspectionMessageOwner, string, int64) (*InspectionDocument, uint32, error) {
	return nil, 1, nil
}
func (*speakerPickEdits) CreateMessageParts(InspectionMessageOwner, []InspectionMessagePart) (*InspectionDocument, uint32, error) {
	return nil, 1, nil
}
func (*speakerPickEdits) PointMessageParts(InspectionMessageOwner, uint32, []InspectionMessagePart) (*InspectionDocument, error) {
	return nil, nil
}
func (*speakerPickEdits) UnitSpeaker(unit int) (int64, SpeakerMiss) {
	if n, ok := map[int]int64{0: 5, 2: 7}[unit]; ok {
		return n, SpeakerMiss{}
	}
	return 0, SpeakerMiss{Cause: SpeakerNotFirst, Record: 5, Cell: image.Pt(9, 8)}
}

// speakerPickEditor opens a 64 by 64 map with the three units in lang at
// 1920x1080 and the message popup on page 1, in the window's top-left
// corner clear of the cells the clicks aim at.
func speakerPickEditor(t *testing.T, lang string) (*MapEditor, *speakerPickEdits) {
	t.Helper()
	host := &speakerPickEdits{editorTextEdits: &editorTextEdits{}}
	build := func(string) (*InspectionDocument, error) {
		v, err := NewViewer("speakers", terrain.Grid{Width: 64, Height: 64, Tiles: make([]uint16, 64*64)}, &terrain.Tileset{})
		if err != nil {
			return nil, err
		}
		d := &InspectionDocument{Viewer: v, Source: "speakers", Width: 64, Height: 64, SourceBytes: []byte("map"), Edits: host}
		for i, c := range []image.Point{cellShownUnit, cellUnshownUnit, cellOtherUnit} {
			d.Records = append(d.Records, InspectionRecord{Kind: "Unit", Index: i, Spatial: true, Cell: c, Size: image.Pt(1, 1), Player: 1})
		}
		return d, nil
	}
	font := wordsFont(lang)
	e := NewMapEditor(font, nil, build)
	e.SetBodyFont(font)
	e.SetWords(words.For(lang))
	e.Layout(1920, 1080)
	if err := e.Open("speakers"); err != nil {
		t.Fatal(err)
	}
	e.places[popObject].at, e.places[popObject].placed = image.Pt(e.window.X, e.window.Y), true
	e.Fit()
	e.Select(0, false)
	e.openMessage(1)
	if _, _, ok := e.objectSpec(); !ok || e.messagePage != 1 {
		t.Fatal("the message popup did not open")
	}
	return e, host
}

// pressSpeakerPick clicks the "on the map" button of card c's speaker row.
func pressSpeakerPick(t *testing.T, e *MapEditor, c int) {
	t.Helper()
	if _, _, ok := e.objectSpec(); !ok {
		t.Fatal("the message popup is not shown")
	}
	row := e.fieldRow(2*c + 1)
	k := slices.Index(e.obj.rowActs[row], actPickParam)
	if k < 0 {
		t.Fatalf("card %d's speaker row offers %q, no pick on the map", c, e.obj.rowActs[row])
	}
	r, ok := e.FindControl(ControlQuery{Role: ControlRowButton, N: 2*c + 1, K: k, Flag: false})
	if !ok {
		t.Fatalf("card %d's pick button is not shown", c)
	}
	modesClick(e, center(r))
}

// The speaker row's "on the map" button arms the pick and the status line
// says what the click picks; a second press and Esc disarm it. A click on
// open ground or on a unit no speaker record shows leaves the speaker as it
// was, keeps the pick armed and says so; a click on a unit sets the speaker
// that shows it and ends the pick.
func TestMessageSpeakerTakesAClickOnAUnit(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, _ := speakerPickEditor(t, lang)
			pickWord, missWord := e.word("editor.param.pickspeaker"), e.word("editor.param.nospeakerunit")
			cancel := map[string]string{"en": "Esc cancels", "ru": "Esc отменяет"}[lang]
			if !strings.Contains(pickWord, cancel) || missWord == "editor.param.nospeakerunit" {
				t.Fatalf("the %s words are %q and %q", lang, pickWord, missWord)
			}
			speaker := func(c int) string { return e.obj.buf[2*c+1] }

			pressSpeakerPick(t, e, 0)
			if e.pick != pickParamUnit || !strings.Contains(e.statusText(), pickWord) || strings.Contains(e.statusText(), missWord) {
				t.Fatalf("the button left pick %q, status %q", e.pick, e.statusText())
			}
			pressSpeakerPick(t, e, 0)
			if e.pick != pickNone || strings.Contains(e.statusText(), pickWord) {
				t.Fatalf("a second press left pick %q, status %q", e.pick, e.statusText())
			}
			pressSpeakerPick(t, e, 0)
			_ = e.Key("escape")
			if e.pick != pickNone || speaker(0) != "-1" || e.messagePage != 1 {
				t.Fatalf("Esc left pick %q, speaker %q, page %d", e.pick, speaker(0), e.messagePage)
			}

			pressSpeakerPick(t, e, 0)
			for _, cell := range []image.Point{cellOpenGround, cellUnshownUnit} {
				clickCell(e, cell)
				if e.pick != pickParamUnit || speaker(0) != "-1" {
					t.Fatalf("a click at %v left pick %q, speaker %q", cell, e.pick, speaker(0))
				}
				if status := e.statusText(); !strings.Contains(status, pickWord) || !strings.Contains(status, missWord) {
					t.Fatalf("after a click at %v the status line reads %q", cell, status)
				}
			}
			clickCell(e, cellShownUnit)
			if e.pick != pickNone || speaker(0) != "5" || speaker(1) != "-1" {
				t.Fatalf("a click on the unit left pick %q, speakers %q and %q", e.pick, speaker(0), speaker(1))
			}
			pressSpeakerPick(t, e, 1)
			clickCell(e, cellOtherUnit)
			if e.pick != pickNone || speaker(0) != "5" || speaker(1) != "7" {
				t.Fatalf("the second card's pick left pick %q, speakers %q and %q", e.pick, speaker(0), speaker(1))
			}
		})
	}
}

// A speaker picked on the map writes, on Apply, the parts the same speaker
// chosen from the list writes.
func TestPickedSpeakerAppliesAsTheListsChoice(t *testing.T) {
	picked, host := speakerPickEditor(t, "en")
	pressSpeakerPick(t, picked, 0)
	clickCell(picked, cellShownUnit)
	picked.applyMessage()

	chosen, listed := speakerPickEditor(t, "en")
	field, ok := chosen.FindControl(ControlQuery{Role: ControlField, N: 1})
	if !ok {
		t.Fatal("the first card's speaker field is not shown")
	}
	modesClick(chosen, center(field))
	row := slices.Index(chosen.DropdownChoices(), "5: Anna")
	r, ok := chosen.FindControl(ControlQuery{Role: ControlDropdownRow, N: row})
	if row < 0 || !ok {
		t.Fatalf("the speaker list shows %q", chosen.DropdownChoices())
	}
	modesClick(chosen, center(r))
	if chosen.obj.buf[1] != "5" {
		t.Fatalf("the list chose %q", chosen.obj.buf[1])
	}
	chosen.applyMessage()

	if len(host.writes) != 1 || len(listed.writes) != 1 || !slices.Equal(host.writes[0], listed.writes[0]) || host.writes[0][0].Speaker != 5 {
		t.Fatalf("the pick wrote %v, the list %v", host.writes, listed.writes)
	}
}

// A captioned button of a card's title row is as wide as its caption in the
// font the editor draws it with, the cross keeps its square, the speaker
// field narrows by the button, and no two of them overlap.
func TestCardRowButtonCoversItsCaption(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, _ := speakerPickEditor(t, lang)
			caption := e.actWord(actPickParam)
			cross, okX := e.FindControl(ControlQuery{Role: ControlRowButton, N: 1, K: 0, Flag: false})
			pick, okP := e.FindControl(ControlQuery{Role: ControlRowButton, N: 1, K: 1, Flag: false})
			field, okF := e.FindControl(ControlQuery{Role: ControlField, N: 1})
			if !okX || !okP || !okF {
				t.Fatalf("the title row shows cross %v, button %v, field %v", okX, okP, okF)
			}
			if need := e.bodyFont().Advance(caption); pick.Dx() < need {
				t.Errorf("the button is %d wide, its caption %q needs %d", pick.Dx(), caption, need)
			}
			if cross.Dx() != cross.Dy() {
				t.Errorf("the cross is %v, not a square", cross)
			}
			for _, pair := range [][2]image.Rectangle{{cross, pick}, {field, cross}, {field, pick}} {
				if pair[0].Overlaps(pair[1]) {
					t.Errorf("%v overlaps %v", pair[0], pair[1])
				}
			}
		})
	}
}

// Moving the armed card moves the armed field with it: the click that
// follows sets the speaker of the card that was armed, not of the card that
// now stands in its place.
func TestSpeakerPickFollowsItsCard(t *testing.T) {
	for _, move := range []string{"key", "drag"} {
		t.Run(move, func(t *testing.T) {
			e, _ := speakerPickEditor(t, "en")
			pressSpeakerPick(t, e, 0)
			if move == "key" {
				_ = e.Key("carddown")
			} else {
				e.moveMessageCard(0, 2)
			}
			if first := e.obj.messageParts[1].Number; first != 1 {
				t.Fatalf("the card did not move: numbers %d and %d", e.obj.messageParts[0].Number, first)
			}
			clickCell(e, cellShownUnit)
			if got, other := e.obj.buf[3], e.obj.buf[1]; got != "5" || other != "-1" {
				t.Fatalf("the moved card has speaker %q and the other %q", got, other)
			}
		})
	}
}

// Apply, Cancel and the cross of the armed card end the pick with the
// speaker field: the pick is none and the next click on a unit selects it.
func TestSpeakerPickEndsWithItsTarget(t *testing.T) {
	for _, tc := range []struct {
		name string
		end  func(*testing.T, *MapEditor)
	}{
		{"apply", func(t *testing.T, e *MapEditor) {
			e.obj.buf[2] = "Changed"
			r, ok := e.FindControl(ControlQuery{Role: ControlPopupButton, N: 0})
			if !ok {
				t.Fatal("no Apply button")
			}
			modesClick(e, center(r))
		}},
		{"cancel", func(t *testing.T, e *MapEditor) {
			r, ok := e.FindControl(ControlQuery{Role: ControlPopupButton, N: 1})
			if !ok {
				t.Fatal("no Cancel button")
			}
			modesClick(e, center(r))
		}},
		{"cross", func(t *testing.T, e *MapEditor) {
			r, ok := e.FindControl(ControlQuery{Role: ControlRowButton, N: 1, K: 0, Flag: false})
			if !ok {
				t.Fatal("no cross")
			}
			modesClick(e, center(r))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, _ := speakerPickEditor(t, "en")
			pressSpeakerPick(t, e, 0)
			tc.end(t, e)
			if e.pick != pickNone || strings.Contains(e.statusText(), e.word("editor.param.pickspeaker")) {
				t.Fatalf("the pick is %q, status %q", e.pick, e.statusText())
			}
			clickCell(e, cellOtherUnit)
			if e.selected != 2 {
				t.Fatalf("the next click left record %d selected, want the unit's 2", e.selected)
			}
		})
	}
}
