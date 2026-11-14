package game

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mod"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// handoutTalks are the talk texts of the graph registry's tavern rows and
// of the rows the hand-out tests add.
var handoutTalks = []string{"main/text/inn/npc/npc05m30.txt", "main/text/inn/npc/npc06m30.txt",
	"main/text/inn/npc/npc07m40.txt", "main/text/inn/npc/npc05m31.txt", "main/text/inn/npc/npc08m32.txt",
	"main/text/inn/npc/npc05m32.txt", "main/text/inn/npc/npc00m30.txt"}

// handoutEditFixture is the graph registry's install with a mod holding
// mission 10's row and map, and towns when it is not empty, open in the
// editor on the campaign graph with Edit missions on.
func handoutEditFixture(t *testing.T, entry, towns string) (*ui.MapEditor, *mapInspectionEdits, string) {
	t.Helper()
	f, e := graphFixtureIn(t, entry, nil, []string{"Ford", "Ambush", "Vault"})
	f.x.table = companionTable(t)
	for _, p := range handoutTalks {
		f.x.archives.Containers.Overlay(p, func() ([]byte, error) { return []byte("<part=1>\r\nwords"), nil })
	}
	dir := writeEditorMod(t, t.TempDir(), graphEditableRows)
	writeFileAll(t, filepath.Join(dir, "maps", "first.alm"), synth.ALM(synth.ALMOptions{Width: 24, Height: 24, Name: "first"}))
	if towns != "" {
		writeFileAll(t, filepath.Join(dir, filepath.FromSlash(mod.TownsFile)), []byte(towns))
	}
	if err := f.x.SetMod(dir); err != nil {
		t.Fatal(err)
	}
	if f.x.mod.refusal != "" || f.x.mod.contextIssue != "" {
		t.Fatalf("open: %s %s", f.x.mod.refusal, f.x.mod.contextIssue)
	}
	if err := e.Open(filepath.Join(dir, "maps", "first.alm")); err != nil {
		t.Fatal(err)
	}
	showGraph(t, e)
	s := e.Document().Edits.(*mapInspectionEdits)
	graphAction(t, e, "graph.edit")
	return e, s, dir
}

// clickPick clicks the graph node whose pick starts with prefix.
func clickPick(t *testing.T, e *ui.MapEditor, s *mapInspectionEdits, prefix string) {
	t.Helper()
	page := s.CampaignGraph()
	i := slices.IndexFunc(page.Nodes, func(n ui.CampaignGraphNode) bool { return strings.HasPrefix(n.Pick, prefix) })
	if i < 0 {
		t.Fatalf("no node picked %q in %q", prefix, graphLines(page))
	}
	clickCanvasBox(t, e, i)
	if !strings.HasPrefix(s.handout, prefix) {
		t.Fatalf("the click selected %q, not %q", s.handout, prefix)
	}
}

// handoutEdit is one edit of the graph's hand-out fields: the node it
// selects by its pick, the field and the value.
type handoutEdit struct{ pick, key, value string }

// handoutEdits are one edit of each field of a hand-out, an addition to a
// town and a switch off, in an order each passes the game's check.
var handoutEdits = []handoutEdit{
	{"row shop", "won", "20"},
	{"row tavern 1 30 5", "mission", "31"},
	{"town 30", "add.tavern", "8 32"},
	{"row tavern 1 30 6", "off", ""},
	{"row school", "from", "1"},
	{"row tavern 1 30 8", "chapter", "40"},
	{"row tavern 1 40 8", "npc", "5"},
	{"row school", "building", "1"},
	{"town 40", "add.shop", "31"},
	{"row shop 1 40", "mission", "30"},
}

// handoutMade is the towns file a hand-made mod of handoutEdits' result
// holds, as the one row writer writes a mod with no towns file.
const handoutMade = `[[shop]]
town    = 1
chapter = 30
mission = 31
won     = [20]
[[tavern]]
town    = 1
chapter = 30
npc     = 5
mission = 31
[[tavern]]
town    = 1
chapter = 40
npc     = 5
mission = 32
[[tavern]]
town    = 1
chapter = 30
npc     = 6
off     = true
[[school]]
town    = 1
chapter = 30
mission = 32
off     = true
[[shop]]
town    = 1
from    = 30
mission = 32
[[shop]]
town    = 1
chapter = 40
mission = 30
`

