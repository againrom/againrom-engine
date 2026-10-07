package synth

import (
	"image/color"
	"sort"
	"strconv"
)

// ---------------------------------------------------------------------------
// structures/structures.reg and the structure sheets its classes name
// (see docs/0054-structure-art/spec.md, "The sheet and its grid")
// ---------------------------------------------------------------------------

// The structure registry is the ONE class registry with no [Files] table and no
// inheritance: [Global] carries Count alone, and each [StructureN] section spells
// its own sprite path in File — a path, backslash-separated and extensionless,
// never an index (REG-STR-081). So StructuresReg takes no file table, which is
// the whole shape difference from ObjectsReg and UnitsReg.
//
// A structure sheet is a ROW-MAJOR GRID of tile-sized frames, TileWidth columns
// by FullHeight rows, image cell (k, c) at index k*TileWidth + c, every frame
// exactly one map cell (SPR256-STR-040). StructureSheet builds a sheet of a
// stated frame count at a stated frame size, so the three blocks a class can
// address — base, animation, ruin — are expressed by asking for the right total
// rather than by a builder that knows what a block is.

// StructuresReg assembles a synthetic structures/structures.reg: [Global] with
// Count, and one [StructureN] section per element of classes, dense over
// [0, len(classes)).
//
// It is deliberately MINIMAL, exactly as classReg is: it writes the two
// structural facts a loader cannot work without — the section names and
// their dense numbering — and asserts nothing about the keys.
//
// Count is DERIVED from the sections rather than taken as a parameter, so this
// builder cannot produce a registry whose [Global] disagrees with its own body.
// A fixture that needs that disagreement is a RegRaw one.
//
// The root's children go out NAME-SORTED under the sorted-children flag, as
// classReg's do and as every shipped registry is written: Global, Structure0,
// Structure1, Structure10, ... Inside a section the keys stay in the order the
// caller gave them, no shipped registry marking a section's child list sorted.
func StructuresReg(classes ...[]RegNode) []byte {
	children := []RegNode{
		{Name: "Global", Kind: regKindDir, Children: []RegNode{
			{Name: "Count", Kind: regKindInt, Int: int32(len(classes))},
		}},
	}
	for i, keys := range classes {
		children = append(children, RegNode{
			Name:     "Structure" + strconv.Itoa(i),
			Kind:     regKindDir,
			Children: keys,
		})
	}
	sort.Slice(children, func(i, j int) bool { return children[i].Name < children[j].Name })
	return Reg(regRootSorted, children)
}

// StructureSheetOptions parameterises a synthetic structure sheet.
//
// The zero value builds a palette-bearing sheet of no frames, which is a legal
// stream and the "sheet short of its own base block" fixture at its extreme.
type StructureSheetOptions struct {
	// Frames is how many frames the sheet holds, in grid order. A class whose
	// grid is TileWidth x FullHeight addresses [0, TileWidth*FullHeight) as its
	// base block, so a sheet of exactly that many frames carries the base block
	// and nothing else, and one of k*TileWidth*FullHeight + Phases*Live carries
	// the animation and ruin blocks too (SPR256-STR-041).
	Frames int

	// Width and Height are EVERY frame's size, unless Odd overrides it below.
	// The contract holds a structure sheet to CellSize-square frames and excludes
	// a class whose sheet breaks it (spec, seam 3), so a fixture for that
	// exclusion states a size that is not square or not 32.
	Width, Height int

	// Odd maps a frame index to the {width, height} that frame carries instead
	// of Width x Height. It is what builds the "a sheet holding ONE frame that is
	// not CellSize square" fixture without a caller laying out every other frame
	// by hand — the exclusion is about one frame, and a sheet that broke the size
	// everywhere could not tell a whole-sheet rule from a per-frame one. Nil is a
	// uniform sheet, which is what 130 of 130 shipped structure sheets are.
	Odd map[int][2]int

	// NoPalette omits the palette block and clears the trailer's has-palette
	// bit: the palette-less exclusion, an opt-OUT so the zero value builds the
	// ordinary sheet.
	NoPalette bool

	// Palette is the sheet's colours, written [B, G, R, 0] each; entries beyond
	// the slice stay zero.
	Palette []color.RGBA

	// Ink answers the pixel at (frame, x, y). A nil Ink makes every frame wholly
	// transparent, which decodes to a frame of the stated size holding no opaque
	// pixel — legal, and drawable.
	Ink func(frame, x, y int) (index uint8, opaque bool)
}

// StructureSheet assembles a structure sheet: Frames frames of Width x Height,
// each filled from Ink, with the palette unless NoPalette.
//
// It is written over Sheet256 rather than beside it, so exactly one function in
// this package knows the .256 byte layout and a structure fixture cannot come to
// disagree with an object one about what a sheet is.
func StructureSheet(o StructureSheetOptions) []byte {
	frames := make([]Frame256, 0, max(o.Frames, 0))
	for i := 0; i < o.Frames; i++ {
		w, h := o.Width, o.Height
		if odd, ok := o.Odd[i]; ok {
			w, h = odd[0], odd[1]
		}
		f := Frame256{Width: w, Height: h}
		if o.Ink != nil && w > 0 && h > 0 {
			f.Pixels = make([]Pixel256, w*h)
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					idx, opaque := o.Ink(i, x, y)
					f.Pixels[y*w+x] = Pixel256{Index: idx, Opaque: opaque}
				}
			}
		}
		frames = append(frames, f)
	}
	return Sheet256(Sheet256Options{
		Palette:   o.Palette,
		NoPalette: o.NoPalette,
		Frames:    frames,
	})
}
