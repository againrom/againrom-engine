package mod

import (
	"reflect"
	"testing"
)

func TestResolvePagesOrderSourceAndLocale(t *testing.T) {
	install := PageLayer{Source: PageInstall, Lang: "en", Files: map[string][]byte{
		"event01.txt": []byte("install one"), "event02.txt": []byte("install two"), "event03.txt": []byte("install three"), "notes.txt": []byte("notes")}}
	modded := PageLayer{Source: PageMod, Lang: "en", Files: map[string][]byte{
		"event02.txt": []byte("mod two"), "event03.txt": []byte("mod other three"), "event04.txt": []byte("mod other four")},
		Other: map[string]bool{"event03.txt": true, "event04.txt": true}}
	mapped := PageLayer{Source: PageMap, Lang: "en", Files: map[string][]byte{
		"event02.txt": []byte("map two"), "event04.txt": []byte("map other four"), "event05.txt": []byte("map five")},
		Other: map[string]bool{"event04.txt": true}}
	keep := func(name string) bool { return name != "notes.txt" }
	got := ResolvePages(keep, install, modded, mapped)
	want := map[string]ResolvedPage{
		"event01.txt": {Bytes: []byte("install one"), Source: PageInstall, Locale: "en"},
		"event02.txt": {Bytes: []byte("map two"), Source: PageMap, Locale: "en"},
		"event03.txt": {Bytes: []byte("install three"), Source: PageInstall, Locale: "en"},
		"event04.txt": {Bytes: []byte("map other four"), Source: PageMap, Locale: "ru"},
		"event05.txt": {Bytes: []byte("map five"), Source: PageMap, Locale: "en"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolved %+v, want %+v", got, want)
	}
	if only := ResolvePages(nil, modded); len(only) != 3 || only["event03.txt"].Source != PageMod || only["event03.txt"].Locale != "ru" {
		t.Fatalf("one layer resolved %+v", only)
	}
}

func TestHeldPagesCarryTheEntryAndTheirLanguage(t *testing.T) {
	m := Mission{Mod: "m", Texts: "texts/ru", TextFiles: map[string][]byte{"event01.txt": []byte("one"), "event02.txt": []byte("two")},
		OtherLanguageTexts: map[string]bool{"event02.txt": true}}
	p := m.HeldPages()
	if p.ID != "m" || p.Entry.Texts != "texts/ru" || p.Err != nil {
		t.Fatalf("held pages %+v", p)
	}
	if p.Pages["event01.txt"].Locale != "ru" || p.Pages["event02.txt"].Locale != "en" || p.Pages["event02.txt"].Source != PageMod {
		t.Fatalf("held pages %+v", p.Pages)
	}
	if got := p.Files(); !reflect.DeepEqual(got, m.TextFiles) {
		t.Fatalf("files %q", got)
	}
}
