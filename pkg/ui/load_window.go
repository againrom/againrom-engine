package ui

import (
	"image"
	"image/color"
	"image/draw"
	"time"
)

type LoadWindowWords struct{ Title, Subtitle, OK, Delete, Cancel, Confirm string }

func defaultLoadWindowWords() LoadWindowWords {
	return LoadWindowWords{"Load Saved Game", "Load the Game", "OK", "Delete", "Cancel", "Delete selected saved game?"}
}

type loadWindow struct {
	quick         QuickSaveControls
	words         LoadWindowWords
	press         loadTarget
	drag          bool
	grab          int
	confirm       bool
	canDelete     func(string) bool
	prepareDelete func(string) (func() error, error)
	remove        func() error
	lastName      string
	lastClick     time.Time
	// completed counts the loads that replaced the session, so presentation can
	// tell a fresh load from an unchanged screen. It is never reset.
	completed uint32
}

func (w *loadWindow) resetClick() { w.lastName, w.lastClick = "", time.Time{} }

func (w *loadWindow) resetPointer() {
	w.press, w.drag, w.grab = loadTarget{}, false, 0
}

func (a *App) SetLoadWindowWords(w LoadWindowWords) { a.flow.loadUI.words = w }
func (a *App) SetSaveDelete(can func(string) bool, prepare func(string) (func() error, error)) {
	a.flow.loadUI.canDelete, a.flow.loadUI.prepareDelete = can, prepare
}

const loadVisibleRows = 10

type loadHitKind uint8

const (
	loadHitNone loadHitKind = iota
	loadHitRow
	loadHitButton
	loadHitUp
	loadHitDown
	loadHitTrack
)

type loadTarget struct {
	kind  loadHitKind
	index int
}

var (
	loadPanel        = image.Rect(80, 56, 560, 420)
	loadRowsBox      = image.Rect(122, 152, 502, 342)
	loadTrack        = image.Rect(504, 176, 528, 318)
	loadMessageBox   = image.Rect(122, 344, 522, 375)
	loadSelectedText = color.RGBA{218, 183, 71, 255}
	loadDisabledText = color.RGBA{112, 122, 115, 255}
)

func loadRowRect(i int) image.Rectangle    { return image.Rect(122, 152+i*19, 502, 171+i*19) }
func loadButtonRect(i int) image.Rectangle { return image.Rect(156+i*124, 380, 252+i*124, 404) }

func loadThumb(list *Picker) image.Rectangle {
	y := loadTrack.Min.Y
	if list != nil && len(list.Rows()) > 1 {
		y += list.Selection() * (loadTrack.Dy() - 24) / (len(list.Rows()) - 1)
	}
	return image.Rect(loadTrack.Min.X, y, loadTrack.Max.X, y+24)
}

func loadHit(p image.Point, list *Picker) loadTarget {
	for i := 0; i < 3; i++ {
		if p.In(loadButtonRect(i)) {
			return loadTarget{kind: loadHitButton, index: i}
		}
	}
	if p.In(image.Rect(504, 152, 528, 176)) {
		return loadTarget{kind: loadHitUp}
	}
	if p.In(image.Rect(504, 318, 528, 342)) {
		return loadTarget{kind: loadHitDown}
	}
	if p.In(loadTrack) {
		return loadTarget{kind: loadHitTrack}
	}
	if list != nil && p.In(loadRowsBox) {
		top, n := list.Visible()
		i := (p.Y - loadRowsBox.Min.Y) / 19
		if i < n {
			return loadTarget{kind: loadHitRow, index: top + i}
		}
	}
	return loadTarget{}
}

func (f *flow) canDeleteLoad() bool {
	if f.loadList == nil || f.loadUI.canDelete == nil || f.loadUI.prepareDelete == nil {
		return false
	}
	i := f.loadList.Selection()
	return i >= 0 && i < len(f.saves) && f.loadUI.canDelete(f.saves[i].Name)
}

func (a *App) acceptLoad() {
	f := a.flow
	f.loadUI.resetPointer()
	f.loadUI.resetClick()
	if !f.loadUI.confirm {
		a.activatePicker(f.loadList, f.chooseLoad)
		a.syncViewerLayout()
		return
	}
	if !f.canDeleteLoad() {
		f.loadUI.confirm = false
		return
	}
	i := f.loadList.Selection()
	if f.loadUI.remove == nil {
		return
	}
	if err := f.loadUI.remove(); err != nil {
		f.msg = err.Error()
		return
	}
	back := f.loadBack
	f.openLoad(back)
	if f.loadList != nil {
		f.loadList.Move(i)
	}
}

func (f *flow) confirmDeleteLoad() {
	if !f.canDeleteLoad() {
		return
	}
	remove, err := f.loadUI.prepareDelete(f.saves[f.loadList.Selection()].Name)
	if err != nil {
		f.msg = err.Error()
		return
	}
	f.loadUI.remove, f.loadUI.confirm, f.msg = remove, true, ""
}

