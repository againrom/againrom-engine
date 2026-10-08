package sim

import (
	"slices"
	"sort"
)

// StructureID is a structure's identity within a world, on EntityID's own terms
// (world.go): an OPAQUE HANDLE a script's Target_Structure reference is bound
// to at compile time, not the map's own authored identifier. pkg/mapload
// resolves the type-4 record's own +0x12 word (`ALM-TRIG-046`) against the
// table it built the structure list from, the same way a Target_Unit
// reference is resolved against the map's own unit id before an EntityID ever
// reaches this package.
type StructureID uint32

// Structure is one live building. Fresh placements use their map-object index
// as ID. Saved source-only buildings receive independent native handles; absent
// ALM buildings are absent here too. Field42 is current health. Saved cell links
// select area targets independently of this record's geometric footprint.
//
// Field42 keeps the original offset in its name because script opcodes expose
// that word directly (`TRIG-CHECK-053`). Structure spawn and save evidence tie
// it to current health, paired with maximum health at +0x44 (`ALM-CLS-053`,
// `SAV-BLDG-037`), and the spell/destruction paths consume that same word.
//
// THE INITIAL VALUE COMES FROM THE DEFINITION TABLE, NOT FROM THE MAP.
// `ALM-CLS-053` names the type-4 spawn's own write of the Buildings table's
// `healthMax` into `obj+0x44`/`+0x42`, and `SAV-BLDG-037` measures that pair
// in the owner's original saves at the `healthMax` values `DAT-BLD-005` gives
// for those kinds. A builder seeds the field from that position;
// pkg/mapload.Structures is this tree's one such builder. A caller that
// constructs a Structure directly, as every test in this package does, gets
// the zero value and no table is consulted.
type Structure struct {
	Kind      uint16
	UseAmount int32
	ID        StructureID
	Field42   uint16
	MaxHealth uint16
	Col, Row  int32
	Width     uint8
	Height    uint8
	Attach    uint32
	Blocking  uint32
}

// rebuildStructureSlots restores explicit saved cell links when present. Legacy
// worlds replay the immutable placement list in canonical ID order.
// UNIT-STRUCTCELL-070: a selected
// mask bit registers one alias, and the first collision ends this placement
// without rolling back its prefix. The key truncates the sum, not x and y
// separately, and registration does not clip against the playable map.
//
// Slots never depend on current HP. Againrom retains
// ruined structures and their slots; UNIT-STRUCTDETACH-074 leaves the original
// HP-to-destructor transition Unknown. No runtime structure spawn/detach or
// shape mutation is implemented. The cache derives from the persisted saved
// cell overlay or the legacy shape, never from a constructor replay on SAV LOAD.
func (w *World) rebuildStructureSlots() {
	w.nativeStructurePlanes = w.hasSavedStructures && (len(w.savedStructures) == 0 || slices.ContainsFunc(w.savedStructures, func(s SavedStructure) bool { return !s.Class.Generated() }))
	w.structureSlots = nil
	if w.hasSavedStructures {
		for _, c := range w.savedStructureCells {
			if c.HasStructure {
				if w.structureSlots == nil {
					w.structureSlots = make(map[uint16]int)
				}
				w.structureSlots[c.Cell] = indexOfStructure(w.structures, c.ID)
			}
		}
		return
	}
	for i, s := range w.structures {
		if s.Attach == 0 {
			continue
		}
	registration:
		for dy := int32(0); dy < int32(s.Height); dy++ {
			for dx := int32(0); dx < int32(s.Width); dx++ {
				if s.Attach>>(uint32(dy*int32(s.Width)+dx)&31)&1 == 0 {
					continue
				}
				key := uint16((int32(uint8(s.Row))+dy)*256 + int32(uint8(s.Col)) + dx)
				if _, occupied := w.structureSlots[key]; occupied {
					break registration
				}
				if w.structureSlots == nil {
					w.structureSlots = make(map[uint16]int)
				}
				w.structureSlots[key] = i
			}
		}
	}
}

type StructureCellBinding struct {
	Cell uint16
	ID   StructureID
}

func (w *World) StructureOccupancy() []StructureCellBinding {
	rows := make([]StructureCellBinding, 0, len(w.structureSlots))
	for cell, index := range w.structureSlots {
		rows = append(rows, StructureCellBinding{cell, w.structures[index].ID})
	}
	slices.SortFunc(rows, func(a, b StructureCellBinding) int { return int(a.Cell) - int(b.Cell) })
	return rows
}

// indexOfStructure is indexOfEntity's own binary search (step.go), over a
// slice kept sorted by ascending id, returning -1 when the slice holds no
// such structure.
func indexOfStructure(structs []Structure, id StructureID) int {
	i := sort.Search(len(structs), func(i int) bool { return structs[i].ID >= id })
	if i < len(structs) && structs[i].ID == id {
		return i
	}
	return -1
}

// scriptStructure resolves a compiled reference to the structure a world
// still holds, on scriptEntity's own terms (script.go): a reference that
// never resolved at compile time, and one naming a structure this world does
// not hold, are the same answer — there is nothing to measure.
func (w *World) scriptStructure(id StructureID, has bool) (Structure, bool) {
	if !has {
		return Structure{}, false
	}
	i := indexOfStructure(w.structures, id)
	if i < 0 {
		return Structure{}, false
	}
	return w.structures[i], true
}
