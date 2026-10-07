package ui

import (
	"image"
	"image/color"
	"image/draw"

	"againrom/pkg/render/frame"
)

// gameOptionRect places controls inside the snapped Game Options frame.
// Authored controls use compact rows below the formation and retreat groups.
func gameOptionRect(action gameMenuAction) image.Rectangle {
	g := gameOptionsDialog
	w := g.W()
	switch action {
	case gameMenuSpeedDown, gameMenuSpeedUp:
		return g.Rect(40, 84, 232, 108)
	case gameMenuDayNight:
		return g.Rect(40, 114, 250, 138)
	case gameMenuSmoothing:
		return g.Rect(40, 140, 250, 164)
	case gameMenuShadows:
		return g.Rect(40, 166, 250, 190)
	case gameMenuLighting:
		return g.Rect(40, 192, 250, 216)
	case gameMenuAnimation:
		return g.Rect(40, 218, 250, 242)
	case gameMenuTooltipDelay:
		return g.Rect(40, 244, 250, 264)
	case gameMenuFormation:
		return g.Rect(40, 264, 232, 346)
	case gameMenuHealth:
		return g.Rect(256, 56, 472, 80)
	case gameMenuDamage:
		return g.Rect(256, 84, 472, 108)
	case gameMenuToggleTips:
		return g.Rect(256, 112, 472, 136)
	case gameMenuAutoHealing:
		return g.Rect(256, 140, 448, 232)
	case gameMenuPathfinding:
		return g.Rect(256, 232, 472, 262)
	case gameMenuRetreat:
		return g.Rect(256, 264, 424, 346)
	case gameMenuTimedAutosave:
		return g.Rect(40, 348, 250, 372)
	case gameMenuAutosaveMinutes:
		return g.Rect(256, 348, 472, 372)
	case gameMenuPageReturn:
		return g.Rect(w/7, 380, 3*w/7, 404)
	case gameMenuOptionsCancel:
		return g.Rect(4*w/7, 380, 6*w/7, 404)
	}
	return image.Rectangle{}
}

// gameOptionSpeedLabel is the slider's caption rectangle.
func gameOptionSpeedLabel() image.Rectangle { return gameOptionsDialog.Rect(40, 56, 232, 80) }

// gameOptionChoiceRect fits three radio items below the group caption.
func gameOptionChoiceRect(action gameMenuAction, choice int) image.Rectangle {
	r := gameOptionRect(action)
	h := (r.Dy() - 22) / 3
	r.Min.Y = r.Max.Y - 3*h + choice*h
	r.Max.Y = r.Min.Y + h
	return r
}

// gameOptionSliderPosition maps a frame x to a slider position.
func gameOptionSliderPosition(x int) int {
	track := gameOptionRect(gameMenuSpeedDown)
	return min(max((x-track.Min.X)*gameSpeedLevels*2/max(1, track.Dx()-1)+1, 0), gameSpeedLevels*2) / 2
}

func isRadioAction(a gameMenuAction) bool {
	return a == gameMenuFormation || a == gameMenuRetreat || a == gameMenuAutoHealing
}

// stepGameOptionsPointer handles the speed slider's keys and drag. It reports
// whether the input was consumed.
func (a *App) stepGameOptionsPointer(in appInput) bool {
	f := a.flow
	d := f.gameOptions.draft
	if d == nil {
		return false
	}
	rows := f.menuRows()
	focus := rows[f.menuList.Selection()].Action
	if (in.Left || in.Right) && focus == gameMenuAutosaveMinutes {
		delta := -1
		if in.Right {
			delta = 1
		}
		d.autosave.Minutes = min(max(d.autosave.Minutes+delta, 1), MaxAutosaveMinutes)
		f.rebuildGameMenu(gameMenuGameOptionsPage, f.menuList.Selection())
		return true
	}
	if (in.Left || in.Right) && (focus == gameMenuSpeedDown || focus == gameMenuSpeedUp) {
		delta := -1
		if in.Right {
			delta = 1
		}
		f.setDraftSpeed(d.speed + delta)
		f.rebuildGameMenu(gameMenuGameOptionsPage, f.menuList.Selection())
		return true
	}
	if in.Unfocused {
		d.dragging = false
		return false
	}
	p, inFrame := a.windowToNativeFrame(in.CursorX, in.CursorY)
	track := gameOptionRect(gameMenuSpeedDown)
	if in.PrimaryPressed && inFrame && p.In(track.Inset(-4)) {
		d.dragging = true
	}
	if d.dragging && inFrame {
		f.setDraftSpeed(gameOptionSliderPosition(p.X))
		f.rebuildGameMenu(gameMenuGameOptionsPage, f.menuList.Selection())
	}
	if in.PrimaryReleased && d.dragging {
		d.dragging = false
		a.playUISound(UISoundCommonControl)
		return true
	}
	return d.dragging
}

func (a *App) clickGameOptions(p image.Point) {
	f := a.flow
	for i, row := range f.menuRows() {
		if row.Status || !row.Enabled || row.Action == gameMenuSpeedDown || row.Action == gameMenuSpeedUp ||
			!p.In(gameOptionRect(row.Action)) {
			continue
		}
		f.menuList.Select(i)
		if row.Action == gameMenuAutosaveMinutes {
			r := gameOptionRect(row.Action)
			delta := 1
			if p.X < r.Min.X+r.Dx()/2 {
				delta = -1
			}
			f.gameOptions.draft.autosave.Minutes = min(max(f.gameOptions.draft.autosave.Minutes+delta, 1), MaxAutosaveMinutes)
			f.rebuildGameMenu(gameMenuGameOptionsPage, i)
			a.playUISound(UISoundCommonControl)
			return
		}
		if isRadioAction(row.Action) {
			for n := 0; n < 3; n++ {
				if p.In(gameOptionChoiceRect(row.Action, n)) {
					a.playUISound(UISoundCommonControl)
					f.setDraftOption(GameOption(row.Action-gameMenuDayNight), n)
					f.rebuildGameMenu(gameMenuGameOptionsPage, i)
					return
				}
			}
			return
		}
		a.chooseGameMenu()
		a.syncViewerLayout()
		return
	}
}

