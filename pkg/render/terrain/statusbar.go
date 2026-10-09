package terrain

import (
	"image"
	"image/color"
)

// StatusBarKind names one of a unit's two overhead status bars.
type StatusBarKind int

const (
	HealthBar StatusBarKind = iota
	ManaBar
)

const (
	StatusBarWidth    = 32
	StatusBarHeight   = 4
	statusBarCap      = 4
	StatusBarInterior = StatusBarWidth - 2*statusBarCap
)

var statusBarAbove = [...]int{HealthBar: 16, ManaBar: 12}

const statusBarOverBox = 2

func statusBarRGB(v uint32) color.RGBA {
	return color.RGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}
}

// Cap pixels retain the screenshot-based picture; their native sprites are unknown.
var statusBarCapPixels = [StatusBarHeight][statusBarCap]color.RGBA{
	{{}, statusBarRGB(0x6b4129), statusBarRGB(0x4a2c18), {}},
	{statusBarRGB(0x9c6542), statusBarRGB(0xce926b), statusBarRGB(0x6b4129), statusBarRGB(0x422810)},
	{statusBarRGB(0x6b4129), statusBarRGB(0x9c6542), statusBarRGB(0x4a2810), statusBarRGB(0x392410)},
	{statusBarRGB(0x000400), statusBarRGB(0x392410), statusBarRGB(0x211408), {}},
}

var statusBarIntensities = [...]uint8{128, 255, 192, 128}
var statusBarGrey = [...]uint8{64, 128, 96, 64}

func statusBarRow(kind StatusBarKind, value, maximum, row int) color.RGBA {
	c := color.RGBA{A: 255}
	intensity := statusBarIntensities[row]
	switch {
	case kind == ManaBar:
		c.B = intensity
	case value < maximum/4:
		c.R = intensity
	case value < maximum/2:
		c.R, c.G = intensity, intensity
	default:
		c.G = intensity
	}
	return c
}

func statusBarWord(c color.RGBA) uint16 {
	return uint16(c.R>>3)<<11 | uint16(c.G>>2)<<5 | uint16(c.B>>3)
}

func statusBarWordColor(w uint16) color.RGBA {
	return color.RGBA{uint8(uint32(w>>11) * 255 / 31),
		uint8(uint32(w>>5&63) * 255 / 63), uint8(uint32(w&31) * 255 / 31), 255}
}

// StatusBarBlend uses TERR-225's RGB565 half words. DIV-2572
func StatusBarBlend(source, destination color.RGBA) color.RGBA {
	s, d := statusBarWord(source), statusBarWord(destination)
	return statusBarWordColor(((s >> 1) & 0x7bef) + ((d >> 1) & 0x7bef))
}

// StatusBarFill follows TERR-227 without saturating an overfull pool.
// Nonpositive maxima are omitted; arithmetic uses the engine's integer domain.
func StatusBarFill(kind StatusBarKind, width, value, maximum int) (fill int, ok bool) {
	if kind < HealthBar || kind > ManaBar || maximum <= 0 {
		return 0, false
	}
	interior := max(width-2*statusBarCap, 0)
	fill = interior * value / maximum
	if fill == 0 && (kind == ManaBar || value != 0) && interior > 0 {
		fill = 1
	}
	return fill, true
}

// StatusBarRect uses the class selection width when it holds both caps.
// Vertical placement retains the screenshot-based rule in DIV-1460.
func StatusBarRect(kind StatusBarKind, col, row, cols, rows, side int, c *UnitClass) (image.Rectangle, bool) {
	if kind < HealthBar || kind > ManaBar || markerCellRejected(col, row, cols, rows, CellSize) {
		return image.Rectangle{}, false
	}
	if c != nil && c.Selection.Dx() >= 2*statusBarCap {
		x0 := col*CellSize + CellSize/2 + c.Selection.Min.X - c.CenterX
		y0 := row*CellSize - statusBarAbove[kind]
		if c.TileSize > 1 {
			y0 = row*CellSize + CellSize/2 + c.Selection.Min.Y - c.CenterY - statusBarOverBox +
				statusBarAbove[HealthBar] - statusBarAbove[kind]
		}
		return image.Rect(x0, y0, x0+c.Selection.Dx(), y0+StatusBarHeight), true
	}
	side = max(side, 1)
	x0 := col*CellSize + side*CellSize/2 - StatusBarWidth/2
	y0 := row*CellSize - statusBarAbove[kind]
	return image.Rect(x0, y0, x0+StatusBarWidth, y0+StatusBarHeight), true
}

// StatusBarRun paints one row rectangle. HalfAdd selects destination halving.
type StatusBarRun struct {
	Rect    image.Rectangle
	Color   color.RGBA
	HalfAdd bool
}

// AppendStatusBarRuns paints caps, selected grey remainder and filled rows.
// Unselected show-health bars leave the remainder untouched. TERR-224/225/226
func AppendStatusBarRuns(dst []StatusBarRun, kind StatusBarKind, bar image.Rectangle, value, maximum int, faded bool) []StatusBarRun {
	if bar.Dx() < 2*statusBarCap {
		return dst
	}
	fill, ok := StatusBarFill(kind, bar.Dx(), value, maximum)
	if !ok {
		return dst
	}
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
		p := bar.Min.Add(image.Pt(statusBarCap, y))
		if !faded && fill < bar.Dx()-2*statusBarCap {
			start := p.Add(image.Pt(max(fill, 0), 0))
			g := statusBarGrey[y]
			dst = append(dst, StatusBarRun{Rect: image.Rect(start.X, start.Y, bar.Max.X-statusBarCap, start.Y+1),
				Color: statusBarWordColor(statusBarWord(color.RGBA{g, g, g, 255}))})
		}
		if fill > 0 {
			dst = append(dst, StatusBarRun{Rect: image.Rect(p.X, p.Y, p.X+fill, p.Y+1),
				Color: statusBarWordColor(statusBarWord(statusBarRow(kind, value, maximum, y))), HalfAdd: faded})
		}
	}
	return dst
}
