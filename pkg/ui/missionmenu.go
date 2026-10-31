package ui

import (
	"image"
	"image/color"

	"againrom/pkg/render/frame"
)

type menuWindowAnchor int

const (
	menuAnchorWideFrame menuWindowAnchor = iota
	menuAnchorMissionView
)

// menuWindowPlacement places a window the menu hosts: anchor view, centred
// or decoded panel origins, backdrop. Its frame is missionMenuRect.
type menuWindowPlacement struct {
	anchor            menuWindowAnchor
	centred           bool
	missionAt, townAt image.Point
	backdrop          color.RGBA
}

// wideFramePlacement is the first game's.
var wideFramePlacement = menuWindowPlacement{anchor: menuAnchorWideFrame,
	missionAt: image.Pt(gameMenuPanelLeft, gameMenuMissionTop), townAt: image.Pt(gameMenuPanelLeft, gameMenuTownTop)}

// placement is the placement builder.
func (p GameMenuProfile) placement() menuWindowPlacement {
	pl := wideFramePlacement
	if p.NativeMissionWindows {
		pl = menuWindowPlacement{anchor: menuAnchorMissionView, centred: true}
	}
	pl.backdrop = p.MissionBackdrop
	return pl
}

func (pl menuWindowPlacement) placement() menuWindowPlacement { return pl }

type menuPlacer interface{ placement() menuWindowPlacement }

func placementOf(at []menuPlacer) menuWindowPlacement {
	if len(at) > 0 {
		return at[0].placement()
	}
	return wideFramePlacement
}

func (pl menuWindowPlacement) overMission() bool { return pl.anchor == menuAnchorMissionView }

func (pl menuWindowPlacement) panel(s gameMenuSurface) image.Rectangle {
	h, at := gameMenuMissionH, pl.missionAt
	if s == gameMenuTown {
		h, at = gameMenuTownH, pl.townAt
	}
	if pl.centred {
		return newDialogGeometry(gameMenuPanelW, h).Body()
	}
	return image.Rect(at.X, at.Y, at.X+gameMenuPanelW, at.Y+h)
}

// menuPlacement is the profile's over a mission, else the wide frame's.
func (f *flow) menuPlacement() menuWindowPlacement {
	if f == nil {
		return wideFramePlacement
	}
	pl := f.pages.menu.placement()
	if !pl.overMission() || f.menuBack != ScreenMap || !f.menuWindowShown() {
		backdrop := pl.backdrop
		pl = wideFramePlacement
		pl.backdrop = backdrop
	}
	return pl
}

func (f *flow) menuWindowShown() bool {
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

func (f *flow) menuGeometryProfile() menuWindowPlacement { return f.menuPlacement() }

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
	if a.flow.menuPlacement().overMission() && a.flow.viewer != nil {
		v, r := a.flow.viewer, a.missionMenuRect()
		return v.place, image.Pt((v.frameW-r.Dx())/2-r.Min.X, (v.frameH-r.Dy())/2-r.Min.Y)
	}
	at, _ := newWideFrameLayout(a.wideFrameWidth(), wideFrameCentered).nativeToWide(image.Point{})
	return a.place, at
}
