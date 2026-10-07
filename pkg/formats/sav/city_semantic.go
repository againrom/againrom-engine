package sav

import (
	"encoding/binary"
	"fmt"
	"sort"
)

// cityObject is a detached, typed CArchive object. It deliberately retains no
// source offsets, encoded tags, compressed bytes, or backing File. Fixed-width
// fields whose meanings are not yet known remain attached to their named class
// member, so a city can be reminted without inventing a value for them.
type cityObject struct {
	sourceIndex uint16
	class       string
	player      *cityPlayer
	unit        *cityUnit
	item        *cityItem
	effect      *cityEffect
	spell       *citySpell
	diary       *cityDiary
}

type cityPlayer struct {
	name   string
	fixed  []byte
	groups []cityGroup
	raw32  []byte
	diary  cityDiary
}

type cityGroup struct {
	words20 []uint16
	raw80   []byte
	words3c []uint16
	actors  []*cityObject
	f1c     uint32
	f40     uint32
	f44     uint32
}

type cityDiary struct {
	dwords    []uint32
	words     []uint16
	reference uint32
}

type cityUnit struct {
	token          []byte
	effects        []*cityObject
	words15c       []uint16
	words178       []uint16
	rawA6          []byte
	rawBE          []byte
	raw114         []byte
	rawD4          []byte
	raw154         []byte
	raw158         []byte
	words158       []uint16
	scalar1        []byte
	reference74    *cityObject
	reference78    *cityObject
	name           string
	scalar2        []byte
	reference68    *cityObject
	containerFlag  byte
	container      []*cityObject
	containerTails [2]uint32
	spellbookFlag  byte
	spellbookDWord uint32
	spellbookCount uint32
	spells         []*cityObject
	scalarTail     []byte
	xp             []byte
	equipment      []*cityObject
}

type cityItem struct {
	token       []byte
	effects     []*cityObject
	fields      []byte
	derived     []byte
	weaponExtra *cityObject
}

type cityEffect struct {
	token  []byte
	fields []byte
}

type citySpell struct{ fields []byte }

type cityDocument struct {
	counter04    uint32
	counter00    uint32
	mapName      string
	head         [13]uint32
	playerList   uint32
	players      []*cityObject
	deadActors   []*cityObject
	worldHalf    byte
	marker       uint32
	globalDWord  uint32
	trailerState []byte
	objects      map[uint16]*cityObject
}

type cityCursor struct {
	b []byte
	p int
}

func (c *cityCursor) take(n int, what string) ([]byte, error) {
	if n < 0 || c.p < 0 || c.p+n > len(c.b) {
		return nil, fmt.Errorf("sav: city %s at %d needs %d bytes (size %d)", what, c.p, n, len(c.b))
	}
	b := c.b[c.p : c.p+n]
	c.p += n
	return b, nil
}

func (c *cityCursor) byte(what string) (byte, error) {
	b, err := c.take(1, what)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (c *cityCursor) u16(what string) (uint16, error) {
	b, err := c.take(2, what)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(b), nil
}

func (c *cityCursor) u32(what string) (uint32, error) {
	b, err := c.take(4, what)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}

func (c *cityCursor) count(what string) (uint32, error) {
	n, err := c.u16(what + " short count")
	if err != nil || n != 0xffff {
		return uint32(n), err
	}
	return c.u32(what + " wide count")
}

func (c *cityCursor) cstring(what string) (string, error) {
	n8, err := c.byte(what + " length8")
	if err != nil {
		return "", err
	}
	n := uint32(n8)
	if n8 == 0xff {
		n16, err := c.u16(what + " length16")
		if err != nil {
			return "", err
		}
		n = uint32(n16)
		if n16 == 0xffff {
			n, err = c.u32(what + " length32")
			if err != nil {
				return "", err
			}
		}
	}
	if uint64(n) > uint64(len(c.b)-c.p) {
		return "", fmt.Errorf("sav: city %s length %d overruns %d remaining bytes", what, n, len(c.b)-c.p)
	}
	b, err := c.take(int(n), what+" bytes")
	return string(b), err
}

func cityCopyField(c *cityCursor, n int, what string) ([]byte, error) {
	b, err := c.take(n, what)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), b...), nil
}

