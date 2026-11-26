package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/scriptprofile"
	"againrom/pkg/sim"
)

// runsOn reports whether a script of one node of op compiles with no gap,
// on the extended machine where extended, else on the original.
func runsOn(t *testing.T, op scriptprofile.OpID, extended bool) bool {
	t.Helper()
	var checks []sim.ScriptCheck
	var instants []sim.ScriptInstant
	if op.Condition {
		checks = []sim.ScriptCheck{{Op: int32(op.ID)}}
	} else {
		instants = []sim.ScriptInstant{{Op: int32(op.ID)}}
	}
	s, err := sim.NewScript(checks, instants, nil)
	if extended {
		s, err = sim.NewExtendedScript(sim.ScriptROM1, 100, checks, instants, nil)
	}
	if err != nil {
		t.Fatalf("op %#x: %v", op.ID, err)
	}
	return len(s.Unsupported()) == 0
}

// The simulation's check and instant gates read the row of an
// extended-record operation: the extended machine runs every row, the
// original machine a row marked Original; a test-only row is gated the same
// way with no other code.
func TestScriptGatesReadTheExtendedRow(t *testing.T) {
	check := func(rows []scriptprofile.ExtendedOp) {
		t.Helper()
		for _, o := range rows {
			if o.Interactive {
				continue
			}
			for _, op := range o.Ops {
				if ext, orig := runsOn(t, op, true), runsOn(t, op, false); !ext || orig != o.Original {
					t.Errorf("row %s op %#x: extended %v, original %v, want true and %v", o.Name, op.ID, ext, orig, o.Original)
				}
			}
		}
	}
	check(scriptprofile.ExtendedOps)
	for op, want := range map[scriptprofile.OpID]bool{{true, scriptprofile.UnitLeftCellOp}: false, {true, scriptprofile.ReadGlobalOp}: true,
		{false, scriptprofile.SetGlobalOp}: true, {false, scriptprofile.TeleportCellOp}: true} {
		if runsOn(t, op, false) != want {
			t.Errorf("op %#x on the original machine moved from %v", op.ID, want)
		}
	}
	defer func(rows []scriptprofile.ExtendedOp) { scriptprofile.ExtendedOps = rows }(scriptprofile.ExtendedOps)
	probe := scriptprofile.ExtendedOp{Name: "probe", Ops: []scriptprofile.OpID{{Condition: true, ID: 0x41fe}, {ID: 0x41ff}}}
	if runsOn(t, probe.Ops[0], true) || runsOn(t, probe.Ops[1], true) {
		t.Fatal("the probe opcodes run before their row exists")
	}
	scriptprofile.ExtendedOps = append(slices.Clone(scriptprofile.ExtendedOps), probe)
	check([]scriptprofile.ExtendedOp{probe})
}

