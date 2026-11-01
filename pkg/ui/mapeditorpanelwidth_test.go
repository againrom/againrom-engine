package ui

import (
	"image"
	"testing"
)

// panelSizes are the window sizes the side panel proofs run at.
var panelSizes = []image.Point{{640, 480}, {1024, 768}, {1920, 1080}}

// The side panels have no greatest width: the inner-edge drag grows each of
// them until the map keeps its least width beside the other panel's shown
// width, and never further. The bottom panel stops at its greatest height.
func TestMapEditorSidePanelsGrowToTheRoomTheMapLeaves(t *testing.T) {
	for _, size := range panelSizes {
		for _, lang := range []string{"", "russian"} {
			name := size.String() + lang
			// The left panel beside the other side panel's shown width.
			e, _, _ := panelsEditor(t, size.X, size.Y, lang)
			if e.panelOpen(panelLeft) {
				dragPanelEdge(t, e, popTree, 3000)
				shown := e.panelsShown()
				if want := size.X - mapLeastWidth - shown[panelRight]; shown[panelLeft] != want {
					t.Errorf("%s: the left panel grew to %d, want the room %d", name, shown[panelLeft], want)
				}
				if got := e.mapArea().Dx(); got < mapLeastWidth {
					t.Errorf("%s: the map keeps %d, less than %d", name, got, mapLeastWidth)
				}
				if size.X >= 1920 && shown[panelLeft] <= 420 {
					t.Errorf("%s: the left panel stopped at %d", name, shown[panelLeft])
				}
			}
			// The right panel, the left one minimized so that it fits.
			e, _, _ = panelsEditor(t, size.X, size.Y, lang)
			_ = e.Key(sidePanels[panelLeft].key)
			if e.panelOpen(panelRight) {
				dragPanelEdge(t, e, popProps, -3000)
				shown := e.panelsShown()
				if want := size.X - mapLeastWidth - shown[panelLeft]; shown[panelRight] != want {
					t.Errorf("%s: the right panel grew to %d, want the room %d", name, shown[panelRight], want)
				}
				if got := e.mapArea().Dx(); got < mapLeastWidth {
					t.Errorf("%s: the map keeps %d, less than %d", name, got, mapLeastWidth)
				}
				if size.X >= 1920 && shown[panelRight] <= 800 {
					t.Errorf("%s: the right panel stopped at %d", name, shown[panelRight])
				}
			}
			// The bottom panel keeps its greatest height.
			e, _, _ = panelsEditor(t, size.X, size.Y, lang)
			_ = e.Key(sidePanels[panelBottom].key)
			if e.panelOpen(panelBottom) {
				dragBottomEdge(t, e, -3000)
				got := e.panelsShown()[panelBottom]
				top := e.toolbarRect().Max.Y
				room := e.mapBottom(top) - top - mapLeastHeight
				if want := min(sidePanels[panelBottom].most, room); got != want {
					t.Errorf("%s: the bottom panel grew to %d, want %d", name, got, want)
				}
			}
		}
	}
}
