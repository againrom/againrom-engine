package sim

import (
	"encoding/binary"
	"fmt"
)

// maxVictimKinds is the most victim kinds an attacker's list holds
// (R2-ENGINE-1157).
const maxVictimKinds = 5

// victimKindEntry is one remembered victim kind and the count of kill-route
// calls for it after the first. The count is a word: the original keeps it in
// the high word of the entry's dword, so it wraps at 65536 back to zero.
type victimKindEntry struct {
	kind  uint32
	count uint16
}

// victimKindList is the repeated-victim list of an attacker (R2-ENGINE-1157).
// No SAV record carries it: only the byte form's victim kind section does, so
// a loaded actor starts with an empty list (DIV-4840).
type victimKindList struct {
	n       uint8
	entries [maxVictimKinds]victimKindEntry
}

// victimKind is the key a victim is counted under: its definition table and
// row, the partition of the original's kind word +0x14c except for the rows
// DIV-4844 names.
func victimKind(e Entity) uint32 {
	return uint32(e.SourceBinding.ActorClass())<<16 | uint32(e.SourceBinding.TokenRow)
}

// find returns the count of the first entry holding kind.
func (l *victimKindList) find(kind uint32) (uint16, bool) {
	for _, e := range l.entries[:l.n] {
		if e.kind == kind {
			return e.count, true
		}
	}
	return 0, false
}

// count records one kill-route call for kind: a held kind counts one more;
// a new kind drops the oldest entry when limit entries are held and is
// appended with count zero.
func (l *victimKindList) count(kind uint32, limit int) {
	for i := range l.entries[:l.n] {
		if l.entries[i].kind == kind {
			l.entries[i].count++
			return
		}
	}
	if int(l.n) >= limit {
		copy(l.entries[:], l.entries[1:l.n])
		l.n--
		l.entries[l.n] = victimKindEntry{}
	}
	l.entries[l.n] = victimKindEntry{kind: kind}
	l.n++
}

// countVictimKind is the list update the kill route makes after its award
// call returns, whether the award paid or not. The game's award profile
// sets how many kinds are kept; zero keeps none.
func (w *World) countVictimKind(ai, ti int) {
	limit := w.game.skillAward.victimKinds
	a := &w.entities[ai]
	if limit <= 0 || !a.GainsXP || !InPersistBand(a.TypeID) {
		return
	}
	a.victimKinds.count(victimKind(w.entities[ti]), limit)
}

// victimKindCut is the repeated-victim cut X = 10X / (n + 10), computed as the
// original's 32-bit product and unsigned division, for a victim whose kind the
// attacker's list holds with count n.
func victimKindCut(x int64, n uint16) int64 {
	return int64(int32(uint32(int32(x)*10) / (uint32(n) + 10)))
}

// victimKindFormVersion carries every non-empty victim kind list. A World
// with none writes no form, so the first game's Worlds and every second-game
// World before its first kill-route call keep their bytes.
const victimKindFormVersion byte = 136

func (w *World) appendVictimKinds(b []byte) []byte {
	ws := beginWrapper(b)
	le := binary.LittleEndian
	for i := range w.entities {
		l := &w.entities[i].victimKinds
		if l.n == 0 {
			continue
		}
		b = le.AppendUint32(b, uint32(i))
		b = append(b, l.n)
		for _, e := range l.entries[:l.n] {
			b = le.AppendUint32(b, e.kind)
			b = le.AppendUint16(b, e.count)
		}
	}
	if len(b) == ws.at {
		return b
	}
	return endWrapper(b, ws, victimKindFormVersion)
}

func (w *World) unmarshalVictimKinds(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed victim kind section") }
	env, ok := readWrapper(data, victimKindFormVersion)
	if !ok {
		return fail()
	}
	s := env.section()
	if len(s) == 0 {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	entities := append([]Entity(nil), next.entities...)
	le := binary.LittleEndian
	prior := int64(-1)
	for at := 0; at < len(s); {
		if len(s)-at < 5 {
			return fail()
		}
		index, n := le.Uint32(s[at:]), int(s[at+4])
		at += 5
		if int64(index) <= prior || uint64(index) >= uint64(len(entities)) || n == 0 || n > maxVictimKinds || len(s)-at < 6*n {
			return fail()
		}
		prior = int64(index)
		var l victimKindList
		for j := 0; j < n; j++ {
			kind := le.Uint32(s[at:])
			if _, held := l.find(kind); held {
				return fail()
			}
			l.entries[j] = victimKindEntry{kind: kind, count: le.Uint16(s[at+4:])}
			l.n++
			at += 6
		}
		entities[index].victimKinds = l
	}
	next.entities = entities
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
