package ui

import (
	"testing"

	"againrom/pkg/words"
)

// TestSignatureRoleWordsResolveInEveryLanguage checks that each role of the one
// role-word table has a word in English and in Russian, and that an unlisted
// role has none.
func TestSignatureRoleWordsResolveInEveryLanguage(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	for _, lang := range []string{"", "russian"} {
		e.SetWords(words.For(lang))
		for role, id := range signatureRoleWords {
			got, ok := e.roleWord(role)
			if !ok || got == "" || got == id {
				t.Errorf("%q: role %q reads %q (ok %v), want its word for %s", lang, role, got, ok, id)
			}
		}
		if got, ok := e.roleWord("no-such-role"); ok || got != "" {
			t.Errorf("%q: an unlisted role reads %q (ok %v)", lang, got, ok)
		}
	}
}
