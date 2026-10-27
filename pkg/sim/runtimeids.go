package sim

import (
	"encoding/binary"
	"fmt"
	"slices"
)

// The runtime-id allocator (SAV-1368, SAV-ID-015, MOVE-ID-016): one 65,536-bit
// set gives Token +0x04 to actors, Buildings and Sacks. Bit 0 is always
// marked. The allocator gives the lowest clear bit. A bit is marked while an
// object carries it, so a removed Sack, an actor that left the world and an
// actor zeroed at decay stage 5 release theirs (SAV-1370). runtimeOrphans
// holds the bits a World LOAD marked that no object carries (SAV-1371,
// SAV-1372); nothing releases them. Items and Effects take no id.
const runtimeIDSpace = 1 << 16

type runtimeIDMarks [runtimeIDSpace / 64]uint64

func (m *runtimeIDMarks) mark(id uint32) {
	b := uint16(id)
	m[b>>6] |= 1 << (b & 63)
}

func (m *runtimeIDMarks) marked(id uint16) bool { return m[id>>6]&(1<<(id&63)) != 0 }

// lowestClear returns the lowest clear bit, or 0 when every bit is marked.
func (m *runtimeIDMarks) lowestClear() uint32 {
	for i, word := range m {
		if word != ^uint64(0) {
			for b := 0; b < 64; b++ {
				if word&(1<<b) == 0 {
					return uint32(i*64 + b)
				}
			}
		}
	}
	return 0
}

// RuntimeIDLoad is one Token-derived record a World LOAD reads, in archive
// order: its saved +0x04 and whether its class is Sack, whose constructor
// first takes the lowest clear bit (SAV-1371).
type RuntimeIDLoad struct {
	Saved uint32
	Sack  bool
}

// eachRuntimeIDOwner calls f with the runtime id each World object carries.
func (w *World) eachRuntimeIDOwner(f func(uint32)) {
	for _, e := range w.entities {
		f(e.SourceBinding.RuntimeID)
	}
	for _, r := range w.originalDead {
		if r.terminal.Stage != 0 {
			f(r.terminal.RuntimeID)
		} else {
			f(r.Source.State.RuntimeID)
		}
	}
	for _, s := range w.savedStructures {
		f(s.RuntimeID)
	}
	if w.savedObjects != nil {
		for _, s := range w.savedObjects.Sacks {
			if !s.Retired {
				f(s.Token.RuntimeID)
			}
		}
	}
	for _, s := range w.sacks {
		f(s.runtimeID)
	}
}

func (w *World) runtimeIDMarks() *runtimeIDMarks {
	m := new(runtimeIDMarks)
	m.mark(0)
	w.eachRuntimeIDOwner(func(id uint32) {
		if id != 0 {
			m.mark(id)
		}
	})
	for _, id := range w.runtimeOrphans {
		m.mark(uint32(id))
	}
	return m
}

// allocateRuntimeID returns the lowest clear bit. The caller stores it in its
// object before the next allocation.
func (w *World) allocateRuntimeID() uint32 { return w.runtimeIDMarks().lowestClear() }

// NextRuntimeID is the id the allocator gives next.
func (w *World) NextRuntimeID() uint32 { return w.allocateRuntimeID() }

// RuntimeIDMarks lists the marked bits in ascending order, bit 0 included.
func (w *World) RuntimeIDMarks() []uint16 {
	m := w.runtimeIDMarks()
	var out []uint16
	for id := 0; id < runtimeIDSpace; id++ {
		if m.marked(uint16(id)) {
			out = append(out, uint16(id))
		}
	}
	return out
}

// SackRuntimeIDs is the runtime id of each Sack in Sacks order: its object
// row's Token, else the id the Sack holds.
func (w *World) SackRuntimeIDs() []uint32 {
	out := make([]uint32, len(w.sacks))
	for i, s := range w.sacks {
		out[i] = w.sackRuntimeID(s)
	}
	return out
}

func (w *World) sackRuntimeID(s Sack) uint32 {
	if s.ObjectID != 0 && w.savedObjects != nil {
		if row := w.savedObjects.sack(s.ObjectID); row != nil {
			return row.Token.RuntimeID
		}
	}
	return s.runtimeID
}

// keepSackRuntimeID carries old's runtime id into its copy next: a copy with
// no object row holds it itself, a copy with a row leaves it to the row.
func (w *World) keepSackRuntimeID(next *Sack, old Sack) {
	next.runtimeID = 0
	if next.ObjectID == 0 {
		next.runtimeID = w.sackRuntimeID(old)
	}
}

