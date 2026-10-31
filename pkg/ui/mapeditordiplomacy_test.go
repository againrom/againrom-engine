package ui

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/words"
)

func diplomacyEditor(t *testing.T, language string, stored uint32) (*MapEditor, *wizardWrites) {
	t.Helper()
	host := &wizardWrites{editorTextEdits: &editorTextEdits{}}
	build := func(string) (*InspectionDocument, error) {
		v, err := NewViewer("diplomacy", terrain.Grid{Width: 64, Height: 64, Tiles: make([]uint16, 64*64)}, &terrain.Tileset{})
		if err != nil {
			return nil, err
		}
		d := &InspectionDocument{Viewer: v, Source: "diplomacy", Width: 64, Height: 64, SourceBytes: []byte("map"), Edits: host}
		d.Records = []InspectionRecord{{Kind: "Trigger", Index: 0, Section: 7},
			{Kind: "Condition", Index: 0, Section: 7}, {Kind: "Condition", Index: 1, Section: 7}, {Kind: "Action", Index: 0, Section: 7}}
		d.Script = &InspectionScriptModel{
			Triggers: []InspectionTrigger{{Record: 0, Name: "Veglud", Once: 1, Left: [3]uint32{1}, Right: [3]uint32{2}, Cmp: [3]uint32{1}, Acts: [4]uint32{11}}},
			Conditions: []InspectionScriptNode{
				{Record: 1, ID: 1, Opcode: 10, Types: [10]uint32{scriptPlayer, scriptPlayer}, Values: [10]uint32{8, 1}},
				{Record: 2, ID: 2, Opcode: scriptConstantOp, Types: [10]uint32{scriptInt}, Values: [10]uint32{0}}},
			Actions: []InspectionScriptNode{{Record: 3, ID: 11, Opcode: 10, Types: [10]uint32{scriptInt, scriptInt, scriptInt}, Values: [10]uint32{8, 1, stored}}},
			Targets: map[uint32][]InspectionScriptTarget{scriptPlayer: {
				{Value: 1, Record: -1, Name: "Self", Choice: "Player Self"},
				{Value: 8, Record: -1, Name: "Veglud", Choice: "Player Veglud"}}},
		}
		return d, nil
	}
	e := NewMapEditor(panelFont(), nil, build)
	e.SetBodyFont(panelFont())
	e.SetWords(words.For(language))
	e.Layout(800, 600)
	if err := e.Open("diplomacy"); err != nil {
		t.Fatal(err)
	}
	e.Select(0, false)
	if !e.WizardShown() {
		t.Fatalf("the trigger opens no wizard:\n%s", strings.Join(e.PopupLines(), "\n"))
	}
	return e, host
}

func relationChoices(e *MapEditor, key string) []string {
	var out []string
	for _, c := range e.obj.fields[e.WizardField(key)].Choices {
		out = append(out, c.Label)
	}
	return out
}

func TestTriggerWindowReadsDiplomacyInPlainWords(t *testing.T) {
	for _, tc := range []struct {
		language string
		item     [2]string
		names    string
		relation []string
		shown    string
		hint     string
	}{
		{"", [2]string{"Relation between players", "Change relation"}, "Veglud", []string{"Neutral", "Hostile", "Ally"}, "Hostile", "needs a second action"},
		{"russian", [2]string{"Отношение игроков", "Изменить отношение"}, "Veglud", []string{"Нейтралитет", "Вражда", "Союз"}, "Вражда", "нужно второе действие"},
	} {
		e, _ := diplomacyEditor(t, tc.language, 1)
		lines := strings.Join(e.PopupLines(), "\n")
		for _, want := range append(tc.item[:], tc.names, tc.shown, tc.hint) {
			if !strings.Contains(lines, want) {
				t.Fatalf("%q: the window lacks %q:\n%s", tc.language, want, lines)
			}
		}
		if strings.Contains(lines, "#") {
			t.Fatalf("%q: a diplomacy row still reads as Other:\n%s", tc.language, lines)
		}
		for _, key := range []string{"c0.p2", "a0.p2"} {
			if got := relationChoices(e, key); !slices.Equal(got, tc.relation) {
				t.Fatalf("%q: %s offers %q, want %q", tc.language, key, got, tc.relation)
			}
		}
	}
}

func TestTriggerWindowKeepsAnUnknownRelationAndWritesTheChosenOne(t *testing.T) {
	e, host := diplomacyEditor(t, "", 7)
	if got := relationChoices(e, "a0.p2"); !slices.Equal(got, []string{"Neutral", "Hostile", "Ally", "7"}) {
		t.Fatalf("a stored 7 offers %q", got)
	}
	v, _ := e.scriptPopup()
	e.wizardButton(v, 0)
	if len(host.plans) != 1 {
		t.Fatalf("Apply wrote %d triggers (note %q)", len(host.plans), e.obj.note)
	}
	if n := host.plans[0].Actions[0]; !n.Keep || n.ID != 11 {
		t.Fatalf("an unchanged diplomacy row writes %+v, want node 11 kept", n)
	}
	if p := host.plans[0].Pairs[0]; !p.Left.Keep || !p.Right.Keep || p.Cmp != 1 {
		t.Fatalf("an unchanged check writes %+v, want its nodes kept", p)
	}

	wizardChoose(t, e, "a0.p2", 2)
	wizardChoose(t, e, "c0.p2", 1)
	v, _ = e.scriptPopup()
	plan, err := e.wizardPlan(v)
	if err != nil {
		t.Fatal(err)
	}
	want := WizardNode{Opcode: 10, Types: [10]uint32{scriptInt, scriptInt, scriptInt}, Values: [10]uint32{8, 1, 2}}
	if n := plan.Actions[0]; n.Keep || n.Opcode != want.Opcode || n.Types != want.Types || n.Values != want.Values {
		t.Fatalf("the chosen alliance writes %+v, want %+v", n, want)
	}
	p := plan.Pairs[0]
	if p.Left.Keep || p.Left.Opcode != 10 || p.Left.Types != [10]uint32{scriptPlayer, scriptPlayer} || p.Left.Values != [10]uint32{8, 1} {
		t.Fatalf("the check writes %+v", p.Left)
	}
	if p.Right.Opcode != scriptConstantOp || p.Right.Values[0] != 1 || p.Cmp != 1 {
		t.Fatalf("the check's relation writes %+v with comparison %d, want the constant 1 and comparison 1", p.Right, p.Cmp)
	}
}
