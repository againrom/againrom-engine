package game

import (
	"againrom/pkg/data"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	terrain "againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

func (x *MapInspector) cellSpellPalette() []ui.InspectionChoice {
	var out []ui.InspectionChoice
	for _, kind := range []uint16{0, 8, 4} {
		out = append(out, ui.InspectionChoice{Value: int64(mapedit.SpellConstructor(kind)), Label: x.inspectionUTF8(x.word(castWords[spellNewWord[kind]].Word))})
	}
	return out
}

// The map editor's bar lists placeable records in three palettes, and each
// record in exactly one (owner rule): Structures holds what a plain click
// selects and what can be destroyed or switched, plus bridges; Objects
// holds decoration and no tree; Trees holds the trees. A record the class
// data does not settle lists in Structures. This file is the one table
// that decides it; a placed record keeps its own record kind whatever
// palette placed it.

// objectsKinds are the structure kinds the owner placed in Objects, which
// their class fields cannot separate from the fixed bridges and floor marks
// that list in Structures: the Magic Symbol (structures.reg ID 27).
var objectsKinds = map[int32]bool{27: true}

var structureGridMapPalettes = map[ui.Palette]bool{ui.PaletteObjects: true}

// paletteInput is the one input of the palette decision: the facts a record
// family's decoder reads from a class. A family sets the fields its class
// carries and leaves the rest false; the decision names no family.
type paletteInput struct {
	// Owner: the owner placed the kind in Objects (objectsKinds).
	Owner bool
	// Unplaceable: a record of it loads and saves unchanged, and no palette
	// offers it.
	Unplaceable bool
	// Bridge: a variable-size class the renderer lays out.
	Bridge bool
	// Use: a hover hit and the target of the use order.
	Use bool
	// Select: a plain click selects it.
	Select bool
	// Tree: a tree or bush, standing or dead.
	Tree bool
	// Unsettled names the field that fails when the data does not settle
	// the record; it lists in Structures.
	Unsettled string
}

// editorPalette is the one decision: the palette the row lists in, and when
// the data does not settle it, the field that fails. A third record family
// adds a decoder and no branch here.
func editorPalette(r paletteInput) (ui.Palette, string) {
	switch {
	case r.Owner:
		return ui.PaletteObjects, ""
	case r.Unplaceable:
		return ui.PaletteNone, ""
	case r.Bridge, r.Use, r.Select:
		return ui.PaletteStructures, ""
	case r.Unsettled != "":
		return ui.PaletteStructures, r.Unsettled
	case r.Tree:
		return ui.PaletteTrees, ""
	}
	// Indestructible, not usable, not flat: it draws no change and takes no
	// click (TERR-STRUCT-210), whatever its maximum health.
	return ui.PaletteObjects, ""
}

// structurePaletteInput decodes structure class c, given whether the Buildings table
// has an entry for it.
func structurePaletteInput(c *terrain.StructureClass, inTable bool) paletteInput {
	switch {
	case c == nil:
		return paletteInput{Unsettled: "structures.reg: no class"}
	case objectsKinds[c.ID]:
		return paletteInput{Owner: true}
	case c.VariableSize && c.VariableLayout != terrain.VariableStructureVerticalNinePatch:
		// A variable-size class the map format gives no extension and the
		// renderer no layout (TERR-STRUCT-105): the Horizontal Wooden Bridge.
		// A placed record of it still loads and
		// saves unchanged; only the palettes stop offering it.
		return paletteInput{Unplaceable: true}
	}
	r := paletteInput{
		Bridge: c.VariableSize,
		// A hover hit and the target of the use order (UNIT-153,
		// UNIT-STRUCTHIT-138).
		Use: c.Usable,
		// A plain click selects it (MISSION-065, AI-SELECT-122).
		Select: !c.Indestructible,
	}
	switch {
	case c.Flat:
		// VariableSize is the only bridge field: a flat class may be a
		// fixed bridge, which the data cannot tell from a floor mark.
		r.Unsettled = "Flat: no field marks a fixed bridge"
	case !inTable:
		r.Unsettled = "Buildings: no maximum health"
	}
	return r
}

// objectPaletteInput decodes object class c: a still-standing destructible class
// names the class it leaves (DeadObject, REG-OBJ-047), and each class left
// carries FireObject -2. On shipped data these are the trees and bushes,
// standing and dead; every other class is decoration.
func objectPaletteInput(c *data.ObjectClass) paletteInput {
	return paletteInput{Tree: c != nil && (c.DeadObject >= 0 || c.FireObject == -2)}
}

// structurePalettes is the palette of every structure kind of set by value,
// with the unsettled ones' failing field.
func structurePalettes(set *terrain.StructureSet, table *mapload.Table) (map[int64]ui.Palette, map[int64]string) {
	palettes, unsettled := map[int64]ui.Palette{}, map[int64]string{}
	if set == nil {
		return palettes, unsettled
	}
	for id, c := range set.Classes {
		if c == nil {
			continue
		}
		_, ok := mapload.StructureMaxHealth(table, byte(id))
		p, why := editorPalette(structurePaletteInput(c, ok))
		palettes[int64(id)] = p
		if why != "" {
			unsettled[int64(id)] = why
		}
	}
	return palettes, unsettled
}
