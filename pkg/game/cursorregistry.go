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

// attackSlot is the registration AttackCursorPath names: the sheet the attack
// pointer and the registry's "attack" slot share, read and decoded once.
const attackSlot = 3

var attackRegistration = cursorRegistrations[attackSlot]

// LoadCursorRegistry resolves all 28 cursor registrations to pictures a
// front-end can hand to pkg/ui (App.SetCursorRegistry, Viewer.SetCursorManager
// through the shared manager).
//
// A FAILURE ON ANY ONE SLOT FAILS THE WHOLE REGISTRY, on LoadAttackPointer's
// own rule: every failure is an error and none is a partial picture, named by
// the slot and the address it failed at, so a caller cannot silently run 27
// of 28 cursors.
func LoadCursorRegistry(src terrain.EntrySource) (*ui.CursorRegistry, error) {
	art := loadCursorArt(src)
	return art.registry, art.registryErr
}

// cursorArt is the attack pointer and the cursor registry, each with the
// reason it is missing.
type cursorArt struct {
	pointer     *image.RGBA
	pointerErr  error
	registry    *ui.CursorRegistry
	registryErr error
}

// loadCursorArt decodes every registration once: the attack pointer is frame 0
// of the same decoded attack sheet the registry's slot holds.
func loadCursorArt(src terrain.EntrySource) cursorArt {
	if src == nil {
		return cursorArt{
			pointerErr:  fmt.Errorf("%s: no graphics archive", AttackCursorPath),
			registryErr: fmt.Errorf("cursor registry: no graphics archive"),
		}
	}
	var art cursorArt
	slots := make([]ui.CursorSlot, len(cursorRegistrations))
	for i, reg := range cursorRegistrations {
		frames, err := loadCursorSlot(src, reg)
		if i == attackSlot {
			art.pointer, art.pointerErr = attackPointer(frames, err)
		}
		if err != nil {
			if art.registryErr == nil {
				art.registryErr = fmt.Errorf("cursor slot %q: %w", reg.name, err)
			}
			continue
		}
		slots[i] = ui.CursorSlot{
			Name:         reg.name,
			Frames:       frames,
			Hotspot:      image.Pt(reg.hotspotX, reg.hotspotY),
			FrameCount:   len(frames),
			PeriodMillis: reg.periodMillis,
		}
	}
	if art.registryErr == nil {
		art.registry = ui.NewCursorRegistry(slots)
	}
	return art
}

// loadCursorSlot reads one registration's sheet and decodes every frame. A read
// error comes back unwrapped, naming the address; a decode error is wrapped
// with it.
func loadCursorSlot(src terrain.EntrySource, reg cursorRegistration) ([]*image.RGBA, error) {
	path := graphicsPrefix + reg.path
	b, err := src.ReadFile(path)
	if err != nil {
		return nil, err
	}
	switch reg.format {
	case cursor16a:
		return decodeCursor16AFrames(path, b)
	case cursor256:
		return decodeCursor256Frames(path, b)
	}
	return nil, fmt.Errorf("%s: unhandled format", path)
}

// decodeCursor16AFrames decodes every frame of a .16a cursor sheet to a
// premultiplied picture through the one .16a frame converter. THE PALETTE IS
// DECLARED PRESENT: the container's trailer carries a has-palette flag set on
// every palette-bearing sheet in the archive, and every cursor sheet is one. A
// zero-area frame is refused: it would draw nothing, indistinguishable at the
// draw from a missing mode.
func decodeCursor16AFrames(path string, b []byte) ([]*image.RGBA, error) {
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
		out[i] = f.RGBA(sprite.Palette)
	}
	return out, nil
}

// decodeCursor256Frames decodes every frame of a .256 cursor sheet through the
// sheet's own palette and the one .256 frame converter: a pixel is opaque at
// its entry or a hole. A sheet without a palette is refused.
func decodeCursor256Frames(path string, b []byte) ([]*image.RGBA, error) {
	sprite, err := spr256.Decode(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	table, ok := sprite.Table()
	if !ok {
		return nil, fmt.Errorf("%s: no palette", path)
	}
	if len(sprite.Frames) == 0 {
		return nil, fmt.Errorf("%s holds no frames", path)
	}
	out := make([]*image.RGBA, len(sprite.Frames))
	for i, f := range sprite.Frames {
		if f.Width <= 0 || f.Height <= 0 {
			return nil, fmt.Errorf("%s frame %d is %dx%d and would draw nothing", path, i, f.Width, f.Height)
		}
		out[i] = f.RGBA(table)
	}
	return out, nil
}
