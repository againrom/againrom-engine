package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
)

// The purse transaction. The inventory's purse cell can be dragged to the
// ground, which requests one share or the whole available balance, or
// double-clicked, which opens a bounded amount editor. Both end in one GoldDrop
// request the tier above turns into a player-addressed command (MENU-090,
// MENU-093). Nothing here debits anything: the viewer holds a preview
// reservation and a draft, and the simulation is the authority on the balance.

const (
	// goldDragShare is the amount a plain drag of the purse requests (MENU-093).
	goldDragShare = 1000
	// goldTextLimit bounds the edit text. The native limit is Unknown; the
	// bound keeps a draft finite.
	goldTextLimit = 24
)

type goldFocus uint8

const (
	goldFocusEdit goldFocus = iota
	goldFocusAction
	goldFocusCancel
	goldFocusCount
)

// GoldDrop is one purse request. When AtSubject is set the cell is the
// selected inventory owner's own cell, which the tier above reads; otherwise
// X and Y are the released world cell.
type GoldDrop struct {
	Amount    uint32
	X, Y      int32
	AtSubject bool
}

type goldState struct {
	open      bool
	text      string
	caret     int
	selStart  int
	selEnd    int
	focus     goldFocus
	press     int
	held      uint32
	request   GoldDrop
	requested bool
}

// HeadlessGoldState is the amount editor and the held preview as a scenario
// sees them.
type HeadlessGoldState struct {
	Open     bool
	Text     string
	Caret    int
	SelStart int
	SelEnd   int
	Focus    string
	Held     uint32
}

var goldFocusNames = [goldFocusCount]string{"edit", "action", "cancel"}

func (v *Viewer) goldModalOpen() bool { return v != nil && v.gold.open }

// goldPurse is the pack index of the purse cell and its previewed quantity.
func (v *Viewer) goldPurse() (int, uint32, bool) {
	for i := len(v.invSubject.Pack) - 1; i >= 0; i-- {
		if i < len(v.invSubject.PackPurse) && v.invSubject.PackPurse[i] && i < len(v.invSubject.PackCount) {
			return i, v.invSubject.PackCount[i], true
		}
	}
	return 0, 0, false
}

func (v *Viewer) isPurseCell(idx int) bool {
	return idx >= 0 && idx < len(v.invSubject.PackPurse) && v.invSubject.PackPurse[idx]
}

// goldAvailable is the previewed purse less whatever a drag currently holds.
func (v *Viewer) goldAvailable() uint32 {
	_, n, ok := v.goldPurse()
	if !ok || v.gold.held >= n {
		return 0
	}
	return n - v.gold.held
}

// armGoldDrag reserves a drag's share in the preview. The share is one
// thousand, or the whole available balance when shift is held; it never
// exceeds the balance.
func (v *Viewer) armGoldDrag(idx int, shift bool) {
	v.gold.held = 0
	if !v.isPurseCell(idx) || idx >= len(v.invSubject.PackCount) {
		return
	}
	n := v.invSubject.PackCount[idx]
	if shift {
		v.gold.held = n
		return
	}
	v.gold.held = min(n, goldDragShare)
}

// releaseGoldDrag spends the held share as a request at a world cell, or
// returns it to the preview when the release is not on the ground.
func (v *Viewer) releaseGoldDrag(onGround bool, x, y int32) {
	held := v.gold.held
	v.gold.held = 0
	if onGround && held > 0 {
		v.gold.request, v.gold.requested = GoldDrop{Amount: held, X: x, Y: y}, true
	}
}

// TakeGoldDrop reads the pending purse request and clears it in the same
// statement.
func (v *Viewer) TakeGoldDrop() (GoldDrop, bool) {
	if v == nil || !v.gold.requested {
		return GoldDrop{}, false
	}
	r := v.gold.request
	v.gold.request, v.gold.requested = GoldDrop{}, false
	return r, true
}

// GoldHeld is the amount the viewer reserves from the previewed purse while a
// drag carries it.
func (v *Viewer) GoldHeld() uint32 {
	if v == nil {
		return 0
	}
	return v.gold.held
}

// openGoldModal opens the editor with the literal text 0. The reopened caret
// and selection are Unknown; each open starts from the constructor state.
func (v *Viewer) openGoldModal() {
	if _, n, ok := v.goldPurse(); !ok || n == 0 {
		return
	}
	v.gold.open, v.gold.text = true, "0"
	v.gold.caret, v.gold.selStart, v.gold.selEnd = 0, 0, 0
	v.gold.focus, v.gold.press = goldFocusEdit, 0
}

