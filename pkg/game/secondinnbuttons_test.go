package game

import (
	"image"
	"testing"

	"againrom/pkg/ui"
)

// The second game's inn lists Sleep, Hire, Talk, Exit in every town, and the
// four plaques take the shop's four-button layout: the rects the shop's
// command panel draws its plaques at, every one inside the panel picture
// (shopmenu.bmp over the upper region), none touching another or the
// panel's border.
func TestSecondInnButtonsAreSleepFirstInTheShopsLayout(t *testing.T) {
	order := []string{"sleep", "hire", "talk", "exit"}
	for _, id := range []int{1, 2, 3} {
		_, s := sleepTown(id)
		r := s.townDescription().Room("inn")
		if len(r.Buttons) != len(order) {
			t.Fatalf("town %d: %d buttons", id, len(r.Buttons))
		}
		panel := ui.TownWideUpperRegion
		var rects []image.Rectangle
		for i, b := range r.Buttons {
			if b.Name != order[i] {
				t.Errorf("town %d: button %d is %q, want %q", id, i, b.Name, order[i])
			}
			rect := b.Rect.Rectangle()
			if want := ui.ShopButtonRect(i); rect != want {
				t.Errorf("town %d: %s at %v, the shop's plaque %d is at %v", id, b.Name, rect, i, want)
			}
			if !rect.In(panel.Inset(1)) {
				t.Errorf("town %d: %s %v touches the border of the panel %v", id, b.Name, rect, panel)
			}
			for _, o := range rects {
				if rect.Overlaps(o) {
					t.Errorf("town %d: %s %v overlaps %v", id, b.Name, rect, o)
				}
			}
			rects = append(rects, rect)
		}
		rows := s.Rows()
		if len(rows) < 3 || rows[0].Text != s.in.Words.TavernSleep || rows[len(rows)-1].Text != secondInnExit {
			t.Errorf("town %d: rows %v do not start with Sleep and end with Exit", id, rows)
		}
	}
}
