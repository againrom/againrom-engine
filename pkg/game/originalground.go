package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// originalGroundState is detached from the archive and validated before a
// frontend reset. Only the authoritative top-level Sack population reaches it;
// corpse inventories and fresh ALM loot are not substitutes.
//
// An item's own unsupported Effect record
// (sav.Piece.UnsupportedEffectStates: a nonzero Token state, or a class
// other than "Effect") no longer refuses the Sack that holds it:
// originalItemInstance already carries only the supported Effects into live
// state, so the rest of the item — and every other item in the same Sack
// — restores regardless.
func originalGroundState(f *sav.File) ([]sim.Sack, []uint32, bool, int, error) {
	source, present, err := f.GroundSacks()
	if err != nil || !present {
		return nil, nil, present, 0, err
	}
	// The native world keeps one value per unit. Bound that expansion before
	// allocating: a tiny hostile file must not expand into billions of items.
	const maxGroundValues = 1 << 20
	var count uint64
	for _, saved := range source {
		for _, piece := range saved.Items {
			if piece.Code == 0 || piece.Stack == 0 {
				return nil, nil, true, 0, fmt.Errorf("original Sack %#x: zero item code or count", saved.Identity)
			}
			count += uint64(piece.Stack) * uint64(1+len(piece.Effects))
			if count > maxGroundValues {
				return nil, nil, true, 0, fmt.Errorf("original ground loot exceeds %d expanded item/effect values", maxGroundValues)
			}
		}
	}
	out := make([]sim.Sack, 0, len(source))
	ids := make([]uint32, 0, len(source))
	var unsupported int
	for _, saved := range source {
		sack := sim.Sack{X: int32(saved.Cell & 255), Y: int32(saved.Cell >> 8), Gold: saved.Gold}
		for _, piece := range saved.Items {
			unsupported += len(piece.UnsupportedEffectStates)
			item := originalItemInstance(piece, nil)
			// ITEM-STACK-003: +42 is the count, distinct from the +08
			// flags set by pickup. SAV-POSTLOAD-223 drains quantity one
			// repeatedly and deep-copies effects when a stack splits.
			for range int(piece.Stack) {
				sack.ItemInstances = append(sack.ItemInstances, item.Clone())
			}
		}
		out = append(out, sack)
		ids = append(ids, saved.RuntimeID)
	}
	return out, ids, true, unsupported, nil
}

func applyOriginalGround(ms *Mission, sacks []sim.Sack, ids []uint32, present bool, unsupported int, table *mapload.Table, r *OriginalSaveResume) error {
	if !present {
		return nil
	}
	if ms == nil || ms.World == nil {
		return fmt.Errorf("original ground loot: mission has no world")
	}
	for i := range sacks {
		for j := range sacks[i].ItemInstances {
			sacks[i].ItemInstances[j] = mapload.BindSourceItemDefinition(sacks[i].ItemInstances[j], table)
		}
	}
	if err := ms.World.ReplaceGroundSacks(sacks); err != nil {
		return fmt.Errorf("original ground loot: %w", err)
	}
	// Each Sack keeps the runtime id its first record in the cell carries; a
	// merged record's id stays marked by the LOAD replay (SAV-1371).
	placed := ms.World.Sacks()
	carried := make([]uint32, len(placed))
	for i, s := range placed {
		for j := range sacks {
			if sacks[j].X == s.X && sacks[j].Y == s.Y && j < len(ids) {
				carried[i] = ids[j]
				break
			}
		}
	}
	if err := ms.World.RestoreGroundSackRuntimeIDs(carried); err != nil {
		return fmt.Errorf("original ground loot: %w", err)
	}
	var codes []uint16
	for _, sack := range sacks {
		for _, item := range sack.ItemInstances {
			codes = append(codes, item.Code)
		}
	}
	mapload.DeclareCodeWeights(ms.World, table, codes)
	if r != nil {
		r.Sacks, r.GroundItems, r.GroundApplied = len(ms.World.Sacks()), len(codes), true
		r.UnsupportedItemEffects += unsupported
	}
	return nil
}
