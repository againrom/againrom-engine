package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShortcutSlotLayoutHasOneRow(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	banned := map[string][]string{
		"pkg/game/quickspells.go":        {"slots[i+1]", "shortcutStart", "i+4", "j+4"},
		"pkg/game/currentsession.go":     {"ShortcutSlots{1:", "slot <= 4", "shortcutStart", "slot++"},
		"pkg/game/originalfilesource.go": {"ShortcutSlots{1:"},
		"pkg/ui/missionbindings.go":      {"(slot + 8)", "ebiten.KeyF5"},
		"pkg/ui/spellbook.go":            {`"F" + strconv`},
	}
	for name, words := range banned {
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range words {
			if strings.Contains(string(b), w) {
				t.Errorf("%s names the shortcut slot layout by %q; it belongs to ui.ShortcutLayout", name, w)
			}
		}
	}
}
