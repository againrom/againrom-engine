package game

import (
	"fmt"
	"image"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/databin"
	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/vfs"
)

// StructureRegistry is the ADDRESS the structure classes are loaded from:
// the graphics container's identity segment and the entry inside it, folded
// and forward-separated as every address is compared.
//
// It is spelt beside ObjectRegistry and off the same graphicsPrefix, so one
// renaming of the container moves both.
const StructureRegistry = graphicsPrefix + "structures/structures.reg"

const verticalWoodenBridgeClass = 33

// LoadStructures decodes the structure classes and their sheets out of the
// graphics container into the render tier's bundle.
//
// THE BUNDLE IS KEYED BY THE PLACEMENT KEY'S LOW BYTE, so the walk below
// runs 1..255 through pkg/data's ByID once and the registry's identity
// convention stays in the package that decoded it. This registry's class
// array is subscripted by ID rather than by section index (REG-KEY-044), so
// there is no offset here to get wrong — which is the one way this walk
// differs from the object loader's b-1. Byte 0 is asked for nothing and
// Classes[0] stays nil: the ID domain is 1..66, so no key's low byte of 0
// can name a class. A class whose ID is outside [1, 255] is consequently
// unreachable by any placement key and is not loaded — there is no byte
// that could name it.
//
// ONLY AN UNREADABLE OR UNPARSEABLE REGISTRY IS AN ERROR. The read error comes
// back unwrapped because the source's own read already yields an *fs.PathError
// naming the address, which is what lets a front-end's -check test it with
// errors.Is(err, fs.ErrNotExist).
//
// EVERY SHEET IS TAKEN WHOLE. A structure addresses its frames by a grid index
// and then by two further blocks past it (spec, "The three blocks"), so there is
// no single frame for a class to select and no Index key to select one with —
// which is why nothing here has the object loader's frame/frames pair. The memo
// behind frames() is per path, so two classes naming one sheet receive ONE
// slice, pointer for pointer.
func LoadStructures(src terrain.EntrySource) (*terrain.StructureSet, error) {
	if src == nil {
		return nil, fmt.Errorf("%s: no graphics archive", StructureRegistry)
	}
	stream, err := src.ReadFile(StructureRegistry)
	if err != nil {
		return nil, err
	}
	r, err := reg.Parse(stream)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", StructureRegistry, err)
	}
	classes, err := data.LoadStructureClasses(r)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", StructureRegistry, err)
	}

	// The OBJECT loader's own cache type, built here rather than a second one
	// declared beside it: presence in the maps, never the value, says a path has
	// been tried, and two caches would be two answers to that.
	sheets := sheetCache{
		src:       src,
		decoded:   make(map[string]*spr256.Sprite),
		converted: make(map[string][]*terrain.StaticFrame),
	}
	set := new(terrain.StructureSet)
	portraits := make(map[string]*image.RGBA)
	for b := 1; b <= 0xff; b++ {
		c, ok := classes.ByID(int32(b))
		if !ok {
			continue
		}
		var portrait *image.RGBA
		if c.Picture != "" {
			addr := graphicsPrefix + data.PortraitPath(c.Picture)
			var tried bool
			portrait, tried = portraits[addr]
			if !tried {
				portrait = loadPortrait(src, addr)
				portraits[addr] = portrait
			}
		}
		sc := &terrain.StructureClass{
			ID: c.ID, Name: c.DescText, Picture: c.Picture,
			Portrait:  portrait,
			Selection: image.Rectangle{Min: image.Pt(int(c.SelectionX1), int(c.SelectionY1)), Max: image.Pt(int(c.SelectionX2), int(c.SelectionY2))},
			// The three flags are the registry's own scalars as BOOLEANS: the render
			// tier never sees the int32 and so cannot come to read one of them as a
			// count. Each is "non-zero", which is the same answer at the Go zero an
			// omitted key resolves to today and at the registry's own decoded default
			// of 0 — the whole reason this story does not wait on that repair.
			VariableSize:   c.VariableSize != 0,
			Flat:           c.Flat != 0,
			Indestructible: c.Indestructible != 0,
			Usable:         c.Usable != 0,

			// ShadowY is copied across verbatim: this tier does not learn the
			// registry's spelling and recomputes no default of its own, exactly as
			// the three flags above do not.
			ShadowY: int(c.ShadowY),

			// LightRadius/LightPulse likewise cross unchanged (hotfix,
			// DIV-1313): see terrain.StructureClass's own doc for why this
			// tier reads them as a bare selector and nothing more.
			LightRadius: int(c.LightRadius),
			LightPulse:  int(c.LightPulse),
		}

		if int(c.TileWidth) > 0 && int(c.TileHeight) > 0 && int(c.FullHeight) > 0 {
			sc.TileWidth, sc.TileHeight, sc.FullHeight =
				int(c.TileWidth), int(c.TileHeight), int(c.FullHeight)

			if !sc.VariableSize {
				sc.Frames = cellSquareSheet(sheets.frames(c.SpritePath()))
				structureAnimation(c, sc)
			} else if b == verticalWoodenBridgeClass {
				// bridge1v is the established variable selector: its exact nine
				// frames are a 3x3 top/middle/bottom by left/centre/right patch.
				frames := cellSquareSheet(sheets.frames(c.SpritePath()))
				if len(frames) == 9 {
					sc.Frames = frames
					sc.VariableLayout = terrain.VariableStructureVerticalNinePatch
				}
			}
		}
		set.Classes[b] = sc
	}
	return set, nil
}

