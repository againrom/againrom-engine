package game

import (
	"encoding/binary"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func projectSavedWorldEffects(state *SnapshotSAVDocument, world *sim.World) error {
	// Drop obsolete area notices; the final writer constructs current roots.
	const nativeAreaNotice = "new native area objects have no original SAV graph/layer identities"
	const nativeDeliveryNotice = "pending native spell deliveries have no original SAV child graph bindings (save as AGS)"
	if (world.HasNativeAreaEffects() || world.PendingSpellDeliveries() > 0) && state.WorldEffects == nil {
		state.WorldEffects = &SnapshotSAVWorldEffects{Version: 1}
	}
	if state.WorldEffects != nil {
		notices := strings.Split(state.WorldEffects.Unavailable, "; ")
		notices = slices.DeleteFunc(notices, func(s string) bool { return s == "" || s == nativeAreaNotice || s == nativeDeliveryNotice })
		state.WorldEffects.Unavailable = strings.Join(notices, "; ")
	}
	drivers := world.SavedWorldEffectDrivers()
	if drivers != nil && state.Document == nil && len(drivers.Projectiles) > 0 {
		// Projectile records with no Document to continue into: the native
		// World still holds them, and the state says they have no bindings.
		if state.WorldEffects == nil {
			state.WorldEffects = &SnapshotSAVWorldEffects{Version: 1}
		}
		state.WorldEffects.Unavailable = unboundCarriedWorldEffects
		return nil
	}
	if drivers != nil && len(drivers.Projectiles) > 0 && (state.WorldEffects != nil || len(drivers.Areas) == 0) {
		if state.WorldEffects == nil {
			state.WorldEffects = &SnapshotSAVWorldEffects{Version: 1}
		}
		state.WorldEffects.bindProjectileDrivers(drivers.Projectiles)
	}
	if state.WorldEffects == nil {
		if drivers != nil && len(drivers.Areas)+len(drivers.Projectiles) != 0 {
			return fmt.Errorf("current world-effect Document bindings lost")
		}
		if len(world.SavedSpellEffects())+len(world.SavedProjectiles().Items) > 0 {
			state.WorldEffects = &SnapshotSAVWorldEffects{Version: 1, Unavailable: unboundCarriedWorldEffects}
			return nil
		}
		// No record is left but the allocator has moved: the Document's own
		// counter follows it.
		if current := world.SavedProjectiles(); state.Document != nil && current.FreeIndex != 0 {
			return projectProjectileCounter(state.Document, current)
		}
		return nil
	}
	if err := projectSavedSpellGraph(state, world); err != nil {
		return err
	}
	// A World that carries projectile records and no area driver answers its
	// Document's area rows as a World with no drivers does. Without a
	// Document there is no cell row to recompute.
	if drivers != nil && (len(drivers.Areas) > 0 || len(state.WorldEffects.Areas) == 0 && state.Document != nil) {
		if len(drivers.Areas) != len(state.WorldEffects.Areas) {
			return fmt.Errorf("current area binding count differs")
		}
		var retired []uint16
		for i, d := range drivers.Areas {
			if world.SavedSpellGraph() != nil {
				break
			}
			row := &state.WorldEffects.Areas[i]
			if row.ID != d.ID {
				return fmt.Errorf("current area identity differs")
			}
			if row.ObjectIndex == 0 {
				if d.Root >= 0 {
					return fmt.Errorf("current area was implicitly retired")
				}
				continue
			}
			if d.Root < 0 {
				state.Document.World.Effects = slices.DeleteFunc(state.Document.World.Effects, func(index uint16) bool { return index == row.ObjectIndex })
				retired = append(retired, row.ObjectIndex, row.ChildIndex)
				row.ObjectIndex, row.ChildIndex = 0, 0
				continue
			}
			e := world.SavedSpellEffects()[d.Root]
			object := &state.Document.Objects[row.ObjectIndex-1]
			savedObjectSetValue(object, "AE4C", uint32(e.AE4C))
			savedObjectSetValue(object, "SE40", uint32(e.SE40))
			block, err := savedActorRaw(object, "AE48", 4)
			if err != nil {
				return err
			}
			copy(block, e.AE48[:])
		}
		// Layer removal also recomputes the occupied-six-layer count, as
		// MAGIC-MAPLAYER-040 specifies. Other cell fields stay independently owned.
		byCell := map[uint16]sim.SavedCellRecord{}
		for _, cell := range world.SavedCellRecords() {
			byCell[cell.Cell] = cell
		}
		for i := range state.Document.World.Cells {
			cell := &state.Document.World.Cells[i]
			if current, ok := byCell[cell.Cell]; ok {
				cell.Layers = current.SpellEffects
				cell.LayerCount = current.LayerCount
			}
		}
		if len(retired) > 0 {
			doc, permutation, err := sav.RetireDocumentData(*state.Document, retired)
			if err != nil {
				return err
			}
			state.Document = &doc
			if err := remapSavedSackDocument(state, permutation); err != nil {
				return err
			}
		}
	} else if len(state.WorldEffects.Areas) != 0 {
		var retired []uint16
		for _, row := range state.WorldEffects.Areas {
			if row.ObjectIndex == 0 {
				continue
			}
			state.Document.World.Effects = slices.DeleteFunc(state.Document.World.Effects, func(index uint16) bool { return index == row.ObjectIndex })
			retired = append(retired, row.ObjectIndex, row.ChildIndex)
		}
		state.WorldEffects.Areas = nil
		if len(retired) != 0 {
			doc, permutation, err := sav.RetireDocumentData(*state.Document, retired)
			if err != nil {
				return err
			}
			state.Document = &doc
			if err := remapSavedSackDocument(state, permutation); err != nil {
				return err
			}
		}
	}
	if state.WorldEffects.Projectiles && len(state.WorldEffects.ProjectileIDs) > 0 {
		if err := projectSavedProjectiles(state.Document, world.SavedProjectiles(), state.WorldEffects.ProjectileIDs); err != nil {
			return err
		}
		if drivers == nil {
			state.WorldEffects.ProjectileIDs = nil
		}
	} else if current := world.SavedProjectiles(); state.Document != nil && len(current.Items) == 0 && current.FreeIndex != 0 {
		if err := projectProjectileCounter(state.Document, current); err != nil {
			return err
		}
	}
	if drivers != nil {
		state.WorldEffects.followProjectileDrivers(drivers.Projectiles)
	}
	return validateSavedWorldEffectsWorld(state, world)
}

// followProjectileDrivers makes the bound identity list exactly the live
// driver rows. A bound record whose driver is gone has been written out of the
// Document by the projection above, so its identity leaves the list with it.
func (m *SnapshotSAVWorldEffects) followProjectileDrivers(rows []sim.SavedProjectileDriver) {
	ids := make([]uint16, 0, len(rows))
	for _, d := range rows {
		ids = append(ids, d.ID)
	}
	slices.Sort(ids)
	if len(ids) == 0 {
		ids = nil
	}
	m.ProjectileIDs = ids
}

// projectProjectileCounter writes the allocator to a Document that carries no
// record: the counter leaf follows the World, and nothing else in the store
// changes. A Document without the leaf gets the whole empty section.
func projectProjectileCounter(doc *sav.DocumentData, current sim.SavedProjectiles) error {
	n, err := patchStateLeaf(doc.State.RootKind, doc.State.DirectoryRecords, doc.State.ValueRecords, "/Projectiles/FreeIndex", func(v *sav.CityStateValueData) error {
		if uint16(v.Int32) != current.FreeIndex {
			v.Int32 = int32(current.FreeIndex)
		}
		return nil
	})
	if err != nil || n > 0 {
		return err
	}
	return constructProjectileLeaves(doc, current, nil, map[string]bool{})
}

// bindProjectileDrivers admits every driver row whose record the Document
// does not yet carry: a record built by a release, not read from a SAV. A row
// already bound changes nothing; a retired row is bound too, so the identity
// list stays one to one with the driver rows.
func (m *SnapshotSAVWorldEffects) bindProjectileDrivers(rows []sim.SavedProjectileDriver) {
	for _, d := range rows {
		if slices.Contains(m.ProjectileIDs, d.ID) {
			continue
		}
		m.Projectiles = true
		m.ProjectileIDs = append(m.ProjectileIDs, d.ID)
	}
	slices.Sort(m.ProjectileIDs)
}

var projectileFieldNames = [...]string{"x", "y", "z", "picture", "dir", "phase", "lastaction", "action", "actiondir", "actiontarget", "actionx", "actiony", "actionz", "actionphase", "actionsegments", "actionspell"}

// Existing records retain their spelling and inactive raw domains. Complete
// current registered objects may create a Prj section and allocator/ID leaves;
// a partial existing section is never repaired by inventing defaults.
func projectSavedProjectiles(doc *sav.DocumentData, current sim.SavedProjectiles, boundIDs []uint16) error {
	values := map[string]int32{}
	items := map[string]bool{}
	for _, p := range current.Items {
		if !slices.Contains(boundIDs, p.ID) {
			continue
		}
		root := "/Prj" + strconv.Itoa(int(p.ID))
		items[root] = true
		fields := [...]int32{p.X, p.Y, p.Z, p.Picture, p.Dir, p.Phase, p.LastAction, p.Action, p.ActionDir, p.ActionTarget, p.ActionX, p.ActionY, p.ActionZ, p.ActionPhase, p.ActionSegments, p.ActionSpell}
		for i, name := range projectileFieldNames {
			values[root+"/"+name] = fields[i]
		}
	}
	var retired []string
	for _, id := range boundIDs {
		root := "Prj" + strconv.Itoa(int(id))
		if !items["/"+root] {
			retired = append(retired, root)
		}
	}
	retiredPath := func(path string) bool {
		return slices.ContainsFunc(retired, func(root string) bool { return stateRootMatches(doc.State.RootKind, path, root) })
	}
	doc.State.DirectoryRecords = slices.DeleteFunc(doc.State.DirectoryRecords, func(r sav.CityStateDirectoryData) bool { return retiredPath(r.Path) })
	doc.State.ValueRecords = slices.DeleteFunc(doc.State.ValueRecords, func(r sav.CityStateRecordData) bool { return retiredPath(r.Path) })
	state := &doc.State
	found := map[string]bool{}
	for _, path := range slices.Sorted(maps.Keys(values)) {
		n, _ := patchStateLeaf(state.RootKind, state.DirectoryRecords, state.ValueRecords, path, func(v *sav.CityStateValueData) error {
			v.Int32 = values[path]
			return nil
		})
		if n > 0 {
			found[path] = true
		}
	}
	free, _ := patchStateLeaf(state.RootKind, state.DirectoryRecords, state.ValueRecords, "/Projectiles/FreeIndex", func(v *sav.CityStateValueData) error {
		if uint16(v.Int32) != current.FreeIndex {
			v.Int32 = int32(current.FreeIndex)
		}
		return nil
	})
	var projectedIDs []uint16
	idLeaves, err := patchStateLeaf(state.RootKind, state.DirectoryRecords, state.ValueRecords, "/Projectiles/IDs", func(v *sav.CityStateValueData) error {
		if v.Kind == 2 {
			if slices.Contains(current.IDs, uint16(v.Int32)) {
				projectedIDs = append(projectedIDs, uint16(v.Int32))
				return nil
			}
			v.Kind, v.Int32, v.Bytes = 6, 0, nil
			return nil
		}
		if v.Kind != 6 || len(v.Bytes)%4 != 0 {
			return fmt.Errorf("projectile IDs encoding changed")
		}
		var kept []byte
		for at := 0; at < len(v.Bytes); at += 4 {
			word := binary.LittleEndian.Uint32(v.Bytes[at:])
			if slices.Contains(current.IDs, uint16(word)) {
				kept = append(kept, v.Bytes[at:at+4]...)
				projectedIDs = append(projectedIDs, uint16(word))
			}
		}
		v.Bytes = kept
		return nil
	})
	if err != nil {
		return err
	}
	if !slices.Equal(projectedIDs, current.IDs) {
		_, _ = patchStateLeaf(state.RootKind, state.DirectoryRecords, state.ValueRecords, "/Projectiles/IDs", func(v *sav.CityStateValueData) error {
			*v = sav.CityStateValueData{Kind: 6, Bytes: make([]byte, 4*len(current.IDs))}
			for j, id := range current.IDs {
				binary.LittleEndian.PutUint32(v.Bytes[4*j:], uint32(id))
			}
			return nil
		})
	}
	hasFree, hasIDs := free > 0, idLeaves > 0
	if !hasFree && current.FreeIndex != 0 || !hasIDs || len(found) != len(values) {
		if err := constructProjectileLeaves(doc, current, boundIDs, found); err != nil {
			return err
		}
	}
	return nil
}

// Compare current fields to the persisted state store before native adoption.
func validateSavedProjectileDocument(doc *sav.DocumentData, current sim.SavedProjectiles, boundIDs []uint16) error {
	expected := *doc
	expected.State = doc.State
	expected.State.DirectoryRecords = slices.Clone(doc.State.DirectoryRecords)
	expected.State.ValueRecords = slices.Clone(doc.State.ValueRecords)
	if err := projectSavedProjectiles(&expected, current, boundIDs); err != nil {
		return err
	}
	if !reflect.DeepEqual(expected.State, doc.State) {
		return fmt.Errorf("native current projectile Document differs")
	}
	return nil
}

func completeSavedProjectileRecord(doc *sav.DocumentData, id uint16) bool {
	ids, present, err := doc.State.ReadLeaf("/Projectiles/IDs")
	if err != nil || !present || ids.Kind != 2 && (ids.Kind != 6 || len(ids.Bytes)%4 != 0) {
		return false
	}
	free, present, err := doc.State.ReadLeaf("/Projectiles/FreeIndex")
	if err != nil || present && free.Kind != 2 {
		return false
	}
	root := "/Prj" + strconv.Itoa(int(id))
	for _, name := range projectileFieldNames {
		value, present, err := doc.State.ReadLeaf(root + "/" + name)
		if err != nil || !present || value.Kind != 2 {
			return false
		}
	}
	return true
}
