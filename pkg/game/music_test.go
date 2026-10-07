package game_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/game"
)

func writeMusicArchive(t *testing.T, root string, files []synth.File) {
	t.Helper()
	dir := filepath.Join(root, "Allods")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, game.MusicArchive), synth.Archive(files), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestMusicBankOpensLazilyAndReadsOneTrack(t *testing.T) {
	root := t.TempDir()
	bank := game.OpenMusic(root)
	// The archive appears after construction. A constructor-time open would
	// have permanently recorded its earlier absence.
	writeMusicArchive(t, root, []synth.File{{Path: "menu.wav", Data: tinyWAV()}})
	track, ok := bank.Track("MENU.WAV")
	if !ok || track.Rate <= 0 || track.Frames() == 0 {
		t.Fatalf("Track(menu) = rate %d frames %d, %v", track.Rate, track.Frames(), ok)
	}
	if _, ok := bank.Track("inn_ssi_extra.wav"); ok {
		t.Fatal("a name outside the fixed manifest resolved")
	}
}

func TestMusicBankManifestAndFailureAreTotal(t *testing.T) {
	root := t.TempDir()
	want := []string{"b00.wav", "menu.wav", "town.wav"}
	writeMusicArchive(t, root, []synth.File{
		{Path: "B00.wav", Data: tinyWAV()},
		{Path: "menu.wav", Data: tinyWAV()},
		{Path: "town.wav", Data: []byte("not wave")},
	})
	bank := game.OpenMusic(root)
	got := bank.Manifest()
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("manifest = %v, want %v", got, want)
	}
	if _, ok := bank.Track("town.wav"); ok {
		t.Fatal("malformed track decoded")
	}
	if _, ok := bank.Track("shop.wav"); ok {
		t.Fatal("missing track decoded")
	}

	missing := game.OpenMusic(t.TempDir())
	if got := missing.Manifest(); got != nil {
		t.Fatalf("missing manifest = %v, want nil", got)
	}
	if _, ok := missing.Track("menu.wav"); ok {
		t.Fatal("missing archive decoded a track")
	}
	var nilBank *game.MusicBank
	if _, ok := nilBank.Track("menu.wav"); ok || nilBank.Manifest() != nil {
		t.Fatal("nil bank was not silent")
	}
}

func TestMusicTrackManifestIsTheExactTwentyOneNamePopulation(t *testing.T) {
	got := game.MusicTrackNames()
	if len(got) != 21 {
		t.Fatalf("manifest has %d names, want 21: %v", len(got), got)
	}
	seen := make(map[string]bool)
	for _, name := range got {
		folded := strings.ToLower(name)
		if seen[folded] {
			t.Fatalf("duplicate %q in %v", name, got)
		}
		seen[folded] = true
	}
	if !seen["inn_ssi.wav"] {
		t.Fatal("archive-only inn_ssi.wav missing from manifest")
	}
}
