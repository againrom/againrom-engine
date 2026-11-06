package mapedit

import (
	"cmp"
	"slices"
	"sort"

	"againrom/pkg/formats/alm"
)

type recordIdentity struct {
	key  uint64
	cell bool
}

// RecordKey is the record's session identity in the byte edit history.
func (e *Editor) RecordKey(k Kind, index int) uint64 {
	w, err := e.walkIndex(k, "identify", index)
	if err != nil {
		return 0
	}
	at, _ := w.at(index)
	return e.recordKey(at, false)
}

// SceneryKey names the object at a row-major overlay cell until it is cleared.
func (e *Editor) SceneryKey(index int) uint64 {
	w, h := e.GridSize()
	s, err := e.locate().optional("identify scenery", alm.TypeOverlay)
	if err != nil || index < 0 || index >= w*h || e.data[s.abs(index)] == 0 {
		return 0
	}
	return e.recordKey(s.abs(index), true)
}

// SectionKey names a physical section, including overridden and unknown ones.
func (e *Editor) SectionKey(index int) uint64 {
	f := e.locate()
	if index < 0 || index >= len(f.physical) {
		return 0
	}
	return e.recordKey(f.physical[index].headerOff, false)
}

func (e *Editor) recordKey(at int, cell bool) uint64 {
	if identity := e.keysAt[e.cursor][at]; identity.key != 0 {
		return identity.key
	}
	*e.nextKey++
	e.keysAt[e.cursor][at] = recordIdentity{key: *e.nextKey, cell: cell}
	return *e.nextKey
}

func (e *Editor) shiftRecordKeys(ed edit) map[int]recordIdentity {
	ordered := slices.Clone(ed)
	slices.SortFunc(ordered, func(a, b part) int { return cmp.Compare(a.off, b.off) })
	delta := make([]int, len(ordered)+1)
	for i, p := range ordered {
		delta[i+1] = delta[i] + len(p.inserted) - len(p.removed)
	}
	keys := make(map[int]recordIdentity, len(e.keysAt[e.cursor]))
	for at, identity := range e.keysAt[e.cursor] {
		i := sort.Search(len(ordered), func(i int) bool { return ordered[i].off > at }) - 1
		if i < 0 {
			keys[at] = identity
			continue
		}
		p := ordered[i]
		if at >= p.off+len(p.removed) {
			at += delta[i+1]
		} else {
			if len(p.inserted) == 0 {
				continue
			}
			if p.relocated != nil {
				at = p.relocated[at]
			}
			at += delta[i]
		}
		keys[at] = identity
	}
	return keys
}
