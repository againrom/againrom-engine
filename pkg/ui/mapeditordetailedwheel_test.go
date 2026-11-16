package ui

import (
	"fmt"
	"image"
	"testing"
)

// The detailed person view scrolls only where its rows do not fit the room:
// at 1920x1080 and larger five wheel notches leave the window and the shown
// rows as they were, the fields' notes standing behind their (?) marks; at
// 1280x720 the rows still scroll.
func TestEditorDetailedViewWheelScrollsOnlyWhereRowsOverflow(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, size := range []image.Point{{1280, 720}, {1920, 1080}, {2560, 1440}, {3840, 2160}} {
			t.Run(fmt.Sprintf("%s/%v", lang, size), func(t *testing.T) {
				e := cornerEditor(t, lang)
				e.props.plain = false
				e.Layout(size.X, size.Y)
				e.Select(viewsUnit, false)
				before, _, _ := e.objectSpec()
				rect := layoutPopup(before, e.font, e.bodyFont()).Rect
				rows := layoutPopup(before, e.font, e.bodyFont()).Rows
				e.ScrollDetails(5)
				after, _, _ := e.objectSpec()
				l := layoutPopup(after, e.font, e.bodyFont())
				if size.X == 1280 {
					// The card's and the figure's rows are one line each and
					// many tall, so the window's height follows the rows shown.
					if after.Top == 0 || !l.Rect.In(after.Bounds) || !rect.In(before.Bounds) {
						t.Fatalf("1280x720: top %d, window %v then %v", after.Top, rect, l.Rect)
					}
					return
				}
				if after.Top != 0 || l.Rect != rect || len(l.Rows) != len(rows) {
					t.Fatalf("top %d, window %v then %v, rows %d then %d", after.Top, rect, l.Rect, len(rows), len(l.Rows))
				}
			})
		}
	}
}
