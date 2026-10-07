package ui

import (
	"bytes"
	"image"
	"image/color"
	"slices"
	"testing"

	"againrom/pkg/render/terrain"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestSpellCellRimsRequireDiagnosticsAndKeepUnitEffectArt(t *testing.T) {
	for _, tc := range []struct {
		name   string
		school int
		color  color.RGBA
	}{
		{"air", 4, color.RGBA{230, 230, 106, 255}},
		{"life", 5, color.RGBA{140, 255, 154, 255}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := overlayViewer(t, 3, 3, 96, 96)
			v.SetFlat(true)
			v.cam.X, v.cam.Y, v.cam.Zoom = 0, 0, 1
			art := unitArt(1, 1, 0, 0, 1, 1)
			art.Frames[0].Palette[5] = color.RGBA{20, 80, 30, 255}
			sheet := markSheet()
			for i := range sheet.Frames[0].Pixels {
				sheet.Frames[0].Pixels[i] = color.RGBA{220, 60, 200, 255}
			}
			e := withFrame(image.Pt(1, 1), art)
			e.ID, e.SpellFX, e.SpellFXSchool = 1, 8, tc.school
			e.HP, e.MaxHP, e.Mana, e.MaxMana = 80, 100, 20, 40
			e.Marks = []UnitMark{{Sheet: sheet, Mark: terrain.EffectMark{}}}
			v.SetEntities([]MapEntity{e})
			var first statusDrawRecorder
			for i, show := range []bool{false, true, false} {
				if i > 0 {
					v.SetUnits(show, nil)
				}
				frame := image.NewRGBA(image.Rect(0, 0, 96, 96))
				ground := color.RGBA{16, 24, 32, 255}
				frame.SetRGBA(40, 63, ground)
				for _, pass := range v.overlayPasses() {
					for _, r := range pass.Rects {
						blendScreenRect(frame, r, pass.Color)
					}
				}
				want := ground
				if show {
					want = tc.color
				}
				if got := frame.RGBAAt(40, 63); got != want {
					t.Errorf("toggle%d diagnostics%v: border pixel=%v, want literal%v", i, show, got, want)
				}
				var rec statusDrawRecorder
				v.drawArt(&rec)
				unit, effect := v.staticImage(e.Frame), v.effectImage(sheet.Frames[0])
				if !slices.Contains(rec.imgs, unit) || !slices.Contains(rec.imgs, effect) {
					t.Fatalf("toggle%d omitted the nonempty unit or pink effect submission", i)
				}
				draws, err := v.HeadlessArtDraws()
				if err != nil || len(draws) != len(rec.imgs) {
					t.Fatalf("headless art witness: draws%d ordinary%d error%v", len(draws), len(rec.imgs), err)
				}
				unitPixels, effectPixels := 0, 0
				for _, d := range draws {
					if d.Kind == "sprite" && d.Frame == e.Frame && bytes.Equal(d.Pixels.Pix, v.spritePixels(e.Frame).Pix) {
						unitPixels++
					}
					if d.Kind == "effect" && d.Effect == sheet.Frames[0] && bytes.Equal(d.Pixels.Pix, sheet.Frames[0].RGBA().Pix) {
						effectPixels++
					}
				}
				if unitPixels != 1 || effectPixels != 1 {
					t.Fatalf("headless witness source pixels: unit%d pink effect%d, want1each", unitPixels, effectPixels)
				}
				if i == 0 {
					first = rec
				} else if !slices.Equal(rec.imgs, first.imgs) || !slices.Equal(rec.alpha, first.alpha) {
					t.Fatalf("toggle%d changed production drawArt submissions", i)
				}
			}
			v.sel, v.healthBarsHidden = selection{e.ID}, true
			health, _ := v.healthBarScreenRects()
			mana, _ := v.manaBarScreenRects()
			if len(health) != 1 || len(mana) != 1 || len(v.spellEffectPasses()) != 0 {
				t.Fatal("selected marked unit lost overhead status or regained a diagnostic rim")
			}
		})
	}
}

func TestHeadlessArtWitnessRejectsUnidentifiedTexture(t *testing.T) {
	r := headlessArtTarget{v: overlayViewer(t, 3, 3, 96, 96)}
	r.DrawImage(ebiten.NewImage(1, 1), &ebiten.DrawImageOptions{})
	if r.err == nil {
		t.Fatal("unknown production texture produced a partial witness without an error")
	}
}
