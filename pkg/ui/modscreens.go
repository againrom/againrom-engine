package ui

import (
	"fmt"
	"image"
	"image/draw"
	"strings"
	"time"

	"againrom/pkg/render/debugtext"
	"againrom/pkg/render/frame"
)

// modScreenState is the state of the registered mod screens: the screens, the
// one open (an index, or -1), the screen it was opened from, the in-game menu row
// to select again on return, the scroll offset, the pressed back control, and
// the 1-based main menu entries under and pressed by the pointer (0 for none).
type modScreenState struct {
	screens   []ModScreen
	open      int
	back      Screen
	returnRow int
	top       int
	backPress bool
	hover     int
	press     int
	bar       scrollBarInput
	// pending is the 1-based index of the action entry whose confirming page is
	// open, zero for none.
	pending int
}

// ScreenMod shows one screen a mod declares in data. It is declared apart from
// the const block of the shipped screens: it exists in every build, but only a
// mod gives it something to show, so the shipped screen census does not count
// it.
const ScreenMod Screen = ScreenCredits + 1

// ModScreenKind is the page layout a data-defined screen is drawn with.
type ModScreenKind int

const (
	// ModScreenInfo is a title and paragraphs.
	ModScreenInfo ModScreenKind = iota + 1
	// ModScreenList is a title and a bulleted list.
	ModScreenList
	// ModScreenTable is a title and rows of a label and a value.
	ModScreenTable
	// ModActionAbandon is an in-game menu entry that leaves the mission without
	// victory and returns to the town it was entered from.
	ModActionAbandon
	// ModActionRestart is an in-game menu entry that reloads the mission from
	// its entry point.
	ModActionRestart
)

// isAction reports that the kind is an entry that runs an action, not a page.
func (k ModScreenKind) isAction() bool { return k == ModActionAbandon || k == ModActionRestart }

// The menus have a fixed number of slots for the entries mods register.
const (
	MaxModMainEntries = 3
	MaxModGameEntries = 2
)

// ModScreen is one screen a mod declares. The strings are final text in the
// game's language; a mod supplies no drawing code.
type ModScreen struct {
	Mod       string
	Key       string
	Kind      ModScreenKind
	Title     string
	MenuLabel string
	// Main and Game place the entry that opens the screen in the main menu, the
	// in-game menu, or both.
	Main bool
	Game bool

	Paragraphs []string
	Items      []string
	Rows       [][2]string
}

// SetModScreens registers the screens mods declare. The main menu and the
// in-game menu each gain one entry per screen placed in them. It is called once,
// before Run, and refuses a screen no kind draws or a menu with more entries
// than slots.
func (a *App) SetModScreens(screens []ModScreen) error {
	main, game := 0, 0
	for _, s := range screens {
		switch s.Kind {
		case ModScreenInfo, ModScreenList, ModScreenTable:
		case ModActionAbandon, ModActionRestart:
			if s.Main || !s.Game {
				return fmt.Errorf("mod %q action %q: an action entry is in the in-game menu only", s.Mod, s.Key)
			}
		default:
			return fmt.Errorf("mod %q screen %q: unknown kind %d", s.Mod, s.Key, int(s.Kind))
		}
		if strings.TrimSpace(s.MenuLabel) == "" || strings.TrimSpace(s.Title) == "" {
			return fmt.Errorf("mod %q screen %q: no title or menu label", s.Mod, s.Key)
		}
		if !s.Main && !s.Game {
			return fmt.Errorf("mod %q screen %q: placed in no menu", s.Mod, s.Key)
		}
		if s.Main {
			main++
		}
		if s.Game {
			game++
		}
	}
	if main > MaxModMainEntries || game > MaxModGameEntries {
		return fmt.Errorf("mod screens: %d main menu and %d game menu entries; the menus have %d and %d slots",
			main, game, MaxModMainEntries, MaxModGameEntries)
	}
	a.flow.modUI.screens = append([]ModScreen(nil), screens...)
	a.flow.modUI.open = -1
	a.hasMenu = false
	return nil
}

// modScreenIndexes lists the screens placed in a menu, in registration order.
func (f *flow) modScreenIndexes(game bool) []int {
	var out []int
	for i, s := range f.modUI.screens {
		if game && s.Game || !game && s.Main {
			out = append(out, i)
		}
	}
	return out
}

