package ui

import (
	"image"
	"reflect"
	"testing"
)

func TestMapMarqueeRetainsItsReleaseAcrossHUD(t *testing.T) {
	for _, size := range []image.Point{{640, 480}, {1024, 768}, {1920, 1080}} {
		for _, surface := range []string{"commands", "portrait", "minimap", "book", "pack"} {
			for _, back := range []bool{false, true} {
				t.Run(size.String()+"/"+surface+"/"+map[bool]string{false: "release-on-HUD", true: "return-to-map"}[back], func(t *testing.T) {
					a, v := inspectionFixture(t, size)
					v.SetLocalOwner(1)
					v.SetEntities([]MapEntity{{ID: 1, Owner: 1, Life: LifeAlive, Cell: image.Pt(4, 4)},
						{ID: 2, Owner: 1, Life: LifeAlive, Cell: image.Pt(6, 4)},
						{ID: 9, Owner: 2, Life: LifeAlive, Cell: image.Pt(7, 4)}})
					v.sel = selection{1}
					v.SetSpellbook(1, sbBook())
					var r image.Rectangle
					var ok bool
					switch surface {
					case "commands":
						r, ok = v.commandPanelBar()
					case "portrait":
						r, ok = v.characterPaneRect()
					case "minimap":
						var geom minimapGeom
						geom, ok = v.minimapGeometry()
						r = geom.Box
					case "book":
						if !v.hudShown(hudPanelBook) {
							v.toggleHudPanel(hudPanelBook)
						}
						r, _, ok = v.spellbookBar()
					case "pack":
						if !v.hudShown(hudPanelPack) {
							v.toggleHudPanel(hudPanelPack)
						}
						r, _, ok = v.packBar()
					}
					if !ok {
						t.Fatal("missing HUD fixture")
					}
					end := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
					start := image.Pt(90, 90)
					if surface == "minimap" {
						start.Y = 300
					}
					if !v.groundSurfaceCaptures(end.X, end.Y) {
						t.Fatalf("%v is not HUD", end)
					}
					pointer := func(edge string, p image.Point) {
						x, y := marqueeWindowPoint(t, v, p)
						if err := a.HeadlessPointer(edge, x, y); err != nil {
							t.Fatal(err)
						}
					}
					beforeHud, beforeMode := v.hudHidden, v.missionMode()
					pointer("press", start)
					pointer("move", end)
					if back {
						end = image.Pt(350, 250)
						if surface == "minimap" {
							end.Y = 90
						}
						pointer("move", end)
					}
					pointer("release", end)
					if got := a.HeadlessSelection(); !reflect.DeepEqual(got, []uint32{1, 2}) {
						t.Fatalf("selected %v, want both owned units", got)
					}
					if v.hudHidden != beforeHud || v.missionMode() != beforeMode || v.held || v.invGrab || v.minimapGrab {
						t.Fatal("map marquee activated a HUD control or retained capture")
					}
				})
			}
		}
	}
}

// A downscaled window cannot address every frame pixel. These gestures have
// broad bounds, so the nearest representable pixel is sufficient.
func marqueeWindowPoint(t *testing.T, v *Viewer, p image.Point) (int, int) {
	t.Helper()
	for dy := 0; dy <= 2; dy++ {
		for dx := 0; dx <= 2; dx++ {
			if x, y, err := v.frameToWindow(p.Add(image.Pt(dx, dy)), "marquee witness"); err == nil {
				return x, y
			}
		}
	}
	t.Fatalf("no window pixel near %v", p)
	return 0, 0
}

func TestMapMarqueeFastReleaseAndShiftAcrossActiveCommand(t *testing.T) {
	for _, heldFrame := range []bool{false, true} {
		a, v := inspectionFixture(t, image.Pt(1024, 768))
		v.SetLocalOwner(1)
		v.SetEntities([]MapEntity{
			{ID: 1, Owner: 1, Life: LifeAlive, Cell: image.Pt(4, 4)},
			{ID: 2, Owner: 1, Life: LifeAlive, Cell: image.Pt(6, 4)},
			{ID: 3, Owner: 1, Life: LifeAlive, Cell: image.Pt(1, 1)},
		})
		v.sel = selection{3}
		x, y, err := a.HeadlessCommandPoint(int(commandCellAttack))
		if err != nil {
			t.Fatal(err)
		}
		step := func(pressed, released, down bool, x, y int) {
			t.Helper()
			if a.step(appInput{PrimaryPressed: pressed, PrimaryReleased: released, CursorX: x, CursorY: y,
				Viewer: Input{PrimaryDown: down, Shift: true, CursorX: x, CursorY: y}}, a.headlessAt()) {
				t.Fatal("exit")
			}
		}
		step(true, false, true, 90, 90)
		if heldFrame {
			step(false, false, true, x, y)
		}
		step(false, true, false, x, y)
		if got := a.HeadlessSelection(); !reflect.DeepEqual(got, []uint32{1, 2, 3}) {
			t.Fatalf("shift selection = %v", got)
		}
		if v.armed || v.aimed != commandNone || v.held {
			t.Fatal("marquee pressed Attack or retained capture")
		}
	}
}
