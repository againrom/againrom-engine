// Package locale holds the install languages the engine knows, one record per
// language. Every surface that depends on the install's language reads its
// record here instead of naming a language itself: adding a language is a
// row of the table.
package locale

import "strings"

// Locale is one install language.
type Locale struct {
	// Entry is the language entry of the install's main archive.
	Entry string
	// Selector is the install's language selector, the last digit of the
	// language file.
	Selector int
	// Code names the language's text tables and mod labels.
	Code string
	// BaseID is the id mods name the first game's base of this language by.
	BaseID string
	// CodePage is the code page the install's text is written in; zero is
	// ASCII with no further characters the engine can name.
	CodePage int
	// FontRemap reports that the installed font moves a byte before it
	// selects a record (text.Convert).
	FontRemap bool
}

// Fallback is the code of the reference language: the engine's own words are
// written in it, and a lookup a language lacks reads it.
const Fallback = "en"

var table = [...]Locale{
	{Entry: "english", Selector: 0, Code: Fallback, BaseID: "rom1-en"},
	{Entry: "russian", Selector: 1, Code: "ru", BaseID: "rom1-ru", CodePage: 866, FontRemap: true},
}

// All is every known language, in table order.
func All() []Locale { return append([]Locale(nil), table[:]...) }

// ByEntry is the language of a main archive language entry.
func ByEntry(entry string) (Locale, bool) {
	for _, l := range table {
		if l.Entry == entry {
			return l, true
		}
	}
	return Locale{}, false
}

// BySelector is the language of an install language selector.
func BySelector(selector int) (Locale, bool) {
	for _, l := range table {
		if l.Selector == selector {
			return l, true
		}
	}
	return Locale{}, false
}

// ByCode is the language a table code names.
func ByCode(code string) (Locale, bool) {
	for _, l := range table {
		if l.Code == code {
			return l, true
		}
	}
	return Locale{}, false
}

// ByBaseSuffix is the language whose code ends a base id after a hyphen, as
// "rom1-ru" and "rom2-ru" end in "ru".
func ByBaseSuffix(base string) (Locale, bool) {
	for _, l := range table {
		if strings.HasSuffix(base, "-"+l.Code) {
			return l, true
		}
	}
	return Locale{}, false
}

// FontRemaps reports whether the installed font of selector moves a byte
// before it selects a record.
func FontRemaps(selector int) bool {
	l, ok := BySelector(selector)
	return ok && l.FontRemap
}

// CodePageOf is the code page of selector's text, zero for an unknown one.
func CodePageOf(selector int) int {
	l, _ := BySelector(selector)
	return l.CodePage
}
