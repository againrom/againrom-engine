package ui

import (
	"image"

	"againrom/pkg/render/frame"
)

// settingsPage is one in-game menu settings page as data: its frame, the
// width its title centres over, the title, each menu row's control and the
// controls outside the focus order. One picture and one step serve each page.
type settingsPage struct {
	dialog    dialogGeometry
	titleSpan int
	title     func(*flow) string
	control   func(a *App, row gameMenuRow) settingsControl
	extras    func(a *App) []settingsControl
	// notice is where the page's message wraps, two lines at most.
	notice image.Rectangle
}

type settingsKind uint8

const (
	settingsHidden settingsKind = iota
	settingsCaption
	settingsText
	settingsCheck
	settingsRadio
	settingsSlider
	settingsPush
	settingsList
)

// settingsControl is one control; its kind picks how it draws, what a press
// does and which keys it takes.
type settingsControl struct {
	kind     settingsKind
	rect     image.Rectangle
	label    string
	literal  bool
	on       bool
	disabled bool
	group    choiceGroup
	slider   hSlider
	list     *Picker
	box      listBox
	wheel    scrollSpec
	text     listRowText
	labelAt  image.Point
	// set takes a radio choice or a slider position and answers the
	// position the slider shows.
	set func(int) int
	// adjust moves the focused control one step left (-1) or right (+1).
	adjust func(delta int)
	// focusing: a drag released inside the window focuses the slider's row.
	focusing bool
	released func()
	// picked runs after a press selects a list row.
	picked func()
	// activate replaces choosing the row when a push button activates.
	activate func(p image.Point)
}

// settingsHold is the control a held button began on, its row plus one,
// with a slider's gesture and the position it shows.
type settingsHold struct {
	row    int
	value  int
	slider sliderInput
	bar    scrollBarInput
}

func (c settingsControl) hit() image.Rectangle {
	if c.kind == settingsSlider {
		return c.slider.Rect
	}
	return c.rect
}

// settingsPage is the open settings page, when its controls are wired.
func (f *flow) settingsPage() (settingsPage, bool) {
	switch {
	case f.menuPage == gameMenuGameOptionsPage && f.gameOptions.Read != nil:
		return gameOptionsPage(), true
	case f.menuPage == gameMenuSoundOptionsPage && f.soundOptions.Read != nil:
		return soundOptionsPage(), true
	case f.menuPage == gameMenuKeysPage:
		return keysPage(f.menuFont), true
	}
	return settingsPage{}, false
}

func (a *App) settingsPicture(p settingsPage) *image.RGBA {
	f := a.flow
	g := p.dialog
	dst := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	drawFrame(dst, g.frame(f.menuArt))
	font := f.menuFont
	title := gameMenuLabelText(p.title(f))
	font.Draw(dst, title, g.Min.X+(p.titleSpan-font.Advance(title))/2, g.Min.Y+20, gameMenuText)
	pointer, pointerOK := a.pointerFrame()
	for i, row := range f.menuRows() {
		a.drawSettingsControl(dst, p.control(a, row), i, i == f.menuList.Selection(), pointer, pointerOK)
	}
	if p.extras != nil {
		for _, c := range p.extras(a) {
			a.drawSettingsControl(dst, c, -1, false, pointer, pointerOK)
		}
	}
	if !p.notice.Empty() {
		for i, line := range noticeFlowLines(font, f.menuDisplayText(f.msg), p.notice.Dx()) {
			if i == 2 {
				break
			}
			font.Draw(dst, string(line), p.notice.Min.X, p.notice.Min.Y+i*(font.Height()+1), gameMenuText)
		}
	}
	return dst
}

