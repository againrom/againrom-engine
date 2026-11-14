package ui

import (
	"fmt"
	"image"
	"testing"
)

func TestModWindowButtonsLieInsideTheScreenOnEveryPageInBothLanguages(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, size := range []image.Point{{640, 480}, {800, 600}} {
			t.Run(fmt.Sprintf("%s/%dx%d", lang, size.X, size.Y), func(t *testing.T) {
				e := modesWordsEditor(t, "workspace", lang, size)
				host := e.doc.Edits.(*modesWorkspace)
				host.overview, host.graph = modesOverview, modesGraph()
				host.actionsOpen = true
				for i := range 9 {
					host.actions = append(host.actions, ModMapAction{Label: fmt.Sprintf("map%d.alm: unused", i), Buttons: []ModMapButton{{Caption: "Make mission", Key: "makemission", Value: "x"}, {Caption: "Remove", Key: "removemap", Value: "x"}}})
				}
				if err := e.Key("package"); err != nil {
					t.Fatal(err)
				}
				screen := image.Rect(0, 0, size.X, size.Y)
				for tab, page := range e.modPages() {
					e.turnModPage(tab)
					for top := 0; top < 14; top++ {
						e.objectTop = top
						s, _, _ := e.modSpec()
						l := layoutPopup(s, e.font, e.bodyFont())
						if len(l.Buttons) != len(modButtons) {
							t.Fatalf("page %d: %d buttons", page, len(l.Buttons))
						}
						for n, r := range l.Buttons {
							if !r.In(screen) {
								t.Errorf("page %d top %d: button %d %v lies outside %v", page, top, n, r, screen)
							}
						}
						if !l.Rect.In(screen) {
							t.Errorf("page %d top %d: window %v lies outside %v", page, top, l.Rect, screen)
						}
					}
				}
			})
		}
	}
}
