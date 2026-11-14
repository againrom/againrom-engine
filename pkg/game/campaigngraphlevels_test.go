package game

import (
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/bitmapfont/v4"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"golang.org/x/text/encoding/charmap"

	"againrom/internal/synth"
	"againrom/internal/witness"
	"againrom/pkg/mod"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// graphFrameFont is a font of the bitmap face the module carries, one glyph
// a byte of the code page the inspector writes its text in, so a frame of
// synthetic content reads in both languages without an installed font.
func graphFrameFont() *text.Font {
	face := bitmapfont.Face
	m := face.Metrics()
	h := (m.Ascent + m.Descent).Ceil()
	f := &text.Font{}
	for b := text.FirstChar; b < 256; b++ {
		r := charmap.Windows1251.DecodeByte(byte(b))
		advance, ok := face.GlyphAdvance(r)
		w := advance.Ceil()
		if !ok || w <= 0 {
			w = h / 2
		}
		pic := image.NewAlpha(image.Rect(0, 0, w, h))
		d := font.Drawer{Dst: pic, Src: image.Opaque, Face: face, Dot: fixed.Point26_6{Y: m.Ascent}}
		d.DrawString(string(r))
		g := text.Glyph{Width: w, Height: h, Advance: w, Pixels: make([]text.Pixel, w*h)}
		for y := range h {
			for x := range w {
				if pic.AlphaAt(x, y).A > 0x7f {
					g.Pixels[y*w+x] = text.Pixel{Level: text.MaxLevel, Painted: true}
				}
			}
		}
		f.Glyphs = append(f.Glyphs, g)
	}
	return f
}

// graphKindKeys are the word-book keys of the legend's kinds, in the order
// of the colour table.
var graphKindKeys = []string{"start", "main", "side", "town", "talk", "handout", "film"}

// The Campaign graph page of graphRegistry, whose install holds a start
// film and mission 10's victory film: the missions of one ten share a row in
// number order and the next ten stands below; each box carries its kind's
// colour from the one table; the legend names each kind shown in table
// order; a film node names its file in the first column at its mission's row. An install whose
// video archive holds neither film shows no film node and no film in the
// legend. In English and Russian. With AGAINROM_GRAPH_LEVEL_FRAMES naming a
// folder the frames of the page, of the install and of a mod, are written
// there.
func TestCampaignGraphStandsByTensInKindColoursWithFilms(t *testing.T) {
	dir := os.Getenv("AGAINROM_GRAPH_LEVEL_FRAMES")
	for _, entry := range []string{"english", "russian"} {
		lang, names := words.Language(entry), []string{"Ford", "Ambush", "Vault"}
		if lang == "ru" {
			names = []string{"Брод", "Засада", "Подвал"}
		}
		book := words.For(entry)
		for _, tc := range []struct {
			name  string
			films []string
			mod   string
		}{
			{"films", []string{"start/01.smk", "m10/01.smk"}, ""},
			{"no-films", []string{"intro/01.smk"}, ""},
			{"mod", []string{"start/01.smk", "m10/01.smk"}, graphModMissions},
		} {
			t.Run(lang+"-"+tc.name, func(t *testing.T) {
				f, e := graphFixtureDrawn(t, entry, nil, graphFrameFont(), tc.films, names)
				if tc.mod != "" {
					folder := graphMod(t, tc.mod)
					writeModText(t, folder, "mission/"+lang+"/map.alm", string(synth.ALM(synth.ALMOptions{Width: 8, Height: 8, Name: "mission"})))
					if err := workspaceField(e, "package", 0, folder); err != nil {
						t.Fatal(err)
					}
					if f.x.mod == nil || f.x.mod.refusal != "" || f.x.mod.contextIssue != "" {
						t.Fatal("the mod is not open as the game admits it")
					}
				}
				showGraph(t, e)
				drawn := func(id string) string { return f.x.inspectionUTF8(book.Text(id)) }
				page := f.x.modHost().(ui.ModCampaignGraph).CampaignGraph()
				mission := map[int]ui.CampaignGraphNode{}
				var films []ui.CampaignGraphNode
				for _, n := range page.Nodes {
					switch n.Kind {
					case ui.GraphMission:
						mission[n.Number] = n
					case ui.GraphFilm:
						films = append(films, n)
					}
				}
				wantGraphLevels(t, mission)
				wantGraphFilms(t, page, mission, films, tc.name != "no-films")

				// The legend names each kind the page shows, in table order.
				var want []string
				for _, key := range graphKindKeys {
					if key != "film" || tc.name != "no-films" {
						want = append(want, drawn("editor.graph.kind."+key))
					}
				}
				lines := e.PopupLines()
				at := slices.Index(lines, want[0])
				if at < 0 || at+len(want) > len(lines) || !slices.Equal(lines[at:at+len(want)], want) {
					t.Fatalf("the legend is not %q in:\n%s", want, strings.Join(lines, "\n"))
				}
				if tc.name == "no-films" && slices.Contains(lines, drawn("editor.graph.kind.film")) {
					t.Fatal("the legend names films the page does not show")
				}

				// Each box shown carries its kind's colour in its band.
				// The sheet is larger than the view: the graph is moved down a
				// page at a time, then right, until every kind has shown a box.
				top, need := e.ChromeImage(), len(graphKindKeys)
				if tc.name == "no-films" {
					need--
				}
				pans := []string{""}
				for range 3 {
					pans = append(pans, slices.Repeat([]string{"pagedown"}, 5)...)
					pans = append(pans, "home", "right", "right", "right", "right")
				}
				seen := map[string]bool{}
				for _, key := range pans {
					if key != "" {
						if err := e.Key(key); err != nil {
							t.Fatal(err)
						}
					}
					pic := e.ChromeImage()
					_, rects := e.PopupCanvasBoxes()
					view := e.PopupCanvasView()
					for i, n := range page.Nodes {
						r := rects[i]
						p := image.Pt(r.Min.X+3, r.Min.Y+2)
						if !p.In(view) {
							continue
						}
						if got := pic.RGBAAt(p.X, p.Y); got != ui.GraphInk(n) {
							t.Errorf("node %d (%d %d) has band %v, its kind's colour is %v", i, n.Kind, n.Number, got, ui.GraphInk(n))
						}
						seen[graphKindName(n)] = true
					}
					if len(seen) >= need {
						break
					}
				}
				for _, key := range graphKindKeys {
					if !seen[key] && (key != "film" || tc.name != "no-films") {
						t.Errorf("no %s box shown to check its colour", key)
					}
				}
				if dir != "" && tc.name != "no-films" {
					name := "graph-levels-" + lang
					if tc.mod != "" {
						name += "-mod"
					}
					witness.PNG(t, filepath.Join(dir, name+".png"), top)
				}
			})
		}
	}
}

// graphKindName is the legend key of node n's kind, read from the node's
// own fields apart from the painter's table.
func graphKindName(n ui.CampaignGraphNode) string {
	switch n.Kind {
	case ui.GraphStart:
		return "start"
	case ui.GraphTown:
		return "town"
	case ui.GraphFilm:
		return "film"
	case ui.GraphHandout:
		if n.Place == ui.GraphInn {
			return "talk"
		}
		return "handout"
	}
	if n.Side {
		return "side"
	}
	return "main"
}

// wantGraphLevels holds the missions of graphRegistry to their levels: 30,
// 31 and 32 share a row in number order, 10, 20, 30 and 40 each stand below
// the ten before, and the campaign's main missions are those numbered by
// tens.
func wantGraphLevels(t *testing.T, mission map[int]ui.CampaignGraphNode) {
	t.Helper()
	for _, n := range []int{10, 20, 30, 31, 32, 40} {
		if _, ok := mission[n]; !ok {
			t.Fatalf("no mission %d among %v", n, mission)
		}
	}
	if a, b, c := mission[30], mission[31], mission[32]; a.Row != b.Row || b.Row != c.Row || !(a.Column < b.Column && b.Column < c.Column) {
		t.Errorf("the ten of 30 is not one row in number order: %d,%d %d,%d %d,%d", a.Column, a.Row, b.Column, b.Row, c.Column, c.Row)
	}
	for _, pair := range [][2]int{{10, 20}, {20, 30}, {30, 40}} {
		if mission[pair[0]].Row >= mission[pair[1]].Row {
			t.Errorf("mission %d stands on row %d, not below mission %d on row %d", pair[1], mission[pair[1]].Row, pair[0], mission[pair[0]].Row)
		}
	}
	for n, node := range mission {
		if node.Side != (n%10 != 0) {
			t.Errorf("mission %d is side %t", n, node.Side)
		}
	}
}

// TestCampaignGraphFilmsStandInTheirColumn holds the films of the start and
// of missions 10 and 20, on two levels, to the film column: one column
// directly left of the first mission column, each film at its node's row,
// and no level taller than without the films.
func TestCampaignGraphFilmsStandInTheirColumn(t *testing.T) {
	c := graphCampaign(t)
	shelf := func(directory string) []string {
		switch directory {
		case ui.StartCutscenes:
			return []string{"start/01.smk"}
		case ui.MissionCutscenes(10):
			return []string{"m10/01.smk"}
		case ui.MissionCutscenes(20):
			return []string{"m20/01.smk"}
		}
		return nil
	}
	with := campaignGraphOf(c, c, mod.MissionData{}, shelf)
	without := campaignGraphOf(c, c, mod.MissionData{}, nil)
	mission, plain := map[int]ui.CampaignGraphNode{}, map[int]ui.CampaignGraphNode{}
	var start ui.CampaignGraphNode
	rows := func(nodes []ui.CampaignGraphNode) int {
		r := 0
		for _, n := range nodes {
			if n.Kind != ui.GraphFilm {
				r = max(r, n.Row+1)
			}
		}
		return r
	}
	for _, n := range with.Nodes {
		switch n.Kind {
		case ui.GraphStart:
			start = n
		case ui.GraphMission:
			mission[n.Number] = n
		}
	}
	for _, n := range without.Nodes {
		if n.Kind == ui.GraphMission {
			plain[n.Number] = n
		}
	}
	if got, want := rows(with.Nodes), rows(without.Nodes); got != want {
		t.Errorf("the films make the graph %d rows tall, without them %d", got, want)
	}
	for n, m := range mission {
		if p := plain[n]; m.Row != p.Row || m.Column != p.Column {
			t.Errorf("the film moved mission %d to %d,%d from %d,%d", n, m.Column, m.Row, p.Column, p.Row)
		}
	}
	for _, e := range with.Edges {
		if with.Nodes[e.To].Kind == ui.GraphFilm || with.Nodes[e.From].Kind == ui.GraphFilm {
			t.Errorf("the edge %d to %d ties a film, want none", e.From, e.To)
		}
	}
	films := 0
	for _, f := range with.Nodes {
		if f.Kind != ui.GraphFilm {
			continue
		}
		films++
		if f.Column != 0 {
			t.Errorf("the film of %d stands in column %d, want the first column 0", f.Number, f.Column)
		}
		if !f.Dropped {
			t.Errorf("the film of %d is not dropped below its row", f.Number)
		}
		want := start.Row
		if f.Number != 0 {
			want = mission[f.Number].Row
		}
		if f.Row != want {
			t.Errorf("the film of %d stands on row %d, its node on row %d", f.Number, f.Row, want)
		}
	}
	if films != 3 {
		t.Errorf("%d film nodes, want 3", films)
	}
	if start.Row != 0 {
		t.Errorf("the start stands on row %d", start.Row)
	}
}

// wantGraphFilms holds the film nodes to the films the install holds: with
// them, the start's film and mission 10's, each naming its file, placed
// in the first column at its node's row, dropped below it; without them, none.
func wantGraphFilms(t *testing.T, page ui.CampaignGraphPage, mission map[int]ui.CampaignGraphNode, films []ui.CampaignGraphNode, held bool) {
	t.Helper()
	if !held {
		if len(films) != 0 {
			t.Fatalf("film nodes %v with no film the game finds", films)
		}
		return
	}
	if len(films) != 2 || films[0].Number != 0 || films[0].Title != "start/01.smk" || films[1].Number != 10 || films[1].Title != "m10/01.smk" {
		t.Fatalf("film nodes %v", films)
	}
	if m, f := mission[10], films[1]; f.Column != 0 || f.Row != m.Row || !f.Dropped {
		t.Errorf("mission 10's film stands at %d,%d, mission 10 at %d,%d", f.Column, f.Row, m.Column, m.Row)
	}
	tied := 0
	for _, e := range page.Edges {
		from, to := page.Nodes[e.From], page.Nodes[e.To]
		if to.Kind == ui.GraphFilm && (from.Kind == ui.GraphStart && to.Number == 0 || from.Kind == ui.GraphMission && from.Number == to.Number) {
			tied++
		}
	}
	if tied != 0 {
		t.Errorf("%d arrows tie a film, want none", tied)
	}
}

// TestCampaignGraphFilmsOfOneLevelStackDown holds the films of a level, one
// for each of its missions, to a stack down the film column from the row of
// the missions, in the missions' order; a level is as tall as its rows, and
// taller only when the stack runs past them.
func TestCampaignGraphFilmsOfOneLevelStackDown(t *testing.T) {
	c := graphCampaign(t)
	shelf := func(directory string) []string { return []string{directory + "/01.smk"} }
	with := campaignGraphOf(c, c, mod.MissionData{}, shelf)
	without := campaignGraphOf(c, c, mod.MissionData{}, nil)
	first, count := map[int]int{}, map[int]int{}
	for _, n := range with.Nodes {
		if n.Kind == ui.GraphMission {
			level := n.Number / mainMissionStride
			if _, ok := first[level]; !ok || n.Row < first[level] {
				first[level] = n.Row
			}
			count[level]++
		}
	}
	plain := map[int]int{}
	var levels []int
	for _, n := range without.Nodes {
		if n.Kind == ui.GraphMission {
			level := n.Number / mainMissionStride
			if _, ok := plain[level]; !ok {
				levels = append(levels, level)
				plain[level] = n.Row
			}
		}
	}
	slices.Sort(levels)
	stacked := false
	for _, n := range with.Nodes {
		if n.Kind != ui.GraphFilm || n.Number == 0 {
			continue
		}
		if count[n.Number/mainMissionStride] > 1 {
			stacked = true
		}
	}
	if !stacked {
		t.Skip("the fixture has no level with two missions")
	}
	for _, level := range levels {
		var rows []int
		for _, n := range with.Nodes {
			if n.Kind == ui.GraphFilm && n.Number != 0 && n.Number/mainMissionStride == level {
				rows = append(rows, n.Row)
			}
		}
		slices.Sort(rows)
		for k, r := range rows {
			if r != first[level]+k {
				t.Errorf("level %d: film %d on row %d, want %d", level, k, r, first[level]+k)
			}
		}
	}
	for k := 0; k+1 < len(levels); k++ {
		a, b := levels[k], levels[k+1]
		got, want := first[b]-first[a], max(plain[b]-plain[a], count[a])
		if got != want {
			t.Errorf("level %d is %d rows tall, want %d", a, got, want)
		}
	}
}
