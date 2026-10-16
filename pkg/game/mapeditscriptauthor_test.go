package game

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func scriptWindowAdd(t *testing.T, e *ui.MapEditor, kind string) int {
	t.Helper()
	if err := e.Key("scripts"); err != nil {
		t.Fatal(err)
	}
	if err := e.Key("player"); err != nil {
		t.Fatal(err)
	}
	value := map[string]string{"Action": "0", "Condition": "1", "Trigger": "2"}[kind]
	_ = e.Key("backspace")
	e.TextInput(value)
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if e.Selected() < 0 || e.Document().Records[e.Selected()].Kind != kind {
		t.Fatalf("no new %s selected: %d %s", kind, e.Selected(), e.Message())
	}
	return e.Document().Records[e.Selected()].Index
}

func scriptWindowSet(t *testing.T, e *ui.MapEditor, kind string, index int, key, value string) {
	t.Helper()
	if !e.Select(recordOf(e.Document(), kind, index), false) {
		t.Fatalf("no %s %d", kind, index)
	}
	fields := e.Document().Edits.Fields(kind, index)
	at := -1
	for i, f := range fields {
		if f.Key == key {
			at = i
		}
	}
	if at < 0 {
		t.Fatalf("no %s %d field %s", kind, index, key)
	}
	if err := e.Key("field"); err != nil {
		t.Fatal(err)
	}
	for range at {
		if err := e.Key("tab"); err != nil {
			t.Fatal(err)
		}
	}
	old := strconv.FormatInt(fields[at].Value, 10)
	if fields[at].Text {
		old = fields[at].TextValue
	}
	for range []rune(old) {
		_ = e.Key("backspace")
	}
	e.TextInput(value)
	if err := e.Key("enter"); err != nil {
		t.Fatalf("%s %d %s=%s: %v", kind, index, key, value, err)
	}
}

func scriptWindowScenario(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	for range 3 {
		scriptWindowAdd(t, e, "Condition")
	}
	scriptWindowSet(t, e, "Condition", 1, "opcode", "19")
	scriptWindowSet(t, e, "Condition", 1, "value0", "99")
	scriptWindowSet(t, e, "Condition", 2, "value0", "3")
	for range 4 {
		scriptWindowAdd(t, e, "Action")
	}
	scriptWindowSet(t, e, "Action", 0, "value0", "99")
	scriptWindowSet(t, e, "Action", 0, "opcode", "3")
	scriptWindowSet(t, e, "Action", 0, "type1", "1")
	scriptWindowSet(t, e, "Action", 0, "value1", "2")
	scriptWindowSet(t, e, "Action", 1, "value0", "99")
	scriptWindowSet(t, e, "Action", 1, "opcode", "8")
	scriptWindowSet(t, e, "Action", 3, "opcode", "4")
	for range 2 {
		scriptWindowAdd(t, e, "Trigger")
	}
	for _, v := range []struct {
		i    int
		k, v string
	}{
		{0, "left0", "2"}, {0, "right0", "3"}, {0, "cmp0", "3"},
		{0, "left1", "1"}, {0, "right1", "1"}, {0, "cmp1", "0"},
		{0, "action0", "1"}, {0, "action1", "2"}, {0, "action2", "3"},
		{1, "left0", "2"}, {1, "right0", "3"}, {1, "cmp0", "0"}, {1, "action0", "4"},
	} {
		scriptWindowSet(t, e, "Trigger", v.i, v.k, v.v)
	}
}

