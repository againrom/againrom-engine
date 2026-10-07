package sim

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"maps"
	"reflect"
	"testing"
)

// The public import fixture starts with no object authority. Reusing the cell
// fixture does not supply any of the acceptance/refusal expectations below.
func unboundGoldWorld1115(t *testing.T) (*World, *SavedObjects, []SavedSackBinding) {
	t.Helper()
	w := goldPickupSavedWorld(t, 31)
	r := w.SavedObjects()
	w.savedObjects = nil
	w.sacks[0].ObjectID = 0
	return w, r, []SavedSackBinding{{ID: 0x100000002, X: 10, Y: 20, Gold: 31}}
}

func TestSavedObjectsWorld1115ImportRefusalsAreAtomic(t *testing.T) {
	type request struct {
		w *World
		r *SavedObjects
		b []SavedSackBinding
	}
	for _, tc := range []struct {
		name string
		edit func(*request)
	}{
		{"nil-registry", func(q *request) { q.r = nil }},
		{"duplicate-binding-handle", func(q *request) { q.b = append(q.b, q.b[0]) }},
		{"zero-binding-handle", func(q *request) { q.b[0].ID = 0 }},
		{"unknown-binding-handle", func(q *request) { q.b[0].ID = 0x100000003 }},
		{"same-low-word-wrong-high-word", func(q *request) { q.b[0].ID = 2 }},
		{"empty-source-identity", func(q *request) { q.r.Sacks[0].Token.Identity = 0 }},
		{"unknown-source-identity", func(q *request) { q.r.Sacks[0].Token.Identity = 0x11223344 }},
		{"wrong-token-cell", func(q *request) { q.r.Sacks[0].Token.Position[2] = 11 }},
		{"wrong-binding-cell", func(q *request) { q.b[0].X = 11 }},
		{"negative-x", func(q *request) { q.b[0].X = -1 }},
		{"x-wrap", func(q *request) { q.b[0].X = 266 }},
		{"negative-y", func(q *request) { q.b[0].Y = -1 }},
		{"y-wrap", func(q *request) { q.b[0].Y = 276 }},
		{"registry-gold-mismatch", func(q *request) { q.r.Sacks[0].Gold = 32 }},
		{"binding-gold-mismatch", func(q *request) { q.b[0].Gold = 32 }},
		{"native-gold-mismatch", func(q *request) { q.w.sacks[0].Gold = 32 }},
		{"missing-native-sack", func(q *request) { q.w.sacks = nil }},
		{"missing-binding", func(q *request) { q.b = nil }},
		{"missing-cell-node", func(q *request) {
			q.w.savedMotion.Cells, q.w.savedStructureCells = nil, nil
		}},
		{"zero-registry-id", func(q *request) { q.r.Sacks[0].ID = 0 }},
		{"duplicate-registry-id", func(q *request) { q.r.Sacks = append(q.r.Sacks, q.r.Sacks[0]) }},
		{"id-at-allocation-ceiling", func(q *request) { q.r.NextID = 0x100000002 }},
		{"wrong-registry-version", func(q *request) { q.r.Version = 3 }},
		{"missing-root", func(q *request) { q.r.SackRoots = nil }},
		{"unknown-root", func(q *request) { q.r.SackRoots[1] = 0x100000003 }},
		{"missing-container", func(q *request) { q.r.Containers = nil }},
		{"retired-source-sack", func(q *request) {
			q.r.Sacks[0].Retired, q.r.Sacks[0].Gold, q.b[0].Gold = true, 0, 0
			q.r.SackRoots, q.r.Containers = nil, nil
			q.w.sacks[0].Gold = 0
		}},
		{"native-item-needs-own-producer", func(q *request) {
			q.w.sacks[0] = makeSack(10, 20, 31, []ItemInstance{PlainItem(0x3185)})
		}},
		{"source-item-needs-own-producer", func(q *request) { q.r.Items = []SavedItemObject{{ID: 4}} }},
		{"source-effect-needs-own-producer", func(q *request) { q.r.Effects = []SavedEffectObject{{ID: 4}} }},
		{"source-spell-needs-own-producer", func(q *request) { q.r.Spells = []SavedSpellObject{{ID: 4}} }},
		{"two-identities-one-native-sack", func(q *request) {
			row := q.r.Sacks[0]
			row.ID = 0x100000003
			q.r.NextID = 0x100000004
			q.r.Sacks = append(q.r.Sacks, row)
			q.r.SackRoots = append(q.r.SackRoots, row.ID)
			q.r.Containers = append(q.r.Containers, SavedObjectContainer{Owner: SavedObjectOwner{Kind: SavedOwnerSack, Object: row.ID}, Present: true})
			q.b = append(q.b, SavedSackBinding{ID: row.ID, X: 10, Y: 20, Gold: 31})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, r, b := unboundGoldWorld1115(t)
			q := request{w, r, b}
			tc.edit(&q)
			before, hash, source := mustMarshal(t, q.w), q.w.Hash(), q.r.Clone()
			if err := q.w.ImportSavedObjects(q.r, q.b); err == nil {
				t.Fatal("unsupported import succeeded")
			}
			if !bytes.Equal(before, mustMarshal(t, q.w)) || hash != q.w.Hash() || q.w.savedObjects != nil || !reflect.DeepEqual(source, q.r) {
				t.Fatal("rejected import mutated the world or caller registry")
			}
			// The first candidate binding may already have been written when a
			// later check refuses. Neither it nor caller-owned arrays may leak.
			if q.r != nil {
				q.r.Sacks[0].Gold ^= 0xffffffff
				q.r.Sacks[0].Token.Position[4]++
				if len(q.r.SackRoots) != 0 {
					q.r.SackRoots[0]++
				}
				if len(q.r.Containers) != 0 {
					q.r.Containers[0].Accumulator++
				}
			}
			if !bytes.Equal(before, mustMarshal(t, q.w)) || hash != q.w.Hash() {
				t.Fatal("rejected import retained a mutable caller field")
			}
		})
	}
	var absent *World
	if err := absent.ImportSavedObjects(&SavedObjects{Version: 1, NextID: 1}, nil); err == nil {
		t.Fatal("nil World admitted an import")
	}
}

