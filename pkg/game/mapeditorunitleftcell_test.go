package game

import (
	"bytes"
	"errors"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestUnitLeftCellEditorRoundTrip(t *testing.T) {
	for _, lang := range scriptAddLangs {
		t.Run(lang.name, func(t *testing.T) {
			e, _, word := scriptAddEditor(t, lang, triggerMap(t))
			e.Fit()
			wizNewTrigger(t, e, "Departure")
			wizPress(t, e, "", 0)
			trigChoose(t, e, wizField(t, e, "c0.item"), word("editor.wizard.c.leftcell"))
			trigChoose(t, e, wizField(t, e, "c0.p0"), "15,14")
			if got := trigLines(e); !strings.Contains(got, "15,14") {
				t.Fatal("the list choice has no starting cell", got)
			}
			wizPress(t, e, "c0.p0", 0)
			clickCell(e, image.Pt(12, 14))
			if got := trigLines(e); !strings.Contains(got, "12,14") {
				t.Fatal("the map pick has no starting cell", got)
			}
			wizPress(t, e, "+action", 0)
			trigChoose(t, e, wizField(t, e, "a0.item"), word("editor.wizard.a.win"))
			r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 0})
			trigClick(t, e, r, ok, "Apply")
			r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlPromptButton, N: 0})
			trigClick(t, e, r, ok, "Extend")
			host := formHost(e)
			path := filepath.Join(t.TempDir(), "departure.almx")
			if _, err := host.SaveAs(path); err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			sc := wizScript(t, b)
			tr := sc.Triggers[len(sc.Triggers)-1]
			n := wizNode(t, sc.Conditions, tr.Left[0])
			if !sc.Extended || n.Opcode != 0x4105 || n.Type != [10]uint32{4, 5, 6} || n.Value != [10]uint32{10, 12, 14} ||
				tr.Cmp[0] != uint32(sim.ScriptCmpEQ) || wizNode(t, sc.Conditions, tr.Right[0]).Value[0] != 1 {
				t.Fatalf("saved departure: %+v, %+v", tr, n)
			}
			reopened, _, _ := scriptAddEditor(t, lang, b)
			reopened.Select(trigRecord(t, reopened, "Trigger", len(sc.Triggers)-1), true)
			if lines := trigLines(reopened); !strings.Contains(lines, word("editor.wizard.c.leftcell")) || !strings.Contains(lines, "12,14") {
				t.Fatal("reopened row", lines)
			}
			if _, err := host.FoldScript(); err == nil || !bytes.Equal(b, host.model.Bytes()) {
				t.Fatal("fold did not preserve the extended map", err)
			}
			moved, err := host.Move(ui.KindUnit, 0, image.Pt(13, 14))
			if err != nil {
				t.Fatal(err)
			}
			reopened, _, _ = scriptAddEditor(t, lang, moved.SourceBytes)
			reopened.Select(trigRecord(t, reopened, "Trigger", len(sc.Triggers)-1), true)
			if lines := trigLines(reopened); !strings.Contains(lines, "13,14") || !strings.Contains(lines, word("editor.wizard.startcell", 12, 14)) {
				t.Fatal("the row lost its recorded cell after a placement edit", lines)
			}
		})
	}
}

func TestUnitLeftCellInAPlainMapRequiresConversion(t *testing.T) {
	n := scriptNode{label: "Departure", opcode: 0x4105, id: 1, values: []uint32{91, 15, 16}, tags: []uint32{4, 5, 6}}
	plain := synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Units: extendedUnits, Type7Payload: scriptPayload(nil, []scriptNode{n})})
	e, _ := openGoalMap(t, plain)
	host := formHost(e)
	out := filepath.Join(t.TempDir(), "plain.alm")
	if _, err := host.SaveAs(out); !errors.As(err, new(*ui.ScriptFormNeeded)) {
		t.Fatalf("plain save: %v", err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) || !bytes.Equal(plain, host.model.Bytes()) {
		t.Fatal("conversion request wrote the map", err)
	}
}

func TestUnitLeftCellMissionSave(t *testing.T) {
	actions := []scriptNode{{label: "Drop", opcode: dropActionOpcode, id: 1, values: []uint32{16, 16}, tags: []uint32{5, 6}},
		{label: "Count", opcode: 8, id: 2, values: []uint32{7}, tags: []uint32{1}}}
	conditions := []scriptNode{{label: "Departure", opcode: 0x4105, id: 1, values: []uint32{91, 15, 16}, tags: []uint32{4, 5, 6}},
		{label: "One", opcode: 0x10002, id: 2, values: []uint32{1}, tags: []uint32{1}}}
	plain := synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Units: extendedUnits, Type7Payload: scriptPayload(actions, conditions)})
	m, err := alm.AppendScriptRecord(plain, alm.ScriptRecord{Variables: 100, Triggers: []alm.ScriptTrigger{{Name: "Left", Once: 1, Left: []uint32{1}, Right: []uint32{2}, Acts: []uint32{2}}}})
	if err != nil {
		t.Fatal(err)
	}
	f := extendedScriptFront(t, m)
	w := openExtendedMission(t, f, 32)
	if w.ScriptLatched(0) || w.ScriptRegister(7) != 0 {
		t.Fatal("already fired before departure")
	}
	roundTrip := func() *FrontEnd {
		s, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(s, label)
		if err != nil {
			t.Fatal(err)
		}
		back := extendedScriptFront(t, m)
		open, town, err := back.RestoreOriginal(raw)
		if err == nil && !town {
			err = back.App("departure").OpenMission(open)
		}
		if err != nil || town || back.live.world.Hash() != w.Hash() {
			t.Fatal("cold LOAD lost the mission state", town, err)
		}
		return back
	}
	back := roundTrip()
	cmds := []sim.Command{sim.MoveTo(w.Script().Checks()[0].Unit, sim.CellPoint{X: 15, Y: 17})}
	for range 96 {
		sim.Step(w, cmds)
		sim.Step(back.live.world, cmds)
		cmds = nil
		if w.Hash() != back.live.world.Hash() {
			t.Fatal("resumed departure diverged")
		}
	}
	if !w.ScriptLatched(0) || w.ScriptRegister(7) != 1 {
		t.Fatalf("departure fired %d times", w.ScriptRegister(7))
	}
	back = roundTrip()
	for range 32 {
		sim.Step(back.live.world, nil)
	}
	if back.live.world.ScriptRegister(7) != 1 {
		t.Fatal("once trigger fired again after LOAD")
	}
}