// cancelGold drops every unsubmitted gold draft: the open editor, a held drag
// share and the drag that carried it. Nothing reaches the simulation.
func (v *Viewer) cancelGold() {
	if v == nil {
		return
	}
	g := &v.gold
	pending, requested := g.request, g.requested
	*g = goldState{request: pending, requested: requested}
	if v.dragCandKind == dragFromPack {
		v.dragActive, v.dragCandKind, v.dragIcon = false, dragNone, nil
		v.invEquipTap = false
	}
}

func (g *goldState) clampEdit() {
	n := len(g.text)
	g.caret = max(0, min(g.caret, n))
	g.selStart = max(0, min(g.selStart, n))
	g.selEnd = max(0, min(g.selEnd, n))
}

// move places the caret. Without shift the selection end follows the stored
// selection start; with shift the endpoint the caret left follows it.
func (g *goldState) move(to int, shift bool) {
	g.clampEdit()
	old := g.caret
	g.caret = max(0, min(to, len(g.text)))
	if !shift {
		g.selEnd = g.selStart
		return
	}
	if g.selStart == g.selEnd {
		g.selStart, g.selEnd = old, old
	}
	switch {
	case old == g.selEnd:
		g.selEnd = g.caret
	case old == g.selStart:
		g.selStart = g.caret
	default:
		g.selStart, g.selEnd = min(old, g.caret), max(old, g.caret)
	}
	if g.selStart > g.selEnd {
		g.selStart, g.selEnd = g.selEnd, g.selStart
	}
}

// backspace removes the byte before a positive caret and changes neither
// selection endpoint. It reports whether the edit handled the key; at caret
// zero it did not.
func (g *goldState) backspace() bool {
	g.clampEdit()
	if g.caret == 0 {
		return false
	}
	g.text = g.text[:g.caret-1] + g.text[g.caret:]
	g.caret--
	return true
}

// deleteForward removes a positive selection, or else the byte at the caret.
func (g *goldState) deleteForward() {
	g.clampEdit()
	if g.selEnd-g.selStart > 0 {
		g.text = g.text[:g.selStart] + g.text[g.selEnd:]
		g.caret, g.selEnd = g.selStart, g.selStart
		return
	}
	if g.caret < len(g.text) {
		g.text = g.text[:g.caret] + g.text[g.caret+1:]
	}
}

// insert places printable ASCII at the caret, replacing a positive selection.
// Other characters are dropped; the text stays within goldTextLimit.
func (g *goldState) insert(s string) {
	for _, r := range s {
		if r < 0x20 || r > 0x7e {
			continue
		}
		g.clampEdit()
		if g.selEnd-g.selStart > 0 {
			g.text = g.text[:g.selStart] + g.text[g.selEnd:]
			g.caret, g.selEnd = g.selStart, g.selStart
		}
		if len(g.text) >= goldTextLimit {
			continue
		}
		g.text = g.text[:g.caret] + string(r) + g.text[g.caret:]
		g.caret++
		g.selStart, g.selEnd = g.caret, g.caret
	}
}

// parseGoldAmount reads the edit text as a decimal number and truncates it
// toward zero. The native parser's results for malformed, percent, exponent,
// locale and out-of-range text are Unknown (MENU-092); this build accepts an
// optional plus sign, digits and an optional fraction, saturates at the
// 32-bit maximum and refuses everything else. A refusal is amount zero.
func parseGoldAmount(text string) uint32 {
	t := strings.TrimSpace(text)
	t = strings.TrimPrefix(t, "+")
	whole, frac, _ := strings.Cut(t, ".")
	if whole == "" && frac == "" {
		return 0
	}
	for _, part := range []string{whole, frac} {
		for i := 0; i < len(part); i++ {
			if part[i] < '0' || part[i] > '9' {
				return 0
			}
		}
	}
	var n uint64
	for i := 0; i < len(whole); i++ {
		n = n*10 + uint64(whole[i]-'0')
		if n > math.MaxUint32 {
			return math.MaxUint32
		}
	}
	return uint32(n)
}

// submitGold is the action: it requests the admitted amount at the selected
// owner's cell and closes the editor. An amount of zero, or one the previewed
// purse cannot cover at all, requests nothing.
func (v *Viewer) submitGold() {
	amount := min(parseGoldAmount(v.gold.text), v.goldAvailable())
	v.gold = goldState{request: v.gold.request, requested: v.gold.requested}
	if amount > 0 {
		v.gold.request, v.gold.requested = GoldDrop{Amount: amount, AtSubject: true}, true
	}
}