// Each hand-out edit of the graph is one draft step of the mod: the graph
// shows it at once, Undo restores the draft and the graph, Redo applies it
// again, and no file is written before Ctrl+S; in both languages of the
// word book.
func TestCampaignGraphHandoutEditsAreDraftsWithOneUndo(t *testing.T) {
	for _, entry := range []string{"english", "russian"} {
		t.Run(entry, func(t *testing.T) {
			e, s, dir := handoutEditFixture(t, entry, "")
			files := tree(t, dir)
			for _, edit := range handoutEdits {
				clickPick(t, e, s, edit.pick)
				before, steps, page := workspaceBytes(s.workspace), len(s.undoWorkspace), graphLines(s.CampaignGraph())
				var err error
				if edit.key == "off" {
					graphAction(t, e, edit.key)
				} else {
					err = graphField(t, e, edit.key, edit.value)
				}
				after := workspaceBytes(s.workspace)
				if err != nil || bytes.Equal(before, after) || len(s.undoWorkspace) != steps+1 {
					t.Fatalf("%v: not one draft step: %v", edit, err)
				}
				if slices.Equal(page, graphLines(s.CampaignGraph())) {
					t.Fatalf("%v: the graph kept the saved hand-outs", edit)
				}
				if err := e.Key("undo"); err != nil || !bytes.Equal(workspaceBytes(s.workspace), before) || !slices.Equal(page, graphLines(s.CampaignGraph())) {
					t.Fatalf("%v: undo did not restore the draft and the graph: %v", edit, err)
				}
				if err := e.Key("redo"); err != nil || !bytes.Equal(workspaceBytes(s.workspace), after) {
					t.Fatalf("%v redo: %v", edit, err)
				}
			}
			if !reflect.DeepEqual(files, tree(t, dir)) {
				t.Fatal("a hand-out edit wrote before Ctrl+S")
			}
		})
	}
}

// The edits written on Ctrl+S are the towns file a hand-made mod of the
// same content holds; the mod reopens as the game admits it with the
// graph the draft showed, an added row marked new and a changed install
// row marked changed.
func TestCampaignGraphHandoutsSaveAsTheHandMadeModAndReopen(t *testing.T) {
	for _, entry := range []string{"english", "russian"} {
		t.Run(entry, func(t *testing.T) {
			e, s, dir := handoutEditFixture(t, entry, "")
			files := tree(t, dir)
			for _, edit := range handoutEdits {
				clickPick(t, e, s, edit.pick)
				if edit.key == "off" {
					graphAction(t, e, edit.key)
				} else if err := graphField(t, e, edit.key, edit.value); err != nil {
					t.Fatalf("%v: %v", edit, err)
				}
			}
			page := s.CampaignGraph()
			marks := map[string]string{}
			for _, n := range page.Nodes {
				if n.Kind == ui.GraphHandout {
					marks[n.Pick] = fmt.Sprint(n.AddedMark, n.FlowMark)
				}
			}
			if marks["row shop 1 30 0 31"] != "false true" || marks["row tavern 1 40 5 0"] != "true false" || marks["row tavern 1 40 7 0"] != "false false" {
				t.Fatalf("hand-out marks %v", marks)
			}
			if err := e.Key("saveas"); err != nil {
				t.Fatal(err)
			}
			if s.Dirty() {
				t.Fatal("the save kept the draft dirty")
			}
			files[mod.TownsFile] = handoutMade
			if got := tree(t, dir); !reflect.DeepEqual(files, got) {
				t.Fatalf("the save differs from the hand-made mod:\n%s", got[mod.TownsFile])
			}
			if want, got := graphLines(page), reopenedGraph(t, dir); !slices.Equal(want, got) {
				t.Fatalf("the reopened mod's graph differs:\n%q\nwant\n%q", got, want)
			}
		})
	}
}

