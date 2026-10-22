package sim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const passiveBodyFormVersion byte = 127

type passiveBodyContinuation struct {
	Bodies []PassiveBody
}

// appendPassiveBodies writes the retained terminal contents; a World with none
// writes no form. It is the outermost section and stays out of Hash: the
// contents are passive, so no later state of play depends on them.
func (w *World) appendPassiveBodies(b []byte) []byte {
	if len(w.passiveBodies) == 0 {
		return b
	}
	ws := beginWrapper(b)
	raw, _ := json.Marshal(passiveBodyContinuation{Bodies: w.passiveBodies})
	return endWrapper(append(b, raw...), ws, passiveBodyFormVersion)
}

func (w *World) unmarshalPassiveBodies(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed passive body section") }
	env, ok := readWrapper(data, passiveBodyFormVersion)
	size := uint64(len(env.section()))
	if !ok || size == 0 || size > maxCarriedResumeBytes {
		return fail()
	}
	var continuation passiveBodyContinuation
	d := json.NewDecoder(bytes.NewReader(env.section()))
	d.DisallowUnknownFields()
	if d.Decode(&continuation) != nil || d.Decode(new(any)) != io.EOF || len(continuation.Bodies) == 0 {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	next.passiveBodies = continuation.Bodies
	if err := next.passiveBodiesFault(); err != nil {
		return err
	}
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
