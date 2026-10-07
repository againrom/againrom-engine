package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/sim"
)

func (want mover1160Source) documentDifferences(state *SnapshotSAVDocument) []string {
	if state == nil || state.Document == nil || state.Unavailable != "" {
		return []string{"complete mover Document unavailable"}
	}
	var differences []string
	population := 0
	for _, r := range state.Document.Objects {
		if unit1156Class(r.Class) {
			population++
		}
	}
	if population != len(want.records) || len(state.Document.Objects) != len(want.origins) {
		differences = append(differences, fmt.Sprintf("retained actors/objects %d/%d, raw %d/%d", population, len(state.Document.Objects), len(want.records), len(want.origins)))
	}
	seen := map[uint16]bool{}
	for _, r := range want.records {
		prefix := fmt.Sprintf("archive %d %s", r.archive, r.class)
		index := want.origins[r.archive]
		if index == 0 || int(index) > len(state.Document.Objects) || seen[index] {
			differences = append(differences, prefix+": absent/aliased DTO")
			continue
		}
		seen[index] = true
		got := state.Document.Objects[index-1]
		if got.Class != r.class {
			differences = append(differences, prefix+": retained class differs")
		}
		for _, block := range r.blocks() {
			count := 0
			var raw []byte
			for _, field := range got.Raw {
				if field.Name == block.Name {
					count++
					raw = field.Bytes
				}
			}
			if count != 1 || !bytes.Equal(raw, block.Bytes) {
				differences = append(differences, fmt.Sprintf("%s %s: retained count=%d bytes=%x, raw=%x", prefix, block.Name, count, raw, block.Bytes))
			}
		}
		for i, name := range mover1160Lists {
			count, value := 0, uint32(0)
			for _, field := range got.Counts {
				if field.Name == name {
					count++
					value = field.Count
				}
			}
			if count != 1 || value != uint32(len(r.routes[i])) {
				differences = append(differences, fmt.Sprintf("%s %s count: occurrences=%d retained=%d raw=%d", prefix, name, count, value, len(r.routes[i])))
			}
		}
	}
	return differences
}

type mover1160Population struct{ compared, current, active, handedOff, dying int }

func (want mover1160Source) worldDifferences(w *sim.World, state *SnapshotSAVDocument) ([]string, []string, mover1160Population) {
	motions, _, _, hasMotion := w.SavedActorMotions()
	_, orders, hasOrders := w.SavedGroups()
	differences, excluded, population := want.carrierDifferences(w.Entities(), motions, orders, state)
	if !hasMotion || !hasOrders {
		differences = append(differences, "saved motion/order authority absent")
	}
	return differences, excluded, population
}

// Admission is source-driven: every raw living or supported stage1 actor
// needs both carriers even if another survives. Unsupported execution is a
// named issue, never an exclusion from byte retention. Late-dead records
// still owe the complete Document comparison. Native, archive and DTO IDs
// are distinct namespaces; Token identity cannot redirect this join.
func (want mover1160Source) carrierDifferences(entities []sim.Entity, motions []sim.SavedActorMotion, orders []sim.SavedActorOrder, state *SnapshotSAVDocument) (differences, excluded []string, population mover1160Population) {
	byArchive := map[uint16]sim.Entity{}
	byMotion := map[sim.EntityID]sim.SavedActorMotion{}
	byOrder := map[sim.EntityID]sim.SavedActorOrder{}
	for _, e := range entities {
		if e.SourceBinding.Class == 0 {
			continue
		}
		if _, dup := byArchive[e.SourceBinding.ArchiveIndex]; dup {
			differences = append(differences, "duplicate live archive binding")
		}
		byArchive[e.SourceBinding.ArchiveIndex] = e
	}
	for _, m := range motions {
		if _, dup := byMotion[m.Entity]; dup {
			differences = append(differences, "duplicate motion entity")
		}
		byMotion[m.Entity] = m
	}
	for _, o := range orders {
		if _, dup := byOrder[o.Entity]; dup {
			differences = append(differences, "duplicate order entity")
		}
		byOrder[o.Entity] = o
	}
	for _, r := range want.records {
		prefix := fmt.Sprintf("archive %d %s offset %d", r.archive, r.class, r.off)
		e, found := byArchive[r.archive]
		if !found {
			if r.stage == 1 || r.stage == 0 && r.hp > 0 {
				differences = append(differences, prefix+": expected live mover/order source unavailable")
			} else {
				excluded = append(excluded, fmt.Sprintf("%s: late-dead/no live carrier stage=%d signed HP=%d; complete Document compared", prefix, r.stage, r.hp))
			}
			continue
		}
		delete(byArchive, r.archive)
		population.compared++
		if r.stage == 1 {
			population.dying++
		}
		if savedActorClass(e.SourceBinding.Class) != r.class {
			differences = append(differences, prefix+": live class differs")
		}
		bound := 0
		if state != nil {
			for _, b := range state.Actors {
				if b.EntityID == e.ID || b.ObjectIndex == want.origins[r.archive] {
					bound++
					if b.EntityID != e.ID || b.ObjectIndex != want.origins[r.archive] || b.Retired {
						differences = append(differences, prefix+": archive/DTO/entity projection differs")
					}
				}
			}
		}
		if bound != 1 {
			differences = append(differences, prefix+": absent/duplicate DTO projection")
		}
		if e.X != int32(r.position[0]) || e.Y != int32(r.position[1]) || e.Facing != r.mover[0] {
			differences = append(differences, prefix+": live cell/facing differs")
		}
		if !e.ActorLoad.Present || e.ActorLoad.Source.MoverSpeed != r.mover[10] {
			differences = append(differences, prefix+": live source mover byte10 differs")
		}
		m, found := byMotion[e.ID]
		delete(byMotion, e.ID)
		if !found {
			differences = append(differences, prefix+": required motion missing")
		} else {
			if m.Current {
				population.current++
			} else {
				population.handedOff++
			}
			if m.Active {
				population.active++
			}
			p := r.position
			expected := sim.SavedActorPosition{Cell: binary.LittleEndian.Uint16(p[:]), PackedCell: binary.LittleEndian.Uint16(p[2:]), FineX: p[4], FineY: p[5], Residue: binary.LittleEndian.Uint16(p[6:]), TerrainKey: binary.LittleEndian.Uint32(p[8:])}
			if m.Position != expected {
				differences = append(differences, prefix+": motion Position12 differs")
			}
			if m.Mover != r.mover {
				differences = append(differences, prefix+": motion Mover180 differs")
			}
			if !slices.Equal(m.StaticRoute, r.routes[0]) || !slices.Equal(m.DynamicRoute, r.routes[1]) {
				differences = append(differences, prefix+": separate static/dynamic route differs")
			}
		}
		o, found := byOrder[e.ID]
		delete(byOrder, e.ID)
		if !found {
			differences = append(differences, prefix+": required order missing")
		} else {
			if !bytes.Equal(o.Raw[:], r.order[:144]) || o.Authored || o.RepairStage != r.stage {
				differences = append(differences, prefix+": live Order144/stage/authorship differs")
			}
			if !slices.Equal(o.Patrol, r.routes[2]) {
				differences = append(differences, prefix+": order path differs")
			}
		}
	}
	for id := range byArchive {
		differences = append(differences, fmt.Sprintf("live archive %d absent from raw actor population", id))
	}
	for id := range byMotion {
		differences = append(differences, fmt.Sprintf("orphan motion entity %d", id))
	}
	for id := range byOrder {
		differences = append(differences, fmt.Sprintf("orphan order entity %d", id))
	}
	slices.Sort(differences)
	return
}
