package game

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/ui"
)

func armOpenButton(t *testing.T, e *ui.MapEditor, row int) image.Rectangle {
	t.Helper()
	for i := 0; i < 200; i++ {
		if r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlGoalOpen, N: row}); ok {
			clearPopup(e, r.Min.Add(r.Size().Div(2)))
			if r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlGoalOpen, N: row}); ok {
				return r
			}
		}
		_ = e.Key("pagedown")
	}
	t.Fatalf("arm row %d has no reachable Open", row)
	return image.Rectangle{}
}

func backFromArm(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	for n := 0; n < 8; n++ {
		r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: n})
		if !ok {
			if n == 0 {
				t.Fatal("popup missing")
			}
			r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: n - 1})
			trigClick(t, e, r, ok, "Back")
			return
		}
	}
	t.Fatal("Back button not found")
}

func TestReleaseMapEditorArmsFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/131.alm"); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	_ = e.Key("goals")
	dir, lang := os.Getenv("AGAINROM_EDITOR_FRAMES"), installLanguage(x)
	writeEditorFrame(t, e, filepath.Join(dir, "arms-panel-"+lang+".png"), dir != "", image.Rectangle{})
	r := armOpenButton(t, e, 0)
	trigClick(t, e, r, true, "arm Open")
	writeEditorFrame(t, e, filepath.Join(dir, "arms-trigger-"+lang+".png"), dir != "", image.Rectangle{})
}
