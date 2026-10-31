package ui

import (
	"image"
	"image/color"
	"image/draw"
	"time"

	"againrom/pkg/render/frame"
)

// menuWindowPlacement places a window the menu hosts: its panels, its wash
// and its host, whose operations are what every surface asks. Its frame is
// missionMenuRect.
type menuWindowPlacement struct {
	menuWindowHost
	mission, town image.Rectangle
	wash          color.RGBA
}

// menuWindowHost is one row of where a hosted window stands, given as the
// operations a surface calls: the map held under it, the art behind it, the
// mapping between window and frame, and where its hover tip is painted.
type menuWindowHost struct {
	centred    bool
	underlay   menuUnderlay
	backdrop   menuBackdrop
	transform  menuTransform
	tipOnPanel menuTip
}

type (
	// menuUnderlay is the mission map held and drawn under the window, or none.
	menuUnderlay int
	// menuBackdrop is the menu art composed behind the window, or none.
	menuBackdrop int
	// menuTransform maps between the window and the frame, with its offset.
	menuTransform int
	// menuTip is the canvas the hover tip is painted on.
	menuTip int
)

const (
	noUnderlay menuUnderlay = iota
	mapUnderlay
)

const (
	menuArtBackdrop menuBackdrop = iota
	noBackdrop
)

const (
	wideFrameTransform menuTransform = iota
	missionViewTransform
)

const (
	tipOnFrame menuTip = iota
	tipOnWindowPanel
)

// wideFrameHost is the first game's; missionViewHost centres every hosted
// window over the held mission view.
var (
	wideFrameHost   = menuWindowHost{}
	missionViewHost = menuWindowHost{centred: true, underlay: mapUnderlay, backdrop: noBackdrop,
		transform: missionViewTransform, tipOnPanel: tipOnWindowPanel}
)

// wideFramePlacement is the first game's.
var wideFramePlacement = menuWindowPlacement{menuWindowHost: wideFrameHost,
	mission: image.Rect(gameMenuPanelLeft, gameMenuMissionTop, gameMenuPanelLeft+gameMenuPanelW, gameMenuMissionTop+gameMenuMissionH),
	town:    image.Rect(gameMenuPanelLeft, gameMenuTownTop, gameMenuPanelLeft+gameMenuPanelW, gameMenuTownTop+gameMenuTownH)}

