package ui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"image"
	"image/color"
	"math"
	"reflect"
	"testing"

	"againrom/pkg/render/terrain"
	"github.com/hajimehoshi/ebiten/v2"
)

// spellPassTarget rasterizes the actual production DrawImage submissions on
// CPU. Texture pixels come from the same immutable frames that were uploaded;
// no GPU readback or expected pass list supplies the observed composition.
// It is a nearest-sampled test instrument, not a second production renderer.
type spellPassTarget struct {
	t       *testing.T
	v       *Viewer
	pixels  *image.RGBA
	effects map[*terrain.EffectFrame]string
	bodies  map[*terrain.StaticFrame]string
	calls   []string
}

func (r *spellPassTarget) source(img *ebiten.Image) (*image.RGBA, string) {
	for f, tex := range r.v.effectImages {
		if tex == img {
			return f.RGBA(), r.effects[f]
		}
	}
	for f, tex := range r.v.shadowMasks {
		if tex == img {
			return terrain.ShadowMask(f), "shadow " + r.bodies[f]
		}
	}
	for key, tex := range r.v.staticImages {
		if tex == img {
			return r.v.spritePixels(key.frame), r.bodies[key.frame]
		}
	}
	r.t.Fatal("production submitted an unidentified texture")
	return nil, ""
}

func (r *spellPassTarget) DrawImage(img *ebiten.Image, op *ebiten.DrawImageOptions) {
	src, name := r.source(img)
	r.calls = append(r.calls, name)
	if op.Filter != ebiten.FilterNearest {
		r.t.Fatal("witness requires nearest sampling")
	}
	inverse := op.GeoM
	inverse.Invert()
	for y := r.pixels.Rect.Min.Y; y < r.pixels.Rect.Max.Y; y++ {
		for x := r.pixels.Rect.Min.X; x < r.pixels.Rect.Max.X; x++ {
			sx, sy := inverse.Apply(float64(x)+0.5, float64(y)+0.5)
			p := image.Pt(int(math.Floor(sx)), int(math.Floor(sy)))
			if !p.In(src.Rect) {
				continue
			}
			c, d := src.RGBAAt(p.X, p.Y), r.pixels.RGBAAt(x, y)
			a := float64(c.A) * float64(op.ColorScale.A()) / 255
			channel := func(source, dest uint8, scale float32) uint8 {
				return uint8(math.Round(float64(source)*float64(scale) + float64(dest)*(1-a)))
			}
			if op.Blend == shadowBlend {
				r.pixels.SetRGBA(x, y, color.RGBA{channel(0, d.R, 0), channel(0, d.G, 0), channel(0, d.B, 0), d.A})
			} else {
				r.pixels.SetRGBA(x, y, color.RGBA{channel(c.R, d.R, op.ColorScale.R()), channel(c.G, d.G, op.ColorScale.G()), channel(c.B, d.B, op.ColorScale.B()), channel(c.A, d.A, op.ColorScale.A())})
			}
		}
	}
}

func solidPassSheet(c color.RGBA) *terrain.EffectSheet {
	s := uiSheet(1, 32, 32, 16, 16)
	for i := range s.Frames[0].Pixels {
		s.Frames[0].Pixels[i] = c
	}
	return s
}

func newSpellPassFixture(t *testing.T) (*Viewer, *spellPassTarget) {
	t.Helper()
	v := commandViewer(t)
	v.cam.X, v.cam.Y = 0, 0
	v.unshaded = true
	v.sun.Theta, v.sun.ShroudObject, v.sun.ShroudUnit = 0.05, 8, 4 // the fixture units are invisible owners: each casts one quarter-darkness shadow at the unit-path level
	r := &spellPassTarget{t: t, v: v, pixels: image.NewRGBA(image.Rect(0, 0, 256, 256)),
		effects: make(map[*terrain.EffectFrame]string), bodies: make(map[*terrain.StaticFrame]string)}
	var entities []MapEntity
	for i, c := range []color.RGBA{{200, 40, 0, 255}, {0, 120, 240, 255}} {
		art := unitArt(32, 32, 16, 16, 32, 32)
		art.Frames[0].Palette[5] = c
		entity := withFrame(image.Pt(4, 4), art)
		entity.ID, entity.Translucent = uint32(i+1), true
		entities = append(entities, entity)
		r.bodies[art.Frames[0]] = []string{"body1", "body2"}[i]
	}
	v.SetEntities(entities)
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			r.pixels.SetRGBA(x, y, color.RGBA{16, 24, 32, 255})
		}
	}
	return v, r
}

