package ui

import (
	"math/rand"
	"testing"

	"againrom/pkg/render/terrain"
)

func TestPlayableCellRectEdgeReadMatchesFullScan(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	for trial := 0; trial < 3000; trial++ {
		w, h := 1+rng.Intn(14), 1+rng.Intn(14)
		block := make([]uint8, w*h)
		switch trial % 6 {
		case 0:
			l, r, tp, b := rng.Intn(4), rng.Intn(4), rng.Intn(4), rng.Intn(4)
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					if x < l || x >= w-r || y < tp || y >= h-b {
						block[y*w+x] = borderBit
					}
				}
			}
		case 1:
			for y := 0; y < h; y++ {
				l, r := rng.Intn(w+1), rng.Intn(w+1)
				for x := 0; x < w; x++ {
					if x < l || x >= w-r {
						block[y*w+x] = borderBit
					}
				}
			}
		case 2:
			for i := range block {
				if rng.Intn(5) == 0 {
					block[i] = borderBit
				}
			}
		case 3:
			for i := range block {
				block[i] = 0x01
			}
		case 4:
			if w > 2 && h > 2 {
				block[(1+rng.Intn(h-2))*w+1+rng.Intn(w-2)] = borderBit
			}
		case 5:
			for i := range block {
				block[i] = borderBit
			}
			if rng.Intn(2) == 0 {
				block = block[:rng.Intn(len(block)+1)]
			}
		}
		v := &Viewer{grid: terrain.Grid{Width: w, Height: h, Block: block}}
		wantRect, wantOK := v.playableCellRectScan()
		gotRect, gotOK := v.playableCellRect()
		if gotOK != wantOK || gotRect != wantRect {
			t.Fatalf("trial %d (%dx%d, block %v): got %v %v, full scan %v %v", trial, w, h, block, gotRect, gotOK, wantRect, wantOK)
		}
	}
}
