package archtest

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestLiveTreeClean loads the real module tree and asserts the production import
// graph obeys the DAG and that pkg/sim's tests import only the standard library.
func TestLiveTreeClean(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	prod, simTests, err := Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(prod) == 0 {
		t.Fatal("loader found no packages; the walk or module root is wrong")
	}
	if vs := Check(prod); len(vs) != 0 {
		for _, v := range vs {
			t.Errorf("unexpected DAG violation: %s", v)
		}
	}
	if vs := CheckSimTests(simTests); len(vs) != 0 {
		for _, v := range vs {
			t.Errorf("unexpected sim-test import: %s", v)
		}
	}
}

// TestCheckNamesOffendingEdge drives the pure evaluator with synthetic import
// graphs — the negative path AC-2 requires ("fails, naming the offending edge")
// is proven here by a committed test rather than a transient manual injection.
func TestCheckNamesOffendingEdge(t *testing.T) {
	cases := []struct {
		name       string
		imports    map[string][]string
		wantEdge   string // "From -> Import" expected in a violation; "" means expect none
		wantReason string // substring expected in that violation's message
	}{
		{
			name:       "sim importing a formats package (determinism wall)",
			imports:    map[string][]string{"pkg/sim": {"againrom/pkg/formats/res"}},
			wantEdge:   "pkg/sim -> pkg/formats/res",
			wantReason: "dependency DAG",
		},
		{
			name:       "vfs importing data (against DAG direction)",
			imports:    map[string][]string{"pkg/vfs": {"againrom/pkg/data"}},
			wantEdge:   "pkg/vfs -> pkg/data",
			wantReason: "dependency DAG",
		},
		{
			name:       "unregistered package is fail-closed",
			imports:    map[string][]string{"pkg/newthing": {}},
			wantEdge:   "pkg/newthing",
			wantReason: "fail-closed",
		},
		{
			name: "debug face is registered with its CPU font inputs",
			imports: map[string][]string{"pkg/render/debugtext": {
				"image", "againrom/pkg/render/text",
				"github.com/hajimehoshi/bitmapfont/v4",
				"golang.org/x/image/font", "golang.org/x/image/math/fixed",
			}},
			wantEdge: "",
		},
		{
			name:       "debug face cannot reach simulation",
			imports:    map[string][]string{"pkg/render/debugtext": {"againrom/pkg/sim"}},
			wantEdge:   "pkg/render/debugtext -> pkg/sim",
			wantReason: "dependency DAG",
		},
		{
			name:       "debug face cannot reach another rendering leaf",
			imports:    map[string][]string{"pkg/render/debugtext": {"againrom/pkg/render/refraction"}},
			wantEdge:   "pkg/render/debugtext -> pkg/render/refraction",
			wantReason: "dependency DAG",
		},
		{
			name:       "debug face cannot use Ebitengine",
			imports:    map[string][]string{"pkg/render/debugtext": {"github.com/hajimehoshi/ebiten/v2"}},
			wantEdge:   "pkg/render/debugtext -> github.com/hajimehoshi/ebiten/v2",
			wantReason: "external import not permitted",
		},
		{
			name:       "debug face cannot use unrelated image packages",
			imports:    map[string][]string{"pkg/render/debugtext": {"golang.org/x/image/draw"}},
			wantEdge:   "pkg/render/debugtext -> golang.org/x/image/draw",
			wantReason: "external import not permitted",
		},
		{
			name:       "debug face cannot use bitmapfont child packages",
			imports:    map[string][]string{"pkg/render/debugtext": {"github.com/hajimehoshi/bitmapfont/v4/other"}},
			wantEdge:   "pkg/render/debugtext -> github.com/hajimehoshi/bitmapfont/v4/other",
			wantReason: "external import not permitted",
		},
		{
			name:       "debug face external grants do not reach terrain",
			imports:    map[string][]string{"pkg/render/terrain": {"github.com/hajimehoshi/bitmapfont/v4"}},
			wantEdge:   "pkg/render/terrain -> github.com/hajimehoshi/bitmapfont/v4",
			wantReason: "external import not permitted",
		},
		{
			name:     "UI may call the debug face",
			imports:  map[string][]string{"pkg/ui": {"againrom/pkg/render/debugtext"}},
			wantEdge: "",
		},
		{
			name: "the launcher may draw with the bundled font and Ebitengine",
			imports: map[string][]string{"cmd/starter": {
				"github.com/hajimehoshi/bitmapfont/v4", "golang.org/x/image/font", "golang.org/x/image/math/fixed",
				"github.com/hajimehoshi/ebiten/v2", "againrom/pkg/game", "againrom/pkg/ini", "againrom/pkg/mod",
			}},
			wantEdge: "",
		},
		{
			name:       "another command may not name the font package",
			imports:    map[string][]string{"cmd/restool": {"github.com/hajimehoshi/bitmapfont/v4"}},
			wantEdge:   "cmd/restool -> github.com/hajimehoshi/bitmapfont/v4",
			wantReason: "external import not permitted",
		},
		{
			name:       "the launcher cannot reach the simulation",
			imports:    map[string][]string{"cmd/starter": {"againrom/pkg/sim"}},
			wantEdge:   "cmd/starter -> pkg/sim",
			wantReason: "dependency DAG",
		},
		{
			name:       "the simulation cannot reach the mod reader",
			imports:    map[string][]string{"pkg/sim": {"againrom/pkg/mod"}},
			wantEdge:   "pkg/sim -> pkg/mod",
			wantReason: "dependency DAG",
		},
		{
			name:     "the mod runtime may import the interpreter and the mod and rules packages",
			imports:  map[string][]string{"pkg/modrt": {"go.starlark.net/starlark", "go.starlark.net/syntax", "againrom/pkg/mod", "againrom/pkg/rules"}},
			wantEdge: "",
		},
		{
			name:       "the simulation cannot import the interpreter",
			imports:    map[string][]string{"pkg/sim": {"go.starlark.net/starlark"}},
			wantEdge:   "pkg/sim -> go.starlark.net/starlark",
			wantReason: "external import not permitted",
		},
		{
			name:       "the mod reader cannot import the interpreter",
			imports:    map[string][]string{"pkg/mod": {"go.starlark.net/starlark"}},
			wantEdge:   "pkg/mod -> go.starlark.net/starlark",
			wantReason: "external import not permitted",
		},
		{
			name:       "the simulation cannot reach the mod runtime",
			imports:    map[string][]string{"pkg/sim": {"againrom/pkg/modrt"}},
			wantEdge:   "pkg/sim -> pkg/modrt",
			wantReason: "dependency DAG",
		},
		{
			name:       "the data package cannot reach the mod runtime",
			imports:    map[string][]string{"pkg/data": {"againrom/pkg/modrt"}},
			wantEdge:   "pkg/data -> pkg/modrt",
			wantReason: "dependency DAG",
		},
		{
			name:       "the mod reader is a leaf",
			imports:    map[string][]string{"pkg/mod": {"againrom/pkg/ini"}},
			wantEdge:   "pkg/mod -> pkg/ini",
			wantReason: "dependency DAG",
		},
		{
			name:       "disallowed external import",
			imports:    map[string][]string{"pkg/vfs": {"github.com/foo/bar"}},
			wantEdge:   "pkg/vfs -> github.com/foo/bar",
			wantReason: "external import not permitted",
		},
		{
			name:     "allowed edge: ui -> render",
			imports:  map[string][]string{"pkg/ui": {"againrom/pkg/render"}},
			wantEdge: "",
		},
		{
			name:     "allowed edges: data -> vfs, formats/reg",
			imports:  map[string][]string{"pkg/data": {"againrom/pkg/vfs", "againrom/pkg/formats/reg"}},
			wantEdge: "",
		},
		{
			name:     "allowed external: formats may use x/text",
			imports:  map[string][]string{"pkg/formats/res": {"golang.org/x/text", "golang.org/x/text/encoding/charmap"}},
			wantEdge: "",
		},
		{
			name:     "text input is a registered formats leaf with the codec grant",
			imports:  map[string][]string{"pkg/formats/textinput": {"golang.org/x/text/encoding/charmap"}},
			wantEdge: "",
		},
		{
			// The tier-wide x/text grant is denied to pkg/formats/reg, which
			// converts no text: the row above proves the deny is narrow.
			name:       "denied external: formats/reg is held to the standard library",
			imports:    map[string][]string{"pkg/formats/reg": {"golang.org/x/text/encoding/charmap"}},
			wantEdge:   "pkg/formats/reg -> golang.org/x/text/encoding/charmap",
			wantReason: "external import not permitted",
		},
		{
			// The definition table's parser is registered — a package with no
			// row is a fail-closed violation — and takes no intra-module import
			// at all: it reads bytes and returns values.
			name:     "formats/databin is registered and is a leaf",
			imports:  map[string][]string{"pkg/formats/databin": {"encoding/binary", "fmt"}},
			wantEdge: "",
		},
		{
			// And it is denied the formats tier's text grant: it converts no
			// text, so the documented intention costs one line to enforce.
			name:       "denied external: formats/databin is held to the standard library",
			imports:    map[string][]string{"pkg/formats/databin": {"golang.org/x/text/encoding/charmap"}},
			wantEdge:   "pkg/formats/databin -> golang.org/x/text/encoding/charmap",
			wantReason: "external import not permitted",
		},
		{
			name:     "allowed edges: regtool -> formats/reg, formats/res",
			imports:  map[string][]string{"cmd/regtool": {"againrom/pkg/formats/reg", "againrom/pkg/formats/res"}},
			wantEdge: "",
		},
		{
			name:     "allowed edges: almtool -> formats/alm, mapload",
			imports:  map[string][]string{"cmd/almtool": {"againrom/pkg/formats/alm", "againrom/pkg/mapload"}},
			wantEdge: "",
		},
		{
			// The edge above is a GRANT and not a wildcard: almtool's census
			// verb reaches the tier that owns the classifier and nothing else,
			// so a tool that reached for the world beside it fails here.
			name:       "almtool may not reach past the tier its census counts through",
			imports:    map[string][]string{"cmd/almtool": {"againrom/pkg/sim"}},
			wantEdge:   "cmd/almtool -> pkg/sim",
			wantReason: "dependency DAG",
		},
		{
			name:     "game wildcard: may import any pkg/*",
			imports:  map[string][]string{"pkg/game": {"againrom/pkg/ui", "againrom/pkg/formats/res", "againrom/pkg/sim"}},
			wantEdge: "",
		},
		{
			name:     "stdlib in sim is not a DAG violation (CheckSimDeterminism catches these)",
			imports:  map[string][]string{"pkg/sim": {"time", "math/rand", "os"}},
			wantEdge: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vs := Check(tc.imports)
			if tc.wantEdge == "" {
				if len(vs) != 0 {
					t.Fatalf("expected no violations, got %v", vs)
				}
				return
			}
			var found *Violation
			for i := range vs {
				edge := vs[i].From
				if vs[i].Import != "" {
					edge = vs[i].From + " -> " + vs[i].Import
				}
				if edge == tc.wantEdge {
					found = &vs[i]
					break
				}
			}
			if found == nil {
				t.Fatalf("expected a violation naming %q; got %v", tc.wantEdge, vs)
			}
			if !strings.Contains(found.String(), tc.wantReason) {
				t.Errorf("violation %q does not mention %q", found.String(), tc.wantReason)
			}
		})
	}
}

