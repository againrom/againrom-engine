package game

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// A SAVE is built from the World alone. The one exception is a field whose
// meaning research has not established: savDocumentFallbacks names each such
// span with its format page, and only those bytes come from the loaded file.
// An object joins its loaded counterpart by World identity, never by position.

type unknownSpan struct {
	offsets []int
	indices [2]int // inclusive array index range; -1 when the field is no array
}

// unknownRecordSpans maps "Class.v.Name" and "Class.r.Name" to their spans.
func unknownRecordSpans() map[string][]unknownSpan {
	out := map[string][]unknownSpan{}
	for _, entry := range savDocumentFallbacks {
		out[entry.Pattern] = append(out[entry.Pattern], parseUnknownSpan(entry))
	}
	return out
}

func parseUnknownSpan(entry savDocumentFallback) unknownSpan {
	span := unknownSpan{indices: [2]int{-1, -1}}
	for _, part := range strings.Split(strings.TrimSuffix(entry.Span, " each"), ",") {
		lo, hi := parseUnknownRange(part)
		for at := lo; at <= hi; at++ {
			span.offsets = append(span.offsets, at)
		}
	}
	if entry.Indices != "" {
		span.indices[0], span.indices[1] = parseUnknownRange(entry.Indices)
	}
	return span
}

func parseUnknownRange(text string) (int, int) {
	text = strings.TrimSpace(text)
	lo, hi, found := strings.Cut(text, "-")
	a, _ := strconv.Atoi(lo)
	if !found {
		return a, a
	}
	b, _ := strconv.Atoi(hi)
	return a, b
}

func copyUnknownBytes(dst, src []byte, offsets []int) {
	for _, at := range offsets {
		if at < len(dst) && at < len(src) {
			dst[at] = src[at]
		}
	}
}

func copyUnknownValue(dst *uint32, src uint32, offsets []int) {
	var a, b [4]byte
	binary.LittleEndian.PutUint32(a[:], *dst)
	binary.LittleEndian.PutUint32(b[:], src)
	copyUnknownBytes(a[:], b[:], offsets)
	*dst = binary.LittleEndian.Uint32(a[:])
}

// worldHeldUnknown names unknown-meaning fields a World carrier holds: a
// structure's saved block and token, and a registry Effect token. The World
// value is written; the loaded bytes are not read.
var worldHeldUnknown = map[string]bool{
	"Building.r.B52": true, "Building.r.Block12": true, "Building.v.B46": true, "Building.v.B48": true, "Building.v.T1C": true,
	"Effect.r.Block12": true, "Effect.v.T08": true, "Effect.v.T1C": true,
}

// heldBasisScalar reports an actor value the World's native basis holds.
func heldBasisScalar(basis *sim.NativeActorBasis, field string) bool {
	if basis == nil {
		return false
	}
	for slot, name := range nativeScalarFields {
		if name == field && slot < sim.ScalarU50 && basis.ScalarIsKnown(slot) {
			return true
		}
	}
	return false
}

// heldBasisByte reports an actor raw byte the World's native basis holds.
func heldBasisByte(basis *sim.NativeActorBasis, field string, at int) bool {
	if basis == nil {
		return false
	}
	switch field {
	case "U114":
		return basis.BasePresent && at < 32 && basis.BaseKnown&(1<<at) != 0
	case "UD4":
		return basis.ModifierPresent && at < 64 && basis.ModifierKnown&(1<<at) != 0
	case "UA6":
		return basis.AttackPresent && at < 32 && basis.AttackKnown&(1<<at) != 0
	case "UBE":
		return basis.DefencePresent && at < 32 && basis.DefenceKnown&(1<<at) != 0
	}
	return false
}

