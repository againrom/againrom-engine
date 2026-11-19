//go:build release

package game

import (
	"fmt"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/ui"
)

// Every check and action of a second-game map is a named arm or an explicit
// "not decoded" row; the register limit is the second game's.
func TestReleaseSecondMapScriptRowsAreNamedOrNotDecoded(t *testing.T) {
	x := installedInspector(t)
	if x.installGame() != base.GameROM2 {
		t.Skip("requires a second-game install")
	}
	var maps, triggers, named, undecoded, compiler int
	for _, entry := range x.Maps {
		source := ui.MapSourceKey(ui.SourceLoose, entry.Source)
		if entry.FromArchive {
			source = ui.MapSourceKey(ui.SourceScenario, entry.Source)
		}
		d, err := x.Open(source)
		if err != nil || d.Script == nil {
			t.Fatalf("%s: open %v, script %v", entry.Source, err, d != nil && d.Script != nil)
		}
		maps++
		sc := d.Script
		if p := sc.Profile(); p != scriptprofile.Second || p.Registers != 1000 {
			t.Fatalf("%s: profile %+v, want the second game's", entry.Source, p)
		}
		if !strings.HasSuffix(d.ScriptNotice, "/ 1000 registers") && d.ScriptNotice != "No authored triggers" {
			t.Errorf("%s: notice %q lacks the second game's register limit", entry.Source, d.ScriptNotice)
		}
		triggers += len(sc.Triggers)
		for _, family := range []struct {
			condition bool
			nodes     []ui.InspectionScriptNode
			ops       []ui.InspectionScriptOperation
		}{{true, sc.Conditions, sc.Catalog.Checks}, {false, sc.Actions, sc.Catalog.Actions}} {
			for _, n := range family.nodes {
				line := d.Records[n.Record].Lines[0].Text
				var op *ui.InspectionScriptOperation
				for i := range family.ops {
					if family.ops[i].Opcode == n.Opcode {
						op = &family.ops[i]
					}
				}
				switch {
				case op != nil && op.Named:
					named++
					if n.Opcode == mapload.ScriptConstant {
						compiler++
						if !strings.Contains(line, "Compiler parameter") {
							t.Errorf("%s: row %q is not a compiler parameter", entry.Source, line)
						}
					}
				case op == nil && strings.HasSuffix(line, fmt.Sprintf("Operation %d, not decoded", n.Opcode)):
					undecoded++
				default:
					t.Errorf("%s: opcode %d is neither named nor not decoded: %q", entry.Source, n.Opcode, line)
				}
			}
		}
	}
	if maps == 0 {
		t.Fatal("no maps")
	}
	t.Logf("PASS %s: %d maps, %d triggers, %d named rows (%d compiler parameters), %d not-decoded rows", installLanguage(x), maps, triggers, named, compiler, undecoded)
}

// The first game's window keeps its profile, its register limit and its names.
func TestReleaseFirstMapScriptKeepsItsProfile(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	d, err := x.Open("scenario/10.alm")
	if err != nil || d.Script == nil {
		t.Fatal("open", err)
	}
	sc := d.Script
	if p := sc.Profile(); p != scriptprofile.First || p.Registers != 100 {
		t.Fatalf("profile %+v, want the first game's", p)
	}
	if !strings.HasSuffix(d.ScriptNotice, "/ 100 registers") && d.ScriptNotice != "No authored triggers" {
		t.Errorf("notice %q", d.ScriptNotice)
	}
	for _, condition := range []bool{true, false} {
		ops, want := sc.Catalog.Actions, mapload.ScriptOperations(false)
		if condition {
			ops, want = sc.Catalog.Checks, mapload.ScriptOperations(true)
		}
		if len(ops) != len(want) {
			t.Fatalf("condition %v: %d arms, want %d", condition, len(ops), len(want))
		}
		for i, op := range ops {
			if op.Opcode != want[i].Opcode || op.Named != mapload.ScriptOperationNamed(condition, op.Opcode) {
				t.Errorf("condition %v arm %d: %+v", condition, i, op)
			}
		}
	}
}
