package ui

import (
	"fmt"
	"image"
	"image/color"
	"testing"
	"time"

	"againrom/pkg/ui/systemclick"
)

func inventoryPairApp(t *testing.T, window time.Duration) *App {
	t.Helper()
	a := inventoryTestApp(t)
	v := a.flow.viewer
	layoutViewport(v, MenuWindowW, wornBoxFixtureH)
	v.SetEntities([]MapEntity{{ID: 7, Cell: image.Pt(1, 1)}})
	v.sel = selection{7}
	s := unequipSubject(7)
	s.Pack = []*image.RGBA{
		solidPic(8, 8, color.RGBA{R: 255, A: 255}),
		solidPic(8, 8, color.RGBA{B: 255, A: 255}),
	}
	s.PackCount = []uint32{1, 1}
	v.SetInventorySubject(s)
	a.clicks.read = func() systemclick.Setting {
		return systemclick.Setting{Time: window, Width: 4, Height: 4}
	}
	return a
}

func inventoryPairPoint(t *testing.T, v *Viewer, worn bool, cell int) image.Point {
	t.Helper()
	if worn {
		x, y := wornCellCenter(t, v, cell)
		return image.Pt(x, y)
	}
	x, y := packCellCenter(t, v, cell)
	return image.Pt(x, y)
}

func inventoryPairFrame(a *App, p image.Point, at time.Duration, press, release bool) {
	a.step(appInput{CursorX: p.X, CursorY: p.Y, PrimaryPressed: press, PrimaryReleased: release,
		Viewer: Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: press}},
		time.Unix(1_700_000_000, 0).Add(at))
}

func inventoryPairTake(v *Viewer, worn bool) (int, bool) {
	if worn {
		return v.TakeInventoryUnequip()
	}
	return v.TakeInventoryEquip()
}

func TestMissionInventoryPairingUsesSystemWindowAtAnyFrameRate(t *testing.T) {
	for _, worn := range []bool{false, true} {
		for _, fps := range []int{30, 60} {
			for _, tc := range []struct {
				name   string
				window time.Duration
				want   bool
			}{
				{"inside", 900 * time.Millisecond, true},
				{"outside", 700 * time.Millisecond, false},
			} {
				t.Run(fmt.Sprintf("worn=%v/fps=%d/%s", worn, fps, tc.name), func(t *testing.T) {
					a := inventoryPairApp(t, tc.window)
					v := a.flow.viewer
					p := inventoryPairPoint(t, v, worn, 1)
					frame := time.Second / time.Duration(fps)
					inventoryPairFrame(a, p, 0, true, false)
					inventoryPairFrame(a, p, frame, false, true)
					for at := 2 * frame; at < 800*time.Millisecond; at += frame {
						inventoryPairFrame(a, p, at, false, false)
					}
					inventoryPairFrame(a, p, 800*time.Millisecond, true, false)
					idx, ok := inventoryPairTake(v, worn)
					if want := worn && tc.want; ok != want || ok && idx != 1 {
						t.Fatalf("at second press: request=(%d,%v), want active=%v", idx, ok, want)
					}
					inventoryPairFrame(a, p, 800*time.Millisecond+frame, false, true)
					idx, ok = inventoryPairTake(v, worn)
					if want := !worn && tc.want; ok != want || ok && idx != 1 {
						t.Fatalf("at second release: request=(%d,%v), want active=%v", idx, ok, want)
					}
					if idx, ok := inventoryPairTake(v, worn); ok {
						t.Fatalf("duplicate request %d", idx)
					}
					inventoryPairFrame(a, p, 850*time.Millisecond, true, true)
					if _, ok := inventoryPairTake(v, worn); tc.want && ok {
						t.Fatal("third press reused the completed pair")
					}
				})
			}
		}
	}
}

func TestMissionInventoryPairingRestartsOnAnotherCell(t *testing.T) {
	for _, worn := range []bool{false, true} {
		for _, width := range []int{4, 2048} {
			t.Run(fmt.Sprintf("worn=%v/width=%d", worn, width), func(t *testing.T) {
				a := inventoryPairApp(t, 500*time.Millisecond)
				a.clicks.read = func() systemclick.Setting {
					return systemclick.Setting{Time: 500 * time.Millisecond, Width: width, Height: 2048}
				}
				v := a.flow.viewer
				for i, cell := range []int{0, 1, 1} {
					p := inventoryPairPoint(t, v, worn, cell)
					inventoryPairFrame(a, p, time.Duration(i)*100*time.Millisecond, true, true)
					idx, ok := inventoryPairTake(v, worn)
					if want := i == 2; ok != want || ok && idx != 1 {
						t.Fatalf("press %d: request=(%d,%v), want active=%v on cell 1", i, idx, ok, want)
					}
				}
			})
		}
	}
}

func TestMissionInventoryPairingRequiresReleaseAndSystemRectangle(t *testing.T) {
	for _, worn := range []bool{false, true} {
		for _, control := range []string{"held", "moved", "secondary", "other box"} {
			t.Run(fmt.Sprintf("worn=%v/%s", worn, control), func(t *testing.T) {
				a := inventoryPairApp(t, 500*time.Millisecond)
				v := a.flow.viewer
				p := inventoryPairPoint(t, v, worn, 1)
				inventoryPairFrame(a, p, 0, true, control != "held")
				switch control {
				case "moved":
					p.X += 3
				case "secondary":
					a.step(appInput{CursorX: p.X, CursorY: p.Y, SecondaryPressed: true}, time.Unix(1_700_000_000, 0).Add(50*time.Millisecond))
				case "other box":
					other := inventoryPairPoint(t, v, !worn, 1)
					inventoryPairFrame(a, other, 50*time.Millisecond, true, true)
				}
				inventoryPairFrame(a, p, 100*time.Millisecond, true, false)
				inventoryPairFrame(a, p, 110*time.Millisecond, false, true)
				if idx, ok := inventoryPairTake(v, worn); ok {
					t.Fatalf("unpaired press raised request %d", idx)
				}
			})
		}
	}
}
