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
	// The fields are appended one by one in the little-endian widths
	// encoding/binary gives them; savedgroupslayout_test.go holds the two
	// forms equal.
	le := binary.LittleEndian
	start := len(b)
	words := func(v []uint16) {
		b = le.AppendUint32(b, uint32(len(v)))
		for _, x := range v {
			b = le.AppendUint16(b, x)
		}
	}
	reference := func(r SavedGroupReference) {
		b = le.AppendUint32(b, r.Key)
		b = le.AppendUint16(b, r.Archive)
		b = le.AppendUint32(append(b, r.Class), r.Owner)
	}
	if s := w.groupsForm(); s != nil {
		b = le.AppendUint32(b, uint32(len(s.Groups)))
		for _, g := range s.Groups {
			b = le.AppendUint32(b, g.ID)
			b = le.AppendUint32(b, g.Selector)
			b = append(b, layoutBool(g.Authored))
			reference(g.Reference)
			reference(g.Owner)
			b = append(b, g.AI[:]...)
			words(g.Words)
			words(g.Path)
			b = le.AppendUint32(b, uint32(len(g.Members)))
			for _, m := range g.Members {
				b = le.AppendUint16(b, m.Archive)
				b = le.AppendUint32(b, uint32(m.Entity))
				b = append(b, layoutBool(m.Bound))
			}
		}
		b = le.AppendUint32(b, uint32(len(s.Orders)))
		for _, o := range s.Orders {
			b = le.AppendUint32(b, uint32(o.Entity))
			b = le.AppendUint32(b, o.State)
			// Bit 1 of the construction byte is the stood-down walk.
			b = append(b, layoutBool(o.Authored)|layoutBool(s.stoodDownWalk(&o))<<1, o.RepairStage)
			b = le.AppendUint32(b, uint32(o.EscortTarget))
			b = append(b, layoutBool(o.EscortBound))
			raw := w.maskedOrderRaw(o)
			b = append(b, raw[:]...)
			words(o.Patrol)
		}
		// A constructed registry written because it no longer equals the
		// constructed one keeps that mark, so the decoded World omits it
		// again exactly when the live one does. A form without the byte
		// holds an imported registry.
		if s.fresh && !w.SecondGame() {
			b = append(b, 1+layoutBool(s.native))
		}
	}
	return le.AppendUint32(b, uint32(len(b)-start))
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
		if authored > 3 {
			err = fmt.Errorf("sim: invalid actor order construction mode")
		}
		o.Authored, o.stoodDown = authored&1 == 1, authored&2 != 0
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
	if err == nil && r.Len() == 1 {
		var fresh uint8
		get(&fresh)
		if fresh != 1 && fresh != 2 {
			err = fmt.Errorf("sim: invalid saved Group construction mark")
		}
		s.fresh, s.native = true, fresh == 2
	}
	if err != nil || r.Len() != 0 {
		return nil, nil, fmt.Errorf("sim: invalid saved Group payload: %v; remaining %d", err, r.Len())
	}
	s.HighWater = maxSavedGroupID(s.Groups)
	return b[:start], s, nil
}
