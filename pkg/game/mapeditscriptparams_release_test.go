package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func parameterField(n ui.InspectionScriptNode, slot int) int {
	field := 2
	for p := 0; p < slot; p++ {
		if n.Types[p] == 0 {
			continue
		}
		if n.Types[p] == 5 && p+1 < len(n.Types) && n.Types[p+1] == 6 {
			p++
		}
		field++
	}
	return field
}

func parameterTag(tag uint32) bool {
	return tag == 2 || tag == 3 || tag == 4 || tag == 8 || tag == 9 || tag == ui.ScriptVariableParameter
}

func chooseParameterValue(t *testing.T, e *ui.MapEditor, field int, value uint32, targets []ui.InspectionScriptTarget) {
	t.Helper()
	r, ok := e.FieldRect(field)
	for k := 0; !ok && k < 30; k++ {
		e.ScrollDetails(1)
		r, ok = e.FieldRect(field)
	}
	trigClick(t, e, r, ok, "parameter")
	e.ScrollDropdown(-len(e.DropdownChoices()))
	row := -1
	for i, target := range targets {
		if target.Value == value {
			row = i
			break
		}
	}
	if row < 0 {
		for i, text := range e.DropdownChoices() {
			if text == strconv.FormatUint(uint64(value), 10) {
				row = i
				break
			}
		}
	}
	if row < 0 {
		t.Fatalf("no value %d in field %d choices %q", value, field, e.DropdownChoices())
	}
	for k := 0; k < 800; k++ {
		if r, ok := e.DropdownRowRect(row); ok {
			trigClick(t, e, r, ok, "parameter value")
			return
		}
		e.ScrollDropdown(1)
	}
	t.Fatalf("choice %d never visible", row)
}

