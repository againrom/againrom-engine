// Package words holds the engine's own words: one strings table per language,
// keyed by a stable message id, in the mod text layout text/<lang>/strings.toml.
// The tables are read through the mods' own lookup (mod.Lookup), so an
// engine word and a mod word are one mechanism: the language's own text, else
// the English one.
//
// The language is the install's: Language turns the base profile's language
// entry into a table language. Adding a language is a table file; no screen
// names a language.
package words

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync"

	"againrom/pkg/locale"
	"againrom/pkg/mod"
)

// English is the table language every lookup falls back to, and the
// reference set of message ids.
const English = locale.Fallback

//go:embed text
var engineTables embed.FS

// Language is the table language of an install language entry, the base
// profile's Language: a known entry reads its locale code, an empty entry
// reads English, and any other entry names its table directly.
func Language(entry string) string {
	if entry == "" {
		return English
	}
	if l, ok := locale.ByEntry(entry); ok {
		return l.Code
	}
	return entry
}

// Book is the words of one language. The zero Book is the engine's English,
// and For returns it for English, so two English word sets compare equal.
type Book struct{ b *book }

type book struct {
	lang string
	text func(id string) (string, bool)
}

// Load reads the words of lang from the text/<lang>/strings.toml tables of
// fsys, English standing in for an id lang's table lacks.
func Load(fsys fs.FS, lang string) (Book, error) {
	text, err := mod.Lookup(lang, func(l string) (mod.Strings, error) { return readTable(fsys, l) })
	if err != nil {
		return Book{}, err
	}
	return Book{&book{lang: lang, text: text}}, nil
}

func readTable(fsys fs.FS, lang string) (mod.Strings, error) {
	file := mod.StringsFile(lang)
	data, err := fs.ReadFile(fsys, file)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return mod.ParseStrings(file, data)
}

var engineBooks sync.Map // table language -> Book

// For is the engine's words for the install language entry, loaded once per
// language. A language the engine ships no table for reads English. A
// malformed shipped table is a build defect the package tests reject; it
// panics here rather than drawing ids.
func For(entry string) Book {
	lang := Language(entry)
	if lang == English {
		return Book{}
	}
	if _, err := fs.Stat(engineTables, mod.StringsFile(lang)); err != nil {
		return Book{}
	}
	return engineBook(lang)
}

func engineBook(lang string) Book {
	if b, ok := engineBooks.Load(lang); ok {
		return b.(Book)
	}
	b, err := Load(engineTables, lang)
	if err != nil {
		panic(fmt.Sprintf("engine words %s: %v", lang, err))
	}
	actual, _ := engineBooks.LoadOrStore(lang, b)
	return actual.(Book)
}

// Lang is the book's table language.
func (b Book) Lang() string {
	if b.b == nil {
		return English
	}
	return b.b.lang
}

// Text is the word of id. An id no table holds reads as the id itself, so a
// missing word shows where it is drawn; TestEveryUsedIDIsInTheEnglishTable
// keeps the engine's ids in the English table.
func (b Book) Text(id string) string {
	if b.b == nil {
		b = engineBook(English)
	}
	if s, ok := b.b.text(id); ok {
		return s
	}
	return id
}

// IDs are the message ids of the English table of fsys, sorted.
func IDs(fsys fs.FS) ([]string, error) {
	t, err := readTable(fsys, English)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(t))
	for id := range t {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

// Lacking lists the English ids the lang table of fsys does not hold, sorted:
// the words that language shows in English.
func Lacking(fsys fs.FS, lang string) ([]string, error) {
	ids, err := IDs(fsys)
	if err != nil {
		return nil, err
	}
	t, err := readTable(fsys, lang)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, id := range ids {
		if _, ok := t[id]; !ok {
			out = append(out, id)
		}
	}
	return out, nil
}

// Engine is the file system of the engine's own tables, for the reports above.
func Engine() fs.FS { return engineTables }

// Languages are the table languages the engine ships, sorted.
func Languages() []string {
	entries, err := fs.ReadDir(engineTables, "text")
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			out = append(out, e.Name())
		}
	}
	return out
}
