package terrain

import (
	"image"
	"image/color"
)

// UnitClass is one unit class as this layer draws it: the canvas geometry
// its anchor is measured in, every frame of its sheet, and the animation
// descriptor selection reads.
//
// Width/Height are the class's declared canvas and (CenterX, CenterY) its
// ground-touching pixel, carried across as plain ints. They are NOT a frame's
// size — frames of one unit sheet need not share a size — which is why
// UnitPlace takes the canvas and the drawn frame's size apart, as
// StaticAnchor's own signature does for the object layer.
//
// Frames is the WHOLE SHEET in sheet order, and a POINTER INTO THE LOADER'S
// PER-PATH MEMO: two classes naming one sheet carry the one slice, pointer
// identity included, so the frame-keyed texture cache downstream uploads a
// shared sheet once. It is nil for a class the loader resolved but could not
// draw — an absent, undecodable or palette-less sheet, or one holding no
// frame at all. That is a skip and never an error (0024 spec, error cases):
// the entity keeps its square, and a resolved class with nil Frames stays a
// different answer from an id naming no class at all.
//
// Anim is the class's animation descriptor, copied value for value from the
// data tier's derivation by the loader (the StaticPixel precedent — this
// tier may not import pkg/data and re-derives nothing from a registry). Its
// values are carried, never validated: a selection it misdirects is caught
// by SelectUnitFrame's bounds guard against the sheet's own count.
type UnitClass struct {
	// Z selects the drawable category (REG-UNITS-061). It is not a pixel lift.
	Z       int
	Width   int
	Height  int
	CenterX int
	CenterY int
	Frames  []*StaticFrame
	Anim    UnitAnim

	// TileSize is the `units.reg` key of the same name, the actor's footprint
	// scale. Its registry default is 1, and a zero reaching a builder is
	// treated as 1 there.
	TileSize int

	// Selection is the class's `units.reg` selection box, SelectionX1..Y2
	// (REG-UNITS-049), in canvas pixels as the registry holds it. The status
	// bars of a class wider than one cell span its top edge (DIV-1460).
	Selection image.Rectangle

	// Name is this class's own name text, as the registry handed it over —
	// the loader's bytes, with NO character encoding applied and nothing
	// derived from them.
	//
	// It is carried VERBATIM, and that is load-bearing rather than lazy. The
	// font a consumer draws it with indexes BYTES — record k is character
	// 32+k, and a string is walked one byte at a time and never transcoded — so
	// a conversion applied here would be a second codec in front of a correct
	// one, and it would be the wrong one on exactly the release whose text is
	// not ASCII.
	//
	// The empty string is a class that carries none, which is a legal registry
	// section rather than an error. It is a plain string beside the geometry
	// for the reason Anim and Corpse are here: this type is the render tier's
	// mirror of one registry class, filled by a loader in the tier that may
	// read a registry, and a second map keyed by class id would be a parallel
	// structure with the same lifetime and the same filler.
	Name string

	// Portrait is this class's own picture NAME — the registry's `InfoPicture`,
	// verbatim — for the actors that are drawn from a flat bitmap rather than
	// from a composed figure (`UNIT-PICT-035`, `UNIT-PICT-036`).
	//
	// IT IS A NAME AND NOT A PATH, and not a picture either. Turning it into an
	// address is `pkg/data`'s (PortraitPath, PortraitTierPath — and there are
	// TWO addresses off this one name, which is the second reason it is carried
	// unresolved); opening that address is the wiring tier's. This tier neither
	// formats nor reads it.
	//
	// It is here for Name's own reason, one field up, and it inherits Name's
	// own caveat: on THIRTEEN shipped human classes it is DEAD DATA. Those
	// classes compose a doll and can never reach the formatter, and twelve of
	// the values they carry name no node on either root (`REG-PICT-083`). A
	// consumer must therefore ask data.ComposesFigure before it believes this
	// field means anything.
	//
	// The empty string is a class carrying no such key, which one of the 34
	// shipped classes is.
	Portrait string

	// Corpse is the class whose sheet carries this one's body — resolved by
	// the loader from the registry key naming it, ONCE, so no draw path
	// performs a lookup.
	//
	// It is a whole class and not a sheet because the substitution is a whole
	// class: the frames, the layout pair the direction rule reads and the
	// phase scalars the block arithmetic is derived from all come from the
	// resolved one, and its canvas comes with them.
	//
	// It POINTS AT ITSELF for a class naming itself, which is the common case,
	// and is NIL for one naming a class this bundle does not hold — the death
	// path's first refusal, and never an error. A nil Corpse is a different
	// answer from a Corpse whose Frames are nil, exactly as a missing entry is
	// a different answer from a frameless one.
	Corpse *UnitClass

	// Tiers is one frame slice per TIER of this class, Tiers[0] being tier 1
	// — the same sheet, every frame of it, resolved through that tier's own
	// shipped colour table.
	//
	// A TIER IS A COLOUR TABLE AND NOTHING ELSE. A tier's frames are the frames
	// of Frames with one field changed: same size, same pixel indices, the very
	// same pixel memory, a different palette. Nothing about a tier moves an
	// anchor, selects a different frame, changes a count or reaches the
	// animation descriptor, so every selection, placement and cull already
	// written is untouched by which slice it indexes.
	//
	// IT IS FILLED BY THE LOADER, like Frames, and this tier neither builds nor
	// validates one: decoding a colour table needs packages this tier may not
	// import. An entry may be nil or empty — a tier whose table was absent or
	// refused — and TierFrames answers the base slice for it, which is the
	// whole of "a tier that could not be built draws the sheet's own colours".
	//
	// Where a tier's table EQUALS the sheet's own, the loader puts the base
	// slice here, pointer for pointer, rather than a recoloured copy of it. So
	// on shipped data tier 1 IS Frames, and a consumer keyed on frame identity
	// — the window's texture cache is one — holds one entry for the two rather
	// than uploading one picture twice.
	Tiers [][]*StaticFrame

	// OwnerShaded says this class takes the shared human owner palette rather
	// than a class tier palette. It is the registry's Palette == 0 arm carried
	// by the loader; this tier does not infer it from Tiers, whose empty value
	// also covers a declared table which failed to load.
	OwnerShaded bool

	// Projectile is the projectiles.reg picture this class's ranged swing
	// releases, 0 for none, and ShootDelay the swing tick it leaves on
	// (ANIM-STATE-023). Both are the registry's own values.
	Projectile int
	ShootDelay int

	// ShootOffset is the registry's sixteen-entry release offset array: one
	// (x, y) pair per eight facings, in sprite pixels. Absent for a class
	// that declares none.
	ShootOffset []int

	// Boundary is the sheet's spritesb sibling, one frame per frame of Frames
	// at the same index. The shadow pass stamps the pair as the unit's second
	// silhouette. A class whose sibling sheet is absent or of another length
	// holds none.
	Boundary []*StaticFrame
}