func TestSavedSackImportUsesExactCellOperandWithoutVisibilityBit(t *testing.T) {
	w, r, bindings := unboundGoldWorld1115(t)
	w.savedCellPlanes.Static[0x140a] = 0x12
	w.savedMotion.Blocks[0].Static = 0x12
	planes, motion := *w.savedCellPlanes, cloneActorMotions(w.savedMotion)
	if w.SavedSackCellKey(0x140a) != 0 {
		t.Fatal("fixture unexpectedly exposes a gameplay lookup key")
	}
	if err := w.ImportSavedObjects(r, bindings); err != nil {
		t.Fatal(err)
	}
	if w.sacks[0].ObjectID != bindings[0].ID || !reflect.DeepEqual(w.SavedObjects(), r) || *w.savedCellPlanes != planes || !reflect.DeepEqual(w.savedMotion, motion) || w.SavedSackCellKey(0x140a) != 0 {
		t.Fatal("exact collection binding changed plane bits or lookup visibility")
	}
}

func TestSavedSackImportKeepsAbsentPlaneAuthority(t *testing.T) {
	w, r, bindings := unboundGoldWorld1115(t)
	w.savedCellPlanes = nil
	motion := cloneActorMotions(w.savedMotion)
	if err := w.ImportSavedObjects(r, bindings); err != nil {
		t.Fatal(err)
	}
	if w.sacks[0].ObjectID != bindings[0].ID || !reflect.DeepEqual(w.SavedObjects(), r) || w.savedCellPlanes != nil || !reflect.DeepEqual(w.savedMotion, motion) || w.SavedSackCellKey(0x140a) != 0 {
		t.Fatal("exact collection binding fabricated plane authority or lookup visibility")
	}
}

