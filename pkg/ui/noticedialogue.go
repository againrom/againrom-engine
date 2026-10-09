package ui

import (
	"image"
	"image/color"
	"math/big"
	"strings"

	"againrom/pkg/render/text"
)

const (
	dialogueIndent                 = 10
	dialogueTextShadow             = 1
	noticeSurfaceW, noticeSurfaceH = 88, 108
	noticeFillX, noticeFillY       = 8, 7
	noticeFillW, noticeFillH       = 72, 94
	dialogueButtonShadow           = 2
	dialogueLabelRaise             = 8
)

var (
	dialogueInk        = color.RGBA{255, 255, 255, 255}
	dialogueButtonInk  = color.RGBA{185, 159, 73, 255}
	dialogueBevelLight = color.RGBA{41, 69, 63, 255}
	dialogueBevelDark  = color.RGBA{7, 12, 9, 255}
)

const dialogueTrimBytes = " \t\r\n\v\f"

type dialogueLine struct {
	text         string
	indent       bool
	justify      bool
	paragraphEnd bool
}

// dialogueWrap preserves the selected wrapper's markers. A nonprogressing
// remainder is emitted once, whole, so custom text remains reachable (DIV-1619).
func dialogueWrap(f *text.Font, s string, width int) []dialogueLine {
	if f == nil || len(f.Glyphs) == 0 || width <= 0 || s == "" {
		return nil
	}
	var raw []string
	var markers []bool
	appendLine := func(line string, end bool) { raw = append(raw, line); markers = append(markers, end) }
	var pieces []string
	for remaining := s; remaining != ""; {
		end := strings.Index(remaining, "\r\n")
		if end < 0 {
			pieces = append(pieces, remaining)
			break
		}
		pieces = append(pieces, strings.TrimLeft(remaining[:end], dialogueTrimBytes))
		remaining = strings.TrimLeft(remaining[end+2:], dialogueTrimBytes)
	}
	for _, para := range pieces {
		rest := strings.Trim(para, dialogueTrimBytes)
		// Bare-CR paragraph replay does not progress; preserve its remainder.
		if strings.Contains(para, "\r") {
			rest = para
		}
		if rest == "" {
			appendLine("", false)
			continue
		}
		for rest != "" {
			if strings.Contains(rest, "\r") {
				appendLine(rest, false)
				break
			}
			if f.Advance(rest) < width {
				appendLine(rest+" \r", true)
				break
			}
			words := strings.FieldsFunc(rest, func(r rune) bool { return r == ' ' || r == '\t' })
			if len(words) == 0 {
				appendLine("", false)
				break
			}
			if f.Advance(words[0]) > width {
				appendLine(words[0], false)
				rest = strings.Join(words[1:], " ")
				continue
			}
			n := 0
			for n < len(words) && f.Advance(strings.Join(words[:n+1], " ")+" ") < width {
				n++
			}
			if n == 0 {
				appendLine(rest, false)
				break
			}
			appendLine(strings.Join(words[:n], " ")+" ", false)
			rest = strings.Join(words[n:], " ")
		}
	}
	out := make([]dialogueLine, len(raw))
	for i, line := range raw {
		out[i] = dialogueLine{text: line, indent: i == 0 || markers[i-1], justify: i < len(raw)-1 && !markers[i], paragraphEnd: markers[i]}
	}
	return out
}

func (ln dialogueLine) visible() string {
	if ln.paragraphEnd {
		return strings.TrimSuffix(ln.text, "\r")
	}
	return ln.text
}

// drawDialogueRun draws s at (x, y) as the original draws every run of text:
// a flat shadow one pixel right and down, then the ink over it.
func drawDialogueRun(dst *image.RGBA, f *text.Font, s string, x, y int, ink color.RGBA) {
	f.DrawFlat(dst, s, x+dialogueTextShadow, y+dialogueTextShadow, messageShadowColor)
	f.Draw(dst, s, x, y, ink)
}

// DialogueArithmetic supplies intermediate precision and rounding. Zero selects
// nearest-even PC53; native FPU configuration is not established (DIV-1620).
type DialogueArithmetic struct {
	Precision uint
	Rounding  big.RoundingMode
}

func (a DialogueArithmetic) precision() uint {
	switch a.Precision {
	case 24, 53, 64:
		return a.Precision
	default:
		return 53
	}
}

