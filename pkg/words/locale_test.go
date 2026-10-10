package words

import (
	"testing"

	"againrom/pkg/locale"
)

// Language reads each install entry's locale code; an empty entry reads the
// reference language and an unknown entry names its table directly.
func TestLanguageReadsTheLocaleRecord(t *testing.T) {
	for _, l := range locale.All() {
		if got := Language(l.Entry); got != l.Code {
			t.Errorf("Language(%q) = %q, want %q", l.Entry, got, l.Code)
		}
	}
	if Language("") != English || English != locale.Fallback || Language("de") != "de" {
		t.Fatal("empty or unknown entry")
	}
}