// reopenedGraph is the graph of the mod in dir opened afresh over the
// graph registry's install.
func reopenedGraph(t *testing.T, dir string) []string {
	t.Helper()
	x := contextInspector(t)
	x.table = companionTable(t)
	x.archives.Containers.Overlay(ScenarioRegistry, func() ([]byte, error) { return graphRegistry(), nil })
	x.archives.Containers.Overlay("scenario/10.alm", func() ([]byte, error) { return synth.ALM(synth.ALMOptions{Width: 24, Height: 24}), nil })
	for _, p := range handoutTalks {
		x.archives.Containers.Overlay(p, func() ([]byte, error) { return []byte("<part=1>\r\nwords"), nil })
	}
	if err := x.SetMod(dir); err != nil || x.mod.refusal != "" || x.mod.contextIssue != "" {
		t.Fatalf("reopen: %v %s %s", err, x.mod.refusal, x.mod.contextIssue)
	}
	return graphLines((&mapInspectionEdits{inspector: x}).CampaignGraph())
}

// placeSource is a mod's towns file holding a tavern talk, a shop and a
// school row of chapter 40, each under a comment.
const placeSource = `# The mod's hand-outs.
[[tavern]]
town = 1
chapter = 40
npc = 8
mission = 32

# The shop of chapter 40.
[[shop]]
town = 1
chapter = 40
mission = 31

# The school of chapter 40.
[[school]]
town = 1
chapter = 40
mission = 30
`

// placeMade is placeSource after its three chapter 40 rows changed place:
// each leaves with its own lines and the row of its new table is added.
const placeMade = `# The mod's hand-outs.

# The shop of chapter 40.

# The school of chapter 40.
[[tavern]]
town    = 1
chapter = 40
npc     = 5
mission = 30
[[school]]
town    = 1
chapter = 40
mission = 31
[[shop]]
town    = 1
chapter = 40
mission = 32
`

// "Hands out" on a row the mod's file holds moves the row to the table of
// its new place: tavern to shop, shop to school and school to tavern with
// an NPC given are each one draft step that Undo restores, and Ctrl+S
// writes the file a hand-made mod holds, which reopens with the same graph.
func TestCampaignGraphHandoutPlaceChangeOfAModRow(t *testing.T) {
	for _, entry := range []string{"english", "russian"} {
		t.Run(entry, func(t *testing.T) {
			e, s, dir := handoutEditFixture(t, entry, placeSource)
			for _, edit := range []handoutEdit{
				{"row school 1 40 0 30", "building", "0"},
				{"row tavern 1 40 0", "npc", "5"},
				{"row shop 1 40 0 31", "building", "2"},
				{"row tavern 1 40 8", "building", "1"},
			} {
				clickPick(t, e, s, edit.pick)
				before, steps := workspaceBytes(s.workspace), len(s.undoWorkspace)
				if err := graphField(t, e, edit.key, edit.value); err != nil {
					t.Fatalf("%v: %v", edit, err)
				}
				after := workspaceBytes(s.workspace)
				if bytes.Equal(before, after) || len(s.undoWorkspace) != steps+1 || strings.HasPrefix(s.handout, edit.pick) {
					t.Fatalf("%v: not one draft step of the selected row (now %q)", edit, s.handout)
				}
				if err := e.Key("undo"); err != nil || !bytes.Equal(workspaceBytes(s.workspace), before) {
					t.Fatalf("%v: undo did not restore the draft: %v", edit, err)
				}
				if err := e.Key("redo"); err != nil || !bytes.Equal(workspaceBytes(s.workspace), after) {
					t.Fatalf("%v redo: %v", edit, err)
				}
			}
			page := graphLines(s.CampaignGraph())
			if err := e.Key("saveas"); err != nil {
				t.Fatal(err)
			}
			if got := tree(t, dir)[mod.TownsFile]; got != placeMade {
				t.Fatalf("written:\n%s\nwant:\n%s", got, placeMade)
			}
			if got := reopenedGraph(t, dir); !slices.Equal(page, got) {
				t.Fatalf("the reopened mod's graph differs:\n%q\nwant\n%q", got, page)
			}
		})
	}
}