// cellSquareSheet is the frame-size exclusion: the sheet unchanged when every
// frame of it is CellSize square, and nil when any frame is not (spec, seam 3).
//
// A FRAME IS A CELL, and that is what makes the anchor a cell: the strip's whole
// arithmetic — one draw per rectangle cell, the vertical step, the overhang
// stacking above the back row — is stated in whole cells, and a frame of another
// size has no defined place in it. 130 of 130 shipped structure sheets are
// uniform 32x32 (SPR256-STR-040), so this is unreachable on lawful data; it is
// the seam's guard, not a fallback, and it refuses the WHOLE class rather than
// dropping the odd frame, because a sheet that is not a grid of cells is not a
// grid this contract can address.
//
// The slice is returned unchanged and never copied, so two classes naming one
// sheet keep the pointer identity the memo gave them and the texture cache
// uploads it once. An empty sheet answers nil: a class with no frames draws
// nothing, which is the same answer an absent one gives.
func cellSquareSheet(frames []*terrain.StaticFrame) []*terrain.StaticFrame {
	if len(frames) == 0 {
		return nil
	}
	for _, f := range frames {
		if f == nil || f.Width != terrain.CellSize || f.Height != terrain.CellSize {
			return nil
		}
	}
	return frames
}

// StructureRecords is the ONE conversion from a decoded map's type-4 records
// to the placement records the structure layer reads.
//
// It copies three fields and derives nothing: the stored fixed-point anchor
// and the whole stored key travel across as they are, and the two
// conventions applied to them — the anchor cell's shift and the key's low
// byte — are applied together, at the builder's own one site. A conversion
// that masked the key here would put the registry's identity convention into
// the map layer, where a later reader could not tell a masked key from a
// stored one.
//
// It reads m.Objects, which is the type-4 record list whatever its field is
// called: the "objects/structures" label on that record was refuted, and a type-4
// record resolves against the structure roster alone (ALM-OBJ-019, amended).
func StructureRecords(objects []alm.Object) []terrain.StructureRecord {
	out := make([]terrain.StructureRecord, 0, len(objects))
	for i, o := range objects {
		rec := terrain.StructureRecord{ID: uint32(i), X: o.X, Y: o.Y, Key: o.Kind}
		if o.Kind == 0x21 && len(o.Ext) >= 8 {
			w, h := int(o.Ext[0]), int(o.Ext[4])
			if w+h > 0 {
				rec.VariableWidth, rec.VariableHeight = w, h
			}
		}
		out = append(out, rec)
	}
	return out
}

