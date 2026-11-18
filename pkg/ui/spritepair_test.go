package ui

import (
	"image"
	"image/color"
	"math"
	"reflect"
	"testing"

	"againrom/pkg/render/terrain"
)

func attachSibling(t *testing.T, base any, boundary *terrain.StaticFrame) {
	t.Helper()
	f := reflect.ValueOf(base).Elem().FieldByName("Boundary")
	if !f.IsValid() {
		t.Fatal("frame does not carry its paired boundary")
	}
	f.Set(reflect.ValueOf(boundary))
}

func TestSpritePairsReachEveryBodyPass(t *testing.T) {
	for _, family := range []string{"unit", "hero", "object", "structure", "flat structure", "sack"} {
		for _, mirror := range []bool{false, true} {
			t.Run(family+map[bool]string{false: "/plain", true: "/mirror"}[mirror], func(t *testing.T) {
				v := sackIdentityViewer(t)
				base, boundary := sackFrame(8, 8), sackFrame(10, 9)
				cell := image.Pt(2, 2)
				v.showStaticArt, v.showStructureArt = true, true
				switch family {
				case "unit", "hero":
					e := withFrame(cell, entityArtA())
					e.Frame, e.Boundary, e.Mirror = base, boundary, mirror
					e.PlayerCharacter = family == "hero"
					v.SetEntities([]MapEntity{e})
				case "object":
					attachSibling(t, base, boundary)
					v.staticsFlat = []terrain.StaticPlacement{{Cell: cell, TopLeft: image.Pt(64, 64), Frame: base, Mirror: mirror}}
				case "structure", "flat structure":
					attachSibling(t, base, boundary)
					v.structuresFlat = []terrain.StructurePlacement{{Cell: cell, TopLeft: image.Pt(64, 64), Frame: base, Class: &terrain.StructureClass{Flat: family == "flat structure"}}}
				case "sack":
					v.SetSackFrames([]*terrain.StaticFrame{base})
					v.SetSackBoundaries([]*terrain.StaticFrame{boundary})
					v.SetSacks([]MapSack{{Cell: cell}})
				}
				v.planeOrder = terrain.PlaneOrder(v.structuresFlat, v.staticsFlat)
				for _, smoothing := range []bool{false, true, false} {
					v.SetGraphicsOptions(GraphicsOptions{Smoothing: smoothing})
					var rec graphicsAlphaRecorder
					if family == "flat structure" {
						v.drawStructuresFlat(&rec)
					} else {
						v.drawPlane(&rec)
					}
					want := graphicsAlphaRecorder{1}
					if smoothing {
						want = append(want, 0.5)
					}
					if !reflect.DeepEqual(rec, want) {
						t.Fatalf("smoothing %v: draw alpha %v, want %v", smoothing, rec, want)
					}
					if smoothing && family != "flat structure" {
						pair := v.planeSprites()
						if pair[0].Frame != base || pair[1].Frame != boundary {
							t.Fatal("base/sibling order changed")
						}
						if pair[0].Y != pair[1].Y || pair[0].Brightness != pair[1].Brightness || pair[0].ShadeZero != pair[1].ShadeZero || pair[0].Stone != pair[1].Stone || pair[0].Tint != pair[1].Tint {
							t.Fatal("sibling lost the base look")
						}
						g0, g1 := spriteGeoM(pair[0], v.cam.Zoom), spriteGeoM(pair[1], v.cam.Zoom)
						l0, y0 := g0.Apply(0, 0)
						r0, _ := g0.Apply(float64(base.Width), 0)
						l1, y1 := g1.Apply(0, 0)
						r1, _ := g1.Apply(float64(boundary.Width), 0)
						if l0+r0 != l1+r1 || y0 != y1 {
							t.Fatalf("sprite centres differ: %v,%v / %v,%v", (l0+r0)/2, y0, (l1+r1)/2, y1)
						}
					}
				}
				if family == "unit" || family == "hero" {
					v.entities[0].Boundary = nil
				} else if family == "sack" {
					v.SetSackBoundaries(nil)
				} else {
					attachSibling(t, base, nil)
				}
				v.SetGraphicsOptions(GraphicsOptions{Smoothing: true})
				var missing graphicsAlphaRecorder
				if family == "flat structure" {
					v.drawStructuresFlat(&missing)
				} else {
					v.drawPlane(&missing)
				}
				if !reflect.DeepEqual(missing, graphicsAlphaRecorder{1}) {
					t.Fatal("missing sibling changed the base draw", missing)
				}
			})
		}
	}
}

