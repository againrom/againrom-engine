package ui

import "testing"

func TestEditorBrushPaletteKeysClamp(t *testing.T) {
	for _, mode := range []string{"terrain", "height", "objects"} {
		t.Run(mode, func(t *testing.T) {
			e, _ := brushEditor(t)
			if err := e.Key(mode); err != nil {
				t.Fatal(err)
			}
			n := len(e.brushRows())
			if n < 2 {
				t.Fatalf("need multiple palette rows, got %d", n)
			}
			e.brushRowClick(0)
			_ = e.Key("up")
			if got := e.brush.sel[e.brush.mode]; got != 0 {
				t.Errorf("Up at first row selected %d of %d", got, n)
			}
			e.brushRowClick(0)
			for i := 0; i < n-1; i++ {
				_ = e.Key("down")
				if got := e.brush.sel[e.brush.mode]; got != i+1 {
					t.Fatalf("Down selected %d, want %d", got, i+1)
				}
			}
			_ = e.Key("down")
			if got := e.brush.sel[e.brush.mode]; got != n-1 {
				t.Errorf("Down at last row selected %d of %d", got, n)
			}
		})
	}
}

func TestEditorLegacyFieldChoicesClamp(t *testing.T) {
	e, h := popupEditor(t)
	if err := e.Open("popups"); err != nil {
		t.Fatal(err)
	}
	e.Select(2, false)
	e.openPrompt(editorPrompt{kind: promptField, field: 1, path: "1"})
	for _, c := range []struct {
		delta int
		want  string
	}{{-1, "1"}, {1, "2"}, {1, "3"}, {1, "3"}, {-1, "2"}, {-1, "1"}, {-1, "1"}} {
		if !e.stepChoice(c.delta) || e.prompt.path != c.want {
			t.Fatalf("stepChoice(%d) selected %q, want %q", c.delta, e.prompt.path, c.want)
		}
	}
	if e.Dirty() || len(h.calls) != 0 {
		t.Fatal("choice navigation wrote the map")
	}
}
