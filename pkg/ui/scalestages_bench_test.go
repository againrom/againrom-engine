package ui

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// BenchmarkMissionFrameDraw times what Viewer.Draw submits for one mission
// frame on a 2560 by 1440 window, at zoom 1 and zoomed in and out, with
// Smoothing on and off. It measures the draw calls' CPU cost; no GPU runs.
func BenchmarkMissionFrameDraw(b *testing.B) {
	for _, z := range []float64{1, 1.2, 1 / 1.2} {
		for _, smoothing := range []bool{true, false} {
			b.Run(fmt.Sprintf("zoom%.2f/smoothing=%v", z, smoothing), func(b *testing.B) {
				v, err := NewViewer("zoom", grid(60, 60), zoomTileset(false))
				if err != nil {
					b.Fatal(err)
				}
				v.SetGraphicsOptions(GraphicsOptions{Smoothing: smoothing})
				v.Layout(2560, 1440)
				v.cam.SetZoom(z)
				v.cam.X, v.cam.Y = 300.4, 280.7
				screen := ebiten.NewImage(2560, 1440)
				v.Draw(screen)
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					v.Draw(screen)
				}
			})
		}
	}
}
