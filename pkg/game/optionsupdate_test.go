package game

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func optionsSetters() map[string]func(OptionsStore) error {
	return map[string]func(OptionsStore) error{
		"TextSmoothing":   func(s OptionsStore) error { return s.SetTextSmoothing(true) },
		"TipsMode":        func(s OptionsStore) error { return s.SetTipsMode(false) },
		"GameSpeed":       func(s OptionsStore) error { return s.SetGameSpeed(5) },
		"WimpyMode":       func(s OptionsStore) error { return s.SetWimpyMode(1) },
		"FrameSmoothing":  func(s OptionsStore) error { return s.SetFrameSmoothing(true) },
		"GameOption":      func(s OptionsStore) error { return s.setGameOption(ui.GameOptionAutoCast, 1) },
		"TooltipDelay":    func(s OptionsStore) error { return s.SetTooltipDelay(0) },
		"EncounterMovies": func(s OptionsStore) error { return s.EncounterCutscene("intro") },
	}
}

// A file that exists but cannot be read is left as it is and each setter
// returns the read error; a missing file starts empty.
func TestOptionsStoreSettersRefuseUnreadableFileAndCreateMissingOne(t *testing.T) {
	failing := errors.New("injected read failure")
	for name, set := range optionsSetters() {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "options.txt")
			const before = "GameSpeed=3\nTipsMode=1\n"
			if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
				t.Fatal(err)
			}
			store := OptionsStore{Path: path}
			prev := optionsReadFile
			optionsReadFile = func(string) ([]byte, error) { return nil, failing }
			err := set(store)
			optionsReadFile = prev
			if !errors.Is(err, failing) {
				t.Fatalf("unreadable file: setter returned %v, want the read error", err)
			}
			got, rerr := os.ReadFile(path)
			if rerr != nil || string(got) != before {
				t.Fatalf("unreadable file changed: %q, %v", got, rerr)
			}

			missing := OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
			if err := set(missing); err != nil {
				t.Fatalf("missing file: %v", err)
			}
			b, err := os.ReadFile(missing.Path)
			if err != nil {
				t.Fatal(err)
			}
			if n := strings.Count(string(b), "\n"); n != 1 {
				t.Fatalf("missing file: want one key, got %q", b)
			}
		})
	}
}
