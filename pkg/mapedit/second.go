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

// The second format's row: these sections, the three type-7 arrays whose
// adds and removals keep the type-0 word +0x28 at their sum, and its own
// script profile.
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
			reshape:     reshapeSecondStructure,
			renumber:    secondRenumberStructure,
			fields:      secondStructureFields,
			removeParts: secondRemoveParts,
		},
		KindLoot: {name: "sack", typeID: alm.TypeLoot, count: alm.MetaLootCount, x: alm.LootX, y: alm.LootY, create: true,
			length:      lootLength,
			cellless:    func(rec []byte) bool { return binary.LittleEndian.Uint32(rec[alm.LootOwner:]) != 0 },
			fields:      secondLootFields,
			cloneParts:  secondCloneSackParts,
			removeParts: secondRemoveParts,
		},
		KindEnchantment: enchantmentSection(secondEnchantmentFields),
		// The roster row has the first game's layout; its words are read in
		// secondPlayerBaseFields.
		KindPlayer: {name: "player", typeID: alm.TypeGroups, count: alm.MetaGroupCount, create: true,
			length:        func(_ *Editor, b []byte) int { return fixedLength(b, alm.GroupSize) },
			cellless:      func([]byte) bool { return true },
			fields:        secondPlayerFields,
			locked:        secondPlayerLocked,
			validatePlace: validateSecondPlayerPlacement,
			blank:         blankSecondPlayer,
			deleteParts: func(*Editor, walk, int) ([]part, error) {
				return nil, fmt.Errorf("mapedit: a second-game player is not deleted: the owner words that name it are not renumbered")
			},
		},
	}
	// A node's ten text fields are the loader's (R2-ENGINE-1681).
	texts := nodeTextFields()
	for k, s := range map[Kind]*section{KindAction: scriptSection("action", 0, texts), KindCondition: scriptSection("condition", 1, texts), KindTrigger: scriptSection("trigger", 2, nil)} {
		s.tally = alm.SecondMetaScriptCount
		own[k] = s
	}
	// The type-1 word's bit 0x2000 and the height plane are read by the game;
	// the object brush only clears, since what a type-3 value shows is unread
	// (R2-ENGINE-1667); the type-4 bridge is unread. A nonzero type-3 byte
	// stores 5 over the bit's 1 (R2-ENGINE-1340), so the passability stroke
	// clears it to leave the cell closed to move kind 1 alone or open
	// (R2-ENGINE-1805).
	formatRows[alm.SecondFormat] = formatRow{own: own, profile: &scriptprofile.Second, planes: map[Plane]PlaneRule{
		PlaneTile:       {},
		PlaneHeight:     {Band: secondBand},
		PlaneImpassable: {Band: secondBand, ClearsObjects: true},
		PlaneObject:     {Values: []int{0}, Refusal: "editor.map.secondobjects", Band: secondBand},
		PlaneStructure:  {Values: []int{}, Refusal: "editor.map.secondbridge"},
	}}
}

// secondBand is the edge ring the game's map band admits no cell of: x from
// 8 to W - 9 and y from 8 to H - 9 (R2-ENGINE-1668). The server overwrites
// the ring's block plane whatever the cell holds (R2-ENGINE-1341), and the
// client copies no type-3 byte outside x 8..W - 9, y 8..H - 5
// (R2-ENGINE-1667).
const secondBand = 8

// secondUnitFields are the settable fields of a second-game type-6 unit record
// (R2-ENGINE-1510 to R2-ENGINE-1512, R2-ASSET-313 to R2-ASSET-316). The load
// looks 0x08 up to pick the object it builds (R2-ENGINE-513); 0x10 carries the
// Inn offer bits and the no-drop bit (R2-ENGINE-537, R2-ENGINE-535); no reader
// of 0x0a or 0x20 is found (R2-ENGINE-1801).
var secondUnitFields = []Field{
	{Key: "owner", Label: "editor.plain.owner", Offset: alm.SecondUnitOwner, Width: 4, Min: 1, Max: math.MaxUint16, Check: CheckRosterSlot},
	{Key: "definition", Label: "editor.second.unit.definition", Offset: alm.SecondUnitDef, Width: 4, Min: 0, Max: math.MaxUint32},
	{Key: "hp", Label: "editor.second.unit.hp", Offset: alm.SecondUnitHP, Width: 2, Signed: true, Min: -1, Max: math.MaxInt16},
	{Key: "hpmax", Label: "editor.second.unit.hpmax", Offset: alm.SecondUnitHPMax, Width: 2, Signed: true, Min: -1, Max: math.MaxInt16},
	{Key: "flags", Label: "editor.second.unit.flags", Offset: alm.SecondUnitFlags, Width: 4, Min: 0, Max: math.MaxUint32},
	{Key: "group", Label: "editor.unit.field.group", Offset: alm.SecondUnitGroup, Width: 4, Min: 0, Max: math.MaxUint32},
	{Key: "field08", Label: "editor.second.unit.lookup", Offset: alm.SecondUnitClass, Width: 2, Signed: true, Min: math.MinInt16, Max: math.MaxInt16},
	{Key: "field0a", Label: "editor.second.unit.word0a", Offset: alm.SecondUnitField0A, Width: 2, Min: 0, Max: math.MaxUint16},
	{Key: "field10", Label: "editor.second.unit.offers", Offset: alm.SecondUnitFlags10, Width: 4, Min: 0, Max: math.MaxUint32},
	{Key: "field20", Label: "editor.second.unit.dword20", Offset: alm.SecondUnitField20, Width: 4, Min: 0, Max: math.MaxUint32},
}

