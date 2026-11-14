package game

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mod"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// graphRegistry is a scenario whose mission 10 leads to mission 20 with no
// town between. The town of chapter 30 hands out its main mission and
// nothing at the tavern, a side mission at the shop and one at the school;
// the town of chapter 40 hands out the last mission. Chapter 30 grants
// companion 22, whom a mod's guest mission may play.
func graphRegistry() []byte {
	return synth.Reg(kindRoot, []synth.RegNode{
		{Name: "Mission10", Kind: kindDir, Children: []synth.RegNode{{Name: "AutoGetMission", Kind: kindInt, Int: 20}}},
		{Name: "Mission20", Kind: kindDir, Children: []synth.RegNode{{Name: "Mercenaries", Kind: kindInt, Int: 1}}},
		{Name: "Mission30", Kind: kindDir, Children: []synth.RegNode{
			{Name: "InnNPC", Kind: kindIntArray, Ints: []int32{5, 6}},
			{Name: "InnMission", Kind: kindIntArray, Ints: []int32{30, 0}},
			{Name: "ShopMission", Kind: kindIntArray, Ints: []int32{31}},
			{Name: "TCMission", Kind: kindIntArray, Ints: []int32{32}},
			{Name: "AddHero", Kind: kindIntArray, Ints: []int32{22}},
		}},
		{Name: "Mission31", Kind: kindDir, Children: []synth.RegNode{{Name: "Payment", Kind: kindInt, Int: 700}}},
		{Name: "Mission32", Kind: kindDir, Children: []synth.RegNode{{Name: "Payment", Kind: kindInt, Int: 800}}},
		{Name: "Mission40", Kind: kindDir, Children: []synth.RegNode{
			{Name: "InnNPC", Kind: kindIntArray, Ints: []int32{7}},
			{Name: "InnMission", Kind: kindIntArray, Ints: []int32{40}},
		}},
	})
}

func graphCampaign(t *testing.T) Campaign {
	t.Helper()
	r, err := reg.Parse(graphRegistry())
	if err != nil {
		t.Fatal(err)
	}
	return ReadCampaign(r)
}

// graphModMissions is a mod over graphRegistry holding one row of each
// change the graph marks: mission 20 on another map, mission 31 leading to
// an added mission 41, and a guest mission 25 played after mission 20.
const graphModMissions = `[[mission]]
kind = "replace"
key = "ambush"
number = 20
map = "mission/{lang}/map.alm"
[[mission]]
kind = "campaign"
key = "shop"
number = 31
AutoGetMission = 41
[[mission]]
kind = "campaign"
key = "cellar"
number = 41
map = "mission/{lang}/map.alm"
[[mission]]
key = "visit"
number = 25
map = "mission/{lang}/map.alm"
after = 20
guest = 22
chapter = 30
`

// graphLines is a graph as text: a line a node with its column, row and
// marks, then a line an edge, a mod's edge ending in "(mod)".
func graphLines(page ui.CampaignGraphPage) []string {
	label := func(i int) string {
		n := page.Nodes[i]
		switch n.Kind {
		case ui.GraphStart:
			return "start"
		case ui.GraphTown:
			return fmt.Sprintf("town %d", n.Number)
		case ui.GraphMission:
			return fmt.Sprintf("mission %d", n.Number)
		}
		switch n.Place {
		case ui.GraphShop:
			return fmt.Sprintf("shop row %d", n.Row)
		case ui.GraphSchool:
			return fmt.Sprintf("school row %d", n.Row)
		}
		return fmt.Sprintf("inn %d", n.Number)
	}
	var out []string
	for i, n := range page.Nodes {
		line := fmt.Sprintf("%s at %d,%d", label(i), n.Column, n.Row)
		for _, m := range []struct {
			on   bool
			name string
		}{{n.Empty, "empty"}, {n.MapMark, "map"}, {n.FlowMark, "flow"}, {n.AddedMark, "added"}, {n.PartyMark, "party"}} {
			if m.on {
				line += " " + m.name
			}
		}
		out = append(out, line)
	}
	for _, e := range page.Edges {
		line := label(e.From) + " > " + label(e.To)
		if e.Marked {
			line += " (mod)"
		}
		if e.Heroes != 0 {
			line += " for " + ui.HeroWord(e.Heroes)
		}
		out = append(out, line)
	}
	return out
}