// graftUnknownRecord copies every unknown span of one record class that the
// World does not hold.
func graftUnknownRecord(current, loaded *sav.DocumentRecordData, spans map[string][]unknownSpan, basis *sim.NativeActorBasis) {
	if current.Class != loaded.Class {
		return
	}
	for i := range current.Values {
		v := &current.Values[i]
		pattern := current.Class + ".v." + v.Name
		for _, span := range spans[pattern] {
			if old, err := savedStructureValue(loaded, v.Name); err == nil && !worldHeldUnknown[pattern] && !heldBasisScalar(basis, v.Name) {
				copyUnknownValue(&v.Value, old, span.offsets)
			}
		}
	}
	for i := range current.Raw {
		r := &current.Raw[i]
		pattern := current.Class + ".r." + r.Name
		for _, span := range spans[pattern] {
			if worldHeldUnknown[pattern] {
				continue
			}
			var offsets []int
			for _, at := range span.offsets {
				if !heldBasisByte(basis, r.Name, at) {
					offsets = append(offsets, at)
				}
			}
			for _, old := range loaded.Raw {
				if old.Name == r.Name {
					copyUnknownBytes(r.Bytes, old.Bytes, offsets)
				}
			}
		}
	}
}

// graftUnknownObjects joins each World-built object to its loaded record:
// an actor by its current entity, a Player by its current Player ID, else by
// a unique class and identity key, and copies that record's unknown spans.
func graftUnknownObjects(doc *sav.DocumentData, state, loaded *SnapshotSAVDocument, world *sim.World) error {
	if loaded == nil || loaded.Document == nil || state == nil {
		return nil
	}
	if err := graftUnknownSingletons(doc, loaded, world); err != nil {
		return err
	}
	bases, err := graftHeldBases(doc, state, world)
	if err != nil {
		return err
	}
	old := loaded.Document
	spans := unknownRecordSpans()
	joined := map[uint16]uint16{}
	join := func(to, from uint16) {
		if to == 0 || from == 0 || int(to) > len(doc.Objects) || int(from) > len(old.Objects) {
			return
		}
		if _, done := joined[to]; !done {
			joined[to] = from
		}
	}
	entities := map[sim.EntityID]uint16{}
	for _, a := range loaded.Actors {
		if !a.Retired {
			entities[a.EntityID] = a.ObjectIndex
		}
	}
	for _, a := range state.Actors {
		if !a.Retired {
			join(a.ObjectIndex, entities[a.EntityID])
		}
	}
	if state.GroupBindings != nil && loaded.GroupBindings != nil {
		players := map[uint32]uint16{}
		for _, p := range loaded.GroupBindings.Players {
			players[p.ID] = p.ObjectIndex
		}
		for _, p := range state.GroupBindings.Players {
			join(p.ObjectIndex, players[p.ID])
		}
	}
	keyed := func(d *sav.DocumentData) map[string]uint16 {
		out, repeated := map[string]uint16{}, map[string]bool{}
		for i := range d.Objects {
			r := &d.Objects[i]
			name := "Identity"
			if r.Class == "Player" {
				name = "Slot"
			}
			v, err := savedStructureValue(r, name)
			if err != nil || v == 0 && name == "Identity" {
				continue
			}
			k := r.Class + "/" + strconv.FormatUint(uint64(v), 16)
			if _, seen := out[k]; seen {
				repeated[k] = true
			}
			out[k] = uint16(i + 1)
		}
		for k := range repeated {
			delete(out, k)
		}
		return out
	}
	from := keyed(old)
	for k, to := range keyed(doc) {
		join(to, from[k])
	}
	for to, from := range joined {
		graftUnknownRecord(&doc.Objects[to-1], &old.Objects[from-1], spans, bases[to])
	}
	return nil
}

