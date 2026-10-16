package sim

import (
	"encoding/binary"
	"fmt"
)

const flightRunFormVersion byte = 124
const flightRunRecordLen = 11

func validFlightRun(r FlightRun) bool {
	if !r.Active {
		return r == (FlightRun{})
	}
	return (r.Kind == AttackCharging || r.Kind == AttackCasting) && r.Phase.defined()
}

func (w *World) appendFlightRuns(b []byte) []byte {
	ws := beginWrapper(b)
	for _, e := range w.entities {
		r := e.FlightRun
		if !r.Active && !e.ClientClassFixed {
			continue
		}
		flags := byte(0)
		if r.Active {
			flags |= 1
		}
		if e.ClientClassFixed {
			flags |= 2
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
		b = append(b, flags, byte(r.Kind), byte(r.Phase))
		b = binary.LittleEndian.AppendUint32(b, r.Elapsed)
	}
	if len(b) == ws.at {
		return b
	}
	return endWrapper(b, ws, flightRunFormVersion)
}

func (w *World) unmarshalFlightRuns(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("sim: malformed flight release run span") }
	env, ok := readWrapper(data, flightRunFormVersion)
	span := uint64(len(env.section()))
	if !ok || span == 0 || span%flightRunRecordLen != 0 {
		return fail()
	}
	start := env.start
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	var previous EntityID
	for at := start; at < env.end; at += flightRunRecordLen {
		id := EntityID(binary.LittleEndian.Uint32(data[at:]))
		i := indexOfEntity(next.entities, id)
		if i < 0 || at > start && id <= previous {
			return fail()
		}
		flags := data[at+4]
		if flags == 0 || flags&^3 != 0 {
			return fail()
		}
		r := FlightRun{Active: flags&1 != 0, Kind: AttackPhase(data[at+5]), Phase: AttackPhase(data[at+6]), Elapsed: binary.LittleEndian.Uint32(data[at+7:])}
		if !validFlightRun(r) {
			return fail()
		}
		next.entities[i].FlightRun = r
		next.entities[i].ClientClassFixed = flags&2 != 0
		previous = id
	}
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
