package game

import (
	"image"
	"image/color"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// secondGeneratorInkCount counts the pixels of r in frame that are exactly c.
func secondGeneratorInkCount(frame *image.RGBA, r image.Rectangle, c color.RGBA) int {
	return secondGeneratorNearCount(frame, r, c, 0)
}

// secondGeneratorNearCount counts the pixels of r in frame within tol of c in
// every channel.
func secondGeneratorNearCount(frame *image.RGBA, r image.Rectangle, c color.RGBA, tol int) int {
	near := func(a, b uint8) bool { return int(a)-int(b) <= tol && int(b)-int(a) <= tol }
	n := 0
	r = r.Intersect(frame.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if p := frame.RGBAAt(x, y); near(p.R, c.R) && near(p.G, c.G) && near(p.B, c.B) {
				n++
			}
		}
	}
	return n
}

// The second game's generator on the installed art draws as the original
// does: a hovered difficulty shows main.txt 247 (R2-ENGINE-384); the prompt
// draws gold and the name yellow (R2-ENGINE-386); the detail tip stands
// between the card and the portrait (DIV-3015); the values and the pool
// draw gold in font4 over black (R2-ENGINE-387); four commands stand on the
// command panel (DIV-3014); a hovered weapon class shows its authored lines
// in (160,120,50) over (8,8,8) (R2-ENGINE-385); and the card's labels stand
// inside its border (R2-ENGINE-388, DIV-3017).
func TestReleaseSecondGameGeneratorDrawsAsTheOriginal(t *testing.T) {
	f := secondGameFront(t)
	if f.ChargenAssets == nil {
		t.Fatal("the second game's install loaded no generator")
	}
	frames, c := generatorWitnessFrames(t, f)
	byName := map[string]generatorWitnessFrame{}
	for _, w := range frames {
		byName[w.name] = w
	}
	gold, yellow := color.RGBA{185, 159, 73, 255}, color.RGBA{255, 255, 116, 255}
	t.Run("difficulty hover help", func(t *testing.T) {
		if w := byName["precreate-level-hover"]; !w.tooltip.Visible || !strings.HasSuffix(w.tooltip.Target, "/247") {
			t.Errorf("hovered difficulty: visible %v, target %q; want main.txt 247 shown", w.tooltip.Visible, w.tooltip.Target)
		}
	})
	t.Run("name inks", func(t *testing.T) {
		frame := byName["precreate-name"].frame
		if n := secondGeneratorInkCount(frame, image.Rect(300, 430, 464, 452), yellow); n < 20 {
			t.Errorf("%d yellow name pixels", n)
		}
		if n := secondGeneratorInkCount(frame, image.Rect(200, 430, 296, 452), gold); n < 20 {
			t.Errorf("%d gold prompt pixels", n)
		}
		if n := secondGeneratorInkCount(frame, frame.Bounds(), color.RGBA{101, 39, 61, 255}); n != 0 {
			t.Errorf("%d pixels in the first game's name ink", n)
		}
	})
	t.Run("detail tip position", func(t *testing.T) {
		if r := c.TipPanel().Rect; r != image.Rect(160, 280, 472, 480) {
			t.Errorf("detail tip at %v, want (160,280)-(472,480)", r)
		}
	})
	t.Run("attribute values", func(t *testing.T) {
		frame := byName["detail"].frame
		st := &f.generator().Detail.Stats
		for i, box := range st.Value {
			r := box.Rectangle()
			if g, b := secondGeneratorInkCount(frame, r, gold), secondGeneratorInkCount(frame, r, color.RGBA{0, 0, 0, 255}); g < 5 || b < 5 {
				t.Errorf("value %d: %d gold and %d black pixels", i, g, b)
			}
			if n := secondGeneratorInkCount(frame, r, color.RGBA{255, 255, 255, 255}); n != 0 {
				t.Errorf("value %d: %d white pixels", i, n)
			}
		}
		if n := secondGeneratorInkCount(frame, st.Pool.Rectangle(), gold); n < 5 {
			t.Errorf("pool: %d gold pixels", n)
		}
	})
	t.Run("four commands", func(t *testing.T) {
		if rects := ui.ChargenDetailedNavLabelRects(c); len(rects) != 4 {
			t.Errorf("%d command labels, want 4", len(rects))
		}
	})
	t.Run("weapon class hover box", func(t *testing.T) {
		w := byName["detail-skill-hover"]
		lines := strings.Count(f.ChargenAssets.SkillHover[0][1], "#") + 1
		if !w.tooltip.Visible || w.tooltip.Bounds.Dy() != 14*lines+5 {
			t.Fatalf("hovered axe: visible %v, box %v; want %d authored lines", w.tooltip.Visible, w.tooltip.Bounds, lines)
		}
		r := w.tooltip.Bounds
		if ink, shadow := secondGeneratorInkCount(w.frame, r, color.RGBA{160, 120, 50, 255}), secondGeneratorInkCount(w.frame, r, color.RGBA{8, 8, 8, 255}); ink < 50 || shadow < 50 {
			t.Errorf("hover box: %d ink and %d shadow pixels", ink, shadow)
		}
		if n := secondGeneratorInkCount(w.frame, r, color.RGBA{0xf2, 0xe6, 0xc4, 0xff}); n != 0 {
			t.Errorf("hover box: %d pixels in the shared hover ink", n)
		}
	})
	t.Run("statistics card", func(t *testing.T) {
		frame := byName["detail"].frame
		// Attribute and skill rows: labels start at the canvas's x 12 plus 6,
		// in shades of the sheet's gold.
		rows := image.Rect(0, 280, 16, 345)
		if n := secondGeneratorNearCount(frame, rows, gold, 12); n != 0 {
			t.Errorf("%d label pixels left of x 16, over the card's border", n)
		}
		if n := secondGeneratorNearCount(frame, image.Rect(16, 280, 60, 345), gold, 12); n < 50 {
			t.Errorf("%d label pixels inside the card", n)
		}
	})
}

