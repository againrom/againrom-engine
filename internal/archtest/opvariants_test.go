package archtest

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOperationVariantsHaveOneRowForm keeps the readings by first parameter
// in Op.Variants: the older command and form fields are not written again.
func TestOperationVariantsHaveOneRowForm(t *testing.T) {
	gone := []string{"CommandSignatures", "CommandWord:", "op.CommandWord", ".FormOf(", "scriptprofile.Form{", "Forms: []", "Commands: groupCommands", "func groupCommand("}
	root := filepath.Join("..", "..")
	for _, dir := range []string{"pkg", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, g := range gone {
				if strings.Contains(string(src), g) {
					t.Errorf("%s holds %q; an operation's readings by first parameter are Op.Variants", path, g)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
