package ui

import (
	"fmt"
	"image"
)

// HeadlessPart names one picture a headless caller can ask the App for.
type HeadlessPart int

// The parts HeadlessPicture composes. Each is one row of headlessPartRows.
const (
	HeadlessPartFrame HeadlessPart = iota
	HeadlessPartChat
	HeadlessPartGameOptions
	HeadlessPartNotice
	HeadlessPartCharacterPane
	HeadlessPartInventoryPack
	HeadlessPartBottomHUD
	HeadlessPartMinimap
	HeadlessPartMinimapComposed
	HeadlessPartMinimapCells
	HeadlessPartStructureReadout
	HeadlessPartMissionCursor
	HeadlessPartMissionCard
	HeadlessPartDrawnMissionCard
	HeadlessPartCommandPanel
	HeadlessPartTooltip
	HeadlessPartGameMenuPanel
	headlessPartCount
)

// HeadlessRequest asks the App for one picture. Cells is the cell rectangle
// of a HeadlessPartMinimapCells request and is ignored otherwise. Scaled asks
// a HeadlessPartFrame request for the frame as the window shows it: the
// native composition placed in the laid-out window by the final-frame
// scaler's reference, with the stages it passed in the note.
type HeadlessRequest struct {
	Part   HeadlessPart
	Cells  image.Rectangle
	Scaled bool
}

// HeadlessExtras holds what a part returns beside its picture; a part leaves
// the fields it has nothing for at their zero values.
type HeadlessExtras struct {
	Note       string
	Statistics bool
	At         image.Point
	Lines      [3]string
	Cells      image.Rectangle
	Tooltip    HeadlessTooltipState
}

// headlessPartRow composes one part.
type headlessPartRow func(a *App, req HeadlessRequest) (*image.RGBA, HeadlessExtras, error)

func pictureOnly(pix *image.RGBA, err error) (*image.RGBA, HeadlessExtras, error) {
	return pix, HeadlessExtras{}, err
}

func pictureAt(pix *image.RGBA, at image.Point, err error) (*image.RGBA, HeadlessExtras, error) {
	return pix, HeadlessExtras{At: at}, err
}

// headlessPartRows is the one table from a part to its composer.
var headlessPartRows = [headlessPartCount]headlessPartRow{
	HeadlessPartFrame: func(a *App, req HeadlessRequest) (*image.RGBA, HeadlessExtras, error) {
		pix, note, err := a.headlessFrame(req.Scaled)
		return pix, HeadlessExtras{Note: note}, err
	},
	HeadlessPartChat: func(a *App, _ HeadlessRequest) (*image.RGBA, HeadlessExtras, error) {
		return pictureOnly(a.headlessChatFrame())
	},
	HeadlessPartGameOptions: func(a *App, _ HeadlessRequest) (*image.RGBA, HeadlessExtras, error) {
		return pictureOnly(a.headlessGameOptionsFrame())
	},
	HeadlessPartNotice: func(a *App, _ HeadlessRequest) (*image.RGBA, HeadlessExtras, error) {
		return pictureOnly(a.headlessNoticeFrame())
	},
	HeadlessPartCharacterPane: func(a *App, _ HeadlessRequest) (*image.RGBA, HeadlessExtras, error) {
		pix, statistics, err := a.headlessCharacterPane()
		return pix, HeadlessExtras{Statistics: statistics}, err
	},
	HeadlessPartInventoryPack: func(a *App, _ HeadlessRequest) (*image.RGBA, HeadlessExtras, error) {
		return pictureOnly(a.headlessInventoryPack())
	},
	HeadlessPartBottomHUD: func(a *App, _ HeadlessRequest) (*image.RGBA, HeadlessExtras, error) {
		return pictureOnly(a.headlessBottomHUD())
	},
	HeadlessPartMinimap: func(a *App, _ HeadlessRequest) (*image.RGBA, HeadlessExtras, error) {
		return pictureOnly(a.headlessMinimap())
	},
	HeadlessPartMinimapComposed: func(a *App, _ HeadlessRequest) (*image.RGBA, HeadlessExtras, error) {
		return pictureOnly(a.headlessMinimapComposed())
	},
	HeadlessPartMinimapCells: func(a *App, req HeadlessRequest) (*image.RGBA, HeadlessExtras, error) {
		pix, cells, err := a.headlessMinimapCells(req.Cells)
		return pix, HeadlessExtras{Cells: cells}, err
	},
	HeadlessPartStructureReadout: func(a *App, _ HeadlessRequest) (*image.RGBA, HeadlessExtras, error) {
		pix, at, lines, err := a.headlessStructureReadout()
		return pix, HeadlessExtras{At: at, Lines: lines}, err
	},
	HeadlessPartMissionCursor: func(a *App, _ HeadlessRequest) (*image.RGBA, HeadlessExtras, error) {
		return pictureAt(a.headlessMissionCursor())
	},
	HeadlessPartMissionCard: func(a *App, _ HeadlessRequest) (*image.RGBA, HeadlessExtras, error) {
		return pictureOnly(a.headlessMissionCard())
	},
	HeadlessPartDrawnMissionCard: func(a *App, _ HeadlessRequest) (*image.RGBA, HeadlessExtras, error) {
		return pictureAt(a.headlessDrawnMissionCard())
	},
	HeadlessPartCommandPanel: func(a *App, _ HeadlessRequest) (*image.RGBA, HeadlessExtras, error) {
		return pictureOnly(a.headlessCommandPanel())
	},
	HeadlessPartTooltip: func(a *App, _ HeadlessRequest) (*image.RGBA, HeadlessExtras, error) {
		state, pix := a.headlessTooltip()
		return pix, HeadlessExtras{Tooltip: state}, nil
	},
	HeadlessPartGameMenuPanel: func(a *App, _ HeadlessRequest) (*image.RGBA, HeadlessExtras, error) {
		return a.GameMenuPanel(), HeadlessExtras{}, nil
	},
}

// HeadlessPicture composes the requested part without a window and returns
// its picture, what the part returns beside it, and the part's refusal. It is
// the one headless picture entry point: a new part is a new row.
func (a *App) HeadlessPicture(req HeadlessRequest) (*image.RGBA, HeadlessExtras, error) {
	if req.Part < 0 || req.Part >= headlessPartCount {
		return nil, HeadlessExtras{}, fmt.Errorf("headless picture: unknown part %d", req.Part)
	}
	return headlessPartRows[req.Part](a, req)
}