// placement is the placement builder.
func (p GameMenuProfile) placement() menuWindowPlacement {
	pl := wideFramePlacement
	if p.NativeMissionWindows {
		pl.menuWindowHost = missionViewHost
	}
	pl.wash = p.MissionBackdrop
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

// grow lengthens panel p to need; a centred panel is centred again with its
// shadow in the native canvas.
func (pl menuWindowPlacement) grow(p image.Rectangle, need int) image.Rectangle {
	p.Max.Y = p.Min.Y + need
	if pl.centred {
		box := windowFrame(p, nil).rect()
		p = p.Add(image.Pt(0, (frame.H-box.Dy())/2-box.Min.Y))
	}
	return p
}

// hold runs the map's held step under a window over the mission.
func (u menuUnderlay) hold(a *App, in appInput, now time.Time) {
	if u == mapUnderlay {
		a.holdMapUnderMenu(in, now)
	}
}

// raise raises the viewer's half of the popup answer under a window over the
// mission.
func (u menuUnderlay) raise(f *flow) {
	if u == mapUnderlay {
		f.setMenuUp(true)
	}
}

// layer is a hosted window's own picture, drawn over the held map; nil when
// no map is held under it.
func (u menuUnderlay) layer(a *App) *image.RGBA {
	if u != mapUnderlay {
		return nil
	}
	pic, _ := a.composeScreen()
	return pic
}

// shows is the screen showing under a window: the map when it is held there,
// else s.
func (u menuUnderlay) shows(s Screen) Screen {
	if u == mapUnderlay {
		return ScreenMap
	}
	return s
}

// drawOver runs draw when a menu layer stands over the map: every window held
// over the mission, and the game menu's own list.
func (u menuUnderlay) drawOver(f *flow, draw func()) {
	if u == mapUnderlay || f.screen == ScreenGameMenu && f.menuList != nil {
		draw()
	}
}

// paint composes the menu art over dst; a window over the mission keeps its
// transparent backing.
func (b menuBackdrop) paint(a *App, dst *image.RGBA) {
	if b == menuArtBackdrop && a.assets != nil {
		draw.Draw(dst, dst.Bounds(), a.assets.Compose(a.sel.State()), image.Point{}, draw.Src)
	}
}

// place is the window's placement and the frame offset its window is drawn at.
func (t menuTransform) place(a *App) (frame.Placement, image.Point) {
	if t == missionViewTransform && a.flow.viewer != nil {
		v, r := a.flow.viewer, a.missionMenuRect()
		return v.place, image.Pt((v.frameW-r.Dx())/2-r.Min.X, (v.frameH-r.Dy())/2-r.Min.Y)
	}
	at, _ := newWideFrameLayout(a.wideFrameWidth(), wideFrameCentered).nativeToWide(image.Point{})
	return a.place, at
}

// toFrame maps a window point to the native frame.
func (t menuTransform) toFrame(a *App, x, y int) (image.Point, bool) {
	if t == missionViewTransform {
		place, at := t.place(a)
		p, ok := place.WindowToFrame(x, y)
		p = p.Sub(at)
		return p, ok && p.In(image.Rect(0, 0, frame.W, frame.H))
	}
	wide, ok := a.place.WindowToFrame(x, y)
	if !ok {
		return image.Point{}, false
	}
	return a.inputWideFrameLayout().wideToNative(wide)
}

// toWindow maps a native frame point to the window.
func (t menuTransform) toWindow(a *App, p image.Point) (int, int, bool) {
	if t == missionViewTransform {
		place, at := t.place(a)
		return place.FrameToWindow(p.Add(at))
	}
	wide, ok := a.inputWideFrameLayout().nativeToWide(p)
	if !ok {
		return 0, 0, false
	}
	return a.place.FrameToWindow(wide)
}

// tip maps a window point onto the canvas the hover tip is painted on: the
// wide composite, or the window's panel over the mission.
func (t menuTransform) tip(a *App, x, y int) (image.Point, bool) {
	if t == missionViewTransform {
		return t.toFrame(a, x, y)
	}
	return a.place.WindowToFrame(x, y)
}

// placedPanel is the window's panel placed into the window, the CPU frame of
// a window over the mission; nil on the wide frame or without a panel.
func (t menuTransform) placedPanel(a *App) *image.RGBA {
	if t != missionViewTransform {
		return nil
	}
	pic := a.GameMenuPanel()
	if pic == nil {
		return nil
	}
	place, at := t.place(a)
	dst := image.NewRGBA(image.Rectangle{Max: place.WindowSize()})
	for y := range dst.Bounds().Dy() {
		for x := range dst.Bounds().Dx() {
			if p, ok := place.WindowToFrame(x, y); ok {
				dst.Set(x, y, pic.At(p.X-at.X, p.Y-at.Y))
			}
		}
	}
	return dst
}

// paint paints the hover tip onto the panel picture of a window over the
// mission; on the wide frame the composite carries the tip.
func (t menuTip) paint(a *App, pic *image.RGBA) {
	if t == tipOnWindowPanel && pic != nil {
		a.paintTooltip(pic)
	}
}

// menuPlacement is the profile's over a mission, else the wide frame's; the
// profile's town root panel stands for the town root only.
func (f *flow) menuPlacement() menuWindowPlacement {
	if f == nil {
		return wideFramePlacement
	}
	pl := f.pages.menu.placement()
	if f.menuBack != ScreenMap || !f.menuWindowShown() {
		pl.menuWindowHost = wideFrameHost
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
	return a.flow.menuPlacement().transform.place(a)
}