// Restore on the second game returns the picture's opening values and skill
// after a spend, a skill change or a Reset, as the first game's does
// (DIV-2868, DIV-3014); Reset keeps its own rule (R2-ENGINE-286). Accept
// commits the draft's result, and Restore leaves exactly the opening result
// to commit, so the hero a SAV records is the one Accept would commit
// without the detour.
func TestReleaseSecondGameGeneratorRestoreReturnsTheOpeningDraft(t *testing.T) {
	f := secondGameFront(t)
	if f.ChargenAssets == nil {
		t.Fatal("the second game's install loaded no generator")
	}
	setup := f.ChargenSetup()
	for i := range f.generator().PreCreate.Heroes {
		c := ui.NewChargen(setup)
		c.SelectPreChoice(i)
		c.Forward()
		opening, ok := c.Result()
		if !ok {
			t.Fatalf("hero %d: no result after Forward", i)
		}
		pool := c.Remaining()
		skill := (opening.Choices[chargenChoiceSkill] + 1) % 4
		for _, e := range []struct {
			name string
			edit func()
		}{
			{"spend", func() { c.AdjustStat(0, -1); c.AdjustStat(1, 1) }},
			{"skill", func() { c.SelectSkill(skill) }},
			{"reset", func() { c.SelectSkill(skill); c.Reset() }},
		} {
			name := e.name
			e.edit()
			c.Restore()
			r, _ := c.Result()
			if !slices.Equal(r.Stats, opening.Stats) || r.Choices[chargenChoiceSkill] != opening.Choices[chargenChoiceSkill] || c.Remaining() != pool ||
				r.Name != opening.Name || r.Difficulty != opening.Difficulty || !slices.Equal(r.Choices, opening.Choices) {
				t.Errorf("hero %d: Restore after %s gave %+v, pool %d; want %+v, pool %d", i, name, r, c.Remaining(), opening, pool)
			}
		}
	}
}