// BoundaryOf is the spritesb frame paired with f, a frame of Frames or of any
// tier, or nil. It is defined on a nil receiver.
func (c *UnitClass) BoundaryOf(f *StaticFrame) *StaticFrame {
	if c == nil || f == nil || len(c.Boundary) == 0 {
		return nil
	}
	for i, g := range c.Frames {
		if g == f && i < len(c.Boundary) {
			return c.Boundary[i]
		}
	}
	for _, tier := range c.Tiers {
		if len(tier) != len(c.Boundary) {
			continue
		}
		for i, g := range tier {
			if g == f {
				return c.Boundary[i]
			}
		}
	}
	return nil
}

// TierFrames is the frame slice this class draws at a tier: its own where
// that tier has one, and the sheet's otherwise.
//
// IT IS TOTAL, and the four ways to have no tier of one's own are ONE ARM
// rather than four checks each caller performs: tier 0, which is what a
// placement stating no tier carries; a negative tier; a tier past this class's
// count; and a tier whose slice the loader left empty because its table was
// absent or refused. Every one of them answers Frames — the colours the sheet
// ships — which is what leaves a class with no tiers, and this build before
// this story, drawing exactly the picture it drew.
//
// THE TIER IS ONE-BASED because the shipped numbering is: a creature is its
// class's first, second, third or fourth version, and the column that says
// which reads 1 through 4. Subtracting one here, at the single site that
// subscripts, is what keeps that offset out of every caller.
//
// It is defined on a NIL RECEIVER — the answer is nil, which is the answer a
// class with no frames already gives — so a caller holding an unresolved class
// needs no guard of its own, exactly as the map lookup that produced it needed
// none.
//
// It reads its receiver and its argument, builds nothing and writes nothing, so
// two calls at one tier answer the same slice and no draw order can move it.
func (c *UnitClass) TierFrames(tier int) []*StaticFrame {
	if c == nil {
		return nil
	}
	if tier >= 1 && tier <= len(c.Tiers) {
		if f := c.Tiers[tier-1]; len(f) > 0 {
			return f
		}
	}
	return c.Frames
}

