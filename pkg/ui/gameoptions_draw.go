package ui

import (
	"image"
)

// gameOptionRect places controls inside the snapped Game Options frame.
// The checkboxes, radio groups and slider keep MENU-073's 24-pixel rows; the
// engine's own controls fit around them. The page's frame is one tile row
// taller than the original's for the engine's rows; every original control
// keeps its offset from the frame's top edge and OK and Cancel keep the
// frame's bottom margin. An option's rectangle is its table row's.
func gameOptionRect(action gameMenuAction) image.Rectangle {
	g := gameOptionsDialog
	w := g.Body().Dx()
	if o, ok := actionOption(action); ok {
		r := gameOptionTable[o].rect
		return g.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y)
	}
	switch action {
	case gameMenuSpeedDown, gameMenuSpeedUp:
		return g.Rect(24, 76, 216, 100)
	case gameMenuTooltipDelay:
		return g.Rect(24, 228, 234, 248)
	case gameMenuToggleTips:
		return g.Rect(240, 104, 456, 128)
	case gameMenuTimedAutosave:
		return g.Rect(24, 340, 234, 364)
	case gameMenuAutosaveMinutes:
		return g.Rect(24, 364, 234, 388)
	case gameMenuPageReturn:
		return g.Rect(w/7, 436, 3*w/7, 460)
	case gameMenuOptionsCancel:
		return g.Rect(4*w/7, 436, 6*w/7, 460)
	}
	return image.Rectangle{}
}

// gameOptionSpeedLabel is the slider's caption rectangle.
func gameOptionSpeedLabel() image.Rectangle { return gameOptionsDialog.Rect(24, 48, 216, 72) }

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
	if o, ok := actionOption(a); ok {
		return o.stepping()
	}
	return a == gameMenuTooltipDelay || a == gameMenuAutosaveMinutes || a == gameMenuPageReturn || a == gameMenuOptionsCancel
}

// isGameOptionCheck reports whether a Game Options row is a checkbox.
func isGameOptionCheck(a gameMenuAction) bool {
	if a == gameMenuToggleTips || a == gameMenuTimedAutosave {
		return true
	}
	o, ok := actionOption(a)
	return ok && o.Choices() == 2
}

func isRadioAction(a gameMenuAction) bool {
	o, ok := actionOption(a)
	return ok && o.Choices() == 3 && !o.stepping()
}

// actionOption is the option a page action names.
func actionOption(a gameMenuAction) (GameOption, bool) {
	if a < gameMenuDayNight || a >= gameMenuDayNight+gameMenuAction(gameOptionCount) {
		return 0, false
	}
	return GameOption(a - gameMenuDayNight), true
}

func gameOptionsPage() settingsPage {
	g := gameOptionsDialog
	return settingsPage{dialog: g, titleSpan: g.Body().Dx(), title: func(f *flow) string { return f.gameOptions.Words.Title },
		control: (*App).gameOptionsControl}
}

// gameOptionsControl is a Game Options row's control. The speed slider is
// two rows over one rectangle (MENU-118); the second draws nothing.
func (a *App) gameOptionsControl(row gameMenuRow) settingsControl {
	f := a.flow
	if row.Status {
		return settingsControl{kind: settingsCaption, rect: gameOptionSpeedLabel(), label: row.Label}
	}
	r := gameOptionRect(row.Action)
	c := settingsControl{rect: r, label: row.Label, literal: row.Literal, disabled: !row.Enabled}
	d, w := f.gameOptions.draft, f.gameOptions.Words
	rebuild := func() { f.rebuildGameMenu(gameMenuGameOptionsPage, f.menuList.Selection()) }
	switch {
	case r.Empty():
	case row.Action == gameMenuSpeedDown || row.Action == gameMenuSpeedUp:
		s := hSlider{Rect: r, Pos: d.speed, Max: gameSpeedLevels}
		c.adjust = func(delta int) { f.setDraftSpeed(d.speed + delta*s.step()); rebuild() }
		if row.Action == gameMenuSpeedDown {
			c.kind, c.label, c.disabled, c.slider = settingsSlider, "", false, s
			c.set = func(pos int) int { f.setDraftSpeed(pos); rebuild(); return d.speed }
		}
	case isRadioAction(row.Action):
		o := GameOption(row.Action - gameMenuDayNight)
		c.kind, c.label = settingsRadio, w.Labels[o]
		c.group = choiceGroup{Kind: choiceRadio, Rect: gameOptionRadioRect(row.Action), Labels: w.Choices[o][:3],
			Selected: f.optionValues()[o], Off: f.gameOptions.Radios[0], On: f.gameOptions.Radios[1]}
		c.set = func(n int) int { f.setDraftOption(o, n); rebuild(); return n }
	case isGameOptionButton(row.Action):
		c.kind = settingsPush
		if row.Action == gameMenuAutosaveMinutes {
			minutes := func(delta int) {
				d.autosave.Minutes = min(max(d.autosave.Minutes+delta, 1), MaxAutosaveMinutes)
				rebuild()
			}
			c.adjust = minutes
			c.activate = func(p image.Point) {
				// The button's left half lowers the minutes, the right half raises them.
				if p.X < r.Min.X+r.Dx()/2 {
					minutes(-1)
				} else {
					minutes(1)
				}
				a.playUISound(UISoundCommonControl)
			}
		}
	default:
		c.kind = settingsCheck
		switch row.Action {
		case gameMenuToggleTips:
			c.label, c.on = w.Tips, d.tips
		case gameMenuTimedAutosave:
			c.label, _ = f.timedAutosaveLabels(d)
			c.on = d.autosave.Enabled
		default:
			o := GameOption(row.Action - gameMenuDayNight)
			c.label, c.on = w.Labels[o], f.optionValues()[o] != 0
		}
	}
	return c
}