func parameterOutput(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("AGAINROM_SCRIPT_PARAMS_DIR")
	if dir == "" {
		return t.TempDir()
	}
	dir = filepath.Join(dir, strings.ReplaceAll(t.Name(), "/", "-"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestReleaseMapEditorParameterCurrentPicksPreserveBytes(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	dir := parameterOutput(t)
	counts := map[uint32]int{}
	maps := 0
	for _, key := range installedMapKeys(x) {
		e := x.Editor()
		if err := e.Open(key); err != nil {
			t.Fatal(key, err)
		}
		e.Layout(1280, 800)
		closeMapPanel(t, e)
		sc := e.Document().Script
		if sc == nil {
			continue
		}
		used := map[uint32]bool{}
		for _, nodes := range [][]ui.InspectionScriptNode{sc.Actions, sc.Conditions} {
			for _, node := range nodes {
				if node.AddressesVariable {
					used[node.VariableRegister] = true
				}
			}
		}
		wantNew := uint32(len(sc.Conditions))
		for wantNew < 100 && used[wantNew] {
			wantNew++
		}
		foundNew := false
		for _, target := range sc.Targets[ui.ScriptVariableParameter] {
			if target.Record != -1 {
				continue
			}
			foundNew = true
			if target.Value != wantNew || target.Value >= 100 || uint64(target.Value) < uint64(len(sc.Conditions)) || used[target.Value] {
				t.Fatalf("%s New variable register %d, want free slot %d above %d conditions", key, target.Value, wantNew, len(sc.Conditions))
			}
			t.Logf("New variable %s register=%d conditions=%d", key, target.Value, len(sc.Conditions))
		}
		if !foundNew {
			if wantNew < 100 {
				t.Fatalf("%s New variable omits free slot %d", key, wantNew)
			}
			t.Logf("New variable %s unavailable conditions=%d", key, len(sc.Conditions))
		}
		for _, target := range sc.Targets[8] {
			number, _, ok := strings.Cut(target.ChoiceText(), ": ")
			if !ok || number != strconv.FormatUint(uint64(target.Value), 10) {
				t.Fatalf("%s item row %q writes %d", key, target.ChoiceText(), target.Value)
			}
		}
		if key == "scenario/60.alm" {
			for _, tag := range []uint32{4, 2, 3, 9, 10, 8} {
				t.Logf("choice population %s kind=%d count=%d", key, tag, len(sc.Targets[tag]))
			}
		}
		before := bytes.Clone(e.Document().SourceBytes)
		picked := 0
		for _, nodes := range [][]ui.InspectionScriptNode{sc.Actions, sc.Conditions} {
			for _, n := range nodes {
				for p := range n.Types {
					tag := scriptParameterKind(e.Document().Records[n.Record].Kind == "Condition", n.Opcode, n.Types, p)
					if !parameterTag(tag) {
						continue
					}
					e.Select(n.Record, false)
					chooseParameterValue(t, e, parameterField(n, p), n.Values[p], sc.Targets[tag])
					r, ok := e.PopupButtonRect(0)
					trigClick(t, e, r, ok, "Apply current")
					if !bytes.Equal(before, e.Document().SourceBytes) || e.Dirty() {
						t.Fatalf("%s record %d Par%d current pick changed bytes", key, n.Record, p)
					}
					counts[tag]++
					picked++
				}
			}
		}
		if picked == 0 {
			continue
		}
		path := filepath.Join(dir, fmt.Sprintf("current-%03d.alm", maps))
		if _, err := e.Document().Edits.SaveAs(path); err != nil {
			t.Fatal(key, err)
		}
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, before) {
			t.Fatal(key, "Save As differs", err)
		}
		maps++
	}
	for _, tag := range []uint32{2, 3, 4, 8, 9, ui.ScriptVariableParameter} {
		if counts[tag] == 0 {
			t.Errorf("kind %d not driven", tag)
		}
	}
	t.Logf("installed current picks, maps=%d counts=%v", maps, counts)
}

func TestReleaseMapEditorParameterChangedPicksSaveReopenCompile(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	dir := parameterOutput(t)
	done := map[uint32]bool{}
	for _, key := range installedMapKeys(x) {
		e := x.Editor()
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		e.Layout(1280, 800)
		closeMapPanel(t, e)
		sc := e.Document().Script
		if sc == nil {
			continue
		}
		for _, nodes := range [][]ui.InspectionScriptNode{sc.Actions, sc.Conditions} {
			for _, n := range nodes {
				for p := range n.Types {
					tag := scriptParameterKind(e.Document().Records[n.Record].Kind == "Condition", n.Opcode, n.Types, p)
					if !parameterTag(tag) || done[tag] {
						continue
					}
					for _, target := range sc.Targets[tag] {
						if tag == ui.ScriptVariableParameter {
							m, _ := objectEdits(t, e).model.Map()
							wire, _ := m.Script()
							if scriptRegisterAllowed(wire, target.Value) != nil {
								continue
							}
						}
						if target.Value == n.Values[p] || tag == 4 && target.Value >= 10001 || tag == 2 && target.Value == 0 {
							continue
						}
						e.Select(n.Record, false)
						chooseParameterValue(t, e, parameterField(n, p), target.Value, sc.Targets[tag])
						r, ok := e.PopupButtonRect(0)
						trigClick(t, e, r, ok, "Apply different")
						if bytes.Equal(e.Document().SourceBytes, objectEdits(t, e).saved) {
							t.Fatal("changed pick did not apply", tag, trigLines(e))
						}
						path := filepath.Join(dir, fmt.Sprintf("changed-%d.alm", tag))
						if _, err := e.Document().Edits.SaveAs(path); err != nil {
							t.Fatal(key, tag, err)
						}
						cold := x.Editor()
						if err := cold.Open(path); err != nil {
							t.Fatal(err)
						}
						m, err := alm.Open(cold.Document().SourceBytes)
						if err != nil {
							t.Fatal(err)
						}
						wire, _ := m.Script()
						raw := wire.Actions
						kind := e.Document().Records[n.Record].Kind
						if kind == "Condition" {
							raw = wire.Conditions
						}
						index := e.Document().Records[n.Record].Index
						if raw[index].Value[p] != target.Value {
							t.Fatal("cold value", raw[index].Value[p], target.Value)
						}
						if _, _, err := mapload.CompileScript(m, mapload.ScriptRefs{Units: mapload.ScriptUnits(m, nil), Structures: mapload.ScriptStructures(m)}); err != nil {
							t.Fatal(err)
						}
						parameterCompiledValue(t, m, raw[index], kind == "Condition", tag, target.Value)
						t.Logf("kind %d %s %s[%d] Par%d %d -> %d; cold value and compile PASS", tag, key, kind, index, p, n.Values[p], target.Value)
						done[tag] = true
						break
					}
				}
			}
		}
		if len(done) == 6 {
			break
		}
	}
	if len(done) != 6 {
		t.Fatalf("changed kinds %v", done)
	}
}

func parameterCompiledValue(t *testing.T, m *alm.Map, n alm.ScriptNode, condition bool, tag, value uint32) {
	t.Helper()
	src := alm.Script{Actions: []alm.ScriptNode{n}}
	if condition {
		src.Actions, src.Conditions = nil, []alm.ScriptNode{n}
	}
	refs := mapload.ScriptRefs{Units: mapload.ScriptUnits(m, nil), Structures: mapload.ScriptStructures(m)}
	program, report, err := mapload.CompileScriptFrom(src, refs)
	if err != nil || program == nil || len(report.OmittedActions) != 0 || len(report.OmittedChecks) != 0 {
		t.Fatalf("changed node not compiled: %+v %v", report, err)
	}
	var record any
	if condition {
		if len(program.Checks()) != 1 {
			t.Fatal("changed check missing")
		}
		record = program.Checks()[0]
	} else {
		if len(program.Instants()) != 1 {
			t.Fatal("changed action missing")
		}
		record = program.Instants()[0]
	}
	r := reflect.ValueOf(record)
	if tag == ui.ScriptVariableParameter {
		if got := r.FieldByName("Args").Index(0).Int(); got != int64(value) {
			t.Fatalf("compiled register %d want %d", got, value)
		}
		return
	}
	field := map[uint32]string{4: "Unit", 2: "Group", 3: "Player", 9: "Structure", 8: "Item"}[tag]
	want := uint64(value)
	if tag == 4 {
		want = uint64(refs.Units[uint16(value)])
	}
	if tag == 9 {
		want = uint64(refs.Structures[uint16(value)])
	}
	if tag == 8 {
		want = uint64(uint16(value + 0xe18))
	}
	if !r.FieldByName("Has"+field).Bool() || r.FieldByName(field).Uint() != want {
		t.Fatalf("compiled %s differs from pick %d: %+v", field, value, record)
	}
}

func TestReleaseMapEditorParameterFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	dir := parameterOutput(t)
	done := map[uint32]bool{}
	summary := false
	for _, key := range installedMapKeys(x) {
		e := x.Editor()
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		e.Layout(1280, 800)
		closeMapPanel(t, e)
		sc := e.Document().Script
		if sc == nil {
			continue
		}
		if !summary {
			for _, trigger := range sc.Triggers {
				e.Select(trigger.Record, false)
				lines := trigLines(e)
				if strings.Contains(lines, x.inspectionUTF8(x.word("editor.param.unit"))) && strings.Contains(lines, x.inspectionUTF8(x.word("editor.param.player"))) {
					editorShot(t, e, dir, "parameter-trigger-summary", image.Rectangle{})
					t.Logf("summary frame from %s trigger %q", key, trigger.Name)
					summary = true
					break
				}
			}
		}
		for _, nodes := range [][]ui.InspectionScriptNode{sc.Actions, sc.Conditions} {
			for _, n := range nodes {
				for p := range n.Types {
					tag := scriptParameterKind(e.Document().Records[n.Record].Kind == "Condition", n.Opcode, n.Types, p)
					if (tag != 3 && tag != 4 && tag != 8 && tag != ui.ScriptVariableParameter) || done[tag] || tag == 4 && n.Values[p] >= 10001 {
						continue
					}
					e.Select(n.Record, false)
					r, ok := e.FieldRect(parameterField(n, p))
					trigClick(t, e, r, ok, "dropdown")
					editorShot(t, e, dir, fmt.Sprintf("parameter-%d-dropdown", tag), image.Rect(r.Min.X, r.Max.Y, 1280, 800))
					_ = e.Key("escape")
					if tag == 4 {
						for row, line := range e.PopupLines() {
							if !strings.HasPrefix(line, x.inspectionUTF8(x.word("editor.param.open"))) {
								continue
							}
							r, ok := e.PopupRowRect(row)
							trigClick(t, e, r, ok, "Open")
							editorShot(t, e, dir, "parameter-open", image.Rectangle{})
							r, ok = e.PopupButtonRect(2)
							trigClick(t, e, r, ok, "Back")
							editorShot(t, e, dir, "parameter-back", image.Rectangle{})
							break
						}
					}
					done[tag] = true
					t.Logf("frame kind %d from %s", tag, key)
				}
			}
		}
		if len(done) == 4 && summary {
			break
		}
	}
	if len(done) != 4 || !summary {
		t.Fatalf("frame kinds %v, trigger summary %v", done, summary)
	}
}
