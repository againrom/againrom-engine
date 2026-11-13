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

// modesHandoutFields are a hand-out's fields as a host gives them.
func modesHandoutFields() []InspectionField {
	return []InspectionField{
		{Key: "building", Label: "Hands out", Value: 0, Choices: []InspectionChoice{{Value: 0, Label: "Tavern talk"}, {Value: 1, Label: "Shop"}, {Value: 2, Label: "School"}}},
		{Key: "mission", Label: "Mission", Text: true, TextValue: "30"},
		{Key: "won", Label: "After wins of", Text: true},
		{Key: "off", Label: "Switch off", Action: true},
	}
}

// modesHandoutGraph is modesGraph of a mod whose towns and hand-outs the
// host picks by their numbers.
func modesHandoutGraph() CampaignGraphPage {
	page := modesGraph()
	page.Mod = true
	for i := range page.Nodes {
		if n := &page.Nodes[i]; n.Kind == GraphTown || n.Kind == GraphHandout {
			n.Pick = fmt.Sprintf("node %d", i)
		}
	}
	return page
}

// firstHandout is the node of modesGraph's first hand-out.
func firstHandout(page CampaignGraphPage) int {
	return slices.IndexFunc(page.Nodes, func(n CampaignGraphNode) bool { return n.Kind == GraphHandout })
}

// With missions edited, a click on a town or a hand-out the host picks
// selects it through the host's one field route, its fields are the host's
// and each edit names the selection's scope; a click on a mission returns
// to the mission's fields. With missions not edited the click opens
// nothing and selects nothing.
func TestCampaignGraphHandoutNodesSelectThroughTheHost(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			page := modesHandoutGraph()
			e, calls := graphEditor(t, page)
			e.Layout(1280, 1080)
			e.SetWords(words.For(lang))
			e.dropObject()
			host := e.doc.Edits.(*modesWorkspace)
			host.graphFields, host.handoutFields = modesCampaignFields(), modesHandoutFields()
			at := firstHandout(page)
			_, l := graphLayout(t, e)
			count := len(calls.lines)
			modesClick(e, center(l.CanvasBoxes.rect(at)))
			if len(calls.lines) != count || e.graphPicked {
				t.Fatalf("a click outside Edit missions selected: %q", calls.lines[count:])
			}
			graphEditAction(t, e, "graph.edit")
			_, l = graphLayout(t, e)
			modesClick(e, center(l.CanvasBoxes.rect(at)))
			if got, want := calls.lines[len(calls.lines)-1], fmt.Sprintf("Workspace handout select %q", page.Nodes[at].Pick); got != want {
				t.Fatalf("select: %s, want %s", got, want)
			}
			if !slices.ContainsFunc(e.ObjectFields(), func(f InspectionField) bool { return f.Key == "won" }) {
				t.Fatalf("the hand-out's fields are not the host's: %v", e.ObjectFields())
			}
			if !e.FocusModField("won") {
				t.Fatal("no won field")
			}
			_ = e.Key("clear")
			e.TextInput("31")
			if err := e.Key("enter"); err != nil {
				t.Fatal(err)
			}
			if got := calls.lines[len(calls.lines)-1]; got != `Workspace handout won "31"` {
				t.Fatalf("won: %s", got)
			}
			graphEditAction(t, e, "off")
			if got := calls.lines[len(calls.lines)-1]; !strings.HasPrefix(got, "Workspace handout off") {
				t.Fatalf("off: %s", got)
			}
			graphEditClick(t, e, 10)
			if e.graphPicked || e.modFieldScope() != "campaign.10" {
				t.Fatalf("a mission click kept the hand-out selected: %s", e.modFieldScope())
			}
		})
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