// RestoreGroundSackRuntimeIDs gives each Sack without an object row, in Sacks
// order, the runtime id its saved record carries. LOAD calls it before
// RestoreRuntimeIDs.
func (w *World) RestoreGroundSackRuntimeIDs(ids []uint32) error {
	if len(ids) != len(w.sacks) {
		return fmt.Errorf("sim: %d Sack runtime ids for %d Sacks", len(ids), len(w.sacks))
	}
	next := slices.Clone(w.sacks)
	for i, id := range ids {
		if next[i].ObjectID != 0 {
			return fmt.Errorf("sim: Sack %d cannot take runtime id %#x", i, id)
		}
		next[i].runtimeID = id
	}
	n := *w
	n.sacks = next
	if err := n.runtimeIDFault(); err != nil {
		return err
	}
	w.sacks = next
	return nil
}

// RestoreRuntimeIDs rebuilds the allocator as World LOAD leaves it
// (SAV-1371): bit 0, then each record in archive order marks the low 16 bits
// of its non-zero saved id, a Sack's constructor first taking the lowest clear
// bit. Bits no World object carries stay marked.
func (w *World) RestoreRuntimeIDs(records []RuntimeIDLoad) {
	m := loadedRuntimeIDMarks(records)
	owned := new(runtimeIDMarks)
	owned.mark(0)
	w.eachRuntimeIDOwner(func(id uint32) {
		if id != 0 {
			owned.mark(id)
		}
	})
	w.runtimeOrphans = nil
	for id := 1; id < runtimeIDSpace; id++ {
		if m.marked(uint16(id)) && !owned.marked(uint16(id)) {
			w.runtimeOrphans = append(w.runtimeOrphans, uint16(id))
		}
	}
}

// loadedRuntimeIDMarks is the bitmap World LOAD leaves (SAV-1371): bit 0,
// then each record in archive order marks the low 16 bits of its non-zero
// saved id, a Sack's constructor first taking the lowest clear bit.
func loadedRuntimeIDMarks(records []RuntimeIDLoad) *runtimeIDMarks {
	m := new(runtimeIDMarks)
	m.mark(0)
	for _, r := range records {
		if r.Sack {
			m.mark(m.lowestClear())
		}
		if r.Saved != 0 {
			m.mark(r.Saved)
		}
	}
	return m
}

// RuntimeIDsAfterLoad is the n ids the allocator gives after a LOAD of
// records once the released ids are cleared (SAV-1370): each the lowest clear
// bit, marked before the next is given. It is shorter than n only when the
// id space is full.
func RuntimeIDsAfterLoad(records []RuntimeIDLoad, released []uint32, n int) []uint32 {
	m := loadedRuntimeIDMarks(records)
	for _, id := range released {
		if b := uint16(id); b != 0 {
			m[b>>6] &^= 1 << (b & 63)
		}
	}
	out := make([]uint32, 0, n)
	for range n {
		id := m.lowestClear()
		if id == 0 {
			break
		}
		m.mark(id)
		out = append(out, id)
	}
	return out
}

// ActorRuntimeIDLoad is one live actor's saved Token +0x04 a LOAD reads.
type ActorRuntimeIDLoad struct {
	Entity EntityID
	Saved  uint32
}

// RestoreActorRuntimeIDs gives each listed actor the id its saved record
// carries, so a World form an older SAVE wrote beside renumbered records
// loads as the file says (SAV-1371). A body a dead manager holds keeps none.
func (w *World) RestoreActorRuntimeIDs(ids []ActorRuntimeIDLoad) {
	held := map[EntityID]bool{}
	for _, d := range w.originalDead {
		held[d.ID] = true
	}
	for _, r := range ids {
		if i := indexOfEntity(w.entities, r.Entity); i >= 0 && !held[r.Entity] {
			w.entities[i].SourceBinding.RuntimeID = r.Saved
		}
	}
}

// InsertActorRuntimeIDs gives each listed on-map actor that carries no
// runtime id the lowest clear bit, in list order (SAV-ID-015).
func (w *World) InsertActorRuntimeIDs(ids []EntityID) {
	marks := w.runtimeIDMarks()
	for _, id := range ids {
		if i := indexOfEntity(w.entities, id); i >= 0 && !w.entities[i].OffMap && w.carriedRuntimeID(i) == 0 {
			w.entities[i].SourceBinding.RuntimeID = marks.lowestClear()
			marks.mark(w.entities[i].SourceBinding.RuntimeID)
		}
	}
}

