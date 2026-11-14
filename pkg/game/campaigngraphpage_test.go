package game

import (
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/internal/witness"
	"againrom/pkg/words"
)

// W's graph page draws the install's campaign and a mod's with its marks in
// each language of the word book: its caption, its boxes inside the window
// and, for a mod, its legend. With AGAINROM_GRAPH_FRAMES naming a folder the
// frames are written there, in the fonts and the language of the install
// AGAINROM_ASSETS names when it names one.
func TestEditorGraphPageDrawsInEachLanguage(t *testing.T) {
	dir, entries := os.Getenv("AGAINROM_GRAPH_FRAMES"), []string{"english", "russian"}
	var fonts *Archives
	if root := os.Getenv("AGAINROM_ASSETS"); dir != "" && root != "" {
		a, err := OpenArchives(root)
		if err != nil {
			t.Fatal(err)
		}
		fonts, entries = a, []string{a.Base.Profile.Language}
	}
	for _, entry := range entries {
		lang, names := words.Language(entry), []string{"Ford", "Ambush", "Vault"}
		if lang == "ru" {
			names = []string{"Брод", "Засада", "Подвал"}
		}
		for _, missions := range []string{"", graphModMissions} {
			name, caption := "graph-"+lang, "editor.graph.install"
			if missions != "" {
				name, caption = name+"-mod", "editor.graph.mod"
			}
			t.Run(name, func(t *testing.T) {
				f, e := graphFixtureIn(t, entry, fonts, names)
				e.Layout(1280, 800)
				if missions != "" {
					folder := graphMod(t, missions)
					writeModText(t, folder, "mission/"+lang+"/map.alm", string(synth.ALM(synth.ALMOptions{Width: 8, Height: 8, Name: "mission"})))
					if err := workspaceField(e, "package", 0, folder); err != nil {
						t.Fatal(err)
					}
					if f.x.mod == nil || f.x.mod.refusal != "" || f.x.mod.contextIssue != "" {
						t.Fatal("the mod is not open as the game admits it")
					}
				}
				showGraph(t, e)
				book := words.For(entry)
				drawn := func(id string) string { return f.x.inspectionUTF8(book.Text(id)) }
				lines := strings.Join(e.PopupLines(), "\n")
				first, _, _ := strings.Cut(drawn(caption), ".")
				want := []string{first, drawn("editor.graph.start")}
				if missions != "" {
					want = append(want, drawn("editor.graph.legend.map"), drawn("editor.graph.legend.added"), drawn("editor.graph.legend.party"))
				}
				for _, w := range want {
					if !strings.Contains(lines, w) {
						t.Fatalf("no %q on the page:\n%s", w, lines)
					}
				}
				pic := e.ChromeImage()
				texts, rects := e.PopupCanvasBoxes()
				if i := slices.Index(texts, drawn("editor.graph.start")); i < 0 {
					t.Fatalf("no start box among %q", texts)
				}
				for i, r := range rects {
					if r.Empty() {
						continue // scrolled out of the canvas
					}
					if !r.Min.In(image.Rect(0, 0, 1280, 800)) || pic.RGBAAt(r.Min.X, r.Min.Y).A == 0 {
						t.Fatalf("box %q at %v is not drawn inside the window", texts[i], r)
					}
				}
				if dir != "" {
					witness.PNG(t, filepath.Join(dir, name+".png"), pic)
				}
			})
		}
	}
}