// structureAnimation fills the class's cycle, or leaves it empty.
//
// THE GATE IS A CONJUNCTION OF THREE and it is applied HERE, once, at load:
//
//   - Phases > 1. One phase is not a cycle, and the scalar is the registry's own
//     gate on the three animation keys (REG-STR-080).
//   - the run-length expansion of AnimTime/AnimFrame is non-empty. Ten classes
//     spell Phases > 1 and then spell no timeline at all, so the scalar alone
//     does not say a class animates and the expansion is the only thing that can.
//   - AnimMask is non-empty and its length is EXACTLY TileWidth * FullHeight.
//
// THE MASK'S LENGTH IS A PRECONDITION AND NOT A HINT (seam 2, REG-STR-082). The
// mask pictures the SHEET GRID and not the rectangle, and the rival reading — the
// footprint's own TileWidth * TileHeight — is measurably wrong on six of the
// fourteen classes that spell a mask. A consumer taking it gets a mask too short,
// mis-ranks the live cells from the first divergence onward, and draws the wrong
// animation frame while failing nowhere. So a mask of another length fails the
// WHOLE gate rather than being truncated, padded or indexed: there is no correct
// interpretation of one to fall back to.
//
// Every scalar it reads decides the same thing at the Go zero an omitted key
// resolves to today and at the registry's own decoded default: Phases fails
// "> 1" at 0 and at -1 alike, and an absent AnimMask, AnimTime or AnimFrame
// is empty either way.
func structureAnimation(c *data.StructureClass, sc *terrain.StructureClass) {
	cells := sc.GridCells()
	if c.Phases <= 1 || cells <= 0 || len(c.AnimMask) != cells {
		return
	}
	timeline := c.Timeline()
	if len(timeline) == 0 {
		return
	}
	sc.Timeline = timeline
	sc.Rank, sc.Live = structureRanks(c.AnimMask)
}

// structureRanks reads the mask once: for each grid index, how many LIVE cells
// precede it, or -1 for a cell the mask retires; and how many are live in all.
//
// '-' RETIRES A CELL and every other byte leaves it live (REG-STR-082). The rank
// is EXCLUSIVE — cells strictly before this one — because the animation block
// stores one frame per live cell per phase, in grid order, so the first live cell
// is at offset 0 of its phase and not at offset 1.
//
// It is computed here and nowhere else, so the mask string is never
// re-scanned at draw time: a dead cell and a live one are one array read
// apart.
func structureRanks(mask string) ([]int, int) {
	rank := make([]int, len(mask))
	live := 0
	for i := 0; i < len(mask); i++ {
		if mask[i] == structureMaskDead {
			rank[i] = -1
			continue
		}
		rank[i] = live
		live++
	}
	return rank, live
}

// structureMaskDead is the mask byte that retires a grid cell: the cell never
// animates and always draws its base frame (REG-STR-082).
const structureMaskDead = '-'

// StructureTableRectangles is the DEFINITION TABLE's own rectangle per placement
// key: the sizeX/sizeY pair its buildings collection holds, keyed by the low byte
// a map's type-4 record names it with (spec AC-10's third observation).
//
// IT IS AN OBSERVATION'S INPUT AND NOTHING ELSE. This story's contract reads the
// REGISTRY rectangle — TileWidth x TileHeight — and nothing here reaches the
// draw: the two tables are independently sourced, nothing at this pin asserts
// that they agree, and a story that reads only the registry cannot be wrong about
// its own rectangle. The corpus harness reports the disagreement count and
// reconciles nothing.
//
// It lives HERE and not in the tool for one structural reason: the footprint
// resolver is in a tier the cmd layer may not import, and this package may import
// everything. So the harness names no table type at all — it asks this function
// and prints what comes back.
//
// It asks the resolver ONE CLASS AT A TIME through a PROBE MAP: one placement per
// key in [1, 255], each anchored at a column that IS its key. The resolver
// answers per placement and drops the keys it cannot resolve, so the anchor is
// what carries the identity through — a positional match against the input would
// be wrong the moment one key failed to resolve.
//
// A key absent from the result is one the table does not carry, which is a
// different fact from a rectangle of zero and is reported as absence.
func StructureTableRectangles(fsys *vfs.FS) (map[int][2]int, error) {
	if fsys == nil {
		return nil, fmt.Errorf("%s: no world archive", tableAddress)
	}
	b, err := fsys.ReadFile(tableAddress)
	if err != nil {
		return nil, err
	}
	f, err := databin.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", tableAddress, err)
	}

	probe := &alm.Map{Objects: make([]alm.Object, 0, 0xff)}
	for k := 1; k <= 0xff; k++ {
		probe.Objects = append(probe.Objects, alm.Object{X: uint32(k) << 8, Kind: uint32(k)})
	}
	footprints, _ := mapload.Footprints(probe, &mapload.Table{Buildings: f.Collection(databin.Buildings)})

	out := make(map[int][2]int, len(footprints))
	for _, fp := range footprints {
		out[fp.Col] = [2]int{fp.Width, fp.Height}
	}
	return out, nil
}
