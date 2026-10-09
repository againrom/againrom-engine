package ui

import (
	"image"

	"againrom/pkg/render/text"

	"github.com/hajimehoshi/ebiten/v2"
)

// missionTip is the mission tip popup, child 0x10 of the campaign window
// (TRIG-TIPS-087): one at a time at MissionTipRect, built by the shared tip
// panel. It does not stop the world; its Close and checkbox take their own
// presses and its body passes them to the map (TOWN-480).
type missionTip struct {
	text        string
	dress       MissionTipDress
	closeArmed  bool
	pic         *ebiten.Image
	glyphs      []text.DrawCall
	rgba        *image.RGBA
	composedFor TipPanelView
}

// MissionTipDress is what the mission popup draws with: the shared tip panel
// art and font, the Close and checkbox labels (main.txt 127 and 128), and the
// TipsMode word its checkbox reads and writes (MENU-135).
type MissionTipDress struct {
	Art                     *TipPanelArt
	Font                    *text.Font
	CloseLabel, ToggleLabel string
	TipsOn                  func() bool
	SetTipsOn               func(bool)
}

// SetMissionTipDress dresses the mission popup.
func (v *Viewer) SetMissionTipDress(d MissionTipDress) {
	v.mtip.dress, v.mtip.pic = d, nil
}

// ShowMissionTip shows text in the mission popup, replacing an open one.
func (v *Viewer) ShowMissionTip(text string) {
	v.mtip.text, v.mtip.closeArmed = text, false
}

// ClearMissionTip deletes the mission popup.
func (v *Viewer) ClearMissionTip() {
	v.mtip.text, v.mtip.closeArmed = "", false
}

// MissionTip is the mission popup as it would draw now.
func (v *Viewer) MissionTip() TipPanelView {
	if v == nil || v.mtip.text == "" {
		return TipPanelView{}
	}
	d := v.mtip.dress
	font := d.Font
	if font == nil {
		font = v.font
	}
	on := true
	if d.TipsOn != nil {
		on = d.TipsOn()
	}
	view := TipPanelView{
		Rect:        MissionTipRect,
		Text:        v.mtip.text,
		ToggleOn:    on,
		CloseLabel:  d.CloseLabel,
		ToggleLabel: d.ToggleLabel,
		Art:         d.Art,
		Font:        font,
	}
	if v.hasCursor {
		view.Pointer, view.PointerOK = image.Pt(v.cursorX, v.cursorY), true
		view.CloseHover = view.Pointer.In(TipPanelCloseRect(view.Rect))
		view.ClosePressed = view.CloseHover && v.mtip.closeArmed
	}
	return view
}

// missionTipGesture runs the popup's controls for one tick of primary edges
// in frame pixels and reports whether it took the edge: the checkbox writes
// TipsMode on the press, Close deletes the popup on a release that follows its
// own press, and a release on the list is consumed.
func (v *Viewer) missionTipGesture(p image.Point, pressed, released bool) bool {
	view := v.MissionTip()
	if !view.Showing() {
		v.mtip.closeArmed = false
		return false
	}
	if pressed {
		v.mtip.closeArmed = false
		kind, consumed := TipPanelEventAt(view, p, true)
		switch kind {
		case TipControlToggle:
			if v.mtip.dress.SetTipsOn != nil {
				v.mtip.dress.SetTipsOn(!view.ToggleOn)
			}
		case TipControlClose:
			v.mtip.closeArmed = true
		}
		return consumed
	}
	if released {
		armed := v.mtip.closeArmed
		v.mtip.closeArmed = false
		kind, consumed := TipPanelEventAt(view, p, false)
		if kind == TipControlClose && armed {
			v.ClearMissionTip()
			return true
		}
		return consumed || armed
	}
	return false
}

// paintMissionTip composes the popup onto the frame.
func (v *Viewer) paintMissionTip(screen *ebiten.Image) {
	view := v.MissionTip()
	if !view.Showing() {
		return
	}
	r := view.Rect
	capture := beginTextCapture(v.textSmoothingEnabled)
	defer endTextCapture(capture, r.Min)
	if v.mtip.pic == nil || v.mtip.composedFor != view {
		local := view
		local.Rect = r.Sub(r.Min)
		local.Pointer = view.Pointer.Sub(r.Min)
		pic := image.NewRGBA(local.Rect)
		v.mtip.glyphs = text.Record(func() { ComposeTipPanel(pic, local) })
		if v.mtip.pic == nil {
			v.mtip.pic = ebiten.NewImage(r.Dx(), r.Dy())
		}
		v.mtip.pic.WritePixels(pic.Pix)
		v.mtip.rgba = pic
		v.mtip.composedFor = view
	}
	text.Append(v.mtip.glyphs, 0, 0)
	var op ebiten.DrawImageOptions
	op.Filter = ebiten.FilterNearest
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	screen.DrawImage(v.mtip.pic, &op)
	v.canvasLog.over(v.mtip.rgba, r.Min)
}

// ComposeMissionTip composes the viewer's mission popup alone on a
// transparent canvas of the given size, for witnesses without a window.
func ComposeMissionTip(v *Viewer, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	if v != nil {
		ComposeTipPanel(dst, v.MissionTip())
	}
	return dst
}

// MissionTipsMode is the TipsMode word as the popup's checkbox shows it; with
// no dress it reads the original's default, set (MENU-135).
func (v *Viewer) MissionTipsMode() bool {
	if v == nil || v.mtip.dress.TipsOn == nil {
		return true
	}
	return v.mtip.dress.TipsOn()
}