func TestSpellPassesComposeLiteralSameCellAlphaInProductionOrder(t *testing.T) {
	for _, tc := range []struct {
		name     string
		category terrain.UnitCategory
		calls    []string
		pixel    color.RGBA
	}{
		{"ordinary", terrain.UnitOrdinary, []string{"shadow body1", "body1", "shadow body2", "body2", "fire", "earth", "projectile", "freezing"}, color.RGBA{18, 100, 135, 255}},
		{"alternate", terrain.UnitAlternate, []string{"shadow body1", "body1", "shadow body2", "body2", "fire", "earth", "projectile", "freezing"}, color.RGBA{18, 100, 135, 255}},
		{"air", terrain.UnitAir, []string{"fire", "earth", "shadow body1", "shadow body2", "projectile", "freezing", "body1", "body2"}, color.RGBA{53, 91, 153, 255}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, target := newSpellPassFixture(t)
			for i := range v.entities {
				v.entities[i].DrawCategory = tc.category
			}
			var bolts []SpellBolt
			// Deliberately interleave the submitted tags. The source list cannot be
			// used as an expected pass sequence, and one all-spell pass cannot pass.
			for _, row := range []struct {
				name string
				pass SpellPass
				c    color.RGBA
			}{
				{"projectile", SpellProjectiles, color.RGBA{0, 0, 128, 128}},
				{"fire", SpellOverlayA, color.RGBA{128, 0, 0, 128}},
				{"freezing", SpellOverlayB, color.RGBA{0, 64, 64, 128}},
				{"earth", SpellOverlayA, color.RGBA{0, 128, 0, 128}},
			} {
				sheet := solidPassSheet(row.c)
				target.effects[sheet.Frames[0]] = row.name
				bolts = append(bolts, SpellBolt{Cell: image.Pt(4, 4), To: image.Pt(4, 4),
					Pos: image.Pt(4*ShotScale, 4*ShotScale), Sheet: sheet, Pass: row.pass})
			}
			v.SetSpellBolts(bolts)
			v.drawArt(target)
			wantCalls := tc.calls
			if !reflect.DeepEqual(target.calls, wantCalls) {
				t.Fatalf("production calls=%v want=%v", target.calls, wantCalls)
			}
			// At (144,144), all eight sources overlap. The literals start at
			// (16,24,32), apply their stated alpha and round after each submission.
			if got, want := target.pixels.RGBAAt(144, 144), tc.pixel; got != want {
				t.Fatalf("same-cell composition=%v want literal %v", got, want)
			}
		})
	}
}

func TestMissionFrameUsesTheUnifiedSpellCompositionExactlyOnce(t *testing.T) {
	source, err := parser.ParseFile(token.NewFileSet(), "viewer.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	artCalls, oldSpellCalls := 0, 0
	for _, decl := range source.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "drawFrame" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if ok {
				switch selector.Sel.Name {
				case "drawArt":
					artCalls++
				case "drawSpellArt":
					oldSpellCalls++
				}
			}
			return true
		})
	}
	if artCalls != 1 || oldSpellCalls != 0 {
		t.Fatalf("drawFrame has unified=%d separate-spell=%d calls, want 1/0", artCalls, oldSpellCalls)
	}
}

func TestSpellPassPartitionKeepsProjectileOrderAndClearsReplacedOverlays(t *testing.T) {
	v, target := newSpellPassFixture(t)
	v.SetEntities(nil)
	var bolts []SpellBolt
	for _, item := range []struct {
		name string
		pass SpellPass
	}{{"p2", SpellProjectiles}, {"cloud", SpellOverlayB}, {"p1", SpellProjectiles}, {"wall", SpellOverlayA}} {
		sheet := solidPassSheet(color.RGBA{0, 0, 64, 128})
		target.effects[sheet.Frames[0]] = item.name
		bolts = append(bolts, SpellBolt{Cell: image.Pt(4, 4), To: image.Pt(4, 4), Pos: image.Pt(4*ShotScale, 4*ShotScale), Sheet: sheet, Pass: item.pass})
	}
	v.SetSpellBolts(bolts)
	v.drawArt(target)
	if want := []string{"wall", "p2", "p1", "cloud"}; !reflect.DeepEqual(target.calls, want) {
		t.Fatalf("partition sorted projectile traversal: %v want %v", target.calls, want)
	}
	v.SetSpellBolts(nil)
	target.calls = nil
	v.drawArt(target)
	if len(target.calls) != 0 {
		t.Fatalf("replaced spell snapshot retained draws: %v", target.calls)
	}
}
