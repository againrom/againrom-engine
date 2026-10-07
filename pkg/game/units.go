package game

import (
	"fmt"
	"image"
	"image/color"

	"againrom/pkg/data"
	"againrom/pkg/formats/pal"
	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/render/terrain"
)

// UnitRegistry is the ADDRESS the unit classes are loaded from —
// ObjectRegistry's sibling, and the same shape: the graphics container's
// identity segment and the entry inside it, folded and forward-separated.
const UnitRegistry = graphicsPrefix + "units/units.reg"

// UnitOwnerPalette is the shared sixteen-table human owner palette.
const UnitOwnerPalette = graphicsPrefix + "units/humans/human.pal"

// LoadUnits decodes the unit classes, their sheets' every frame and their
// animation descriptors out of the graphics container into the render tier's
// bundle.
//
// The bundle is keyed by class ID — the key a map's unit record stores — used
// DIRECTLY: unlike the object layer there is no placement byte and no offset,
// so the walk is one entry per loaded class at its own ID and a record's id
// resolves by map lookup, missing cleanly on any value naming no class.
//
// ONLY AN UNREADABLE OR UNPARSEABLE REGISTRY IS AN ERROR, and the read error
// is returned unwrapped: the source's own read already yields an *fs.PathError
// naming the address, which is what lets a front-end's -check test it with
// errors.Is(err, fs.ErrNotExist).
//
// Frames is sheetCache.frames(SpritePath()) — THE WHOLE SHEET, converted
// once per distinct path and SHARED: two classes naming one sheet hold the
// one slice, pointer identity included, so the frame-keyed texture cache
// downstream uploads a shared sheet once. frames already answers nil for
// exactly the contract's exclusions — an absent, undecodable or
// palette-less sheet, a sheet of no frames included — so a class it
// refuses keeps its canvas and nil Frames, never fails the load. The cache
// is the statics loader's own type, instantiated per call and dropped with
// it.
//
// Anim is the data tier's own derivation copied value for value into the
// render tier's mirror type — the StaticPixel precedent: that tier may not
// import pkg/data, so the loader converts and re-derives nothing. The
// descriptor is filled for EVERY loaded class, frameless ones included: it
// is arithmetic over the class's resolved scalars, a fact about the class
// and not about any sheet.
func LoadUnits(src terrain.EntrySource) (*terrain.UnitSet, error) {
	if src == nil {
		return nil, fmt.Errorf("%s: no graphics archive", UnitRegistry)
	}
	stream, err := src.ReadFile(UnitRegistry)
	if err != nil {
		return nil, err
	}
	r, err := reg.Parse(stream)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", UnitRegistry, err)
	}
	classes, err := data.LoadUnitClasses(r)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", UnitRegistry, err)
	}

	sheets := sheetCache{
		src:       src,
		decoded:   make(map[string]*spr256.Sprite),
		converted: make(map[string][]*terrain.StaticFrame),
	}
	// The PER-TIER memo, beside the sheet cache rather than on it. It is keyed
	// on the sheet address AND the colour table's address together, because a
	// recoloured slice depends on both and a key naming one alone would hand a
	// second sheet the first sheet's pixels. It is a local of this loader
	// because tiers are a unit-registry fact: a field on the cache would be one
	// two of that type's three callers never touch. Presence, never the value,
	// says a pair has been tried.
	recoloured := make(map[string][]*terrain.StaticFrame)
	set := &terrain.UnitSet{Classes: make(map[int32]*terrain.UnitClass)}
	if raw, err := src.ReadFile(UnitOwnerPalette); err == nil {
		if tables, err := pal.DecodeOwnerTables(raw); err == nil {
			for i, table := range tables {
				set.OwnerPalettes[i] = tableRGBA(table)
			}
			set.HasOwnerPalettes = true
		}
	}
	dying := make(map[int32]int32, len(classes.All()))
	for _, c := range classes.All() {
		// The canvas is carried across whatever it holds: StaticAnchor is
		// defined at either sign, and refusing a shape here would invent an
		// exclusion the contract does not have.
		set.Classes[c.ID] = &terrain.UnitClass{
			Z:       int(c.Z),
			Width:   int(c.Width),
			Height:  int(c.Height),
			CenterX: int(c.CenterX),
			CenterY: int(c.CenterY),
			Frames:  sheets.frames(c.SpritePath()),
			Anim:    unitAnim(c.Anim()),
			// The registry's own TileSize, carried across unconverted: the
			// scale every effect-mark offset is multiplied by (1002).
			TileSize:  int(c.TileSize),
			Selection: image.Rectangle{Min: image.Pt(int(c.SelectionX1), int(c.SelectionY1)), Max: image.Pt(int(c.SelectionX2), int(c.SelectionY2))},
			// The class's own name text, ASSIGNED AND NOT CONVERTED. DescText is the
			// registry's bytes and this is the one site that could apply an encoding
			// to them; what makes that wrong rather than merely unnecessary is that
			// the font downstream indexes bytes, so a codec here would sit in front
			// of a correct one. A class carrying no DescText carries the empty name,
			// which is the zero value either way.
			Name: c.DescText,
			// The class's own picture NAME, carried across exactly as
			// DescText is and for the same reason (`UNIT-PICT-036`): it
			// is the registry's bytes, this loader neither formats an
			// address from it nor reads one, and a class that composes a
			// doll carries a value that names nothing.
			Portrait: c.InfoPicture,
			// One entry per tier the class declares, in tier order, nil for one that
			// could not be built. Empty for the 18 shipped classes that declare none.
			Tiers: classTiers(&sheets, recoloured, c),
			// Palette 0 is the shared-human owner-colour arm. This is
			// carried separately from len(Tiers): a class declaring a tier
			// whose file is absent also has no usable tier, but must not be
			// reclassified as a human body.
			OwnerShaded: c.Palette == 0,
			Projectile:  int(c.Projectile),
			ShootDelay:  int(c.ShootDelay),
			ShootOffset: shootOffsets(c.ShootOffset),
		}
		dying[c.ID] = c.Dying
	}
	for _, c := range classes.All() {
		cls := set.Classes[c.ID]
		cls.Boundary = sheets.boundaryFrames(cls.Frames, c.SpritePath(), c.OverlayPath())
	}
	linkCorpses(set.Classes, dying)
	return set, nil
}

