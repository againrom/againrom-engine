package modrt

import (
	"testing"

	"againrom/pkg/locale"
)

// A base id reads the locale whose code ends it; any other base reads the
// reference language.
func TestLanguageForReadsTheLocaleRecord(t *testing.T) {
	for _, l := range locale.All() {
		if got := LanguageFor(l.BaseID); got != l.Code {
			t.Errorf("LanguageFor(%q) = %q, want %q", l.BaseID, got, l.Code)
		}
	}
	if LanguageFor("rom2-ru") != "ru" || LanguageFor("rom1-demo") != locale.Fallback || LanguageFor("rom1") != locale.Fallback {
		t.Fatal("second-game or unknown base")
	}
}
