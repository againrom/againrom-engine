package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"os"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

func playerWindow(t *testing.T) (*MapInspector, *ui.MapEditor, string) {
	t.Helper()
	spec := alm.DefaultNewMap()
	spec.Width, spec.Height = 32, 32
	doc, err := alm.NewMap(spec)
	if err != nil {
		t.Fatal(err)
	}
	return openTestMapEditor(t, doc.Write())
}

func windowPlayers(t *testing.T, e *ui.MapEditor) []alm.Group {
	t.Helper()
	m, err := alm.Open(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	return m.Groups
}

func selectedPlayer(t *testing.T, e *ui.MapEditor, index int) {
	t.Helper()
	if e.Selected() < 0 || e.Selected() >= len(e.Document().Records) {
		t.Fatal("no selected player")
	}
	r := e.Document().Records[e.Selected()]
	if r.Kind != "Player" || r.Index != index || r.Spatial {
		t.Fatalf("selected record %+v, want player %d without a map cell", r, index)
	}
}

func TestMapEditorPlayersWindowEditsAreSeparateUndoSteps(t *testing.T) {
	_, e, _ := playerWindow(t)
	history := [][]byte{bytes.Clone(e.Document().SourceBytes)}
	if err := e.Key("players"); err != nil {
		t.Fatal(err)
	}
	selectedPlayer(t, e, 0)
	if err := e.Key("player"); err != nil {
		t.Fatal(err)
	}
	selectedPlayer(t, e, 2)
	if len(windowPlayers(t, e)) != 3 {
		t.Fatal("add key did not create a third player")
	}
	history = append(history, bytes.Clone(e.Document().SourceBytes))
	// The name is typed in the player's row; the numbers go through the
	// editor's set_field command, the one the row switch and the matrix use.
	_, must := editorMCP(t, e)
	set := func(field int, value string) error {
		if field == 0 {
			return enterField(e, 0, value)
		}
		n, err := strconv.Atoi(value)
		if err == nil {
			must("set_field", map[string]any{"kind": "Player", "index": 2, "key": e.Document().Edits.Fields("Player", 2)[field].Key, "value": n})
		}
		return err
	}
	for _, step := range []struct {
		field int
		value string
		check func(alm.Group) bool
	}{
		{0, "Copper", func(g alm.Group) bool { return g.Name == "Copper" }},
		{1, "9", func(g alm.Group) bool { return g.Color == 9 }},
		{2, "0", func(g alm.Group) bool { return g.Participant == 0 }},
		{3, "0", func(g alm.Group) bool { return g.Relation[0] == 0 }},
	} {
		if err := set(step.field, step.value); err != nil {
			t.Fatalf("field %d refused %q: %v", step.field, step.value, err)
		}
		if !step.check(windowPlayers(t, e)[2]) {
			t.Fatalf("field %d did not reach the roster", step.field)
		}
		if bytes.Equal(history[len(history)-1], e.Document().SourceBytes) {
			t.Fatalf("field %d changed no byte", step.field)
		}
		history = append(history, bytes.Clone(e.Document().SourceBytes))
	}
	for i := len(history) - 2; i >= 0; i-- {
		if err := e.Key("undo"); err != nil || !bytes.Equal(history[i], e.Document().SourceBytes) {
			t.Fatalf("undo %d did not restore the preceding edit: %v", i, err)
		}
	}
	if e.Dirty() || e.Document().Edits.CanUndo() {
		t.Fatal("all player edits undone did not reach the clean original")
	}
	for i := 1; i < len(history); i++ {
		if err := e.Key("redo"); err != nil || !bytes.Equal(history[i], e.Document().SourceBytes) {
			t.Fatalf("redo %d did not restore the next edit: %v", i, err)
		}
	}
	e.Select(recordIndex(t, e, "Player", 2), false)
	// Del asks first; Enter answers the question.
	if err := e.Key("delete"); err != nil || len(windowPlayers(t, e)) != 3 {
		t.Fatal("Del deleted before it asked", err)
	}
	if err := e.Key("enter"); err != nil || len(windowPlayers(t, e)) != 2 {
		t.Fatal("delete unreferenced player", err)
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(history[len(history)-1], e.Document().SourceBytes) {
		t.Fatal("player delete was not one exact undo", err)
	}
}

func TestMapEditorPlayersRefuseInvalidFieldsAndSpatialEdits(t *testing.T) {
	_, e, _ := playerWindow(t)
	_ = e.Key("players")
	selectedPlayer(t, e, 0)
	before := bytes.Clone(e.Document().SourceBytes)
	if err := e.Key("delete"); err == nil || !strings.Contains(strings.ToLower(e.Message()), "first player") {
		t.Fatal("first-player delete did not explain its refusal", err, e.Message())
	}
	if !bytes.Equal(before, e.Document().SourceBytes) || e.Document().Edits.CanUndo() {
		t.Fatal("first-player refusal changed bytes or history")
	}
	_ = e.Key("escape")
	tool, _ := editorMCP(t, e)
	for _, tc := range []struct {
		field int
		value any
	}{{0, strings.Repeat("x", 33)}, {0, "🙂"}, {1, 16}, {2, 2}, {3, 3}} {
		args := map[string]any{"kind": "Player", "index": 0, "key": e.Document().Edits.Fields("Player", 0)[tc.field].Key}
		if text, ok := tc.value.(string); ok {
			args["text"] = text
		} else {
			args["value"] = tc.value
		}
		if r := tool("set_field", args); !r.IsError || len(r.Content) == 0 || r.Content[0].Text == "" {
			t.Fatalf("field %d accepted %v or gave no reason", tc.field, tc.value)
		}
		if !bytes.Equal(before, e.Document().SourceBytes) || e.Document().Edits.CanUndo() {
			t.Fatal("invalid player field changed bytes or history")
		}
	}
	_ = e.Key("clone")
	clickCell(e, image.Pt(18, 18))
	if !bytes.Equal(before, e.Document().SourceBytes) || e.Document().Edits.CanUndo() {
		t.Fatal("a player clone took a map cell")
	}
	for _, apply := range []func() (*ui.InspectionDocument, error){
		func() (*ui.InspectionDocument, error) { return e.Document().Edits.Clone("Player", 0, image.Pt(18, 18)) },
		func() (*ui.InspectionDocument, error) { return e.Document().Edits.Move("Player", 0, image.Pt(18, 18)) },
	} {
		if _, err := apply(); err == nil || !bytes.Equal(before, e.Document().SourceBytes) {
			t.Fatal("the host accepted a spatial player edit", err)
		}
	}
}

func TestMapEditorPlayerDeleteNamesEachReference(t *testing.T) {
	for _, kind := range []string{"unit", "structure", "action", "condition"} {
		t.Run(kind, func(t *testing.T) {
			o := synth.ALMOptions{Width: 32, Height: 32, Type7Payload: scriptPayload(nil, nil)}
			switch kind {
			case "unit":
				o.Units = []synth.ALMUnit{{X: 12 * 256, Y: 12 * 256, ClassID: 40, ClassSubID: 2}}
			case "structure":
				o.Objects = []synth.ALMObject{{X: 12 * 256, Y: 12 * 256, Kind: 1, Field0E: 2}}
			case "action":
				o.Type7Payload = scriptPayload([]scriptNode{{label: "owner", opcode: 0xffff, id: 7, values: []uint32{2}, tags: []uint32{3}}}, nil)
			case "condition":
				o.Type7Payload = scriptPayload(nil, []scriptNode{{label: "owner", opcode: 0xffff, id: 7, values: []uint32{2}, tags: []uint32{3}}})
			}
			m := built(t, o, func(m *mapedit.Editor) {
				rec, err := m.NewPlayerRecord()
				if err != nil {
					t.Fatal(err)
				}
				if _, err := m.Place(mapedit.KindPlayer, rec); err != nil {
					t.Fatal(err)
				}
			}, func(raw []byte, payload func(uint32) int) {
				if kind == "unit" {
					binary.LittleEndian.PutUint32(raw[payload(6)+0x14:], 2)
				}
			})
			_, e, _ := openTestMapEditor(t, m.Bytes())
			_ = e.Key("players")
			e.Select(recordIndex(t, e, "Player", 1), false)
			before := bytes.Clone(e.Document().SourceBytes)
			// Del asks first; the answer meets the refusal.
			if err := e.Key("delete"); err != nil || len(e.PromptNotes()) == 0 {
				t.Fatal("Del did not ask before the delete", err)
			}
			_ = e.Key("enter")
			if !strings.Contains(strings.ToLower(e.Message()), kind) || !strings.Contains(e.Message(), "2") {
				t.Fatal("the window did not name the player reference", e.Message())
			}
			if !bytes.Equal(before, e.Document().SourceBytes) || e.Document().Edits.CanUndo() {
				t.Fatal("a refused player delete changed bytes or history")
			}
		})
	}
}

func TestMapEditorPlayersSaveAsReopensTheirBytes(t *testing.T) {
	x, e, source := playerWindow(t)
	before := bytes.Clone(e.Document().SourceBytes)
	_ = e.Key("players")
	_ = e.Key("player")
	selectedPlayer(t, e, 2)
	if err := enterField(e, 0, "Copper"); err != nil {
		t.Fatal(err)
	}
	want := bytes.Clone(e.Document().SourceBytes)
	target := source
	if err := e.Key("saveas"); err != nil || e.Dirty() {
		t.Fatal("save player roster", err, e.Message())
	}
	saved, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(saved, want) || bytes.Equal(saved, before) {
		t.Fatal("saved roster differs from the editor", err)
	}
	fresh := x.Editor()
	if err := fresh.Open(target); err != nil {
		t.Fatal(err)
	}
	if fresh.Dirty() || !bytes.Equal(fresh.Document().SourceBytes, want) || windowPlayers(t, fresh)[2].Name != "Copper" {
		t.Fatal("fresh open did not retain the saved player roster")
	}
}
