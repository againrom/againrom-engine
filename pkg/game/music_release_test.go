package game

import (
	"os"
	"sort"
	"strings"
	"testing"
)

// TestReleaseStaticMusicPopulation validates the complete 21-name archive and
// decodes every track sequentially through the production disk-indexed bank.
// check-release-tests runs it once per preserved EN/RU root.
func TestReleaseStaticMusicPopulation(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: static music needs a lawful install")
	}
	bank := OpenMusic(root)
	got := bank.Manifest()
	for i := range got {
		got[i] = strings.ToLower(got[i])
	}
	want := MusicTrackNames()
	for i := range want {
		want[i] = strings.ToLower(want[i])
	}
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("MUSIC.RES manifest has %d names, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("MUSIC.RES manifest = %v, want %v", got, want)
		}
	}

	for _, name := range MusicTrackNames() {
		track, ok := bank.Track(name)
		if !ok {
			t.Errorf("%s did not resolve and decode", name)
			continue
		}
		if track.Rate != 22050 || track.Frames() == 0 || len(track.StereoPCM)%4 != 0 {
			t.Errorf("%s = rate %d frames %d bytes %d", name, track.Rate, track.Frames(), len(track.StereoPCM))
		}
		track.StereoPCM = nil
	}
}
