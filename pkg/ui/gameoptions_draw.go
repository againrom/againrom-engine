package ui

import (
	"image"

	"againrom/pkg/render/frame"
)

// gameOptionRect places controls inside the snapped Game Options frame.
// The checkboxes, radio groups and slider keep MENU-073's 24-pixel rows; the
// engine's own controls fit around them.
func gameOptionRect(action gameMenuAction) image.Rectangle {
	g := gameOptionsDialog
	w := g.W()
	switch action {
	case gameMenuSpeedDown, gameMenuSpeedUp:
		return g.Rect(40, 84, 232, 108)
	case gameMenuDayNight:
		return g.Rect(40, 112, 250, 136)
	case gameMenuSmoothing:
		return g.Rect(40, 136, 250, 160)
	case gameMenuShadows:
		return g.Rect(40, 160, 250, 184)
	case gameMenuLighting:
		return g.Rect(40, 184, 250, 208)
	case gameMenuAnimation:
		return g.Rect(40, 208, 250, 232)
	case gameMenuTooltipDelay:
		return g.Rect(40, 234, 250, 254)
	case gameMenuFormation:
		return g.Rect(40, 256, 232, 346)
	case gameMenuHealth:
		return g.Rect(256, 56, 472, 80)
	case gameMenuDamage:
		return g.Rect(256, 84, 472, 108)
	case gameMenuToggleTips:
		return g.Rect(256, 112, 472, 136)
	case gameMenuAutoHealing:
		return g.Rect(256, 140, 448, 232)
	case gameMenuPathfinding:
		return g.Rect(256, 232, 472, 256)
	case gameMenuRetreat:
		return g.Rect(256, 256, 424, 346)
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

// gameOptionRadioRect is a radio group's rows: three 24-pixel rows at the
// bottom of its rectangle, under the group caption.
func gameOptionRadioRect(action gameMenuAction) image.Rectangle {
	r := gameOptionRect(action)
	return image.Rect(r.Min.X, r.Max.Y-3*24, r.Max.X, r.Max.Y)
}

// gameOptionChoiceRect is radio row choice of a group.
func gameOptionChoiceRect(action gameMenuAction, choice int) image.Rectangle {
	return choiceGroup{Rect: gameOptionRadioRect(action)}.RowRect(choice)
}

// isGameOptionButton reports the push buttons among the page's rows.
func isGameOptionButton(a gameMenuAction) bool {
	return a == gameMenuTooltipDelay || a == gameMenuAutosaveMinutes || a == gameMenuPageReturn || a == gameMenuOptionsCancel
}

// gameSpeedSlider is the speed control: the shared slider over the engine's
// speed positions (MENU-118).
func (a *App) gameSpeedSlider(speed int) hSlider {
	p, ok := a.pointerFrame()
	return hSlider{Rect: gameOptionRect(gameMenuSpeedDown), Pos: speed, Max: gameSpeedLevels}.withPointer(p, ok)
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
		delta := -a.gameSpeedSlider(d.speed).step()
		if in.Right {
			delta = -delta
		}
		f.setDraftSpeed(d.speed + delta)
		f.rebuildGameMenu(gameMenuGameOptionsPage, f.menuList.Selection())
		return true
	}
	if in.Unfocused {
		d.slider = sliderInput{}
		return false
	}
	p, inFrame := a.windowToNativeFrame(in.CursorX, in.CursorY)
	if d.radio != 0 && !in.PrimaryPressed && in.Viewer.PrimaryDown && inFrame {
		// A held button moved over the group selects again (MENU-123).
		a.selectGameOptionRadio(d.radio, p)
	}
	held := d.slider.active()
	if pos, set := d.slider.step(a.gameSpeedSlider(d.speed), p, inFrame, in); set {
		f.setDraftSpeed(pos)
		f.rebuildGameMenu(gameMenuGameOptionsPage, f.menuList.Selection())
	}
	if held && !d.slider.active() {
		a.playUISound(UISoundCommonControl)
		return true
	}
	return d.slider.active()
}

// gameOptionsRowAt is the enabled control row under p, outside the slider.
func (a *App) gameOptionsRowAt(p image.Point) (int, gameMenuRow, bool) {
	for i, row := range a.flow.menuRows() {
		if row.Status || !row.Enabled || row.Action == gameMenuSpeedDown || row.Action == gameMenuSpeedUp ||
			!p.In(gameOptionRect(row.Action)) {
			continue
		}
		return i, row, true
	}
	return 0, gameMenuRow{}, false
}

// pressGameOptions is the page's button-down: a checkbox toggles and a radio
// row selects on the press (MENU-123, MENU-124); a push button latches.
func (a *App) pressGameOptions(p image.Point, ok bool) {
	f := a.flow
	f.menuPress.clear()
	i, row, hit := a.gameOptionsRowAt(p)
	if !ok || !hit {
		return
	}
	f.menuList.Select(i)
	switch {
	case isGameOptionButton(row.Action):
		f.menuPress.press(i, true)
	case isRadioAction(row.Action):
		f.gameOptions.draft.radio = row.Action
		a.selectGameOptionRadio(row.Action, p)
	default:
		a.chooseGameMenu()
		a.syncViewerLayout()
	}
}

// selectGameOptionRadio selects the radio row under p, if it changes.
func (a *App) selectGameOptionRadio(action gameMenuAction, p image.Point) {
	f := a.flow
	n, ok := choiceGroup{Rect: gameOptionRadioRect(action), Labels: make([]string, 3)}.RowAt(p)
	o := GameOption(action - gameMenuDayNight)
	if !ok || f.optionValues()[o] == n {
		return
	}
	a.playUISound(UISoundCommonControl)
	f.setDraftOption(o, n)
	f.rebuildGameMenu(gameMenuGameOptionsPage, f.menuList.Selection())
}

// releaseGameOptions activates the latched push button on a release inside.
func (a *App) releaseGameOptions(p image.Point, ok bool) {
	f := a.flow
	f.gameOptions.draft.radio = 0
	i, row, hit := a.gameOptionsRowAt(p)
	at, activated := f.menuPress.release(i, ok && hit && isGameOptionButton(row.Action))
	if !activated {
		return
	}
	f.menuList.Select(at)
	if row.Action == gameMenuAutosaveMinutes {
		r := gameOptionRect(row.Action)
		delta := 1
		if p.X < r.Min.X+r.Dx()/2 {
			delta = -1
		}
		f.gameOptions.draft.autosave.Minutes = min(max(f.gameOptions.draft.autosave.Minutes+delta, 1), MaxAutosaveMinutes)
		f.rebuildGameMenu(gameMenuGameOptionsPage, at)
		a.playUISound(UISoundCommonControl)
		return
	}
	a.chooseGameMenu()
	a.syncViewerLayout()
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
	pointer, pointerOK := a.pointerFrame()
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
		switch {
		case row.Action == gameMenuSpeedDown || row.Action == gameMenuSpeedUp:
			if row.Action == gameMenuSpeedDown {
				drawHSlider(dst, a.media.scroll, a.gameSpeedSlider(d.speed))
			}
		case isRadioAction(row.Action):
			o := GameOption(row.Action - gameMenuDayNight)
			since := markCapture()
			font.Draw(dst.SubImage(r).(*image.RGBA), gameMenuLabelText(w.Labels[o]), r.Min.X+2, r.Min.Y+2, gameMenuText)
			if !row.Enabled {
				dimDisabledRow(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, gameOptionRadioRect(row.Action).Min.Y), since)
			}
			choices := w.Formation
			if o == GameOptionRetreat {
				choices = w.Retreat
			} else if o == GameOptionAutoHealing {
				choices = w.AutoHealing
			}
			drawChoiceGroup(dst, font, choiceGroup{Kind: choiceRadio, Rect: gameOptionRadioRect(row.Action), Labels: choices[:],
				Selected: values[o], Focus: focused, Disabled: !row.Enabled, Off: f.gameOptions.Radios[0], On: f.gameOptions.Radios[1]})
		case isGameOptionButton(row.Action):
			inside := pointerOK && pointer.In(r)
			drawPushButton(dst, font, pushButton{Rect: r, Label: row.Label, Literal: row.Literal, Hover: inside, Inside: inside,
				Focus: focused, Pressed: f.menuPress.pressed(i), Disabled: !row.Enabled})
		default:
			label, on := "", false
			switch row.Action {
			case gameMenuToggleTips:
				label, on = w.Tips, d.tips
			case gameMenuTimedAutosave:
				label, _ = f.timedAutosaveLabels(d)
				on = d.autosave.Enabled
			default:
				o := GameOption(row.Action - gameMenuDayNight)
				label, on = w.Labels[o], values[o] != 0
			}
			a.drawGameOption(dst, r, label, on, focused, !row.Enabled)
		}
	}
	return dst
}

// drawGameOption draws one standard checkbox row through the shared
// builder (MENU-124).
func (a *App) drawGameOption(dst *image.RGBA, r image.Rectangle, label string, on, focused, disabled bool) {
	g := choiceGroup{Kind: choiceCheck, Rect: r, Labels: []string{label}, Focus: focused, Disabled: disabled,
		Off: a.flow.gameOptions.Checks[0], On: a.flow.gameOptions.Checks[1]}
	if on {
		g.Mask = 1
	}
	drawChoiceGroup(dst, a.flow.menuFont, g)
}
