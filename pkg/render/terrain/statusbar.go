package terrain

import (
	"image"
	"image/color"
)

// StatusBarKind names one of a unit's two overhead status bars.
type StatusBarKind int

const (
	// HealthBar is drawn over a unit's hit points.
	HealthBar StatusBarKind = iota
	// ManaBar is drawn over a mage's mana, directly under the health bar.
	ManaBar
)

// A status bar is the original's own picture, measured from the owner's
// screenshots at native scale: a 4 by 4 end cap, the interior columns, then
// the same cap again, not mirrored, 4 rows tall. A one-cell unit's bar is 32
// wide with 24 interior columns; a wider unit's is as wide as its class's
// selection box. The original draws the bars in a pass of their own over the
// unit's hit points and mana (TERR-SPR-048); no claim gives their geometry or
// colours, and the choices the screenshots do not decide are DIV-1459,
// DIV-1460 and DIV-1481.
const (
	StatusBarWidth    = 32
	StatusBarHeight   = 4
	statusBarCap      = 4
	StatusBarInterior = StatusBarWidth - 2*statusBarCap
)

// statusBarAbove is how far each kind's top edge stands above the top edge of
// the unit's footprint: health from 16 to 13 rows above it and mana directly
// under health, 12 to 9 rows above. The owner's screenshot shows a standing
// mage's two bars at exactly these rows over its cell (DIV-1460).
var statusBarAbove = [...]int{HealthBar: 16, ManaBar: 12}

// statusBarOverBox is how far a wide unit's health bar stands above the top
// edge of its class's selection box. It is the one-cell rule's offset on the
// owner's mage, whose box's top (units.reg CenterY 78, SelectionY1 48) stands
// 14 rows above its cell's top edge.
const statusBarOverBox = 2

func statusBarRGB(v uint32) color.RGBA {
	return color.RGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}
}

// statusBarCapPixels is the end cap, rows top to bottom; a zero colour is a
// transparent pixel.
var statusBarCapPixels = [StatusBarHeight][statusBarCap]color.RGBA{
	{{}, statusBarRGB(0x6b4129), statusBarRGB(0x4a2c18), {}},
	{statusBarRGB(0x9c6542), statusBarRGB(0xce926b), statusBarRGB(0x6b4129), statusBarRGB(0x422810)},
	{statusBarRGB(0x6b4129), statusBarRGB(0x9c6542), statusBarRGB(0x4a2810), statusBarRGB(0x392410)},
	{statusBarRGB(0x000400), statusBarRGB(0x392410), statusBarRGB(0x211408), {}},
}

// statusBarRows are each kind's interior rows, top to bottom.
var statusBarRows = [...][StatusBarHeight]color.RGBA{
	HealthBar: {statusBarRGB(0x008200), statusBarRGB(0x00ff00), statusBarRGB(0x00c300), statusBarRGB(0x008200)},
	ManaBar:   {statusBarRGB(0x000084), statusBarRGB(0x0000ff), statusBarRGB(0x0000c6), statusBarRGB(0x000084)},
}

// StatusBarFaded is c at half opacity, premultiplied as color.RGBA is: alpha
// 128, the nearest an 8-bit alpha comes to one half, and each channel scaled
// by 128/255. Blended source-over it keeps 127/255 of what lies under it.
func StatusBarFaded(c color.RGBA) color.RGBA {
	half := func(v uint8) uint8 { return uint8((uint32(v)*128 + 127) / 255) }
	return color.RGBA{R: half(c.R), G: half(c.G), B: half(c.B), A: 128}
}

// StatusBarFill is how many interior columns one pool fills in a bar width
// pixels wide, whose interior is the width less both caps:
// interior*value/maximum, truncated, clamped into 0..interior. A value below
// zero draws an empty interior and one above the maximum a full one. ok is
// false when the maximum is not positive: a unit with no such pool has no bar
// at all.
func StatusBarFill(width, value, maximum int) (fill int, ok bool) {
	if maximum <= 0 {
		return 0, false
	}
	interior := max(width-2*statusBarCap, 0)
	return min(max(interior*value/maximum, 0), interior), true
}