// openModScreen shows screen i, remembering the screen it was opened from. row is
// the in-game menu row to select again when the player comes back.
func (f *flow) openModScreen(i int, back Screen, row int) {
	if i < 0 || i >= len(f.modUI.screens) {
		return
	}
	f.modUI.open, f.modUI.back, f.modUI.returnRow = i, back, row
	f.modUI.top, f.modUI.backPress, f.modUI.bar = 0, false, scrollBarInput{}
	f.msg = ""
	f.setScreen(ScreenMod)
}

// closeModScreen returns to the screen the mod screen was opened from.
func (f *flow) closeModScreen() {
	back := f.modUI.back
	f.modUI.open, f.modUI.top, f.modUI.backPress = -1, 0, false
	f.modUI.hover, f.modUI.press = 0, 0
	f.msg = ""
	if back == ScreenGameMenu {
		f.setScreen(ScreenGameMenu)
		f.rebuildGameMenu(gameMenuRoot, f.modUI.returnRow)
		f.setMenuUp(true)
		return
	}
	f.setScreen(back)
}

// The mod screen's page, in the fixed 640x480 frame.
var (
	modPanel      = image.Rect(96, 52, 544, 428)
	modTitleBox   = image.Rect(128, 72, 512, 104)
	modBodyBox    = image.Rect(124, 116, 504, 348)
	modBackButton = image.Rect(260, 366, 380, 396)
)

const (
	modScrollBarX     = 512
	modWheelLines     = 3
	modIndent         = 14
	modEntryW         = 200
	modEntryH         = 28
	modEntryPitch     = 32
	modEntryMargin    = 10
	modFallbackLineH  = debugtext.LineHeight
	modTableLabelFrac = 5 // label column is 5/12 of the body, the value column the rest
)

// modLine is one drawn line of a screen body: text at the left, and for a table
// row the value at the start of the right column.
type modLine struct {
	left, right string
	indent      bool
}

// modMeasure returns the pixel width of s in the font the page is drawn with.
func (a *App) modMeasure(s string) int {
	if f := a.flow.menuFont; f != nil {
		w, _ := f.Measure(a.flow.menuDisplayText(s))
		return w
	}
	return len([]rune(s)) * debugtext.CellWidth
}

// modLineHeight is the pitch of a body line.
func (a *App) modLineHeight() int {
	if f := a.flow.menuFont; f != nil {
		return f.Height() + 3
	}
	return modFallbackLineH
}

// modWrap breaks s at spaces into lines no wider than width pixels. A word wider
// than the line is cut where it no longer fits.
func (a *App) modWrap(s string, width int) []string {
	var out []string
	line := ""
	flush := func() {
		if line != "" {
			out = append(out, line)
			line = ""
		}
	}
	for _, word := range strings.Fields(s) {
		for a.modMeasure(word) > width {
			r := []rune(word)
			n := len(r)
			for n > 1 && a.modMeasure(string(r[:n])) > width {
				n--
			}
			if line != "" {
				flush()
			}
			out = append(out, string(r[:n]))
			word = string(r[n:])
		}
		if word == "" {
			continue
		}
		if line == "" {
			line = word
		} else if a.modMeasure(line+" "+word) <= width {
			line += " " + word
		} else {
			flush()
			line = word
		}
	}
	flush()
	return out
}

// modLines lays the open screen's body out as lines.
func (a *App) modLines(s ModScreen) []modLine {
	width := modBodyBox.Dx()
	var lines []modLine
	switch s.Kind {
	case ModScreenInfo:
		for i, p := range s.Paragraphs {
			if i > 0 {
				lines = append(lines, modLine{})
			}
			for _, l := range a.modWrap(p, width) {
				lines = append(lines, modLine{left: l})
			}
		}
	case ModScreenList:
		for _, item := range s.Items {
			for i, l := range a.modWrap(item, width-modIndent) {
				if i == 0 {
					lines = append(lines, modLine{left: "-", right: l})
					continue
				}
				lines = append(lines, modLine{right: l})
			}
		}
	case ModScreenTable:
		labelW := width * modTableLabelFrac / 12
		valueW := width - labelW - modIndent
		for _, row := range s.Rows {
			l, v := a.modWrap(row[0], labelW), a.modWrap(row[1], valueW)
			for i := 0; i < len(l) || i < len(v); i++ {
				var ml modLine
				if i < len(l) {
					ml.left = l[i]
				}
				if i < len(v) {
					ml.right = v[i]
				}
				ml.indent = true
				lines = append(lines, ml)
			}
		}
	}
	return lines
}

