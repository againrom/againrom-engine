package sav

import (
	"fmt"
	"sort"
)

// SpellEffectArchiveLocation is the archive cursor immediately BEFORE the
// world's counted SpellEffect list. It contains only framing accumulated in
// earlier collections: no SpellEffect count, member, child or endpoint is read.
// This lets independent acceptance instruments read that list from Body while
// resolving prior-class tags and back-references without replaying actors.
type SpellEffectArchiveLocation struct {
	Off, NextIndex int
	Classes        map[uint16]string
	Objects        []SpellEffectPriorObject
}

// SpellEffectPriorObject locates an already introduced object. Fields and
// reference edges are deliberately absent; Off is its first body byte.
type SpellEffectPriorObject struct {
	Index uint16
	Off   int
	Class string
}

func (f *File) SpellEffectArchiveLocation() (SpellEffectArchiveLocation, bool, error) {
	if f == nil || f.Head.End < 0 || f.Head.End > len(f.Body) {
		return SpellEffectArchiveLocation{}, false, fmt.Errorf("sav: invalid SpellEffect archive prefix")
	}
	w := &walker{b: f.Body, p: f.Head.End, next: 1,
		classes: map[uint16]string{}, objects: map[uint16]*Record{}}
	if _, err := w.playerRecords(f.Head.PlayerCount); err != nil {
		return SpellEffectArchiveLocation{}, false, err
	}
	if _, err := w.groundCountedList("dead actors", "Unit"); err != nil {
		return SpellEffectArchiveLocation{}, false, err
	}
	if w.p >= len(w.b) {
		return SpellEffectArchiveLocation{}, false, fmt.Errorf("sav: missing world selector")
	}
	present := w.b[w.p] != 0
	w.p++
	if !present {
		return SpellEffectArchiveLocation{}, false, nil
	}
	if _, err := w.groundCountedList("Buildings", "Building"); err != nil {
		return SpellEffectArchiveLocation{}, true, err
	}
	loc := SpellEffectArchiveLocation{Off: w.p, NextIndex: int(w.next), Classes: w.classes}
	for id, r := range w.objects {
		loc.Objects = append(loc.Objects, SpellEffectPriorObject{Index: id, Off: r.Off, Class: r.Class})
	}
	sort.Slice(loc.Objects, func(i, j int) bool { return loc.Objects[i].Index < loc.Objects[j].Index })
	return loc, true, nil
}
