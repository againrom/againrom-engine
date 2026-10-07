package sim

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

const currentTerminalFormVersion byte = 99

type currentTerminalContinuation struct {
	Actors []CurrentTerminalActor
}

func (w *World) appendCurrentTerminalActors(b []byte) []byte {
	if len(w.currentTerminalActors) == 0 {
		return b
	}
	base := b[0]
	b[0] = currentTerminalFormVersion
	raw, _ := json.Marshal(currentTerminalContinuation{Actors: w.currentTerminalActors})
	b = append(b, raw...)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(raw)))
	return append(b, base, 'C', 'T', 'A', '1')
}

func (w *World) unmarshalCurrentTerminalActors(data []byte) error {
	fail := func() error { return fmt.Errorf("malformed current terminal actor section") }
	if len(data) < headerLen+10 || !bytes.Equal(data[len(data)-4:], []byte("CTA1")) {
		return fail()
	}
	baseVersion := data[len(data)-5]
	size := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if size == 0 || size > maxCarriedResumeBytes || size > uint64(len(data)-9-headerLen) {
		return fail()
	}
	start := len(data) - 9 - int(size)
	var continuation currentTerminalContinuation
	d := json.NewDecoder(bytes.NewReader(data[start : len(data)-9]))
	d.DisallowUnknownFields()
	if d.Decode(&continuation) != nil || d.Decode(new(any)) != io.EOF {
		return fail()
	}
	base := bytes.Clone(data[:start])
	base[0] = baseVersion
	next := *w
	if err := next.UnmarshalBinary(base); err != nil {
		return err
	}
	if err := currentTerminalActorsFault(continuation.Actors, next.bounds); err != nil {
		return err
	}
	for _, row := range continuation.Actors {
		if indexOfEntity(next.entities, row.ID) >= 0 {
			return fail()
		}
		for _, dead := range next.originalDead {
			if dead.ID == row.ID {
				return fail()
			}
		}
	}
	canonical, _ := json.Marshal(continuation)
	if !bytes.Equal(canonical, data[start:len(data)-9]) {
		return fail()
	}
	next.currentTerminalActors = continuation.Actors
	*w = next
	return nil
}