func TestSavedObjectsWorld1115ExactHandlesAliasesAndDetachedInput(t *testing.T) {
	for _, id := range []SavedObjectID{1, 0x100000002, 0xfedcba9876543210} {
		t.Run(fmt.Sprintf("%016x", id), func(t *testing.T) {
			w, r, b := unboundGoldWorld1115(t)
			r.Sacks[0].ID, r.NextID, b[0].ID = id, id+1, id
			r.SackRoots = []SavedObjectID{id, id, id}
			r.Containers[0].Owner.Object = id
			want := r.Clone()
			if err := w.ImportSavedObjects(r, b); err != nil {
				t.Fatal(err)
			}
			if len(w.sacks) != 1 || w.sacks[0].ObjectID != id || !reflect.DeepEqual(w.SavedObjects(), want) || w.SavedSackCellKey(0x140a) != 0x78563412 {
				t.Fatal("native handle, root aliases or opaque cell key were folded")
			}
			before, hash := mustMarshal(t, w), w.Hash()
			r.Sacks[0].Gold, r.Sacks[0].Token.Position[4] = 0, 3
			r.SackRoots[0] = 0
			r.Containers[0].Accumulator = 99
			b[0] = SavedSackBinding{}
			getter := w.SavedObjects()
			getter.Sacks[0].Token.Identity = 0
			getter.SackRoots[1] = 0
			getter.Containers[0].InsertIndex = 0
			if !bytes.Equal(before, mustMarshal(t, w)) || hash != w.Hash() || !reflect.DeepEqual(w.SavedObjects(), want) {
				t.Fatal("successful import/getter shared mutable registry storage")
			}
			if err := w.ImportSavedObjects(want, []SavedSackBinding{{ID: id, X: 10, Y: 20, Gold: 31}}); err == nil || !bytes.Equal(before, mustMarshal(t, w)) {
				t.Fatal("second import replaced existing authority")
			}
			cold := copySackCellSavedWorld(t, w)
			if !bytes.Equal(before, mustMarshal(t, cold)) || cold.Hash() != hash || cold.sacks[0].ObjectID != id || !reflect.DeepEqual(cold.SavedObjects(), want) {
				t.Fatal("native reload lost exact high bits or root aliases")
			}
		})
	}
}

func TestSavedObjectsWorld1115EmptyPresentIsNotAbsent(t *testing.T) {
	w, _, _ := unboundGoldWorld1115(t)
	absent, absentHash := mustMarshal(t, w), w.Hash()
	if w.SavedObjects() != nil || binary.LittleEndian.Uint32(absent[len(absent)-entityIDFloorLen-spellDeliverySpanLen-36:len(absent)-entityIDFloorLen-spellDeliverySpanLen-32]) != 0 {
		t.Fatal("unbound native Sack fabricated an object registry")
	}
	r := &SavedObjects{Version: SavedObjectsVersion, NextID: 1}
	if err := w.ImportSavedObjects(r, nil); err != nil {
		t.Fatal(err)
	}
	present := mustMarshal(t, w)
	if bytes.Equal(present, absent) || w.Hash() == absentHash || binary.LittleEndian.Uint32(present[len(present)-entityIDFloorLen-spellDeliverySpanLen-36:len(present)-entityIDFloorLen-spellDeliverySpanLen-32]) == 0 || !reflect.DeepEqual(w.SavedObjects(), r) || w.sacks[0].ObjectID != 0 {
		t.Fatal("empty authority collapsed into absence or adopted an ID-zero Sack")
	}
	var cold World
	if err := cold.UnmarshalBinary(present); err != nil || cold.SavedObjects() == nil || !bytes.Equal(present, mustMarshal(t, &cold)) {
		t.Fatalf("empty-present native reload: %v", err)
	}
}

func boundPackWorld1115(t *testing.T) *World {
	t.Helper()
	w := goldPickupSavedWorld(t, 31)
	owner := SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 7}
	value := ItemStack{ObjectID: 19, Code: 0x3185, Count: 1}
	w.savedObjects.Items = []SavedItemObject{{ID: 19, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Value: value}}
	w.savedObjects.Containers = append(w.savedObjects.Containers, SavedObjectContainer{Owner: owner, Present: true, Items: []SavedObjectID{19}})
	w.carried[0] = []ItemStack{value}
	_ = mustMarshal(t, w)
	return w
}

