package sav

import "fmt"

// TownActorState is the part of an actor record a town document carries from
// the last mission or from the actor's constructor (SAV-1353): the state word
// +0x50, the byte +0x136, the action-run end +0x138, the byte +0x6c, the
// Mover with its two word lists and the Order with its word ring.
type TownActorState struct {
	State    uint32
	Done     uint8
	Deadline uint32
	Timer    uint8
	// Mover is the 180-byte Mover record, nil when the record holds none.
	Mover                     []byte
	StaticRoute, DynamicRoute []uint16
	// Order is the 148-byte Order record, nil when the record holds none.
	Order     []byte
	OrderRing []uint16
}

func townActorState(r *Record) (TownActorState, error) {
	s := TownActorState{Done: uint8(r.value("U136")), Deadline: r.value("U138"), Timer: uint8(r.value("U6C"))}
	if raw := r.Raw["U50"]; len(raw) == 4 {
		s.State = u32(raw, 0)
	} else if len(raw) != 0 {
		return TownActorState{}, fmt.Errorf("sav: the %s at %d has a %d-byte state word", r.Class, r.Off, len(raw))
	}
	if raw := r.Raw["U154"]; len(raw) == 180 {
		s.Mover = append([]byte(nil), raw...)
	} else if len(raw) != 0 {
		return TownActorState{}, fmt.Errorf("sav: the %s at %d has a %d-byte Mover", r.Class, r.Off, len(raw))
	}
	if raw := r.Raw["U158"]; len(raw) == 148 {
		s.Order = append([]byte(nil), raw...)
	} else if len(raw) != 0 {
		return TownActorState{}, fmt.Errorf("sav: the %s at %d has a %d-byte Order", r.Class, r.Off, len(raw))
	}
	for _, list := range []struct {
		name   string
		target *[]uint16
	}{{"U15C", &s.StaticRoute}, {"U178", &s.DynamicRoute}, {"U158_90", &s.OrderRing}} {
		raw := r.Raw[list.name]
		if len(raw) != 2*r.Counts[list.name] {
			return TownActorState{}, fmt.Errorf("sav: the %s at %d has an incomplete %s list: raw %d count %d", r.Class, r.Off, list.name, len(raw), r.Counts[list.name])
		}
		for i := 0; i < len(raw); i += 2 {
			*list.target = append(*list.target, u16(raw, i))
		}
	}
	return s, nil
}