// InsertSackRuntimeIDs gives each Sack without an object row or an id the
// lowest clear bit, in Sacks order: no saved Sack holds id 0 (SAV-1374), and
// the corpus gives ground Sacks the ids after the actors' (SAV-ID-015).
func (w *World) InsertSackRuntimeIDs() {
	marks := w.runtimeIDMarks()
	for i := range w.sacks {
		if s := &w.sacks[i]; s.ObjectID == 0 && s.runtimeID == 0 {
			s.runtimeID = marks.lowestClear()
			marks.mark(s.runtimeID)
		}
	}
}

// insertActorRuntimeID is the tick-list insertion's allocation (MOVE-ID-016).
func (w *World) insertActorRuntimeID(id EntityID) {
	i := indexOfEntity(w.entities, id)
	if i < 0 || w.entities[i].OffMap || w.carriedRuntimeID(i) != 0 {
		return
	}
	w.entities[i].SourceBinding.RuntimeID = w.allocateRuntimeID()
}

// carriedRuntimeID is the actor's own id, else the one its dead manager holds.
func (w *World) carriedRuntimeID(i int) uint32 {
	if id := w.entities[i].SourceBinding.RuntimeID; id != 0 {
		return id
	}
	for _, r := range w.originalDead {
		if r.ID == w.entities[i].ID && r.terminal.Stage == 0 {
			return r.Source.State.RuntimeID
		}
	}
	return 0
}

// runtimeIDFault refuses a Sack that carries its id twice, an unordered orphan
// list, and an orphan bit an object carries.
func (w *World) runtimeIDFault() error {
	for _, s := range w.sacks {
		if s.runtimeID != 0 && s.ObjectID != 0 {
			return fmt.Errorf("sim: Sack runtime id has two carriers")
		}
	}
	if !slices.IsSorted(w.runtimeOrphans) || slices.Contains(w.runtimeOrphans, 0) || len(slices.Compact(slices.Clone(w.runtimeOrphans))) != len(w.runtimeOrphans) {
		return fmt.Errorf("sim: runtime orphan bits are not canonical")
	}
	if len(w.runtimeOrphans) == 0 {
		return nil
	}
	owned := new(runtimeIDMarks)
	w.eachRuntimeIDOwner(func(id uint32) {
		if id != 0 {
			owned.mark(id)
		}
	})
	for _, id := range w.runtimeOrphans {
		if owned.marked(id) {
			return fmt.Errorf("sim: runtime orphan bit %d is carried", id)
		}
	}
	return nil
}

const runtimeIDFormVersion byte = 128

// appendRuntimeIDs writes the orphan bits and the ids of Sacks that have no
// object row; a World with neither writes no form.
func (w *World) appendRuntimeIDs(b []byte) []byte {
	var sacks []int
	for i, s := range w.sacks {
		if s.runtimeID != 0 {
			sacks = append(sacks, i)
		}
	}
	if len(w.runtimeOrphans) == 0 && len(sacks) == 0 {
		return b
	}
	ws := beginWrapper(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(w.runtimeOrphans)))
	for _, id := range w.runtimeOrphans {
		b = binary.LittleEndian.AppendUint16(b, id)
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(sacks)))
	for _, i := range sacks {
		b = binary.LittleEndian.AppendUint32(b, uint32(i))
		b = binary.LittleEndian.AppendUint32(b, w.sacks[i].runtimeID)
	}
	return endWrapper(b, ws, runtimeIDFormVersion)
}

func (w *World) unmarshalRuntimeIDs(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed runtime id section") }
	env, ok := readWrapper(data, runtimeIDFormVersion)
	if !ok {
		return fail()
	}
	s := env.section()
	if len(s) < 8 {
		return fail()
	}
	n := uint64(binary.LittleEndian.Uint32(s))
	if n > runtimeIDSpace || uint64(len(s)) < 8+2*n {
		return fail()
	}
	orphans := make([]uint16, n)
	for i := range orphans {
		orphans[i] = binary.LittleEndian.Uint16(s[4+2*i:])
	}
	s = s[4+2*n:]
	k := uint64(binary.LittleEndian.Uint32(s))
	s = s[4:]
	if uint64(len(s)) != 8*k {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	next.sacks = slices.Clone(next.sacks)
	for j := uint64(0); j < k; j++ {
		i := uint64(binary.LittleEndian.Uint32(s[8*j:]))
		id := binary.LittleEndian.Uint32(s[8*j+4:])
		if i >= uint64(len(next.sacks)) || id == 0 || next.sacks[i].runtimeID != 0 {
			return fail()
		}
		next.sacks[i].runtimeID = uint32(id)
	}
	if n != 0 {
		next.runtimeOrphans = orphans
	}
	if next.runtimeIDFault() != nil || !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