// classTiers is the class's per-tier frame slices: one entry per tier the
// class DECLARES, in tier order, nil for one that could not be built.
//
// EVERY FAILURE IS A SKIP. An absent entry, a stream the decoder refuses and a
// class whose sheet did not load each leave that tier nil, and none is reported:
// the render tier answers the sheet's own frames for a nil tier, so the class
// still draws.
func classTiers(sheets *sheetCache, memo map[string][]*terrain.StaticFrame, c *data.UnitClass) [][]*terrain.StaticFrame {
	n := c.TierCount()
	if n == 0 {
		return nil
	}
	out := make([][]*terrain.StaticFrame, n)
	for i := range out {
		out[i] = sheets.tier(memo, c.SpritePath(), c.PalettePath(i+1))
	}
	return out
}

// tier is the sheet at sheetPath resolved through the colour table at palPath,
// memoised on the PAIR, or nil for any of the three ways it cannot be built: a
// sheet this loader already refused, an entry that is not there, and a stream
// the table decoder refuses.
//
// The address is prefixed with the graphics identity AT THE READ, exactly as
// the sheet's own is: the registry names an entry inside its own container,
// and the address that reaches it is that name plus the segment saying which
// container. A class resolving no File arrives here with both paths empty,
// which addresses a bare identity with no remainder and is absent by the
// grammar's own rule — no branch of its own needed.
func (c *sheetCache) tier(memo map[string][]*terrain.StaticFrame, sheetPath, palPath string) []*terrain.StaticFrame {
	// The two addresses are joined by a byte no archive address can hold, so
	// no two distinct pairs can collide on one key.
	key := sheetPath + "\x00" + palPath
	if out, tried := memo[key]; tried {
		return out
	}
	var out []*terrain.StaticFrame
	if base := c.frames(sheetPath); len(base) > 0 {
		if raw, err := c.src.ReadFile(graphicsPrefix + palPath); err == nil {
			if table, err := pal.Decode(raw); err == nil {
				out = repalette(base, tableRGBA(table))
			}
		}
	}
	memo[key] = out
	return out
}

// tableRGBA is the decoded colour table as the render tier's own palette array:
// the three read bytes at FULL OPACITY, exactly as a sheet's own palette is
// converted one file over.
//
// The alpha is not the table's — a table has none, its fourth byte being
// reserved and read by nobody — and it is not a transparency channel here
// either: which pixels of a frame are holes is that frame's own per-pixel
// answer, index 0 included.
func tableRGBA(t pal.Table) [256]color.RGBA {
	var out [256]color.RGBA
	for i, e := range t {
		out[i] = color.RGBA{R: e.R, G: e.G, B: e.B, A: 0xff}
	}
	return out
}