func scriptAuthorWorld(t *testing.T, b []byte) *sim.World {
	t.Helper()
	m, err := alm.Open(b)
	if err != nil {
		t.Fatal(err)
	}
	program, report, err := mapload.CompileScript(m, mapload.ScriptRefs{})
	if err != nil || len(report.OmittedActions)+len(report.OmittedChecks)+len(program.Unsupported()) != 0 {
		t.Fatalf("compile %v %+v", err, report)
	}
	w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: int32(m.Width), Height: int32(m.Height)}, sim.ModeCanonical, nil, nil, program)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestScriptWindowAuthorsACompoundMissionWithOrderedActions(t *testing.T) {
	b := synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12)})
	e, path := openGoalMap(t, b)
	scriptWindowScenario(t, e)
	authored := append([]byte(nil), e.Document().SourceBytes...)
	if _, err := e.Document().Edits.SaveAs(path + ".saved.alm"); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path + ".saved.alm")
	if err != nil || !bytes.Equal(saved, authored) {
		t.Fatalf("save %v", err)
	}
	reopened, _ := openGoalMap(t, saved)
	w := scriptAuthorWorld(t, reopened.Document().SourceBytes)
	if w.ScriptRegister(99) != 0 {
		t.Fatal("nonzero initial variable")
	}
	for i := 0; i < 100 && w.ScriptRegister(99) == 0; i++ {
		sim.Step(w, nil)
	}
	if w.ScriptRegister(99) != 3 || w.Outcome() != sim.OutcomeUndecided {
		t.Fatalf("ordered stage: variable=%d outcome=%v", w.ScriptRegister(99), w.Outcome())
	}
	for i := 0; i < 100 && w.Outcome() == sim.OutcomeUndecided; i++ {
		sim.Step(w, nil)
	}
	if w.Outcome() != sim.OutcomeWon {
		t.Fatalf("result=%v", w.Outcome())
	}
	for _, c := range []struct{ name, key, value string }{
		{"false compound clause", "cmp1", "1"},
		{"reversed actions first", "action0", "2"},
	} {
		t.Run(c.name, func(t *testing.T) {
			control, _ := openGoalMap(t, saved)
			scriptWindowSet(t, control, "Trigger", 0, c.key, c.value)
			if c.name == "reversed actions first" {
				scriptWindowSet(t, control, "Trigger", 0, "action1", "1")
			}
			cw := scriptAuthorWorld(t, control.Document().SourceBytes)
			for range 100 {
				sim.Step(cw, nil)
			}
			if cw.Outcome() != sim.OutcomeUndecided {
				t.Fatal("loss control won")
			}
		})
	}
	control, _ := openGoalMap(t, saved)
	scriptWindowSet(t, control, "Condition", 2, "value0", "4")
	for i := range 2 {
		r := control.Document().Records[recordOf(control.Document(), "Trigger", i)]
		seen := false
		for _, line := range r.Lines {
			if strings.Contains(line.Text, "Par0 Int/Enum [tag 1] = 4") {
				seen = true
			}
		}
		if !seen {
			t.Fatalf("trigger%d did not show the changed shared condition", i)
		}
	}
	cw := scriptAuthorWorld(t, control.Document().SourceBytes)
	for range 100 {
		sim.Step(cw, nil)
	}
	if cw.Outcome() != sim.OutcomeUndecided || cw.ScriptRegister(99) != 3 {
		t.Fatal("shared condition control did not change mission result")
	}
}

func TestScriptVariablesAndUnreferencedConditionsDeleteSafely(t *testing.T) {
	e, _ := openGoalMap(t, synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12)}))
	scriptWindowAdd(t, e, "Condition")
	scriptWindowAdd(t, e, "Action")
	scriptWindowSet(t, e, "Action", 0, "value0", "0")
	scriptWindowSet(t, e, "Action", 0, "opcode", "3")
	s := e.Document().Edits
	before := append([]byte(nil), e.Document().SourceBytes...)
	if _, err := s.Delete("Condition", 0); err == nil {
		t.Fatal("deleted variable preset while an action names its register")
	}
	if !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("refusal lost preset")
	}
	if _, err := s.Delete("Action", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Delete("Condition", 0); err != nil {
		t.Fatal(err)
	}
	_ = e.Key("undo")
	_ = e.Key("undo")
	if !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("variable delete Undo lost input")
	}
	e, _ = openGoalMap(t, synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12)}))
	for range 3 {
		scriptWindowAdd(t, e, "Condition")
	}
	scriptWindowAdd(t, e, "Action")
	scriptWindowSet(t, e, "Action", 0, "opcode", "4")
	scriptWindowAdd(t, e, "Trigger")
	for _, v := range []struct{ k, v string }{{"left0", "2"}, {"right0", "3"}, {"action0", "1"}} {
		scriptWindowSet(t, e, "Trigger", 0, v.k, v.v)
	}
	if _, err := e.Document().Edits.Delete("Condition", 0); err != nil {
		t.Fatal(err)
	}
	_ = e.Key("undo")
	_ = e.Key("redo")
	w := scriptAuthorWorld(t, e.Document().SourceBytes)
	for range 50 {
		sim.Step(w, nil)
	}
	if w.Outcome() != sim.OutcomeWon {
		t.Fatal("deleting unreferenced lower condition changed surviving ID references")
	}
}