// A field the check refuses is refused at the edit with the check's words,
// naming the towns file and its line; a field that is no number with the
// word book's; neither leaves a draft step or a file.
func TestCampaignGraphHandoutRefusalKeepsTheDraft(t *testing.T) {
	for _, entry := range []string{"english", "russian"} {
		t.Run(entry, func(t *testing.T) {
			e, s, dir := handoutEditFixture(t, entry, "")
			clickPick(t, e, s, "row shop")
			before, steps, files := workspaceBytes(s.workspace), len(s.undoWorkspace), tree(t, dir)
			w := cloneWorkspace(s.workspace)
			w.Project.Towns = append(w.Project.Towns, mod.OfferRecord{Source: mod.Source{Mod: "ambush", File: mod.TownsFile},
				Table: town.ShopTable, Row: town.OfferRow{Kind: town.OfferMission, Topic: 31, Town: 1, Stages: []int{30}, Place: town.PlaceShop, Won: []int{999}},
				Fields: []mod.PatchField{{Name: "won"}}})
			_, want := s.missionDraft(w)
			_, got := s.SetWorkspaceField("handout", "won", "999")
			if want == nil || got == nil || got.Error() != want.Error() || !strings.Contains(got.Error(), mod.TownsFile) {
				t.Fatalf("got %v, want %v", got, want)
			}
			if _, err := s.SetWorkspaceField("handout", "town", "2"); err == nil || !strings.Contains(err.Error(), "town 2 is not a town of the catalog") {
				t.Fatalf("town 2: %v", err)
			}
			_, words := s.SetWorkspaceField("handout", "won", "x")
			if words == nil || !strings.Contains(words.Error(), s.inspector.word("editor.handout.won")) {
				t.Fatalf("a non-number: %v", words)
			}
			if !bytes.Equal(before, workspaceBytes(s.workspace)) || steps != len(s.undoWorkspace) || !reflect.DeepEqual(files, tree(t, dir)) {
				t.Fatal("a refusal changed the draft, the history or the files")
			}
		})
	}
}

// An added shop hand-out that a shop row standing from an earlier chapter
// already gives in its chapter is refused with the check's words, naming
// the towns file; the draft keeps the floor row alone.
func TestCampaignGraphHandoutRefusesAMissionTwiceInOnePlace(t *testing.T) {
	for _, entry := range []string{"english", "russian"} {
		t.Run(entry, func(t *testing.T) {
			e, s, _ := handoutEditFixture(t, entry, "")
			clickPick(t, e, s, "row shop 1 30 0 31")
			if err := graphField(t, e, "from", "1"); err != nil {
				t.Fatal(err)
			}
			clickPick(t, e, s, "town 40")
			before, steps := workspaceBytes(s.workspace), len(s.undoWorkspace)
			_, err := s.SetWorkspaceField("handout", "add.shop", "31")
			if err == nil || !strings.Contains(err.Error(), mod.TownsFile) || !strings.Contains(err.Error(), "the shop of town 1 already hands out mission 31 in chapter 40") {
				t.Fatalf("got %v", err)
			}
			if !bytes.Equal(before, workspaceBytes(s.workspace)) || steps != len(s.undoWorkspace) {
				t.Fatal("the refusal changed the draft or the history")
			}
		})
	}
}

// handoutSource is a mod's towns file with comments and an untouched row.
const handoutSource = `# The tavern of chapter 30.
[[tavern]]
town = 1
chapter = 30
npc = 5
mission = 30   # the main mission
won = [20]

# Kept as written.
[[school]]
town = 1
chapter = 30
mission = 32
`

// An edit of a mod's own row rewrites the edited value alone: the comments,
// the order and the untouched row stay as the author wrote them; the page
// says which talk text the row speaks and that the install holds it.
func TestCampaignGraphHandoutEditKeepsTheModsSource(t *testing.T) {
	for _, entry := range []string{"english", "russian"} {
		t.Run(entry, func(t *testing.T) {
			e, s, dir := handoutEditFixture(t, entry, handoutSource)
			clickPick(t, e, s, "row tavern 1 30 5")
			talk := fmt.Sprintf(s.inspector.word("editor.handout.talk.install"), "main/text/inn/npc/npc05m30.txt")
			if !slices.ContainsFunc(e.ObjectFields(), func(f ui.InspectionField) bool { return f.Key == "talk" && f.TextValue == talk }) {
				t.Fatalf("no talk line %q in %v", talk, e.ObjectFields())
			}
			if err := graphField(t, e, "won", "10, 20"); err != nil {
				t.Fatal(err)
			}
			if err := e.Key("saveas"); err != nil {
				t.Fatal(err)
			}
			want := strings.Replace(handoutSource, "won = [20]", "won = [10, 20]", 1)
			if got := tree(t, dir)[mod.TownsFile]; got != want {
				t.Fatalf("written:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}