// UnitSet is the loaded unit-class bundle, keyed by class ID — the opaque
// signed key a map's unit record stores, used directly (spec, "Class id").
//
// The set is a map and not an array like StaticSet's because there is no
// placement byte to index by: the domain is a sparse signed 32-bit key, and
// a lookup misses cleanly at ANY value naming no class, negative included.
// The two artless answers are kept apart — no entry is an id naming no
// class, an entry whose Frames is nil is the excluded class — so a
// consumer can count them apart, as the object layer counts NoClass from
// NoFrame.
//
// A zero UnitSet is a legal, empty one — a lookup in a nil map is a miss —
// and this package establishes no invariant over the field, because it is not
// this package that fills it.
type UnitSet struct {
	Classes map[int32]*UnitClass

	// OwnerPalettes is the shared human palette's sixteen complete tables.
	// HasOwnerPalettes distinguishes a successfully decoded all-zero table set
	// from the zero value, which is the cosmetic fallback used when the resource
	// is absent or refused.
	OwnerPalettes    [16][256]color.RGBA
	HasOwnerPalettes bool

	// Bodies is the art a PLAYER'S CHARACTER is drawn from, keyed by body name.
	//
	// IT IS A SECOND MAP BECAUSE IT ANSWERS A SECOND QUESTION, and the two
	// populations are not one. An actor a map places is drawn as the class its
	// record names, through that record's own art; a player's character is drawn
	// as whatever its visible equipment says, from a sheet composed out of a
	// NAME — so a single map keyed by class id could not hold both, two
	// different characters resolving to one class id being ordinary.
	//
	// An entry IS a UnitClass and not a new type: what a body has to supply is
	// the canvas, the descriptor, the name, the corpse link and the frames,
	// which is exactly this type. The loader builds one by copying the class
	// record the body name resolved to and replacing the frames, so every
	// selection, placement, cull and texture path downstream receives what it
	// already received.
	//
	// KEYED BY THE PLAIN STRING, this tier being unable to import the package
	// that owns the body-name type at all. Filled by a loader outside this tier,
	// exactly as Classes is; an empty or nil map is a bundle whose loader
	// resolved no body, and every lookup in it misses — which draws the class
	// record's own art and is the picture this map's absence always gave.
	Bodies map[string]*UnitClass
}

// OwnerPalette returns the shared human palette selected by owner, or nil when
// the resource or class does not take that colouring arm. The original selector
// is the owner's low nibble, so owner 16 wraps to table 0 and owner 17 to table
// 1. The returned table is immutable after loading.
func (s *UnitSet) OwnerPalette(c *UnitClass, owner uint32) *[256]color.RGBA {
	if s == nil || !s.HasOwnerPalettes || c == nil || !c.OwnerShaded {
		return nil
	}
	return &s.OwnerPalettes[owner&0x0f]
}

func UnitPlace(col, row int, c *UnitClass, f *StaticFrame, mirror bool, lift, originY int) (StaticPlacement, bool) {
	if c == nil || f == nil {
		return StaticPlacement{}, false
	}
	destX, destY, anchorX, anchorY := StaticAnchor(
		col, row, c.Width, c.Height, c.CenterX, c.CenterY, f.Width, f.Height, lift, originY)
	return StaticPlacement{
		Cell:    image.Point{X: col, Y: row},
		TopLeft: image.Point{X: destX, Y: destY},
		Anchor:  image.Point{X: anchorX, Y: anchorY},
		Frame:   f,
		Mirror:  mirror,
	}, true
}
