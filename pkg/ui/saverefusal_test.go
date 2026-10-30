package ui

import (
	"errors"
	"image"
	"testing"
)

// refusalSeams is a save dialog whose admission refuses with reason, or
// admits when reason is empty, and records every request.
type refusalSeams struct {
	reason string
	asked  []SaveRequest
}

func (s *refusalSeams) install(a *App) {
	a.SetSaveDialogSeams(SaveDialogSeams{Prepare: func(r SaveRequest) (PreparedSave, error) {
		s.asked = append(s.asked, r)
		if r.Admission && s.reason == "" {
			return PreparedSave{}, nil
		}
		return PreparedSave{}, errors.New(s.reason)
	}})
}

// savePointTalkTown is a town with a talk open.
type savePointTalkTown struct {
	*stubTown
	pic *image.RGBA
}

func (s *savePointTalkTown) TownDialogue() (*image.RGBA, bool) { return s.pic, s.pic != nil }
func (s *savePointTalkTown) TownDialogueButton() (image.Rectangle, bool) {
	return image.Rectangle{}, false
}
func (s *savePointTalkTown) AdvanceTownDialogue() TownAction { return TownAction{} }

// A SAVE asked where the save dialog stays closed shows the admission's
// reason on the screen's refusal line: F2 and the menu SAVE in a town that
// cannot save, F2 over an open talk, F2 under a map notice. The dialog stays
// closed and no file is asked for. An admission that names no reason shows
// nothing. Fails while those points drop the SAVE silently.
func TestSaveRefusalNamesItsReasonWhereTheDialogStaysClosed(t *testing.T) {
	const reason = "Close the message before saving."
	for _, route := range []string{"town F2", "town menu", "talk F2", "map notice F2"} {
		for _, named := range []bool{true, false} {
			t.Run(route, func(t *testing.T) {
				h := newHaltFix(t, haltOpts{})
				onMap := route == "map notice F2"
				rows := []TownRow{{Text: "town", Choosable: true}}
				switch route {
				case "town F2", "town menu":
					h.a.SetTown(&savePointTown{stubTown: &stubTown{rows: rows}})
					h.a.flow.showTown("")
				case "talk F2":
					h.a.SetTown(&savePointTalkTown{stubTown: &stubTown{rows: rows}, pic: image.NewRGBA(image.Rect(0, 0, 40, 20))})
					h.a.flow.showTown("")
				default:
					h.v.SetNotice("a message", NoticeDialogue)
				}
				screen := h.a.Screen()
				seams := &refusalSeams{}
				if named {
					seams.reason = reason
				}
				seams.install(h.a)
				if route == "town menu" {
					h.frame(appInput{Escape: true})
					if screen = h.a.Screen(); screen != ScreenGameMenu {
						t.Fatal("town menu unavailable", screen)
					}
					h.a.flow.applyGameMenuAction(gameMenuSave)
				} else {
					h.frame(appInput{SaveGame: true})
				}
				if len(seams.asked) != 1 || !seams.asked[0].Admission || seams.asked[0].OnMap != onMap {
					t.Fatalf("requests %+v, want one admission on map %v", seams.asked, onMap)
				}
				if h.a.Screen() != screen {
					t.Fatalf("screen %v, want %v", h.a.Screen(), screen)
				}
				shown := h.a.flow.msg
				if onMap {
					shown = ""
					if lines := h.v.MessageLines(); len(lines) != 0 {
						shown = lines[len(lines)-1].Text
					}
					if !h.v.NoticeOpen() {
						t.Fatal("the refusal closed the notice")
					}
				}
				if want := seams.reason; shown != want {
					t.Fatalf("refusal line %q, want %q", shown, want)
				}
				if !onMap && named && h.a.flow.saveRefused != reason {
					t.Fatal("the refusal is not marked for the menu font")
				}
			})
		}
	}
}
