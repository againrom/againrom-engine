package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

var seedRuleFuncs = map[string]bool{"castSeed": true, "visualCastSeed": true, "restoredDefaultSeed": true}

var seedResolverFuncs = map[string]bool{"presentationSeed": true, "visualCastSeed": true}

var seedResolverFiles = map[string]bool{"pkg/game/flight.go": true, "pkg/game/actionvisual.go": true}

func presentationSeedBypasses(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	for name, source := range files {
		if !strings.HasPrefix(name, "pkg/game/") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			fn, isFunc := decl.(*ast.FuncDecl)
			inResolver := isFunc && seedResolverFuncs[fn.Name.Name]
			ast.Inspect(decl, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.CallExpr:
					switch fun := n.Fun.(type) {
					case *ast.Ident:
						if seedRuleFuncs[fun.Name] && !inResolver {
							out = append(out, name+":"+fun.Name)
						}
					case *ast.SelectorExpr:
						if seedRuleFuncs[fun.Sel.Name] && !inResolver {
							out = append(out, name+":"+fun.Sel.Name)
						}
						if x, ok := fun.X.(*ast.Ident); ok && x.Name == "rand" && fun.Sel.Name == "New" && seedResolverFiles[name] && !inResolver {
							out = append(out, name+":rand.New")
						}
					}
				case *ast.CompositeLit:
					if typ, ok := n.Type.(*ast.Ident); !ok || typ.Name != "flightLook" {
						return true
					}
					for _, elt := range n.Elts {
						kv, ok := elt.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						if key, ok := kv.Key.(*ast.Ident); !ok || key.Name != "seed" {
							continue
						}
						if !callsResolver(kv.Value) {
							out = append(out, name+":flightLook.seed")
						}
					}
				}
				return true
			})
		}
	}
	sort.Strings(out)
	return out
}

func callsResolver(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "presentationSeed"
}

func TestPresentationSeedScanNamesEachBypass(t *testing.T) {
	got := presentationSeedBypasses(t, map[string]string{
		"pkg/game/flight.go": "package game\nfunc (mw *mapWorld) a(p P) { _ = flightLook{seed: uint32(p.ID) + 1}; _ = flightLook{seed: mw.presentationSeed(id)}; _ = rand.New(src) }\n",
		"pkg/game/actionvisual.go": "package game\nfunc (mw *mapWorld) presentationSeed(id I) uint32 { return castSeed(0, 0, 0) + restoredDefaultSeed(1) }\n" +
			"func (mw *mapWorld) visualCastSeed(c, t E, s int, a bool) uint32 { return castSeed(c, t, s) }\n",
		"pkg/game/effectmark.go": "package game\nfunc (mw *mapWorld) b() uint32 { return castSeed(0, mw.visualActorLabel(1), 2) + mw.visualCastSeed(1, 2, 3, false) }\n",
		"pkg/game/shop.go":       "package game\nfunc c() { _ = rand.New(src) }\n",
	})
	want := []string{"pkg/game/effectmark.go:castSeed", "pkg/game/effectmark.go:visualCastSeed", "pkg/game/flight.go:flightLook.seed", "pkg/game/flight.go:rand.New"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("presentationSeedBypasses = %v", got)
	}
}

// presentationSeed builds every presentation seed.
func TestPresentationSeedHasOneResolver(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for name := range seedResolverFiles {
		if _, ok := files[name]; !ok {
			t.Fatalf("%s is missing; the scan guards nothing", name)
		}
	}
	if !strings.Contains(files["pkg/game/actionvisual.go"], "func (mw *mapWorld) presentationSeed(") {
		t.Fatal("the seed resolver left pkg/game/actionvisual.go; the scan guards nothing")
	}
	if got := presentationSeedBypasses(t, files); len(got) != 0 {
		t.Fatalf("presentation seeds made outside the resolver: %v", got)
	}
}
