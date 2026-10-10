package words

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
)

// sameWord maps a site id of the replaced code onto the message id that site
// reads now: the Game Options page draws the game menu's own slower, faster
// and unit words, which the replaced code also held once.
var sameWord = map[string]string{
	"options.slower":       "speed.slower",
	"options.faster":       "speed.faster",
	"tooltip.unit_options": "tooltip.unit",
}

// TestTablesEqualTheReplacedWords is the no-word-change proof: every word the
// replaced per-screen code produced (testdata/replaced.tsv, drawn through that
// code on both languages) is the table's word for that id, and the English
// table holds no id that code did not produce.
func TestTablesEqualTheReplacedWords(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "replaced.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	seen := map[string]bool{}
	perLang := map[string]int{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cols := strings.SplitN(line, "\t", 3)
		if len(cols) != 3 {
			t.Fatalf("malformed row %q", line)
		}
		want, err := strconv.Unquote(cols[2])
		if err != nil {
			t.Fatalf("row %q: %v", line, err)
		}
		id := cols[1]
		if to, ok := sameWord[id]; ok {
			id = to
		}
		entry := map[string]string{"en": "english", "ru": "russian"}[cols[0]]
		if got := For(entry).Text(id); got != want {
			t.Errorf("%s %s = %q, the replaced code drew %q", cols[0], id, got, want)
		}
		seen[id] = true
		perLang[cols[0]]++
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	ids, err := IDs(Engine())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		if !seen[id] {
			t.Errorf("English id %s is a word the replaced code did not draw", id)
		}
	}
	if len(seen) != len(ids) || perLang["en"] != perLang["ru"] {
		t.Errorf("ids: dump %d, table %d; rows en %d, ru %d", len(seen), len(ids), perLang["en"], perLang["ru"])
	}
	t.Logf("%d message ids; %d site rows per language", len(ids), perLang["en"])
}

// TestShippedTablesLackNoID reports, per shipped table, the English ids it
// lacks; those draw in English. English and Russian lack none.
func TestShippedTablesLackNoID(t *testing.T) {
	langs := Languages()
	if !slices.Equal(langs, []string{"en", "ru"}) {
		t.Fatalf("shipped tables %v, want [en ru]", langs)
	}
	for _, lang := range langs {
		lacking, err := Lacking(Engine(), lang)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s lacks %d ids: %v", lang, len(lacking), lacking)
		if len(lacking) != 0 {
			t.Errorf("table %s lacks %v", lang, lacking)
		}
	}
}

func TestAThirdLanguageIsATableAndFallsBackToEnglish(t *testing.T) {
	fsys := fstest.MapFS{
		"text/en/strings.toml": {Data: []byte("\"a\" = \"A\"\n\"b\" = \"B\"\n")},
		"text/xx/strings.toml": {Data: []byte("# synthetic\n\"a\" = \"xa\"\n")},
	}
	b, err := Load(fsys, "xx")
	if err != nil {
		t.Fatal(err)
	}
	if b.Lang() != "xx" || b.Text("a") != "xa" || b.Text("b") != "B" || b.Text("c") != "c" {
		t.Fatalf("xx book: lang %q a=%q b=%q c=%q", b.Lang(), b.Text("a"), b.Text("b"), b.Text("c"))
	}
	lacking, err := Lacking(fsys, "xx")
	if err != nil || !slices.Equal(lacking, []string{"b"}) {
		t.Fatalf("xx lacks %v (%v), want [b]", lacking, err)
	}
	if _, err := Load(fstest.MapFS{"text/xx/strings.toml": {Data: []byte("[t]\n")}}, "xx"); err == nil {
		t.Fatal("a table holding a table header loaded")
	}
}

func TestLanguageReadsTheInstallEntry(t *testing.T) {
	for entry, want := range map[string]string{"": "en", "english": "en", "russian": "ru", "xx": "xx"} {
		if got := Language(entry); got != want {
			t.Errorf("Language(%q) = %q, want %q", entry, got, want)
		}
	}
	if (Book{}).Lang() != English || (Book{}).Text("save.title") != For("english").Text("save.title") {
		t.Fatal("the zero Book is not English")
	}
	if For("language selector 2").Text("save.title") != For("").Text("save.title") {
		t.Fatal("a language without a table did not read English")
	}
}

// TestEveryUsedIDIsInTheEnglishTable reads the production sources of pkg/ui and
// pkg/game for word lookups by a literal id: Text, word and menuWord. Every id
// read is in the English table, and every English id is read.
func TestEveryUsedIDIsInTheEnglishTable(t *testing.T) {
	ids, err := IDs(Engine())
	if err != nil {
		t.Fatal(err)
	}
	table := map[string]bool{}
	for _, id := range ids {
		table[id] = true
	}
	used := map[string]bool{}
	for _, dir := range []string{"../ui", "../game"} {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range files {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) != 1 {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Text" && sel.Sel.Name != "word" && sel.Sel.Name != "menuWord" {
					return true
				}
				lit, ok := call.Args[0].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				id, err := strconv.Unquote(lit.Value)
				if err != nil || !strings.Contains(id, ".") || strings.ContainsAny(id, " /\\") {
					return true
				}
				used[id] = true
				if !table[id] {
					t.Errorf("%s reads %q, which the English table lacks", filepath.Base(name), id)
				}
				return true
			})
		}
	}
	var unused []string
	for _, id := range ids {
		if !used[id] {
			unused = append(unused, id)
		}
	}
	sort.Strings(unused)
	if len(unused) != 0 {
		t.Errorf("English ids no production source reads: %v", unused)
	}
}
