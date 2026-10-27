package ui

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/render/frame"
	"againrom/pkg/render/textfit"
)

func nativeMenuApp(t *testing.T, size image.Point) *App {
	t.Helper()
	a := newTestApp(t, appRows(1), okLoader(t))
	a.flow = openMissionMenu(t)
	a.flow.menuFont = chargenTestFont()
	a.flow.viewer = backdropViewer(t)
	a.SetGameMenuProfile(GameMenuProfile{NativeMissionWindows: true, MissionBackdrop: color.RGBA{A: 64}, WindowsCloseMenu: true})
	a.Layout(size.X, size.Y)
	a.flow.setMenuUp(true)
	return a
}

func TestMissionMenuNativePlacementAndControlCentres(t *testing.T) {
	for _, size := range []image.Point{{1024, 768}, {1920, 1080}, {640, 480}} {
		for _, page := range []gameMenuPage{gameMenuRoot, gameMenuGameOptionsPage, gameMenuSoundOptionsPage, gameMenuQuestObjectivesPage, gameMenuEndQuestConfirmation, gameMenuAbortGameConfirmation, gameMenuDiplomacyPage, gameMenuModActionConfirmation} {
			a := nativeMenuApp(t, size)
			a.flow.rebuildGameMenu(page, 0)
			if page == gameMenuGameOptionsPage {
				a.flow.gameOptions.Read = func(bool) GameOptionValues { return GameOptionValues{} }
			}
			if page == gameMenuSoundOptionsPage {
				a.flow.soundOptions.Read = func() audio.ChannelVolumes { return audio.ChannelVolumes{} }
			}
			place, at := a.missionMenuPlacement()
			rect := a.missionMenuRect().Add(at)
			if place.Scale() != a.flow.viewer.place.Scale() || rect.Min.X != (a.flow.viewer.frameW-rect.Dx())/2 || rect.Min.Y != (a.flow.viewer.frameH-rect.Dy())/2 {
				t.Fatalf("size %v page %v: scale %v rect %v", size, page, place.Scale(), rect)
			}
			if page == gameMenuRoot && rect.Size() != image.Pt(296, 296) {
				t.Fatalf("menu frame %v", rect)
			}
			if (page == gameMenuEndQuestConfirmation || page == gameMenuAbortGameConfirmation) && rect.Size() != image.Pt(296, 232) {
				t.Fatalf("confirmation frame %v", rect)
			}
			var control image.Rectangle
			switch page {
			case gameMenuGameOptionsPage:
				control = gameOptionRect(gameMenuOptionsCancel)
			case gameMenuSoundOptionsPage:
				control = soundOptionRect(gameMenuMusicPlay)
			case gameMenuQuestObjectivesPage:
				control = questButtonRect
			}
			if !control.Empty() {
				centre := control.Min.Add(control.Size().Div(2))
				ox, oy := place.Origin()
				x := int(ox + float64(centre.X+at.X)*place.Scale())
				y := int(oy + float64(centre.Y+at.Y)*place.Scale())
				got, hit := a.windowToNativeFrame(x, y)
				if !hit || !got.In(control) {
					t.Fatalf("page %v drawn control centre misses %v: %v", page, control, got)
				}
			}
			if page == gameMenuRoot || page == gameMenuEndQuestConfirmation || page == gameMenuAbortGameConfirmation {
				row := gameMenuRowRect(gameMenuPanelRect(a.flow.menuPanelSurface(), a.flow.pages.menu), 0)
				p := row.Min.Add(row.Size().Div(2))
				x, y, ok := a.nativeFrameToWindow(p)
				if !ok {
					t.Fatal("control centre not represented")
				}
				a.pointer = image.Pt(x, y)
				got, ok := a.pointerFrame()
				if !ok || got != p {
					t.Fatalf("hover %v %v want %v", got, ok, p)
				}
				hit, ok := a.gameMenuButtonAt(appInput{CursorX: x, CursorY: y})
				if !ok || hit != 0 {
					t.Fatalf("click at drawn centre hits %d %v", hit, ok)
				}
			}
		}
	}
}