// modVisibleLines is how many body lines the page shows at once.
func (a *App) modVisibleLines() int {
	n := modBodyBox.Dy() / a.modLineHeight()
	if n < 1 {
		return 1
	}
	return n
}

// modMaxTop is the largest scroll offset of the open screen.
func (a *App) modMaxTop() int {
	if a.flow.modUI.open < 0 || a.flow.modUI.open >= len(a.flow.modUI.screens) {
		return 0
	}
	over := len(a.modLines(a.flow.modUI.screens[a.flow.modUI.open])) - a.modVisibleLines()
	if over < 0 {
		return 0
	}
	return over
}

// modBackCaption is the install's own OK word, in the install's code page, the
// caption the load window gives its way out.
func (a *App) modBackCaption() string {
	w := a.flow.loadUI.words
	if w.OK == "" {
		w = defaultLoadWindowWords()
	}
	return w.OK
}

// drawModText draws s with the page's font at (x, y), top left. The text is
// converted to the install's code page as every other row is.
func (a *App) drawModText(pix *image.RGBA, s string, x, y int) {
	if s == "" {
		return
	}
	if f := a.flow.menuFont; f != nil {
		f.Draw(pix, a.flow.menuDisplayText(s), x, y, townShellText)
		return
	}
	debugtext.Draw(pix, s, x, y)
}

// drawModCentered draws s centred in r.
func (a *App) drawModCentered(pix *image.RGBA, s string, r image.Rectangle) {
	h := debugtext.LineHeight
	if f := a.flow.menuFont; f != nil {
		h = f.Height()
	}
	w := a.modMeasure(s)
	a.drawModText(pix, s, r.Min.X+(r.Dx()-w)/2, r.Min.Y+(r.Dy()-h)/2)
}

// composeModScreen draws the open mod screen over the menu's own background, in
// the frame, font and colours of the load window.
func (a *App) composeModScreen() (*image.RGBA, error) {
	f := a.flow
	if f.modUI.open < 0 || f.modUI.open >= len(f.modUI.screens) {
		return nil, fmt.Errorf("no mod screen is open")
	}
	s := f.modUI.screens[f.modUI.open]
	pix := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	if a.assets != nil {
		draw.Draw(pix, pix.Bounds(), a.assets.Compose(a.sel.State()), image.Point{}, draw.Src)
	}
	f.menuArt.Draw(pix, modPanel)
	a.drawModCentered(pix, s.Title, modTitleBox)

	lines := a.modLines(s)
	pitch, visible := a.modLineHeight(), a.modVisibleLines()
	top := f.modUI.top
	if max := len(lines) - visible; top > max {
		top = max
	}
	if top < 0 {
		top = 0
	}
	labelW := modBodyBox.Dx() * modTableLabelFrac / 12
	for i := 0; i < visible && top+i < len(lines); i++ {
		l := lines[top+i]
		y := modBodyBox.Min.Y + i*pitch
		x := modBodyBox.Min.X
		switch s.Kind {
		case ModScreenList:
			a.drawModText(pix, l.left, x, y)
			a.drawModText(pix, l.right, x+modIndent, y)
		case ModScreenTable:
			a.drawModText(pix, l.left, x, y)
			a.drawModText(pix, l.right, x+labelW+modIndent, y)
		default:
			a.drawModText(pix, l.left, x, y)
		}
	}
	pointer, pointerOK := a.pointerFrame()
	if len(lines) > visible {
		drawVScrollBar(pix, a.media.scroll, a.modBar(top).withPointer(pointer, pointerOK))
	}
	inside := pointerOK && pointer.In(modBackButton)
	a.drawModButton(pix, pushButton{Rect: modBackButton, Label: a.modBackCaption(), Literal: true,
		Hover: inside, Inside: inside, Pressed: f.modUI.backPress}, false)
	return pix, nil
}

// modBar is the mod screen's bar: the shared vertical bar over the body's
// top-line positions.
func (a *App) modBar(top int) vScrollBar {
	return vScrollBar{Rect: image.Rect(modScrollBarX, modBodyBox.Min.Y, modScrollBarX+widgetSpriteSize, modBodyBox.Max.Y),
		Pos: top, Count: a.modMaxTop() + 1}
}

