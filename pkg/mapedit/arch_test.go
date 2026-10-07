package mapedit_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
	"testing"
)

// fileScope keys a reference that is not inside any function — a constant, a
// variable or a struct field declared at the top level of a file.
const fileScope = "<declarations>"

// nameTable maps "<file>/<function>" to the sorted, deduplicated set of watched
// names that declaration references. A set, not a list: a function that names
// the same thing twice is the same fact as naming it once, and a table that
// counted call sites would have to be rewritten by every refactor that did not
// change what the function reaches for.
type nameTable map[string][]string

// ---------------------------------------------------------------------------
// The declared tables
// ---------------------------------------------------------------------------

// almReferences is the whole of this package's traffic with pkg/formats/alm.
// The file is part of the key on purpose: the offsets and the entry points are
// deliberately spread one file per setter family, so a reference moving between
// files is a change worth declaring rather than a refactor to absorb silently.
var almReferences = nameTable{
	"editor.go/New":                    {"alm.OpenDocument"},
	"editor.go/(*Editor).Map":          {"alm.Map", "alm.Open"},
	"meta.go/(*Editor).SetName":        {"alm.EncodeName"},
	"meta.go/(*Editor).SetDescription": {"alm.EncodeDescription"},
}

// payloadLengthNames are the two identifiers by which a record's own payload
// length can be reached in this package: the frame's decoded field, and the
// offset of the length word in a record header.
//
// Both are watched because the unit count could be derived from either — one as
// span.payloadSize/70, the other by reading the header word at that offset — and
// a scan that watched only the first would leave the second route open.
var payloadLengthNames = map[string]bool{
	"payloadSize":    true,
	"recPayloadSize": true,
}

// payloadLengthReferences is every place either of those names may appear.
//
// What the table says by omission is the point: no function that produces or
// uses the unit count appears in it at all.
var payloadLengthReferences = nameTable{
	"editor.go/" + fileScope:         {"payloadSize", "recPayloadSize"},
	"editor.go/(*Editor).locate":     {"payloadSize", "recPayloadSize"},
	"unit.go/(*Editor).sizeWordPart": {"recPayloadSize"},
}

// ---------------------------------------------------------------------------
// The scan
// ---------------------------------------------------------------------------

// packageSources lists this package's non-test .go files. The working directory
// of a test binary is its own package directory, which is what makes the list
// the package's own rather than a path this file would have to know.
//
// An empty list is a failure, not a clean scan: a fence that has stopped finding
// sources reports no findings, and that must not read as a pass.
func packageSources(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("cannot read the package directory: %v", err)
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		out = append(out, name)
	}
	if len(out) == 0 {
		t.Fatal("the scan found no non-test source in the package directory: it would report every fence as clean")
	}
	slices.Sort(out)
	return out
}

// scanSources parses every non-test source and attributes each name watch
// reports to the declaration enclosing it: a func or method by its own name, and
// anything else to that file's declarations.
//
// Attribution is by walking f.Decls and inspecting each one whole, so a name in a
// signature belongs to that function as much as one in its body — the view
// method's alm.Map result type is a reference to alm and is counted as one.
func scanSources(t *testing.T, watch func(ast.Node) (string, bool)) nameTable {
	t.Helper()
	got := make(nameTable)
	for _, name := range packageSources(t) {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s does not parse: %v", name, err)
		}
		for _, decl := range f.Decls {
			key := name + "/" + fileScope
			if fn, ok := decl.(*ast.FuncDecl); ok {
				key = name + "/" + funcName(fn)
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				if hit, ok := watch(n); ok {
					got[key] = append(got[key], hit)
				}
				return true
			})
		}
	}
	for key, names := range got {
		slices.Sort(names)
		got[key] = slices.Compact(names)
	}
	return got
}

// funcName renders a declaration's name the way the tables spell it: New for a
// plain function, (*Editor).Map for a method.
func funcName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	return "(" + typeName(fn.Recv.List[0].Type) + ")." + fn.Name.Name
}

func typeName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return "*" + typeName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return "?"
}

// almQualified reports an alm.X qualified identifier. It matches a selector
// whose left side is the bare package name, so alm in a comment, in a string or
// as part of a longer identifier is not a finding.
func almQualified(n ast.Node) (string, bool) {
	sel, ok := n.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "alm" {
		return "", false
	}
	return "alm." + sel.Sel.Name, true
}

// payloadLength reports either watched identifier, wherever it stands: a field
// declaration, a composite-literal key, a selector's right side or a plain
// reference. All four are ways of naming the payload length, and the scan does
// not care which one a divergence would be written as.
func payloadLength(n ast.Node) (string, bool) {
	id, ok := n.(*ast.Ident)
	if !ok || !payloadLengthNames[id.Name] {
		return "", false
	}
	return id.Name, true
}

// ---------------------------------------------------------------------------
// The comparison
// ---------------------------------------------------------------------------

// mismatches compares a scan against a declared table for exact equality and
// returns one line per difference, in key order. Three kinds, and all three are
// failures: a declaration the table does not list, a table entry the scan did not
// find, and a declaration whose set differs from the declared one.
func mismatches(got, want nameTable) []string {
	var out []string
	for _, key := range unionKeys(got, want) {
		g, inGot := got[key]
		w, inWant := want[key]
		switch {
		case !inWant:
			out = append(out, key+" names "+list(g)+", and the table does not list it at all")
		case !inGot:
			out = append(out, key+" is in the table naming "+list(w)+", but the scan found no such declaration naming anything")
		case !slices.Equal(g, w):
			out = append(out, key+" names "+list(g)+", the table says "+list(w))
		}
	}
	return out
}

