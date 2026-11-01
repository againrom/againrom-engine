package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// witnessHome is the one package in which a test file may encode, write or
// decode an image; every other test hands its image to witness.PNG.
const witnessHome = "internal/witness"

// witnessImagePackages are the imports through which a test could encode or
// decode an image by itself.
var witnessImagePackages = map[string]bool{
	"image/png": true, "image/jpeg": true, "image/gif": true,
	"golang.org/x/image/bmp": true, "golang.org/x/image/tiff": true,
	"golang.org/x/image/webp": true, "golang.org/x/image/vp8": true,
	"golang.org/x/image/vp8l": true,
}

// witnessFileCalls are the os functions that open an image file for writing.
var witnessFileCalls = map[string]bool{"Create": true, "WriteFile": true, "OpenFile": true}

var witnessImageSuffixes = []string{".png", ".jpg", ".jpeg", ".gif"}

// WitnessSites lists the places in test files, outside the witness package,
// where a test writes an image by its own means: an import of an image codec,
// and a call that opens or writes a file whose name is spelled as an image
// file. Production code is not scanned.
func WitnessSites(root string) ([]string, error) {
	var sites []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "knowledge", ".git", "testdata", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		if !strings.HasSuffix(rel, "_test.go") || strings.HasPrefix(rel, witnessHome+"/") {
			return nil
		}
		src, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		sites = append(sites, witnessInSource(rel, src)...)
		return nil
	})
	sort.Strings(sites)
	return sites, err
}

// WitnessHomeHolds reports whether the witness package still holds PNG, so
// that a rename cannot leave the scan with nothing to point at.
func WitnessHomeHolds(root string) (bool, error) {
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(witnessHome)))
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if e.IsDir() || strings.HasSuffix(e.Name(), "_test.go") || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(witnessHome), e.Name()))
		if err != nil {
			return false, err
		}
		file, err := parser.ParseFile(token.NewFileSet(), e.Name(), src, 0)
		if err != nil {
			return false, err
		}
		for _, d := range file.Decls {
			if f, ok := d.(*ast.FuncDecl); ok && f.Recv == nil && f.Name.Name == "PNG" {
				return true, nil
			}
		}
	}
	return false, nil
}

func witnessInSource(name string, src []byte) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil
	}
	var hits []string
	add := func(pos token.Pos, what string) {
		hits = append(hits, name+":"+strconv.Itoa(fset.Position(pos).Line)+" "+what)
	}
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err == nil && witnessImagePackages[path] {
			add(imp.Pos(), "imports "+path)
		}
	}
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "os" || !witnessFileCalls[sel.Sel.Name] {
			return true
		}
		for _, arg := range call.Args {
			if lit := imageFileLiteral(arg); lit != "" {
				add(call.Pos(), "os."+sel.Sel.Name+" opens "+lit)
				break
			}
		}
		return true
	})
	return hits
}

// imageFileLiteral returns the first string literal inside e that ends in an
// image file extension. It looks through joins, sums and calls, and not into
// composite literals or function literals, whose strings name other things.
func imageFileLiteral(e ast.Expr) string {
	found := ""
	ast.Inspect(e, func(n ast.Node) bool {
		if found != "" {
			return false
		}
		switch n := n.(type) {
		case *ast.CompositeLit, *ast.FuncLit:
			return false
		case *ast.BasicLit:
			if n.Kind != token.STRING {
				return true
			}
			s, err := strconv.Unquote(n.Value)
			if err != nil {
				return true
			}
			lower := strings.ToLower(s)
			for _, suffix := range witnessImageSuffixes {
				if strings.HasSuffix(lower, suffix) {
					found = n.Value
					return false
				}
			}
		}
		return true
	})
	return found
}
