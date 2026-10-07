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
	dst := composeGameMenuPanel(a.flow.menuFont, a.flow.menuPanelSurface(), a.flow.menuRows(), a.flow.menuList, a.flow.menuArt)
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

func composeGameMenuPanel(font *text.Font, s gameMenuSurface, rows []gameMenuRow, list *Picker, art *DialogFrame) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	art.Draw(dst, gameMenuPanelRectFor(s, rows))
	if list == nil {
		return dst
	}
	top, count := list.Visible()
	for slot := 0; slot < count && top+slot < len(rows); slot++ {
		row := rows[top+slot]
		r := gameMenuRowRect(s, slot)
		since := markCapture()
		if top+slot == list.Selection() {
			drawMovieBox(dst, r, true)
		}
		label := row.text()
		x, y := r.Min.X+gameMenuTextX, r.Min.Y+(gameMenuRowPitch-font.Height())/2
		font.Draw(dst.SubImage(r).(*image.RGBA), label, x, y, gameMenuText)
		if col := gameMenuAcceleratorColumn(row.Label); !row.Literal && col >= 0 && col < len(label) {
			x += font.Advance(label[:col])
			w := font.Advance(label[col : col+1])
			y += font.Height()
			draw.Draw(dst, image.Rect(x, y, x+w, y+1), &image.Uniform{C: gameMenuBorder}, image.Point{}, draw.Src)
		}
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