func TestMissionSaveLoadUseTheSharedNativeHost(t *testing.T) {
	fake := &pageSeamsFake{entries: []SaveEntry{{Name: "game0000.sav", Label: "Alpha"}}}
	a := lookApp(t, fake)
	a.flow.viewer = backdropViewer(t)
	a.flow.pages.save.page.Panel = image.Rect(82, 48, 558, 392)
	a.flow.pages.loadSet.Panel = image.Rect(82, 48, 558, 392)
	for i := range 3 {
		a.flow.pages.save.page.Buttons[i] = PageButton{Rect: image.Rect(154+i*122, 318, 250+i*122, 342), Caption: "Button"}
		a.flow.pages.loadSet.Buttons[i] = PageButton{Rect: image.Rect(154+i*122, 338, 250+i*122, 362), Caption: "Button"}
	}
	a.SetGameMenuProfile(GameMenuProfile{NativeMissionWindows: true, MissionBackdrop: color.RGBA{A: 64}, WindowsCloseMenu: true})
	a.Layout(1024, 768)
	a.flow.setMenuUp(true)
	for _, screen := range []Screen{ScreenSave, ScreenLoad} {
		a.flow.screen = screen
		a.flow.loadBack = ScreenGameMenu
		if screen == ScreenLoad {
			a.flow.applyGameMenuAction(gameMenuLoad)
		}
		if !a.flow.viewer.popupOpen() {
			t.Fatal("page no longer holds mission popup")
		}
		if screen == ScreenLoad {
			if err := a.HeadlessPointer("hover", 701, 401); err != nil {
				t.Fatal(err)
			}
			if v := a.flow.viewer; v.cursorX != 701 || v.cursorY != 401 {
				t.Fatalf("Load left the mission pointer at %d,%d", v.cursorX, v.cursorY)
			}
		}
		for _, confirm := range []bool{false, true} {
			v := a.flow.loadView()
			if screen == ScreenSave {
				v = a.flow.pages.save.view()
			}
			v.ui.confirm = confirm
			if !a.flow.mapShowing() {
				t.Fatalf("%v confirmation %v hides mission", screen, confirm)
			}
			pix, err := a.composePage(v)
			if err != nil {
				t.Fatal(err)
			}
			if got := pix.RGBAAt(0, 0); got.A != 0 {
				t.Fatalf("main-menu wall remains: %v", got)
			}
			place, at := a.missionMenuPlacement()
			rect := a.missionMenuRect().Add(at)
			if rect.Min != image.Pt((1024-rect.Dx())/2, (768-rect.Dy())/2) || place.Scale() != 1 {
				t.Fatalf("page placement %v scale %v", rect, place.Scale())
			}
			button := v.page.Buttons[loadCancelButton].Rect
			p := button.Min.Add(button.Size().Div(2))
			x, y, ok := a.nativeFrameToWindow(p)
			if !ok {
				t.Fatal("button not represented")
			}
			a.pointer = image.Pt(x, y)
			hover, ok := a.pointerFrame()
			if !ok || !hover.In(button) {
				t.Fatalf("hover misses drawn button: %v", hover)
			}
			a.stepPage(v, appInput{CursorX: x, CursorY: y, PrimaryPressed: true})
			if !v.ui.press.Pressed(loadCancelButton) {
				t.Fatal("drawn Cancel centre did not latch Cancel")
			}
			v.ui.resetPointer()
		}
	}
	a.flow.screen = ScreenLoad
	a.flow.loadBack = ScreenMenu
	pix, err := a.composePage(a.flow.loadView())
	if err != nil || a.flow.mapShowing() || pix.RGBAAt(0, 0).A != 255 {
		t.Fatal("main-menu Load changed", err)
	}
}

