package sim

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"testing"
)

// referenceAppendSavedGroups is the encoding/binary form of the saved-Group
// payload, which the field-by-field form must equal.
func referenceAppendSavedGroups(w *World, b []byte) []byte {
	var payload bytes.Buffer
	put := func(v any) { _ = binary.Write(&payload, binary.LittleEndian, v) }
	words := func(v []uint16) { put(uint32(len(v))); put(v) }
	if s := w.savedGroups; s != nil {
		put(uint32(len(s.Groups)))
		for _, g := range s.Groups {
			put(g.ID)
			put(g.Selector)
			put(g.Authored)
			put(g.Reference)
			put(g.Owner)
			put(g.AI)
			words(g.Words)
			words(g.Path)
			put(uint32(len(g.Members)))
			for _, m := range g.Members {
				put(m.Archive)
				put(m.Entity)
				put(m.Bound)
			}
		}
		put(uint32(len(s.Orders)))
		for _, o := range s.Orders {
			put(o.Entity)
			put(o.State)
			put(o.Authored)
			put(o.RepairStage)
			put(o.EscortTarget)
			put(o.EscortBound)
			put(w.maskedOrderRaw(o))
			words(o.Patrol)
		}
	}
	b = append(b, payload.Bytes()...)
	return binary.LittleEndian.AppendUint32(b, uint32(payload.Len()))
}

func randomSavedGroups(rng *rand.Rand) *savedGroupState {
	s := &savedGroupState{}
	words := func() []uint16 {
		if rng.Intn(3) == 0 {
			return nil
		}
		v := make([]uint16, rng.Intn(5))
		for i := range v {
			v[i] = uint16(rng.Uint32())
		}
		return v
	}
	reference := func() SavedGroupReference {
		return SavedGroupReference{Key: rng.Uint32(), Archive: uint16(rng.Uint32()), Class: uint8(rng.Uint32()), Owner: rng.Uint32()}
	}
	for i := rng.Intn(4); i > 0; i-- {
		g := SavedGroup{ID: rng.Uint32(), ContainerID: rng.Uint32(), OwnerID: rng.Uint32(), Selector: rng.Uint32(), Authored: rng.Intn(2) == 1,
			Reference: reference(), Owner: reference(), Words: words(), Path: words()}
		rng.Read(g.AI[:])
		for j := rng.Intn(4); j > 0; j-- {
			g.Members = append(g.Members, SavedGroupMember{Archive: uint16(rng.Uint32()), Entity: EntityID(rng.Uint32()), Bound: rng.Intn(2) == 1})
		}
		s.Groups = append(s.Groups, g)
	}
	for i := rng.Intn(4); i > 0; i-- {
		o := SavedActorOrder{Entity: EntityID(rng.Intn(3)), State: rng.Uint32(), Authored: rng.Intn(2) == 1, RepairStage: uint8(rng.Uint32()),
			EscortTarget: EntityID(rng.Uint32()), EscortBound: rng.Intn(2) == 1, Patrol: words()}
		rng.Read(o.Raw[:])
		s.Orders = append(s.Orders, o)
	}
	return s
}

func TestSavedGroupPayloadMatchesEncodingBinary(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	w := &World{}
	if got, want := w.appendSavedGroups([]byte{1, 2}), referenceAppendSavedGroups(w, []byte{1, 2}); !bytes.Equal(got, want) {
		t.Fatalf("absent groups: %x, want %x", got, want)
	}
	for n := 0; n < 300; n++ {
		w.savedGroups = randomSavedGroups(rng)
		got, want := w.appendSavedGroups([]byte{9}), referenceAppendSavedGroups(w, []byte{9})
		if !bytes.Equal(got, want) {
			t.Fatalf("case %d: %x, want %x", n, got, want)
		}
	}
}
