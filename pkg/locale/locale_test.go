package locale

import "testing"

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
	if en != (Locale{Entry: "english", Selector: 0, Code: "en", BaseID: "rom1-en"}) {
		t.Fatalf("english = %+v", en)
	}
	if ru != (Locale{Entry: "russian", Selector: 1, Code: "ru", BaseID: "rom1-ru", CodePage: 866, FontRemap: true}) {
		t.Fatalf("russian = %+v", ru)
	}
}
