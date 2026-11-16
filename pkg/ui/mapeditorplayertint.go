package ui

import (
	"image"
	"image/color"
	"image/draw"
)

// playerCount is how many players the open map has: its player records.
func (e *MapEditor) playerCount() int {
	n := 0
	if e.doc != nil {
		for _, r := range e.doc.Records {
			if r.Kind == KindPlayer {
				n = max(n, r.Index+1)
			}
		}
	}
	return min(n, len(editorPlayerTints))
}

// playerTint is the sprite tint the map view applies while the player colours
// are on: the colour of the player that owns a unit or structure of the open
// map, and none for an object of owner 0 or of an owner the map has no player
// for. It reads the records as they are now, so a move, an undo or a redo
// keeps the tint with the object. Off, it is nil. While a player record is
// selected only that player's objects answer, and with the brighter focus tint;
// every other object answers none.
func (e *MapEditor) playerTint() func(InspectionSubject) spriteTint {
	if !e.view.playerColours || e.doc == nil {
		return nil
	}
	players := e.playerCount()
	owners := map[InspectionSubject]int{}
	for _, r := range e.doc.Records {
		switch r.Kind {
		case KindUnit:
			owners[InspectionSubject{Kind: InspectionUnit, ID: uint32(r.Index)}] = r.Player
		case KindStructure:
			owners[InspectionSubject{Kind: InspectionStructure, ID: uint32(r.Index)}] = r.Player
		}
	}
	focus := e.focusedPlayer()
	return func(s InspectionSubject) spriteTint {
		owner := owners[s]
		if owner < 1 || owner > players || (focus != 0 && owner != focus) {
			return spriteTint{}
		}
		return spriteTint{Color: editorPlayerTints[owner-1], On: true, Focus: focus != 0}
	}
}

// focusedPlayer is the number of the selected player record, or 0 when the
// selection is not a player.
func (e *MapEditor) focusedPlayer() int {
	if kind, index := e.selectedRecord(); kind == KindPlayer && index >= 0 {
		return index + 1
	}
	return 0
}

// applyPlayerTint hands the map view's sprite painter the tint of the moment,
// or none while the player colours are off.
func (e *MapEditor) applyPlayerTint() {
	if e.doc != nil {
		e.doc.Viewer.SetSubjectTint(e.playerTint())
	}
}

// keyPlayerTint turns the player colours on or off.
func (e *MapEditor) keyPlayerTint(string) error {
	e.view.playerColours = !e.view.playerColours
	return nil
}

// legendSwatch is the side of a player's colour square in the legend.
const legendSwatch = 10

// legendRow is one player of the legend: its number, its record's label and
// its colour. focus marks the selected player; muted marks the other rows
// while one player is selected, since their colour is not shown on the map.
type legendRow struct {
	rect  image.Rectangle
	label string
	color color.RGBA
	focus bool
	muted bool
}

// playerLegend is the legend box and its rows, in window coordinates, while
// the player colours are on. The box stands in the map view's bottom left
// corner, just above the status strip: the bar is above the map, the panels
// and the report open at its top corners, and nothing else of the editor is
// anchored there. A popup the map maker drags over it is drawn after it.
func (e *MapEditor) playerLegend() (image.Rectangle, []legendRow) {
	if !e.view.playerColours || e.doc == nil {
		return image.Rectangle{}, nil
	}
	f := e.bodyFont()
	pitch := fontHeight(f, 15) + 2
	labels := make([]string, len(editorPlayerTints))
	have := make([]bool, len(editorPlayerTints))
	width := 0
	count := 0
	for _, r := range e.doc.Records {
		if r.Kind != KindPlayer || r.Index < 0 || r.Index >= len(labels) {
			continue
		}
		labels[r.Index], have[r.Index] = r.Label, true
		count++
		if f != nil {
			width = max(width, f.Advance(r.Label))
		}
	}
	if count == 0 {
		return image.Rectangle{}, nil
	}
	const pad = 4
	area := e.mapArea()
	box := image.Rect(area.Min.X+6, 0, area.Min.X+6+pad+legendSwatch+pad+width+pad, 0)
	box.Max.Y = area.Max.Y - 6
	box.Min.Y = box.Max.Y - (2*pad + count*pitch)
	box = box.Intersect(area)
	focus := e.focusedPlayer()
	var rows []legendRow
	y := box.Min.Y + pad
	for i := range labels {
		if !have[i] {
			continue
		}
		rect := image.Rect(box.Min.X+pad, y, box.Max.X-pad, y+pitch)
		rows = append(rows, legendRow{rect: rect, label: labels[i], color: editorPlayerTints[i],
			focus: focus == i+1, muted: focus != 0 && focus != i+1})
		y += pitch
	}
	return box, rows
}

// drawPlayerLegend paints the legend on the chrome picture with the editor's
// box and text painters.
func (e *MapEditor) drawPlayerLegend(pic *image.RGBA) {
	box, rows := e.playerLegend()
	if box.Empty() {
		return
	}
	f := e.bodyFont()
	draw.Draw(pic, box, image.NewUniform(editorInk), image.Point{}, draw.Src)
	drawFrame(pic, panelFrame(box, editorInk, editorGold))
	clip := pic.SubImage(box).(*image.RGBA)
	for _, r := range rows {
		swatch := image.Rect(r.rect.Min.X, r.rect.Min.Y+(r.rect.Dy()-legendSwatch)/2, r.rect.Min.X+legendSwatch, r.rect.Min.Y+(r.rect.Dy()-legendSwatch)/2+legendSwatch)
		ink := editorPaper
		switch {
		case r.muted:
			ink = legendMutedInk
			draw.Draw(clip, swatch, image.NewUniform(r.color), image.Point{}, draw.Src)
			draw.Draw(clip, swatch.Inset(1), image.NewUniform(editorInk), image.Point{}, draw.Src)
		default:
			draw.Draw(clip, swatch, image.NewUniform(r.color), image.Point{}, draw.Src)
		}
		if r.focus {
			outline(clip, r.rect.Intersect(box), editorGold)
		}
		if f != nil {
			f.Draw(clip, r.label, swatch.Max.X+4, r.rect.Min.Y+1, ink)
		}
	}
}