func (a *App) drawSettingsControl(dst *image.RGBA, c settingsControl, row int, focused bool, pointer image.Point, pointerOK bool) {
	f := a.flow
	font := f.menuFont
	r := c.rect
	since := markCapture()
	switch c.kind {
	case settingsCaption:
		font.Draw(dst.SubImage(r).(*image.RGBA), gameMenuLabelText(c.label), r.Min.X, r.Min.Y, gameMenuText)
	case settingsText:
		font.Draw(dst, c.label, r.Min.X+r.Dx()/2-font.Advance(c.label)/2, r.Min.Y+(r.Dy()-font.Height())/2, gameMenuText)
	case settingsCheck:
		g := choiceGroup{Kind: choiceCheck, Rect: r, Labels: []string{c.label}, Focus: focused, Disabled: c.disabled,
			Off: f.gameOptions.Checks[0], On: f.gameOptions.Checks[1]}
		if c.on {
			g.Mask = 1
		}
		drawChoiceGroup(dst, font, g)
	case settingsRadio:
		font.Draw(dst.SubImage(r).(*image.RGBA), gameMenuLabelText(c.label), r.Min.X+2, r.Min.Y+2, gameMenuText)
		if c.disabled {
			dimDisabledRow(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, c.group.Rect.Min.Y), since)
		}
		g := c.group
		g.Focus, g.Disabled = focused, c.disabled
		drawChoiceGroup(dst, font, g)
	case settingsSlider:
		if c.label != "" {
			tone := gameMenuText
			if focused {
				tone = loadSelectedText
			}
			font.Draw(dst.SubImage(r).(*image.RGBA), gameMenuLabelText(c.label), r.Min.X, r.Min.Y, tone)
		}
		s := a.settingsSlider(c, row)
		drawHSlider(dst, a.media.scroll, s)
		if c.disabled {
			dimDisabledRow(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, s.Rect.Min.Y), since)
		}
	case settingsPush:
		inside := pointerOK && pointer.In(r)
		drawPushButton(dst, font, pushButton{Rect: r, Label: c.label, Literal: c.literal, Hover: inside, Inside: inside, Focus: focused,
			Pressed: row >= 0 && f.menuPress.Pressed(row), Disabled: c.disabled})
	case settingsList:
		font.Draw(dst, gameMenuLabelText(c.label), c.labelAt.X, c.labelAt.Y, loadSelectedText)
		drawListBox(dst, font, a.media.scroll, c.box, c.list, c.text, pointer, pointerOK)
		if c.disabled {
			dimDisabledRow(dst, r, since)
		}
	}
}

// settingsSlider is a slider as drawn: a held drag shows its own position.
func (a *App) settingsSlider(c settingsControl, row int) hSlider {
	s := c.slider
	if h := a.flow.settingsHold; h.row == row+1 && h.slider.active() {
		s.Pos = h.value
	}
	p, ok := a.pointerFrame()
	s = s.withPointer(p, ok)
	s.Disabled = c.disabled
	return s
}

// stepSettingsPage is a settings page's input: a key takes the tick;
// otherwise the pointer drives the control under it by its kind.
func (a *App) stepSettingsPage(p settingsPage, in appInput) bool {
	f := a.flow
	if f.menuList == nil {
		return false
	}
	if in.Unfocused {
		f.settingsHold = settingsHold{}
		return false
	}
	if a.settingsKey(p, in) {
		if in.PrimaryReleased {
			// A key in the release's tick takes the tick and drops the latch.
			f.menuPress.Clear()
			f.settingsHold.row = 0
		}
		return f.takeMenuExit()
	}
	a.settingsPointer(p, in)
	return f.takeMenuExit()
}

func (a *App) settingsKey(p settingsPage, in appInput) bool {
	f := a.flow
	rows := f.menuRows()
	at := f.menuList.Selection()
	if at < 0 || at >= len(rows) {
		return false
	}
	c := p.control(a, rows[at])
	if (in.Left || in.Right) && c.adjust != nil {
		delta := -1
		if in.Right {
			delta = 1
		}
		c.adjust(delta)
		return true
	}
	if in.PaneMode {
		// Tab and Shift+Tab move the focus; a focused control keeps its arrows.
		delta := 1
		if in.ShiftHeld {
			delta = -1
		}
		f.menuList.Move(delta)
		return true
	}
	if c.kind == settingsList && c.list != nil && (in.Up || in.Down || in.PageUp || in.PageDown || in.Home || in.End) {
		switch {
		case listKey(c.list, in.Up, in.Down, in.PageUp, in.PageDown):
		case in.Home:
			c.list.Select(0)
		case in.End:
			c.list.Select(c.list.Len() - 1)
		}
		return true
	}
	if (in.Up || in.Down) && c.kind == settingsRadio {
		// Focused arrows move the group's own selection and keep the focus
		// (MENU-123); past the group's end they move to the next control
		// (DIV-2592).
		n := c.group.Selected + 1
		if in.Up {
			n -= 2
		}
		if n >= 0 && n < len(c.group.Labels) {
			c.set(n)
			return true
		}
	}
	if key := pickerList.keyStep(in.Up, in.Down); key != 0 {
		f.menuList.Move(key)
		return true
	}
	if in.Enter || in.Panels && c.kind == settingsCheck {
		// Focused Space toggles a checkbox (MENU-124).
		a.chooseGameMenu()
		a.syncViewerLayout()
		return true
	}
	if in.Typed == "" {
		return false
	}
	for _, r := range in.Typed {
		if f.chooseGameMenuAccelerator(r, a.beforeGameMenuAction) {
			a.syncViewerLayout()
			break
		}
	}
	return true
}