// drawModButton draws a mod page button through the shared painter. A label
// in the page's own text is converted to the install's code page; without
// the install font the fallback text is centred over the bevel.
func (a *App) drawModButton(pix *image.RGBA, b pushButton, convert bool) {
	font := a.flow.menuFont
	if font == nil {
		label := b.Label
		b.Label = ""
		drawPushButton(pix, nil, b)
		a.drawModCentered(pix, label, b.Rect)
		return
	}
	if convert {
		b.Label = a.flow.menuDisplayText(b.Label)
	}
	drawPushButton(pix, font, b)
}

// stepModScreen drives an open mod screen: Up, Down and the wheel scroll, and
// Enter, Escape or a press and release on the back control return.
func (a *App) stepModScreen(in appInput) {
	f := a.flow
	if f.modUI.open < 0 || f.modUI.open >= len(f.modUI.screens) {
		return
	}
	switch {
	case in.Up:
		f.modUI.top = max(0, f.modUI.top-1)
	case in.Down:
		f.modUI.top = min(a.modMaxTop(), f.modUI.top+1)
	case in.WheelY > 0:
		f.modUI.top = max(0, f.modUI.top-modWheelLines)
	case in.WheelY < 0:
		f.modUI.top = min(a.modMaxTop(), f.modUI.top+modWheelLines)
	case in.Enter:
		a.playUISound(UISoundCommonControl)
		f.closeModScreen()
		a.syncViewerLayout()
		return
	}
	p, inFrame := a.windowToNativeFrame(in.CursorX, in.CursorY)
	if a.modMaxTop() > 0 {
		if req, pos := f.modUI.bar.step(a.modBar(f.modUI.top), p, inFrame, in); req != barNone {
			visible := a.modVisibleLines()
			switch req {
			case barSetPos:
				f.modUI.top = pos
			case barLineUp:
				f.modUI.top--
			case barLineDown:
				f.modUI.top++
			case barPageUp:
				f.modUI.top -= visible
			case barPageDown:
				f.modUI.top += visible
			}
			f.modUI.top = min(max(f.modUI.top, 0), a.modMaxTop())
		}
		if f.modUI.bar.active() {
			return
		}
	}
	onBack := inFrame && p.In(modBackButton)
	if in.PrimaryPressed {
		f.modUI.backPress = onBack
	}
	if in.PrimaryReleased {
		pressed := f.modUI.backPress
		f.modUI.backPress = false
		if pressed && onBack {
			a.playUISound(UISoundCommonControl)
			f.closeModScreen()
			a.syncViewerLayout()
		}
	}
}

// modMenuEntryRects are the rectangles of the main menu's mod entries, stacked
// up from the bottom left corner of the frame.
func modMenuEntryRects(n int) []image.Rectangle {
	out := make([]image.Rectangle, n)
	for i := range out {
		y := frame.H - modEntryMargin - (n-i)*modEntryPitch + (modEntryPitch - modEntryH)
		out[i] = image.Rect(modEntryMargin, y, modEntryMargin+modEntryW, y+modEntryH)
	}
	return out
}

// drawModMenuEntries paints the main menu's mod entries onto a composed frame.
func (a *App) drawModMenuEntries(pix *image.RGBA) {
	f := a.flow
	idx := f.modScreenIndexes(false)
	for k, r := range modMenuEntryRects(len(idx)) {
		over := f.modUI.hover == k+1
		label := a.fitModText(f.modUI.screens[idx[k]].MenuLabel, r.Dx()-12)
		a.drawModButton(pix, pushButton{Rect: r, Label: label, Literal: true, Hover: over, Inside: over,
			Pressed: f.modUI.press == k+1}, true)
	}
}