// secondSpellMax is the highest spell a shipped type-9 record names
// (R2-ENGINE-1009).
const secondSpellMax = 29

// secondEnchantmentFields are the settable fields of a second-game type-9
// record. A cell record is the cell event the stage hands the writer: event
// byte, power byte, source from entry 0, target from entry 1 (R2-ENGINE-1698).
// The profile's cell move takes its event alone and keeps power and source,
// which it reads neither of (R2-ENGINE-1779); another event is a trap that
// casts that spell (R2-ENGINE-673); a record of no event yet lists both. A
// mode-4 record is a structure's caster (R2-ENGINE-1009, R2-ENGINE-1560).
// Every other record, and every byte no field names, keeps its bytes.
func secondEnchantmentFields(e *Editor, rec []byte) []Field {
	move := int64(scriptprofile.VocabularyOf(e.Profile().Ops).CellMove)
	if !spellRecordKnown(rec) || notCellSpell(rec) {
		return nil
	}
	switch mode := binary.LittleEndian.Uint16(rec[12:]); {
	case mode == 4:
		return slices.Clone(secondCasterFields)
	case mode > 4 || move == 0 || move > math.MaxUint8:
		return nil
	}
	cell := []Field{
		{Key: "triggerx", Label: "editor.cast.x", Offset: 4, Width: 4, Min: 0, Max: 255},
		{Key: "triggery", Label: "editor.cast.y", Offset: 8, Width: 4, Min: 0, Max: 255},
		{Key: "targetx", Label: "editor.cast.targetx", Offset: 32, Width: 2, Min: 0, Max: 255},
		{Key: "targety", Label: "editor.cast.targety", Offset: 34, Width: 2, Min: 0, Max: 255},
	}
	spell := Field{Key: "spell", Label: "editor.cast.spell", Offset: 18, Width: 1, Min: 1, Max: secondSpellMax}
	switch int64(rec[18]) {
	case move:
		spell.Label, spell.Min, spell.Max = "editor.cast.event", move, move
		return append([]Field{spell}, cell...)
	case 0:
	default:
		spell.Check = CheckCellSpell
	}
	return append([]Field{spell, {Key: "power", Label: "editor.cast.castpower", Offset: 20, Width: 1, Min: 0, Max: math.MaxUint8},
		cell[0], cell[1],
		{Key: "sourcex", Label: "editor.cast.sourcex", Offset: 26, Width: 2, Min: 0, Max: 255},
		{Key: "sourcey", Label: "editor.cast.sourcey", Offset: 28, Width: 2, Min: 0, Max: 255}}, cell[2:]...)
}

// secondCasterFields are a structure caster's spell byte, power byte and the
// radius of its unit scan, the low byte of +0x00 (R2-ENGINE-1009,
// R2-ENGINE-1560). Its count and entries are read by neither the stage nor
// the caster (R2-ENGINE-1669) and keep their bytes.
var secondCasterFields = []Field{
	{Key: "spell", Label: "editor.cast.spell", Offset: 18, Width: 1, Min: 1, Max: secondSpellMax},
	{Key: "power", Label: "editor.cast.castpower", Offset: 20, Width: 1, Min: 0, Max: math.MaxUint8},
	{Key: "range", Label: "editor.cast.range", Offset: 0, Width: 1, Min: 0, Max: math.MaxUint8},
}

// secondStructureFieldsBase and secondStructureExtFields are the settable
// fields of a second-game type-4 structure record (R2-ASSET-317, R2-ASSET-318).
// The extension replaces the size of the building's row (R2-ENGINE-1289).
var secondStructureFieldsBase = []Field{
	{Key: "kind", Label: "editor.plain.type", Offset: alm.ObjectKind, Width: 4, Min: 1, Max: 255, Check: CheckStructureKind},
	{Key: "owner", Label: "editor.plain.owner", Offset: alm.ObjectOwner, Width: 2, Min: 1, Max: 16, Check: CheckRosterSlot},
	{Key: "field0c", Label: "editor.second.structure.word0c", Offset: alm.ObjectPlacement, Width: 2, Min: 0, Max: math.MaxUint16},
}

