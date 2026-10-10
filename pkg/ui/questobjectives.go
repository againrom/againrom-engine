package ui

import (
	"image"
	"image/color"
	"strings"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
	"github.com/hajimehoshi/ebiten/v2"
)

// The owner reference shows one paragraph panel and a centred OK control.
// Geometry uses the native frame and the existing installed popup art.
var questPanelRect = image.Rect(80, 130, 560, 350)
var questButtonRect = image.Rect(270, 304, 370, 328)

func (a *App) SetGameMenuArt(art *MenuPanelArt) {
	a.flow.menuArt = art
	if a.flow.viewer != nil {
		a.flow.viewer.SetDialogFrame(art)
	}
}

func (f *flow) questLines() []string {
	s := strings.TrimSpace(f.menuContext.Objective)
	if s == "" {
		s = f.word("quest.empty")
	}
	if f.menuFont == nil {
		return wrapGameMenuText(s)
	}
	return wrapShopTip(f.menuFont, f.menuDisplayText(s), 400)
}

func (f *flow) questRows() []gameMenuRow {
	rows := []gameMenuRow{{Label: f.words.QuestHeading, Literal: true, Status: true}}
	for _, line := range f.questLines() {
		rows = append(rows, gameMenuRow{Label: line, Literal: true, Status: true})
	}
	label := f.words.NoticeButton
	if label == "" {
		label = "OK"
	}
	return append(rows, gameMenuRow{Label: label, Literal: true, Action: gameMenuPageReturn, Enabled: true})
}

func (a *App) questPicture() *image.RGBA {
	f := a.flow
	pix := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	drawFrame(pix, windowFrame(questPanelRect, f.menuArt))
	font := f.menuFont
	if font == nil {
		return pix
	}
	label := func(s string, x, y int) {
		font.Draw(pix, s, x+1, y+1, color.RGBA{A: 255})
		font.Draw(pix, s, x, y, townShellText)
	}
	label(f.words.QuestHeading, 120, 174)
	lines := f.questLines()
	pitch := max(1, font.Height()+2)
	count := max(1, (294-194)/pitch)
	top := min(max(0, f.questTop), max(0, len(lines)-count))
	for i := 0; i < count && top+i < len(lines); i++ {
		label(lines[top+i], 120, 194+i*pitch)
	}
	pointer, pointerOK := a.pointerFrame()
	if len(lines) > count {
		drawVScrollBar(pix, a.media.scroll, questBar(top, len(lines)-count).withPointer(pointer, pointerOK))
	}
	rows := f.questRows()
	inside := pointerOK && pointer.In(questButtonRect)
	drawPushButton(pix, font, pushButton{Rect: questButtonRect, Label: rows[len(rows)-1].Label, Literal: true,
		Hover: inside, Inside: inside, Pressed: f.questPress.Pressed(0)})
	return pix
}

// questBar is the objective text's bar: the shared vertical bar over its
// top-line positions 0..most.
func questBar(top, most int) vScrollBar {
	return vScrollBar{Rect: questBarRect, Pos: top, Count: most + 1}
}

var questBarRect = image.Rect(530, 194, 554, 294)

// QuestObjectivePanel exposes the same CPU composition uploaded by Draw.
func (a *App) QuestObjectivePanel() *image.RGBA {
	if a == nil || a.flow == nil || a.flow.screen != ScreenGameMenu || a.flow.menuPage != gameMenuQuestObjectivesPage {
		return nil
	}
	return a.questPicture()
}

func (a *App) paintCurrentGameMenu(dst *ebiten.Image) {
	since := markCapture()
	if pic := a.GameMenuPanel(); pic != nil {
		blit := pic
		if a.textSmoothingEnabled && a.gameMenuAcknowledgement() != "" {
			var status []text.DrawCall
			calls := text.Captured()
			for i := since; i < len(calls); i++ {
				if calls[i].Y == pickerMessageY {
					status = append(status, calls[i])
					calls[i].Erased = true
				}
			}
			if len(status) > 0 {
				blit = image.NewRGBA(pic.Bounds())
				copy(blit.Pix, pic.Pix)
				textsmooth.Erase(blit, status)
			}
		}
		dst.DrawImage(ebiten.NewImageFromImage(blit), nil)
		a.menuLog.over(pic, image.Point{})
		return
	}
	paintGameMenu(dst, a.flow.menuFont, a.flow.menuPanelSurface(), a.flow.menuRows(), a.flow.menuList, &a.menuLog)
}

func (a *App) stepQuestObjectives(in appInput) {
	f := a.flow
	if in.Unfocused {
		f.questPress, f.questBar = buttonLatch{}, scrollBarInput{}
		return
	}
	if in.Enter {
		f.rebuildGameMenu(gameMenuRoot, 0)
		f.questPress.Clear()
		return
	}
	if in.Up {
		f.questTop = max(0, f.questTop-1)
	}
	if in.Down {
		f.questTop++
	}
	if in.WheelY != 0 {
		f.questTop = max(0, f.questTop-int(in.WheelY)*3)
	}
	pitch := 16
	if f.menuFont != nil {
		pitch = max(1, f.menuFont.Height()+2)
	}
	most := max(0, len(f.questLines())-max(1, 100/pitch))
	f.questTop = min(f.questTop, most)
	p, valid := a.windowToNativeFrame(in.CursorX, in.CursorY)
	if most > 0 {
		if req, pos := f.questBar.step(questBar(f.questTop, most), p, valid, in); req != barNone {
			switch req {
			case barSetPos:
				f.questTop = pos
			case barLineUp:
				f.questTop--
			case barLineDown:
				f.questTop++
			case barPageUp:
				f.questTop -= max(1, 100/pitch)
			case barPageDown:
				f.questTop += max(1, 100/pitch)
			}
			f.questTop = min(max(f.questTop, 0), most)
		}
		if f.questBar.active() {
			return
		}
	}
	hit := valid && p.In(questButtonRect)
	if in.PrimaryPressed {
		f.questPress.Press(0, hit)
	}
	if in.PrimaryReleased {
		_, activate := f.questPress.Release(0, hit)
		if activate {
			a.playUISound(UISoundCommonControl)
			f.rebuildGameMenu(gameMenuRoot, 0)
		}
	}
}
