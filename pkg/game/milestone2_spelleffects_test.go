package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"sort"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// The source side uses only decompression and the archive cursor preceding
// World.BuildingsEnd. All counts, tags, Token bytes, members and graph edges
// below are read here, independently of File.SpellEffects, Record, DocumentData
// decoding and the importers. SAV-DOC-053 fixes the u32 list; SAV-STREAM-013 /
// SAV-ARCHREL-253 fix the shared index state; SAV-TOKEN-034 fixes the 37-byte
// Token; SAV-CLASSSER-173..175 and SAV-EFFCHAIN-046 fix the six body programmes.
// The earlier prefix's class/index/offset locators remain a trusted boundary.
type spell1152Node struct {
	id     uint16
	off    int
	class  string
	values map[string]uint32
	raw    map[string][]byte
	refs   map[string]uint16
}

type spell1152Graph struct {
	roots []uint16
	nodes map[uint16]*spell1152Node
	span  int
}

type spell1152Reader struct {
	b            []byte
	p, end, next int
	classes      map[uint16]string
	prior        map[uint16]sav.SpellEffectPriorObject
	graph        spell1152Graph
	err          error
}

func spell1152Expected(f *sav.File) (spell1152Graph, error) {
	loc, present, err := f.SpellEffectArchiveLocation()
	if err != nil || !present || f.World == nil {
		return spell1152Graph{}, fmt.Errorf("SpellEffect prefix: present=%v err=%v", present, err)
	}
	if loc.Off != f.World.BuildingsEnd {
		return spell1152Graph{}, fmt.Errorf("SpellEffect prefix ends at %d, BuildingsEnd=%d", loc.Off, f.World.BuildingsEnd)
	}
	return spell1152Read(f.Body, loc, f.World.BlocksOff)
}

func spell1152Read(b []byte, loc sav.SpellEffectArchiveLocation, end int) (spell1152Graph, error) {
	r := &spell1152Reader{b: b, p: loc.Off, end: end, next: loc.NextIndex,
		classes: map[uint16]string{}, prior: map[uint16]sav.SpellEffectPriorObject{},
		graph: spell1152Graph{nodes: map[uint16]*spell1152Node{}}}
	if loc.Off < 0 || loc.Off > end || end > len(b) || loc.NextIndex < 1 || loc.NextIndex > 0x8000 {
		return r.graph, fmt.Errorf("invalid SpellEffect span or archive cursor")
	}
	for id, class := range loc.Classes {
		r.classes[id] = class
	}
	for _, obj := range loc.Objects {
		r.prior[obj.Index] = obj
	}
	n := r.number(4)
	if r.err == nil && uint64(n)*2 > uint64(end-r.p) {
		r.err = fmt.Errorf("SpellEffect root count %d exceeds remaining span", n)
	}
	for i := uint32(0); i < n && r.err == nil; i++ {
		r.graph.roots = append(r.graph.roots, r.ref("SpellEffect", 0))
	}
	if r.err == nil && r.p != end {
		r.err = fmt.Errorf("SpellEffect list ends at %d, terrain begins at %d", r.p, end)
	}
	r.graph.span = end - loc.Off
	return r.graph, r.err
}

func (r *spell1152Reader) take(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || r.p < 0 || r.p > r.end-n {
		r.err = fmt.Errorf("truncated SpellEffect field at %d, need %d bytes", r.p, n)
		return nil
	}
	b := r.b[r.p : r.p+n]
	r.p += n
	return b
}

func (r *spell1152Reader) number(n int) uint32 {
	b := r.take(n)
	if b == nil {
		return 0
	}
	switch n {
	case 1:
		return uint32(b[0])
	case 2:
		return uint32(binary.LittleEndian.Uint16(b))
	default:
		return binary.LittleEndian.Uint32(b)
	}
}

func (r *spell1152Reader) index() uint16 {
	if r.next < 1 || r.next >= 0x8000 {
		r.err = fmt.Errorf("SpellEffect archive index space exhausted at %d", r.p)
		return 0
	}
	id := uint16(r.next)
	r.next++
	return id
}

func spell1152Class(class, base string) bool {
	if base == "Effect" {
		return class == "Effect" || class == "Effect_DirectDamage"
	}
	if base == "AreaEffect" {
		return class == "AreaEffect"
	}
	return class == "SpellEffect" || class == "PointEffect" || class == "AreaEffect" || class == "SpellTransport"
}