func TestShotSiblingUsesTheSharedHalfAlphaDraw(t *testing.T) {
	v := sackIdentityViewer(t)
	f := &terrain.EffectFrame{Width: 8, Height: 8}
	boundary := sackFrame(10, 9)
	attachSibling(t, f, boundary)
	v.SetSpellBolts([]SpellBolt{{Cell: image.Pt(2, 2), To: image.Pt(2, 2), Pos: image.Pt(2*ShotScale, 2*ShotScale), Sheet: &terrain.EffectSheet{Frames: []*terrain.EffectFrame{f}}, Mirror: true}})
	for _, on := range []bool{true, false} {
		v.SetGraphicsOptions(GraphicsOptions{Smoothing: on})
		var rec graphicsAlphaRecorder
		v.drawSpellArt(&rec, v.spellArtPlacements(), SpellProjectiles)
		want := graphicsAlphaRecorder{1}
		if on {
			want = append(want, 0.5)
		}
		if !reflect.DeepEqual(rec, want) {
			t.Fatalf("smoothing %v: %v want %v", on, rec, want)
		}
	}
	v.SetSpellBolts([]SpellBolt{{Cell: image.Pt(2, 2), To: image.Pt(2, 2), Pos: image.Pt(-ShotScale/4, 2*ShotScale), AbsolutePosition: true, Sheet: &terrain.EffectSheet{Frames: []*terrain.EffectFrame{f}}}})
	v.SetGraphicsOptions(GraphicsOptions{Smoothing: true})
	var edge graphicsAlphaRecorder
	v.drawSpellArt(&edge, v.spellArtPlacements(), SpellProjectiles)
	if !reflect.DeepEqual(edge, graphicsAlphaRecorder{0.5}) {
		t.Fatal("visible shot sibling was culled with its base", edge)
	}
}

func TestSpritePairKeepsTheBaseLookAndMirrorPivot(t *testing.T) {
	v := sackIdentityViewer(t)
	v.SetGraphicsOptions(GraphicsOptions{Smoothing: true})
	base, boundary := spriteLightFrame(8, 8), spriteLightFrame(10, 9)
	for _, mirror := range []bool{false, true} {
		for _, stone := range []bool{false, true} {
			for _, translucent := range []bool{false, true} {
				s := staticScreenRect{screenRect: screenRect{X: 64, Y: 64, W: 8, H: 8}, Frame: base, Mirror: mirror, Stone: stone, Translucent: translucent, Brightness: 0.75, ShadeZero: true, Tint: spriteTint{On: true, Color: color.RGBA{R: 75, A: 255}}}
				pair := v.spritePair(s, boundary)
				if len(pair) != 2 {
					t.Fatal("lost pair")
				}
				var rec staticRecorder
				for _, p := range pair {
					v.drawPlaneSprite(&rec, p)
				}
				if len(rec.imgs) != 2 || rec.imgs[0] != v.tintedImage(base, stone, s.Tint.Color, s.Tint.Focus, 0) || rec.imgs[1] != v.tintedImage(boundary, stone, s.Tint.Color, s.Tint.Focus, 0) {
					t.Fatal("sibling changed shade, stone or inspection tint")
				}
				if drawnSpan(rec.geo[0], 8, mirror) != drawnSpan(rec.geo[1], 10, mirror) {
					t.Fatal("mirror pivot differs: the sibling is not centred on its body")
				}
				if pair[1].Brightness != s.Brightness || pair[1].ShadeZero != s.ShadeZero {
					t.Fatal("brightness or hover shade changed")
				}
			}
		}
	}
}

func TestSpritePairCullsItsFramesSeparately(t *testing.T) {
	v := sackIdentityViewer(t)
	v.SetGraphicsOptions(GraphicsOptions{Smoothing: true})
	base, boundary := sackFrame(8, 8), sackFrame(10, 9)
	got := v.spritePair(staticScreenRect{screenRect: screenRect{X: -8, Y: 64, W: 8, H: 8}, Frame: base}, boundary)
	if len(got) != 1 || got[0].Frame != boundary || got[0].X != -9 {
		t.Fatal("visible boundary was lost with its culled base", got)
	}
}

// drawnSpan is the horizontal centre, doubled, and the top of a sprite drawn
// w art pixels wide through the recorded transform geo.
func drawnSpan(geo [6]float64, w int, mirror bool) [2]float64 {
	left := geo[2]
	if mirror {
		left += float64(w) * geo[0]
	}
	return [2]float64{2*left + float64(w)*math.Abs(geo[0]), geo[5]}
}

// A sibling of another size is centred by its own size on its body's anchor,
// at the camera's zoom and in either reflection.
func TestSpritePairCentresASiblingOfAnotherSizeOnTheBodyAnchor(t *testing.T) {
	v := sackIdentityViewer(t)
	v.SetGraphicsOptions(GraphicsOptions{Smoothing: true})
	v.Camera().Zoom = 1.5
	for _, c := range []struct {
		bw, bh, sw, sh int
		dx, dy         float64
	}{{40, 72, 48, 72, -4, 0}, {32, 56, 32, 64, 0, -4}, {32, 56, 32, 56, 0, 0}, {8, 8, 11, 9, -1, 0}} {
		for _, mirror := range []bool{false, true} {
			base := staticScreenRect{screenRect: screenRect{X: 16, Y: 16, W: float64(c.bw) * 1.5, H: float64(c.bh) * 1.5}, Frame: sackFrame(c.bw, c.bh), Mirror: mirror}
			pair := v.spritePair(base, sackFrame(c.sw, c.sh))
			if len(pair) != 2 {
				t.Fatal("lost pair")
			}
			s := pair[1].screenRect
			want := screenRect{X: 16 + c.dx*1.5, Y: 16 + c.dy*1.5, W: float64(c.sw) * 1.5, H: float64(c.sh) * 1.5}
			if s != want || pair[1].Mirror != mirror {
				t.Fatalf("%dx%d body, %dx%d sibling, mirror %v: sibling at %+v, want %+v", c.bw, c.bh, c.sw, c.sh, mirror, s, want)
			}
		}
	}
}