func justifiedStarts(x0, w int, widths []int) []int {
	return justifiedStartsWith(DialogueArithmetic{}, x0, w, widths)
}

func justifiedStartsWith(a DialogueArithmetic, x0, w int, widths []int) []int {
	if len(widths) == 0 {
		return nil
	}
	if len(widths) == 1 {
		return []int{x0}
	}
	mode := a.Rounding
	if mode > big.ToPositiveInf {
		mode = big.ToNearestEven
	}
	num := func() *big.Float { return new(big.Float).SetPrec(a.precision()).SetMode(mode) }
	spill := func(v *big.Float) *big.Float {
		stored := new(big.Float).SetPrec(53).SetMode(mode).Set(v)
		return num().Set(stored)
	}
	sum := 0
	for _, m := range widths {
		sum += m
	}
	gap := spill(num().Quo(num().SetInt64(int64(w-sum)), num().SetInt64(int64(len(widths)-1))))
	at := spill(num().SetInt64(int64(x0)))
	starts := make([]int, len(widths))
	for i, m := range widths {
		whole, _ := at.Int64()
		starts[i] = int(whole)
		at = spill(num().Add(num().Add(num().SetInt64(int64(m)), at), gap))
	}
	return starts
}

// drawDialogueLine draws one line whose control is w wide from x0, where x0
// counts from the design space's origin less origin. A justified line of two or
// more words is spread over w by justifiedStarts; any other line is drawn whole
// from x0.
func drawDialogueLine(dst *image.RGBA, f *text.Font, ln dialogueLine, origin, x0, y, w int, arithmetic DialogueArithmetic, ink color.RGBA) {
	visible := ln.visible()
	words := strings.FieldsFunc(visible, func(r rune) bool { return r == ' ' || r == '\t' })
	if !ln.justify || len(words) < 2 {
		drawDialogueRun(dst, f, visible, x0, y, ink)
		return
	}
	widths := make([]int, len(words))
	for i, word := range words {
		widths[i] = f.Advance(word)
	}
	for i, start := range justifiedStartsWith(arithmetic, origin+x0, w, widths) {
		drawDialogueRun(dst, f, words[i], start-origin, y, ink)
	}
}

// renderDialogueNotice composes the dialogue style: the frame and its shadow
// band, the portrait pane, the button and the text, in that order, into a
// picture the size of the box.
func renderDialogueNotice(l NoticeLayout, f *text.Font, s string, face *image.RGBA) *image.RGBA {
	size := l.Box.Size()
	img := image.NewRGBA(image.Rectangle{Max: size})
	body := image.Rect(0, 0, size.X-noticeShadow, size.Y-noticeShadow)
	if l.Frame.valid() {
		localPolicy := l.DialogueBackdrop
		localPolicy.FrameClipped = false
		l.Frame.paintDialogueBody(img, body, localPolicy)
	} else {
		fillPanelFrame(img, body.Size(), l.Fill, l.Border)
	}
	if p := l.Portrait; p.Dx() > 0 && p.Dy() > 0 {
		drawDialoguePortrait(img, l, face)
	}
	if b := l.Button; b.Dx() > 0 && b.Dy() > 0 {
		drawPushButton(img, f, dialogueButton(b, l.ButtonLabel, l))
	}
	pitch := f.Height() + l.Pitch
	if pitch <= 0 {
		return img
	}
	lines := dialogueWrap(f, s, l.Text.Dx())
	total, visible := len(lines), max(l.Text.Dy()/pitch, 0)
	first := min(max(l.FirstLine, 0), max(total-visible, 0))
	lines = lines[first:]
	if visible < len(lines) {
		lines = lines[:visible]
	}
	if !l.Scrollbar.Empty() {
		drawHelpScrollbar(img, l, first, total, visible, l.ScrollbarTopHot, l.ScrollbarBottomHot)
	}
	ink := dialogueInk
	if l.Ink.A != 0 {
		ink = l.Ink
	}
	for i, ln := range lines {
		x0, w := l.Text.Min.X, l.Text.Dx()
		if ln.indent {
			x0, w = x0+dialogueIndent, w-dialogueIndent
		}
		drawDialogueLine(img, f, ln, l.Box.Min.X, x0, l.Text.Min.Y+i*pitch, w, l.DialogueArithmetic, ink)
	}
	return img
}
