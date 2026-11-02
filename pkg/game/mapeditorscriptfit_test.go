package game

import (
	"fmt"
	"image"
	"os"
	"slices"
	"testing"

	"againrom/pkg/render/text"
	"againrom/pkg/words"
)

// scriptFitFont is a fixture font whose every glyph is seven pixels wide, so
// a caption's advance is long enough to overrun the default button width.
func scriptFitFont(selector int) *text.Font {
	f := &text.Font{Spacing: 1, Selector: selector, Glyphs: make([]text.Glyph, 224)}
	for k := range f.Glyphs {
		f.Glyphs[k] = text.Glyph{Width: 6, Height: 9, Advance: 7, Pixels: make([]text.Pixel, 54)}
	}
	return f
}

// scriptFitCases are the editors the fit is measured in: both languages with
// the fixture font, or the one install AGAINROM_ASSETS names with its font.
func scriptFitCases(t *testing.T) []struct {
	name string
	x    *MapInspector
} {
	t.Helper()
	if root := os.Getenv("AGAINROM_ASSETS"); root != "" {
		x, err := NewMapInspector(root)
		if err != nil {
			t.Fatal(err)
		}
		return []struct {
			name string
			x    *MapInspector
		}{{"install", x}}
	}
	var out []struct {
		name string
		x    *MapInspector
	}
	for _, l := range scriptAddLangs {
		x := heroLabelInspector(t, l.profile, l.sel)
		x.font = scriptFitFont(l.sel)
		out = append(out, struct {
			name string
			x    *MapInspector
		}{l.name, x})
	}
	return out
}

// The new buttons of the Scripts list's popups, Add to trigger on a listed
// condition and Label..., Clear label and Cancel on a variable, are each as
// wide as their caption needs and lie inside the popup, with the buttons
// kept apart.
func TestScriptsPopupButtonsFitTheirCaptions(t *testing.T) {
	b := scriptAddMap(t)
	for _, c := range scriptFitCases(t) {
		x := c.x
		e := openHeroLabelMap(t, x, b)
		e.Layout(1920, 1080)
		closeMapPanel(t, e)
		book := words.For(x.archives.Base.Profile.Language)
		w := func(id string, a ...any) string { return x.inspectionUTF8(fmt.Sprintf(book.Text(id), a...)) }
		check := func(what string, want []string) {
			t.Helper()
			buttons := e.PopupButtons()
			popup, ok := e.PopupRect()
			if !ok {
				t.Fatalf("%s %s: no popup", c.name, what)
			}
			var rects []image.Rectangle
			for _, id := range want {
				at := slices.Index(buttons, w(id))
				if at < 0 {
					t.Fatalf("%s %s: the popup offers %q, not %q", c.name, what, buttons, w(id))
				}
				r, ok := e.PopupButtonRect(at)
				if !ok {
					t.Fatalf("%s %s: button %q has no rectangle", c.name, what, w(id))
				}
				if adv := x.font.Advance(w(id)); adv > r.Dx() {
					t.Errorf("%s %s: %q is %d px wide in a %d px button", c.name, what, w(id), adv, r.Dx())
				}
				if !r.In(popup) {
					t.Errorf("%s %s: button %q at %v is outside the popup %v", c.name, what, w(id), r, popup)
				}
				for _, o := range rects {
					if r.Overlaps(o) {
						t.Errorf("%s %s: button %q at %v overlaps %v", c.name, what, w(id), r, o)
					}
				}
				rects = append(rects, r)
			}
		}

		e.Select(trigRecord(t, e, "Condition", 2), false)
		check("listed condition", []string{"editor.button.apply", "editor.button.cancel", "editor.scripts.totrigger"})

		r0 := variableTarget(t, e)
		newVariable(t, e, "Gold")
		openVariable(t, e, w("editor.scripts.variable", r0)+"  Gold")
		check("variable", []string{"editor.labels.edit", "editor.labels.clear", "editor.button.cancel"})
	}
}
