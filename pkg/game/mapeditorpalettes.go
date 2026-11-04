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

// structurePalette is the palette structure kind c lists in, given
// whether the Buildings table has an entry for it, and when the data does
// not settle it, the field that fails.
func structurePalette(c *terrain.StructureClass, inTable bool) (ui.Palette, string) {
	switch {
	case c == nil:
		return ui.PaletteStructures, "structures.reg: no class"
	case objectsKinds[c.ID]:
		return ui.PaletteObjects, ""
	case c.VariableSize && c.VariableLayout != terrain.VariableStructureVerticalNinePatch:
		// A variable-size class the map format gives no extension and the
		// renderer no layout (TERR-STRUCT-105): the Horizontal Wooden Bridge.
		// A placed record of it still loads and
		// saves unchanged; only the palettes stop offering it.
		return ui.PaletteNone, ""
	case c.VariableSize:
		return ui.PaletteStructures, ""
	case c.Usable:
		// A hover hit and the target of the use order (UNIT-153,
		// UNIT-STRUCTHIT-138).
		return ui.PaletteStructures, ""
	case !c.Indestructible:
		// A plain click selects it (MISSION-065, AI-SELECT-122).
		return ui.PaletteStructures, ""
	case c.Flat:
		// VariableSize is the only bridge field: a flat class may be a
		// fixed bridge, which the data cannot tell from a floor mark.
		return ui.PaletteStructures, "Flat: no field marks a fixed bridge"
	case !inTable:
		return ui.PaletteStructures, "Buildings: no maximum health"
	}
	// Indestructible, not usable, not flat: it draws no change and takes no
	// click (TERR-STRUCT-210), whatever its maximum health.
	return ui.PaletteObjects, ""
}

// objectPalette is the palette object class c lists in: a still-standing
// destructible class names the class it leaves (DeadObject, REG-OBJ-047),
// and each class left carries FireObject -2. On shipped data these are the
// trees and bushes, standing and dead; every other class is decoration.
func objectPalette(c *data.ObjectClass) ui.Palette {
	if c != nil && (c.DeadObject >= 0 || c.FireObject == -2) {
		return ui.PaletteTrees
	}
	return ui.PaletteObjects
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
		p, why := structurePalette(c, ok)
		palettes[int64(id)] = p
		if why != "" {
			unsettled[int64(id)] = why
		}
	}
	return palettes, unsettled
}
