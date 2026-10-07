package sim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

type CurrentROM2GroupActivity struct {
	Owner, Group uint32
	Forced       bool
}

type CurrentROM2Policy struct {
	Scenario [1024]int32
	Groups   []CurrentROM2GroupActivity
}

func (p *CurrentROM2Policy) UnmarshalJSON(raw []byte) error {
	var fields struct {
		Scenario json.RawMessage
		Groups   json.RawMessage
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&fields); err != nil {
		return err
	}
	bank, err := currentROM2Rows[int32](fields.Scenario, 1024)
	if err != nil || len(bank) != 1024 {
		return fmt.Errorf("sim: current ROM2 scenario bank must contain 1024 values")
	}
	groups, err := currentROM2Rows[CurrentROM2GroupActivity](fields.Groups, maxROM2Groups)
	if err != nil {
		return err
	}
	next := CurrentROM2Policy{Groups: groups}
	copy(next.Scenario[:], bank)
	if err := next.validate(); err != nil {
		return err
	}
	*p = next
	return nil
}

func (p *CurrentROM2Policy) validate() error {
	if len(p.Groups) > maxROM2Groups {
		return fmt.Errorf("sim: current ROM2 group activity exceeds its bound")
	}
	for i, g := range p.Groups {
		if i > 0 {
			prior := p.Groups[i-1]
			if g.Owner < prior.Owner || g.Owner == prior.Owner && g.Group <= prior.Group {
				return fmt.Errorf("sim: current ROM2 group activity is not canonical")
			}
		}
	}
	return nil
}

func (w *World) currentROM2Policy() *CurrentROM2Policy {
	if w.rom2 == nil {
		return nil
	}
	p := &CurrentROM2Policy{Scenario: w.rom2.Scenario}
	for _, g := range w.rom2.Groups {
		p.Groups = append(p.Groups, CurrentROM2GroupActivity{g.Owner, g.Group, g.Forced})
	}
	return p
}

func (w *World) restoreCurrentROM2Policy(p *CurrentROM2Policy) error {
	if (p != nil) != (w.script.Dialect() == ScriptROM2) {
		return fmt.Errorf("sim: current ROM2 policy and script dialect disagree")
	}
	if p == nil {
		return nil
	}
	if err := p.validate(); err != nil {
		return err
	}
	state := &rom2ScriptState{Scenario: p.Scenario}
	for _, g := range p.Groups {
		state.Groups = append(state.Groups, rom2GroupActivity{g.Owner, g.Group, g.Forced})
	}
	w.rom2 = state
	return nil
}

func currentROM2Rows[T any](raw []byte, limit int) ([]T, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	token, err := d.Token()
	if err != nil || token != json.Delim('[') {
		return nil, fmt.Errorf("sim: current ROM2 field is not an array")
	}
	var rows []T
	for d.More() {
		if len(rows) == limit {
			return nil, fmt.Errorf("sim: current ROM2 field exceeds its bound")
		}
		var row T
		if err := d.Decode(&row); err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	if _, err := d.Token(); err != nil {
		return nil, err
	}
	var tail any
	if err := d.Decode(&tail); err != io.EOF {
		return nil, fmt.Errorf("sim: current ROM2 field has trailing data")
	}
	return rows, nil
}