// The editor's rectangle is the original's own (MENU-COMBAT-017): 296x168 at x 100,
// bottom edge 32 above the frame's.
func (v *Viewer) goldModalBox() image.Rectangle {
	return image.Rect(100, v.frameH-200, 396, v.frameH-32)
}

func (v *Viewer) goldModalControls() (edit, action, cancel image.Rectangle) {
	box := v.goldModalBox()
	edit = image.Rect(box.Min.X+16, box.Min.Y+56, box.Max.X-16, box.Min.Y+84)
	action = image.Rect(box.Min.X+16, box.Max.Y-48, box.Min.X+16+120, box.Max.Y-16)
	cancel = image.Rect(box.Max.X-16-120, box.Max.Y-48, box.Max.X-16, box.Max.Y-16)
	return edit, action, cancel
}

func (v *Viewer) goldControlAt(p image.Point) (goldFocus, bool) {
	edit, action, cancel := v.goldModalControls()
	switch {
	case p.In(edit):
		return goldFocusEdit, true
	case p.In(action):
		return goldFocusAction, true
	case p.In(cancel):
		return goldFocusCancel, true
	}
	return 0, false
}

// goldKeys are the keyboard edges the editor reads.
type goldKeys struct {
	Typed                                  string
	Backspace, Delete, Home, End           bool
	Left, Right, Tab, Enter, Escape, Shift bool
}

// stepGold applies one frame of editor input. Keys reach the edit only while
// it holds focus; a focused button answers Enter alone. Escape is the cancel
// alias, Enter the action alias except on a focused cancel button.
func (v *Viewer) stepGold(k goldKeys, unfocused bool) {
	if !v.gold.open || unfocused {
		return
	}
	g := &v.gold
	if k.Escape {
		v.cancelGold()
		return
	}
	if k.Tab {
		step := goldFocus(1)
		if k.Shift {
			step = goldFocusCount - 1
		}
		g.focus = (g.focus + step) % goldFocusCount
	}
	if g.focus == goldFocusEdit {
		switch {
		case k.Home:
			g.move(0, k.Shift)
		case k.End:
			g.move(len(g.text), k.Shift)
		case k.Left:
			g.move(g.caret-1, k.Shift)
		case k.Right:
			g.move(g.caret+1, k.Shift)
		}
		if k.Delete {
			g.deleteForward()
		}
		if k.Backspace {
			g.backspace()
		}
		if k.Typed != "" {
			g.insert(k.Typed)
		}
	}
	if k.Enter {
		if g.focus == goldFocusCancel {
			v.cancelGold()
		} else {
			v.submitGold()
		}
	}
}

// stepGoldPointer applies the primary button's edges to the editor. A press
// focuses a control and a release on the same control activates a button.
func (v *Viewer) stepGoldPointer(x, y int, pressed, released bool) {
	if !v.gold.open {
		return
	}
	fx, fy := v.windowToFrame(x, y)
	c, hit := v.goldControlAt(image.Pt(fx, fy))
	if pressed {
		v.gold.press = 0
		if hit {
			v.gold.focus, v.gold.press = c, int(c)+1
		}
	}
	if released {
		armed := v.gold.press - 1
		v.gold.press = 0
		if hit && armed == int(c) {
			switch c {
			case goldFocusAction:
				v.submitGold()
			case goldFocusCancel:
				v.cancelGold()
			}
		}
	}
}

// HeadlessGold reports the editor and the held preview. The second result is
// false while neither exists.
func (a *App) HeadlessGold() (HeadlessGoldState, bool) {
	if a == nil || a.flow == nil || a.flow.viewer == nil {
		return HeadlessGoldState{}, false
	}
	g := a.flow.viewer.gold
	return HeadlessGoldState{Open: g.open, Text: g.text, Caret: g.caret, SelStart: g.selStart, SelEnd: g.selEnd,
		Focus: goldFocusNames[g.focus%goldFocusCount], Held: g.held}, g.open || g.held != 0
}

// HeadlessGoldControlPoint is a window pixel inside one editor control
// ("edit", "action" or "cancel").
func (a *App) HeadlessGoldControlPoint(name string) (int, int, error) {
	v, err := a.headlessViewer()
	if err != nil {
		return 0, 0, err
	}
	edit, action, cancel := v.goldModalControls()
	r := map[string]image.Rectangle{"edit": edit, "action": action, "cancel": cancel}[name]
	if r.Empty() || !v.gold.open {
		return 0, 0, fmt.Errorf("headless gold control %q: editor open %v", name, v.gold.open)
	}
	return v.frameToWindow(r.Min.Add(r.Max).Div(2), "headless gold control")
}