func TestMissionMenuBackdropDarkensWithoutTint(t *testing.T) {
	a := nativeMenuApp(t, image.Pt(1024, 768))
	v := a.flow.viewer
	v.noticeBackdrop = color.RGBA{R: 1, G: 10, B: 35, A: 128}
	v.SetPackShown(true)
	bar, _, shown := v.packBarShown()
	if !shown || v.cam.ViewH >= bar.Max.Y || v.cam.ViewW >= 960 {
		t.Fatalf("missing HUD outside map: camera %dx%d pack %v", v.cam.ViewW, v.cam.ViewH, bar)
	}
	rect, c, ok := v.noticeBackdropOf()
	if !ok || c != (color.RGBA{A: 64}) || rect != image.Rect(0, 0, 1024, 768) {
		t.Fatalf("backdrop %v %v %v", rect, c, ok)
	}
	dst := image.NewRGBA(image.Rect(0, 0, 1024, 768))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.RGBA{R: 160, G: 120, B: 80, A: 255}), image.Point{}, draw.Src)
	draw.Draw(dst, rect, image.NewUniform(c), image.Point{}, draw.Over)
	for _, p := range []image.Point{{0, 0}, {960, 384}, bar.Min.Add(bar.Size().Div(2)), {800, 767}} {
		if got := dst.RGBAAt(p.X, p.Y); got != (color.RGBA{R: 120, G: 90, B: 60, A: 255}) {
			t.Fatalf("tinted or undimmed mission at %v: %v", p, got)
		}
	}
	v.noticeOpen = true
	v.noticeKind = NoticeDialogue
	v.font = chargenTestFont()
	v.dialogueBackdrop.shows = 1
	if _, shows, _ := v.DialogueBackdropPlan(); shows != 0 {
		t.Fatal("dialogue tint covers menu dim")
	}
	a.SetGameMenuProfile(GameMenuProfile{})
	a.flow.setMenuUp(true)
	if r, c, _ := v.noticeBackdropOf(); c != v.noticeBackdrop || r != image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH) {
		t.Fatal("first-game menu backdrop changed")
	}
	a.flow.setMenuUp(false)
	if r, c, _ := v.noticeBackdropOf(); c != v.noticeBackdrop || r != image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH) {
		t.Fatal("notice backdrop changed")
	}
}

func TestNativeMissionMenuModRowsFitAndClick(t *testing.T) {
	for mods := 0; mods <= 5; mods++ {
		t.Run(string(rune('0'+mods)), func(t *testing.T) {
			a := nativeMenuApp(t, image.Pt(1024, 768))
			a.flow.menuArt = dialogueClaimFrame()
			screens := make([]ModScreen, mods)
			for i := range screens {
				screens[i] = ModScreen{Kind: ModScreenInfo, Game: true, MenuLabel: "Details", Title: "Details"}
			}
			a.flow.modUI.screens = screens
			a.flow.rebuildGameMenu(gameMenuRoot, 0)
			rows := a.flow.menuRows()
			p := gameMenuPanelRectFor(gameMenuMission, rows, a.flow.menuGeometryProfile())
			box := windowFrame(p, a.flow.menuArt).rect()
			if !box.In(image.Rect(0, 0, 640, 480)) || box.Min != image.Pt((640-box.Dx())/2, (480-box.Dy())/2) {
				t.Fatalf("%d mods: frame and shadow %v outside or off-centre in canvas", mods, box)
			}
			_, at := a.missionMenuPlacement()
			placed := box.Add(at)
			if placed.Min != image.Pt((1024-box.Dx())/2, (768-box.Dy())/2) {
				t.Fatalf("mission placement %v", placed)
			}
			pic := a.GameMenuPanel()
			if pic == nil || pic.RGBAAt(p.Min.X, p.Max.Y-1).A == 0 {
				t.Fatal("bottom frame edge missing")
			}
			var painted image.Rectangle
			for y := 0; y < 480; y++ {
				for x := 0; x < 640; x++ {
					if pic.RGBAAt(x, y).A != 0 {
						painted = painted.Union(image.Rect(x, y, x+1, y+1))
					}
				}
			}
			if painted != box || pic.RGBAAt(p.Max.X+2, p.Max.Y+2) != frameShadowTone {
				t.Fatalf("painted frame %v, want %v including shadow", painted, box)
			}
			for n := range rows {
				centre := image.Pt(p.Min.X+gameMenuRowLeft+(p.Dx()-gameMenuRowLeft-gameMenuRowInsetR)/2,
					p.Min.Y+40+30*n+15)
				x, y, ok := a.nativeFrameToWindow(centre)
				if !ok {
					t.Fatalf("row %d centre outside canvas: %v", n, centre)
				}
				if rows[n].Enabled {
					if hit, ok := a.gameMenuButtonAt(appInput{CursorX: x, CursorY: y}); !ok || hit != n {
						t.Fatalf("drawn row %d centre hits %d %v", n, hit, ok)
					}
				}
				if mods > 0 && n == len(rows)-1 {
					for _, edge := range []string{"hover", "press", "release"} {
						if err := a.HeadlessPointer(edge, x, y); err != nil {
							t.Fatal(err)
						}
					}
					if a.Screen() != ScreenMod || a.flow.modUI.open != mods-1 {
						t.Fatal("last drawn mod row did not open its page")
					}
				}
			}
		})
	}
	t.Run("label", func(t *testing.T) {
		a := nativeMenuApp(t, image.Pt(1024, 768))
		label := strings.Repeat("Wide label ", 30)
		if err := a.SetModScreens([]ModScreen{{Kind: ModScreenInfo, Game: true, MenuLabel: label, Title: "Details"}}); err != nil {
			t.Fatal(err)
		}
		a.flow.rebuildGameMenu(gameMenuRoot, 0)
		rows := a.flow.menuRows()
		got := rows[len(rows)-1].Label
		if want := menuFit(a.flow.menuFont).Fit(label, 200-gameMenuTextX); got != want || !strings.HasSuffix(got, textfit.Mark) {
			t.Fatalf("native label %q, want %q", got, want)
		}
		if w, _ := a.flow.menuFont.Measure(got); w > 200-gameMenuTextX {
			t.Fatalf("native label width %d", w)
		}
		if pic := a.GameMenuPanel(); pic == nil {
			t.Fatal("label did not compose")
		}
	})
}