func unionKeys(a, b nameTable) []string {
	seen := make(map[string]bool, len(a)+len(b))
	keys := make([]string, 0, len(a)+len(b))
	for _, t := range []nameTable{a, b} {
		for k := range t {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	slices.Sort(keys)
	return keys
}

func list(names []string) string { return strings.Join(names, ", ") }

// without and withEntry build the deliberately wrong tables the negative cases
// use, leaving the declared ones untouched.
func without(base nameTable, key string) nameTable {
	out := make(nameTable, len(base))
	for k, v := range base {
		if k != key {
			out[k] = v
		}
	}
	return out
}

func withEntry(base nameTable, key string, names ...string) nameTable {
	out := make(nameTable, len(base)+1)
	for k, v := range base {
		out[k] = v
	}
	out[key] = names
	return out
}

// ---------------------------------------------------------------------------
// SC-9 — the fences
// ---------------------------------------------------------------------------

func TestNonTestSourceNamesOnlyItsDeclaredAlmIdentifiers(t *testing.T) {
	got := scanSources(t, almQualified)
	for _, m := range mismatches(got, almReferences) {
		t.Error(m)
	}

	for _, decoder := range []string{"alm.Open", "alm.OpenDocument"} {
		var where []string
		for key, names := range got {
			if slices.Contains(names, decoder) {
				where = append(where, key)
			}
		}
		slices.Sort(where)
		if len(where) != 1 {
			t.Errorf("%s is named in %d functions (%s); a decoded value must be obtainable in exactly one",
				decoder, len(where), list(where))
		}
	}
}

func TestTheUnitSettersNeverNameTheTypeSixPayloadLength(t *testing.T) {
	got := scanSources(t, payloadLength)
	for _, m := range mismatches(got, payloadLengthReferences) {
		t.Error(m)
	}
}

// TestTheScanReadsParsedSyntaxAndNotFileText is the fence on the fence. Both
// scans would be easy to write over file text and would then be wrong in a way
// that only prose reveals: doc.go names alm.OpenDocument to say where acceptance
// is taken and contains no code at all, and editor.go names it twice more in
// comments than it calls it.
func TestTheScanReadsParsedSyntaxAndNotFileText(t *testing.T) {
	got := scanSources(t, almQualified)

	parsed := 0
	for key, names := range got {
		if strings.HasPrefix(key, "doc.go/") {
			t.Errorf("the scan attributed %s to doc.go, which declares no code", list(names))
		}
		parsed += strings.Count(list(names), "alm.OpenDocument")
	}

	text := 0
	for _, name := range packageSources(t) {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("cannot read %s: %v", name, err)
		}
		text += strings.Count(string(b), "alm.OpenDocument")
	}

	if parsed != 1 {
		t.Errorf("the scan found alm.OpenDocument %d times, want the one call", parsed)
	}
	if text <= parsed {
		t.Errorf("the sources name alm.OpenDocument %d times as text and %d as syntax; with no prose mention left, this case witnesses nothing",
			text, parsed)
	}
}

// ---------------------------------------------------------------------------
// The negative cases — a wrong table must fail
// ---------------------------------------------------------------------------

// wrongTableCases builds the ways a table can be wrong, given a correct one and a
// key it holds: too permissive, too strict, missing, invented, and relocated. The
// empty table is the case that separates equality from an absence search — a scan
// that only looked for a forbidden pattern would pass with nothing declared.
func wrongTableCases(base nameTable, key string, extra string) []struct {
	name  string
	table nameTable
} {
	return []struct {
		name  string
		table nameTable
	}{
		{"an entry granted one name too many", withEntry(base, key, append(slices.Clone(base[key]), extra)...)},
		{"an entry stripped of a name it has", withEntry(base, key)},
		{"an entry dropped altogether", without(base, key)},
		{"an entry for a function that names nothing", withEntry(base, "grid.go/(*Editor).SetTile", extra)},
		{"the same entry moved to another file", withEntry(without(base, key), "grid.go/"+strings.SplitN(key, "/", 2)[1], base[key]...)},
		{"nothing declared at all", nameTable{}},
	}
}

func TestTheAlmFenceFailsUnderADeliberatelyWrongTable(t *testing.T) {
	got := scanSources(t, almQualified)
	for _, c := range wrongTableCases(almReferences, "editor.go/New", "alm.Open") {
		t.Run(c.name, func(t *testing.T) {
			if m := mismatches(got, c.table); len(m) == 0 {
				t.Error("the wrong table was accepted, so the fence witnesses nothing")
			}
		})
	}
}

func TestThePayloadLengthFenceFailsUnderADeliberatelyWrongTable(t *testing.T) {
	got := scanSources(t, payloadLength)
	cases := wrongTableCases(payloadLengthReferences, "unit.go/(*Editor).sizeWordPart", "payloadSize")

	// And the case the fence exists for: a table that would let the count be
	// derived from the payload length. The source does not do it, so the entry
	// has nothing to match and the comparison must say so.
	cases = append(cases, struct {
		name  string
		table nameTable
	}{
		"the count granted the payload length",
		withEntry(payloadLengthReferences, "unit.go/(*Editor).unitCount", "payloadSize"),
	})

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if m := mismatches(got, c.table); len(m) == 0 {
				t.Error("the wrong table was accepted, so the fence witnesses nothing")
			}
		})
	}
}