// TestLoadSkipsNestedModule pins that the loader does not police packages that
// belong to a nested module (a directory with its own go.mod, e.g. a vendored
// dependency that ships one) — only this module's own packages are subject to
// the DAG.
func TestLoadSkipsNestedModule(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module againrom\n\ngo 1.26\n")
	write("pkg/foo/foo.go", "package foo\n\nimport \"strings\"\n\nvar _ = strings.TrimSpace\n")
	// nested module (a synthetic stand-in for any vendored submodule that ships
	// its own go.mod) — must not be policed
	write("extmod/go.mod", "module extmod\n\ngo 1.26\n")
	write("extmod/tools/x/main.go", "package main\n\nimport \"github.com/foo/bar\"\n")

	prod, _, err := Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, ok := prod["pkg/foo"]; !ok {
		t.Errorf("expected pkg/foo to be policed; loaded %d packages", len(prod))
	}
	for k := range prod {
		if strings.HasPrefix(k, "extmod/") {
			t.Errorf("nested-module package %q must not be policed", k)
		}
	}
}

// TestUITierAllowanceIsPinned freezes the one allow-map entry the
// front-end's simulation seam rests on (SC-9).
//
// It asserts the entry's EXACT contents rather than the absence of one name: an
// entry checked only for "pkg/sim is not in it" would not notice pkg/mapload,
// pkg/data or pkg/game arriving instead, and those widen the same boundary. The
// second half then says what the entry MEANS, by driving the evaluator itself,
// so the pin cannot go hollow if intraAllowed's prefix rule changes underneath
// it — the data and its interpretation are checked separately.
//
// What it does not do is stop anyone editing the map: it makes the edit fail
// here, where it has to be argued for, rather than pass unnoticed.
//
// PKG/AUDIO IS THE ONE ARGUED WIDENING (0126 plan T2): sound.go's SetAudio
// takes an audio.Player and sounddev.go builds the one this tree ships, so
// pkg/ui gains the audio leaf beside the render tier it already had.
func TestUITierAllowanceIsPinned(t *testing.T) {
	// Transcribed from docs/ARCHITECTURE.md's tier table, not read back out of
	// dag.go: the UI tier may use pkg/render and everything under it, plus the
	// audio leaf (0126) and video leaf (1074), and nothing else in the module.
	// Video transports presentation frames and has no game/sim dependency.
	want := []string{"pkg/render", "pkg/render/", "pkg/audio", "pkg/video"}

	got, registered := allow["pkg/ui"]
	if !registered {
		t.Fatalf("pkg/ui is not registered in the allow-map at all")
	}
	gotSorted := slices.Sorted(slices.Values(got))
	wantSorted := slices.Sorted(slices.Values(want))
	if !slices.Equal(gotSorted, wantSorted) {
		t.Fatalf("allow[\"pkg/ui\"] = %v, want exactly %v — 0020 FR-9 pins this entry, so a widening "+
			"is a decision to argue for here rather than a line in a diff", got, want)
	}

	// What that entry MEANS, edge by edge. Every tier the window package must not
	// reach is rejected by name — pkg/sim first, the boundary the seam exists to
	// keep — and the render tier and the audio leaf it may reach are not.
	for _, rel := range []string{
		"pkg/sim", "pkg/mapload", "pkg/game", "pkg/data", "pkg/vfs",
		"pkg/formats/alm", "pkg/formats/res", "pkg/formats/reg", "pkg/formats/spr256",
	} {
		vs := Check(map[string][]string{"pkg/ui": {ModulePath + "/" + rel}})
		if len(vs) != 1 || vs[0].From != "pkg/ui" || vs[0].Import != rel {
			t.Errorf("pkg/ui -> %s: got %v, want exactly one violation naming that edge", rel, vs)
		}
	}
	for _, rel := range []string{
		"pkg/render", "pkg/render/terrain", "pkg/render/camera",
		"pkg/render/frame", "pkg/render/menu", "pkg/audio", "pkg/video",
	} {
		if vs := Check(map[string][]string{"pkg/ui": {ModulePath + "/" + rel}}); len(vs) != 0 {
			t.Errorf("pkg/ui -> %s: got %v, want no violation — the render tier and the audio leaf are the UI's grant", rel, vs)
		}
	}
}