func TestSavedObjectsWorld1115NativeValidationRefusesContradictions(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*World)
	}{
		{"sack-without-registry", func(w *World) { w.savedObjects = nil; w.carried[0] = nil }},
		{"unknown-live-sack", func(w *World) { w.sacks[0].ObjectID = 0x100000003 }},
		{"duplicate-live-sack-handle", func(w *World) {
			s := w.sacks[0]
			s.X = 11
			w.sacks = append(w.sacks, s)
		}},
		{"live-registry-missing-native-sack", func(w *World) { w.sacks = nil }},
		{"live-sack-hidden-as-id-zero", func(w *World) { w.sacks[0].ObjectID = 0 }},
		{"retired-sack-still-live", func(w *World) {
			w.sacks[0].Gold = 0
			w.savedObjects.Sacks[0].Gold, w.savedObjects.Sacks[0].Retired = 0, true
			w.savedObjects.SackRoots = nil
			w.savedObjects.Containers = w.savedObjects.Containers[1:]
		}},
		{"native-gold-disagrees", func(w *World) { w.sacks[0].Gold = 32 }},
		{"native-position-disagrees", func(w *World) { w.sacks[0].Y = 21 }},
		{"unknown-live-pack-item", func(w *World) { w.carried[0][0].ObjectID = 20 }},
		{"hidden-live-empty-worn-slot", func(w *World) { w.equipment[0][4].ObjectID = 22 }},
		{"unknown-live-worn-item", func(w *World) { w.equipment[0][4] = ItemInstance{ObjectID: 22, Code: 0x3185} }},
		{"unknown-live-sack-item", func(w *World) {
			w.sacks[0].ItemInstances = []ItemInstance{{ObjectID: 22, Code: 0x3185}}
			w.sacks[0].Items = []uint16{0x3185}
		}},
		{"unknown-detached-scroll", func(w *World) { w.scrollCasts = []ScrollCast{{Item: ItemInstance{ObjectID: 22, Code: 0x3185}}} }},
		{"live-item-with-no-owner", func(w *World) { w.savedObjects.Containers[1].Items = nil }},
		{"live-item-with-no-container-edge", func(w *World) { w.savedObjects.Containers[1].Items = nil }},
		{"live-item-wrong-owner", func(w *World) {
			owner := SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 8}
			w.savedObjects.Items[0].Owner, w.savedObjects.Containers[1].Owner = owner, owner
		}},
		{"registry-item-missing-live-value", func(w *World) { w.carried[0] = nil }},
		{"retired-item-still-live", func(w *World) {
			w.savedObjects.Items[0].Retired = true
			w.savedObjects.Containers[1].Items = nil
		}},
		{"inflight-item-at-save-boundary", func(w *World) {
			w.savedObjects.Items[0].InFlight = 1
			w.savedObjects.Containers[1].Items = nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := boundPackWorld1115(t)
			tc.edit(w)
			before, hash := w.encode(), w.Hash()
			if got, err := w.MarshalBinary(); err == nil || got != nil {
				t.Fatal("contradictory native state was serializable")
			}
			if !bytes.Equal(before, w.encode()) || hash != w.Hash() {
				t.Fatal("Marshal repaired or otherwise mutated a contradictory state")
			}
		})
	}
}

