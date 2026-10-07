package sav

import (
	"encoding/binary"
	"fmt"
	"strconv"
)

// The Projectiles state-store subtree (SAV-PROJSTORE-428, SAV-PROJLOAD-429):
// the manager's allocator word, the live id set in wire order, and one
// Prj<decimal u16 id> section per distinct id with its own sixteen kind-2
// leaves. world_state.go's worldProjectileLeaves already names the shape for
// parseWorldState/serializeWorldState (save_document.go's own codec); this
// file is the typed read/write projection File.Open's own Marshal round trip
// actually uses.
//
// No claim narrows any leaf's range: SAV-PROJSTORE-428 names what the
// producer writes and SAV-PROJLOAD-429 what the loader reads, and neither
// gives this project a flight model to simulate the leaves against. So this
// package carries the sixteen fields opaquely, unnarrowed int32 — the same
// choice Fog.FirstState already made for a leaf this package does not
// interpret either.

const projectilesSection = "Projectiles"
const (
	projectilesFreeIndex = "FreeIndex"
	projectilesIDs       = "IDs"
)

// projectileLeafNames is worldProjectileLeaves' own name order, spelled once
// more so the reader and writer below read as direct field-by-field
// mirrors of each other. projectile_test.go pins this list against
// worldProjectileLeaves directly, so the two can never silently drift apart.
var projectileLeafNames = []string{
	"x", "y", "z", "picture", "dir", "phase", "lastaction", "action",
	"actiondir", "actiontarget", "actionx", "actiony", "actionz",
	"actionphase", "actionsegments", "actionspell",
}

// Projectile is one Prj<id> section's sixteen leaves: a spawned bolt's
// position, facing, and sequencing/action state (SAV-PROJSTORE-428). This
// build has no live projectile registry (pkg/game/projectiles.go is a
// cosmetic renderer only), so every leaf stays an opaque carried value —
// never simulated, never given a narrower range than the wire int it is.
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
// IDs is the wire array's own order and multiplicity, exactly as read — the
// world-state writer never sorts, dedupes or narrows it (world_state.go).
// Items holds one entry per DISTINCT id named by IDs, first-occurrence
// order: YA1 has one Prj<id> section regardless of how many times its id
// repeats in IDs, so a repeat reads the same section again rather than
// producing a second Items entry.
type ProjectileStore struct {
	FreeIndex uint16
	IDs       []uint16
	Items     []Projectile
}

// Projectiles reads the Projectiles state-store subtree through the same
// loader-tolerant path SAV-PROJLOAD-429 describes: FreeIndex and IDs each
// default independently when absent, IDs accepts both its canonical kind-6
// array form and the kind-2 singleton compatibility arm, and this reports
// false — not an error — where the tail carries a store but neither leaf is
// present at all, which is a between-mission save's own shape.
//
// A present id whose Prj<id> section lacks one of the sixteen leaves is
// accepted rather than refused, defaulting the missing leaf to zero: the
// loader itself reads a missing leaf as "the constructed object's own
// field", which this package cannot reproduce byte for byte with no live
// object to default from, and SAV-PROJCORP-430's exhaustive corpus audit
// never exercises this arm — every observed Prj<id> section carries all
// sixteen leaves.
func (f *File) Projectiles() (ProjectileStore, bool, error) {
	r, ok := f.StateStore()
	if !ok {
		return ProjectileStore{}, false, nil
	}
	freeIndex, hasFreeIndex := r.GetInt(projectilesSection, projectilesFreeIndex)
	array, hasArray := r.GetIntArray(projectilesSection, projectilesIDs)
	var singleton int32
	hasSingleton := false
	if !hasArray {
		singleton, hasSingleton = r.GetInt(projectilesSection, projectilesIDs)
	}
	if !hasFreeIndex && !hasArray && !hasSingleton {
		return ProjectileStore{}, false, nil
	}
	store := ProjectileStore{}
	if hasFreeIndex {
		store.FreeIndex = uint16(uint32(freeIndex))
	}
	switch {
	case hasArray:
		store.IDs = make([]uint16, len(array))
		for i, v := range array {
			store.IDs[i] = uint16(uint32(v))
		}
	case hasSingleton:
		store.IDs = []uint16{uint16(uint32(singleton))}
	}
	seen := make(map[uint16]bool, len(store.IDs))
	for _, id := range store.IDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		section := "Prj" + strconv.FormatUint(uint64(id), 10)
		leaf := func(name string) int32 {
			v, _ := r.GetInt(section, name)
			return v
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
	}
	return store, true, nil
}

// SetProjectiles replaces the Projectiles state-store subtree with a
// canonical encoding of store: FreeIndex and IDs both present, IDs kind 6
// (SAV-PROJSTORE-428's own canonical producer shape, taken even where the
// loader would also accept the singleton compatibility arm), and one whole
// Prj<id> section per distinct id named by store.IDs.
//
// It targets a world-present save's own state store; a between-mission save
// (f.World == nil) has no Projectiles subtree to hold and refuses. It builds
// on parseWorldState/serializeWorldState — the shape-strict codec — because
// a producer always emits the complete canonical form, never the partial
// shapes the loader-tolerant reader above accepts.
//
// A Prj<id> section this write keeps (its id named by both the previous and
// the new store) has its sixteen leaf VALUES overwritten but its own
// DIRECTORY KIND WORD left exactly as parsed: leaf kinds are fully
// determined by worldProjectileLeaves and validateStateShape compares them
// by strict equality, but a directory's kind word may carry flag bits (e.g.
// 0x10) validateStateShape never inspects, and this write must not invent a
// value for one it did not read. Only a section actually created here (its
// id absent from the file before this call) gets a fresh kind word, and
// exactly the shape validateStateShape already accepts (1: a plain
// directory, none of its own checked bits set). The Projectiles directory
// itself follows the same rule.
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
