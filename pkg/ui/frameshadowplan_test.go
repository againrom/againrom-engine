package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/backdrop"
	"againrom/pkg/render/text"
)

// dialogueShadowMask is the dialogue window's shadow mask over body at at,
// scaled by scale and clipped to clip, as the one producer plans it.
func dialogueShadowMask(art *DialogFrame, body image.Rectangle, at image.Point, scale float64, clip image.Rectangle) *image.RGBA {
	return planFrameShadows(windowFrame(body, art), at, scale, clip, shadowRemap, DialogueBackdrop{}.shadowTable).mask
}

// TestFrameShadowPlanCountsEachShadowTile checks the plan against its own
// tiles: the mask counts, at each pixel, the opaque piece pixels of the
// shadow tiles over it, and the blend follows the caller's choice.
func TestFrameShadowPlanCountsEachShadowTile(t *testing.T) {
	art := painterPlanFrame(painterPlanPieceSizes[0], 3)
	for _, kind := range []frameKind{frameWindow, frameTip} {
		s := frameSpec{Kind: kind, Rect: image.Rect(10, 12, 10+200+8, 12+150+8), Art: art}
		p := planFrameShadows(s, image.Point{}, 1, image.Rect(0, 0, 640, 480), shadowOverPicture, s.Policy.shadowTable)
		if p.mask == nil || len(p.tiles) == 0 || !p.toned || p.tone != frameShadowTone || p.table == nil {
			t.Fatalf("kind %d: plan %+v lacks its mask, tiles or blend", kind, p)
		}
		want := image.NewRGBA(p.mask.Rect)
		for _, tile := range p.tiles {
			pic := art.Pieces[tile.piece]
			r := tile.bounds(pic).Intersect(want.Rect)
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					if dialogueMaskAt(pic, tile.at, x, y) {
						want.Pix[want.PixOffset(x, y)+3]++
					}
				}
			}
		}
		for i := 3; i < len(want.Pix); i += 4 {
			if want.Pix[i] != p.mask.Pix[i] {
				t.Fatalf("kind %d: mask byte %d = %d, want %d", kind, i, p.mask.Pix[i], want.Pix[i])
			}
		}
		if q := planFrameShadows(s, image.Point{}, 1, image.Rect(0, 0, 640, 480), shadowRemap, s.Policy.shadowTable); q.toned {
			t.Fatalf("kind %d: the remap blend planned a tone", kind)
		}
	}
	panel := panelFrame(image.Rect(0, 0, 50, 50), color.RGBA{A: 255}, color.RGBA{A: 255})
	asked := false
	if p := planFrameShadows(panel, image.Point{}, 1, image.Rect(0, 0, 640, 480), shadowOverPicture, func() *backdrop.Lookup { asked = true; return nil }); p.mask != nil || asked {
		t.Fatal("a panel without shadows planned a mask or asked for a table")
	}
}

// TestFrameShadowExecutorsRunOnePlan runs one remap plan through the CPU
// executor and through the GPU executor's pixel log: the two agree on every
// pixel, and the GPU executor refuses a toned plan.
func TestFrameShadowExecutorsRunOnePlan(t *testing.T) {
	room := painterPlanPicture(image.Rect(0, 0, 320, 240), 5)
	art := painterPlanFrame(painterPlanPieceSizes[1], 4)
	s := dialogueBackdropState{policy: DialogueBackdrop{Layout: backdrop.RGB555, Mode: backdrop.Reduced}}
	p := planFrameShadows(art.dialogueFrame(image.Rect(20, 16, 260, 200), s.policy), image.Pt(3, 2), 1, room.Bounds(), shadowRemap, s.shadowTable)
	cpu := image.NewRGBA(room.Rect)
	copy(cpu.Pix, room.Pix)
	p.paint(cpu, nil)
	var log pixelLog
	log.reset(room.Bounds())
	log.upload(room)
	gpu := newPainterPlanTexture(room)
	s.drawShadowPlan(gpu, &log, p, nil)
	for y := 0; y < room.Rect.Dy(); y++ {
		for x := 0; x < room.Rect.Dx(); x++ {
			if got, ok := log.value(len(log.ops), x, y); !ok || got != cpu.RGBAAt(x, y) {
				t.Fatalf("GPU log %d,%d = %v %v, CPU %v", x, y, got, ok, cpu.RGBAAt(x, y))
			}
		}
	}
	toned := p
	toned.toned, toned.tone = true, frameShadowTone
	defer func() {
		if recover() == nil {
			t.Fatal("the GPU executor ran a toned plan")
		}
	}()
	s.drawShadowPlan(gpu, &log, toned, []text.DrawCall{})
}
