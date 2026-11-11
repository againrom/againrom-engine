package game

import (
	"image/color"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// A speaker added to a mod that had none shows in the overview's file and
// kind lists at once, as a fresh open of the folder shows it.
func TestEditorOverviewListsTheSpeakerJustAdded(t *testing.T) {
	_, s, dir := modSpeakerEditor(t)
	x := s.inspector
	pngFile := filepath.Join(t.TempDir(), "face.png")
	speakerPNG(t, pngFile, color.RGBA{R: 10, G: 200, B: 30, A: 255})
	if _, err := s.AddSpeaker(speakerLady, "Lady", pngFile); err != nil {
		t.Fatal(err)
	}
	_, cold := speakerPickEditor(t)
	if err := cold.inspector.SetMod(dir); err != nil {
		t.Fatal(err)
	}
	for _, heading := range []string{"editor.mod.overview.kinds", "editor.mod.overview.files"} {
		got := overviewLines(s.ModOverview(), x.word(heading))
		want := overviewLines(cold.ModOverview(), x.word(heading))
		if !slices.Equal(got, want) {
			t.Errorf("%s after the add:\n%s\nfresh open:\n%s", heading, strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
	}
	if kinds := overviewLines(s.ModOverview(), x.word("editor.mod.overview.kinds")); !slices.ContainsFunc(kinds, func(l string) bool { return strings.HasPrefix(l, "speakers") }) {
		t.Errorf("no speakers kind: %v", kinds)
	}
}

// Opening a mod over an open map, and closing it, drops the placed people's
// cached sheets: the cache is keyed by the map's records, and a mod that
// changes a person's row changes the sheet the same records draw with.
func TestEditorModContextDropsThePlacedArt(t *testing.T) {
	x, s := contextEditor(t)
	prime := func() {
		t.Helper()
		m, err := s.model.Map()
		if err != nil {
			t.Fatal(err)
		}
		x.placedArt(s.model, m.Units, 1)
		if x.placedArts == nil {
			t.Fatal("the placed art was not cached")
		}
	}
	prime()
	n := openInW(t, s, copyFixture(t, "archer-girl"))
	if x.placedArts != nil {
		t.Fatal("the cache of the previous context's sheets stayed after the mod opened")
	}
	prime()
	x.setEditorMod(nil)
	if x.placedArts != nil {
		t.Fatal("the cache of the mod context's sheets stayed after the mod closed")
	}
	_ = n
}
