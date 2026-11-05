package ui

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"slices"
)

type barRecordKeys struct {
	next    uint64
	current map[RecordRef]uint64
	states  []barRecordVersion
	cursor  int
	history int
}

type barRecordVersion struct {
	state [32]byte
	keys  map[RecordRef]uint64
}

func barRecordState(d *InspectionDocument) [32]byte {
	h := sha256.New()
	h.Write(d.SourceBytes)
	for _, r := range d.Records {
		fmt.Fprintf(h, "%q:%d:%q:%d:%d:%d;", r.Kind, r.Index, r.Label, r.ScriptID, r.Cell.X, r.Cell.Y)
	}
	var sum [32]byte
	copy(sum[:], h.Sum(nil))
	return sum
}

func (e *MapEditor) barRecordKey(ref RecordRef) uint64 {
	if e.doc == nil {
		return 0
	}
	k := &e.barRecords
	if k.current == nil {
		k.current = map[RecordRef]uint64{}
		for _, r := range e.doc.Records {
			k.next++
			k.current[RecordRef{Kind: r.Kind, Index: r.Index}] = k.next
		}
	}
	if ref.Kind == "" || ref.Kind == KindVariable {
		return 0
	}
	if _, ok := k.current[ref]; !ok && e.linkedRecord(ref) >= 0 {
		k.next++
		k.current[ref] = k.next
	}
	return k.current[ref]
}

func (e *MapEditor) carryBarRecordKeys(next *InspectionDocument, removed []RecordRef) {
	k := &e.barRecords
	if k.current == nil {
		return
	}
	previous := barRecordVersion{state: barRecordState(e.doc), keys: maps.Clone(k.current)}
	if len(k.states) == 0 {
		k.states = append(k.states, previous)
	} else {
		k.states[k.cursor] = previous
	}
	state := barRecordState(next)
	if k.history != 0 {
		for at := k.cursor + k.history; at >= 0 && at < len(k.states); at += k.history {
			if version := k.states[at]; version.state == state {
				k.current, k.cursor = maps.Clone(version.keys), at
				return
			}
		}
	}
	if slices.ContainsFunc(removed, func(ref RecordRef) bool { return ref.Kind == KindUnit }) {
		removed = slices.Clone(removed)
		for _, was := range e.doc.Records {
			ref := RecordRef{Kind: was.Kind, Index: was.Index}
			if was.Kind == KindStock && !slices.Contains(removed, ref) && !slices.ContainsFunc(next.Records, func(r InspectionRecord) bool { return r.Kind == KindStock && r.Label == was.Label }) {
				removed = append(removed, ref)
			}
		}
	}
	shifted := map[RecordRef]uint64{}
	for ref, key := range k.current {
		if slices.Contains(removed, ref) {
			continue
		}
		at := ref
		for _, gone := range removed {
			loot := (gone.Kind == KindSack || gone.Kind == KindStock) && (ref.Kind == KindSack || ref.Kind == KindStock)
			if (gone.Kind == ref.Kind || loot) && gone.Index < ref.Index && ref.Kind != KindEnchantment {
				at.Index--
			}
		}
		shifted[at] = key
	}
	k.current = map[RecordRef]uint64{}
	for _, r := range next.Records {
		ref := RecordRef{Kind: r.Kind, Index: r.Index}
		key := shifted[ref]
		if key == 0 {
			k.next++
			key = k.next
		}
		k.current[ref] = key
	}
	k.states = append(k.states[:k.cursor+1], barRecordVersion{state: state, keys: maps.Clone(k.current)})
	k.cursor++
}
