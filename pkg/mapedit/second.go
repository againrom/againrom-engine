package mapedit

import (
	"encoding/binary"
	"fmt"
	"math"
	"slices"

	"againrom/pkg/formats/alm"
	"againrom/pkg/scriptprofile"
)

// The second game's placed elements: structures (type 4), units (type 6) and
// sacks (type 8) are rows of the same section table as the first game's, with
// the hooks below. Ids are one space over structures and units (R2-ASSET-256);
// a unit's sack, the type-9 elements its items bind and a structure's type-11
// entries go with it (R2-ASSET-258).

// secondMaxID is the highest id a new element takes: the game keys its unit
// table by the sign-extended low word (R2-ENGINE-1326).
const secondMaxID = math.MaxInt16

// secondWriteOrder is the record order of the original editor's full save
// (R2-ASSET-231); a record the map lacks is inserted after its predecessors.
var secondWriteOrder = []uint32{0, 1, 2, 3, 5, 11, 4, 9, 8, 6, 7, 10, 12}

// The second format's row: these three sections, and its own script profile.
func init() {
	own := map[Kind]*section{
		KindUnit: {name: "unit", typeID: alm.TypeUnits, count: alm.MetaUnitCount, x: alm.UnitX, y: alm.UnitY, create: true,
			length:      func(_ *Editor, b []byte) int { return fixedLength(b, alm.SecondUnitSize) },
			renumber:    secondRenumberUnit,
			cloneParts:  secondCloneUnitParts,
			fields:      func(*Editor, []byte) []Field { return secondUnitFields },
			removeParts: secondRemoveParts,
			moveParts:   secondMoveUnitParts,
		},
		KindStructure: {name: "structure", typeID: alm.TypeObjects, count: alm.MetaObjectCount, x: alm.ObjectX, y: alm.ObjectY, create: true,
			length:      secondStructureLength,
			renumber:    secondRenumberStructure,
			fields:      func(*Editor, []byte) []Field { return nil },
			removeParts: secondRemoveParts,
		},
		KindLoot: {name: "sack", typeID: alm.TypeLoot, count: alm.MetaLootCount, x: alm.LootX, y: alm.LootY, create: true,
			length:      lootLength,
			cellless:    func(rec []byte) bool { return binary.LittleEndian.Uint32(rec[alm.LootOwner:]) != 0 },
			fields:      func(*Editor, []byte) []Field { return nil },
			cloneParts:  secondCloneSackParts,
			removeParts: secondRemoveParts,
		},
	}
	formatRows[alm.SecondFormat] = formatRow{own: own, profile: &scriptprofile.Second}
}

// secondUnitFields is the one unit field the second game's placement sets:
// the owner slot (R2-ASSET-257); every other field of a placed unit keeps the
// template's bytes.
var secondUnitFields = []Field{
	{Key: "owner", Label: "Owner slot", Offset: alm.SecondUnitOwner, Width: 4, Min: 1, Max: math.MaxUint16, Check: CheckRosterSlot},
	{Key: "group", Label: "Group", Offset: alm.SecondUnitGroup, Width: 4, Min: 0, Max: math.MaxUint32},
}

func secondStructureLength(_ *Editor, b []byte) int {
	if len(b) < alm.ObjectSize {
		return -1
	}
	kind := binary.LittleEndian.Uint32(b[alm.ObjectKind:])
	if kind == alm.ObjectExtKind || kind&alm.SecondObjectExtBit != 0 {
		return fixedLength(b, alm.ObjectSize+alm.ObjectExtSize)
	}
	return alm.ObjectSize
}

// NextID is the id one above the highest the map holds or the session has
// given: the original editor's map word that a placement raises and a delete
// never lowers (R2-ASSET-256).
func (e *Editor) NextID() uint32 {
	used, _ := e.secondIDs()
	next := e.idNext
	for id := range used {
		next = max(next, id+1)
	}
	return max(next, 1)
}

// secondIDs is every id the structures and units of the map hold. A ground
// sack's id is not in the file.
func (e *Editor) secondIDs() (map[uint32]bool, error) {
	used := map[uint32]bool{}
	f := e.locate()
	if f.carries(alm.TypeObjects) {
		w, err := e.walkSection(KindStructure, "read the element ids")
		if err != nil {
			return nil, err
		}
		for i := 0; i < w.n(); i++ {
			at, _ := w.at(i)
			used[uint32(binary.LittleEndian.Uint16(e.data[at+alm.ObjectIdentity:]))] = true
		}
	}
	if f.carries(alm.TypeUnits) {
		w, err := e.walkSection(KindUnit, "read the element ids")
		if err != nil {
			return nil, err
		}
		for i := 0; i < w.n(); i++ {
			at, _ := w.at(i)
			used[e.u32(at+alm.SecondUnitTag)&0xffff] = true
		}
	}
	return used, nil
}

