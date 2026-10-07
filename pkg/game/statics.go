package game

import (
	"fmt"
	"image/color"

	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/render/terrain"
)

// REG-OBJ-047

// ObjectRegistry is the ADDRESS the object classes are loaded from: the graphics
// container's identity segment and the entry inside it, folded and
// forward-separated as every address is compared.
//
// The identity segment is part of the name because ONE string now identifies
// this registry across every container an install ships, so this package
// states which container its classes come from instead of leaving its caller
// to know it out of band.
const ObjectRegistry = graphicsPrefix + "objects/objects.reg"

// LoadStatics decodes the object classes and their drawn frames out of the
// graphics container into the render tier's bundle.
//
// The bundle is keyed by the PLACEMENT BYTE, so the walk below runs 1..255
// through pkg/data's ByCode once and the b-1 offset between a byte and a
// class identity stays in the package that decoded it. No render-tier code
// learns the registry's identity convention, and a per-cell lookup
// downstream is an array read that cannot be out of bounds. Byte 0 is *no
// object* and is never asked: Classes[0] stays nil. A class whose ID is
// outside [0, 254] is consequently unreachable by any placement byte and is
// not loaded — there is no byte that could name it.
//
// ONLY AN UNREADABLE OR UNPARSEABLE REGISTRY IS AN ERROR. Everything the
// contract's "Class to sprite frame" rule excludes is a skip: those classes are
// loaded with their canvas geometry and a nil Frame, so the placement builder
// can count them apart from bytes that name nothing at all. The read error is
// returned unwrapped because the source's own read already yields an
// *fs.PathError naming the address, which is what lets a front-end's -check test
// it with errors.Is(err, fs.ErrNotExist) (AC-7).
//
// Each distinct sheet is read and decoded AT MOST ONCE per load, including the
// ones that fail: the shipped registry's 82 classes share far fewer sheets than
// that, and several select different frames of one sheet by Index
// (SPR256-FRAME-023). The cache is per call and is dropped with it, so nothing
// here holds an archive's bytes past the load.
func LoadStatics(src terrain.EntrySource) (*terrain.StaticSet, error) {
	if src == nil {
		return nil, fmt.Errorf("%s: no graphics archive", ObjectRegistry)
	}
	stream, err := src.ReadFile(ObjectRegistry)
	if err != nil {
		return nil, err
	}
	r, err := reg.Parse(stream)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ObjectRegistry, err)
	}
	classes, err := data.LoadObjectClasses(r)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ObjectRegistry, err)
	}

	sheets := sheetCache{
		src:       src,
		decoded:   make(map[string]*spr256.Sprite),
		converted: make(map[string][]*terrain.StaticFrame),
	}
	set := new(terrain.StaticSet)
	for b := 1; b <= 0xff; b++ {
		c, ok := classes.ByCode(byte(b))
		if !ok {
			continue
		}
		// The canvas is carried across whatever it holds, negative included:
		// pkg/data's absent-everywhere default for these four keys is -1, and
		// StaticAnchor is defined at either sign. Deciding here that a class with
		// a negative canvas is undrawable would invent a fifth exclusion the
		// contract does not have.
		//
		// The WHOLE SHEET rides beside the one frame Index selects. A cycle names
		// its frames as Index + an offset, so every frame the timeline can reach
		// has to be drawable, and the memo behind frames() is per PATH — the
		// classes sharing a sheet share the converted slice rather than each
		// paying for a conversion of their own. Frame is left exactly as it was:
		// it is the drawability answer the census depends on, and this loader's
		// own arithmetic keeps the two in step.
		//
		// The timeline is pkg/data's expansion, carried across as plain ints.
		// Nothing in the render tier re-derives it, and nothing here reads Phases:
		// a class's phase count is set on classes carrying no arrays at all, so it
		// does not say whether a class has a cycle.
		path := c.SpritePath()
		set.Classes[b] = &terrain.StaticClass{
			Width:      int(c.Width),
			Height:     int(c.Height),
			CenterX:    int(c.CenterX),
			CenterY:    int(c.CenterY),
			FireObject: c.FireObject,
			Frame:      sheets.frame(path, c.Index),
			Frames:     sheets.frames(path),
			Index:      int(c.Index),
			Timeline:   c.Timeline(),
		}
	}
	// DeadObject is a section subscript, while the bundle is keyed by the
	// placement byte (subscript+1). The entire dead class supplies geometry;
	// TERR-SPR-042 selects sheet frame0 and suppresses its animation.
	for b := 1; b <= 0xff; b++ {
		c, ok := classes.ByCode(byte(b))
		if !ok || c.DeadObject < 0 || c.DeadObject >= 255 {
			continue
		}
		dead := set.Classes[c.DeadObject+1]
		if dead == nil || len(dead.Frames) == 0 || dead.Frames[0] == nil {
			continue
		}
		variant := *dead
		variant.Frame, variant.Index, variant.Timeline, variant.Dead = dead.Frames[0], 0, nil, nil
		set.Classes[b].Dead = &variant
	}
	return set, nil
}

