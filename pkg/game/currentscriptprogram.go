package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Program presence distinguishes an intentionally empty current program from
// an older SAV whose mission program must still come from the installed map.
type currentScriptProgram struct {
	Dialect  sim.ScriptDialect `json:",omitempty"`
	Checks   []sim.ScriptCheck
	Instants []sim.ScriptInstant
	Triggers []sim.ScriptTrigger
}

const maxCurrentScriptNodes = 65536

func captureCurrentScriptProgram(w *sim.World) *currentScriptProgram {
	p := w.Script()
	return &currentScriptProgram{Dialect: p.Dialect(), Checks: p.Checks(), Instants: p.Instants(), Triggers: p.Triggers()}
}

func (p *currentScriptProgram) compile() (*sim.Script, error) {
	if len(p.Checks) > 100 || len(p.Instants) > maxCurrentScriptNodes || len(p.Triggers) > maxCurrentScriptNodes {
		return nil, fmt.Errorf("current script program exceeds its register/node bounds")
	}
	triggers := slices.Clone(p.Triggers)
	for i := range triggers {
		triggers[i].Inert = false
	}
	var program *sim.Script
	var err error
	switch p.Dialect {
	case sim.ScriptROM1:
		program, err = sim.NewScript(p.Checks, p.Instants, triggers)
	case sim.ScriptROM2:
		program, err = sim.NewROM2Script(p.Checks, p.Instants, triggers)
	default:
		return nil, fmt.Errorf("invalid current script dialect")
	}
	if err != nil {
		return nil, err
	}
	if !slices.Equal(program.Triggers(), p.Triggers) {
		return nil, fmt.Errorf("current script has inconsistent derived trigger state")
	}
	return program, nil
}

// Stream each array under a count bound before allocating its full slice.
// The enclosing native leaf also has its own eight MiB byte bound.
func currentScriptRows[T any](raw []byte, limit int) ([]T, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	token, err := d.Token()
	if err != nil || token != json.Delim('[') {
		return nil, fmt.Errorf("current script field is not an array")
	}
	var rows []T
	for d.More() {
		if len(rows) == limit {
			return nil, fmt.Errorf("current script array exceeds %d nodes", limit)
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
		return nil, fmt.Errorf("current script array contains trailing data")
	}
	return rows, nil
}

func (p *currentScriptProgram) UnmarshalJSON(raw []byte) error {
	if len(raw) > sav.MaxNativeActions {
		return fmt.Errorf("current script program exceeds native leaf bound")
	}
	var fields struct {
		Dialect                    sim.ScriptDialect
		Checks, Instants, Triggers json.RawMessage
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&fields); err != nil {
		return err
	}
	next := currentScriptProgram{Dialect: fields.Dialect}
	var err error
	if next.Checks, err = currentScriptRows[sim.ScriptCheck](fields.Checks, 100); err != nil {
		return err
	}
	if next.Instants, err = currentScriptRows[sim.ScriptInstant](fields.Instants, maxCurrentScriptNodes); err != nil {
		return err
	}
	if next.Triggers, err = currentScriptRows[sim.ScriptTrigger](fields.Triggers, maxCurrentScriptNodes); err != nil {
		return err
	}
	if _, err := next.compile(); err != nil {
		return err
	}
	*p = next
	return nil
}

func (p *currentScriptProgram) remap(ref func(sim.EntityID, bool) (sim.EntityID, error)) error {
	actor := func(id *sim.EntityID, present bool) error {
		if !present {
			return nil
		}
		var err error
		*id, err = ref(*id, false)
		return err
	}
	structure := func(id *sim.StructureID, present bool) error {
		if !present {
			return nil
		}
		mapped, err := ref(sim.EntityID(*id), true)
		if err == nil {
			*id = sim.StructureID(mapped)
		}
		return err
	}
	for i := range p.Checks {
		c := &p.Checks[i]
		if err := actor(&c.Unit, c.HasUnit); err != nil {
			return err
		}
		if err := actor(&c.Unit2, c.HasUnit2); err != nil {
			return err
		}
		if err := structure(&c.Structure, c.HasStructure); err != nil {
			return err
		}
	}
	for i := range p.Instants {
		in := &p.Instants[i]
		if err := actor(&in.Unit, in.HasUnit); err != nil {
			return err
		}
		if err := actor(&in.Unit2, in.HasUnit2); err != nil {
			return err
		}
		if err := structure(&in.Structure, in.HasStructure); err != nil {
			return err
		}
	}
	return nil
}

func restoreCurrentScriptProgram(ms *Mission, table *mapload.Table, p *currentScriptProgram) error {
	if p == nil {
		if err := restoreCurrentScriptBindings(ms, table); err != nil {
			return err
		}
		return restoreTerminalScriptBindings(ms, table)
	}
	program, err := p.compile()
	if err != nil {
		return err
	}
	return ms.World.RestoreScriptProgram(program)
}
