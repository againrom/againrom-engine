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
// or decoded panels, backdrop. Its frame is missionMenuRect.
type menuWindowPlacement struct {
	anchor        menuWindowAnchor
	centred       bool
	mission, town image.Rectangle
	backdrop      color.RGBA
}

// wideFramePlacement is the first game's.
var wideFramePlacement = menuWindowPlacement{anchor: menuAnchorWideFrame,
	mission: image.Rect(gameMenuPanelLeft, gameMenuMissionTop, gameMenuPanelLeft+gameMenuPanelW, gameMenuMissionTop+gameMenuMissionH),
	town:    image.Rect(gameMenuPanelLeft, gameMenuTownTop, gameMenuPanelLeft+gameMenuPanelW, gameMenuTownTop+gameMenuTownH)}

// placement is the placement builder.
func (p GameMenuProfile) placement() menuWindowPlacement {
	pl := wideFramePlacement
	if p.NativeMissionWindows {
		pl.anchor, pl.centred = menuAnchorMissionView, true
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
	h, r := gameMenuMissionH, pl.mission
	if s == gameMenuTown {
		h, r = gameMenuTownH, pl.town
	}
	if pl.centred {
		return newDialogGeometry(gameMenuPanelW, h).Body()
	}
	return r
}

// menuPlacement is the profile's over a mission, else the wide frame's; the
// profile's town root panel stands for the town root only.
func (f *flow) menuPlacement() menuWindowPlacement {
	if f == nil {
		return wideFramePlacement
	}
	pl := f.pages.menu.placement()
	if f.menuBack != ScreenMap || !f.menuWindowShown() {
		pl.anchor, pl.centred = menuAnchorWideFrame, false
	}
	if f.pages.menu.TownRootPanel != (image.Rectangle{}) && f.menuPage == gameMenuRoot {
		pl.town = f.pages.menu.TownRootPanel
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
