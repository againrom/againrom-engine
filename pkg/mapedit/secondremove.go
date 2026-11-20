package mapedit

import (
	"encoding/binary"
	"fmt"
	"maps"
	"slices"

	"againrom/pkg/formats/alm"
)

// secondEdit collects the parts of one second-game removal: byte cuts with
// the size words they move, and count words by their delta.
type secondEdit struct {
	e     *Editor
	parts edit
	sizes map[int]int // record header offset -> payload size delta
	words map[int]int // absolute offset of a count word -> delta
}

func newSecondEdit(e *Editor) *secondEdit {
	return &secondEdit{e: e, sizes: map[int]int{}, words: map[int]int{}}
}

func (b *secondEdit) cut(sp span, abs, n int) {
	b.parts = append(b.parts, replace(abs, b.e.data[abs:abs+n], nil))
	b.sizes[sp.headerOff] -= n
}

func (b *secondEdit) set(at int, v uint32) {
	var img [4]byte
	binary.LittleEndian.PutUint32(img[:], v)
	b.parts = append(b.parts, replace(at, b.e.data[at:at+4], img[:]))
}

func (b *secondEdit) finish() edit {
	for _, h := range slices.Sorted(maps.Keys(b.sizes)) {
		if d := b.sizes[h]; d != 0 {
			b.parts = append(b.parts, b.e.wordPart(h+alm.RecordPayloadSize, d))
		}
	}
	for _, at := range slices.Sorted(maps.Keys(b.words)) {
		if d := b.words[at]; d != 0 {
			b.parts = append(b.parts, b.e.wordPart(at, d))
		}
	}
	return b.parts
}

// secondRemoveParts removes record i of w's section with what goes with it
// (R2-ASSET-258). A unit takes every sack its id owns, the type-9 elements
// those items bind and its spellbook elements (mode 8, key = id). A structure
// takes its mode-4 type-9 elements and its type-11 entries. A ground sack
// takes the elements its items bind. Every later binding and sack index is
// renumbered; no other id changes and no script is touched.
func secondRemoveParts(e *Editor, w walk, i int) (edit, error) {
	f := w.frame
	b := newSecondEdit(e)
	at, l := w.at(i)

	var sacks walk
	hasSacks := f.carries(alm.TypeLoot)
	if hasSacks {
		var err error
		if sacks, err = e.walkSection(KindLoot, "remove with a sack"); err != nil {
			return nil, err
		}
	}
	msp, ms, hasMarkers, err := e.secondMarkers(f)
	if err != nil {
		return nil, err
	}
	goneSack, goneMarker := map[int]bool{}, map[int]bool{}
	switch w.sec.typeID {
	case alm.TypeUnits:
		b.cut(w.span, at, l)
		b.words[w.countAt]--
		id := e.u32(at+alm.SecondUnitTag) & 0xffff
		for j := 0; hasSacks && id != 0 && j < sacks.n(); j++ {
			if p, _ := sacks.at(j); e.u32(p+alm.LootOwner) == id {
				goneSack[j] = true
			}
		}
		for k, m := range ms {
			if m.mode == 8 && m.key == id {
				goneMarker[k] = true
			}
		}
	case alm.TypeObjects:
		b.cut(w.span, at, l)
		b.words[w.countAt]--
		id := uint32(binary.LittleEndian.Uint16(e.data[at+alm.ObjectIdentity:]))
		for k, m := range ms {
			if m.mode == 4 && m.key == id {
				goneMarker[k] = true
			}
		}
		if err := e.secondCutLists(f, b, id); err != nil {
			return nil, err
		}
	case alm.TypeLoot:
		if e.u32(at+alm.LootOwner) != 0 {
			return nil, fmt.Errorf("mapedit: sack %d belongs to a unit and goes with it", i)
		}
		goneSack[i] = true
	}
	// The items of every removed sack bind elements that go too.
	for j := range goneSack {
		p, n := sacks.at(j)
		for q := alm.LootHeadWide; q+alm.LootElementSize <= n; q += alm.LootElementSize {
			if v := int(e.u32(p + q + alm.LootElementMarker)); v >= 1 && v <= len(ms) {
				goneMarker[v-1] = true
			}
		}
	}
	for _, j := range slices.Sorted(maps.Keys(goneSack)) {
		p, n := sacks.at(j)
		b.cut(sacks.span, p, n)
	}
	if len(goneSack) > 0 {
		b.words[sacks.countAt] -= len(goneSack)
	}
	for _, k := range slices.Sorted(maps.Keys(goneMarker)) {
		b.cut(msp, msp.abs(ms[k].at), ms[k].size)
	}
	if len(goneMarker) > 0 {
		b.words[msp.payloadOff] -= len(goneMarker)
	}
	// The kept sacks' bindings follow the elements that remain.
	for j := 0; hasSacks && hasMarkers && j < sacks.n() && len(goneMarker) > 0; j++ {
		if goneSack[j] {
			continue
		}
		p, n := sacks.at(j)
		for q := alm.LootHeadWide; q+alm.LootElementSize <= n; q += alm.LootElementSize {
			slot := p + q + alm.LootElementMarker
			v := int(e.u32(slot))
			if v < 1 || v > len(ms) {
				continue
			}
			nv := 0
			if !goneMarker[v-1] {
				nv = v - below(goneMarker, v-1)
			}
			if nv != v {
				b.set(slot, uint32(nv))
			}
		}
	}
	// The kept units' sack links follow the sacks that remain.
	if len(goneSack) > 0 && f.carries(alm.TypeUnits) {
		uw, err := e.walkSection(KindUnit, "renumber unit sack links")
		if err != nil {
			return nil, err
		}
		for u := 0; u < uw.n(); u++ {
			p, _ := uw.at(u)
			if w.sec.typeID == alm.TypeUnits && u == i {
				continue
			}
			v := int(e.u32(p + alm.SecondUnitSack))
			if v == 0 {
				continue
			}
			nv := 0
			if !goneSack[v-1] {
				nv = v - below(goneSack, v-1)
			}
			if nv != v {
				b.set(p+alm.SecondUnitSack, uint32(nv))
			}
		}
	}
	return b.finish(), nil
}

// below is how many indices of set are under n.
func below(set map[int]bool, n int) int {
	c := 0
	for k := range set {
		if k < n {
			c++
		}
	}
	return c
}

// secondCutLists removes the type-11 entries whose id is the structure's and
// lowers the type-0 counts of their lists.
func (e *Editor) secondCutLists(f frame, b *secondEdit, id uint32) error {
	sp, counts, ok, err := e.secondLists(f)
	if err != nil || !ok {
		return err
	}
	meta := f.required(alm.TypeMeta)
	off := 0
	for a, n := range counts {
		for k := 0; k < n; k++ {
			if e.u32(sp.abs(off)) == id {
				b.cut(sp, sp.abs(off), secondListWidths[a])
				b.words[meta.abs(alm.SecondMeta11Count+4*a)]--
			}
			off += secondListWidths[a]
		}
	}
	return nil
}