func wantGraph(t *testing.T, page ui.CampaignGraphPage, want []string) {
	t.Helper()
	if got := graphLines(page); !slices.Equal(got, want) {
		t.Fatalf("graph:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// The graph of a campaign holds the start, each mission, each chapter's
// town and each hand-out; an arrow for a mission that leads to another with
// no town between, for a win that opens a town, and from a hand-out to its
// mission. A side mission returns to the town that handed it out.
func TestCampaignGraphHoldsTheFlowTheTownsAndEachHandout(t *testing.T) {
	c := graphCampaign(t)
	wantGraph(t, campaignGraphOf(c, c, mod.MissionData{}, nil), []string{
		"start at 0,0",
		"mission 10 at 2,1",
		"mission 20 at 2,2",
		"town 30 at 0,3",
		"inn 5 at 1,3",
		"mission 30 at 2,3",
		"town 40 at 0,7",
		"inn 6 at 1,4 empty",
		"shop row 5 at 1,5",
		"mission 31 at 3,3",
		"school row 6 at 1,6",
		"mission 32 at 4,3",
		"inn 7 at 1,7",
		"mission 40 at 2,7",
		"start > mission 10",
		"mission 10 > mission 20",
		"mission 20 > town 30",
		"town 30 > inn 5",
		"inn 5 > mission 30",
		"mission 30 > town 40",
		"town 30 > inn 6",
		"town 30 > shop row 5",
		"shop row 5 > mission 31",
		"mission 31 > town 30",
		"town 30 > school row 6",
		"school row 6 > mission 32",
		"mission 32 > town 30",
		"town 40 > inn 7",
		"inn 7 > mission 40",
	})
}

// Each change of a mod is a mark: a replaced map, an overridden flow, an
// added mission and a guest party, and the arrows the mod set. The added
// mission's win is set to open the town, so it opens the next chapter's.
func TestCampaignGraphMarksWhatAModChanges(t *testing.T) {
	stock := graphCampaign(t)
	d, err := mod.ParseMissions("marks", mod.MissionsFile, []byte(graphModMissions), func(string) (string, bool) { return "", false }, "en")
	if err != nil {
		t.Fatal(err)
	}
	c, err := stock.withModMissions(d)
	if err != nil {
		t.Fatal(err)
	}
	wantGraph(t, campaignGraphOf(stock, c, d, nil), []string{
		"start at 0,0",
		"mission 10 at 2,1",
		"mission 20 at 2,2 map",
		"mission 25 at 3,2 added party",
		"town 30 at 0,3",
		"inn 5 at 1,3",
		"mission 30 at 2,3",
		"town 40 at 0,7",
		"inn 6 at 1,4 empty",
		"shop row 5 at 1,5",
		"mission 31 at 3,3 flow",
		"mission 41 at 3,7 added",
		"school row 6 at 1,6",
		"mission 32 at 4,3",
		"inn 7 at 1,7",
		"mission 40 at 2,7",
		"start > mission 10",
		"mission 10 > mission 20",
		"mission 20 > mission 25 (mod)",
		"mission 25 > town 30 (mod)",
		"town 30 > inn 5",
		"inn 5 > mission 30",
		"mission 30 > town 40",
		"town 30 > inn 6",
		"town 30 > shop row 5",
		"shop row 5 > mission 31",
		"mission 31 > mission 41 (mod)",
		"mission 41 > town 40 (mod)",
		"town 30 > school row 6",
		"school row 6 > mission 32",
		"mission 32 > town 30",
		"town 40 > inn 7",
		"inn 7 > mission 40",
	})

	// A mod that ends mission 10's lead to mission 20 sends its win to a town.
	off, err := mod.ParseMissions("off", mod.MissionsFile, []byte("[[mission]]\nkind = \"campaign\"\nkey = \"first\"\nnumber = 10\nAutoGetMission = -1\n"),
		func(string) (string, bool) { return "", false }, "en")
	if err != nil {
		t.Fatal(err)
	}
	if c, err = stock.withModMissions(off); err != nil {
		t.Fatal(err)
	}
	lines := graphLines(campaignGraphOf(stock, c, off, nil))
	for _, want := range []string{"mission 10 at 2,1 flow", "mission 10 > town 20 (mod)"} {
		if !slices.Contains(lines, want) {
			t.Fatalf("no %q in:\n%s", want, strings.Join(lines, "\n"))
		}
	}
	if slices.Contains(lines, "mission 10 > mission 20") {
		t.Fatal("the lead the mod ended is still drawn")
	}
}

// A graph's places come from its data: two builds of one campaign are equal
// and no two nodes share a cell.
func TestCampaignGraphLayoutIsTheSameOverTwoBuilds(t *testing.T) {
	stock := graphCampaign(t)
	d, err := mod.ParseMissions("marks", mod.MissionsFile, []byte(graphModMissions), func(string) (string, bool) { return "", false }, "en")
	if err != nil {
		t.Fatal(err)
	}
	c, err := stock.withModMissions(d)
	if err != nil {
		t.Fatal(err)
	}
	first := campaignGraphOf(stock, c, d, nil)
	for range 8 { // the campaign's maps are walked in a new order each time
		if next := campaignGraphOf(stock, c, d, nil); !reflect.DeepEqual(first, next) {
			t.Fatalf("two builds differ:\n%s\n\n%s", strings.Join(graphLines(first), "\n"), strings.Join(graphLines(next), "\n"))
		}
	}
	cells := map[[2]int]bool{}
	for _, n := range first.Nodes {
		if at := [2]int{n.Column, n.Row}; cells[at] {
			t.Fatalf("two nodes stand at column %d, row %d", n.Column, n.Row)
		} else {
			cells[at] = true
		}
	}
}

// graphFixture is an editor with no map open over the test play install,
// whose campaign is graphRegistry and whose map list holds missions 10, 20
// and 31.
func graphFixture(t *testing.T) (*testPlayFixture, *ui.MapEditor) {
	t.Helper()
	return graphFixtureIn(t, "", nil, []string{"Ford", "Ambush", "Vault"})
}

// graphFixtureIn is graphFixture in the install language entry, its three
// maps named as given, drawn in the fonts of the install a when there is one.
func graphFixtureIn(t *testing.T, entry string, a *Archives, names []string) (*testPlayFixture, *ui.MapEditor) {
	t.Helper()
	return graphFixtureDrawn(t, entry, a, nil, nil, names)
}

// graphFixtureDrawn is graphFixtureIn drawn in font when it is set, with
// the install's video archive holding films, the stand-in bytes of each
// named film, when films is set.
func graphFixtureDrawn(t *testing.T, entry string, a *Archives, font *text.Font, films []string, names []string) (*testPlayFixture, *ui.MapEditor) {
	t.Helper()
	f := newTestPlayFixture(t)
	x := f.x
	x.tiles, x.font = &terrain.Tileset{}, &text.Font{Selector: 1}
	if font != nil {
		x.font, x.cardFont, x.body = font, font, font
	}
	if films != nil {
		var files []synth.File
		for _, name := range films {
			files = append(files, synth.File{Path: strings.ToUpper(name), Data: []byte("film stand-in " + name)})
		}
		if err := os.MkdirAll(filepath.Join(f.root, "ALLODS"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(f.root, "ALLODS", "VIDEO4.RES"), synth.Archive(files), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	x.archives.Base.Profile.Language = entry
	if a != nil {
		font, err := LoadFont(a.Containers, DefaultFont, FontShades)
		if err != nil {
			t.Fatal(err)
		}
		font.Selector = LanguageSelector(a.Containers)
		x.font, x.cardFont = font, font
		if body, err := LoadFont(a.Containers, DocumentFont, FontCoverage); err == nil {
			body.Selector = font.Selector
			x.body = body
		}
	}
	withInstallTable(x)
	x.archives.Containers.Overlay(ScenarioRegistry, func() ([]byte, error) { return graphRegistry(), nil })
	for i, name := range names {
		address, _ := MissionMap([]int{10, 20, 31}[i])
		field, err := alm.MapNameText.Encode(name)
		if err != nil {
			t.Fatal(err)
		}
		b := synth.ALM(synth.ALMOptions{Width: 8, Height: 8, Name: string(field)})
		x.archives.Containers.Overlay(address, func() ([]byte, error) { return b, nil })
		x.Maps = append(x.Maps, MapEntry{Source: strings.TrimPrefix(address, scenarioPrefix), FromArchive: true, Name: string(field)})
	}
	e := x.Editor()
	e.Layout(1920, 1080)
	return f, e
}

// graphMod writes a mod over graphRegistry with the given missions and
// their one map.
func graphMod(t *testing.T, missions string) string {
	t.Helper()
	dir := writeEditorMod(t, t.TempDir(), missions)
	writeMissionMap(t, dir)
	return dir
}

// showGraph turns the open W, or opens it and turns it, to its graph page.
func showGraph(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	for range 4 {
		if texts, _ := e.PopupCanvasBoxes(); e.ModWindowOpen() && len(texts) > 0 {
			return
		}
		if err := e.Key("package"); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatalf("W shows no graph page: %q", e.PopupLines())
}

// clickGraphBox clicks the box of the shown graph whose text is want.
func clickGraphBox(t *testing.T, e *ui.MapEditor, want string) {
	t.Helper()
	texts, _ := e.PopupCanvasBoxes()
	i := slices.Index(texts, want)
	if i < 0 {
		t.Fatalf("no box %q in %q", want, texts)
	}
	clickCanvasBox(t, e, i)
}

// clickCanvasBox clicks box i of the shown graph, first turning the wheel
// over the graph, as a user does, while the box's middle lies above or
// below the graph's view.
func clickCanvasBox(t *testing.T, e *ui.MapEditor, i int) {
	t.Helper()
	for range 40 {
		_, rects := e.PopupCanvasBoxes()
		view := e.PopupCanvasView()
		if i < 0 || i >= len(rects) || view.Empty() {
			t.Fatalf("no box %d on the graph", i)
		}
		p := rects[i].Min.Add(rects[i].Size().Div(2))
		if p.In(view) {
			e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Unix(1, 0))
			e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
			return
		}
		if p.X < view.Min.X || p.X >= view.Max.X {
			t.Fatalf("box %d at %v stands beside the view %v", i, rects[i], view)
		}
		at := view.Min.Add(view.Size().Div(2))
		turn := ui.Input{CursorX: at.X, CursorY: at.Y, WheelY: -1}
		if p.Y < view.Min.Y {
			turn.WheelY = 1
		}
		e.Pointer(turn, time.Unix(1, 0))
	}
	t.Fatalf("box %d never came into the graph's view", i)
}

func openSource(e *ui.MapEditor) string {
	if e.Document() == nil {
		return ""
	}
	return e.Document().Source
}

// With no mod open W shows the install's campaign, and a click on a mission
// opens the map the map list opens for it; a mission the install lists no
// map of opens nothing.
func TestEditorGraphOfTheInstallOpensAMissionsInstallMap(t *testing.T) {
	f, e := graphFixture(t)
	showGraph(t, e)
	page := f.x.modHost().(ui.ModCampaignGraph).CampaignGraph()
	if page.Mod || page.Refused || page.Unread {
		t.Fatalf("the install's graph is flagged: mod %t, refused %t, unread %t", page.Mod, page.Refused, page.Unread)
	}
	c := graphCampaign(t)
	if got, want := graphLines(page), graphLines(campaignGraphOf(c, c, mod.MissionData{}, nil)); !slices.Equal(got, want) {
		t.Fatalf("graph:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	lines := strings.Join(e.PopupLines(), "\n")
	for _, want := range []string{"The campaign of the install.", "Start", "Mission 10 Ford", "Town 30", "Tavern, NPC 5", "Tavern, NPC 6: nothing", "Shop", "School"} {
		if !strings.Contains(lines, want) {
			t.Fatalf("no %q on the page:\n%s", want, lines)
		}
	}
	clickGraphBox(t, e, "Mission 30")
	if e.Document() != nil {
		t.Fatalf("a mission with no install map opened %q", openSource(e))
	}
	clickGraphBox(t, e, "Mission 10 Ford")
	if got := openSource(e); got != "scenario/10.alm" || e.Document().Name != "Ford" {
		t.Fatalf("the click opened %q", got)
	}
}

// With a mod open W shows the campaign of the mod's admitted fork with its
// marks, and a click on a mission the mod holds the map of opens the mod's
// map as the row of W's mission list does; any other mission opens its
// install map.
func TestEditorGraphOfAModOpensTheModsMap(t *testing.T) {
	f, e := graphFixture(t)
	dir := graphMod(t, graphModMissions)
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	if f.x.mod == nil || f.x.mod.dir != dir {
		t.Fatal("the mod is not open")
	}
	if f.x.mod.refusal != "" || f.x.mod.contextIssue != "" {
		t.Fatalf("the mod is refused: %q %q", f.x.mod.refusal, f.x.mod.contextIssue)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	page := s.CampaignGraph()
	if !page.Mod || page.Refused {
		t.Fatalf("the mod's graph is flagged: mod %t, refused %t", page.Mod, page.Refused)
	}
	stock := graphCampaign(t)
	d, err := mod.ParseMissions("ambush", mod.MissionsFile, []byte(graphModMissions), func(string) (string, bool) { return "", false }, "en")
	if err != nil {
		t.Fatal(err)
	}
	c, err := stock.withModMissions(d)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := graphLines(page), graphLines(campaignGraphOf(stock, c, d, nil)); !slices.Equal(got, want) {
		t.Fatalf("graph:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	rows := map[int]int{}
	for _, n := range page.Nodes {
		if n.Kind == ui.GraphMission && (n.ModMission != 0 || n.Number == 20) {
			rows[n.Number] = n.ModMission
			if n.Source != "" {
				t.Fatalf("mission %d of the mod still offers the install map %q", n.Number, n.Source)
			}
		}
	}
	if want := map[int]int{20: 1, 41: 3, 25: 4}; !reflect.DeepEqual(rows, want) {
		t.Fatalf("mod mission rows %v, want %v", rows, want)
	}

	showGraph(t, e)
	lines := strings.Join(e.PopupLines(), "\n")
	for _, want := range []string{"The campaign with the open mod's changes.", "Mission 20 Ambush map", "Mission 31 Vault flow", "Mission 41 new", "Mission 25 new party",
		"the mod replaces this mission's map", "the mod adds this mission", "a guest party of the mod plays this mission"} {
		if !strings.Contains(lines, want) {
			t.Fatalf("no %q on the page:\n%s", want, lines)
		}
	}
	modMap := filepath.Join(dir, "mission", "en", "map.alm")
	clickGraphBox(t, e, "Mission 41 new")
	if got := openSource(e); got != modMap || f.x.mod.selected != "cellar" {
		t.Fatalf("the added mission opened %q as %q", got, f.x.mod.selected)
	}
	showGraph(t, e)
	clickGraphBox(t, e, "Mission 20 Ambush map")
	if got := openSource(e); got != modMap || f.x.mod.selected != "ambush" {
		t.Fatalf("the replaced mission opened %q as %q", got, f.x.mod.selected)
	}
	showGraph(t, e)
	clickGraphBox(t, e, "Mission 10 Ford")
	if got := openSource(e); got != "scenario/10.alm" {
		t.Fatalf("an install mission opened %q", got)
	}
}

// A mod whose missions the game refuses shows the install's campaign and
// says so.
func TestEditorGraphOfARefusedModIsTheInstalls(t *testing.T) {
	f, e := graphFixture(t)
	dir := graphMod(t, "[[mission]]\nkind = \"replace\"\nkey = \"nowhere\"\nnumber = 90\nmap = \"mission/{lang}/map.alm\"\n")
	if err := f.x.SetMod(dir); err != nil {
		t.Fatal(err)
	}
	page := f.x.modHost().(ui.ModCampaignGraph).CampaignGraph()
	if !page.Mod || !page.Refused {
		t.Fatalf("mod %t, refused %t: %q", page.Mod, page.Refused, f.x.mod.refusal)
	}
	c := graphCampaign(t)
	if got, want := graphLines(page), graphLines(campaignGraphOf(c, c, mod.MissionData{}, nil)); !slices.Equal(got, want) {
		t.Fatalf("graph:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	showGraph(t, e)
	if lines := strings.Join(e.PopupLines(), "\n"); !strings.Contains(lines, "The game refuses this mod's missions") {
		t.Fatalf("the page does not say the mod is refused:\n%s", lines)
	}
	// A mod the game does not refuse, whose campaign was not built, shows
	// the install's campaign too and says that instead.
	if err := f.x.SetMod(graphMod(t, graphModMissions)); err != nil {
		t.Fatal(err)
	}
	f.x.mod.flow = nil
	page = f.x.modHost().(ui.ModCampaignGraph).CampaignGraph()
	if !page.Mod || page.Refused || !page.Unbuilt {
		t.Fatalf("mod %t, refused %t, unbuilt %t", page.Mod, page.Refused, page.Unbuilt)
	}
	if got, want := graphLines(page), graphLines(campaignGraphOf(c, c, mod.MissionData{}, nil)); !slices.Equal(got, want) {
		t.Fatalf("graph of the unbuilt mod:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if lines := strings.Join(e.PopupLines(), "\n"); !strings.Contains(lines, "The open mod's campaign could not be built") {
		t.Fatalf("the page does not say the mod's campaign was not built:\n%s", lines)
	}
}

// Opening the graph page, moving it and leaving it changes nothing: the mod
// folder, the profile, the document, its draft and its undo history stay as
// they were.
func TestEditorGraphPageChangesNothing(t *testing.T) {
	f, e := graphFixture(t)
	dir := graphMod(t, graphModMissions)
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	if err := e.Key("escape"); err != nil {
		t.Fatal(err)
	}
	doc := e.Document()
	s := doc.Edits.(*mapInspectionEdits)
	files, profile := tree(t, dir), tree(t, f.profile)
	model, source, workspace := bytes.Clone(s.model.Bytes()), bytes.Clone(doc.SourceBytes), cloneWorkspace(s.workspace)
	held, context := s.workspace, f.x.mod
	if e.Dirty() || s.CanUndo() || s.CanRedo() {
		t.Fatal("the opened mod is already edited")
	}

	showGraph(t, e)
	for _, key := range []string{"down", "pagedown", "end", "right", "left", "up", "pageup", "home", "tab", "backtab"} {
		if err := e.Key(key); err != nil {
			t.Fatalf("%s: %v", key, err)
		}
	}
	if texts, _ := e.PopupCanvasBoxes(); len(texts) == 0 {
		t.Fatal("the keys left the graph page")
	}
	e.ChromeImage()
	if err := e.Key("package"); err != nil { // on to the fields page
		t.Fatal(err)
	}
	if err := e.Key("escape"); err != nil {
		t.Fatal(err)
	}
	if e.ModWindowOpen() {
		t.Fatal("W is still open")
	}

	if e.Document() != doc || doc.Edits != ui.InspectionEdits(s) {
		t.Fatal("the page replaced the document")
	}
	if e.Dirty() || s.CanUndo() || s.CanRedo() {
		t.Fatalf("the page edited the draft: dirty %t, undo %t, redo %t", e.Dirty(), s.CanUndo(), s.CanRedo())
	}
	if !bytes.Equal(s.model.Bytes(), model) || !bytes.Equal(doc.SourceBytes, source) {
		t.Fatal("the page changed the map")
	}
	if s.workspace != held || !reflect.DeepEqual(s.workspace, workspace) || f.x.mod != context {
		t.Fatal("the page changed the mod's draft")
	}
	if got := tree(t, dir); !reflect.DeepEqual(got, files) {
		t.Fatal("the page wrote into the mod folder")
	}
	if got := tree(t, f.profile); !reflect.DeepEqual(got, profile) {
		t.Fatalf("the page wrote into the profile: %v", got)
	}
}
