package ui

import (
	"image"
	"strings"
	"testing"
	"time"

	"againrom/pkg/words"

	"github.com/hajimehoshi/ebiten/v2"
)

// dirtyPanelEdits is the Map panel's host with unsaved changes.
type dirtyPanelEdits struct{ mapPanelEdits }

func (*dirtyPanelEdits) Dirty() bool { return true }

func panelCloseEditor(t *testing.T, dirty bool) *MapEditor {
	t.Helper()
	base := mapPanelEdits{facts: playable()}
	e, _ := mapPanelEditor(t, &base, nil)
	if dirty {
		e.doc.Edits = &dirtyPanelEdits{base}
	}
	e.Layout(1920, 1080)
	return e
}

func panelClickAt(e *MapEditor, p image.Point) {
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Unix(1, 0))
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
}

func rectCentre(r image.Rectangle) image.Point {
	return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
}

// panelCloses are the ways a player closes the Map panel or a prompt over it.
var panelCloses = []struct {
	name  string
	setup func(e *MapEditor)
	close func(e *MapEditor) error
	left  func(e *MapEditor) bool // what the one close must not leave open
}{
	{
		name:  "panel, Esc",
		setup: func(*MapEditor) {},
		close: func(e *MapEditor) error { return e.Key("escape") },
		left:  func(e *MapEditor) bool { return e.PanelOpen() },
	},
	{
		name:  "panel, close button",
		setup: func(*MapEditor) {},
		close: func(e *MapEditor) error {
			s, _ := e.panelSpec()
			panelClickAt(e, rectCentre(layoutPopup(s, e.font, e.bodyFont()).Close))
			return nil
		},
		left: func(e *MapEditor) bool { return e.PanelOpen() },
	},
	{
		name:  "metadata prompt, Esc",
		setup: func(e *MapEditor) { e.openMetadata("name") },
		close: func(e *MapEditor) error { return e.Key("escape") },
		left:  func(e *MapEditor) bool { return e.prompt.open() },
	},
	{
		name:  "metadata prompt typed, Esc",
		setup: func(e *MapEditor) { e.openMetadata("name"); e.TextInput("x") },
		close: func(e *MapEditor) error { return e.Key("escape") },
		left:  func(e *MapEditor) bool { return e.prompt.open() },
	},
	{
		name:  "metadata prompt, close button",
		setup: func(e *MapEditor) { e.openMetadata("name") },
		close: func(e *MapEditor) error {
			s, _ := e.promptSpec()
			panelClickAt(e, rectCentre(layoutPopup(s, e.font, e.bodyFont()).Close))
			return nil
		},
		left: func(e *MapEditor) bool { return e.prompt.open() },
	},
	{
		name:  "records list over the panel, Esc",
		setup: func(e *MapEditor) { e.panelFollow(linkPlayers) },
		close: func(e *MapEditor) error { return e.Key("escape") },
		left:  func(e *MapEditor) bool { return e.listShown },
	},
}

func TestMapPanelCloseKeepsTheEditorRunning(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, dirty := range []bool{false, true} {
			for _, c := range panelCloses {
				name := lang + ", " + c.name
				if dirty {
					name += ", dirty"
				}
				t.Run(name, func(t *testing.T) {
					e := panelCloseEditor(t, dirty)
					e.SetWords(words.For(lang))
					c.setup(e)
					if err := c.close(e); err == ebiten.Termination || e.exitAsked {
						t.Fatalf("the close quit the editor (err %v, exitAsked %v)", err, e.exitAsked)
					}
					if e.prompt.kind == promptLeave {
						t.Fatalf("the close asked to leave the editor")
					}
					if c.left(e) {
						t.Fatalf("the close left its popup open")
					}
				})
			}
		}
	}
}

// The map menu's last row still quits a clean map at once and asks about a
// dirty one.
func TestMapMenuExitStillQuitsCleanAndAsksDirty(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e := panelCloseEditor(t, false)
		e.SetWords(words.For(lang))
		e.menuAct("close")
		if !e.ExitAsked() || e.Update() != ebiten.Termination {
			t.Fatalf("%s: a clean map's Exit did not quit", lang)
		}
		e = panelCloseEditor(t, true)
		e.SetWords(words.For(lang))
		e.menuAct("close")
		if e.ExitAsked() || e.prompt.kind != promptLeave {
			t.Fatalf("%s: a dirty map's Exit did not ask", lang)
		}
	}
}

type panickingGame struct{ inUpdate, inDraw bool }

func (g panickingGame) Update() error {
	if g.inUpdate {
		panic("update broke")
	}
	return nil
}
func (g panickingGame) Draw(*ebiten.Image) {
	if g.inDraw {
		panic("draw broke")
	}
}
func (panickingGame) Layout(w, h int) (int, int) { return w, h }

// A panic in the frame loop comes back as the error RunGame returns, with its
// message and stack, instead of ending the program unseen.
func TestFrameGuardTurnsAPanicIntoAnError(t *testing.T) {
	g := &frameGuard{game: panickingGame{inUpdate: true}}
	err := g.Update()
	if err == nil || !strings.Contains(err.Error(), "update broke") || !strings.Contains(err.Error(), "panickingGame") {
		t.Fatalf("Update panic gave %v", err)
	}
	g = &frameGuard{game: panickingGame{inDraw: true}}
	g.Draw(nil)
	if err := g.Update(); err == nil || !strings.Contains(err.Error(), "draw broke") {
		t.Fatalf("Draw panic gave %v", err)
	}
	g = &frameGuard{game: panickingGame{}}
	g.Draw(nil)
	if err := g.Update(); err != nil {
		t.Fatalf("a quiet frame gave %v", err)
	}
}
