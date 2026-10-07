package game

import (
	"fmt"
	"image"

	"againrom/pkg/formats/spr16"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// The 28-cursor registry (docs/1030-cursor-lifecycle B1). This is the tier
// that reaches both the formats and the drawing tiers at once, on
// AttackCursorPath's own reason above: what crosses out of it is 28 RESOLVED
// SLOTS, each a name, a set of premultiplied pictures, a hotspot, a frame
// count and a period — no archive, no container, no per-format knowledge —
// so pkg/ui gains nothing about .16a or .256 for carrying them.
//
// SLOT ORDER, NAMES, ART PATHS, HOTSPOTS AND PERIOD ARGUMENTS are
// `SPR16A-CURSOR-067` (order and names) and `SPR16A-CURSOR-046` /
// `SPR256-CURSOR-046` (each registration's own art, hotspot and period),
// transcribed here as the complete join `cursor-construction.tsv` gives it.
// The table below is in slot order, which the two claims agree is
// construction order: slots 9-16, the eight arrow cursors, are NOT in
// arrow-number order (arrow0, arrow4, arrow6, arrow2, arrow7, arrow5,
// arrow1, arrow3) — transcribed exactly as decoded rather than sorted,
// since sorting them would silently rebuild the wrong registry.
type cursorFormat int

const (
	cursor16a cursorFormat = iota
	cursor256
)

// cursorRegistration is one row of the 28-slot table before its art is
// resolved: the data G2 names as executable constants (hotspot, period) next
// to the data it names as file bytes (the path, and the frame count the
// decoded sheet itself carries).
type cursorRegistration struct {
	name         string
	path         string // relative to graphicsPrefix
	format       cursorFormat
	hotspotX     int
	hotspotY     int
	periodMillis int64
}

// cursorRegistrations is the 28 rows, in slot order (SPR16A-CURSOR-067). Period
// arguments and hotspots are `SPR16A-CURSOR-046`'s and `SPR256-CURSOR-046`'s;
// frame counts are not listed here because they are a property of the decoded
// sheet, read at load and not asserted twice (G2: "no byte of any shipped
// file" changes when this table is edited, which would not be true of a
// second, independent frame-count literal).
var cursorRegistrations = []cursorRegistration{
	{"default", "cursors/default/sprites.16a", cursor16a, 5, 5, 2000000000},
	{"move", "cursors/move/sprites.16a", cursor16a, 15, 15, 100},
	{"swarm", "cursors/swarm/sprites.16a", cursor16a, 21, 21, 100},
	{"attack", "cursors/attack/sprites.16a", cursor16a, 3, 3, 100},
	{"defend", "cursors/defend/sprites.16a", cursor16a, 15, 13, 100},
	{"select", "cursors/select/sprites.16a", cursor16a, 3, 4, 100},
	{"patrol", "cursors/patrol/sprites.16a", cursor16a, 8, 25, 100},
	{"cast", "cursors/cast/sprites.16a", cursor16a, 15, 15, 100},
	{"pickup", "cursors/pickup/sprites.16a", cursor16a, 12, 13, 66},
	{"arrow0", "cursors/arrow0/sprites.16a", cursor16a, 15, 5, 2000000000},
	{"arrow4", "cursors/arrow4/sprites.16a", cursor16a, 16, 25, 2000000000},
	{"arrow6", "cursors/arrow6/sprites.16a", cursor16a, 6, 16, 2000000000},
	{"arrow2", "cursors/arrow2/sprites.16a", cursor16a, 25, 15, 2000000000},
	{"arrow7", "cursors/arrow7/sprites.16a", cursor16a, 8, 9, 2000000000},
	{"arrow5", "cursors/arrow5/sprites.16a", cursor16a, 8, 23, 2000000000},
	{"arrow1", "cursors/arrow1/sprites.16a", cursor16a, 22, 8, 2000000000},
	{"arrow3", "cursors/arrow3/sprites.16a", cursor16a, 23, 22, 2000000000},
	{"sdefault", "cursors/sdefault/sprites.16a", cursor16a, 2, 2, 2000000000},
	{"smove", "cursors/smove.256", cursor256, 0, 0, 2000000000},
	{"sattack", "cursors/sattack.256", cursor256, 0, 0, 2000000000},
	{"sdefend", "cursors/sdefend.256", cursor256, 0, 0, 2000000000},
	{"spatrol", "cursors/spatrol.256", cursor256, 0, 0, 2000000000},
	{"scast", "cursors/scast.256", cursor256, 0, 0, 2000000000},
	{"cantput", "cursors/cantput/sprites.16a", cursor16a, 38, 36, 2000000000},
	{"town", "cursors/town/sprites.16a", cursor16a, 16, 16, 2000000000},
	{"dice", "cursors/dice/sprites.16a", cursor16a, 16, 16, 100},
	{"wait", "cursors/wait/sprites.16a", cursor16a, 16, 16, 100},
	{"backpack", "cursors/backpack/sprites.16a", cursor16a, 16, 16, 100},
}

// LoadCursorRegistry resolves all 28 cursor registrations to pictures a
// front-end can hand to pkg/ui (App.SetCursorRegistry, Viewer.SetCursorManager
// through the shared manager).
//
// A FAILURE ON ANY ONE SLOT FAILS THE WHOLE REGISTRY, on LoadAttackPointer's
// own rule: every failure is an error and none is a partial picture, named by
// the slot and the address it failed at, so a caller cannot silently run 27
// of 28 cursors.
func LoadCursorRegistry(src terrain.EntrySource) (*ui.CursorRegistry, error) {
	if src == nil {
		return nil, fmt.Errorf("cursor registry: no graphics archive")
	}
	slots := make([]ui.CursorSlot, len(cursorRegistrations))
	for i, reg := range cursorRegistrations {
		path := graphicsPrefix + reg.path
		b, err := src.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("cursor slot %q: %w", reg.name, err)
		}
		var frames []*image.RGBA
		switch reg.format {
		case cursor16a:
			frames, err = decodeCursor16AFrames(path, b)
		case cursor256:
			frames, err = decodeCursor256Frames(path, b)
		default:
			return nil, fmt.Errorf("cursor slot %q: unhandled format", reg.name)
		}
		if err != nil {
			return nil, err
		}
		slots[i] = ui.CursorSlot{
			Name:         reg.name,
			Frames:       frames,
			Hotspot:      image.Pt(reg.hotspotX, reg.hotspotY),
			FrameCount:   len(frames),
			PeriodMillis: reg.periodMillis,
		}
	}
	return ui.NewCursorRegistry(slots), nil
}