var goldFill = color.RGBA{R: 0x10, G: 0x12, B: 0x18, A: 0xff}

// Authored labels. The installed strings of the Drop Gold dialog are not
// established, so the editor states its own English words.
const (
	goldTitleWord  = "Drop gold"
	goldCancelWord = "Cancel"
)

func (v *Viewer) goldModalPresent() (*image.RGBA, image.Point, bool) {
	if !v.gold.open || v.frameW <= 0 || v.frameH <= 0 {
		return nil, image.Point{}, false
	}
	box := v.goldModalBox()
	img := image.NewRGBA(image.Rect(0, 0, box.Dx(), box.Dy()))
	drawFrame(img, panelFrame(image.Rectangle{Max: box.Size()}, goldFill, invBorder))
	edit, action, cancel := v.goldModalControls()
	edit, action, cancel = edit.Sub(box.Min), action.Sub(box.Min), cancel.Sub(box.Min)
	f := v.cardFont()
	g := v.gold
	g.clampEdit()
	e := editField{Rect: edit, Focus: g.focus == goldFocusEdit, Phase: v.blink.on()}
	label := func(image.Point) {}
	if f != nil {
		_, e.TextH = f.Measure("0")
		e.Caret, _ = f.Measure(g.text[:g.caret])
		if g.selEnd > g.selStart {
			e.SelFrom, _ = f.Measure(g.text[:g.selStart])
			e.SelTo, _ = f.Measure(g.text[:g.selEnd])
		}
		draw := func(s string, at image.Point) {
			f.DrawFlat(img, s, at.X+shopPriceShadow, at.Y+shopPriceShadow, messageShadowColor)
			f.Draw(img, s, at.X, at.Y, shopPriceInk)
		}
		label = func(at image.Point) { draw(g.text, at) }
		draw(goldTitleWord, image.Pt(16, 16))
	}
	drawEditField(img, e, label)
	ok := v.Words().NoticeButton
	if ok == "" {
		ok = AuthoredNoticeButton
	}
	pointer, pointerOK := image.Pt(v.cursorX, v.cursorY).Sub(box.Min), v.hasCursor
	for i, b := range []struct {
		r image.Rectangle
		s string
	}{{action, ok}, {cancel, goldCancelWord}} {
		inside := pointerOK && pointer.In(b.r)
		drawPushButton(img, f, pushButton{Rect: b.r, Label: b.s, Literal: true, Hover: inside, Inside: inside,
			Focus: g.focus == goldFocusAction+goldFocus(i), Pressed: g.press == int(goldFocusAction)+i+1})
	}
	return img, box.Min, true
}

// previewSubject is the subject the pack bar draws: the purse shows its
// quantity less the share a drag holds. The stored subject is never changed.
func (v *Viewer) previewSubject() InventorySubject {
	s := v.invSubject
	idx, n, ok := v.goldPurse()
	if !ok || v.gold.held == 0 {
		return s
	}
	counts := append([]uint32(nil), s.PackCount...)
	counts[idx] = n - min(n, v.gold.held)
	s.PackCount = counts
	return s
}

// packCellTap is the matching second press on a pack cell: an item cell raises
// the equip request, the purse cell opens the amount editor.
func (v *Viewer) packCellTap(idx int) {
	if v.isPurseCell(idx) {
		v.openGoldModal()
		return
	}
	v.invEquipRequest = idx + 1
}

// stepGoldModal feeds one frame of input to the open editor and returns what
// the map may still see of it: the pointer position and focus, with no key and
// no button edge. The editor owns every key while it is open, so a typed digit
// reaches no map binding and a Backspace the edit declines clears no message.
// Alt+Backspace is a system key message that reaches no GUI child (MENU-082).
func (a *App) stepGoldModal(v *Viewer, in appInput) appInput {
	v.stepGold(goldKeys{
		Typed: in.Typed, Backspace: in.Backspace && !in.Viewer.Alt, Delete: in.Delete, Home: in.Home, End: in.End,
		Left: in.Left, Right: in.Right, Tab: in.PaneMode, Enter: in.Enter, Shift: in.ShiftHeld || in.Viewer.Shift,
	}, in.Unfocused)
	if !in.Unfocused {
		v.stepGoldPointer(in.CursorX, in.CursorY, in.PrimaryPressed, in.PrimaryReleased)
	}
	return appInput{
		CursorX: in.CursorX, CursorY: in.CursorY, Unfocused: in.Unfocused, Close: in.Close,
		Viewer: Input{Unfocused: in.Viewer.Unfocused, CursorX: in.Viewer.CursorX, CursorY: in.Viewer.CursorY},
	}
}
