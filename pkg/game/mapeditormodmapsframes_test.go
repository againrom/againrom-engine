package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/internal/witness"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/words"
)

// W's list of the mod's maps and the Open window's group of them draw every
// map with its role in each language of the word book, at the small window
// and the large: every word of every role stands whole, inside the window.
// With AGAINROM_MODMAPS_FRAMES naming a folder the frames are written there,
// in the fonts and the language of the install AGAINROM_ASSETS names when it
// names one.
func TestEditorModMapListDrawsInEachLanguage(t *testing.T) {
	dir, entries := os.Getenv("AGAINROM_MODMAPS_FRAMES"), []string{"english", "russian"}
	var fonts *Archives
	if root := os.Getenv("AGAINROM_ASSETS"); dir != "" && root != "" {
		a, err := OpenArchives(root)
		if err != nil {
			t.Fatal(err)
		}
		fonts, entries = a, []string{a.Base.Profile.Language}
	}
	roles := map[string][]string{
		"en": {"guest 21 after 20, guest 22 after 21", "missing", "replaces 20 by name", "unused"},
		"ru": {"гостевая 21 после 20, гостевая 22 после 21", "файла нет", "заменяет 20 по имени", "не используется"},
	}
	for _, entry := range entries {
		lang := words.Language(entry)
		for _, size := range []image.Point{{X: 640, Y: 480}, {X: 1280, Y: 800}} {
			t.Run(fmt.Sprintf("%s-%dx%d", lang, size.X, size.Y), func(t *testing.T) {
				f := newTestPlayFixture(t)
				f.x.tiles, f.x.font = &terrain.Tileset{}, &text.Font{Selector: 1}
				f.x.archives.Base.Profile.Language = entry
				if fonts != nil {
					font, err := LoadFont(fonts.Containers, DefaultFont, FontShades)
					if err != nil {
						t.Fatal(err)
					}
					font.Selector = LanguageSelector(fonts.Containers)
					f.x.font, f.x.cardFont = font, font
					if body, err := LoadFont(fonts.Containers, DocumentFont, FontCoverage); err == nil {
						body.Selector = font.Selector
						f.x.body = body
					}
				}
				e := f.x.Editor()
				e.SetWords(words.For(entry))
				e.Layout(size.X, size.Y)
				modMapsMod(t, e, synth.ALM(synth.ALMOptions{Width: 8, Height: 8, Name: "mission"}))
				drawn := f.x.inspectionUTF8
				// whole checks each map the lines show: every word of its role
				// stands whole. With all, every map of the mod is shown; without,
				// the map shown and as many more as the window has room for.
				whole := func(what string, lines []string, all bool) {
					t.Helper()
					have := map[string]bool{}
					for _, w := range strings.Fields(strings.Join(lines, " ")) {
						have[w] = true
					}
					for i, name := range []string{"map1.alm", "gone.alm", "20.alm", "spare.alm"} {
						role := roles[lang][i]
						if !have[name] && !have[name+":"] {
							if all || i == 0 {
								t.Fatalf("%s shows no %s:\n%s", what, name, strings.Join(lines, "\n"))
							}
							continue
						}
						for _, w := range strings.Fields(drawn(role)) {
							if !have[w] {
								t.Fatalf("%s does not show %q of %q whole:\n%s", what, w, role, strings.Join(lines, "\n"))
							}
						}
					}
				}
				shot := func(name string) {
					t.Helper()
					pic := e.ChromeImage()
					if !pic.Bounds().Eq(image.Rectangle{Max: size}) {
						t.Fatalf("%s is drawn %v in a window of %v", name, pic.Bounds(), size)
					}
					if dir != "" {
						witness.PNG(t, filepath.Join(dir, fmt.Sprintf("%s-%s-%dx%d.png", name, lang, size.X, size.Y)), pic)
					}
				}

				if !e.ModWindowOpen() {
					if err := e.Key("package"); err != nil {
						t.Fatal(err)
					}
				}
				modMapsPage(t, e)
				whole("W", e.PopupLines(), true)
				shot("mod-maps")

				_ = e.Key("escape")
				if err := e.Key("catalog"); err != nil {
					t.Fatal(err)
				}
				lines := e.ListLines()
				if len(lines) < 6 || strings.TrimSpace(lines[0]) != drawn(words.For(entry).Text("editor.mod.maps")) {
					t.Fatalf("the Open window's first group: %q", lines)
				}
				whole("the Open window", lines[:6], true)
				shot("open-maps")
			})
		}
	}
}