func cityWords(c *cityCursor, what string) ([]uint16, error) {
	n, err := c.count(what + " count")
	if err != nil {
		return nil, err
	}
	if n > maxListElements || uint64(n) > uint64((len(c.b)-c.p)/2) {
		return nil, fmt.Errorf("sav: city %s count %d exceeds its bounded remaining data", what, n)
	}
	out := make([]uint16, int(n))
	for i := range out {
		out[i], err = c.u16(fmt.Sprintf("%s value %d", what, i))
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func cityReadDiary(c *cityCursor, what string) (cityDiary, error) {
	var d cityDiary
	n, err := c.count(what + " dword count")
	if err != nil {
		return d, err
	}
	if n > maxListElements || uint64(n) > uint64((len(c.b)-c.p)/4) {
		return d, fmt.Errorf("sav: city %s dword count %d exceeds its bounded remaining data", what, n)
	}
	d.dwords = make([]uint32, int(n))
	for i := range d.dwords {
		d.dwords[i], err = c.u32(fmt.Sprintf("%s dword %d", what, i))
		if err != nil {
			return d, err
		}
	}
	d.words, err = cityWords(c, what+" words")
	if err != nil {
		return d, err
	}
	d.reference, err = c.u32(what + " reference")
	return d, err
}

type cityArchiveReader struct {
	c       *cityCursor
	next    uint16
	classes map[uint16]string
	objects map[uint16]*cityObject
}

func newCityArchiveReader(c *cityCursor) *cityArchiveReader {
	return &cityArchiveReader{c: c, next: 1, classes: map[uint16]string{}, objects: map[uint16]*cityObject{}}
}

func (r *cityArchiveReader) takeIndex(what string) (uint16, error) {
	if r.next == 0 || r.next&0x8000 != 0 {
		return 0, fmt.Errorf("sav: city %s exhausts the CArchive index space", what)
	}
	i := r.next
	r.next++
	return i, nil
}

func (r *cityArchiveReader) reference(what string) (*cityObject, error) {
	tag, err := r.c.u16(what + " tag")
	if err != nil || tag == 0 {
		return nil, err
	}
	if tag != 0xffff && tag&0x8000 == 0 {
		obj, ok := r.objects[tag]
		if !ok {
			return nil, fmt.Errorf("sav: city %s backreference %d has no earlier object", what, tag)
		}
		return obj, nil
	}
	var className string
	if tag == 0xffff {
		schema, err := r.c.u16(what + " schema")
		if err != nil {
			return nil, err
		}
		if schema != 1 {
			return nil, fmt.Errorf("sav: city %s schema is %d, want 1", what, schema)
		}
		n, err := r.c.u16(what + " class length")
		if err != nil {
			return nil, err
		}
		if n == 0 || n > maxClassName {
			return nil, fmt.Errorf("sav: city %s class length is %d", what, n)
		}
		name, err := r.c.take(int(n), what+" class name")
		if err != nil {
			return nil, err
		}
		className = string(name)
		classIndex, err := r.takeIndex(what + " class")
		if err != nil {
			return nil, err
		}
		r.classes[classIndex] = className
	} else {
		var ok bool
		className, ok = r.classes[tag&0x7fff]
		if !ok {
			return nil, fmt.Errorf("sav: city %s class index %d is unknown", what, tag&0x7fff)
		}
	}
	index, err := r.takeIndex(what + " object")
	if err != nil {
		return nil, err
	}
	obj := &cityObject{sourceIndex: index, class: className}
	r.objects[index] = obj
	if err := r.body(what+" body", obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func (r *cityArchiveReader) referenceList(what string) ([]*cityObject, error) {
	n, err := r.c.u32(what + " count")
	if err != nil {
		return nil, err
	}
	if n > maxListElements || uint64(n) > uint64((len(r.c.b)-r.c.p)/2) {
		return nil, fmt.Errorf("sav: city %s count %d exceeds its bounded remaining data", what, n)
	}
	out := make([]*cityObject, int(n))
	for i := range out {
		out[i], err = r.reference(fmt.Sprintf("%s reference %d", what, i))
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *cityArchiveReader) body(what string, obj *cityObject) error {
	switch obj.class {
	case "Player":
		p, err := r.player(what)
		obj.player = p
		return err
	case "Unit", "Human", "Humanoid":
		u, err := r.unit(what, obj.class)
		obj.unit = u
		return err
	case "Item", "Armor", "Shield", "Weapon":
		i, err := r.readItem(what, obj.class)
		obj.item = i
		return err
	case "Effect":
		token, err := cityCopyField(r.c, 37, what+" Token")
		if err != nil {
			return err
		}
		fields, err := cityCopyField(r.c, 7, what+" fields")
		obj.effect = &cityEffect{token: token, fields: fields}
		return err
	case "Spell":
		fields, err := cityCopyField(r.c, 9, what+" fields")
		obj.spell = &citySpell{fields: fields}
		return err
	case "Diary":
		diary, err := cityReadDiary(r.c, what)
		obj.diary = &diary
		return err
	default:
		return fmt.Errorf("sav: city %s has unsupported class %q", what, obj.class)
	}
}

func (r *cityArchiveReader) player(what string) (*cityPlayer, error) {
	p := &cityPlayer{}
	var err error
	p.name, err = r.c.cstring(what + " name")
	if err != nil {
		return nil, err
	}
	p.fixed, err = cityCopyField(r.c, 51, what+" fixed fields")
	if err != nil {
		return nil, err
	}
	ng, err := r.c.u32(what + " group count")
	if err != nil {
		return nil, err
	}
	if ng > maxListElements {
		return nil, fmt.Errorf("sav: city %s group count %d exceeds %d", what, ng, maxListElements)
	}
	for i := uint32(0); i < ng; i++ {
		g := cityGroup{}
		g.words20, err = cityWords(r.c, fmt.Sprintf("%s group %d words20", what, i))
		if err != nil {
			return nil, err
		}
		g.raw80, err = cityCopyField(r.c, 80, fmt.Sprintf("%s group %d raw80", what, i))
		if err != nil {
			return nil, err
		}
		g.words3c, err = cityWords(r.c, fmt.Sprintf("%s group %d words3c", what, i))
		if err != nil {
			return nil, err
		}
		g.actors, err = r.referenceList(fmt.Sprintf("%s group %d actors", what, i))
		if err != nil {
			return nil, err
		}
		if g.f1c, err = r.c.u32(fmt.Sprintf("%s group %d f1c", what, i)); err != nil {
			return nil, err
		}
		if g.f40, err = r.c.u32(fmt.Sprintf("%s group %d f40", what, i)); err != nil {
			return nil, err
		}
		if g.f44, err = r.c.u32(fmt.Sprintf("%s group %d f44", what, i)); err != nil {
			return nil, err
		}
		p.groups = append(p.groups, g)
	}
	p.raw32, err = cityCopyField(r.c, 32, what+" raw32")
	if err != nil {
		return nil, err
	}
	p.diary, err = cityReadDiary(r.c, what+" embedded Diary")
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *cityArchiveReader) unit(what, class string) (*cityUnit, error) {
	u := &cityUnit{}
	var err error
	u.token, err = cityCopyField(r.c, 37, what+" Token")
	if err != nil {
		return nil, err
	}
	u.effects, err = r.referenceList(what + " effects")
	if err != nil {
		return nil, err
	}
	u.words15c, err = cityWords(r.c, what+" words15c")
	if err != nil {
		return nil, err
	}
	u.words178, err = cityWords(r.c, what+" words178")
	if err != nil {
		return nil, err
	}
	for _, f := range []struct {
		dst  *[]byte
		n    int
		name string
	}{
		{&u.rawA6, 24, "raw-a6"}, {&u.rawBE, 22, "raw-be"},
		{&u.raw114, 24, "raw-114"}, {&u.rawD4, 64, "raw-d4"},
		{&u.raw154, 180, "raw-154"}, {&u.raw158, 148, "raw-158"},
	} {
		*f.dst, err = cityCopyField(r.c, f.n, what+" "+f.name)
		if err != nil {
			return nil, err
		}
	}
	u.words158, err = cityWords(r.c, what+" words158")
	if err != nil {
		return nil, err
	}
	u.scalar1, err = cityCopyField(r.c, 19, what+" scalar1")
	if err != nil {
		return nil, err
	}
	u.reference74, err = r.reference(what + " reference74")
	if err != nil {
		return nil, err
	}
	u.reference78, err = r.reference(what + " reference78")
	if err != nil {
		return nil, err
	}
	u.name, err = r.c.cstring(what + " name")
	if err != nil {
		return nil, err
	}
	u.scalar2, err = cityCopyField(r.c, 55, what+" scalar2")
	if err != nil {
		return nil, err
	}
	u.reference68, err = r.reference(what + " reference68")
	if err != nil {
		return nil, err
	}
	u.containerFlag, err = r.c.byte(what + " container flag")
	if err != nil {
		return nil, err
	}
	if u.containerFlag > 1 {
		return nil, fmt.Errorf("sav: city %s container flag is %d", what, u.containerFlag)
	}
	if u.containerFlag != 0 {
		u.container, err = r.referenceList(what + " container")
		if err != nil {
			return nil, err
		}
		for i := range u.containerTails {
			u.containerTails[i], err = r.c.u32(fmt.Sprintf("%s container tail %d", what, i))
			if err != nil {
				return nil, err
			}
		}
	}
	u.spellbookFlag, err = r.c.byte(what + " spellbook flag")
	if err != nil {
		return nil, err
	}
	if u.spellbookFlag > 1 {
		return nil, fmt.Errorf("sav: city %s spellbook flag is %d", what, u.spellbookFlag)
	}
	if u.spellbookFlag != 0 {
		u.spellbookDWord, err = r.c.u32(what + " spellbook dword")
		if err != nil {
			return nil, err
		}
		u.spellbookCount, err = r.c.u32(what + " spellbook count")
		if err != nil {
			return nil, err
		}
		if u.spellbookCount == 0 || u.spellbookCount > maxListElements {
			return nil, fmt.Errorf("sav: city %s present spellbook count is %d", what, u.spellbookCount)
		}
		for i := uint32(1); i < u.spellbookCount; i++ {
			spell, err := r.reference(fmt.Sprintf("%s spellbook spell %d", what, i))
			if err != nil {
				return nil, err
			}
			u.spells = append(u.spells, spell)
		}
	}
	u.scalarTail, err = cityCopyField(r.c, 17, what+" scalar tail")
	if err != nil {
		return nil, err
	}
	if class != "Unit" {
		u.xp, err = cityCopyField(r.c, 24, what+" humanoid XP")
		if err != nil {
			return nil, err
		}
		for i := 0; i < 13; i++ {
			ref, err := r.reference(fmt.Sprintf("%s equipment %d", what, i+1))
			if err != nil {
				return nil, err
			}
			u.equipment = append(u.equipment, ref)
		}
	}
	return u, nil
}

func (r *cityArchiveReader) readItem(what, class string) (*cityItem, error) {
	i := &cityItem{}
	var err error
	i.token, err = cityCopyField(r.c, 37, what+" Token")
	if err != nil {
		return nil, err
	}
	i.effects, err = r.referenceList(what + " effects")
	if err != nil {
		return nil, err
	}
	i.fields, err = cityCopyField(r.c, 12, what+" item fields")
	if err != nil {
		return nil, err
	}
	switch class {
	case "Weapon":
		i.derived, err = cityCopyField(r.c, 47, what+" Weapon fields")
		if err == nil {
			i.weaponExtra, err = r.reference(what + " Weapon reference")
		}
	case "Armor":
		i.derived, err = cityCopyField(r.c, 23, what+" Armor fields")
	case "Shield":
		i.derived, err = cityCopyField(r.c, 22, what+" Shield fields")
	}
	if err != nil {
		return nil, err
	}
	return i, nil
}

func parseCityDocument(body []byte) (*cityDocument, error) {
	c := &cityCursor{b: body}
	d := &cityDocument{}
	var err error
	if d.counter04, err = c.u32("document counter04"); err != nil {
		return nil, err
	}
	if d.counter00, err = c.u32("document counter00"); err != nil {
		return nil, err
	}
	if d.mapName, err = c.cstring("document map name"); err != nil {
		return nil, err
	}
	for i := range d.head {
		if d.head[i], err = c.u32(fmt.Sprintf("document head %d", i)); err != nil {
			return nil, err
		}
	}
	if d.playerList, err = c.u32("document player-list field"); err != nil {
		return nil, err
	}
	ar := newCityArchiveReader(c)
	if d.players, err = ar.referenceList("document Players"); err != nil {
		return nil, err
	}
	for i, player := range d.players {
		if player != nil && (player.class != "Player" || player.player == nil) {
			return nil, fmt.Errorf("sav: city Player %d is not a Player object", i)
		}
	}
	if d.deadActors, err = ar.referenceList("document dead actors"); err != nil {
		return nil, err
	}
	if d.worldHalf, err = c.byte("document world-half selector"); err != nil {
		return nil, err
	}
	if d.worldHalf != 0 {
		return nil, fmt.Errorf("sav: city authoring requires a no-world save, selector is %d", d.worldHalf)
	}
	if d.marker, err = c.u32("document marker"); err != nil {
		return nil, err
	}
	if d.marker != 0xbadface1 {
		return nil, fmt.Errorf("sav: city document marker is %#08x", d.marker)
	}
	if d.globalDWord, err = c.u32("document global dword"); err != nil {
		return nil, err
	}
	if d.trailerState, err = cityCopyField(c, 400, "document trailer state"); err != nil {
		return nil, err
	}
	if remain := len(body) - c.p; remain != 0 {
		if remain != 1 || c.p%2 == 0 {
			return nil, fmt.Errorf("sav: city document has %d invalid trailing bytes at %d", remain, c.p)
		}
		c.p++ // one encoder-added byte; it is derived again, never retained
	}
	d.objects = ar.objects
	return d, nil
}

type cityArchiveWriter struct {
	b       []byte
	next    uint16
	classes map[string]uint16
	objects map[*cityObject]uint16
}

func newCityArchiveWriter(prefix []byte) *cityArchiveWriter {
	return &cityArchiveWriter{b: append([]byte(nil), prefix...), next: 1, classes: map[string]uint16{}, objects: map[*cityObject]uint16{}}
}

func cityAppendU16(dst []byte, v uint16) []byte { return binary.LittleEndian.AppendUint16(dst, v) }
func cityAppendU32(dst []byte, v uint32) []byte { return binary.LittleEndian.AppendUint32(dst, v) }

func cityAppendCount(dst []byte, n int) ([]byte, error) {
	if n < 0 || uint64(n) > uint64(^uint32(0)) {
		return nil, fmt.Errorf("sav: city count %d does not fit u32", n)
	}
	if n < 0xffff {
		return cityAppendU16(dst, uint16(n)), nil
	}
	dst = cityAppendU16(dst, 0xffff)
	return cityAppendU32(dst, uint32(n)), nil
}

func cityAppendCString(dst []byte, s string) ([]byte, error) {
	n := len(s)
	if uint64(n) > uint64(^uint32(0)) {
		return nil, fmt.Errorf("sav: city string of %d bytes does not fit u32", n)
	}
	switch {
	case n < 0xff:
		dst = append(dst, byte(n))
	case n < 0xffff:
		dst = append(dst, 0xff)
		dst = cityAppendU16(dst, uint16(n))
	default:
		dst = append(dst, 0xff)
		dst = cityAppendU16(dst, 0xffff)
		dst = cityAppendU32(dst, uint32(n))
	}
	return append(dst, s...), nil
}

func cityAppendWords(dst []byte, values []uint16) ([]byte, error) {
	var err error
	dst, err = cityAppendCount(dst, len(values))
	if err != nil {
		return nil, err
	}
	for _, v := range values {
		dst = cityAppendU16(dst, v)
	}
	return dst, nil
}

func cityAppendDiary(dst []byte, d cityDiary) ([]byte, error) {
	var err error
	dst, err = cityAppendCount(dst, len(d.dwords))
	if err != nil {
		return nil, err
	}
	for _, v := range d.dwords {
		dst = cityAppendU32(dst, v)
	}
	if dst, err = cityAppendWords(dst, d.words); err != nil {
		return nil, err
	}
	return cityAppendU32(dst, d.reference), nil
}

func (w *cityArchiveWriter) takeIndex(what string) (uint16, error) {
	if w.next == 0 || w.next&0x8000 != 0 {
		return 0, fmt.Errorf("sav: city %s exhausts the CArchive index space", what)
	}
	i := w.next
	w.next++
	return i, nil
}

func (w *cityArchiveWriter) reference(obj *cityObject) error {
	if obj == nil {
		w.b = cityAppendU16(w.b, 0)
		return nil
	}
	if index, ok := w.objects[obj]; ok {
		w.b = cityAppendU16(w.b, index)
		return nil
	}
	classIndex, seen := w.classes[obj.class]
	if !seen {
		if obj.class == "" || len(obj.class) > maxClassName {
			return fmt.Errorf("sav: city class name %q has invalid length %d", obj.class, len(obj.class))
		}
		w.b = cityAppendU16(w.b, 0xffff)
		w.b = cityAppendU16(w.b, 1)
		w.b = cityAppendU16(w.b, uint16(len(obj.class)))
		w.b = append(w.b, obj.class...)
		var err error
		classIndex, err = w.takeIndex("class " + obj.class)
		if err != nil {
			return err
		}
		w.classes[obj.class] = classIndex
	} else {
		w.b = cityAppendU16(w.b, 0x8000|classIndex)
	}
	objectIndex, err := w.takeIndex("object " + obj.class)
	if err != nil {
		return err
	}
	w.objects[obj] = objectIndex
	return w.body(obj)
}

func (w *cityArchiveWriter) referenceList(refs []*cityObject) error {
	if len(refs) > maxListElements {
		return fmt.Errorf("sav: city reference list has %d elements", len(refs))
	}
	w.b = cityAppendU32(w.b, uint32(len(refs)))
	for _, ref := range refs {
		if err := w.reference(ref); err != nil {
			return err
		}
	}
	return nil
}

func cityRequireLen(class, field string, b []byte, n int) error {
	if len(b) != n {
		return fmt.Errorf("sav: city %s %s has %d bytes, want %d", class, field, len(b), n)
	}
	return nil
}

func (w *cityArchiveWriter) body(obj *cityObject) error {
	switch obj.class {
	case "Player":
		if obj.player == nil {
			return fmt.Errorf("sav: city Player has no body")
		}
		p := obj.player
		if err := cityRequireLen("Player", "fixed", p.fixed, 51); err != nil {
			return err
		}
		if err := cityRequireLen("Player", "raw32", p.raw32, 32); err != nil {
			return err
		}
		var err error
		w.b, err = cityAppendCString(w.b, p.name)
		if err != nil {
			return err
		}
		w.b = append(w.b, p.fixed...)
		w.b = cityAppendU32(w.b, uint32(len(p.groups)))
		for _, g := range p.groups {
			if err := cityRequireLen("Player", "group raw80", g.raw80, 80); err != nil {
				return err
			}
			if w.b, err = cityAppendWords(w.b, g.words20); err != nil {
				return err
			}
			w.b = append(w.b, g.raw80...)
			if w.b, err = cityAppendWords(w.b, g.words3c); err != nil {
				return err
			}
			if err := w.referenceList(g.actors); err != nil {
				return err
			}
			w.b = cityAppendU32(w.b, g.f1c)
			w.b = cityAppendU32(w.b, g.f40)
			w.b = cityAppendU32(w.b, g.f44)
		}
		w.b = append(w.b, p.raw32...)
		w.b, err = cityAppendDiary(w.b, p.diary)
		return err
	case "Unit", "Human", "Humanoid":
		if obj.unit == nil {
			return fmt.Errorf("sav: city %s has no body", obj.class)
		}
		u := obj.unit
		for _, f := range []struct {
			name string
			b    []byte
			n    int
		}{
			{"Token", u.token, 37}, {"rawA6", u.rawA6, 24}, {"rawBE", u.rawBE, 22},
			{"raw114", u.raw114, 24}, {"rawD4", u.rawD4, 64}, {"raw154", u.raw154, 180},
			{"raw158", u.raw158, 148}, {"scalar1", u.scalar1, 19}, {"scalar2", u.scalar2, 55},
			{"scalarTail", u.scalarTail, 17},
		} {
			if err := cityRequireLen(obj.class, f.name, f.b, f.n); err != nil {
				return err
			}
		}
		if obj.class != "Unit" {
			if err := cityRequireLen(obj.class, "XP", u.xp, 24); err != nil {
				return err
			}
			if len(u.equipment) != 13 {
				return fmt.Errorf("sav: city %s equipment has %d refs, want 13", obj.class, len(u.equipment))
			}
		}
		if u.containerFlag > 1 || u.spellbookFlag > 1 {
			return fmt.Errorf("sav: city %s has invalid presence flags %d/%d", obj.class, u.containerFlag, u.spellbookFlag)
		}
		w.b = append(w.b, u.token...)
		if err := w.referenceList(u.effects); err != nil {
			return err
		}
		var err error
		if w.b, err = cityAppendWords(w.b, u.words15c); err != nil {
			return err
		}
		if w.b, err = cityAppendWords(w.b, u.words178); err != nil {
			return err
		}
		for _, b := range [][]byte{u.rawA6, u.rawBE, u.raw114, u.rawD4, u.raw154, u.raw158} {
			w.b = append(w.b, b...)
		}
		if w.b, err = cityAppendWords(w.b, u.words158); err != nil {
			return err
		}
		w.b = append(w.b, u.scalar1...)
		if err := w.reference(u.reference74); err != nil {
			return err
		}
		if err := w.reference(u.reference78); err != nil {
			return err
		}
		if w.b, err = cityAppendCString(w.b, u.name); err != nil {
			return err
		}
		w.b = append(w.b, u.scalar2...)
		if err := w.reference(u.reference68); err != nil {
			return err
		}
		w.b = append(w.b, u.containerFlag)
		if u.containerFlag != 0 {
			if err := w.referenceList(u.container); err != nil {
				return err
			}
			for _, v := range u.containerTails {
				w.b = cityAppendU32(w.b, v)
			}
		}
		w.b = append(w.b, u.spellbookFlag)
		if u.spellbookFlag != 0 {
			if u.spellbookCount != uint32(len(u.spells))+1 {
				return fmt.Errorf("sav: city %s spellbook count=%d refs=%d", obj.class, u.spellbookCount, len(u.spells))
			}
			w.b = cityAppendU32(w.b, u.spellbookDWord)
			w.b = cityAppendU32(w.b, u.spellbookCount)
			for _, spell := range u.spells {
				if err := w.reference(spell); err != nil {
					return err
				}
			}
		}
		w.b = append(w.b, u.scalarTail...)
		if obj.class != "Unit" {
			w.b = append(w.b, u.xp...)
			for _, item := range u.equipment {
				if err := w.reference(item); err != nil {
					return err
				}
			}
		}
		return nil
	case "Item", "Armor", "Shield", "Weapon":
		if obj.item == nil {
			return fmt.Errorf("sav: city %s has no body", obj.class)
		}
		i := obj.item
		derivedLen := map[string]int{"Item": 0, "Armor": 23, "Shield": 22, "Weapon": 47}[obj.class]
		if err := cityRequireLen(obj.class, "Token", i.token, 37); err != nil {
			return err
		}
		if err := cityRequireLen(obj.class, "fields", i.fields, 12); err != nil {
			return err
		}
		if err := cityRequireLen(obj.class, "derived", i.derived, derivedLen); err != nil {
			return err
		}
		w.b = append(w.b, i.token...)
		if err := w.referenceList(i.effects); err != nil {
			return err
		}
		w.b = append(w.b, i.fields...)
		w.b = append(w.b, i.derived...)
		if obj.class == "Weapon" {
			return w.reference(i.weaponExtra)
		}
		return nil
	case "Effect":
		if obj.effect == nil {
			return fmt.Errorf("sav: city Effect has no body")
		}
		if err := cityRequireLen("Effect", "Token", obj.effect.token, 37); err != nil {
			return err
		}
		if err := cityRequireLen("Effect", "fields", obj.effect.fields, 7); err != nil {
			return err
		}
		w.b = append(w.b, obj.effect.token...)
		w.b = append(w.b, obj.effect.fields...)
		return nil
	case "Spell":
		if obj.spell == nil {
			return fmt.Errorf("sav: city Spell has no body")
		}
		if err := cityRequireLen("Spell", "fields", obj.spell.fields, 9); err != nil {
			return err
		}
		w.b = append(w.b, obj.spell.fields...)
		return nil
	case "Diary":
		if obj.diary == nil {
			return fmt.Errorf("sav: city Diary has no body")
		}
		var err error
		w.b, err = cityAppendDiary(w.b, *obj.diary)
		return err
	default:
		return fmt.Errorf("sav: cannot serialize city class %q", obj.class)
	}
}

func serializeCityDocument(d *cityDocument) ([]byte, error) {
	var prefix []byte
	prefix = cityAppendU32(prefix, d.counter04)
	prefix = cityAppendU32(prefix, d.counter00)
	var err error
	prefix, err = cityAppendCString(prefix, d.mapName)
	if err != nil {
		return nil, err
	}
	for _, v := range d.head {
		prefix = cityAppendU32(prefix, v)
	}
	prefix = cityAppendU32(prefix, d.playerList)
	w := newCityArchiveWriter(prefix)
	if err := w.referenceList(d.players); err != nil {
		return nil, err
	}
	if err := w.referenceList(d.deadActors); err != nil {
		return nil, err
	}
	if d.worldHalf != 0 {
		return nil, fmt.Errorf("sav: city serializer refuses world-half=%d", d.worldHalf)
	}
	if d.marker != 0xbadface1 {
		return nil, fmt.Errorf("sav: city serializer refuses marker %#08x", d.marker)
	}
	if len(d.trailerState) != 400 {
		return nil, fmt.Errorf("sav: city trailer state has %d bytes, want 400", len(d.trailerState))
	}
	w.b = append(w.b, d.worldHalf)
	w.b = cityAppendU32(w.b, d.marker)
	w.b = cityAppendU32(w.b, d.globalDWord)
	w.b = append(w.b, d.trailerState...)
	if len(w.b)%2 != 0 {
		w.b = append(w.b, 0)
	}
	return append([]byte(nil), w.b...), nil
}

func cloneCityDocument(source *cityDocument) *cityDocument {
	objects := map[*cityObject]*cityObject{}
	cloneDiary := func(d cityDiary) cityDiary {
		return cityDiary{dwords: append([]uint32(nil), d.dwords...), words: append([]uint16(nil), d.words...), reference: d.reference}
	}
	var cloneObject func(*cityObject) *cityObject
	cloneRefs := func(source []*cityObject) []*cityObject {
		out := make([]*cityObject, len(source))
		for i, obj := range source {
			out[i] = cloneObject(obj)
		}
		return out
	}
	cloneObject = func(source *cityObject) *cityObject {
		if source == nil {
			return nil
		}
		if out, ok := objects[source]; ok {
			return out
		}
		out := &cityObject{sourceIndex: source.sourceIndex, class: source.class}
		objects[source] = out
		if p := source.player; p != nil {
			cp := &cityPlayer{name: p.name, fixed: append([]byte(nil), p.fixed...), raw32: append([]byte(nil), p.raw32...), diary: cloneDiary(p.diary)}
			for _, g := range p.groups {
				cp.groups = append(cp.groups, cityGroup{words20: append([]uint16(nil), g.words20...), raw80: append([]byte(nil), g.raw80...), words3c: append([]uint16(nil), g.words3c...), actors: cloneRefs(g.actors), f1c: g.f1c, f40: g.f40, f44: g.f44})
			}
			out.player = cp
		}
		if u := source.unit; u != nil {
			out.unit = &cityUnit{
				token: append([]byte(nil), u.token...), effects: cloneRefs(u.effects), words15c: append([]uint16(nil), u.words15c...), words178: append([]uint16(nil), u.words178...),
				rawA6: append([]byte(nil), u.rawA6...), rawBE: append([]byte(nil), u.rawBE...), raw114: append([]byte(nil), u.raw114...), rawD4: append([]byte(nil), u.rawD4...),
				raw154: append([]byte(nil), u.raw154...), raw158: append([]byte(nil), u.raw158...), words158: append([]uint16(nil), u.words158...), scalar1: append([]byte(nil), u.scalar1...),
				reference74: cloneObject(u.reference74), reference78: cloneObject(u.reference78), name: u.name, scalar2: append([]byte(nil), u.scalar2...), reference68: cloneObject(u.reference68),
				containerFlag: u.containerFlag, container: cloneRefs(u.container), containerTails: u.containerTails, spellbookFlag: u.spellbookFlag, spellbookDWord: u.spellbookDWord,
				spellbookCount: u.spellbookCount, spells: cloneRefs(u.spells), scalarTail: append([]byte(nil), u.scalarTail...), xp: append([]byte(nil), u.xp...), equipment: cloneRefs(u.equipment),
			}
		}
		if i := source.item; i != nil {
			out.item = &cityItem{token: append([]byte(nil), i.token...), effects: cloneRefs(i.effects), fields: append([]byte(nil), i.fields...), derived: append([]byte(nil), i.derived...), weaponExtra: cloneObject(i.weaponExtra)}
		}
		if e := source.effect; e != nil {
			out.effect = &cityEffect{token: append([]byte(nil), e.token...), fields: append([]byte(nil), e.fields...)}
		}
		if s := source.spell; s != nil {
			out.spell = &citySpell{fields: append([]byte(nil), s.fields...)}
		}
		if d := source.diary; d != nil {
			cd := cloneDiary(*d)
			out.diary = &cd
		}
		return out
	}
	out := &cityDocument{counter04: source.counter04, counter00: source.counter00, mapName: source.mapName, head: source.head, playerList: source.playerList,
		players: cloneRefs(source.players), deadActors: cloneRefs(source.deadActors), worldHalf: source.worldHalf, marker: source.marker, globalDWord: source.globalDWord,
		trailerState: append([]byte(nil), source.trailerState...), objects: map[uint16]*cityObject{}}
	for index, obj := range source.objects {
		out.objects[index] = cloneObject(obj)
	}
	return out
}

func cityObjectIdentity(obj *cityObject) uint32 {
	if obj == nil {
		return 0
	}
	switch {
	case obj.player != nil && len(obj.player.fixed) == 51:
		return binary.LittleEndian.Uint32(obj.player.fixed[47:51])
	case obj.unit != nil && len(obj.unit.token) == 37:
		return binary.LittleEndian.Uint32(obj.unit.token[29:33])
	case obj.item != nil && len(obj.item.token) == 37:
		return binary.LittleEndian.Uint32(obj.item.token[29:33])
	case obj.effect != nil && len(obj.effect.token) == 37:
		return binary.LittleEndian.Uint32(obj.effect.token[29:33])
	case obj.spell != nil && len(obj.spell.fields) == 9:
		return binary.LittleEndian.Uint32(obj.spell.fields[5:9])
	default:
		return 0
	}
}

// tolerateStaleOwners is true only for a frozen document this package did not
// write itself: parsing a base city.ags at load time. That external document
// may already carry a stale item owner ROM1 (or an earlier session) never
// rewrote, and containerOf recovers it below. It is false for this package's
// own export: a current pack we just merged from live Holdings is state we
// built and validated ourselves, so an owner reference that still fails to
// mint is a real defect to refuse, never a gap to paper over with a guess
// that would also swallow a deliberately corrupted current reference.
func remintCityIdentities(d *cityDocument, tolerateStaleOwners bool) error {
	indices := make([]int, 0, len(d.objects))
	for index := range d.objects {
		indices = append(indices, int(index))
	}
	sort.Ints(indices)
	minted := make(map[uint32]uint32)
	for _, index := range indices {
		obj := d.objects[uint16(index)]
		old := cityObjectIdentity(obj)
		identityBearing := obj.player != nil || obj.unit != nil || obj.item != nil || obj.effect != nil || obj.spell != nil
		if !identityBearing {
			continue
		}
		if old == 0 {
			return fmt.Errorf("sav: city %s object %d has zero identity", obj.class, index)
		}
		if _, duplicate := minted[old]; duplicate {
			return fmt.Errorf("sav: city identity %#08x is duplicated", old)
		}
		minted[old] = 0x01000000 + uint32(len(minted)+1)*0x10
	}
	if len(minted) == 0 {
		return fmt.Errorf("sav: city object graph has no remintable identities")
	}
	// A base document can carry an item whose owner token names a Human who
	// is no longer any object in this same graph — a companion who left the
	// roster before ROM1 or an earlier session wrote this file, with nothing
	// left afterwards to rewrite the item's own frozen owner byte.
	// containerOf names, for an item still reachable from a retained Human's
	// own HeldWeapon/HeldShield/Inventory edge or one of his worn armour
	// slots, that Human as its current owner — the same fact the city
	// format's own references already state, not an invented one. Only
	// tolerateStaleOwners consults it.
	containerOf := make(map[uint32]uint32, len(d.objects))
	if tolerateStaleOwners {
		for _, index := range indices {
			obj := d.objects[uint16(index)]
			if obj.unit == nil {
				continue
			}
			holder := cityObjectIdentity(obj)
			add := func(child *cityObject) {
				if child != nil {
					containerOf[cityObjectIdentity(child)] = holder
				}
			}
			add(obj.unit.reference74)
			add(obj.unit.reference78)
			for _, c := range obj.unit.container {
				add(c)
			}
			for _, c := range obj.unit.equipment {
				add(c)
			}
		}
	}
	translate := func(old uint32, what string) (uint32, error) {
		if old == 0 {
			return 0, nil
		}
		v, ok := minted[old]
		if !ok {
			return 0, fmt.Errorf("sav: city %s references unknown identity %#08x", what, old)
		}
		return v, nil
	}
	for _, index := range indices {
		obj := d.objects[uint16(index)]
		switch {
		case obj.player != nil:
			p := obj.player
			own, err := translate(binary.LittleEndian.Uint32(p.fixed[47:51]), "Player identity")
			if err != nil {
				return err
			}
			hero, err := translate(binary.LittleEndian.Uint32(p.fixed[43:47]), "Player hero")
			if err != nil {
				return err
			}
			binary.LittleEndian.PutUint32(p.fixed[47:51], own)
			binary.LittleEndian.PutUint32(p.fixed[43:47], hero)
			for i := range p.groups {
				v, err := translate(p.groups[i].f44, "group owner")
				if err != nil {
					return err
				}
				p.groups[i].f44 = v
			}
			if p.diary.reference, err = translate(p.diary.reference, "embedded Diary reference"); err != nil {
				return err
			}
		case obj.unit != nil:
			own, err := translate(binary.LittleEndian.Uint32(obj.unit.token[29:33]), obj.class+" identity")
			if err != nil {
				return err
			}
			owner, err := translate(binary.LittleEndian.Uint32(obj.unit.token[33:37]), obj.class+" owner")
			if err != nil {
				return err
			}
			binary.LittleEndian.PutUint32(obj.unit.token[29:33], own)
			binary.LittleEndian.PutUint32(obj.unit.token[33:37], owner)
		case obj.item != nil:
			rawIdentity := binary.LittleEndian.Uint32(obj.item.token[29:33])
			own, err := translate(rawIdentity, obj.class+" identity")
			if err != nil {
				return err
			}
			rawOwner := binary.LittleEndian.Uint32(obj.item.token[33:37])
			if tolerateStaleOwners {
				if _, ok := minted[rawOwner]; rawOwner != 0 && !ok {
					if holder, ok := containerOf[rawIdentity]; ok {
						rawOwner = holder
					}
				}
			}
			owner, err := translate(rawOwner, obj.class+" owner")
			if err != nil {
				return err
			}
			binary.LittleEndian.PutUint32(obj.item.token[29:33], own)
			binary.LittleEndian.PutUint32(obj.item.token[33:37], owner)
		case obj.effect != nil:
			own, err := translate(binary.LittleEndian.Uint32(obj.effect.token[29:33]), "Effect identity")
			if err != nil {
				return err
			}
			owner, err := translate(binary.LittleEndian.Uint32(obj.effect.token[33:37]), "Effect owner")
			if err != nil {
				return err
			}
			binary.LittleEndian.PutUint32(obj.effect.token[29:33], own)
			binary.LittleEndian.PutUint32(obj.effect.token[33:37], owner)
		case obj.spell != nil:
			own, err := translate(binary.LittleEndian.Uint32(obj.spell.fields[5:9]), "Spell identity")
			if err != nil {
				return err
			}
			binary.LittleEndian.PutUint32(obj.spell.fields[5:9], own)
		case obj.diary != nil:
			var err error
			if obj.diary.reference, err = translate(obj.diary.reference, "Diary reference"); err != nil {
				return err
			}
		}
	}
	return nil
}