// settingsPointer: a checkbox toggles and a radio row selects on the press
// (MENU-123, MENU-124); a push button latches and activates on a release
// inside it; a list selects on the press (MENU-120).
func (a *App) settingsPointer(p settingsPage, in appInput) {
	f := a.flow
	h := &f.settingsHold
	pt, inFrame := a.windowToNativeFrame(in.CursorX, in.CursorY)
	rows := f.menuRows()
	controls := make([]settingsControl, len(rows))
	for i, row := range rows {
		controls[i] = p.control(a, row)
	}
	for i, c := range controls {
		if c.kind != settingsList || c.list == nil {
			continue
		}
		if inFrame && pt.In(c.rect) && wheeled(in) {
			c.list.Move(c.wheel.wheelStep(in))
		}
		owner := listOwner(c.box, c.list)
		if h.bar.drive(&owner, pt, inFrame, in) != barNone {
			f.menuList.Select(i)
			return
		}
		if h.bar.active() {
			return
		}
	}
	held := h.row - 1
	if held >= 0 && held < len(controls) && controls[held].kind == settingsRadio && !in.PrimaryPressed && in.Viewer.PrimaryDown && inFrame {
		// A held button moved over the group selects again (MENU-123).
		a.settingsRadioAt(controls[held], pt)
	}
	if in.PrimaryPressed && !f.menuPress.Holds() {
		*h = settingsHold{bar: h.bar}
		if i, ok := settingsControlAt(controls, pt); ok && inFrame {
			switch c := controls[i]; c.kind {
			case settingsCheck:
				f.menuList.Select(i)
				a.chooseGameMenu()
				a.syncViewerLayout()
				return
			case settingsRadio:
				f.menuList.Select(i)
				h.row = i + 1
				a.settingsRadioAt(c, pt)
			case settingsSlider:
				h.row = i + 1
			case settingsPush:
				f.menuList.Select(i)
				f.menuPress.Press(i, true)
			case settingsList:
				if n, ok := c.box.RowAt(pt); ok {
					if top, count := c.list.Visible(); n < count {
						c.list.Select(top + n)
						f.menuList.Select(i)
						if c.picked != nil {
							c.picked()
						}
					}
				}
			}
		}
	}
	if held := h.row - 1; held >= 0 && held < len(controls) && controls[held].kind == settingsSlider {
		c := controls[held]
		was := h.slider.active()
		if pos, set := h.slider.step(a.settingsSlider(c, held), pt, inFrame, in); set {
			h.value = c.set(pos)
		}
		if was && !h.slider.active() {
			h.row = 0
			switch {
			case !c.focusing:
				a.playUISound(UISoundCommonControl)
			case inFrame:
				f.menuList.Select(held)
				a.playUISound(UISoundCommonControl)
				if c.released != nil {
					c.released()
				}
			}
		}
	}
	if !in.PrimaryReleased {
		return
	}
	h.row = 0
	i, ok := settingsControlAt(controls, pt)
	at, activated := f.menuPress.Release(i, inFrame && ok && controls[i].kind == settingsPush)
	if !activated {
		return
	}
	f.menuList.Select(at)
	if c := controls[at]; c.activate != nil {
		c.activate(pt)
		return
	}
	a.chooseGameMenu()
	a.syncViewerLayout()
}

// settingsControlAt is the first enabled control a press at p reaches.
func settingsControlAt(controls []settingsControl, p image.Point) (int, bool) {
	for i, c := range controls {
		if c.disabled || c.kind < settingsCheck || !p.In(c.hit()) {
			continue
		}
		return i, true
	}
	return -1, false
}

// settingsRadioAt selects the radio row under p, if it changes.
func (a *App) settingsRadioAt(c settingsControl, p image.Point) {
	n, ok := c.group.RowAt(p)
	if !ok || n == c.group.Selected {
		return
	}
	a.playUISound(UISoundCommonControl)
	c.set(n)
}