func TestCheckSimTests(t *testing.T) {
	// Structural rule only: sim tests may use the standard library (including
	// time — IO/clock/float purity is CheckSimDeterminism's, over the package's
	// non-test sources) but must not import other tiers or external modules.
	if vs := CheckSimTests([]string{"testing", "bytes", "encoding/binary", "time"}); len(vs) != 0 {
		t.Errorf("stdlib-only sim tests should be clean, got %v", vs)
	}
	vs := CheckSimTests([]string{"againrom/pkg/formats/res", "github.com/foo/bar"})
	if len(vs) != 2 {
		t.Fatalf("expected 2 sim-test violations (cross-tier + external), got %v", vs)
	}
}

// TestNoDeterministicPackageReachesTheInterpreter walks the live import graph
// from the simulation, the derived-statistics package and the rules value and
// fails if any path leads to the mod runtime or to the interpreter itself.
func TestNoDeterministicPackageReachesTheInterpreter(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	prod, _, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, start := range []string{"pkg/sim", "pkg/data", "pkg/rules"} {
		seen := map[string]bool{start: true}
		queue := []string{start}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			for _, imp := range prod[cur] {
				if strings.HasPrefix(imp, starlarkModule) {
					t.Errorf("%s reaches the interpreter through %s (%s)", start, cur, imp)
				}
				rel, ok := strings.CutPrefix(imp, ModulePath+"/")
				if !ok || seen[rel] {
					continue
				}
				if rel == "pkg/modrt" {
					t.Errorf("%s reaches the mod runtime through %s", start, cur)
				}
				seen[rel] = true
				queue = append(queue, rel)
			}
		}
		if len(seen) < 1 || len(prod[start]) == 0 {
			t.Errorf("%s: the walk read no imports", start)
		}
	}
}