func (r *spell1152Reader) ref(base string, depth int) uint16 {
	if r.err != nil {
		return 0
	}
	if depth > 16 {
		r.err = fmt.Errorf("SpellEffect nesting exceeds bounded reader at %d", r.p)
		return 0
	}
	tag := uint16(r.number(2))
	if r.err != nil || tag == 0 {
		return 0
	}
	var id uint16
	var class string
	if tag < 0x8000 {
		id = tag
		if node := r.graph.nodes[id]; node != nil {
			class = node.class
		} else if obj, ok := r.prior[id]; ok {
			class = obj.Class
			// An Effect previously attached to an Item can be named here.
			// Its fixed body has no archive calls; read its bytes, not fields
			// from the prefix locator. Other prior classes fail below.
			if spell1152Class(class, "Effect") {
				p, end := r.p, r.end
				r.p, r.end = obj.Off, p
				r.body(id, class, depth)
				r.p, r.end = p, end
			}
		} else {
			r.err = fmt.Errorf("unallocated SpellEffect back-reference %#x", id)
		}
	} else {
		if tag == 0xffff {
			schema, n := r.number(2), r.number(2)
			if r.err == nil && (schema != 1 || n == 0 || n > 32) {
				r.err = fmt.Errorf("invalid SpellEffect class schema=%d length=%d", schema, n)
			}
			class = string(r.take(int(n)))
			if r.err == nil {
				r.classes[r.index()] = class
			}
		} else {
			class = r.classes[tag&0x7fff]
		}
		if r.err == nil && spell1152Class(class, base) {
			id = r.index()
			r.body(id, class, depth)
		}
	}
	if r.err == nil && (!spell1152Class(class, base) || r.graph.nodes[id] == nil) {
		r.err = fmt.Errorf("SpellEffect reference %#x class %q does not satisfy %s", tag, class, base)
	}
	return id
}

func (r *spell1152Reader) body(id uint16, class string, depth int) {
	n := &spell1152Node{id: id, off: r.p, class: class,
		values: map[string]uint32{}, raw: map[string][]byte{}, refs: map[string]uint16{}}
	r.graph.nodes[id] = n // Register before children: a back-reference is identity.
	raw := func(name string, size int) { n.raw[name] = bytes.Clone(r.take(size)) }
	value := func(name string, size int) { n.values[name] = r.number(size) }
	ref := func(name, base string) { n.refs[name] = r.ref(base, depth+1) }
	raw("Block12", 12)
	value("RuntimeID", 4)
	value("T0C", 1)
	value("T0E", 2)
	value("T08", 4)
	value("T18", 2)
	value("T1C", 4)
	value("Identity", 4)
	value("Reference", 4)
	if spell1152Class(class, "Effect") {
		value("E3C", 1)
		value("E3D", 1)
		value("E40", 4)
		value("E0C", 1)
		if class == "Effect_DirectDamage" {
			raw("EDD48", 24)
		}
		return
	}
	value("SE40", 1)
	value("SE41", 1)
	switch class {
	case "PointEffect":
		ref("PE48", "Effect")
		value("PE44", 4)
	case "AreaEffect":
		raw("AE48", 4)
		value("AE4C", 2)
		ref("AE44", "Effect")
	case "SpellTransport":
		ref("ST44", "SpellEffect")
		ref("ST48", "AreaEffect")
		value("ST4C", 2)
	}
}

// The typed layer deliberately expands aliases and omits Token fields
// (DIV-938/939). Only represented class members and reference presence are
// asserted here; documentDifferences below separately requires the full DAG.
func (g spell1152Graph) typedDifferences(live []sim.SavedSpellEffect) []string {
	var differences []string
	add := func(path, detail string) { differences = append(differences, path+": "+detail) }
	var effect func(uint16, *sim.SavedEffect, string)
	effect = func(id uint16, got *sim.SavedEffect, path string) {
		if id == 0 || got == nil {
			if (id == 0) != (got == nil) {
				add(path, "Effect presence differs")
			}
			return
		}
		n := g.nodes[id]
		want := sim.SavedEffect{Class: n.class, E3C: uint8(n.values["E3C"]), E3D: uint8(n.values["E3D"]), E40: n.values["E40"], E0C: uint8(n.values["E0C"])}
		copy(want.DirectDamage[:], n.raw["EDD48"])
		if *got != want {
			add(path, fmt.Sprintf("typed Effect=%+v file=%+v", *got, want))
		}
	}
	var spell func(uint16, *sim.SavedSpellEffect, string, int)
	spell = func(id uint16, got *sim.SavedSpellEffect, path string, depth int) {
		if id == 0 || got == nil {
			if (id == 0) != (got == nil) {
				add(path, "SpellEffect presence differs")
			}
			return
		}
		if depth > 16 {
			add(path, "typed traversal exceeds bound (cyclic source is unadmitted)")
			return
		}
		n := g.nodes[id]
		want := sim.SavedSpellEffect{Class: n.class, SE40: uint8(n.values["SE40"]), SE41: uint8(n.values["SE41"]), PE44: n.values["PE44"], AE4C: uint16(n.values["AE4C"]), ST4C: uint16(n.values["ST4C"])}
		copy(want.AE48[:], n.raw["AE48"])
		flat := *got
		flat.PE48, flat.AE44, flat.ST44, flat.ST48 = nil, nil, nil, nil
		if flat != want {
			add(path, fmt.Sprintf("typed SpellEffect=%+v file=%+v", flat, want))
		}
		effect(n.refs["PE48"], got.PE48, path+".PE48")
		effect(n.refs["AE44"], got.AE44, path+".AE44")
		spell(n.refs["ST44"], got.ST44, path+".ST44", depth+1)
		spell(n.refs["ST48"], got.ST48, path+".ST48", depth+1)
	}
	if len(live) != len(g.roots) {
		add("typed roots", fmt.Sprintf("live=%d raw=%d", len(live), len(g.roots)))
	}
	for i := 0; i < len(live) && i < len(g.roots); i++ {
		spell(g.roots[i], &live[i], fmt.Sprintf("root[%d]", i), 0)
	}
	return differences
}

