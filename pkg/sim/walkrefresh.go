package sim

import (
	"encoding/binary"
	"fmt"
)

// WalkRefresh is a walking mover's dynamic route state between passes
// (MOVE-REFRESH-012, MOVE-130). Age is Mover +0x78: one per centred walk
// pass while a near route is listed, zeroed by a near search. Searches is
// +0x09: near searches since the last far one. Listed says the head of the
// last near route, NextX/NextY (+0x06), is still owed a step; the step taking
// it ends the list and zeroes Age and the cell. Waited is the pending byte a
// blocked-cell wait leaves in place of the walk's. Every field is zero while
// the mover holds no walk.
type WalkRefresh struct {
	Age, Searches uint8
	Listed        bool
	Waited        bool
	NextX, NextY  int32
}

// walkNext is the listed step of entity i while it is still one cell off.
func (w *World) walkNext(i int) (cell, bool) {
	e := &w.entities[i]
	next := cell{x: e.Walk.NextX, y: e.Walk.NextY}
	return next, e.Walk.Listed && (cell{x: e.X, y: e.Y}).chebyshevTo(next) == 1
}

// waitsBehind is the walk's admission gate (MOVE-WAIT-008, MOVE-130): with
// the stored route's next cell exactly one cell off and held on the plane, a
// nonzero blocked verdict turns entity i toward that cell and ends its pass
// with no search and no step.
func (w *World) waitsBehind(s *routeScratch, i int) bool {
	route := w.routes[i]
	e := &w.entities[i]
	if len(route) == 0 {
		return false
	}
	c := route[0]
	if (cell{x: e.X, y: e.Y}).chebyshevTo(c) != 1 || w.occupancyOpenFootprint(s, i, c.x, c.y) || w.blockedVerdict(i, c) == 0 {
		return false
	}
	w.turnToStep(i, c.x, c.y)
	e.Walk.Waited = true
	return true
}

// blockedVerdict is the blocked-cell verdict for asker i at cell c
// (MOVE-132): 0 lets the asker search round, 1 or 2 makes it wait. Only a
// one-cell unit of a human participant waits. A direct occupant waits it
// while that occupant is alive and either between cells or centred under a
// walk that has not itself stopped to wait, unless c is the asker's own
// listed step or its target. With no direct occupant, a mover still between
// cells that left c waits it.
func (w *World) blockedVerdict(i int, c cell) uint8 {
	a := &w.entities[i]
	if footprintSide(a.TokenSize) != 1 || w.computerControlledUnit(*a) {
		return 0
	}
	layer := a.Domain.layer()
	for j := range w.entities {
		b := &w.entities[j]
		if j == i || !counted(b) || b.Domain.layer() != layer || !entityCoversCell(b, c.x, c.y) {
			continue
		}
		switch {
		case b.InLife(LifeDecaying):
			return 0
		case b.Transit > 0:
			if !a.Walk.Listed || a.Walk.NextX != c.x || a.Walk.NextY != c.y {
				return 1
			}
			return 0
		case b.HasTarget && !b.Walk.Waited:
			if a.TargetX != c.x || a.TargetY != c.y {
				return 2
			}
		}
		return 0
	}
	for j := range w.entities {
		b := &w.entities[j]
		if j != i && counted(b) && b.Domain.layer() == layer && b.stepping() && b.Stride.FromX == c.x && b.Stride.FromY == c.y {
			return 2
		}
	}
	return 0
}

// walkRefreshFormVersion wraps only Worlds with a mover holding walk refresh
// state. Older forms hold none, which is the zero every mover decodes with.
const walkRefreshFormVersion byte = 145

// walkRefreshRecordLen is one record: the entity index, the two counters, the
// flag byte (bit 0 Listed, bit 1 Waited) and the listed cell.
const walkRefreshRecordLen = 15

func (w *World) appendWalkRefresh(b []byte) []byte {
	ws := beginWrapper(b)
	le := binary.LittleEndian
	for i, e := range w.entities {
		r := e.Walk
		if r == (WalkRefresh{}) {
			continue
		}
		var flags byte
		if r.Listed {
			flags |= 1
		}
		if r.Waited {
			flags |= 2
		}
		b = le.AppendUint32(b, uint32(i))
		b = append(b, r.Age, r.Searches, flags)
		b = le.AppendUint32(b, uint32(r.NextX))
		b = le.AppendUint32(b, uint32(r.NextY))
	}
	if len(b) == ws.at {
		return b
	}
	return endWrapper(b, ws, walkRefreshFormVersion)
}

func (w *World) unmarshalWalkRefresh(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed walk refresh section") }
	env, ok := readWrapper(data, walkRefreshFormVersion)
	if !ok {
		return fail()
	}
	s := env.section()
	if len(s) == 0 || len(s)%walkRefreshRecordLen != 0 {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	entities := append([]Entity(nil), next.entities...)
	le := binary.LittleEndian
	prior := int64(-1)
	for at := 0; at < len(s); at += walkRefreshRecordLen {
		index := le.Uint32(s[at:])
		flags := s[at+6]
		if int64(index) <= prior || uint64(index) >= uint64(len(entities)) || flags > 3 || !entities[index].HasTarget {
			return fail()
		}
		prior = int64(index)
		entities[index].Walk = WalkRefresh{Age: s[at+4], Searches: s[at+5], Listed: flags&1 != 0, Waited: flags&2 != 0,
			NextX: int32(le.Uint32(s[at+7:])), NextY: int32(le.Uint32(s[at+11:]))}
	}
	next.entities = entities
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