// sheetCache reads and decodes each .256 sheet once per load, remembering the
// failures as well as the successes: a nil entry is a sheet that is absent or
// will not decode, and a second class naming it must not re-read it to find that
// out again. Presence in the map, never the value, is what says a path has been
// tried.
type sheetCache struct {
	src       terrain.EntrySource
	decoded   map[string]*spr256.Sprite
	converted map[string][]*terrain.StaticFrame
}

// sheet answers the decoded sheet a registry File path names, or nil for the two
// exclusions that are about the STREAM: an absent entry and one spr256 refuses.
//
// The registry's own path is what the memo is KEYED by, and the graphics
// identity is prefixed onto it AT THE READ: the registry names an entry
// inside its own container, and the address that reaches it is that name
// plus the segment saying which container. A class resolving no File at all
// arrives here as the empty path, which addresses a bare identity with no
// remainder and is absent by the grammar's own rule — the absent-sheet
// case, still needing no branch of its own. Neither failure is reported: the
// caller's answer to both is the same nil frame, and a load that stopped on
// either would fail a run on data the contract says to skip.
func (c *sheetCache) sheet(path string) *spr256.Sprite {
	if s, tried := c.decoded[path]; tried {
		return s
	}
	var s *spr256.Sprite
	if raw, err := c.src.ReadFile(graphicsPrefix + path); err == nil {
		if decoded, err := spr256.Decode(raw); err == nil {
			s = decoded
		}
	}
	c.decoded[path] = s
	return s
}

// frame converts the one frame a class's Index selects into the render tier's
// own frame type, or answers nil for any of the four exclusions.
//
// A NON-NIL FRAME IS THE WHOLE DRAWABILITY RULE, and this function is where
// that rule is decided. What it refuses is exactly the contract's list —
// no sheet, no palette, an Index outside the sheet — and nothing else. In
// particular a frame of no area, or one whose pixels are empty, IS returned:
// the blit draws nothing for it and the placement builder still places it,
// so the census records a cell that resolved rather than one that was
// skipped. Refusing such a frame here would move a decision into a place
// where the count can no longer say why a cell drew nothing.
//
// Index is int32 off the registry and is range-checked at BOTH ends. The
// negative end is not defensive: pkg/data's absent-everywhere default for Index
// is -1, so a class that omits the key reaches this with a negative selector,
// and a check written as a single unsigned comparison would select frame
// 4294967295's neighbourhood instead.
//
// The palette is the SHEET'S own and rides on the frame. Its entries are
// written at full opacity: the format's fourth palette byte is reserved and
// not an alpha channel, transparency here is structural, and StaticFrame's
// contract is that nothing may read an entry's alpha as one.
func (c *sheetCache) frame(path string, index int32) *terrain.StaticFrame {
	s := c.sheet(path)
	if s == nil || index < 0 || int(index) >= len(s.Frames) {
		return nil
	}
	src := s.Frames[index]
	out := &terrain.StaticFrame{Width: src.Width, Height: src.Height}

	// The palette-less exclusion. The bound is the DESTINATION's own length, so
	// a sheet whose palette is short of the array this fills cannot leave part of
	// it at the zero colour while the rest is the sheet's.
	if !s.HasPalette || len(s.Palette) != len(out.Palette) {
		return nil
	}
	for i, e := range s.Palette {
		out.Palette[i] = color.RGBA{R: e.R, G: e.G, B: e.B, A: 0xff}
	}

	// Copied, never aliased: spr256's grid is the decoder's own memory, and the
	// cache hands one sheet to every class that names it.
	out.Pixels = make([]terrain.StaticPixel, len(src.Pixels))
	for i, p := range src.Pixels {
		out.Pixels[i] = terrain.StaticPixel{Index: p.Index, Opaque: p.Opaque}
	}
	return out
}

func (c *sheetCache) frames(path string) []*terrain.StaticFrame {
	if out, tried := c.converted[path]; tried {
		return out
	}
	var out []*terrain.StaticFrame
	if s := c.sheet(path); s != nil {
		for i := range s.Frames {
			f := c.frame(path, int32(i))
			if f == nil {
				out = nil
				break
			}
			out = append(out, f)
		}
	}
	c.converted[path] = out
	return out
}
