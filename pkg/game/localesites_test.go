package game

import (
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/locale"
)

// Every game-side language branch answers from the locale record: the
// archive's language entry, the base id, and the preserved sibling installs.
func TestGameLanguageSitesReadTheLocaleRecord(t *testing.T) {
	var codes []string
	for _, l := range locale.All() {
		if got := languageEntry(l.Selector); got != l.Entry {
			t.Errorf("languageEntry(%d) = %q, want %q", l.Selector, got, l.Entry)
		}
		if got := BaseID(InstallInfo{Language: l.Entry}); got != l.BaseID {
			t.Errorf("BaseID(%q) = %q, want %q", l.Entry, got, l.BaseID)
		}
		codes = append(codes, l.Code)
	}
	if got := languageEntry(7); got != "language selector 7" {
		t.Errorf("unknown selector entry = %q", got)
	}
	if got := BaseID(InstallInfo{Language: "klingon"}); got != "rom1" {
		t.Errorf("unknown language base = %q", got)
	}
	root := filepath.Join(t.TempDir(), "gameversions", "en")
	var want []string
	for _, c := range codes {
		want = append(want, filepath.Join(filepath.Dir(root), c))
	}
	if got := preservedPairRoots(root); !slices.Equal(got, want) {
		t.Errorf("preserved pair = %v, want %v", got, want)
	}
	if got := preservedPairRoots(filepath.Join(t.TempDir(), "other", "en")); got != nil {
		t.Errorf("a root outside gameversions fenced %v", got)
	}
}
