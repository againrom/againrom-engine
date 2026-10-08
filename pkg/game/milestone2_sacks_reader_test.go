package game

import (
	"encoding/binary"
	"fmt"
	"slices"
	"sort"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// The expected side reads bytes, never GroundSacks, DocumentData fields or
// savedItemRecord. Only decompression and {archive index, class, body start}
// locations come from sav. SAV-DOC-053 fixes the root count at SessionOff+4374;
// SAV-STREAM-013 fixes tags; SAV-TOKEN-034, ITEM-SAVE-014, SAV-MEMBER-036
// (its unaffected item clauses) and SAV-SPELL-044 fix the seven layouts.
// New-object endpoints and every list count/reference are computed here.
type sackByteRecord struct {
	index    uint16
	class    string
	off, end int
	values   map[string]uint32
	raw      map[string][]byte
	counts   map[string]uint32
	refs     map[string][]uint16
}

type sackByteSource struct {
	count uint32
	roots []uint16
	rows  map[uint16]*sackByteRecord
}

type sack1151Reader struct {
	body    []byte
	byIndex map[uint16]sav.DocumentObjectLocation
	byOff   map[int]sav.DocumentObjectLocation
	source  sackByteSource
	err     error
}

func (r *sack1151Reader) take(p *int, n int) []byte {
	if r.err != nil {
		return make([]byte, n)
	}
	if n < 0 || *p < 0 || *p > len(r.body)-n {
		r.err = fmt.Errorf("Sack byte walk: %d bytes at %d exceed body %d", n, *p, len(r.body))
		return make([]byte, n)
	}
	b := r.body[*p : *p+n]
	*p += n
	return b
}

func (r *sack1151Reader) number(p *int, n int) uint32 {
	b := r.take(p, n)
	switch n {
	case 1:
		return uint32(b[0])
	case 2:
		return uint32(binary.LittleEndian.Uint16(b))
	default:
		return binary.LittleEndian.Uint32(b)
	}
}

func (r *sack1151Reader) list(p *int, depth int) (uint32, []uint16) {
	n := r.number(p, 4)
	if r.err != nil {
		return n, nil
	}
	if n > 65536 || uint64(n)*2 > uint64(len(r.body)-*p) {
		r.err = fmt.Errorf("Sack byte walk: list count %d at %d exceeds remaining tags", n, *p-4)
		return n, nil
	}
	refs := make([]uint16, 0, n)
	for range n {
		refs = append(refs, r.reference(p, depth+1))
		if r.err != nil {
			break
		}
	}
	return n, refs
}

func (r *sack1151Reader) reference(p *int, depth int) uint16 {
	if depth > 16 {
		r.err = fmt.Errorf("Sack byte walk: nesting exceeds 16")
		return 0
	}
	tag := uint16(r.number(p, 2))
	if r.err != nil || tag == 0 {
		return 0
	}
	if tag < 0x8000 {
		loc, ok := r.byIndex[tag]
		if !ok || loc.Off >= *p {
			r.err = fmt.Errorf("Sack byte walk: unknown/forward archive reference %d at %d", tag, *p-2)
			return tag
		}
		r.record(loc, depth)
		return tag
	}
	name := ""
	if tag == 0xffff {
		schema, n := r.number(p, 2), r.number(p, 2)
		if schema != 1 || n == 0 || n > 32 {
			r.err = fmt.Errorf("Sack byte walk: unsupported class schema/name %d/%d", schema, n)
			return 0
		}
		name = string(r.take(p, int(n)))
	}
	loc, ok := r.byOff[*p]
	if !ok || name != "" && loc.Class != name {
		r.err = fmt.Errorf("Sack byte walk: no matching object location at %d (%q)", *p, name)
		return 0
	}
	row := r.record(loc, depth)
	if row != nil {
		*p = row.end
	}
	return loc.ArchiveIndex
}

func (r *sack1151Reader) record(loc sav.DocumentObjectLocation, depth int) *sackByteRecord {
	if old := r.source.rows[loc.ArchiveIndex]; old != nil {
		if old.end == 0 {
			r.err = fmt.Errorf("Sack byte walk: recursive object %d", loc.ArchiveIndex)
		}
		return old
	}
	row := &sackByteRecord{index: loc.ArchiveIndex, class: loc.Class, off: loc.Off,
		values: map[string]uint32{}, raw: map[string][]byte{}, counts: map[string]uint32{}, refs: map[string][]uint16{}}
	r.source.rows[row.index] = row
	p := row.off
	v := func(name string, width int) { row.values[name] = r.number(&p, width) }
	b := func(name string, width int) { row.raw[name] = slices.Clone(r.take(&p, width)) }
	l := func(name string) { row.counts[name], row.refs[name] = r.list(&p, depth) }
	if row.class == "Spell" {
		v("S08", 1)
		v("S09", 1)
		v("S0A", 1)
		v("S0C", 2)
		v("This", 4)
	} else {
		b("Block12", 12)
		v("RuntimeID", 4)
		v("T0C", 1)
		v("T0E", 2)
		v("T08", 4)
		v("T18", 2)
		v("T1C", 4)
		v("Identity", 4)
		v("Reference", 4)
		switch row.class {
		case "Sack":
			v("S3C", 4)
			l("Contents")
			v("Contents1C", 4)
			v("Contents20", 4)
		case "Effect":
			v("E3C", 1)
			v("E3D", 1)
			v("E40", 4)
			v("E0C", 1)
		case "Item", "Weapon", "Armor", "Shield":
			l("Effects")
			v("F40", 2)
			v("F42", 2)
			v("F44", 1)
			v("F45", 1)
			v("F46", 1)
			v("F48", 2)
			v("F4A", 2)
			v("F47", 1)
			switch row.class {
			case "Weapon":
				b("W52", 24)
				b("W6A", 22)
				v("W50", 1)
				row.refs["WeaponSpell"] = []uint16{r.reference(&p, depth+1)}
			case "Armor":
				b("A52", 22)
				v("A50", 1)
			case "Shield":
				b("S50", 22)
			}
		default:
			r.err = fmt.Errorf("Sack byte walk: object %d class %q has no independent layout", row.index, row.class)
		}
	}
	row.end = p
	return row
}

func sackRead(body []byte, start int, locations []sav.DocumentObjectLocation) (sackByteSource, error) {
	r := &sack1151Reader{body: body, byIndex: map[uint16]sav.DocumentObjectLocation{}, byOff: map[int]sav.DocumentObjectLocation{}, source: sackByteSource{rows: map[uint16]*sackByteRecord{}}}
	for _, loc := range locations {
		if _, ok := r.byIndex[loc.ArchiveIndex]; ok {
			return r.source, fmt.Errorf("duplicate archive location %d", loc.ArchiveIndex)
		}
		if _, ok := r.byOff[loc.Off]; ok {
			return r.source, fmt.Errorf("duplicate body location %d", loc.Off)
		}
		r.byIndex[loc.ArchiveIndex], r.byOff[loc.Off] = loc, loc
	}
	r.source.count, r.source.roots = r.list(&start, 0)
	if r.err != nil {
		return r.source, r.err
	}
	for i, index := range r.source.roots {
		if row := r.source.rows[index]; row == nil || row.class != "Sack" {
			return r.source, fmt.Errorf("Sack root slot %d is null or another class", i)
		}
	}
	// A locator cannot silently hide an out-of-list Sack population either.
	for _, loc := range locations {
		if loc.Class == "Sack" && r.source.rows[loc.ArchiveIndex] == nil {
			return r.source, fmt.Errorf("archive Sack %d is absent from raw root walk", loc.ArchiveIndex)
		}
	}
	return r.source, nil
}

func sackByteWalk(f *sav.File) (sackByteSource, error) {
	if f == nil || f.World == nil {
		return sackByteSource{}, fmt.Errorf("Sack acceptance requires a world")
	}
	locations, err := f.DocumentObjectLocations()
	if err != nil {
		return sackByteSource{}, err
	}
	return sackRead(f.Body, f.World.SessionOff+4374, locations)
}

func (s sackByteSource) indices() []uint16 {
	out := make([]uint16, 0, len(s.rows))
	for index := range s.rows {
		out = append(out, index)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func sack1151Token(r *sackByteRecord) sim.SavedObjectToken {
	v := r.values
	t := sim.SavedObjectToken{RuntimeID: v["RuntimeID"], T0C: uint8(v["T0C"]), T0E: uint16(v["T0E"]), T08: v["T08"], T18: uint16(v["T18"]), T1C: v["T1C"], Identity: v["Identity"], Reference: v["Reference"]}
	copy(t.Position[:], r.raw["Block12"])
	return t
}

func sack1151Effect(r *sackByteRecord) sim.ItemEffect {
	return sim.ItemEffect{Kind: uint8(r.values["E3C"]), Mode: uint8(r.values["E3D"]), Operand: r.values["E40"]}
}

func sack1151Spell(r *sackByteRecord) sim.SourceItemSpell {
	if r == nil {
		return sim.SourceItemSpell{}
	}
	return sim.SourceItemSpell{Present: true, ID: uint8(r.values["S08"]), Range: uint8(r.values["S09"]), Defensive: uint8(r.values["S0A"]), ManaCost: uint16(r.values["S0C"])}
}

func (s sackByteSource) item(r *sackByteRecord, id sim.SavedObjectID) sim.ItemStack {
	v := r.values
	x := sim.ItemStack{ObjectID: id, Code: uint16(v["F40"]), Count: v["F42"], Kind: uint8(v["F44"]), Price: int32(v["T1C"]), Weight: int16(v["F4A"]), WeightPresent: true}
	for _, ref := range r.refs["Effects"] {
		child := s.rows[ref]
		if child != nil && child.class == "Effect" && child.values["E0C"] == 0 {
			x.Effects = append(x.Effects, sack1151Effect(child))
		}
	}
	e := &x.SourceEquipment
	switch r.class {
	case "Weapon":
		e.Class, e.OwnKind = sim.SourceWeapon, uint8(v["W50"])
		copy(e.Attack[:], r.raw["W52"])
		copy(e.Defence[:], r.raw["W6A"])
		if refs := r.refs["WeaponSpell"]; len(refs) == 1 {
			e.Spell = sack1151Spell(s.rows[refs[0]])
		}
	case "Armor":
		e.Class, e.OwnKind = sim.SourceArmor, uint8(v["A50"])
		copy(e.Defence[:], r.raw["A52"])
	case "Shield":
		e.Class = sim.SourceShield
		copy(e.Defence[:], r.raw["S50"])
	}
	if e.Class != 0 {
		e.DefinitionRow = uint8(v["T0C"])
	}
	return x
}
