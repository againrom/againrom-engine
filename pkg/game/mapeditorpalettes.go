package game

import (
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	terrain "againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// The map editor's bar lists placeable records in three palettes, and each
// record in exactly one (owner rule): Structures holds what a plain click
// selects and what can be destroyed or switched, plus bridges; Objects
// holds decoration and no tree; Trees holds the trees. A record the class
// data does not settle lists in Structures. This file is the one table
// that decides it; a placed record keeps its own record kind whatever
// palette placed it.

// structurePalette is the palette structure kind c lists in, given
// whether the Buildings table has an entry for it, and when the data does
// not settle it, the field that fails.
func structurePalette(c *terrain.StructureClass, inTable bool) (ui.Palette, string) {
	switch {
	case c == nil:
		return ui.PaletteStructures, "structures.reg: no class"
	case c.VariableSize:
		// The two wooden bridges (TERR-STRUCT-105).
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
