package ui

import (
	"image"
	"image/color"
	"image/draw"
	"strings"

	"againrom/pkg/render/text"
)

const legendSwatch = 10

type popupSwatch struct {
	color        color.RGBA
	focus, muted bool
}

func swatchLines(f *text.Font, label string, width, limit int) ([]string, bool) {
	if f == nil {
		return []string{label}, false
	}
	room := max(1, width-legendSwatch-popupPad-2*popupTextInset)
	lines := editorTextLines(f, label, room)
	cut := len(lines) > limit
	if cut {
		lines[limit-1] = editorClip(f, strings.Join(lines[limit-1:], " "), room)
		lines = lines[:limit]
	}
	return lines, cut
}

func drawPopupSwatch(dst *image.RGBA, rr popupRowRect, swatch popupSwatch, ink popupInk) {
	if swatch.muted {
		drawFrame(dst, frameSpec{Kind: frameOutline, Rect: rr.Swatch, Border: swatch.color})
	} else {
		draw.Draw(dst, rr.Swatch, image.NewUniform(swatch.color), image.Point{}, draw.Src)
	}
	if swatch.focus {
		drawFrame(dst, frameSpec{Kind: frameOutline, Rect: rr.Rect, Border: ink.Edge})
	}
}

func playerSwatchRow(index, focus int, label string) popupRow {
	swatch := &popupSwatch{color: editorPlayerTints[index], focus: focus == index+1, muted: focus != 0 && focus != index+1}
	ink := editorPaper
	if swatch.muted {
		ink = legendMutedInk
	}
	return popupRow{Label: label, Ink: ink, Swatch: swatch}
}

func playerLegendSpec(area image.Rectangle, rows []popupRow, measure func(string) int, page int) popupSpec {
	area = area.Inset(popupPad)
	width := 0
	for _, row := range rows {
		width = max(width, measure(row.Label))
	}
	width += 3*popupPad + legendSwatch + 2*popupTextInset
	return popupSpec{Bare: true, Short: true, Bounds: area, At: image.Pt(area.Min.X, area.Max.Y),
		Width: width, Rows: rows, Page: page}
}
