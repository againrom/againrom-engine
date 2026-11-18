package ui

import (
	"image"

	"againrom/pkg/render/text"
)

type popupBar struct {
	Groups        []popupBarGroup
	Strip         bool
	Grip          *editorIcon
	GripWidth     int
	Sizes         []toolSize
	Height        int
	Inset, Pad    int
	Gap, GroupGap int
	MaxRows       int
	CompactText   bool
}

type popupBarGroup struct {
	Entries []popupBarEntry
	Right   bool
}

type popupBarEntry struct {
	Index          int
	Caption, Tip   string
	Menu           bool
	On, Disabled   bool
	Edit           *popupRow
	PreferredWidth int
	Icon           *editorIcon
	Cell           image.Point
}

type popupBarCell struct {
	popupBarEntry
	Rect image.Rectangle
}

type popupBarLayout struct {
	Cells []popupBarCell
	Grip  image.Rectangle
	Scale int
}

func (l popupBarLayout) cellAt(p image.Point) (popupBarCell, bool) {
	for _, cell := range l.Cells {
		if p.In(cell.Rect) {
			return cell, true
		}
	}
	return popupBarCell{}, false
}

type measuredBarGroup struct {
	popupBarGroup
	Widths []int
	Width  int
}

func measureBarGroups(b popupBar, side, captionMin int, title, body *text.Font) []measuredBarGroup {
	groups := make([]measuredBarGroup, 0, len(b.Groups))
	count := 0
	for _, g := range b.Groups {
		count += len(g.Entries)
	}
	widths := make([]int, count)
	for _, group := range b.Groups {
		if len(group.Entries) == 0 {
			continue
		}
		g := measuredBarGroup{popupBarGroup: group, Widths: widths[:len(group.Entries)]}
		widths = widths[len(group.Entries):]
		for i, entry := range group.Entries {
			w := side
			switch {
			case entry.Edit != nil:
				w = max(popupMinField, entry.PreferredWidth)
				if body != nil {
					w = max(w, body.Advance(entry.Edit.Hint)+2*editTextX)
				}
			case entry.Icon == nil:
				w = popupCaptionWidth(title, entry.Caption, captionMin)
				if entry.Menu {
					w += b.Height
				}
			}
			g.Widths[i] = w
			g.Width += w
		}
		g.Width += (len(g.Widths) - 1) * b.Gap
		groups = append(groups, g)
	}
	return groups
}

func barRunWidth(groups []measuredBarGroup, gap int) int {
	width := max(0, len(groups)-1) * gap
	for _, g := range groups {
		width += g.Width
	}
	return width
}

func layoutBarStrip(b popupBar, room image.Rectangle, title, body *text.Font) popupLayout {
	side, scale := b.Height, 1
	groups := measureBarGroups(b, side, popupButtonW, title, body)
	width := room.Dx() - 2*b.Inset
	grip := 0
	if b.Grip != nil {
		grip = b.GripWidth + b.Gap
	}
	if b.CompactText && barRunWidth(groups, b.GroupGap) > width {
		const captionPadding = 16
		groups = measureBarGroups(b, side, captionPadding, title, body)
	}
	for _, z := range b.Sizes {
		side, scale = z.side, z.scale
		groups = measureBarGroups(b, side, popupButtonW, title, body)
		if grip+barRunWidth(groups, b.GroupGap) <= width && side+2*b.Pad <= room.Dy() {
			break
		}
	}

	l := popupLayout{Rect: image.Rectangle{Min: room.Min, Max: image.Pt(room.Max.X, room.Min.Y)}}
	l.Bar.Scale = scale
	pitch := side + 2*b.Pad
	rows := 0
	for len(groups) > 0 && (b.MaxRows == 0 || rows < b.MaxRows) {
		top := l.Rect.Max.Y
		if top+pitch > room.Max.Y {
			break
		}
		left, right := room.Min.X+b.Inset, room.Max.X-b.Inset
		y := top + b.Pad
		if rows == 0 && grip > 0 && left+b.GripWidth <= right {
			l.Bar.Grip = image.Rect(left, y, left+b.GripWidth, y+side)
			left += grip
		}
		n, used := 0, 0
		for _, g := range groups {
			if used+g.Width > right-left {
				break
			}
			n++
			used += g.Width + b.GroupGap
		}
		if n == 0 {
			if len(groups[0].Entries) == 1 && groups[0].Entries[0].Edit != nil && right-left >= popupMinField {
				groups[0].Widths[0], groups[0].Width = right-left, right-left
				n = 1
			} else if rows == 0 && !l.Bar.Grip.Empty() && groups[0].Width <= width {
				l.Rect.Max.Y += pitch
				rows++
				continue
			} else {
				groups = groups[1:]
				continue
			}
		}
		line := groups[:n]
		for i, g := range line {
			if g.Right && (i == 0 || !line[i-1].Right) {
				left = max(left, right-barRunWidth(line[i:], b.GroupGap))
			}
			for j, entry := range g.Entries {
				w := g.Widths[j]
				l.Bar.Cells = append(l.Bar.Cells, popupBarCell{entry, image.Rect(left, y, left+w, y+side)})
				left += w + b.Gap
			}
			left += b.GroupGap - b.Gap
		}
		l.Rect.Max.Y += pitch
		rows++
		groups = groups[n:]
	}
	return l
}

