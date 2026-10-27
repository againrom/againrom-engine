package ui

import (
	"image"

	"againrom/pkg/render/frame"
)

func (f *flow) nativeMissionMenu() bool {
	if f == nil || !f.pages.menu.NativeMissionWindows || f.menuBack != ScreenMap {
		return false
	}
	switch f.screen {
	case ScreenGameMenu:
		return true
	case ScreenSave:
		return f.pages.save != nil
	case ScreenLoad:
		return f.loadBack == ScreenGameMenu
	case ScreenMod:
		return f.modUI.back == ScreenGameMenu
	}
	return false
}

func (f *flow) menuGeometryProfile() GameMenuProfile {
	p := f.pages.menu
	p.NativeMissionWindows = f.nativeMissionMenu()
	return p
}

func (a *App) missionMenuRect() image.Rectangle {
	f := a.flow
	if f.screen == ScreenSave && f.pages.save != nil {
		return windowFrame(f.pages.save.page.Panel, f.menuArt).rect()
	}
	if f.screen == ScreenLoad {
		return windowFrame(f.loadView().page.Panel, f.menuArt).rect()
	}
	if f.screen == ScreenMod {
		return windowFrame(modPanel, f.menuArt).rect()
	}
	switch f.menuPage {
	case gameMenuGameOptionsPage:
		if f.gameOptions.Read != nil {
			return gameOptionsDialog.Rectangle
		}
	case gameMenuSoundOptionsPage:
		if f.soundOptions.Read != nil {
			return soundOptionsDialog.Rectangle
		}
	case gameMenuQuestObjectivesPage:
		return windowFrame(questPanelRect, f.menuArt).rect()
	}
	return windowFrame(gameMenuPanelRectFor(f.menuPanelSurface(), f.menuRows(), f.menuGeometryProfile()), f.menuArt).rect()
}

func (a *App) missionMenuPlacement() (frame.Placement, image.Point) {
	if a.flow.nativeMissionMenu() && a.flow.viewer != nil {
		v, r := a.flow.viewer, a.missionMenuRect()
		return v.place, image.Pt((v.frameW-r.Dx())/2-r.Min.X, (v.frameH-r.Dy())/2-r.Min.Y)
	}
	at, _ := newWideFrameLayout(a.wideFrameWidth(), wideFrameCentered).nativeToWide(image.Point{})
	return a.place, at
}
