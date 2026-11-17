package widgetkit

import "testing"

// A row that states Maximized has the window maximized last, once, after its
// size and resizing; a row that does not leaves the window as sized.
func TestApplyWindowMaximizesARowThatAsksForIt(t *testing.T) {
	for _, asks := range []bool{true, false} {
		s := &fakeScreen{d: desk(1920, 1040), cw: 640, ch: 480}
		c := s.calls()
		var order []string
		c.SetSize, c.Resizing = func(int, int) { order = append(order, "size") }, func(bool) { order = append(order, "resize") }
		c.Maximize = func() { order = append(order, "maximize") }
		applyWindow(c, WindowRow{W: 1280, H: 800, Resizable: true, Maximized: asks}, WindowForm{})
		want := []string{"size", "resize"}
		if asks {
			want = append(want, "maximize")
		}
		if len(order) != len(want) {
			t.Fatalf("asks %v: calls %v, want %v", asks, order, want)
		}
		for i := range want {
			if order[i] != want[i] {
				t.Fatalf("asks %v: calls %v, want %v", asks, order, want)
			}
		}
	}
}
