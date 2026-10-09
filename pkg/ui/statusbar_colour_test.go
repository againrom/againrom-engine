package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"

	"againrom/pkg/render/terrain"
)

func statusBarPixel(t *testing.T, v *Viewer, img *image.RGBA, x, y int) color.RGBA {
	t.Helper()
	r, ok := v.placeArm(image.Pt(2, 2), image.Rect(64, 48, 96, 52))
	if !ok {
		t.Fatal("bar is outside the camera")
	}
	return img.RGBAAt(int(math.Round(r.X))+x, int(math.Round(r.Y))+y)
}

func TestStatusBarHealthIntegerThresholdsAndRows(t *testing.T) {
	for _, tc := range []struct {
		hp   int
		r, g bool
	}{{24, true, false}, {25, true, true}, {49, true, true}, {50, false, true}} {
		t.Run(fmt.Sprint(tc.hp), func(t *testing.T) {
			v := commandViewer(t)
			v.SetEntities([]MapEntity{{ID: 1, Cell: image.Pt(2, 2), Life: LifeAlive, HP: tc.hp, MaxHP: 101}})
			v.sel = selection{1}
			img := sbCompose(v, 800, 600, sbGround)
			for y := range 4 {
				want := color.RGBA{A: 255}
				if tc.r {
					want.R = [4]uint8{131, 255, 197, 131}[y]
				}
				if tc.g {
					want.G = [4]uint8{129, 255, 194, 129}[y]
				}
				if got := statusBarPixel(t, v, img, 4, y); got != want {
					t.Errorf("HP %d/101 row %d = %v, want %v", tc.hp, y, got, want)
				}
			}
		})
	}
}

func TestStatusBarSelectedGreyAndShowHealthRemainder(t *testing.T) {
	for _, selected := range []bool{true, false} {
		t.Run(fmt.Sprint(selected), func(t *testing.T) {
			v := commandViewer(t)
			v.SetEntities([]MapEntity{{ID: 1, Cell: image.Pt(2, 2), Life: LifeAlive, HP: 25, MaxHP: 101}})
			if selected {
				v.sel = selection{1}
			}
			img := sbCompose(v, 800, 600, sbGround)
			for y := range 4 {
				want := sbGround
				if selected {
					rb, g := [4]uint8{65, 131, 98, 65}[y], [4]uint8{64, 129, 97, 64}[y]
					want = color.RGBA{rb, g, rb, 255}
				}
				if got := statusBarPixel(t, v, img, 16, y); got != want {
					t.Errorf("selected %v remainder row %d = %v, want %v", selected, y, got, want)
				}
			}
			if !selected {
				if got, want := statusBarPixel(t, v, img, 4, 1), (color.RGBA{148, 149, 16, 255}); got != want {
					t.Errorf("show-health yellow blend = %v, want %v", got, want)
				}
			}
		})
	}
}

func TestStatusBarPoolMinimumAndOverfullLength(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		kind                 terrain.StatusBarKind
		value, maximum, want int
	}{{"HP zero", terrain.HealthBar, 0, 100, 0}, {"HP one", terrain.HealthBar, 1, 100, 1},
		{"mana zero", terrain.ManaBar, 0, 100, 1}, {"overfull HP", terrain.HealthBar, 150, 100, 36},
		{"overfull mana", terrain.ManaBar, 150, 100, 36}} {
		t.Run(tc.name, func(t *testing.T) {
			v := commandViewer(t)
			e := MapEntity{ID: 1, Cell: image.Pt(2, 2), Life: LifeAlive}
			if tc.kind == terrain.HealthBar {
				e.HP, e.MaxHP = tc.value, tc.maximum
			} else {
				e.Mana, e.MaxMana = tc.value, tc.maximum
			}
			v.SetEntities([]MapEntity{e})
			bars := v.statusBars()
			if len(bars) != 1 || bars[0].Fill != tc.want {
				t.Fatalf("bars = %+v, want one with N=%d", bars, tc.want)
			}
			if tc.want > 24 {
				y := 1
				want := color.RGBA{G: 255, A: 255}
				if tc.kind == terrain.ManaBar {
					y, want = 5, color.RGBA{B: 255, A: 255}
				}
				v.sel = selection{1}
				img := sbCompose(v, 800, 600, sbGround)
				if got := statusBarPixel(t, v, img, 35, y); got != want {
					t.Errorf("overfull pixel = %v, want %v", got, want)
				}
				for row := range 4 {
					want := color.RGBA{G: [4]uint8{129, 255, 194, 129}[row], A: 255}
					offset := row
					if tc.kind == terrain.ManaBar {
						want = color.RGBA{B: [4]uint8{131, 255, 197, 131}[row], A: 255}
						offset += 4
					}
					for x := 28; x < 32; x++ {
						if got := statusBarPixel(t, v, img, x, offset); got != want {
							t.Errorf("overfull right cap x=%d row=%d got=%v want=%v", x, row, got, want)
						}
					}
				}
			}
		})
	}
}

func TestStatusBarUsesAOneCellClassSelectionWidth(t *testing.T) {
	v := commandViewer(t)
	v.SetEntities([]MapEntity{{ID: 1, Cell: image.Pt(2, 2), Life: LifeAlive, HP: 50, MaxHP: 100,
		Art: &terrain.UnitClass{CenterX: 64, CenterY: 78, TileSize: 1, Selection: image.Rect(40, 40, 88, 88)}}})
	bars := v.statusBars()
	if len(bars) != 1 || bars[0].Rect.Dx() != 48 || bars[0].Fill != 20 {
		t.Fatalf("bars = %+v, want class width 48 and N=20", bars)
	}
}

func TestStatusBarManaStaysBlueAtEveryRatio(t *testing.T) {
	for _, mana := range []int{0, 1, 24, 25, 49, 50, 101} {
		t.Run(fmt.Sprint(mana), func(t *testing.T) {
			v := commandViewer(t)
			v.SetEntities([]MapEntity{{ID: 1, Cell: image.Pt(2, 2), Life: LifeAlive, Mana: mana, MaxMana: 101}})
			v.sel = selection{1}
			img := sbCompose(v, 800, 600, sbGround)
			for y, intensity := range []uint8{131, 255, 197, 131} {
				if got, want := statusBarPixel(t, v, img, 4, 4+y), (color.RGBA{B: intensity, A: 255}); got != want {
					t.Errorf("mana %d/101 row %d = %v, want %v", mana, y, got, want)
				}
			}
		})
	}
}
