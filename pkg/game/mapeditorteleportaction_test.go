package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// A first-game map using the Teleport action asks to extend to .almx, writes
// the action with both cells through the extended script record, refuses an
// .alm save, reads back identically, and runs and round-trips its save in sim;
// a map without the action keeps its .alm bytes unchanged.
func TestFirstGameTeleportActionWritesAlmxReadsBackAndPlays(t *testing.T) {
	plain := formMap(t, 2)
	e, _ := openGoalMap(t, plain)
	s := formHost(e)

	// Control: a map saved without the Teleport action keeps its bytes.
	unchangedPath := filepath.Join(t.TempDir(), "unchanged.alm")
	if _, err := s.SaveAs(unchangedPath); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(unchangedPath); err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("map without teleport changed bytes: %v", err)
	}

	// Node popup SetFields to Teleport on an un-extended map asks to extend.
	var need *ui.ScriptFormNeeded
	if _, err := s.SetFields(ui.KindAction, 1, []ui.InspectionFieldValue{{Key: "opcode", Value: int64(scriptprofile.TeleportCellOp)}}); !errors.As(err, &need) || need.Fold {
		t.Fatalf("SetFields to teleport on plain map answered %v, want ScriptFormNeeded", err)
	}

	// Trigger wizard plan with a repeating nearest-unit check and Teleport action.
	plan := ui.WizardTrigger{
		Index: -1,
		Name:  "Pad",
		Once:  0,
		Pairs: []ui.WizardPair{{
			Left:  ui.WizardNode{Opcode: 15, Types: [10]uint32{mapload.ScriptPlayer, mapload.ScriptX, mapload.ScriptY}, Values: [10]uint32{1, 10, 12}},
			Right: ui.WizardNode{Opcode: 0x10002, Types: [10]uint32{mapload.ScriptInt}, Values: [10]uint32{0}},
			Cmp:   0,
		}},
		Actions: []ui.WizardNode{{
			Opcode: scriptprofile.TeleportCellOp,
			Types:  [10]uint32{mapload.ScriptX, mapload.ScriptY, mapload.ScriptX, mapload.ScriptY},
			Values: [10]uint32{10, 12, 24, 28},
		}},
	}
	if _, err := s.WriteTrigger(plan); !errors.As(err, &need) || need.Fold {
		t.Fatalf("WriteTrigger teleport on plain map answered %v, want ScriptFormNeeded", err)
	}
	if !bytes.Equal(s.model.Bytes(), plain) {
		t.Fatal("refused WriteTrigger mutated map bytes")
	}

	// Extend and write the trigger; .alm save is refused, .almx succeeds.
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	d, err := s.WriteTrigger(plan)
	if err != nil {
		t.Fatal(err)
	}
	if !d.AgainromOnly || !d.Script.Extended {
		t.Fatalf("after writing teleport: againrom=%v extended=%v", d.AgainromOnly, d.Script.Extended)
	}
	if _, err := s.SaveAs(filepath.Join(t.TempDir(), "refused.alm")); err == nil {
		t.Fatal("saving a map with teleport as .alm succeeded")
	}
	almxPath := filepath.Join(t.TempDir(), "pad.almx")
	if _, err := s.SaveAs(almxPath); err != nil {
		t.Fatal(err)
	}

	// Reopen .almx and verify the action with both cells reads back.
	re, err := s.inspector.Open(almxPath)
	if err != nil {
		t.Fatal(err)
	}
	if !re.AgainromOnly || !re.Script.Extended {
		t.Fatalf("reopened .almx: againrom=%v extended=%v", re.AgainromOnly, re.Script.Extended)
	}
	m, err := alm.Open(re.SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	sc, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	var found *alm.ScriptNode
	for i := range sc.Actions {
		if sc.Actions[i].Opcode == scriptprofile.TeleportCellOp {
			found = &sc.Actions[i]
			break
		}
	}
	if found == nil || found.Type != [10]uint32{5, 6, 5, 6} || found.Value != [10]uint32{10, 12, 24, 28} {
		t.Fatalf("reopened teleport node %+v", found)
	}

	// Compile and run in sim, then round-trip the current script program JSON.
	prog, _, err := mapload.CompileScript(m, mapload.ScriptRefs{})
	if err != nil || !prog.Extended() {
		t.Fatalf("CompileScript: %v extended=%v", err, prog != nil && prog.Extended())
	}
	u1 := sim.Entity{ID: 1, Owner: sim.SelfSlot, HP: 50, MaxHP: 50, X: 10, Y: 12, Domain: sim.DomainGround}
	u2 := sim.Entity{ID: 2, Owner: sim.SelfSlot, HP: 50, MaxHP: 50, X: 8, Y: 12, Speed: 16, Domain: sim.DomainGround}
	w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, nil, []sim.Entity{u1, u2}, prog)
	if err != nil {
		t.Fatal(err)
	}
	for range 32 {
		sim.Step(w, nil)
	}
	if e1, _ := w.Entity(1); e1.X != 24 || e1.Y != 28 {
		t.Fatalf("unit 1 at (%d,%d), want (24,28)", e1.X, e1.Y)
	}
	// Current SAV script-program payload round-trips cleanly.
	rawProg, err := json.Marshal(captureCurrentScriptProgram(w))
	if err != nil {
		t.Fatal(err)
	}
	var restored currentScriptProgram
	if err := json.Unmarshal(rawProg, &restored); err != nil {
		t.Fatal(err)
	}
	if _, err := restored.compile(); err != nil {
		t.Fatal(err)
	}
}