// Match indices by a bijection established by the ordered roots and named
// edges. Equal-valued distinct objects must stay distinct; a shared child must
// stay shared. Native DTO indices are local IDs, never raw CArchive indices.
func (g spell1152Graph) documentDifferences(state *SnapshotSAVDocument) []string {
	var differences []string
	add := func(format string, args ...any) { differences = append(differences, fmt.Sprintf(format, args...)) }
	if state == nil || state.Document == nil || state.Document.World == nil || state.Unavailable != "" {
		return []string{"complete retained SpellEffect document is unavailable"}
	}
	doc := state.Document
	if len(doc.World.Effects) != len(g.roots) {
		add("retained roots=%d raw=%d", len(doc.World.Effects), len(g.roots))
	}
	forward, reverse := map[uint16]uint16{}, map[uint16]uint16{}
	var visit func(uint16, uint16, string)
	visit = func(rawID, liveID uint16, path string) {
		if rawID == 0 || liveID == 0 {
			if rawID != liveID {
				add("%s: retained reference presence differs", path)
			}
			return
		}
		if other, exists := forward[rawID]; exists {
			if other != liveID {
				add("%s: raw archive %d split across retained objects %d/%d", path, rawID, other, liveID)
			}
			return
		}
		if other, exists := reverse[liveID]; exists && other != rawID {
			add("%s: distinct raw objects %d/%d collapsed into retained %d", path, other, rawID, liveID)
			return
		}
		if int(liveID) > len(doc.Objects) {
			add("%s: missing retained object %d", path, liveID)
			return
		}
		forward[rawID], reverse[liveID] = liveID, rawID
		n, got := g.nodes[rawID], doc.Objects[liveID-1]
		if got.Class != n.class {
			add("%s: retained class %q, raw %q", path, got.Class, n.class)
		}
		values, raw, refs := map[string]uint32{}, map[string][]byte{}, map[string][]uint16{}
		for _, v := range got.Values {
			values[v.Name] = v.Value
		}
		for _, v := range got.Raw {
			raw[v.Name] = v.Bytes
		}
		for _, v := range got.RefSlots {
			refs[v.Name] = v.Objects
		}
		if len(values) != len(got.Values) || !reflect.DeepEqual(values, n.values) {
			add("%s: retained scalar fields=%v raw=%v", path, values, n.values)
		}
		if len(raw) != len(got.Raw) || !reflect.DeepEqual(raw, n.raw) {
			add("%s: retained raw fields differ (Token Position or class tail)", path)
		}
		if len(got.Texts)+len(got.Counts)+len(got.Inline)+len(got.Groups) != 0 || len(refs) != len(n.refs) || len(refs) != len(got.RefSlots) {
			add("%s: retained field or reference population differs", path)
		}
		for _, name := range spell1152Keys(n.refs) {
			gotRefs := refs[name]
			if len(gotRefs) != 1 {
				add("%s.%s: retained reference count=%d want=1", path, name, len(gotRefs))
				continue
			}
			visit(n.refs[name], gotRefs[0], path+"."+name)
		}
	}
	for i := 0; i < len(g.roots) && i < len(doc.World.Effects); i++ {
		visit(g.roots[i], doc.World.Effects[i], fmt.Sprintf("root[%d]", i))
	}
	return differences
}

func spell1152Keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func (g spell1152Graph) population() (fields, typed, references, aliases int, classes map[string]int) {
	classes = map[string]int{"SpellEffect": 0, "PointEffect": 0, "AreaEffect": 0, "SpellTransport": 0, "Effect": 0, "Effect_DirectDamage": 0}
	incoming := map[uint16]int{}
	for _, id := range g.roots {
		if id != 0 {
			incoming[id]++
		}
	}
	for _, n := range g.nodes {
		classes[n.class]++
		own := map[string]int{"SpellEffect": 2, "PointEffect": 6, "AreaEffect": 8, "SpellTransport": 4, "Effect": 7, "Effect_DirectDamage": 31}[n.class]
		typed += own
		fields += 37 + own
		for _, id := range n.refs {
			references++
			if id != 0 {
				incoming[id]++
			}
		}
	}
	for _, count := range incoming {
		if count > 1 {
			aliases += count - 1
		}
	}
	return
}
