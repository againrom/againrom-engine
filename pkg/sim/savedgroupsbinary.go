package sim

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Form78 appends a bounded saved-Group payload and uint32 byte-span after the
// complete form77, including its clock. A zero span is explicitly absent.
const savedGroupSpanLen = 4

func (w *World) appendSavedGroups(b []byte) []byte {
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

func splitSavedGroups(b []byte) ([]byte, *savedGroupState, error) {
	if len(b) < headerLen+savedGroupSpanLen {
		return nil, nil, fmt.Errorf("sim: missing saved Group span")
	}
	end := len(b) - 4
	n := uint64(binary.LittleEndian.Uint32(b[end:]))
	if n > uint64(end-headerLen) {
		return nil, nil, fmt.Errorf("sim: saved Group span exceeds payload")
	}
	if n == 0 {
		return b[:end], nil, nil
	}
	start := end - int(n)
	r := bytes.NewReader(b[start:end])
	var err error
	get := func(v any) {
		if err == nil {
			err = binary.Read(r, binary.LittleEndian, v)
		}
	}
	count := func(min int) int {
		var v uint32
		get(&v)
		if err != nil {
			return 0
		}
		if uint64(v)*uint64(min) > uint64(r.Len()) {
			err = fmt.Errorf("sim: saved Group count exceeds payload")
			return 0
		}
		return int(v)
	}
	words := func() []uint16 {
		n := count(2)
		if n == 0 {
			return nil
		}
		v := make([]uint16, n)
		get(v)
		return v
	}
	s := &savedGroupState{}
	s.Groups = make([]SavedGroup, count(119))
	for i := range s.Groups {
		g := &s.Groups[i]
		get(&g.ID)
		get(&g.Selector)
		var authored uint8
		get(&authored)
		if authored > 1 {
			err = fmt.Errorf("sim: invalid Group construction mode")
		}
		g.Authored = authored == 1
		get(&g.Reference)
		get(&g.Owner)
		get(&g.AI)
		g.Words, g.Path = words(), words()
		if n := count(7); n > 0 {
			g.Members = make([]SavedGroupMember, n)
		}
		for j := range g.Members {
			m := &g.Members[j]
			get(&m.Archive)
			get(&m.Entity)
			var bound uint8
			get(&bound)
			if bound > 1 {
				err = fmt.Errorf("sim: invalid saved Group member binding")
			}
			m.Bound = bound == 1
		}
	}
	s.Orders = make([]SavedActorOrder, count(163))
	for i := range s.Orders {
		o := &s.Orders[i]
		get(&o.Entity)
		get(&o.State)
		var authored uint8
		get(&authored)
		if authored > 1 {
			err = fmt.Errorf("sim: invalid actor order construction mode")
		}
		o.Authored = authored == 1
		get(&o.RepairStage)
		get(&o.EscortTarget)
		var bound uint8
		get(&bound)
		if bound > 1 {
			err = fmt.Errorf("sim: invalid saved escort binding mode")
		}
		o.EscortBound = bound == 1
		get(&o.Raw)
		o.Patrol = words()
	}
	if err != nil || r.Len() != 0 {
		return nil, nil, fmt.Errorf("sim: invalid saved Group payload: %v; remaining %d", err, r.Len())
	}
	s.HighWater = maxSavedGroupID(s.Groups)
	return b[:start], s, nil
}
