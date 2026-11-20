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

// scrollBarHome is the file of the bar's one press and drag step.
const scrollBarHome = "pkg/ui/widgetscroll.go"

// scrollBarOnly are the calls only the step's file makes: they turn a press
// or a drag row into a bar position. Elsewhere a screen holds a scrollOwner
// and scrollBarInput.drive reads it. listBarRequest is the model's apply and
// is called from scrolllist.go alone.
var scrollBarOnly = map[string]string{
	"dragPos": scrollBarHome, "dragPosFrom": scrollBarHome, "listBarRequest": scrollListHome,
}

// ScrollBarSites lists the places in production code of pkg/ui, outside the
// step's file, that compute a bar position from a press or apply a bar
// request themselves. Tests are not scanned.
func ScrollBarSites(root string) ([]string, error) {
	var sites []string
	err := filepath.WalkDir(filepath.Join(root, "pkg", "ui"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
			return nil
		}
		src, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		sites = append(sites, scrollBarInSource(rel, src)...)
		return nil
	})
	sort.Strings(sites)
	return sites, err
}

func scrollBarInSource(name string, src []byte) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil
	}
	var hits []string
	ast.Inspect(file, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if home, only := scrollBarOnly[calleeName(c)]; only && home != name {
			hits = append(hits, name+":"+strconv.Itoa(fset.Position(c.Pos()).Line)+" "+calleeName(c)+" is called outside "+home)
		}
		return true
	})
	return hits
}