func TestExtendedOpOriginalFlagGatesCompileDispatchAndTrace(t *testing.T) {
	defer func(rows []scriptprofile.ExtendedOp) { scriptprofile.ExtendedOps = rows }(scriptprofile.ExtendedOps)
	rows := slices.Clone(scriptprofile.ExtendedOps)
	for i := range rows {
		if rows[i].Name == "global" || rows[i].Name == "teleport" {
			rows[i].Original = false
		}
	}
	scriptprofile.ExtendedOps = rows

	checks := []sim.ScriptCheck{
		{Op: scriptprofile.ReadGlobalOp, Register: 0, Args: [10]int32{1}},
		{Op: sim.ScriptCheckConstant, Register: 1, Args: [10]int32{1}},
	}
	instants := []sim.ScriptInstant{
		{Op: scriptprofile.SetGlobalOp, Args: [10]int32{1, 99}},
		{Op: scriptprofile.AddGlobalOp, Args: [10]int32{2, 5}},
		{Op: scriptprofile.TeleportCellOp, Args: [10]int32{4, 4, 8, 8}},
	}
	eq := sim.ScriptPair{Left: 1, Right: 1, Cmp: sim.ScriptCmpEQ, Used: true}
	none := [4]int32{sim.ScriptNone, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}
	origTriggers := []sim.ScriptTrigger{{Pairs: [3]sim.ScriptPair{eq}, Instants: [4]int32{0, 1, 2, sim.ScriptNone}, Once: true, Latch: 0}}
	extTriggers := []sim.ScriptTrigger{{Instants: none, Once: true, Latch: 0, Conditions: []sim.ScriptPair{eq}, Actions: []int32{0, 1, 2}}}

	for _, dialect := range []sim.ScriptDialect{sim.ScriptROM1, sim.ScriptROM2} {
		for _, extended := range []bool{false, true} {
			name := "ROM1"
			if dialect == sim.ScriptROM2 {
				name = "ROM2"
			}
			if extended {
				name += "-extended"
			} else {
				name += "-original"
			}
			t.Run(name, func(t *testing.T) {
				var (
					s   *sim.Script
					err error
				)
				if extended {
					s, err = sim.NewExtendedScript(dialect, 100, checks, instants, extTriggers)
				} else if dialect == sim.ScriptROM2 {
					s, err = sim.NewROM2Script(checks, instants, origTriggers)
				} else {
					s, err = sim.NewScript(checks, instants, origTriggers)
				}
				if err != nil {
					t.Fatal(err)
				}
				if extended {
					if len(s.Unsupported()) != 0 {
						t.Fatalf("extended script reported gaps: %+v", s.Unsupported())
					}
				} else if len(s.Unsupported()) != 4 {
					t.Fatalf("original script reported %d gaps, want 4: %+v", len(s.Unsupported()), s.Unsupported())
				}

				unit := sim.Entity{ID: 1, Owner: 1, X: 4, Y: 4, HP: 50, MaxHP: 50}
				w, err := sim.NewScriptedWorld(7, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil, []sim.Entity{unit}, s)
				if err != nil {
					t.Fatal(err)
				}
				if err := w.SeatScriptGlobals([]sim.ScriptGlobal{{ID: 1, Value: 7}, {ID: 2, Value: 10}}); err != nil {
					t.Fatal(err)
				}
				var tr sim.ScriptTrace
				for range 16 {
					if got := sim.StepTraced(w, nil); len(got.Checks) > 0 {
						tr = got
						break
					}
				}
				if len(tr.Checks) != 2 || len(tr.Firings) != 1 || len(tr.Firings[0].Instants) != 3 {
					t.Fatalf("trace checks=%d firings=%+v", len(tr.Checks), tr.Firings)
				}

				wantGlobals := []sim.ScriptGlobal{{ID: 1, Value: 7}, {ID: 2, Value: 10}}
				wantX, wantY := int32(4), int32(4)
				wantReg0 := int32(0)
				wantOutcome := sim.ScriptInstantUnsupported
				if extended {
					wantGlobals = []sim.ScriptGlobal{{ID: 1, Value: 99}, {ID: 2, Value: 15}}
					wantX, wantY = 8, 8
					wantReg0 = 7
					wantOutcome = sim.ScriptInstantStateChanged
				}
				chk := tr.Checks[0]
				if chk.Dispatched != extended || chk.Wrote != extended || chk.Value != wantReg0 {
					t.Errorf("ReadGlobalOp check = %+v, want dispatched=%v wrote=%v value=%d", chk, extended, extended, wantReg0)
				}
				if !extended && (!chk.HasSilence || chk.Silence != sim.ScriptSilenceUnsupported) {
					t.Errorf("ReadGlobalOp silence = %+v, want ScriptSilenceUnsupported", chk)
				}
				if got := w.ScriptGlobals(); !reflect.DeepEqual(got, wantGlobals) {
					t.Errorf("globals = %v, want %v", got, wantGlobals)
				}
				e := w.Entities()[0]
				if e.X != wantX || e.Y != wantY {
					t.Errorf("unit position = (%d,%d), want (%d,%d)", e.X, e.Y, wantX, wantY)
				}
				for k, r := range tr.Firings[0].Instants {
					if r.Supported != extended || r.Outcome != wantOutcome {
						t.Errorf("instant %d (op %#x): Supported=%v Outcome=%v, want %v %v", k, r.Op, r.Supported, r.Outcome, extended, wantOutcome)
					}
				}
			})
		}
	}
}
