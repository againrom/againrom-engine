package sim

import (
	"bytes"
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
	ws := beginWrapper(b)
	raw, _ := json.Marshal(currentTerminalContinuation{Actors: w.currentTerminalActors})
	return endWrapper(append(b, raw...), ws, currentTerminalFormVersion)
}

func (w *World) unmarshalCurrentTerminalActors(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed current terminal actor section") }
	env, ok := readWrapper(data, currentTerminalFormVersion)
	size := uint64(len(env.section()))
	if !ok || size == 0 || size > maxCarriedResumeBytes {
		return fail()
	}
	var continuation currentTerminalContinuation
	d := json.NewDecoder(bytes.NewReader(env.section()))
	d.DisallowUnknownFields()
	if d.Decode(&continuation) != nil || d.Decode(new(any)) != io.EOF {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
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
	if !bytes.Equal(canonical, env.section()) {
		return fail()
	}
	next.currentTerminalActors = continuation.Actors
	*w = next
	return nil
}
