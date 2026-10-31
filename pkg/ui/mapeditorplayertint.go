package ui

import (
	"image"
	"image/color"
	"image/draw"
)

// editorPlayerTints is the one player-tint colour source: the colour of player
// number n is entry n-1, so two players of a map never share one, whatever
// colour field their map records carry.
var editorPlayerTints = [16]color.RGBA{
	{0xe6, 0x19, 0x4b, 255}, // 1 red
	{0x3c, 0xb4, 0x4b, 255}, // 2 green
	{0x43, 0x63, 0xd8, 255}, // 3 blue
	{0xff, 0xe1, 0x19, 255}, // 4 yellow
	{0xf5, 0x82, 0x31, 255}, // 5 orange
	{0x91, 0x1e, 0xb4, 255}, // 6 purple
	{0x42, 0xd4, 0xf4, 255}, // 7 cyan
	{0xf0, 0x32, 0xe6, 255}, // 8 magenta
	{0xbf, 0xef, 0x45, 255}, // 9 lime
	{0xfa, 0xbe, 0xd4, 255}, // 10 pink
	{0x46, 0x99, 0x90, 255}, // 11 teal
	{0xdc, 0xbe, 0xff, 255}, // 12 lavender
	{0x9a, 0x63, 0x24, 255}, // 13 brown
	{0x80, 0x00, 0x00, 255}, // 14 maroon
	{0x00, 0x00, 0x75, 255}, // 15 navy
	{0xa9, 0xa9, 0xa9, 255}, // 16 grey
}

// playerCount is how many players the open map has: its player records.
func (e *MapEditor) playerCount() int {
	n := 0
	if e.doc != nil {
		for _, r := range e.doc.Records {
			if r.Kind == "Player" {
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
// keeps the tint with the object. Off, it is nil.
func (e *MapEditor) playerTint() func(InspectionSubject) (color.RGBA, bool) {
	if !e.view.playerColours || e.doc == nil {
		return nil
	}
	players := e.playerCount()
	owners := map[InspectionSubject]int{}
	for _, r := range e.doc.Records {
		switch r.Kind {
		case "Unit":
			owners[InspectionSubject{Kind: InspectionUnit, ID: uint32(r.Index)}] = r.Player
		case "Structure":
			owners[InspectionSubject{Kind: InspectionStructure, ID: uint32(r.Index)}] = r.Player
		}
	}
	return func(s InspectionSubject) (color.RGBA, bool) {
		owner := owners[s]
		if owner < 1 || owner > players {
			return color.RGBA{}, false
		}
		return editorPlayerTints[owner-1], true
	}
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
// its colour.
type legendRow struct {
	rect  image.Rectangle
	label string
	color color.RGBA
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
		if r.Kind != "Player" || r.Index < 0 || r.Index >= len(labels) {
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
	var rows []legendRow
	y := box.Min.Y + pad
	for i := range labels {
		if !have[i] {
			continue
		}
		rect := image.Rect(box.Min.X+pad, y, box.Max.X-pad, y+pitch)
		rows = append(rows, legendRow{rect: rect, label: labels[i], color: editorPlayerTints[i]})
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
		draw.Draw(clip, swatch, image.NewUniform(r.color), image.Point{}, draw.Src)
		if f != nil {
			f.Draw(clip, r.label, swatch.Max.X+4, r.rect.Min.Y+1, editorPaper)
		}
	}
}
