package game

import (
	"strings"
	"testing"

	"againrom/pkg/mod"
)

// A mod whose records did not load refuses an edit on every page the one page
// step opens, in the read-only line, and writes nothing.
func TestModPageStepRefusesEveryPageWhenContextIsBroken(t *testing.T) {
	x, s, _ := charactersPage(t, "en")
	src := "# knobs\n[speed]  # how fast\nlabel = \"Speed\"\ntype = \"int\"\ndefault = 3\nmin = 1\nmax = 9\n"
	seedProject(t, x, s, map[string]string{mod.SettingsName: src})
	x.mod.contextIssue = "records broken"
	refusal := x.word("editor.mod.readonly")
	calls := map[string]func() error{
		"settings":   func() error { _, err := s.setSettingsField(settingsAddKey, ""); return err },
		"speakers":   func() error { _, err := s.setSpeakersField(speakersAddKey, ""); return err },
		"bodies":     func() error { _, err := s.setBodiesField(bodiesAddKey, "x"); return err },
		"screens":    func() error { _, err := s.setScreensField(screensAddKey, ""); return err },
		"companions": func() error { _, err := s.setCompanionsField(companionsAddKey, ""); return err },
	}
	for name, call := range calls {
		if err := call(); err == nil || !strings.Contains(err.Error(), refusal) {
			t.Errorf("%s: an edit under a broken context gave %v, want the read-only line", name, err)
		}
	}
	if got := string(s.workspace.Project.Files[mod.SettingsName]); got != src {
		t.Fatalf("a refused edit wrote the settings file: %q", got)
	}
}
