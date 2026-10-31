package locale

import (
	"strings"
	"testing"
)

// Each lookup reaches the one record of its language, and an unknown key
// reaches none.
func TestLookupsReachOneRecordPerLanguage(t *testing.T) {
	for _, l := range All() {
		for name, got := range map[string]func() (Locale, bool){
			"entry":    func() (Locale, bool) { return ByEntry(l.Entry) },
			"selector": func() (Locale, bool) { return BySelector(l.Selector) },
			"code":     func() (Locale, bool) { return ByCode(l.Code) },
			"base":     func() (Locale, bool) { return ByBaseSuffix(l.BaseID) },
		} {
			if r, ok := got(); !ok || r != l {
				t.Errorf("%s of %q = %+v %t", name, l.Code, r, ok)
			}
		}
		if FontRemaps(l.Selector) != l.FontRemap || CodePageOf(l.Selector) != l.CodePage {
			t.Errorf("%q selector answers differ from its record", l.Code)
		}
	}
	if _, ok := ByEntry("klingon"); ok || FontRemaps(7) || CodePageOf(7) != 0 {
		t.Fatal("an unknown language answered a record")
	}
	if l, ok := ByBaseSuffix("rom2-ru"); !ok || l.Code != "ru" {
		t.Fatal("a second-game base id did not reach its language")
	}
	if _, ok := ByCode(Fallback); !ok {
		t.Fatal("the reference language has no record")
	}
}

// The shipped records: English is selector 0 with no remap, Russian is
// selector 1, code page 866, with the font remap.
func TestShippedRecords(t *testing.T) {
	en, _ := ByEntry("english")
	ru, _ := ByEntry("russian")
	if en != (Locale{Entry: "english", Selector: 0, Code: "en", BaseID: "rom1-en", Counterpart: "ru", SecondSaveNameLimit: 123}) {
		t.Fatalf("english = %+v", en)
	}
	if ru != (Locale{Entry: "russian", Selector: 1, Code: "ru", BaseID: "rom1-ru", CodePage: 866, WindowsCodePage: 1251, FontRemap: true, Counterpart: "en", SecondSaveNameLimit: 127}) {
		t.Fatalf("russian = %+v", ru)
	}
	if WindowsCyrillic != 1251 || DOSCyrillic != 866 {
		t.Fatalf("cyrillic code pages = %d, %d", WindowsCyrillic, DOSCyrillic)
	}
}

func TestKnownIsTheLanguageSet(t *testing.T) {
	for _, l := range All() {
		if !Known(l.Code) {
			t.Errorf("Known(%q) = false for a table row", l.Code)
		}
		if Known(l.Entry) || Known(l.BaseID) || Known(strings.ToUpper(l.Code)) || Known(l.Code+" ") {
			t.Errorf("Known accepted a spelling of %q that is not its code", l.Code)
		}
	}
	for _, code := range []string{"", "de", "EN", "en-US", "rom1-ru", "english", "/"} {
		if Known(code) {
			t.Errorf("Known(%q) = true", code)
		}
	}
}

func TestFallbackIsARowAndOfEntryFallsBackToIt(t *testing.T) {
	ref, ok := ByCode(Fallback)
	if !ok || !Known(Fallback) {
		t.Fatal("the fallback has no record")
	}
	for _, l := range All() {
		if OfEntry(l.Entry) != l {
			t.Errorf("OfEntry(%q) is not its row", l.Entry)
		}
	}
	for _, entry := range []string{"", "klingon", "English", "en"} {
		if OfEntry(entry) != ref {
			t.Errorf("OfEntry(%q) is not the fallback row", entry)
		}
	}
}

func TestSecondSaveNameLimitPerRow(t *testing.T) {
	want := map[string]int{"english": 123, "russian": 127}
	for _, l := range All() {
		w, ok := want[l.Entry]
		if !ok {
			t.Errorf("row %q has no expected limit; state it", l.Entry)
			continue
		}
		if l.SecondSaveNameLimit != w {
			t.Errorf("%q limit = %d, want %d", l.Entry, l.SecondSaveNameLimit, w)
		}
		if OfEntry(l.Entry).SecondSaveNameLimit != w {
			t.Errorf("%q limit through OfEntry differs", l.Entry)
		}
	}
	if OfEntry("").SecondSaveNameLimit != 123 || OfEntry("klingon").SecondSaveNameLimit != 123 {
		t.Error("an unknown entry did not read the English limit")
	}
}

// Other answers what the callers' old literal pair answered.
func TestOtherAnswersTheLanguagePair(t *testing.T) {
	for code, want := range map[string]string{
		"en":      "ru",
		"ru":      "en",
		"":        "",
		"EN":      "",
		"de":      "",
		"en-US":   "",
		" ru":     "",
		"rom1-ru": "",
	} {
		if got := Other(code); got != want {
			t.Errorf("Other(%q) = %q, want %q", code, got, want)
		}
	}
	for _, l := range All() {
		if back := Other(Other(l.Code)); back != l.Code {
			t.Errorf("Other(Other(%q)) = %q", l.Code, back)
		}
	}
}
