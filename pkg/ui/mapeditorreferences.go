package ui

import (
	"image"
	"image/color"
	"math"

	"againrom/pkg/render/camera"
)

// These DTOs contain presentation data only. The game adapter resolves authored
// identifiers; UI never imports an ALM decoder or compiles a script.
type InspectionLine struct {
	Text, Role string // condition, action, warning, or plain
	Reference  int    // 1-based References index; 0 is not a link
}

type InspectionReference struct {
	Label, Role string
	Targets     []InspectionTarget // empty for unresolved, external or ambiguous
}

type InspectionTarget struct {
	Cell, Size image.Point
}

type inspectionDetailRow struct {
	InspectionLine
	Rect image.Rectangle
}

func inspectionRoleColor(role string) color.RGBA {
	switch role {
	case "condition":
		return editorCondition
	case "action":
		return editorGold
	case "warning":
		return editorWarning
	default:
		return editorPaper
	}
}

// detailLayout is shared by the uploaded rail and Pointer hit tests. A wrapped
// link retains its identity on every visible line, including after a resize.
func (e *MapEditor) detailLayout(h int) []inspectionDetailRow {
	if e.doc == nil || e.selected < 0 || e.selected >= len(e.doc.Records) || e.catalog {
		return nil
	}
	r := e.doc.Records[e.selected]
	top := e.listBottom() + 86
	if r.Preview != nil && h >= 700 && r.Preview.Width > 0 && r.Preview.Height > 0 {
		top += 70
	}
	var lines []InspectionLine
	visible := false
	for _, id := range e.rows() {
		visible = visible || id == e.selected
	}
	if !visible {
		lines = append(lines, InspectionLine{Text: "Hidden by filter: choose All", Role: "warning"})
	}
	// Structured records put their compact authored summary first. Their issues
	// are also shown beside the exact failing reference, not ahead of the logic.
	if r.Warning != "" && len(r.Lines) == 0 {
		lines = append(lines, InspectionLine{Text: "! " + r.Warning, Role: "warning"})
	}
	lines = append(lines, InspectionLine{Text: r.Label})
	for _, s := range r.Details {
		lines = append(lines, InspectionLine{Text: s})
	}
	lines = append(lines, r.Lines...)
	var wrapped []InspectionLine
	for _, l := range lines {
		if l.Reference > 0 {
			l.Text = "> " + l.Text
		}
		for _, s := range editorWrap(e.font, l.Text, 300) {
			row := l
			row.Text = s
			wrapped = append(wrapped, row)
		}
	}
	lineHeight := max(16, e.font.Height()+2)
	limit := max(0, (h-top-45)/lineHeight)
	e.detailOffset = max(0, min(e.detailOffset, max(0, len(wrapped)-limit)))
	var out []inspectionDetailRow
	for i := e.detailOffset; i < min(len(wrapped), e.detailOffset+limit); i++ {
		y := top + (i-e.detailOffset)*lineHeight
		out = append(out, inspectionDetailRow{wrapped[i], image.Rect(10, y, 310, y+lineHeight)})
	}
	return out
}

// FocusReference preserves the selected record, filter, list and detail travel.
// A multi-target group is framed as a group, never reduced to its first member.
func (e *MapEditor) FocusReference(index int) bool {
	if e.doc == nil || e.selected < 0 || e.selected >= len(e.doc.Records) {
		return false
	}
	refs := e.doc.Records[e.selected].References
	if index < 1 || index > len(refs) || len(refs[index-1].Targets) == 0 {
		return false
	}
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	include := func(points [4]image.Point) {
		for _, p := range points {
			minX, minY = math.Min(minX, float64(p.X)), math.Min(minY, float64(p.Y))
			maxX, maxY = math.Max(maxX, float64(p.X)), math.Max(maxY, float64(p.Y))
		}
	}
	for _, t := range refs[index-1].Targets {
		corners, ok := e.referenceWorldCorners(t)
		if !ok {
			continue
		}
		include(corners)
		// A multi-cell footprint's outline need not bound its anchor cell on
		// displaced terrain. Include that cell's four corners too, bounding
		// both the displayed cross and the ground placement's four-corner mean.
		cell := image.Pt(max(0, min(t.Cell.X, e.doc.Width-1)), max(0, min(t.Cell.Y, e.doc.Height-1)))
		anchor, _ := e.referenceWorldCorners(InspectionTarget{Cell: cell, Size: image.Pt(1, 1)})
		include(anchor)
	}
	if math.IsInf(minX, 1) {
		return false
	}
	e.focusedReference = index
	c := e.doc.Viewer.cam
	// Padding is in screen pixels, so the focused nine-pixel cross fits even
	// when a whole-map target requires zoom below the ordinary Fit scale.
	z := math.Min(1, math.Min(float64(max(1, c.ViewW-48))/(maxX-minX), float64(max(1, c.ViewH-48))/(maxY-minY)))
	c.SetMinimumZoom(math.Min(camera.ZoomMin, math.Min(e.fitZoom(), z)))
	c.SetZoom(z)
	c.CenterOn((minX+maxX)/2, (minY+maxY)/2)
	return true
}

// referenceWorldCorners is the clipped outline shared by focus and drawing.
// It projects four corners per target, never every cell in a large raw region.
func (e *MapEditor) referenceWorldCorners(t InspectionTarget) ([4]image.Point, bool) {
	box := image.Rectangle{Min: t.Cell, Max: t.Cell.Add(image.Pt(max(1, t.Size.X), max(1, t.Size.Y)))}.Intersect(image.Rect(0, 0, e.doc.Width, e.doc.Height))
	if box.Empty() {
		return [4]image.Point{}, false
	}
	points := [4]image.Point{box.Min, {X: box.Max.X, Y: box.Min.Y}, box.Max, {X: box.Min.X, Y: box.Max.Y}}
	for j, p := range points {
		x, y := p.X*32, p.Y*32
		if e.doc.Viewer.Mode() == ModeDisplaced {
			x, y = e.doc.Viewer.proj.WorldCorner(p.X, p.Y)
		}
		points[j] = image.Pt(x, y)
	}
	return points, true
}

func (e *MapEditor) referenceMarkerLines() []editorLine {
	if e.doc == nil || e.selected < 0 || e.selected >= len(e.doc.Records) {
		return nil
	}
	v := e.doc.Viewer
	var out []editorLine
	for i, ref := range e.doc.Records[e.selected].References {
		c := inspectionRoleColor(ref.Role)
		for _, target := range ref.Targets {
			points, ok := e.referenceWorldCorners(target)
			if !ok {
				continue
			}
			var xs, ys [4]float32
			for j, p := range points {
				x, y := v.cam.WorldToScreen(float64(p.X), float64(p.Y))
				xs[j], ys[j] = float32(x), float32(y)
			}
			for j := range points {
				k := (j + 1) % 4
				out = append(out, editorLine{xs[j], ys[j], xs[k], ys[k], c})
			}
			wx, wy := e.cellCenter(target.Cell)
			x, y := v.cam.WorldToScreen(wx, wy)
			size := float32(4)
			if i+1 == e.focusedReference {
				size = 9
			}
			out = append(out, editorLine{float32(x) - size, float32(y), float32(x) + size, float32(y), c}, editorLine{float32(x), float32(y) - size, float32(x), float32(y) + size, c})
		}
	}
	return out
}
