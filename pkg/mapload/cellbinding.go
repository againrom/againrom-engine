package mapload

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// BuildingCasters resolves a map's caster records against the buildings the
// world holds, standing, under each map object index. A key no type-4 record
// names builds none and the load goes on (R2-ENGINE-1006). The caster's owner
// is the building's type-4 word (R2-ENGINE-1010), which the engine reads as a
// roster slot as it reads a unit's owner word (DIV-4718).
func BuildingCasters(m *alm.Map, w *sim.World) []sim.BuildingCaster {
	if m == nil || w == nil {
		return nil
	}
	byKey := make(map[uint32]int, len(m.Objects))
	for i, o := range m.Objects {
		byKey[uint32(o.Field12)] = i
	}
	standing := standingStructuresByObject(w)
	var out []sim.BuildingCaster
	for _, r := range m.BuildingCasters() {
		i, ok := byKey[r.Key]
		if !ok {
			continue
		}
		s, ok := standing[i]
		if !ok {
			continue
		}
		out = append(out, sim.BuildingCaster{X: r.X, Y: r.Y, Spell: r.Spell, Power: r.Power, Reach: r.Reach,
			Owner: uint32(uint16(m.Objects[i].Field0E)), Cell: sim.StructureCellWord(s)})
	}
	return out
}

// standingStructuresByObject indexes the world's structures by map object
// index: the structure id on a world built from the map, whose records build
// with no health gate (R2-ENGINE-1006); on a loaded one the saved authored
// index, keeping the buildings still alive at load.
func standingStructuresByObject(w *sim.World) map[int]sim.Structure {
	saved, _, present := w.SavedStructures()
	authored := make(map[sim.StructureID]int, len(saved))
	for _, s := range saved {
		if s.HasAuthored {
			authored[s.ID] = int(s.AuthoredIndex)
		}
	}
	out := make(map[int]sim.Structure)
	for _, s := range w.Structures() {
		object, ok := int(s.ID), true
		if present {
			object, ok = authored[s.ID]
		}
		if ok && (!present || s.MaxHealth == 0 || int16(s.Field42) > 0) {
			out[object] = s
		}
	}
	return out
}

func bindCellTails(w *sim.World, m *alm.Map) error {
	bindings, err := m.CellBindings()
	if err != nil {
		return err
	}
	tails := make([]sim.CellTail, len(bindings))
	for i, b := range bindings {
		tails[i] = sim.CellTail{X: int32(b.X), Y: int32(b.Y),
			Bytes: [6]byte{b.Spell, b.Power, b.SourceX, b.SourceY, b.LastX, b.LastY}}
	}
	return w.DeclareCellTails(tails)
}
