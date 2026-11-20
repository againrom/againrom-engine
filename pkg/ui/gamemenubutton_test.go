package ui

import (
	"image"
	"image/color"
	"testing"
)

// edgeStrip is the top pixel line of r, which a framed button paints and a
// bare text row leaves as the panel's own ground.
func edgeStrip(pic *image.RGBA, r image.Rectangle) []color.RGBA {
	var out []color.RGBA
	for x := r.Min.X; x < r.Max.X; x++ {
		out = append(out, pic.RGBAAt(x, r.Min.Y))
	}
	return out
}

func sameStrip(a, b []color.RGBA) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestModActionRowIsAFramedButtonLikeItsNeighbours(t *testing.T) {
	profiles := map[string]GameMenuProfile{
		"plain":  {},
		"native": {NativeMissionWindows: true, WindowsCloseMenu: true},
	}
	for name, profile := range profiles {
		t.Run(name, func(t *testing.T) {
			var asked []NoticeAction
			var again int
			a := actionApp(t, true, NoticeToTown, &asked, &again)
			a.SetGameMenuProfile(profile)
			a.flow.menuFont = chargenTestFont()
			a.step(appInput{Escape: true}, atAt)
			rows := endMissionRows(a)
			abandon := -1
			for i, r := range rows {
				if r.Action == gameMenuModActionConfirm {
					abandon = i
				}
			}
			if abandon < 0 || abandon+1 >= len(rows) {
				t.Fatalf("rows %+v have no abandon row with a neighbour", rows)
			}
			if !rows[abandon].button() || !rows[abandon+1].button() {
				t.Fatalf("abandon row button %v, neighbour button %v", rows[abandon].button(), rows[abandon+1].button())
			}
			panel := gameMenuPanelRectFor(a.flow.menuPanelSurface(), rows, a.flow.menuGeometryProfile())
			pic := composeGameMenuPanel(a.flow.menuFont, a.flow.menuPanelSurface(), rows, a.flow.menuList, a.flow.menuArt, gameMenuPointer{}, a.flow.menuGeometryProfile())
			ra, rn := gameMenuRowRect(panel, abandon), gameMenuRowRect(panel, abandon+1)
			if ra.Dx() != rn.Dx() || ra.Dy() != rn.Dy() {
				t.Fatalf("abandon rectangle %v, neighbour %v", ra, rn)
			}
			ground := edgeStrip(pic, image.Rect(ra.Min.X, ra.Min.Y-3, ra.Max.X, ra.Min.Y-2))
			if sameStrip(edgeStrip(pic, rn), ground) {
				t.Fatal("the neighbour row shows no frame, so the comparison proves nothing")
			}
			if !sameStrip(edgeStrip(pic, ra), edgeStrip(pic, rn)) {
				t.Fatal("the abandon row is not framed like its neighbour")
			}
			a.flow.menuList.Select(abandon)
			a.chooseGameMenu()
			if len(asked) != 1 || asked[0] != NoticeAbandon {
				t.Fatalf("pressing the abandon row asked %v", asked)
			}
		})
	}
}
