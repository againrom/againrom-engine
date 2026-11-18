package ui

import (
	"image"
	"testing"
)

// panelSizes are the window sizes the side panel proofs run at.
var panelSizes = []image.Point{{640, 480}, {1024, 768}, {1920, 1080}}

// The left panel leaves the map its least width. The bottom panel keeps its height limit.
func TestMapEditorSidePanelsGrowToTheRoomTheMapLeaves(t *testing.T) {
	for _, size := range panelSizes {
		for _, lang := range []string{"", "russian"} {
			name := size.String() + lang
			e, _, _ := panelsEditor(t, size.X, size.Y, lang)
			if e.panelOpen(panelLeft) {
				dragPanelEdge(t, e, popTree, 3000)
				shown := e.panelsShown()
				if want := size.X - mapLeastWidth; shown[panelLeft] != want {
					t.Errorf("%s: the left panel grew to %d, want the room %d", name, shown[panelLeft], want)
				}
				if got := e.mapArea().Dx(); got < mapLeastWidth {
					t.Errorf("%s: the map keeps %d, less than %d", name, got, mapLeastWidth)
				}
				if size.X >= 1920 && shown[panelLeft] <= 420 {
					t.Errorf("%s: the left panel stopped at %d", name, shown[panelLeft])
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