// takeID gives the lowest id from 1 no structure or unit holds and raises the
// session's next id above it.
func (e *Editor) takeID() (uint32, error) {
	used, err := e.secondIDs()
	if err != nil {
		return 0, err
	}
	for id := uint32(1); id <= secondMaxID; id++ {
		if !used[id] {
			e.idNext = max(e.idNext, id+1)
			return id, nil
		}
	}
	return 0, fmt.Errorf("mapedit: every element id up to %d is taken", secondMaxID)
}

func secondRenumberUnit(e *Editor, _ walk, rec []byte) error {
	id, err := e.takeID()
	if err != nil {
		return err
	}
	x := binary.LittleEndian.Uint32(rec[alm.UnitX:])
	binary.LittleEndian.PutUint32(rec[alm.SecondUnitTag:], id|(x&0xffff)<<16)
	return nil
}

func secondRenumberStructure(e *Editor, _ walk, rec []byte) error {
	id, err := e.takeID()
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint16(rec[alm.ObjectIdentity:], uint16(id))
	return nil
}

// secondAnchor is the record a new record of type tid is written after: the
// last one of the map whose type the original's save writes before it.
func (e *Editor) secondAnchor(f frame, tid uint32) span {
	want := slices.Index(secondWriteOrder, tid)
	anchor := f.required(alm.TypeMeta)
	for _, sp := range f.physical {
		if r := slices.Index(secondWriteOrder, e.u32(sp.headerOff+alm.RecordTypeID)); r >= 0 && r < want {
			anchor = sp
		}
	}
	return anchor
}

// secondMarker is one type-9 element: its payload-relative start, its length
// and the two fields a deletion tests.
type secondMarker struct {
	at, size int
	mode     uint16
	key      uint32
}

// secondMarkers walks the type-9 payload: the count word, then the elements
// (R2-ASSET-230). ok is false for a map without the record.
func (e *Editor) secondMarkers(f frame) (sp span, ms []secondMarker, ok bool, err error) {
	if sp, ok = f.at(alm.TypeTileMarkers); !ok {
		return sp, nil, false, nil
	}
	p := e.data[sp.payloadOff : sp.payloadOff+sp.payloadSize]
	if len(p) < 4 {
		return sp, nil, true, fmt.Errorf("mapedit: the type-9 payload is %d bytes", len(p))
	}
	n := int64(binary.LittleEndian.Uint32(p))
	if n*alm.SecondMarkerHead > int64(len(p)) {
		return sp, nil, true, fmt.Errorf("mapedit: the type-9 payload cannot hold %d elements", n)
	}
	off := 4
	for k := int64(0); k < n; k++ {
		if off+alm.SecondMarkerHead > len(p) {
			return sp, nil, true, fmt.Errorf("mapedit: type-9 element %d does not fit", k)
		}
		entries := int64(binary.LittleEndian.Uint32(p[off+alm.SecondMarkerCount:]))
		size := alm.SecondMarkerHead + entries*alm.SecondMarkerEntry
		if size > int64(len(p)-off) {
			return sp, nil, true, fmt.Errorf("mapedit: type-9 element %d does not fit", k)
		}
		ms = append(ms, secondMarker{at: off, size: int(size),
			mode: binary.LittleEndian.Uint16(p[off+alm.SecondMarkerMode:]), key: binary.LittleEndian.Uint32(p[off+alm.SecondMarkerKey:])})
		off += int(size)
	}
	if off != len(p) {
		return sp, nil, true, fmt.Errorf("mapedit: the type-9 walk consumed %d of %d payload bytes", off, len(p))
	}
	return sp, ms, true, nil
}

// secondListWidths are the widths of the type-11 lists in file order.
var secondListWidths = [3]int{alm.SecondType11A, alm.SecondType11B, alm.SecondType11C}

// secondLists locates the type-11 record, whose lists the type-0 words from
// SecondMeta11Count count; ok is false for a map without it.
func (e *Editor) secondLists(f frame) (sp span, counts [3]int, ok bool, err error) {
	for _, p := range f.physical {
		if e.u32(p.headerOff+alm.RecordTypeID) == alm.SecondTypeLists {
			sp, ok = p, true
		}
	}
	if !ok {
		return sp, counts, false, nil
	}
	meta, total := f.required(alm.TypeMeta), 0
	for a := range counts {
		n := int(e.u32(meta.abs(alm.SecondMeta11Count + 4*a)))
		if n < 0 || n > sp.payloadSize {
			return sp, counts, true, fmt.Errorf("mapedit: the type-11 lists do not fit")
		}
		counts[a] = n
		total += n * secondListWidths[a]
	}
	if total != sp.payloadSize {
		return sp, counts, true, fmt.Errorf("mapedit: the type-11 lists hold %d of %d payload bytes", total, sp.payloadSize)
	}
	return sp, counts, true, nil
}