func sourceCopyWorld(t *testing.T) *World {
	t.Helper()
	w := goldPickupSavedWorld(t, 31)
	w.pourSack(10, 20, 0, []ItemInstance{{Code: 0x3185, Effects: []ItemEffect{{Kind: 8, Mode: 1, Operand: 17}}}})
	neighbor := SavedActorCell{Cell: 0x140b, Payload: [52]byte{0: 19, 1: 1, 12: 0x91}}
	w.savedMotion.Cells = append(w.savedMotion.Cells, neighbor)
	source := SavedStructure{ID: 9, Class: SavedBuilding, SourceKey: 0x91, ArchiveIndex: 9, Blocking: 1,
		Position: [12]byte{11, 20}, Base52: [22]byte{14: 1, 15: 1, 18: 1}}
	if err := w.ImportOriginalStructures([]Structure{{ID: 9, Col: 11, Row: 20, Width: 1, Height: 1, Blocking: 1}}, []SavedStructure{source},
		[]SavedStructureCell{{Cell: 0x140a, BaselineCost: 11, BaselineStatic: 2}, {Cell: 0x140b, BaselineCost: 19, BaselineStatic: 1, ID: 9, HasStructure: true}}, bytes.Clone(w.grid)); err != nil {
		t.Fatal(err)
	}
	w.savedCellPlanes.Cost[0x140b], w.savedCellPlanes.CostKnown[0x140b] = 19, 1
	w.savedCellPlanes.Static[0x140b], w.savedCellPlanes.Dynamic[0x140b] = 0x21, 0x21
	w.syncNativeSavedPlaneGrid()
	w.refreshSavedPlaneBlocks()
	w.writeCellTail(0x140a, [6]byte{0, 31, 7, 8, 6, 5})
	w.writeCellTail(0x140b, [6]byte{0, 23, 15, 17, 19, 21})
	_ = mustMarshal(t, w)
	return w
}

func TestSavedObjectsWorld1115SourceMutationCopyOwnsTouchedStorage(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*World)
	}{
		{"sack-scalar", func(w *World) { w.sacks[0].Gold++ }},
		{"sack-native-handle", func(w *World) { w.sacks[0].ObjectID++ }},
		{"sack-code-projection", func(w *World) { w.sacks[0].Items[0]++ }},
		{"sack-instance", func(w *World) { w.sacks[0].ItemInstances[0].Price++ }},
		{"sack-effect", func(w *World) { w.sacks[0].ItemInstances[0].Effects[0].Operand++ }},
		{"registry-sack", func(w *World) { w.savedObjects.Sacks[0].Gold++ }},
		{"registry-root", func(w *World) { w.savedObjects.SackRoots[0]++ }},
		{"registry-container", func(w *World) { w.savedObjects.Containers[0].Accumulator++ }},
		{"cell-payload", func(w *World) { w.savedMotion.Cells[0].Payload[16]++ }},
		{"current-block", func(w *World) { w.savedMotion.Blocks[0].Dyn++ }},
		{"saved-plane-cost", func(w *World) { w.savedCellPlanes.Cost[0x140a]++ }},
		{"saved-plane-cost-known", func(w *World) { w.savedCellPlanes.CostKnown[0x140a] = 0 }},
		{"saved-plane-static", func(w *World) { w.savedCellPlanes.Static[0x140a]++ }},
		{"saved-plane-dynamic", func(w *World) { w.savedCellPlanes.Dynamic[0x140a]++ }},
		{"saved-plane-height", func(w *World) { w.savedCellPlanes.Height[0x140a]++ }},
		{"saved-plane-cost-table", func(w *World) { w.savedCellPlanes.Costs[5]++ }},
		{"native-grid", func(w *World) { w.grid[20*32+10]++ }},
		{"native-cost", func(w *World) { w.cost[20*32+10]++ }},
		{"cell-tail", func(w *World) { w.cellTails[0].Bytes[1]++ }},
		{"saved-structure-cell", func(w *World) { w.savedStructureCells[1].BaselineCost++ }},
		{"structure-cache", func(w *World) { delete(w.structureSlots, 0x140b) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := sourceCopyWorld(t)
			before, hash, slots := mustMarshal(t, w), w.Hash(), maps.Clone(w.structureSlots)
			if len(slots) != 1 || slots[0x140b] != 0 {
				t.Fatal("fixture lacks an independently mutable structure cache")
			}
			copy := w.sourceMutationCopy(0)
			if !bytes.Equal(before, mustMarshal(t, &copy)) || copy.Hash() != hash || !reflect.DeepEqual(copy.structureSlots, slots) {
				t.Fatal("copy changed state before any producer mutation")
			}
			tc.edit(&copy)
			if !bytes.Equal(before, mustMarshal(t, w)) || w.Hash() != hash || !reflect.DeepEqual(w.structureSlots, slots) {
				t.Fatal("candidate mutation escaped into live storage")
			}
		})
	}
}
