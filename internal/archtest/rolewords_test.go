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

// roleWordTableFile holds signatureRoleWords, the one home of the role-word
// locale keys in pkg/ui.
const roleWordTableFile = "pkg/ui/mapeditortriggerwords.go"

const roleWordKeyPrefix = `"editor.role.`

// roleWordSpellers are the non-test pkg/ui files other than the table that
// spell a role-word locale key as a string literal.
func roleWordSpellers(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	for name, source := range files {
		if name == roleWordTableFile || !strings.HasPrefix(name, "pkg/ui/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING && strings.HasPrefix(lit.Value, roleWordKeyPrefix) {
				out = append(out, name+":"+lit.Value)
			}
			return true
		})
	}
	sort.Strings(out)
	return out
}

func TestSignatureRoleWordsHaveOneTable(t *testing.T) {
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
	if _, ok := files[roleWordTableFile]; !ok {
		t.Fatalf("%s is missing; the scan guards nothing", roleWordTableFile)
	}
	for _, s := range roleWordSpellers(t, files) {
		t.Errorf("a role word key is spelled outside %s: %s", roleWordTableFile, s)
	}
}
