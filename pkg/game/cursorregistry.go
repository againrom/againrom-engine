package game

import (
	"fmt"
	"image"

	"againrom/pkg/base"
	"againrom/pkg/formats/spr16"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// Cursor art and periods use SPR16A-CURSOR-046 and SPR256-CURSOR-046.
// AI-771 and R2-ENGINE-852 supply each game's hotspots through gameFiles.
type cursorFormat int

const (
	cursor16a cursorFormat = iota
	cursor256
)

// cursorRegistration names one art sheet and animation period.
type cursorRegistration struct {
	name         string
	path         string // relative to graphicsPrefix
	format       cursorFormat
	periodMillis int64
}

// cursorRegistrations keeps constructor slot order, including the unsorted arrows.
var cursorRegistrations = []cursorRegistration{
	{"default", "cursors/default/sprites.16a", cursor16a, 2000000000},
	{"move", "cursors/move/sprites.16a", cursor16a, 100},
	{"swarm", "cursors/swarm/sprites.16a", cursor16a, 100},
	{"attack", "cursors/attack/sprites.16a", cursor16a, 100},
	{"defend", "cursors/defend/sprites.16a", cursor16a, 100},
	{"select", "cursors/select/sprites.16a", cursor16a, 100},
	{"patrol", "cursors/patrol/sprites.16a", cursor16a, 100},
	{"cast", "cursors/cast/sprites.16a", cursor16a, 100},
	{"pickup", "cursors/pickup/sprites.16a", cursor16a, 66},
	{"arrow0", "cursors/arrow0/sprites.16a", cursor16a, 2000000000},
	{"arrow4", "cursors/arrow4/sprites.16a", cursor16a, 2000000000},
	{"arrow6", "cursors/arrow6/sprites.16a", cursor16a, 2000000000},
	{"arrow2", "cursors/arrow2/sprites.16a", cursor16a, 2000000000},
	{"arrow7", "cursors/arrow7/sprites.16a", cursor16a, 2000000000},
	{"arrow5", "cursors/arrow5/sprites.16a", cursor16a, 2000000000},
	{"arrow1", "cursors/arrow1/sprites.16a", cursor16a, 2000000000},
	{"arrow3", "cursors/arrow3/sprites.16a", cursor16a, 2000000000},
	{"sdefault", "cursors/sdefault/sprites.16a", cursor16a, 2000000000},
	{"smove", "cursors/smove.256", cursor256, 2000000000},
	{"sattack", "cursors/sattack.256", cursor256, 2000000000},
	{"sdefend", "cursors/sdefend.256", cursor256, 2000000000},
	{"spatrol", "cursors/spatrol.256", cursor256, 2000000000},
	{"scast", "cursors/scast.256", cursor256, 2000000000},
	{"cantput", "cursors/cantput/sprites.16a", cursor16a, 2000000000},
	{"town", "cursors/town/sprites.16a", cursor16a, 2000000000},
	{"dice", "cursors/dice/sprites.16a", cursor16a, 100},
	{"wait", "cursors/wait/sprites.16a", cursor16a, 100},
	{"backpack", "cursors/backpack/sprites.16a", cursor16a, 100},
}

// AI-771 and R2-ENGINE-852, in cursor registration order.
var firstCursorHotspots = [28]image.Point{
	{5, 5}, {15, 15}, {21, 21}, {3, 3}, {15, 13}, {3, 4}, {8, 25}, {15, 15}, {12, 13},
	{15, 5}, {16, 25}, {6, 16}, {25, 15}, {8, 9}, {8, 23}, {22, 8}, {23, 22},
	{2, 2}, {}, {}, {}, {}, {}, {38, 36}, {16, 16}, {16, 16}, {16, 16}, {16, 16},
}
var secondCursorHotspots = [28]image.Point{
	{4, 4}, {2, 3}, {2, 3}, {3, 3}, {16, 16}, {3, 3}, {8, 25}, {16, 16}, {16, 12},
	{16, 6}, {16, 28}, {5, 16}, {26, 16}, {8, 9}, {8, 24}, {23, 9}, {23, 24},
	{1, 1}, {}, {}, {}, {}, {}, {38, 36}, {16, 16}, {16, 16}, {16, 16}, {16, 16},
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
func LoadCursorRegistry(src terrain.EntrySource, games ...base.Game) (*ui.CursorRegistry, error) {
	art := loadCursorArt(src, games...)
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
func loadCursorArt(src terrain.EntrySource, games ...base.Game) cursorArt {
	var g base.Game
	if len(games) != 0 {
		g = games[0]
	}
	hotspots := filesOf(g).cursorHotspots
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
			Hotspot:      hotspots[i],
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