// graftHeldBases maps each written actor record to the native basis the
// World holds for it: a bound entity's, retired or not, and a removed actor's
// held basis on its terminal or dead record. Those bytes are never grafted.
func graftHeldBases(doc *sav.DocumentData, state *SnapshotSAVDocument, world *sim.World) (map[uint16]*sim.NativeActorBasis, error) {
	bases := map[uint16]*sim.NativeActorBasis{}
	for _, e := range world.Entities() {
		for _, a := range state.Actors {
			if a.EntityID == e.ID && a.ObjectIndex != 0 && int(a.ObjectIndex) <= len(doc.Objects) {
				basis := e.NativeBasis
				bases[a.ObjectIndex] = &basis
			}
		}
	}
	records, removed, err := removedNativeBasisRecords(state, world)
	if err != nil {
		return nil, err
	}
	for i, record := range records {
		// The identity join below admits only a unique nonzero class key.
		key, _ := savedStructureValue(record, "Identity")
		at, matches := -1, 0
		for j := range doc.Objects {
			if r := &doc.Objects[j]; r.Class == record.Class && key != 0 {
				if k, err := savedStructureValue(r, "Identity"); err == nil && k == key {
					at, matches = j, matches+1
				}
			}
		}
		if matches == 1 {
			basis := removed[i]
			bases[uint16(at+1)] = &basis
		}
	}
	return bases, nil
}

// graftUnknownSingletons copies the unknown spans of the head, trailer,
// global word, session and cell residue that the World does not hold.
func graftUnknownSingletons(doc *sav.DocumentData, loaded *SnapshotSAVDocument, world *sim.World) error {
	if loaded == nil || loaded.Document == nil {
		return nil
	}
	old := loaded.Document
	for _, entry := range savDocumentFallbacks {
		span := parseUnknownSpan(entry)
		switch entry.Pattern {
		case "GlobalDWord":
			copyUnknownValue(&doc.GlobalDWord, old.GlobalDWord, span.offsets)
		case "Head.Reserved[]":
			for i := span.indices[0]; i <= span.indices[1] && i < len(doc.Head.Reserved); i++ {
				copyUnknownValue(&doc.Head.Reserved[i], old.Head.Reserved[i], span.offsets)
			}
		case "Trailer[]":
			for i := span.indices[0]; i <= span.indices[1] && i < len(doc.Trailer); i++ {
				copyUnknownValue(&doc.Trailer[i], old.Trailer[i], span.offsets)
			}
		}
	}
	if doc.World == nil || old.World == nil {
		return nil
	}
	n, o := &doc.World.Session, &old.World.Session
	for _, entry := range savDocumentFallbacks {
		span := parseUnknownSpan(entry)
		switch entry.Pattern {
		case "World.Session.DiplomacyHeader":
			copyUnknownBytes(n.DiplomacyHeader[:], o.DiplomacyHeader[:], span.offsets)
		case "World.Session.Raw08":
			if world.RawSessionHead() == ([48]byte{}) {
				copyUnknownBytes(n.Raw08[:], o.Raw08[:], span.offsets)
			}
		case "World.Session.FlagA48":
			n.FlagA48 = o.FlagA48
		case "World.Session.FlagA49":
			n.FlagA49 = o.FlagA49
		case "World.Session.ValueA4C":
			copyUnknownValue(&n.ValueA4C, o.ValueA4C, span.offsets)
		case "World.Session.ValueB3B0":
			copyUnknownValue(&n.ValueB3B0, o.ValueB3B0, span.offsets)
		case "World.Cells[].Residue03", "World.Cells[].Residue32":
		default:
			if strings.HasPrefix(entry.Pattern, "World.") {
				return fmt.Errorf("unknown-meaning World field %s has no graft", entry.Pattern)
			}
		}
	}
	// LOAD keeps the last overlay of a cell key, so that row holds its residue.
	// A cell record or motion cell of the World holds its own residue.
	cells := map[uint16]sav.DocumentCellData{}
	for _, c := range old.World.Cells {
		cells[c.Cell] = c
	}
	for _, row := range world.SavedCellRecords() {
		delete(cells, row.Cell)
	}
	_, motionCells, _, _ := world.SavedActorMotions()
	for _, row := range motionCells {
		delete(cells, row.Cell)
	}
	for i := range doc.World.Cells {
		c := &doc.World.Cells[i]
		if prior, ok := cells[c.Cell]; ok {
			c.Residue03, c.Residue32 = prior.Residue03, prior.Residue32
		}
	}
	return nil
}
