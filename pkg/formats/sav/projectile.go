package sav

import (
	"encoding/binary"
	"fmt"
	"strconv"

	"againrom/pkg/formats/reg"
)

// The Projectiles state-store subtree (SAV-PROJSTORE-428, SAV-PROJLOAD-429):
// the manager's allocator word, the live IDs in wire order, and one
// Prj<decimal u16 id> section per ID with sixteen kind-2 leaves. Values carry
// the widths of narrowProjectileLeaf (SAV-1322); the rest stay int32.

const projectilesSection = "Projectiles"
const (
	projectilesFreeIndex = "FreeIndex"
	projectilesIDs       = "IDs"
)

// projectileLeafNames is worldProjectileLeaves' name order (pinned by
// projectile_test.go).
var projectileLeafNames = []string{
	"x", "y", "z", "picture", "dir", "phase", "lastaction", "action",
	"actiondir", "actiontarget", "actionx", "actiony", "actionz",
	"actionphase", "actionsegments", "actionspell",
}

// Projectile is one Prj<id> section's sixteen leaves (SAV-PROJSTORE-428).
type Projectile struct {
	ID uint16

	X, Y, Z        int32
	Picture        int32
	Dir            int32
	Phase          int32
	LastAction     int32
	Action         int32
	ActionDir      int32
	ActionTarget   int32
	ActionX        int32
	ActionY        int32
	ActionZ        int32
	ActionPhase    int32
	ActionSegments int32
	ActionSpell    int32
}

// ProjectileStore is one world save's whole Projectiles subtree.
//
// IDs keeps the wire array's order and names each ID once: a repeated ID
// replaces the earlier record in the original's one record map, which SAVE
// walks (SAV-1322). Items holds one entry per ID, in that order.
type ProjectileStore struct {
	FreeIndex uint16
	IDs       []uint16
	Items     []Projectile
}

// Projectiles reads the subtree as SAV-PROJLOAD-429's loader does: FreeIndex
// and IDs default independently, IDs may be kind 6 or a kind-2 singleton, and
// a store with neither leaf reports false. A missing Prj leaf reads as zero;
// the original's constructed default is not established.
func (f *File) Projectiles() (ProjectileStore, bool, error) {
	r := f.StateLeaves()
	free, hasFree, err := r.ReadLeaf("/Projectiles/FreeIndex")
	if err != nil {
		return ProjectileStore{}, false, err
	}
	ids, hasIDs, err := r.ReadLeaf("/Projectiles/IDs")
	if err != nil {
		return ProjectileStore{}, false, err
	}
	hasFree = hasFree && !free.Dir && free.Type == reg.TypeInt
	hasIDs = hasIDs && !ids.Dir && (ids.Type == reg.TypeInt || ids.Type == reg.TypeIntArray)
	if !hasFree && !hasIDs {
		return ProjectileStore{}, false, nil
	}
	store := ProjectileStore{}
	if hasFree {
		store.FreeIndex = uint16(free.Int)
	}
	if hasIDs {
		array := []int32{ids.Int}
		if ids.Type == reg.TypeIntArray {
			array = ids.Ints
		}
		seen := make(map[uint16]bool, len(array))
		store.IDs = make([]uint16, 0, len(array))
		for _, v := range array {
			if !seen[uint16(v)] {
				seen[uint16(v)] = true
				store.IDs = append(store.IDs, uint16(v))
			}
		}
	}
	for _, id := range store.IDs {
		section := "Prj" + strconv.FormatUint(uint64(id), 10)
		var leafErr error
		leaf := func(name string) int32 {
			v, present, err := r.ReadLeaf("/" + section + "/" + name)
			if err != nil && leafErr == nil {
				leafErr = err
			}
			if !present || v.Dir || v.Type != reg.TypeInt {
				return 0
			}
			return narrowProjectileLeaf(name, v.Int)
		}
		store.Items = append(store.Items, Projectile{
			ID: id,
			X:  leaf("x"), Y: leaf("y"), Z: leaf("z"),
			Picture: leaf("picture"), Dir: leaf("dir"), Phase: leaf("phase"),
			LastAction: leaf("lastaction"), Action: leaf("action"),
			ActionDir: leaf("actiondir"), ActionTarget: leaf("actiontarget"),
			ActionX: leaf("actionx"), ActionY: leaf("actiony"), ActionZ: leaf("actionz"),
			ActionPhase: leaf("actionphase"), ActionSegments: leaf("actionsegments"),
			ActionSpell: leaf("actionspell"),
		})
		if leafErr != nil {
			return ProjectileStore{}, false, leafErr
		}
	}
	return store, true, nil
}

// SetProjectiles replaces the subtree with the canonical producer shape
// (SAV-PROJSTORE-428): FreeIndex, IDs kind 6 and one whole Prj<id> section per
// ID. A between-mission save has no world half and refuses. A directory kind
// word already in the file is kept; a created directory gets kind 1.
func (f *File) SetProjectiles(store ProjectileStore) error {
	if f.World == nil {
		return fmt.Errorf("sav: SetProjectiles requires a world-half save")
	}
	state, err := parseWorldState(f.Store)
	if err != nil {
		return fmt.Errorf("sav: SetProjectiles: %w", err)
	}
	old, _, err := f.Projectiles()
	if err != nil {
		return fmt.Errorf("sav: SetProjectiles: %w", err)
	}
	newIDs := make(map[uint16]bool, len(store.Items))
	for _, p := range store.Items {
		newIDs[p.ID] = true
	}
	removed := make(map[uint16]bool, len(old.IDs))
	for _, id := range old.IDs {
		if removed[id] || newIDs[id] {
			continue
		}
		removed[id] = true
		dirPath := "/Prj" + strconv.FormatUint(uint64(id), 10)
		delete(state.directoryKinds, dirPath)
		for _, name := range projectileLeafNames {
			delete(state.values, dirPath+"/"+name)
		}
	}
	const projDir = "/" + projectilesSection
	if _, ok := state.directoryKinds[projDir]; !ok {
		state.directoryKinds[projDir] = 1
	}
	state.values[projDir+"/"+projectilesFreeIndex] = cityStateValue{kind: 2, int32: int32(store.FreeIndex)}
	ids := make([]byte, 4*len(store.IDs))
	for i, id := range store.IDs {
		binary.LittleEndian.PutUint32(ids[4*i:], uint32(id))
	}
	state.values[projDir+"/"+projectilesIDs] = cityStateValue{kind: 6, bytes: ids}
	for _, p := range store.Items {
		dirPath := "/Prj" + strconv.FormatUint(uint64(p.ID), 10)
		if _, ok := state.directoryKinds[dirPath]; !ok {
			state.directoryKinds[dirPath] = 1
		}
		set := func(name string, v int32) { state.values[dirPath+"/"+name] = cityStateValue{kind: 2, int32: v} }
		set("x", p.X)
		set("y", p.Y)
		set("z", p.Z)
		set("picture", p.Picture)
		set("dir", p.Dir)
		set("phase", p.Phase)
		set("lastaction", p.LastAction)
		set("action", p.Action)
		set("actiondir", p.ActionDir)
		set("actiontarget", p.ActionTarget)
		set("actionx", p.ActionX)
		set("actiony", p.ActionY)
		set("actionz", p.ActionZ)
		set("actionphase", p.ActionPhase)
		set("actionsegments", p.ActionSegments)
		set("actionspell", p.ActionSpell)
	}
	encoded, err := serializeWorldState(state)
	if err != nil {
		return fmt.Errorf("sav: SetProjectiles: %w", err)
	}
	f.splitTail(append(encoded, f.TailRest...))
	return nil
}
