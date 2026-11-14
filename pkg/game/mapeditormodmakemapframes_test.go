package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/witness"
	"againrom/pkg/render/text"
)

// The Missions page with an unused map and its actions, and the page after
// Make mission, in each language at the small window and the large. With
// AGAINROM_MAPACTIONS_FRAMES naming a folder the frames are written there.
func TestMissionsPageShowsTheUnusedMapActionsInEachLanguage(t *testing.T) {
	dir := os.Getenv("AGAINROM_MAPACTIONS_FRAMES")
	only := ""
	if root := os.Getenv("AGAINROM_ASSETS"); dir != "" && root != "" {
		a, err := OpenArchives(root)
		if err != nil {
			t.Fatal(err)
		}
		only = a.Base.Profile.Language
		font, err := LoadFont(a.Containers, DefaultFont, FontShades)
		if err != nil {
			t.Fatal(err)
		}
		font.Selector = LanguageSelector(a.Containers)
		body, err := LoadFont(a.Containers, DocumentFont, FontCoverage)
		if err != nil {
			body = font
		}
		body.Selector = font.Selector
		workspaceFonts = []*text.Font{font, body}
		t.Cleanup(func() { workspaceFonts = nil })
	}
	for _, lang := range makeMapLangs {
		if only != "" && only != map[string]string{"": "english", "ru": "russian"}[lang.book] {
			continue
		}
		name := map[string]string{"": "en", "ru": "ru"}[lang.book]
		for _, size := range []image.Point{{X: 640, Y: 480}, {X: 1280, Y: 800}} {
			t.Run(fmt.Sprintf("%s-%dx%d", name, size.X, size.Y), func(t *testing.T) {
				_, e, spare, _ := makeMapEditor(t, lang.book)
				e.Layout(size.X, size.Y)
				for i := 0; i < 6; i++ {
					if err := e.Key("package"); err != nil {
						t.Fatal(err)
					}
					if _, open := spare.ModMapActions(); open && e.ModWindowOpen() {
						break
					}
				}
				if _, open := spare.ModMapActions(); !open || !e.ModWindowOpen() {
					t.Fatal("the Missions page lists no map actions")
				}
				shot := func(what string) {
					t.Helper()
					pic := e.ChromeImage()
					if !pic.Bounds().Eq(image.Rectangle{Max: size}) {
						t.Fatalf("%s is drawn %v in a window of %v", what, pic.Bounds(), size)
					}
					if dir != "" {
						witness.PNG(t, filepath.Join(dir, fmt.Sprintf("%s-%s-%dx%d.png", what, name, size.X, size.Y)), pic)
					}
				}
				if size.X < 1000 {
					e.ScrollDetails(3)
				}
				shot("missions-unused")
				if _, err := spare.SetWorkspaceField("package", "makemission", "maps/spare.alm"); err != nil {
					t.Fatal(err)
				}
				if _, ok := actionOf(t, spare, "maps/spare.alm"); ok {
					t.Fatal("the unused entry stays after Make mission")
				}
				// Undo and Redo reload the page the way a button press does.
				for _, key := range []string{"undo", "redo"} {
					if err := e.Key(key); err != nil {
						t.Fatal(err)
					}
				}
				shot("missions-made")
			})
		}
	}
}
