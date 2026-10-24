package ui

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type dialogueInputEdits struct{ *editorTextEdits }

func (*dialogueInputEdits) MessageParts(uint32) []InspectionMessagePart {
	return []InspectionMessagePart{{Source: 0, Number: 1, Body: "First", Speaker: -1}, {Source: 1, Number: 2, Body: "Second", Speaker: -1}, {Source: 2, Number: 3, Body: "Third", Speaker: -1}}
}
func (*dialogueInputEdits) MessageFields(uint32, ...int) []InspectionField {
	return []InspectionField{{Key: "body", Text: true}, {Key: "speaker", Value: -1}}
}
func (*dialogueInputEdits) SetMessage(uint32, string, int64) (*InspectionDocument, error) {
	return nil, nil
}
func (*dialogueInputEdits) SetMessageParts(uint32, []InspectionMessagePart) (*InspectionDocument, error) {
	return nil, nil
}
func (*dialogueInputEdits) CreateMessage(InspectionMessageOwner, string, int64) (*InspectionDocument, uint32, error) {
	return nil, 1, nil
}
func (*dialogueInputEdits) CreateMessageParts(InspectionMessageOwner, []InspectionMessagePart) (*InspectionDocument, uint32, error) {
	return nil, 1, nil
}

func TestDialogueLaterPageNativeUpdateUsesTextInput(t *testing.T) {
	e, edits := selectedTextEditor(t, panelFont(), image.Pt(1280, 800))
	e.doc.Edits = &dialogueInputEdits{edits}
	e.openMessage(1)
	e.obj.focus = 2
	updateEditorKeys(t, e, ebiten.KeyControl, ebiten.KeyA)
	e.TextInput("Replacement")
	updateEditorKeys(t, e)
	updateEditorKeys(t, e, ebiten.KeyEnter)
	e.TextInput("Line")
	updateEditorKeys(t, e)
	updateEditorKeys(t, e, ebiten.KeyArrowLeft)
	e.TextInput("!")
	if e.obj.buf[0] != "First" || e.obj.buf[2] != "Replacement\nLin!e" || e.obj.buf[4] != "Third" {
		t.Fatal("native input changed a different page or lost multiline/caret editing", e.obj.buf)
	}
}

func TestDialoguePageButtonsArePainted(t *testing.T) {
	e, edits := selectedTextEditor(t, panelFont(), image.Pt(1280, 800))
	e.doc.Edits = &dialogueInputEdits{edits}
	e.openMessage(1)
	pic := e.ChromeImage()
	for part := range 3 {
		for button := range 5 {
			r, ok := e.RowButtonRect(2*part+1, button, false)
			if !ok {
				t.Fatal("page button has no pointer target", part, button)
			}
			ground := pic.RGBAAt(r.Min.X+2, r.Min.Y+2)
			painted := false
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					painted = painted || pic.RGBAAt(x, y) != ground
				}
			}
			if !painted {
				t.Fatal("page button is clickable but invisible", part, button)
			}
		}
	}
}
