package ui

import "github.com/hajimehoshi/ebiten/v2"

// Seam for the editor frame-time probe test (the external test package
// cannot reach these fields): it feeds the editor's input seam, nativeInput,
// and reads the state the probe waits on.

// ProbeInput is one tick of input for the editor: the canvas input and at
// most one key that is just pressed this tick.
type ProbeInput struct {
	Canvas Input
	Key    ebiten.Key
	HasKey bool
	Held   []ebiten.Key
}

// ProbeSetInput routes the editor's Update through next instead of the OS.
func ProbeSetInput(e *MapEditor, next func() ProbeInput) {
	e.nativeInput = func() editorNativeInput {
		in := next()
		held := func(k ebiten.Key) bool {
			for _, h := range in.Held {
				if h == k {
					return true
				}
			}
			return in.HasKey && in.Key == k
		}
		return editorNativeInput{
			pressed:     held,
			justPressed: func(k ebiten.Key) bool { return in.HasKey && in.Key == k },
			canvas:      in.Canvas,
			keysDown:    in.HasKey || len(in.Held) > 0,
		}
	}
}

// ProbeCamera is the viewer camera's position and zoom.
func ProbeCamera(e *MapEditor) (x, y, zoom float64) {
	c := e.doc.Viewer.cam
	return c.X, c.Y, c.Zoom
}

// ProbeWindowOpen reports that the unit or object window shows fields.
func ProbeWindowOpen(e *MapEditor) bool {
	return e.nativeUnitWindow() || len(e.obj.fields) > 0
}

// ProbeMessagePage is the message popup's page, 0 when it is closed.
func ProbeMessagePage(e *MapEditor) uint32 { return e.messagePage }

// ProbeOpenMessage opens the message popup on page as the dialogue card
// route does, placed at the window's top-left corner.
func ProbeOpenMessage(e *MapEditor, page uint32) {
	e.openMessage(page)
}

// ProbeSpeakerPickArmed reports that the speaker pick on the map is armed.
func ProbeSpeakerPickArmed(e *MapEditor) bool { return e.speakerPicking() }

// ProbeSpeakerPickButton is the centre of the "on the map" button of card
// c's speaker row.
func ProbeSpeakerPickButton(e *MapEditor, c int) (x, y int, ok bool) {
	row := e.fieldRow(2*c + 1)
	for k, a := range e.obj.rowActs[row] {
		if a == actPickParam {
			if r, found := e.FindControl(ControlQuery{Role: ControlRowButton, N: 2*c + 1, K: k, Flag: false}); found {
				return (r.Min.X + r.Max.X) / 2, (r.Min.Y + r.Max.Y) / 2, true
			}
		}
	}
	return 0, 0, false
}

// ProbeTreeTop is the first visible row of the open tree list.
func ProbeTreeTop(e *MapEditor) (int, bool) {
	s, ok := e.treeSpec()
	if !ok {
		return 0, false
	}
	top, _ := s.List.Visible()
	return top, true
}

// ProbeListOpen reports that a list, panel or tree popup is open.
func ProbeListOpen(e *MapEditor) bool { return e.listKind() != "" || e.PanelOpen() }
