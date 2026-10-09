package ui

import (
	"image"
	"reflect"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestAttackCursorAddsNoTargetStrokeToMissionFrame(t *testing.T) {
	for _, zoom := range []float64{0.5, 1, 2} {
		t.Run(strconv.FormatFloat(zoom, 'g', -1, 64), func(t *testing.T) {
			a, v, _ := atOnMap(t)
			v.cam.Zoom = zoom
			v.SetTextSmoothing(false)
			v.DeferPointer(true)
			for i := range v.entities {
				v.entities[i].Hostile = v.entities[i].ID == atFoeID
			}
			cols, rows := int(v.grid.Width), int(v.grid.Height)
			fog := make([]byte, cols*rows)
			for i := range fog {
				fog[i] = FogVisible
			}
			v.SetFog(fog, cols, rows)
			x, y := ptHover(a, v, atFoeCol, atFoeRow)
			if _, id, hit := v.hoverMask(x, y); !hit || id != atFoeID {
				t.Fatalf("visible enemy hover = %d/%v", id, hit)
			}
			dst := ebiten.NewImage(v.frameW, v.frameH)
			t.Cleanup(dst.Dispose)
			unknowns := func() []image.Rectangle {
				v.Draw(dst)
				var rects []image.Rectangle
				for _, op := range v.canvasLog.ops {
					if op.kind == pixelUnknown {
						rects = append(rects, op.rect)
					}
				}
				return rects
			}
			want := unknowns()
			a.step(afHeld(x, y), atAt)
			if _, _, shown := v.attackPointerPresent(); !shown {
				t.Fatal("attack cursor is absent")
			}
			if got := unknowns(); !reflect.DeepEqual(got, want) {
				t.Fatalf("attack cursor adds a stroke to the composed frame: unknown regions %v, want %v", got, want)
			}
		})
	}
}