func TestScriptAuthoringReferencesRefuseWithoutMutation(t *testing.T) {
	e, _ := openGoalMap(t, synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12)}))
	scriptWindowScenario(t, e)
	s := e.Document().Edits.(*mapInspectionEdits)
	before := s.model.Bytes()
	for _, c := range []struct {
		kind, key string
		i         int
		v         int64
	}{
		{"Action", "id", 1, 1}, {"Action", "id", 0, 50}, {"Action", "opcode", 0, 0xabc},
		{"Trigger", "left0", 0, 404}, {"Trigger", "action0", 0, 404},
		{"Trigger", "cmp0", 0, 6}, {"Trigger", "once", 0, 2}, {"Action", "value0", 0, 1},
	} {
		if _, err := s.SetField(c.kind, c.i, c.key, c.v); err == nil {
			t.Fatalf("accepted %+v", c)
		}
	}
	for _, kind := range []string{"Action", "Condition"} {
		if _, err := s.Delete(kind, 0); err == nil {
			t.Fatal("deleted referenced " + kind)
		}
	}
	if !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("refusal mutated map")
	}
	scriptWindowSet(t, e, "Trigger", 1, "order", "0")
	m, _ := s.model.Map()
	sc, _ := m.Script()
	if sc.Triggers[0].Acts[0] != 4 {
		t.Fatal("trigger order unchanged")
	}
	_ = e.Key("undo")
	if !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("reorder Undo lost bytes")
	}
	_ = e.Key("redo")
	_ = e.Key("undo")
	if _, err := s.Delete("Trigger", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Delete("Action", 3); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetField("Action", 2, "id", 4); err == nil {
		t.Fatal("renamed referenced message")
	}
	if _, err := s.SetField("Action", 0, "value0", 100); err == nil {
		t.Fatal("out-of-range variable")
	}
}

func TestScriptUnknownRecordIsCarriedAndKnownChoicesShareTheCompiler(t *testing.T) {
	b := synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: goalScript([]scriptNode{{label: "Raw", opcode: 0xabc, id: 7, values: []uint32{19}, tags: []uint32{88}}}, nil, nil)})
	e, _ := openGoalMap(t, b)
	if !strings.Contains(e.Document().Records[recordOf(e.Document(), "Action", 0)].Warning, "Unknown") {
		t.Fatal("unknown input not labelled")
	}
	scriptWindowAdd(t, e, "Condition")
	after, _ := alm.Open(e.Document().SourceBytes)
	sc, _ := after.Script()
	if len(sc.Actions) != 1 || sc.Actions[0].Opcode != 0xabc || sc.Actions[0].Type[0] != 88 || sc.Actions[0].Value[0] != 19 {
		t.Fatal("unknown input rewritten")
	}
	for _, condition := range []bool{false, true} {
		for _, op := range mapload.ScriptOperations(condition) {
			if op.Opcode < 0x10000 && !sim.ScriptOperationSupported(condition, int32(op.Opcode)) {
				t.Fatalf("unsupported authoring choice %+v", op)
			}
		}
	}
}