// secondMoveUnitParts keeps a moved unit's 0x28 high word at the low word of
// its x and its sacks on its position (R2-ASSET-258).
func secondMoveUnitParts(e *Editor, w walk, i int, x, y uint32) []part {
	at, _ := w.at(i)
	var out []part
	tag := e.u32(at + alm.SecondUnitTag)
	if nt := tag&0xffff | (x&0xffff)<<16; nt != tag {
		var img [4]byte
		binary.LittleEndian.PutUint32(img[:], nt)
		out = append(out, replace(at+alm.SecondUnitTag, e.data[at+alm.SecondUnitTag:at+alm.SecondUnitTag+4], img[:]))
	}
	id := tag & 0xffff
	if id == 0 || !w.frame.carries(alm.TypeLoot) {
		return out
	}
	sacks, err := e.walkSection(KindLoot, "move the sacks of a unit")
	if err != nil {
		return out
	}
	var img [8]byte
	binary.LittleEndian.PutUint32(img[:], x)
	binary.LittleEndian.PutUint32(img[4:], y)
	for j := 0; j < sacks.n(); j++ {
		if p, _ := sacks.at(j); e.u32(p+alm.LootOwner) == id {
			out = append(out, replace(p+alm.LootX, e.data[p+alm.LootX:p+alm.LootX+8], img[:]))
		}
	}
	return out
}

// secondRebind copies the type-9 elements the items of sack bind out of src
// and gives each item its element's new 1-based position, after the first
// have elements of the destination's type-9 record. A binding that names no
// element is cleared.
func secondRebind(src *Editor, sack []byte, have int) (added [][]byte, err error) {
	sp, ms, _, err := src.secondMarkers(src.locate())
	if err != nil {
		return nil, err
	}
	for q := alm.LootHeadWide; q+alm.LootElementSize <= len(sack); q += alm.LootElementSize {
		at := q + alm.LootElementMarker
		v := int(binary.LittleEndian.Uint32(sack[at:]))
		if v < 1 || v > len(ms) {
			binary.LittleEndian.PutUint32(sack[at:], 0)
			continue
		}
		m := ms[v-1]
		added = append(added, cloneBytes(src.data[sp.abs(m.at):sp.abs(m.at)+m.size]))
		binary.LittleEndian.PutUint32(sack[at:], uint32(have+len(added)))
	}
	return added, nil
}

// markerAppendParts appends elements to the type-9 record.
func (e *Editor) markerAppendParts(added [][]byte) ([]part, error) {
	if len(added) == 0 {
		return nil, nil
	}
	sp, ok := e.locate().at(alm.TypeTileMarkers)
	if !ok {
		return nil, fmt.Errorf("mapedit: this map carries no type-9 record to hold the bound elements")
	}
	var all []byte
	for _, a := range added {
		all = append(all, a...)
	}
	return []part{
		replace(sp.abs(sp.payloadSize), nil, all),
		e.sizeWordPart(sp, len(all)),
		e.wordPart(sp.payloadOff, len(added)),
	}, nil
}

// secondCloneSackParts gives a copied ground sack its own bound elements.
func secondCloneSackParts(e *Editor, _ walk, src *Editor, _ walk, _ int, rec []byte, _ bool) ([]part, error) {
	_, ms, _, err := e.secondMarkers(e.locate())
	if err != nil {
		return nil, err
	}
	added, err := secondRebind(src, rec, len(ms))
	if err != nil {
		return nil, err
	}
	return e.markerAppendParts(added)
}

// secondCloneUnitParts copies the template unit's sack with the new unit's id
// as owner and its position, with its bound elements, and links it
// (R2-ASSET-257).
func secondCloneUnitParts(e *Editor, _ walk, src *Editor, _ walk, _ int, rec []byte, _ bool) ([]part, error) {
	link := int(binary.LittleEndian.Uint32(rec[alm.SecondUnitSack:]))
	binary.LittleEndian.PutUint32(rec[alm.SecondUnitSack:], 0)
	if link == 0 {
		return nil, nil
	}
	sw, err := src.walkSection(KindLoot, "clone the sack of a unit")
	if err != nil {
		return nil, err
	}
	if link > sw.n() {
		return nil, nil
	}
	at, l := sw.at(link - 1)
	sack := cloneBytes(src.data[at : at+l])
	binary.LittleEndian.PutUint32(sack[alm.LootOwner:], binary.LittleEndian.Uint32(rec[alm.SecondUnitTag:])&0xffff)
	copy(sack[alm.LootX:alm.LootX+8], rec[alm.UnitX:alm.UnitX+8])
	_, ms, _, err := e.secondMarkers(e.locate())
	if err != nil {
		return nil, err
	}
	added, err := secondRebind(src, sack, len(ms))
	if err != nil {
		return nil, err
	}
	w, err := e.placementSection(KindLoot, "clone the sack of a unit")
	if err != nil {
		return nil, err
	}
	n, ed := e.appendRecordParts(w, sack)
	binary.LittleEndian.PutUint32(rec[alm.SecondUnitSack:], uint32(n+1))
	more, err := e.markerAppendParts(added)
	if err != nil {
		return nil, err
	}
	return append(ed, more...), nil
}
