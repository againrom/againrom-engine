package ui

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/words"
)

func modesCampaignFields() []InspectionField {
	return []InspectionField{
		{Key: "AutoGetMission", Label: "Successor", Text: true},
		{Key: "landing", Label: "Party", Value: 2, Choices: []InspectionChoice{{Value: 2, Label: "Main"}, {Value: 1, Label: "Guest party"}}},
		{Key: "insert", Label: "Insert mission", Text: true},
	}
}

func graphEditAction(t *testing.T, e *MapEditor, key string) {
	t.Helper()
	if !e.FocusModField(key) {
		t.Fatalf("no action %s in %v", key, e.ObjectFields())
	}
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
}

func graphEditClick(t *testing.T, e *MapEditor, number int) {
	t.Helper()
	page, _ := e.modGraph()
	index := slices.IndexFunc(page.Nodes, func(n CampaignGraphNode) bool { return n.Kind == GraphMission && n.Number == number })
	_, l := graphLayout(t, e)
	r := l.CanvasBoxes.rect(index).Intersect(l.Canvas)
	if r.Empty() {
		t.Fatalf("mission %d outside canvas", number)
	}
	modesClick(e, center(r))
}

func TestCampaignGraphCanvasEditsUseWorkspaceFields(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			page := modesGraph()
			page.Mod = true
			e, calls := graphEditor(t, page)
			e.Layout(1280, 1080)
			e.SetWords(words.For(lang))
			e.dropObject()
			host := e.doc.Edits.(*modesWorkspace)
			host.graphFields = modesCampaignFields()
			if !slices.ContainsFunc(e.ObjectFields(), func(f InspectionField) bool { return f.Key == "graph.edit" && f.Label == e.word("editor.graph.edit") }) {
				t.Fatal("edit button lacks translated word")
			}
			graphEditAction(t, e, "graph.edit")
			graphEditClick(t, e, 10)
			graphEditAction(t, e, "graph.link")
			graphEditClick(t, e, 20)
			if got := calls.lines[len(calls.lines)-1]; got != `Workspace campaign.10 AutoGetMission "20"` {
				t.Fatalf("link: %s", got)
			}
			if e.graphLink || e.graphMission != 10 {
				t.Fatal("link did not keep its source selected")
			}
			graphEditAction(t, e, "graph.unlink")
			if got := calls.lines[len(calls.lines)-1]; got != `Workspace campaign.10 AutoGetMission "-1"` {
				t.Fatalf("unlink: %s", got)
			}
			for _, row := range []struct{ key, value string }{{"insert", "11 maps/detour.alm"}, {"landing", "1"}} {
				if !e.FocusModField(row.key) {
					t.Fatalf("no field %s", row.key)
				}
				_ = e.Key("clear")
				e.TextInput(row.value)
				if err := e.Key("enter"); err != nil {
					t.Fatal(err)
				}
				if got, want := calls.lines[len(calls.lines)-1], fmt.Sprintf("Workspace campaign.10 %s %q", row.key, row.value); got != want {
					t.Fatalf("got %s, want %s", got, want)
				}
			}
			graphEditAction(t, e, "graph.link")
			_ = e.Key("escape")
			if e.graphLink || !e.modWindow {
				t.Fatal("Esc did not cancel just the pending arrow")
			}
			if err := e.Key("saveas"); err != nil {
				t.Fatal(err)
			}
			if got := calls.lines[len(calls.lines)-1]; got != `Workspace package save ""` {
				t.Fatalf("save: %s", got)
			}
		})
	}
}

func TestCampaignGraphCanvasRefusalKeepsTheSourceAndWords(t *testing.T) {
	page := modesGraph()
	page.Mod = true
	e, calls := graphEditor(t, page)
	e.Layout(1280, 1080)
	host := e.doc.Edits.(*modesWorkspace)
	host.graphFields = modesCampaignFields()
	graphEditAction(t, e, "graph.edit")
	graphEditClick(t, e, 10)
	graphEditAction(t, e, "graph.link")
	host.graphRefusal = errors.New("data/missions.toml:5: AutoGetMission cycle from mission 10 through mission 10")
	count := len(calls.lines)
	graphEditClick(t, e, 10)
	if len(calls.lines) != count || !e.graphLink || e.graphMission != 10 || e.modStatus != host.graphRefusal.Error() {
		t.Fatalf("refused edit: %v %d %q", e.graphLink, e.graphMission, e.modStatus)
	}
}

func TestCampaignGraphWithoutAModStaysReadOnly(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e, _ := graphEditor(t, modesGraph())
		e.SetWords(words.For(lang))
		if !strings.Contains(strings.Join(e.PopupLines(), "\n"), words.For(lang).Text("editor.graph.readonly")) || e.FocusModField("graph.edit") {
			t.Fatalf("%s: no read-only notice or an edit control", lang)
		}
	}
}
