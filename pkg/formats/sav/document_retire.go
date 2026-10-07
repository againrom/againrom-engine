package sav

import (
	"fmt"
	"reflect"
)

// RetireDocumentData removes exactly the named local objects from a deliberately
// edited producer graph. The caller must first remove every surviving root or
// reference to them. References inside records retired together may remain.
// Unlisted unreachable objects are errors, not implicit garbage collection.
//
// The result independently owns a canonical document and an old-local-index to
// new-local-index permutation. Only zero and explicitly retired indices map to
// zero. The producer must remap every external binding in the same transaction.
// Numeric Token keys and cell payload keys are not archive references and are
// never rewritten. Native LOAD, Clone and Encode remain strict and unchanged.
// On any failure the input is untouched and both returned values are empty.
func RetireDocumentData(data DocumentData, retired []uint16) (DocumentData, []uint16, error) {
	if len(data.Objects) > maxDocumentDataObjects || len(retired) > len(data.Objects) {
		return DocumentData{}, nil, fmt.Errorf("sav: document retirement count exceeds bound")
	}
	removed := make([]bool, len(data.Objects)+1)
	for _, id := range retired {
		if id == 0 || int(id) > len(data.Objects) || removed[id] {
			return DocumentData{}, nil, fmt.Errorf("sav: document retirement has zero, unknown or duplicate index %d", id)
		}
		removed[id] = true
	}
	// Check the entire supplied ownership budget and every record, including
	// retired records, before copying. Retirement cannot hide malformed state.
	if err := (&documentDataBudget{}).check(reflect.ValueOf(data), 0); err != nil {
		return DocumentData{}, nil, err
	}
	objects := make([]*Record, len(data.Objects))
	for i, r := range data.Objects {
		objects[i] = newRecord(r.Class, 0, 0)
	}
	for i, r := range data.Objects {
		if err := documentRecordFromData(r, objects[i], objects, false, 0); err != nil {
			return DocumentData{}, nil, fmt.Errorf("sav: retirement input object %d: %w", i+1, err)
		}
	}
	// Preserve actual root encounter order before visiting disconnected records.
	// A noncanonical Objects order is not an archive traversal: starting there
	// could falsely reject a valid depth boundary whose aliases were seen first.
	rootLists := [][]uint16{data.Players, data.DeadActors}
	if w := data.World; w != nil {
		rootLists = append(rootLists, w.Buildings, w.Effects, w.Sacks)
	}
	var forest []*Record
	for _, ids := range rootLists {
		if len(ids) > maxListElements {
			return DocumentData{}, nil, fmt.Errorf("sav: document retirement root count exceeds bound")
		}
		for _, id := range ids {
			if int(id) > len(objects) {
				return DocumentData{}, nil, fmt.Errorf("sav: document retirement root %d is outside object table", id)
			}
			if id != 0 {
				forest = append(forest, objects[id-1])
			}
		}
	}
	// This validation-only forest also checks wire depth, shared tag namespace
	// and decoded-record byte limits for records the output no longer reaches.
	// Actual root classes and the complete output byte size are checked below.
	forest = append(forest, objects...)
	if err := validateDocumentArchive(&archiveDocument{players: forest}); err != nil {
		return DocumentData{}, nil, err
	}
	compact := make([]uint16, len(data.Objects)+1)
	var count uint16
	for i := range data.Objects {
		if !removed[i+1] {
			count++
			compact[i+1] = count
		}
	}
	refs := func(ids []uint16) ([]uint16, error) {
		if len(ids) > maxListElements {
			return nil, fmt.Errorf("sav: document retirement reference count exceeds bound")
		}
		out := make([]uint16, len(ids))
		for i, id := range ids {
			if int(id) >= len(compact) || id != 0 && compact[id] == 0 {
				return nil, fmt.Errorf("sav: surviving document reference %d is unknown or retired", id)
			}
			out[i] = compact[id]
		}
		return out, nil
	}
	var record func(DocumentRecordData) (DocumentRecordData, error)
	record = func(src DocumentRecordData) (DocumentRecordData, error) {
		out := src
		out.RefSlots = make([]DocumentRefsData, len(src.RefSlots))
		for i, slot := range src.RefSlots {
			ids, err := refs(slot.Objects)
			if err != nil {
				return DocumentRecordData{}, err
			}
			out.RefSlots[i] = DocumentRefsData{Name: slot.Name, Objects: ids}
		}
		out.Inline = make([]DocumentInlineData, len(src.Inline))
		for i, inline := range src.Inline {
			r, err := record(inline.Record)
			if err != nil {
				return DocumentRecordData{}, err
			}
			out.Inline[i] = DocumentInlineData{Name: inline.Name, Record: r}
		}
		out.Groups = make([]DocumentRecordData, len(src.Groups))
		for i, group := range src.Groups {
			r, err := record(group)
			if err != nil {
				return DocumentRecordData{}, err
			}
			out.Groups[i] = r
		}
		return out, nil
	}
	next := data
	next.Objects = make([]DocumentRecordData, 0, int(count))
	for i, r := range data.Objects {
		if removed[i+1] {
			continue
		}
		out, err := record(r)
		if err != nil {
			return DocumentData{}, nil, err
		}
		next.Objects = append(next.Objects, out)
	}
	lists := []*[]uint16{&next.Players, &next.DeadActors}
	if data.World != nil {
		world := *data.World
		next.World = &world
		lists = append(lists, &world.Buildings, &world.Effects, &world.Sacks)
	}
	for _, list := range lists {
		out, err := refs(*list)
		if err != nil {
			return DocumentData{}, nil, err
		}
		*list = out
	}
	// Reuse the complete codec's version, root, shape, count, depth, state,
	// campaign and archive bounds. It rejects every remaining orphan and owns
	// all untouched slices too; no speculative edit aliases the caller's data.
	if err := remapNativeActionObjects(&next.State, compact); err != nil {
		return DocumentData{}, nil, err
	}
	out, canonical, err := ReindexDocumentData(next)
	if err != nil {
		return DocumentData{}, nil, err
	}
	for old, index := range compact {
		compact[old] = canonical[index]
	}
	return out, compact, nil
}