func (a *App) stepLoadWindow(in appInput, now time.Time) {
	f := a.flow
	l := f.loadList
	if in.Unfocused || l == nil {
		f.loadUI.resetPointer()
		f.loadUI.resetClick()
		return
	}
	if in.Enter {
		a.acceptLoad()
		return
	}
	if in.Delete && f.canDeleteLoad() {
		f.loadUI.resetPointer()
		f.loadUI.resetClick()
		f.confirmDeleteLoad()
		return
	}
	if !f.loadUI.confirm {
		if in.Up || in.Down || in.Home || in.End || in.WheelY != 0 {
			f.loadUI.resetClick()
		}
		switch {
		case in.Up:
			l.Move(-1)
		case in.Down:
			l.Move(1)
		case in.Home:
			l.Select(0)
		case in.End:
			l.Select(len(l.Rows()) - 1)
		case in.WheelY > 0:
			l.Move(-3)
		case in.WheelY < 0:
			l.Move(3)
		}
	}
	p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY)
	var hit loadTarget
	if ok {
		hit = loadHit(p, l)
	}
	if in.PrimaryPressed {
		f.loadUI.resetPointer()
		f.loadUI.press = hit
		if hit.kind != loadHitRow {
			f.loadUI.resetClick()
		}
		if !f.loadUI.confirm && hit.kind == loadHitTrack && len(l.Rows()) > 1 && p.In(loadThumb(l)) {
			f.loadUI.drag, f.loadUI.grab = true, p.Y-loadThumb(l).Min.Y
			return
		}
	}
	if f.loadUI.drag {
		if !ok || in.PrimaryReleased || !in.Viewer.PrimaryDown {
			f.loadUI.resetPointer()
			return
		}
		span := loadTrack.Dy() - 24
		top := min(max(p.Y-f.loadUI.grab-loadTrack.Min.Y, 0), span)
		l.Select((top*(len(l.Rows())-1) + span/2) / span)
		return
	}
	if !in.PrimaryReleased {
		return
	}
	pressed := f.loadUI.press
	f.loadUI.resetPointer()
	if hit.kind == loadHitNone || hit != pressed {
		f.loadUI.resetClick()
		return
	}
	switch hit.kind {
	case loadHitButton:
		switch hit.index {
		case 0:
			a.acceptLoad()
		case 1:
			if !f.loadUI.confirm && f.canDeleteLoad() {
				f.confirmDeleteLoad()
			}
		case 2:
			if f.loadUI.confirm {
				f.loadUI.confirm = false
				f.msg = ""
			} else {
				f.closeLoad()
			}
		}
	default:
		if f.loadUI.confirm {
			return
		}
		switch hit.kind {
		case loadHitUp:
			l.Move(-1)
		case loadHitDown:
			l.Move(1)
		case loadHitTrack:
			l.Select((p.Y - loadTrack.Min.Y) * max(0, len(l.Rows())-1) / loadTrack.Dy())
		case loadHitRow:
			l.Select(hit.index)
			if hit.index >= len(f.saves) {
				f.loadUI.resetClick()
				return
			}
			name := f.saves[hit.index].Name
			elapsed := now.Sub(f.loadUI.lastClick)
			if name == f.loadUI.lastName && !f.loadUI.lastClick.IsZero() && elapsed >= 0 && elapsed <= 500*time.Millisecond {
				a.acceptLoad()
				return
			}
			f.loadUI.lastName, f.loadUI.lastClick = name, now
		}
	}
}

func (a *App) drawLoadScroll(dst *image.RGBA, l *Picker) {
	if !drawScrollbarSkin(dst, a.media.scroll, image.Rect(504, 152, 528, 176), loadTrack,
		image.Rect(504, 318, 528, 342), loadThumb(l)) {
		drawMovieBox(dst, loadTrack, false)
	}
}

func drawScrollbarSkin(dst *image.RGBA, frames []*image.RGBA, up, track, down, thumb image.Rectangle) bool {
	for _, index := range []int{16, 18, 19, 20} {
		if index >= len(frames) || frames[index] == nil || frames[index].Bounds().Empty() {
			return false
		}
	}
	for y := track.Min.Y; y < track.Max.Y; y += frames[19].Bounds().Dy() {
		copyScrollbarFit(dst, frames[19], image.Rect(track.Min.X, y, track.Max.X, y+frames[19].Bounds().Dy()), track)
	}
	copyScrollbarFit(dst, frames[18], up, up)
	copyScrollbarFit(dst, frames[20], down, down)
	copyScrollbarFit(dst, frames[16], thumb, thumb)
	return true
}

func copyScrollbarFit(dst, src *image.RGBA, target, clip image.Rectangle) {
	if target.Empty() {
		return
	}
	b := src.Bounds()
	if b.Size() == target.Size() {
		copyNativeOver(dst, src, target.Min, clip.Intersect(target))
		return
	}
	fit := image.NewRGBA(target)
	for y := target.Min.Y; y < target.Max.Y; y++ {
		for x := target.Min.X; x < target.Max.X; x++ {
			fit.SetRGBA(x, y, src.RGBAAt(b.Min.X+(x-target.Min.X)*b.Dx()/target.Dx(), b.Min.Y+(y-target.Min.Y)*b.Dy()/target.Dy()))
		}
	}
	visible := target.Intersect(clip)
	draw.Draw(dst, visible, fit, visible.Min, draw.Over)
}