// drawSlider paints a track, its filled part and the thumb for position value
// of max.
func drawSlider(dst *image.RGBA, track image.Rectangle, value, max int, focused bool) {
	drawMovieBox(dst, track, false)
	filled := track.Inset(3)
	filled.Max.X = filled.Min.X + filled.Dx()*value/max
	draw.Draw(dst, filled, &image.Uniform{C: color.RGBA{48, 112, 130, 255}}, image.Point{}, draw.Src)
	x := track.Min.X + value*(track.Dx()-1)/max
	drawMovieBox(dst, image.Rect(x-4, track.Min.Y-3, x+5, track.Max.Y+3), focused)
}

func (a *App) gameOptionsPicture() *image.RGBA {
	f := a.flow
	g := gameOptionsDialog
	dst := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	drawSnappedDialog(dst, f.menuArt, g)
	font := f.menuFont
	w := f.gameOptions.Words
	label := gameMenuLabelText(w.Title)
	font.Draw(dst, label, g.Min.X+(g.W()-font.Advance(label))/2, g.Min.Y+20, gameMenuText)
	values := f.optionValues()
	d := f.gameOptions.draft
	for i, row := range f.menuRows() {
		focused := i == f.menuList.Selection()
		if row.Status {
			r := gameOptionSpeedLabel()
			font.Draw(dst.SubImage(r).(*image.RGBA), gameMenuLabelText(row.Label), r.Min.X, r.Min.Y, gameMenuText)
			continue
		}
		r := gameOptionRect(row.Action)
		if r.Empty() {
			continue
		}
		since := markCapture()
		switch {
		case row.Action == gameMenuSpeedDown || row.Action == gameMenuSpeedUp:
			if row.Action == gameMenuSpeedDown {
				drawSlider(dst, r, d.speed, gameSpeedLevels, focused || d.dragging)
			} else if focused {
				drawMovieBox(dst, r.Inset(-2), true)
			}
			continue
		case isRadioAction(row.Action):
			o := GameOption(row.Action - gameMenuDayNight)
			if focused {
				drawMovieBox(dst, r, true)
			}
			font.Draw(dst.SubImage(r).(*image.RGBA), gameMenuLabelText(w.Labels[o]), r.Min.X+2, r.Min.Y+2, gameMenuText)
			choices := w.Formation
			if o == GameOptionRetreat {
				choices = w.Retreat
			} else if o == GameOptionAutoHealing {
				choices = w.AutoHealing
			}
			for n, name := range choices {
				a.drawGameOption(dst, gameOptionChoiceRect(row.Action, n), name, values[o] == n, true)
			}
		case row.Action == gameMenuToggleTips:
			if focused {
				drawMovieBox(dst, r, true)
			}
			a.drawGameOption(dst, r, w.Tips, d.tips, false)
		case row.Action == gameMenuTimedAutosave:
			if focused {
				drawMovieBox(dst, r, true)
			}
			label, _ := f.timedAutosaveLabels(d)
			a.drawGameOption(dst, r, label, d.autosave.Enabled, false)
		case row.Action == gameMenuTooltipDelay || row.Action == gameMenuPageReturn || row.Action == gameMenuOptionsCancel || row.Action == gameMenuAutosaveMinutes:
			drawMovieBox(dst, r, focused)
			text := row.text()
			if row.Action == gameMenuPageReturn {
				text = gameMenuLabelText(row.Label)
			}
			font.Draw(dst.SubImage(r).(*image.RGBA), text, r.Min.X+(r.Dx()-font.Advance(text))/2,
				r.Min.Y+(r.Dy()-font.Height())/2, gameMenuText)
		default:
			if focused {
				drawMovieBox(dst, r, true)
			}
			o := GameOption(row.Action - gameMenuDayNight)
			a.drawGameOption(dst, r, w.Labels[o], values[o] != 0, false)
		}
		if !row.Enabled {
			dimDisabledRow(dst, r, since)
		}
	}
	return dst
}

func (a *App) drawGameOption(dst *image.RGBA, r image.Rectangle, label string, on, radio bool) {
	art := a.flow.gameOptions.Checks
	if radio {
		art = a.flow.gameOptions.Radios
	}
	i := 0
	if on {
		i = 1
	}
	if pic := art[i]; pic != nil {
		p := image.Pt(r.Min.X+(24-pic.Bounds().Dx())/2, r.Min.Y+(r.Dy()-pic.Bounds().Dy())/2)
		draw.Draw(dst, pic.Bounds().Sub(pic.Bounds().Min).Add(p), pic, pic.Bounds().Min, draw.Over)
	} else {
		y := r.Min.Y + (r.Dy()-16)/2
		drawMovieBox(dst, image.Rect(r.Min.X+3, y, r.Min.X+19, y+16), on)
	}
	font := a.flow.menuFont
	lines := wrapTooltipLine(gameMenuLabelText(label), font, r.Dx()-30)
	y := r.Min.Y + (r.Dy()-len(lines)*font.Height())/2
	for _, line := range lines {
		font.Draw(dst.SubImage(r).(*image.RGBA), line, r.Min.X+30, y, gameMenuText)
		y += font.Height()
	}
}