func pictureBarSize(b popupBar, room image.Rectangle, title *text.Font) (image.Point, toolSize) {
	var cells image.Point
	for _, g := range b.Groups {
		for _, entry := range g.Entries {
			cells.X = max(cells.X, entry.Cell.X+1)
			cells.Y = max(cells.Y, entry.Cell.Y+1)
		}
	}
	var size image.Point
	var z toolSize
	for _, z = range b.Sizes {
		grid := cells.Mul(z.side + b.Gap)
		size = image.Pt(max(popupTitleHeight(title), grid.X-b.Gap+2*b.Inset), popupTitleHeight(title)+grid.Y-b.Gap+2*b.Pad)
		if room.Empty() || (size.X <= room.Dx() && size.Y <= room.Dy()) {
			break
		}
	}
	return size, z
}

func placePictureBar(b popupBar, room image.Rectangle, z toolSize) popupBarLayout {
	l := popupBarLayout{Scale: z.scale}
	start := room.Min.Add(image.Pt(b.Inset, b.Pad))
	for _, g := range b.Groups {
		for _, entry := range g.Entries {
			at := start.Add(entry.Cell.Mul(z.side + b.Gap))
			r := image.Rectangle{Min: at, Max: at.Add(image.Pt(z.side, z.side))}
			if r.In(room) {
				l.Cells = append(l.Cells, popupBarCell{entry, r})
			}
		}
	}
	return l
}

func drawPopupBar(dst *image.RGBA, title, body *text.Font, b popupBar, l popupLayout, ink popupInk, pointer image.Point, pointerOK bool, held *buttonPress) {
	if !l.Bar.Grip.Empty() {
		drawIcon(dst, l.Bar.Grip, b.Grip, false, l.Bar.Scale)
	}
	for _, cell := range l.Bar.Cells {
		r := cell.Rect
		hover := pointerOK && pointer.In(r)
		switch {
		case cell.Icon != nil:
			drawIconButton(dst, iconButton{Rect: r, Icon: cell.Icon, On: cell.On, Disabled: cell.Disabled, Hover: hover, Scale: l.Bar.Scale, Press: held})
		case cell.Edit != nil:
			drawEditRow(dst, body, *cell.Edit, r, ink, ink.Value)
		case cell.Menu:
			drawDropdown(dst, title, nil, dropdown{Rect: r, Text: cell.Caption, Button: true, Disabled: cell.Disabled, Press: held}, nil, pointer, pointerOK)
		default:
			drawPushButton(dst, title, pushButton{Rect: r, Label: cell.Caption, Literal: true, LabelRaise: labelRaise(title),
				Disabled: cell.Disabled, Pressed: cell.On, Inside: cell.On, Hover: hover, Press: held})
		}
	}
}

func barTipAt(bounds image.Rectangle, body *text.Font, l popupLayout, p image.Point) (popupTip, bool) {
	cell, ok := l.Bar.cellAt(p)
	if !ok {
		return popupTip{}, false
	}
	tip := cell.Tip
	if cell.Edit != nil && cell.Edit.Value != "" {
		if shown, _ := editFieldText(body, *cell.Edit, cell.Rect); shown != cell.Edit.Value {
			tip = cell.Edit.Value
		}
	}
	if tip == "" {
		return popupTip{}, false
	}
	return valueTip(bounds, body, cell.Rect, tip), true
}