// fitModText clips s to width pixels.
func (a *App) fitModText(s string, width int) string {
	if a.modMeasure(s) <= width {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && a.modMeasure(string(r)+clipMark) > width {
		r = r[:len(r)-1]
	}
	return string(r) + clipMark
}

// stepModMenu drives the main menu's mod entries. It reports whether the entries
// took the pointer event.
func (a *App) stepModMenu(in appInput) bool {
	f := a.flow
	idx := f.modScreenIndexes(false)
	if len(idx) == 0 {
		return false
	}
	hit := 0
	if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok {
		for k, r := range modMenuEntryRects(len(idx)) {
			if p.In(r) {
				hit = k + 1
			}
		}
	}
	if hit != f.modUI.hover {
		f.modUI.hover = hit
		a.hasMenu = false
	}
	switch {
	case in.PrimaryPressed && hit != 0:
		f.modUI.press = hit
		a.hasMenu = false
		a.sel.Move(0)
		return true
	case in.PrimaryReleased && f.modUI.press != 0:
		pressed := f.modUI.press
		f.modUI.press = 0
		a.hasMenu = false
		if pressed == hit {
			a.playUISound(UISoundCommonControl)
			f.openModScreen(idx[hit-1], ScreenMenu, 0)
		}
		return true
	}
	return false
}

// modEntryPoint is a window pixel inside main menu mod entry k (from zero), for
// a caller driving the menu with the pointer.
func (a *App) modEntryPoint(k int) (int, int, error) {
	rects := modMenuEntryRects(len(a.flow.modScreenIndexes(false)))
	if k < 0 || k >= len(rects) {
		return 0, 0, fmt.Errorf("main menu has no mod entry %d", k)
	}
	r := rects[k]
	x, y, ok := a.nativeFrameToWindow(r.Min.Add(r.Max).Div(2))
	if !ok {
		return 0, 0, fmt.Errorf("mod entry %d has no window pixel at this placement", k)
	}
	return x, y, nil
}

// HeadlessModEntryPoint is a window pixel inside main menu mod entry k, counting
// from zero in registration order, or an error naming why there is none.
func (a *App) HeadlessModEntryPoint(k int) (int, int, error) {
	if a == nil || a.flow == nil {
		return 0, 0, fmt.Errorf("headless mod entry point: nil application")
	}
	if a.flow.screen != ScreenMenu {
		return 0, 0, fmt.Errorf("headless mod entry point: screen is %s, not the main menu", a.flow.screen)
	}
	return a.modEntryPoint(k)
}

// HeadlessModBackPoint is a window pixel inside the open mod screen's back
// control.
func (a *App) HeadlessModBackPoint() (int, int, error) {
	if a == nil || a.flow == nil || a.flow.screen != ScreenMod {
		return 0, 0, fmt.Errorf("headless mod back point: no mod screen is open")
	}
	x, y, ok := a.nativeFrameToWindow(modBackButton.Min.Add(modBackButton.Max).Div(2))
	if !ok {
		return 0, 0, fmt.Errorf("the back control has no window pixel at this placement")
	}
	return x, y, nil
}

func init() {
	screenHandlers[ScreenMod] = ScreenHandler{
		Step: func(a *App, in appInput, now time.Time) bool {
			if a.flow.modUI.back == ScreenGameMenu {
				a.holdMapUnderMenu(in, now)
			}
			a.stepModScreen(in)
			return false
		},
		Compose: func(a *App) (*image.RGBA, error) { return a.composeModScreen() },
		Draw: func(a *App) {
			a.hasMenu = false
			pix, _ := a.composeScreen()
			a.writeCanvas(pix)
		},
	}
}

// runModAction takes the confirmed in-game menu action of mod entry i: the
// mission is left for its town, or reopened from its entry point. The map
// screen's advance seam decides; it answers NoticeStay when the mission cannot
// be left, and the menu stays as it was. A mission left this way is not a
// completed one, so no completion movie is requested.
func (f *flow) runModAction(i int) {
	if i < 0 || i >= len(f.modUI.screens) || f.screen != ScreenGameMenu || f.menuBack != ScreenMap || f.advance == nil {
		return
	}
	action := NoticeAbandon
	if f.modUI.screens[i].Kind == ModActionRestart {
		action = NoticeRestart
	}
	dest, msg, open := f.advance(action)
	if dest != NoticeToTown && dest != NoticeToMission {
		f.rebuildGameMenu(gameMenuRoot, 0)
		f.msg = msg
		return
	}
	f.modUI.pending = 0
	f.setMenuUp(false)
	f.setScreen(ScreenMap)
	f.menuPage = gameMenuRoot
	f.menuList = nil
	f.menuContext = GameMenuContext{}
	f.msg = ""
	f.goNoticeDest(dest, msg, open)
}