// StatusBarRect is where one kind of bar stands, in native map pixels, for a
// unit of class c whose footprint's top-left cell is (col, row) and whose
// footprint is side cells square. ok is false for an unknown kind or a cell
// outside the map.
//
// A class wider than one cell (TileSize above one) whose selection box holds
// both caps spans that box's width. The box is placed as UnitPlace places the
// body, its class centre on the anchor cell's centre; the health bar stands
// statusBarOverBox rows above the box's top edge, mana directly under it. The
// capture of the original's dragon shows this width and height (DIV-1460).
// Any other unit, or one drawn without a class, takes the one-cell rule: 32
// by 4, centred on the whole footprint, at its kind's height above the
// footprint's top edge. A side below one is one.
//
// A bar over a unit near row zero has a negative map Y; the viewer clips it at
// the viewport.
func StatusBarRect(kind StatusBarKind, col, row, cols, rows, side int, c *UnitClass) (image.Rectangle, bool) {
	if kind < HealthBar || kind > ManaBar || markerCellRejected(col, row, cols, rows, CellSize) {
		return image.Rectangle{}, false
	}
	if c != nil && c.TileSize > 1 && c.Selection.Dx() >= 2*statusBarCap {
		x0 := col*CellSize + CellSize/2 + c.Selection.Min.X - c.CenterX
		y0 := row*CellSize + CellSize/2 + c.Selection.Min.Y - c.CenterY - statusBarOverBox +
			statusBarAbove[HealthBar] - statusBarAbove[kind]
		return image.Rect(x0, y0, x0+c.Selection.Dx(), y0+StatusBarHeight), true
	}
	side = max(side, 1)
	x0 := col*CellSize + side*CellSize/2 - StatusBarWidth/2
	y0 := row*CellSize - statusBarAbove[kind]
	return image.Rect(x0, y0, x0+StatusBarWidth, y0+StatusBarHeight), true
}

// StatusBarRun is one piece of a bar in one colour: a native rectangle one
// row tall.
type StatusBarRun struct {
	Rect  image.Rectangle
	Color color.RGBA
}

// AppendStatusBarRuns appends the runs that paint one bar standing at bar (a
// StatusBarRect) with fill interior columns filled: every opaque cap pixel at
// both ends of the bar's own width, then one run per interior row covering the
// filled columns from the left. The unfilled columns get no run, so what lies
// under the bar shows there. A faded bar keeps its caps opaque and draws its
// interior with StatusBarFaded. A bar too narrow for both caps paints nothing.
func AppendStatusBarRuns(dst []StatusBarRun, kind StatusBarKind, bar image.Rectangle, fill int, faded bool) []StatusBarRun {
	if kind < HealthBar || kind > ManaBar || bar.Dx() < 2*statusBarCap {
		return dst
	}
	fill = min(max(fill, 0), bar.Dx()-2*statusBarCap)
	for y := 0; y < StatusBarHeight; y++ {
		for x, c := range statusBarCapPixels[y] {
			if c.A == 0 {
				continue
			}
			for _, left := range [2]int{0, bar.Dx() - statusBarCap} {
				p := bar.Min.Add(image.Pt(left+x, y))
				dst = append(dst, StatusBarRun{Rect: image.Rect(p.X, p.Y, p.X+1, p.Y+1), Color: c})
			}
		}
		if fill == 0 {
			continue
		}
		c := statusBarRows[kind][y]
		if faded {
			c = StatusBarFaded(c)
		}
		p := bar.Min.Add(image.Pt(statusBarCap, y))
		dst = append(dst, StatusBarRun{Rect: image.Rect(p.X, p.Y, p.X+fill, p.Y+1), Color: c})
	}
	return dst
}
