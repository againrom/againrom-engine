package ui

import (
	"image"
	"slices"
	"strings"

	"againrom/pkg/render/text"
	"againrom/pkg/render/textfit"
)

var tabPageMarks = [2]string{"<", ">"}
var tabPageWords = [2]string{"editor.tabs.previous", "editor.tabs.next"}

type tabLayout struct {
	closeTip             string
	pageTips             [2]string
	rects, closes        []image.Rectangle
	captions, full       []string
	first, active, count int
	page                 [2]image.Rectangle
}

func (t tabStrip) rects(origin image.Point, width, pitch int) []image.Rectangle {
	return t.layout(image.Rectangle{Min: origin, Max: origin.Add(image.Pt(width, pitch))}, textfit.Spec{}).rects
}

func (t tabStrip) layout(bounds image.Rectangle, fit textfit.Spec) tabLayout {
	l := tabLayout{count: len(t.Captions), active: t.Active, captions: t.Captions}
	if l.count == 0 || bounds.Empty() {
		return l
	}
	run := bounds
	per := t.PerLine
	if per <= 0 {
		per = l.count
	}
	widths := make([]int, l.count)
	if t.Closable {
		side := run.Dy()
		total := 0
		for i, caption := range t.Captions {
			widths[i] = max(2*side, fit.Measure(caption)+2*popupPad+side)
			total += widths[i]
		}
		if total > run.Dx() {
			run.Max.X -= 2*side + popupPad
			for i := range l.page {
				x := run.Max.X + popupPad + i*side
				l.page[i] = image.Rect(x, run.Min.Y, x+side, run.Max.Y)
			}
		}
	} else {
		for i := range widths {
			widths[i] = ((i%per+1)*run.Dx())/per - ((i%per)*run.Dx())/per
		}
	}

	x, y := run.Min.X, run.Min.Y
	for i, w := range widths {
		if t.Closable {
			w = min(w, run.Dx())
			if x > run.Min.X && x+w > run.Max.X {
				if i > t.Active {
					break
				}
				x, l.first, l.rects = run.Min.X, i, l.rects[:0]
			}
		} else if i > 0 && i%per == 0 {
			x, y = run.Min.X, y+run.Dy()
		}
		l.rects = append(l.rects, image.Rect(x, y, x+w, y+run.Dy()))
		x += w
	}
	if !t.Closable {
		return l
	}
	l.active -= l.first
	l.captions = slices.Clone(t.Captions[l.first : l.first+len(l.rects)])
	l.full = make([]string, len(l.rects))
	for i, r := range l.rects {
		side := r.Dy()
		close := image.Rect(r.Max.X-side, r.Min.Y, r.Max.X, r.Max.Y)
		l.closes = append(l.closes, close)
		l.rects[i].Max.X = close.Min.X
		caption := l.captions[i]
		room := l.rects[i].Dx() - 2*popupPad
		if name, dirty := strings.CutSuffix(caption, " *"); dirty && fit.Measure(caption) > room && fit.Measure(" *") <= room {
			l.captions[i] = fit.Fit(name, room-fit.Measure(" *")) + " *"
		} else {
			l.captions[i] = fit.Fit(caption, room)
		}
		if l.captions[i] != caption {
			l.full[i] = caption
		}
	}
	return l
}

func editorMeasure(f *text.Font) func(string) int {
	if f == nil {
		return func(s string) int { return 8 * len(s) }
	}
	return f.Advance
}

func (l tabLayout) tip(p image.Point) (image.Rectangle, string) {
	switch part, i := l.hit(p); part {
	case popupClose:
		return l.closes[i], l.closeTip
	case popupTabPage:
		return l.page[i], l.pageTips[i]
	case popupTab:
		return l.rects[i], l.full[i]
	}
	return image.Rectangle{}, ""
}

func (l tabLayout) pageTarget(direction int) int {
	if direction == 0 {
		return l.first - 1
	}
	return l.first + len(l.rects)
}

func (l tabLayout) hit(p image.Point) (popupPart, int) {
	for i, r := range l.closes {
		if p.In(r) {
			return popupClose, i
		}
	}
	for i, r := range l.page {
		if p.In(r) {
			return popupTabPage, i
		}
	}
	for i, r := range l.rects {
		if p.In(r) {
			return popupTab, i
		}
	}
	return popupMiss, 0
}

func (t tabStrip) draw(dst *image.RGBA, f *text.Font, l tabLayout, held *buttonPress, part popupPart, at int) {
	for i, tab := range l.rects {
		chosen := t.Switch && l.active == i
		drawPushButton(dst, f, pushButton{Press: held, Rect: tab, Label: l.captions[i], Literal: true, LabelRaise: labelRaise(f), Focus: l.active == i && !t.Switch, Pressed: chosen, Inside: chosen, Hover: part == popupTab && at == i})
	}
	for i, close := range l.closes {
		drawPushButton(dst, f, pushButton{Press: held, Rect: close, Label: "x", Literal: true, LabelRaise: labelRaise(f), Hover: part == popupClose && at == i})
	}
	for i, r := range l.page {
		target := l.pageTarget(i)
		drawPushButton(dst, f, pushButton{Press: held, Rect: r, Label: tabPageMarks[i], Literal: true, LabelRaise: labelRaise(f), Disabled: target < 0 || target >= l.count, Hover: part == popupTabPage && at == i})
	}
}
