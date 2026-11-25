package words

import (
	"bufio"
	"encoding/json"
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

// TestTablesEqualTheReplacedWords checks each historical EN/RU word against
// testdata/replaced.tsv and rejects English ids absent from that dump.
// editorPrefix starts every id of the map editor's own words.
const editorPrefix = "editor."

// recaptioned holds ids whose words were changed on purpose after the dump:
// the shortcut slots now serve books and inventory, not spells only.
var recaptioned = map[string]bool{"keys.action.spell": true}

// laterIDs are ids authored after the dump, like the editor words.
var laterIDs = map[string]bool{"keys.action.bind-item": true, "keys.action.bind-spell": true,
	"characters.title": true, "characters.ok": true, "characters.new": true, "characters.left": true,
	"mapchoice.title": true, "mapchoice.ok": true, "mapchoice.cancel": true,
	"mapchoice.level": true, "mapchoice.empty": true}

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
		if recaptioned[id] {
			// The caption was changed on purpose after the dump.
		} else if got := For(entry).Text(id); got != want {
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
	// Editor words and the shared smoothing caption are authored here.
	fresh := 0
	for _, id := range ids {
		if strings.HasPrefix(id, editorPrefix) || id == "options.smoothing" || laterIDs[id] {
			fresh++
			continue
		}
		if !seen[id] {
			t.Errorf("English id %s is a word the replaced code did not draw", id)
		}
	}
	if len(seen) != len(ids)-fresh || perLang["en"] != perLang["ru"] {
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
// pkg/game for word lookups by a literal id: Text, word and menuWord, and a
// literal id in a field word or ...Word, or in a list whose name ends in
// Words. Every id read is in the English table, and every English id is read.
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
	// Ids built as a literal prefix plus a key (`"editor.mod." + key`) are
	// read by prefix; the scan cannot see which keys reach them.
	var prefixes []string
	for _, dir := range []string{"../ui", "../game", "../mapload", "../mapedit", "../scriptprofile"} {
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
				var args []ast.Expr
				switch n := n.(type) {
				case *ast.KeyValueExpr:
					if key, ok := n.Key.(*ast.Ident); ok {
						switch {
						case key.Name == "word" || key.Name == "CopyRefused" || strings.HasSuffix(key.Name, "Word"):
							args = append(args, n.Value)
						case key.Name == "Hint" || key.Name == "newMapNote" || key.Name == "sceneryLabel" || key.Name == "sceneryDetail":
							args = append(args, n.Value)
						case key.Name == "Label" || key.Name == "save" || key.Name == "refusal" || key.Name == "Refusal" || key.Name == "scriptRefusal" || key.Name == "undecoded" || key.Name == "unstated":
							args = append(args, n.Value)
						case key.Name == "itemSuits":
							if list, ok := n.Value.(*ast.CompositeLit); ok {
								for _, suit := range list.Elts {
									if pair, ok := suit.(*ast.CompositeLit); ok && len(pair.Elts) == 2 {
										args = append(args, pair.Elts[1])
									}
								}
							}
						case strings.HasSuffix(key.Name, "Words"):
							if list, ok := n.Value.(*ast.CompositeLit); ok {
								args = append(args, list.Elts...)
							}
						}
					}
				case *ast.CallExpr:
					sel, ok := n.Fun.(*ast.SelectorExpr)
					if ok && sel.Sel.Name == "modGone" && len(n.Args) >= 1 {
						args = append(args, n.Args[0])
					}
					if ok && len(n.Args) == 1 && (sel.Sel.Name == "Text" || sel.Sel.Name == "word" || sel.Sel.Name == "plainWord" || sel.Sel.Name == "menuWord") {
						args = append(args, n.Args[0])
					}
					if fn, ok := n.Fun.(*ast.Ident); ok && (fn.Name == "wordChoices" || fn.Name == "num" || fn.Name == "choice") {
						args = append(args, n.Args...)
					}
					if fn, ok := n.Fun.(*ast.Ident); ok && fn.Name == "word" && len(n.Args) == 1 {
						args = append(args, n.Args[0])
					}
					// wordErr(id, figures...) holds the id the editor reads in
					// its language.
					if fn, ok := n.Fun.(*ast.Ident); ok && fn.Name == "wordErr" && len(n.Args) >= 1 {
						args = append(args, n.Args[0])
					}
					// wordCause(err, id, figures...) and wordRef(id) hold ids
					// the editor reads in its language.
					if fn, ok := n.Fun.(*ast.Ident); ok && fn.Name == "wordCause" && len(n.Args) >= 2 {
						args = append(args, n.Args[1])
					}
					// secondWord(condition, id) forms the second game's word id.
					if fn, ok := n.Fun.(*ast.Ident); ok && fn.Name == "secondWord" && len(n.Args) == 2 {
						if cond, ok := n.Args[0].(*ast.Ident); ok {
							if num, ok := n.Args[1].(*ast.BasicLit); ok {
								kind := "action"
								if cond.Name == "true" {
									kind = "check"
								}
								args = append(args, &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote("editor.second." + kind + "." + num.Value)})
							}
						}
					}
					if fn, ok := n.Fun.(*ast.Ident); ok && fn.Name == "worded" && len(n.Args) == 2 {
						args = append(args, n.Args[1])
					}
					if fn, ok := n.Fun.(*ast.Ident); ok && fn.Name == "wordRef" && len(n.Args) == 1 {
						args = append(args, n.Args[0])
					}
					// lifted(err, id, figures...) holds the id the editor reads
					// in its language.
					if fn, ok := n.Fun.(*ast.Ident); ok && fn.Name == "lifted" && len(n.Args) >= 2 {
						args = append(args, n.Args[1])
					}
				case *ast.AssignStmt:
					// A word id held in a local named key or word.
					if len(n.Lhs) == len(n.Rhs) {
						for i, l := range n.Lhs {
							if name, ok := l.(*ast.Ident); ok && (name.Name == "key" || name.Name == "word") {
								args = append(args, n.Rhs[i])
							}
							if sel, ok := l.(*ast.SelectorExpr); ok && sel.Sel.Name == "Label" {
								args = append(args, n.Rhs[i])
							}
						}
					}
				case *ast.ValueSpec:
					for i, v := range n.Values {
						list, ok := v.(*ast.CompositeLit)
						if ok && i < len(n.Names) && strings.HasSuffix(n.Names[i].Name, "Words") {
							// A table of words holds its ids as elements,
							// map values or the fields of an element.
							var gather func(elts []ast.Expr)
							gather = func(elts []ast.Expr) {
								for _, x := range elts {
									if kv, ok := x.(*ast.KeyValueExpr); ok {
										x = kv.Value
									}
									if inner, ok := x.(*ast.CompositeLit); ok {
										gather(inner.Elts)
										continue
									}
									args = append(args, x)
								}
							}
							gather(list.Elts)
						}
					}
				}
				for _, arg := range args {
					if sum, ok := arg.(*ast.BinaryExpr); ok && sum.Op == token.ADD {
						if head, ok := sum.X.(*ast.BasicLit); ok && head.Kind == token.STRING {
							if prefix, err := strconv.Unquote(head.Value); err == nil {
								prefixes = append(prefixes, prefix)
							}
						}
					}
					lit, ok := arg.(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					id, err := strconv.Unquote(lit.Value)
					if err != nil || !strings.Contains(id, ".") || strings.ContainsAny(id, " /\\") {
						continue
					}
					// A format id read with a figure (CommandWord) is read by its
					// literal prefix, like a prefix plus a key.
					if head, _, found := strings.Cut(id, "%d"); found {
						prefixes = append(prefixes, head)
						continue
					}
					used[id] = true
					if !table[id] {
						t.Errorf("%s reads %q, which the English table lacks", filepath.Base(name), id)
					}
				}
				return true
			})
		}
	}
	screens, err := filepath.Glob(filepath.Join("../ui/screens", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range screens {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		var doc any
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatal(err)
		}
		for _, id := range screenWordIDs(doc) {
			used[id] = true
			if !table[id] {
				t.Errorf("%s names %q, which the English table lacks", filepath.Base(name), id)
			}
		}
	}
	var unused []string
	for _, id := range ids {
		// itemGroupWord forms these ids from the decoder's row keys; the
		// pkg/game test TestEditorItemWordsAreExactlyTheKeysTheDecoderProduces
		// holds them to those keys.
		if strings.HasPrefix(id, "editor.item.kind.") || strings.HasPrefix(id, "editor.item.material.") || strings.HasPrefix(id, "editor.item.tier.") {
			continue
		}
		// The base menu action table holds these two as positional struct
		// fields, which the scan above does not read.
		if id == "mission.abandon" || id == "mission.abandon.confirm" {
			continue
		}
		// The character store's game rows hold the list's last row word as a
		// struct field.
		if id == "characters.new" {
			continue
		}
		prefixed := false
		for _, prefix := range prefixes {
			prefixed = prefixed || strings.HasPrefix(id, prefix)
		}
		if prefixed {
			continue
		}
		if !used[id] {
			unused = append(unused, id)
		}
	}
	sort.Strings(unused)
	if len(unused) != 0 {
		t.Errorf("English ids no production source reads: %v", unused)
	}
}

// screenWordIDs are the word ids an editor window description names: its
// title, help, labels, hints, tabs, button words and sample words.
func screenWordIDs(doc any) []string {
	var ids []string
	switch v := doc.(type) {
	case map[string]any:
		for k, x := range v {
			switch k {
			case "title", "titleHelp", "label", "hint", "help", "word":
				if id, ok := x.(string); ok && id != "" {
					ids = append(ids, id)
				}
				continue
			case "tabs", "words":
				var list []any
				if m, ok := x.(map[string]any); ok {
					for _, w := range m {
						list = append(list, w)
					}
				} else if l, ok := x.([]any); ok {
					list = l
				}
				for _, w := range list {
					if id, ok := w.(string); ok {
						ids = append(ids, id)
					}
				}
				continue
			}
			ids = append(ids, screenWordIDs(x)...)
		}
	case []any:
		for _, x := range v {
			ids = append(ids, screenWordIDs(x)...)
		}
	}
	return ids
}