var secondStructureExtFields = []Field{
	{Key: "field14", Label: "editor.second.structure.sizex", Offset: alm.ObjectWidth, Width: 4, Min: 0, Max: math.MaxUint32},
	{Key: "field18", Label: "editor.second.structure.sizey", Offset: alm.ObjectHeight, Width: 4, Min: 0, Max: math.MaxUint32},
}

// secondUnreadWordKinds are the kinds the type-4 pass builds a Shop, an Inn
// or a Pointer from; that arm does not read word 0x0c, which the Building arm
// tests for zero (R2-ENGINE-1802). Such a structure lists secondUnreadWord.
var (
	secondUnreadWordKinds = [][2]uint16{{0x22, 0x23}, {0x5d, 0x5f}, {0x69, 0x6b}, {0x43, 0x45}, {0x63, 0x65}, {0x6f, 0x71}, {0x46, 0x4b}}
	secondUnreadWord      = Field{Key: "field0c", Label: "editor.second.structure.word0c.unread", Offset: alm.ObjectPlacement, Width: 2, Min: 0, Max: math.MaxUint16}
)

func secondStructureFields(_ *Editor, rec []byte) []Field {
	out := append([]Field(nil), secondStructureFieldsBase...)
	// A record with the extension is loaded as kind 0x21, a Building.
	kind := binary.LittleEndian.Uint32(rec[alm.ObjectKind:])
	if len(rec) <= alm.ObjectSize && slices.ContainsFunc(secondUnreadWordKinds, func(r [2]uint16) bool {
		return uint16(kind) >= r[0] && uint16(kind) <= r[1]
	}) {
		out[2] = secondUnreadWord
	}
	if len(rec) <= alm.ObjectSize {
		return out
	}
	return append(out, secondStructureExtFields...)
}

// reshapeSecondStructure adds or drops the 8-byte extension when a structure
// kind enters or leaves kind 0x21 or bit 24 (R2-ASSET-317).
func reshapeSecondStructure(rec []byte) []byte {
	kind := binary.LittleEndian.Uint32(rec[alm.ObjectKind:])
	ext := kind == alm.ObjectExtKind || kind&alm.SecondObjectExtBit != 0
	switch {
	case ext && len(rec) == alm.ObjectSize:
		return append(rec, make([]byte, alm.ObjectExtSize)...)
	case !ext && len(rec) > alm.ObjectSize:
		return rec[:alm.ObjectSize]
	}
	return rec
}

// secondLootFields are the settable fields of a second-game type-8 sack record
// (R2-ASSET-254, R2-ASSET-319, R2-ENGINE-1513). A ground sack's bag is placed
// with word 0x10 and takes every item; a unit's sack reads no 0x10, and its
// item word chooses the owner's hand or bag (R2-ENGINE-1328, R2-ENGINE-1803).
func secondLootFields(e *Editor, rec []byte) []Field {
	var out []Field
	head := e.lootHead()
	words := secondSackWords[binary.LittleEndian.Uint32(rec[alm.LootOwner:]) != 0]
	if head == alm.LootHeadWide {
		out = append(out, Field{Key: "field10", Label: words.headWord, Offset: alm.LootGold, Width: 4, Min: 0, Max: math.MaxUint32})
	}
	n := (len(rec) - head) / alm.LootElementSize
	for k := 0; k < n; k++ {
		out = append(out,
			Field{Key: fmt.Sprintf("item%d", k+1), Label: "editor.plain.item",
				Offset: head + k*alm.LootElementSize, Width: 2, Min: 0, Max: math.MaxUint16, Check: CheckItemCode,
				Element: alm.LootElementSize, Counter: alm.LootCount},
			Field{Key: fmt.Sprintf("field04_%d", k+1), Label: words.itemWord,
				Offset: head + k*alm.LootElementSize + alm.LootElementField04, Width: 2, Min: 0, Max: math.MaxUint16},
		)
	}
	out = append(out, Field{Key: "additem", Label: "editor.plain.additem", Offset: len(rec), Width: 2, Min: 0, Max: math.MaxUint16,
		Check: CheckItemCode, Element: alm.LootElementSize, Counter: alm.LootCount})
	return out
}

// secondSackWords word a sack's 0x10 and item word fields, keyed by whether a
// unit owns the sack.
var secondSackWords = map[bool]struct{ headWord, itemWord string }{
	false: {headWord: "editor.second.sack.ground10", itemWord: "editor.second.item.ground"},
	true:  {headWord: "editor.second.sack.owned10", itemWord: "editor.second.item.hand"},
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
