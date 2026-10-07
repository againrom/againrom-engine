package ui

import (
	"fmt"
	"image"
	"image/color"
	"testing"
	"time"
)

func TestFramedLoadSelectScrollConfirmCancelAndExactToken(t *testing.T) {
	a := newTestApp(t, nil, nil)
	a.Layout(640, 480)
	var loaded string
	removed := 0
	rows := make([]SaveEntry, 27)
	for i := range rows {
		rows[i] = SaveEntry{Name: fmt.Sprintf("file-%d.ags", i), Label: fmt.Sprintf("Saved game %d", i)}
	}
	a.SetSaveSeams(nil, func() []SaveEntry { return rows }, func(name string) (MapOpener, bool, error) {
		loaded = name
		return nil, false, fmt.Errorf("read refused")
	})
	a.SetSaveDelete(func(string) bool { return true }, func(string) (func() error, error) { return func() error { removed++; return nil }, nil })
	a.flow.openLoad(ScreenMenu)
	click := func(x, y int) {
		a.stepLoadWindow(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, time.Time{})
		a.stepLoadWindow(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, time.Time{})
	}
	a.stepLoadWindow(appInput{End: true}, time.Time{})
	if a.flow.loadList.Selection() != 26 {
		t.Fatal("End did not reach last save")
	}
	top, n := a.flow.loadList.Visible()
	if top != 17 || n != 10 {
		t.Fatalf("visible=%d,%d", top, n)
	}
	click(140, 158)
	if a.flow.loadList.Selection() != 17 || loaded != "" {
		t.Fatal("row click did not only select the visible save")
	}
	click(200, 392)
	if loaded != "file-17.ags" {
		t.Fatal("OK lost the exact selected token", loaded)
	}
	click(324, 392)
	if !a.flow.loadUI.confirm || removed != 0 {
		t.Fatal("Delete bypassed confirmation")
	}
	click(448, 392)
	if a.flow.loadUI.confirm || a.Screen() != ScreenLoad || removed != 0 {
		t.Fatal("Cancel did not retain save and list")
	}
	a.stepLoadWindow(appInput{Delete: true}, time.Time{})
	a.stepLoadWindow(appInput{Enter: true}, time.Time{})
	if removed != 1 || a.flow.loadUI.confirm {
		t.Fatal("confirmed delete was not performed once")
	}
	click(448, 392)
	if a.Screen() != ScreenMenu {
		t.Fatal("Cancel did not return to opener")
	}
}

func TestFramedLoadRowsAbove100CannotActivateControls(t *testing.T) {
	for _, index := range []int{99, 100, 101, 102, 103, 104, 105, 106, 119} {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			a := newTestApp(t, nil, nil)
			a.Layout(640, 480)
			rows := make([]SaveEntry, 120)
			for i := range rows {
				rows[i] = SaveEntry{Name: fmt.Sprintf("file-%d.ags", i), Label: fmt.Sprintf("Saved game %d", i)}
			}
			var loaded, prepared string
			removed := 0
			a.SetSaveSeams(nil, func() []SaveEntry { return rows }, func(name string) (MapOpener, bool, error) {
				loaded = name
				return nil, false, fmt.Errorf("read refused")
			})
			a.SetSaveDelete(func(string) bool { return true }, func(name string) (func() error, error) {
				prepared = name
				return func() error { removed++; return nil }, nil
			})
			a.flow.openLoad(ScreenMenu)
			last := 109
			if index == 99 {
				last = 108
			} else if index == 119 {
				last = 119
			}
			a.flow.loadList.Select(last)
			top, _ := a.flow.loadList.Visible()
			click := func(x, y int) {
				a.stepLoadWindow(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, time.Time{})
				a.stepLoadWindow(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, time.Time{})
			}
			click(140, 158+19*(index-top))
			if a.Screen() != ScreenLoad || a.flow.loadList.Selection() != index || loaded != "" || prepared != "" || removed != 0 || a.flow.loadUI.confirm {
				t.Fatalf("row %d acted as a control: screen=%v selection=%d loaded=%q prepared=%q removed=%d confirm=%v", index, a.Screen(), a.flow.loadList.Selection(), loaded, prepared, removed, a.flow.loadUI.confirm)
			}
			click(200, 392)
			if loaded != rows[index].Name || prepared != "" || removed != 0 {
				t.Fatalf("OK lost selected row: loaded=%q prepared=%q removed=%d", loaded, prepared, removed)
			}
		})
	}
}

func TestCrystalMinimapKeepsTextureUnderUnseenCellsAndOwnsItsClicks(t *testing.T) {
	v := newStaticsViewer(t, staticsBundle(), true, true)
	crystal := image.NewRGBA(image.Rect(0, 0, 160, 158))
	for y := 0; y < 158; y++ {
		for x := 0; x < 160; x++ {
			crystal.SetRGBA(x, y, color.RGBA{0, uint8(x + y), 3, 255})
		}
	}
	v.SetDialogFrame(&DialogFrame{Minimap: crystal})
	v.SetFog(make([]byte, cliffW*cliffH), cliffW, cliffH)
	pic, _, ok := v.minimapPresent()
	if !ok {
		t.Fatal("missing crystal")
	}
	if pic.Bounds().Size() != image.Pt(160, 158) || pic.RGBAAt(2, 2) != crystal.RGBAAt(2, 2) {
		t.Fatal("frame was stretched or replaced")
	}
	g, ok := v.minimapGeometry()
	if !ok || !g.Content.In(g.Box.Inset(8)) {
		t.Fatal("map crosses the bevel", g)
	}
	halfCell := minimapCellPixel(1, g.Num, g.Den) / 2
	x, y := g.Content.Min.X+halfCell, g.Content.Min.Y+halfCell
	cell, inside := v.minimapCellAt(x, y)
	if !inside || cell != (image.Point{}) {
		t.Fatalf("content hit=%v,%v", cell, inside)
	}
	if _, inside := v.minimapCellAt(g.Box.Min.X+2, g.Box.Min.Y+2); inside {
		t.Fatal("crystal bevel issued a map click")
	}
}
