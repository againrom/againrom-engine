package mod

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/locale"
)

// A label table takes one label per locale code and refuses any other code;
// a missing label reads the reference language's, then the key. Text lookups
// fall back to the reference language.
func TestSettingLabelsAndLookupFollowTheLocaleTable(t *testing.T) {
	var pairs []string
	for _, l := range locale.All() {
		pairs = append(pairs, l.Code+` = "label `+l.Code+`"`)
	}
	set, err := ParseSettings([]byte("[skill]\nlabel = { " + strings.Join(pairs, ", ") + " }\ntype = \"int\"\ndefault = 1\nmin = 0\nmax = 2\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range locale.All() {
		if got := set[0].Label(l.Code); got != "label "+l.Code {
			t.Errorf("Label(%q) = %q", l.Code, got)
		}
	}
	if got := set[0].Label("de"); got != "label "+locale.Fallback {
		t.Errorf("unknown language label = %q", got)
	}
	_, err = ParseSettings([]byte("[skill]\nlabel = { de = \"x\" }\ntype = \"int\"\ndefault = 1\nmin = 0\nmax = 2\n"))
	if err == nil || !strings.Contains(err.Error(), `label has no language "de"`) {
		t.Fatalf("unknown code: %v", err)
	}
	plain, err := ParseSettings([]byte("[skill]\nlabel = \"plain\"\ntype = \"int\"\ndefault = 1\nmin = 0\nmax = 2\n"))
	if err != nil || plain[0].Label("ru") != "plain" {
		t.Fatalf("plain label: %v", err)
	}
	if !slices.Equal(languagesFor(""), []string{locale.Fallback}) || !slices.Equal(languagesFor("ru"), []string{"ru", locale.Fallback}) {
		t.Fatal("lookup languages")
	}
}