func TestNativeMissionMenuClickAndKeyboardReachTheDrawnRow(t *testing.T) {
	a := nativeMenuApp(t, image.Pt(1024, 768))
	r := gameMenuRowRect(gameMenuPanelRect(gameMenuMission, a.flow.pages.menu), 5)
	p := r.Min.Add(r.Size().Div(2))
	x, y, _ := a.nativeFrameToWindow(p)
	now := time.Unix(0, 0)
	a.step(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, now)
	if a.flow.menuPage != gameMenuEndQuestConfirmation {
		t.Fatalf("clicked page %v", a.flow.menuPage)
	}
	a.flow.rebuildGameMenu(gameMenuRoot, 0)
	for range 5 {
		a.step(appInput{Down: true}, now)
	}
	a.step(appInput{Enter: true}, now)
	if a.flow.menuPage != gameMenuEndQuestConfirmation {
		t.Fatal("keyboard focus does not reach End Quest")
	}
	a.flow.pages.menu = GameMenuProfile{}
	if place, _ := a.missionMenuPlacement(); place != frame.Fit(640, 480, 1024, 768) {
		t.Fatal("first-game placement changed")
	}
}

func TestNativeMenuProfileKeepsTownRectangles(t *testing.T) {
	f := openTownMenu(t)
	f.pages.menu.NativeMissionWindows = true
	for _, page := range []gameMenuPage{gameMenuRoot, gameMenuAbortGameConfirmation} {
		f.rebuildGameMenu(page, 0)
		s := f.menuPanelSurface()
		if got, want := gameMenuPanelRectFor(s, f.menuRows(), f.menuGeometryProfile()), gameMenuPanelRectFor(s, f.menuRows()); got != want {
			t.Fatalf("town page %v rectangle %v want %v", page, got, want)
		}
	}
}

func TestNativeMissionModPageUsesSharedHost(t *testing.T) {
	a := nativeMenuApp(t, image.Pt(1024, 768))
	if err := a.SetModScreens([]ModScreen{{Kind: ModScreenInfo, Game: true, MenuLabel: "Details", Title: "Details", Paragraphs: []string{"One page"}}}); err != nil {
		t.Fatal(err)
	}
	a.flow.openModScreen(0, ScreenGameMenu, 0)
	if !a.flow.mapShowing() || !a.flow.popupOpen() {
		t.Fatal("mod page lost the held mission")
	}
	pic := a.GameMenuPanel()
	if pic == nil || pic.RGBAAt(0, 0).A != 0 {
		t.Fatal("mod page retains the main-menu wall")
	}
	place, at := a.missionMenuPlacement()
	r := a.missionMenuRect().Add(at)
	if place.Scale() != 1 || r.Min != image.Pt((1024-r.Dx())/2, (768-r.Dy())/2) {
		t.Fatalf("mod page placement %v scale %v", r, place.Scale())
	}
	x, y, ok := a.nativeFrameToWindow(modBackButton.Min.Add(modBackButton.Size().Div(2)))
	if !ok {
		t.Fatal("mod Back button is not represented")
	}
	for _, edge := range []string{"hover", "press", "release"} {
		if err := a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	if a.Screen() != ScreenGameMenu {
		t.Fatal("drawn Back button did not close the mod page")
	}
}
