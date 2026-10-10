package ui

import (
	"image"
	"image/draw"

	"againrom/pkg/render/text"
)

type TipPanelArt struct {
	Frame         *DialogFrame
	GemOff, GemOn image.Image
}

type TipControlKind uint8

const (
	TipControlNone TipControlKind = iota
	TipControlClose
	TipControlToggle
)

type TipPanelView struct {
	Rect                     image.Rectangle
	Revision                 uint64
	Text                     string
	ToggleOn                 bool
	CloseLabel, ToggleLabel  string
	Art                      *TipPanelArt
	Font                     *text.Font
	Pointer                  image.Point
	PointerOK                bool
	CloseHover, ClosePressed bool
}

func (v TipPanelView) Showing() bool {
	return v.Art != nil && v.Font != nil && v.Text != "" && !v.Rect.Empty()
}

func (v TipPanelView) Covers(p image.Point) bool { return v.Showing() && p.In(v.Rect) }

const (
	tipPanelInset      = 20
	tipPanelButtonRowH = 18
	tipPanelGemSize    = 16
	tipPanelChromeH    = 61
)

var (
	TownTipRect   = image.Rect(328, 0, 640, 200)
	TavernTipRect = image.Rect(160, 0, 472, 200)
	SchoolTipRect = image.Rect(0, 0, 456, 200)
	// MissionTipRect is the mission popup in the campaign window
	// (TRIG-TIPS-087).
	MissionTipRect = image.Rect(10, 20, 370, 188)
)

// TipPanelTextRect is where the text is drawn and its fit is tested: the
// list child itself (MENU-137).
func TipPanelTextRect(r image.Rectangle) image.Rectangle { return TipPanelListRect(r) }

func TipPanelCloseRect(r image.Rectangle) image.Rectangle {
	return image.Rect(r.Max.X-120, r.Max.Y-40, r.Max.X-40, r.Max.Y-22)
}

func TipPanelToggleRect(r image.Rectangle) image.Rectangle {
	return image.Rect(r.Min.X+40, r.Max.Y-40, r.Max.X-124, r.Max.Y-24)
}

func TipPanelControlAt(v TipPanelView, p image.Point) (TipControlKind, bool) {
	if !v.Showing() || !p.In(v.Rect) {
		return TipControlNone, false
	}
	if p.In(TipPanelCloseRect(v.Rect)) {
		return TipControlClose, true
	}
	if p.In(TipPanelToggleRect(v.Rect)) {
		return TipControlToggle, true
	}
	return TipControlNone, true
}

// TipPanelListRect is the popup's list child, (0x14,0x18)-(W-0x1c,H-0x24)
// of the panel (MENU-137).
func TipPanelListRect(r image.Rectangle) image.Rectangle {
	return image.Rectangle{Min: image.Pt(r.Min.X+20, r.Min.Y+24), Max: image.Pt(r.Max.X-28, r.Max.Y-36)}
}

func TipPanelEventAt(v TipPanelView, p image.Point, down bool) (TipControlKind, bool) {
	if !v.Showing() || !p.In(v.Rect) {
		return TipControlNone, false
	}
	if !down && p.In(TipPanelListRect(v.Rect)) {
		return TipControlNone, true
	}
	if p.In(TipPanelCloseRect(v.Rect)) {
		return TipControlClose, true
	}
	if p.In(TipPanelToggleRect(v.Rect)) {
		return TipControlToggle, down
	}
	return TipControlNone, false
}

func TipPanelFits(v TipPanelView) bool {
	return !v.Showing() || tipPanelTextFits(v.Font, v.Text, v.Rect)
}

func tipPanelTextFits(font *text.Font, s string, r image.Rectangle) bool {
	if TipPanelTextRect(r).Dx() <= 0 {
		return false
	}
	return tipPanelLinesFit(font, tipTextLines(font, s, TipPanelTextRect(r).Dx()), r)
}

func tipPanelLinesFit(font *text.Font, lines []dialogueLine, r image.Rectangle) bool {
	if font == nil || font.Height()+2 <= 0 {
		return false
	}
	if len(lines) == 0 {
		return true
	}
	return TipPanelTextRect(r).Min.Y+(len(lines)-1)*(font.Height()+2)+font.Height()+1 <= TipPanelTextRect(r).Max.Y
}

func TipPanelFitHeight(font *text.Font, s string, width int) int {
	if font == nil || s == "" {
		return 104
	}
	if width <= 48 {
		return 456
	}
	lines := tipTextLines(font, s, width-48)
	h := tipPanelChromeH + font.Height()
	if len(lines) > 0 {
		h += (len(lines) - 1) * (font.Height() + 2)
	}
	return min(456, max(104, 72+((max(0, h-72)+31)/32)*32))
}

func TipPanelShrinkRect(bounds image.Rectangle, font *text.Font, s string) image.Rectangle {
	if font == nil || s == "" {
		return bounds
	}
	h := TipPanelFitHeight(font, s, bounds.Dx())
	return image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Min.Y+h)
}

func ComposeTipPanel(dst *image.RGBA, v TipPanelView) {
	if dst == nil || !v.Showing() {
		return
	}
	drawFrame(dst, frameSpec{Kind: frameTip, Rect: v.Rect, Art: v.Art.Frame})
	drawTipText(dst, v.Font, tipTextLines(v.Font, v.Text, TipPanelTextRect(v.Rect).Dx()), TipPanelTextRect(v.Rect))
	closeLabel := v.CloseLabel
	if closeLabel == "" {
		closeLabel = AuthoredTipClose
	}
	raise := 0
	if len(v.Font.Glyphs) != 0 {
		raise = v.Font.Glyphs[0].Width / 2
	}
	drawPushButton(dst, v.Font, pushButton{Rect: TipPanelCloseRect(v.Rect), Label: closeLabel, Literal: true, LabelRaise: raise, Ramp: pushButtonDialogue, Hover: v.CloseHover, Inside: v.CloseHover, Pressed: v.ClosePressed})
	toggleLabel := v.ToggleLabel
	if toggleLabel == "" {
		toggleLabel = AuthoredTipShowNext
	}
	r := TipPanelToggleRect(v.Rect)
	pic := v.Art.GemOff
	if v.ToggleOn {
		pic = v.Art.GemOn
	}
	drawChoiceGroup(dst, nil, choiceGroup{Kind: choiceTipCheck, Rect: r, Labels: []string{toggleLabel}, Off: pic, On: pic})
	ink := dialogueButtonInk
	if v.PointerOK && v.Pointer.In(r) {
		ink = pushButtonHoverInk
	}
	clip := image.Rect(r.Min.X+22, r.Min.Y, r.Max.X, r.Max.Y).Intersect(dst.Bounds())
	v.Font.Draw(dst.SubImage(clip).(*image.RGBA), toggleLabel, r.Min.X+22, r.Min.Y+3, ink)
}

// tileBlitOver repeats a keyed source rectangle inside target.
func tileBlitOver(dst *image.RGBA, target image.Rectangle, src image.Image, srcRect image.Rectangle) {
	sw, sh := srcRect.Dx(), srcRect.Dy()
	if sw <= 0 || sh <= 0 {
		return
	}
	for y := target.Min.Y; y < target.Max.Y; y += sh {
		for x := target.Min.X; x < target.Max.X; x += sw {
			tile := image.Rect(x, y, x+sw, y+sh).Intersect(target)
			if tile.Empty() {
				continue
			}
			draw.Draw(dst, tile, src, srcRect.Min, draw.Over)
		}
	}
}