func TestScriptWindowOnceAndRepeatHaveDifferentLiveEffects(t *testing.T) {
	for _, once := range []string{"0", "1"} {
		e, _ := openGoalMap(t, synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12)}))
		scriptWindowAdd(t, e, "Condition")
		scriptWindowAdd(t, e, "Action")
		scriptWindowSet(t, e, "Action", 0, "value0", "99")
		scriptWindowSet(t, e, "Action", 0, "opcode", "8")
		scriptWindowAdd(t, e, "Trigger")
		for _, v := range []struct{ k, v string }{{"left0", "1"}, {"right0", "1"}, {"action0", "1"}, {"once", once}} {
			scriptWindowSet(t, e, "Trigger", 0, v.k, v.v)
		}
		w := scriptAuthorWorld(t, e.Document().SourceBytes)
		for range 100 {
			sim.Step(w, nil)
		}
		if once == "1" && w.ScriptRegister(99) != 1 {
			t.Fatalf("once register=%d", w.ScriptRegister(99))
		}
		if once == "0" && w.ScriptRegister(99) <= 1 {
			t.Fatalf("repeat register=%d", w.ScriptRegister(99))
		}
	}
}

func TestScriptWindowEditsBothDropActionsIndependently(t *testing.T) {
	e, _ := openGoalMap(t, synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12)}))
	for i := range 2 {
		scriptWindowAdd(t, e, "Action")
		for _, v := range []struct{ k, v string }{{"opcode", "65538"}, {"type0", "5"}, {"value0", strconv.Itoa(7 + i*2)}, {"type1", "6"}, {"value1", strconv.Itoa(8 + i*2)}} {
			scriptWindowSet(t, e, "Action", i, v.k, v.v)
		}
	}
	m, _ := alm.Open(e.Document().SourceBytes)
	want := []mapload.Cell{{X: 7, Y: 8}, {X: 9, Y: 10}}
	got := mapload.DropCells(m)
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("drops %v", got)
	}
	before := append([]byte(nil), e.Document().SourceBytes...)
	if _, err := e.Document().Edits.SetField("Action", 1, "value0", 32); err == nil {
		t.Fatal("out-of-map second drop accepted")
	}
	if !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("refused drop changed bytes")
	}
	scriptWindowSet(t, e, "Action", 1, "value0", "11")
	_ = e.Key("undo")
	if !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("drop Undo lost first/second action")
	}
}

func TestScriptAndWorkspaceEditsShareOneHistory(t *testing.T) {
	_, e := workspaceFixture(t)
	scriptWindowAdd(t, e, "Condition")
	condition := bytes.Clone(e.Document().SourceBytes)
	workspacePage(t, e, "event01.txt", "First stage")
	scriptWindowSet(t, e, "Condition", 0, "value0", "7")
	changed := bytes.Clone(e.Document().SourceBytes)
	workspacePage(t, e, "event01.txt", "Second stage")
	s := e.Document().Edits.(*mapInspectionEdits)
	for _, step := range []struct {
		key, body string
		image     []byte
	}{
		{"undo", "First stage", changed},
		{"undo", "First stage", condition},
		{"undo", "", condition},
		{"redo", "First stage", condition},
		{"redo", "First stage", changed},
		{"redo", "Second stage", changed},
	} {
		if err := e.Key(step.key); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(e.Document().SourceBytes, step.image) || string(s.workspace.Pages["event01.txt"]) != step.body {
			t.Fatalf("%s mixed history lost map/page state", step.key)
		}
	}
	before := bytes.Clone(e.Document().SourceBytes)
	if _, err := s.SetField("Condition", 0, "type0", 40); err == nil {
		t.Fatal("invalid parameter kind accepted")
	}
	if !bytes.Equal(before, e.Document().SourceBytes) || string(s.workspace.Pages["event01.txt"]) != "Second stage" {
		t.Fatal("refused script edit changed workspace")
	}
	_ = e.Key("undo")
	if string(s.workspace.Pages["event01.txt"]) != "First stage" {
		t.Fatal("refusal consumed a history slot")
	}
}