// decodeCursor16AFrames decodes every frame of a .16a cursor sheet to a
// premultiplied picture, on LoadAttackPointer's own per-cell rule
// (cursor.go): a painted cell's colour is the palette entry, its coverage
// (level+1)/16, carried into alpha; an unpainted cell is no pixel at all.
func decodeCursor16AFrames(path string, b []byte) ([]*image.RGBA, error) {
	// THE PALETTE IS DECLARED PRESENT, on LoadAttackPointer's own reasoning:
	// the container's own trailer carries a has-palette flag set on every
	// palette-bearing sheet in the archive, and every cursor sheet this
	// registry reads is one.
	sprite, err := spr16.DecodeA(b, true)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(sprite.Frames) == 0 {
		return nil, fmt.Errorf("%s holds no frames", path)
	}
	out := make([]*image.RGBA, len(sprite.Frames))
	for i, f := range sprite.Frames {
		if f.Width <= 0 || f.Height <= 0 {
			return nil, fmt.Errorf("%s frame %d is %dx%d and would draw nothing", path, i, f.Width, f.Height)
		}
		pic := image.NewRGBA(image.Rect(0, 0, f.Width, f.Height))
		for j, p := range f.Pixels {
			if !p.Painted {
				continue
			}
			c := cursorPixel(sprite.Palette, p)
			o := j * 4
			pic.Pix[o], pic.Pix[o+1], pic.Pix[o+2], pic.Pix[o+3] = c.R, c.G, c.B, c.A
		}
		out[i] = pic
	}
	return out, nil
}

// decodeCursor256Frames decodes every frame of a .256 cursor sheet. There is
// no coverage field in this format's pixel model (spr256.Pixel): a pixel is
// either opaque at the palette entry or fully transparent, so no premultiply
// arithmetic applies (unlike the .16a path above).
func decodeCursor256Frames(path string, b []byte) ([]*image.RGBA, error) {
	sprite, err := spr256.Decode(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(sprite.Frames) == 0 {
		return nil, fmt.Errorf("%s holds no frames", path)
	}
	out := make([]*image.RGBA, len(sprite.Frames))
	for i, f := range sprite.Frames {
		if f.Width <= 0 || f.Height <= 0 {
			return nil, fmt.Errorf("%s frame %d is %dx%d and would draw nothing", path, i, f.Width, f.Height)
		}
		pic := image.NewRGBA(image.Rect(0, 0, f.Width, f.Height))
		for j, p := range f.Pixels {
			if !p.Opaque {
				continue
			}
			var e spr256.Color
			if int(p.Index) < len(sprite.Palette) {
				e = sprite.Palette[p.Index]
			}
			o := j * 4
			pic.Pix[o], pic.Pix[o+1], pic.Pix[o+2] = e.R, e.G, e.B
			pic.Pix[o+3] = cursorFullByte
		}
		out[i] = pic
	}
	return out, nil
}
