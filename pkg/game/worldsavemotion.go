package game

import (
	"againrom/pkg/sim"
	"encoding/binary"
)

func projectMotion(state *SnapshotSAVDocument, world *sim.World) error {
	motions, _, _, present := world.SavedActorMotions()
	if !present {
		for _, a := range state.Actors {
			if !a.Retired {
				motions = append(motions, sim.SavedActorMotion{Entity: a.EntityID, Position: sim.SavedActorPosition{TerrainKey: state.Document.World.TerrainIdentity}})
			}
		}
	}
	actors := map[sim.EntityID]uint16{}
	for _, a := range state.Actors {
		actors[a.EntityID] = a.ObjectIndex
	}
	entities := map[sim.EntityID]sim.Entity{}
	for _, e := range world.Entities() {
		entities[e.ID] = e
	}
	_, orders, _ := world.SavedGroups()
	hasOrder := map[sim.EntityID]bool{}
	for _, order := range orders {
		hasOrder[order.Entity] = true
	}
	// Cell slots and planes already came from the current spatial registry.
	// They are independent of Position; SAVE must not relocate their occupants
	// from a rendered crossing or reconstruct an untouched terrain cell.
	for _, old := range motions {
		if old.Current {
			continue
		}
		e, live := entities[old.Entity]
		index := actors[old.Entity]
		if !live {
			index = 0
			for _, dead := range world.OriginalDeadActors() {
				if dead.ID != old.Entity || dead.Current.Stage != 5 {
					continue
				}
				for _, root := range state.Document.DeadActors {
					if root == 0 {
						continue
					}
					key, _ := savedStructureValue(&state.Document.Objects[root-1], "Identity")
					if key == dead.Source.Identity {
						index = root
					}
				}
			}
			if index == 0 {
				continue
			}
		}
		if index == 0 {
			return worldSaveUnsupportedf("actor %d lacks current motion object binding", old.Entity)
		}
		// Death freezes crossing while retaining the actor row.
		m := old
		if live && old.Issue == "" {
			var err error
			m, err = sim.ProjectActorMotion(e, old, world.Route(e.ID), !present)
			if err != nil {
				return err
			}
		}
		r := &state.Document.Objects[index-1]
		p, err := savedMotionRaw(r, "Block12", 12)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint16(p, m.Position.Cell)
		binary.LittleEndian.PutUint16(p[2:], m.Position.PackedCell)
		p[4], p[5] = m.Position.FineX, m.Position.FineY
		binary.LittleEndian.PutUint16(p[6:], m.Position.Residue)
		binary.LittleEndian.PutUint32(p[8:], m.Position.TerrainKey)
		mover, err := savedMotionRaw(r, "U154", 180)
		if err != nil {
			return err
		}
		saved := m.SavedMover()
		copy(mover, saved[:])
		// Frozen crossing bytes must not override the current turn rate.
		if live {
			if e.ActorLoad.Source.Class != 0 {
				mover[10] = e.SourceNow().MoverSpeed
			} else {
				mover[10] = uint8(e.RotationSpeed)
			}
		}
		action, err := savedMotionRaw(r, "U54", 4)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(action, m.ActorAction)
		next, sites, err := cloneSavedGroupFieldRecord(r)
		if err != nil {
			return err
		}
		if err = sites.wordList(&next, "U15C", m.StaticRoute); err != nil {
			return err
		}
		if err = sites.wordList(&next, "U178", m.DynamicRoute); err != nil {
			return err
		}
		order, err := savedMotionRaw(&next, "U158", 148)
		if err != nil {
			return err
		}
		// An existing Order owns its cursor, mode and progress. Only an
		// absent Order needs the mover's constructor spelling.
		if live && !hasOrder[e.ID] {
			if e.HasTarget {
				order[8] = 1
			}
			var raw [144]byte
			copy(raw[:], order)
			raw = sim.ProjectMotionOrderTarget(raw, e)
			copy(order, raw[:])
			order[9] = 0
			if e.Transit > 0 {
				order[9] = 3
			}
		}
		*r = next
		if live && e.ActorLoad.Source.Class == 0 && e.NativeBasis.BlockKnown != 0 {
			position, err := savedMotionRaw(r, "Block12", 12)
			if err != nil {
				return err
			}
			for n := 4; n < len(e.NativeBasis.Block); n++ {
				if e.NativeBasis.BlockByteKnown(n) {
					position[n+2] = e.NativeBasis.Block[n]
				}
			}
		}
	}
	return nil
}
