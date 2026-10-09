package ui

import (
	"image"
	"image/color"
	"image/draw"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/text"
)

// GameMenuPanel is the CPU image uploaded by the windowed menu. The map or town
// beneath it is composed independently, with the same centered modal placement.
func (a *App) GameMenuPanel() *image.RGBA {
	if a == nil || a.flow == nil || a.flow.screen != ScreenGameMenu || a.flow.menuFont == nil {
		return nil
	}
	if a.flow.menuPage == gameMenuQuestObjectivesPage {
		return a.questPicture()
	}
	if a.flow.menuPage == gameMenuGameOptionsPage && a.flow.gameOptions.Read != nil {
		return a.gameOptionsPicture()
	}
	if a.flow.menuPage == gameMenuSoundOptionsPage && a.flow.soundOptions.Read != nil {
		return a.soundOptionsPicture()
	}
	dst := composeGameMenuPanel(a.flow.menuFont, a.flow.menuPanelSurface(), a.flow.menuRows(), a.flow.menuList, a.flow.menuArt, a.menuPointer())
	if message := a.gameMenuAcknowledgement(); message != "" {
		font := a.flow.menuFont
		r := image.Rect(pickerLeft, pickerMessageY, pickerLeft+pickerCols*pickerAdvance, pickerMessageY+pickerLine)
		if w, _ := font.Measure(message); w > r.Dx() {
			for len(message) > 0 {
				message = message[:len(message)-1]
				if w, _ := font.Measure(message + clipMark); w <= r.Dx() {
					message += clipMark
					break
				}
			}
		}
		font.Draw(dst.SubImage(r).(*image.RGBA), message, r.Min.X, r.Min.Y, color.RGBA{255, 255, 255, 255})
	}
	return dst
}

func (a *App) gameMenuAcknowledgement() string {
	if a.flow.menuFont == nil || a.flow.msg == "" {
		return ""
	}
	message := a.flow.words.SaveAcknowledgement
	if message == "" {
		message = AuthoredWords().SaveAcknowledgement
	}
	if a.flow.msg == message {
		return message
	}
	return ""
}

// gameMenuPointer is what the menu buttons read of the pointer: where it is
// on the frame, and which button holds the press latch.
type gameMenuPointer struct {
	at    image.Point
	ok    bool
	press buttonLatch
}

// menuPointer is the pointer state the menu panel is drawn with.
func (a *App) menuPointer() gameMenuPointer {
	p, ok := a.pointerFrame()
	return gameMenuPointer{at: p, ok: ok, press: a.flow.menuPress}
}

// composeGameMenuPanel draws the frame and one push button per row
// (MENU-115, MENU-ART-014). The selected row is the keyboard focus. Status
// and literal rows are text, not buttons.
func composeGameMenuPanel(font *text.Font, s gameMenuSurface, rows []gameMenuRow, list *Picker, art *DialogFrame, pointer gameMenuPointer) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	art.Draw(dst, gameMenuPanelRectFor(s, rows))
	if list == nil {
		return dst
	}
	top, count := list.Visible()
	for slot := 0; slot < count && top+slot < len(rows); slot++ {
		row := rows[top+slot]
		r := gameMenuRowRect(s, slot)
		if !row.Status && !row.Literal {
			inside := pointer.ok && pointer.at.In(r)
			drawPushButton(dst, font, pushButton{Rect: r, Label: row.Label, Hover: inside,
				Focus: top+slot == list.Selection(), Pressed: pointer.press.pressed(top + slot),
				Inside: inside, Disabled: !row.Enabled})
			continue
		}
		since := markCapture()
		label := row.text()
		x, y := r.Min.X+gameMenuTextX, r.Min.Y+(gameMenuRowPitch-font.Height())/2
		font.Draw(dst.SubImage(r).(*image.RGBA), label, x, y, gameMenuText)
		if !row.Enabled && !row.Status {
			dimDisabledRow(dst, r, since)
		}
	}
	return dst
}

// dimDisabledRow dims row r of dst as a disabled row is, and tells the text
// overlay that the glyphs captured since start now stand under the dim.
func dimDisabledRow(dst *image.RGBA, r image.Rectangle, start int) {
	draw.Draw(dst, r, &image.Uniform{C: gameMenuDisabled}, image.Point{}, draw.Over)
	text.TintSince(start, r, gameMenuDisabled)
}