// repalette is base's every frame resolved through p — or BASE ITSELF
// where p is already base's palette.
//
// THE EQUALITY ARM IS THE POINT, not an optimisation bolted on afterwards. A
// tier whose table equals the sheet's own is not a recolour, and on both lawful
// roots that is what every class's tier 1 is: returning the base slice means one
// picture has one identity, so the frame-keyed texture cache downstream uploads
// it once instead of uploading two byte-identical copies. A modded table that
// genuinely differs takes the other arm and is honoured, which is the same
// decision read from the other side.
//
// THE PIXELS ARE SHARED, NOT COPIED, and by construction rather than by
// discipline: each new frame is a STRUCT COPY of the base frame with one field
// overwritten, so Pixels is the same slice header over the same backing array
// and there is no line here that could have deep-copied it. Everything
// downstream of the loader reads through these pointers and writes through
// none, which is what makes the sharing safe rather than merely cheap. A frame
// carries its palette as a value, so the assignment below cannot reach the base
// frame's own.
//
// The equality walk is per frame rather than on the first alone: a [256]color.RGBA
// is comparable, so each test is one array comparison, and asking every frame
// keeps the answer right for a slice whose frames do not share a palette instead
// of right only because today they do.
func repalette(base []*terrain.StaticFrame, p [256]color.RGBA) []*terrain.StaticFrame {
	unchanged := true
	for _, f := range base {
		if f.Palette != p {
			unchanged = false
			break
		}
	}
	if unchanged {
		return base
	}
	out := make([]*terrain.StaticFrame, len(base))
	for i, f := range base {
		tinted := *f
		tinted.Palette = p
		out[i] = &tinted
	}
	return out
}

// linkCorpses fills every class's corpse link from the id its Dying key
// holds.
//
// ONE HOP, and it is a subscript. The corpse and dying arms of the engine's
// own draw replace the class outright — they index the class table with that
// id and read the resolved class's sheet, layout switch and phase scalars in
// place of the unit's own (REG-UNITS-050) — and they index it ONCE. A chain
// walk with a visited set would be logic no evidence asks for, over a cycle
// branch no data can reach.
//
// It runs as a SECOND PASS, after every class is in the map, because the id a
// class names may belong to one built later in the first walk; resolving in
// place would make the link a function of the registry's own section order.
//
// Both guarded cases are ours to be total about rather than observed: over the
// shipped corpus every class names a dying class and every target resolves. A
// class naming itself lands on itself, which is neither special-cased nor
// forbidden here; a class naming one the bundle does not hold is left nil,
// which is the death path's first refusal and not an error.
//
// It is a plain function over the two maps and not a method on the bundle, so
// it is drivable from a hand-built pair with no registry, no archive and no
// sheet behind it. It DOES range one map, and the outcome cannot depend on
// that order: each round writes one field of one class named by its own key
// and reads a pointer no round writes, so every permutation of the walk leaves
// the same links behind.
func linkCorpses(classes map[int32]*terrain.UnitClass, dying map[int32]int32) {
	for id, target := range dying {
		if c := classes[id]; c != nil {
			c.Corpse = classes[target]
		}
	}
}

// unitAnim copies the data tier's descriptor into the render tier's mirror
// type FIELD FOR FIELD. No field is derived, clamped or validated here: the
// two types are one descriptor spelt in two tiers, and the moment this
// function computes anything the registry's semantics exist in a second
// place. The track slices are the data tier's own — Anim() builds them
// fresh per call and nothing else holds them.
func unitAnim(a data.UnitAnim) terrain.UnitAnim {
	return terrain.UnitAnim{
		S:           a.S,
		D:           a.D,
		MoveBase:    a.MoveBase,
		AttackBase:  a.AttackBase,
		DyingBase:   a.DyingBase,
		TailBase:    a.TailBase,
		MoveSlot:    a.MoveSlot,
		MoveWind:    a.MoveWind,
		DyingSlot:   a.DyingSlot,
		IdleSlot:    a.IdleSlot,
		BoneSlot:    a.BoneSlot,
		AttackSlot:  a.AttackSlot,
		Total:       a.Total,
		MoveTrack:   a.MoveTrack,
		IdleTrack:   a.IdleTrack,
		AttackTrack: a.AttackTrack,
		MoveOK:      a.MoveOK,
		IdleOK:      a.IdleOK,
		AttackOK:    a.AttackOK,
	}
}

// shootOffsets is a class's release offset array as plain ints, nil for an
// absent one.
func shootOffsets(raw []int32) []int {
	if len(raw) == 0 {
		return nil
	}
	out := make([]int, len(raw))
	for i, v := range raw {
		out[i] = int(v)
	}
	return out
}
