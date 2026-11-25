package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGroupCommandOpcodeIsReadFromItsRow keeps the editor's readers of the
// group command off its opcode: the row is found by its shape and carries the
// fallback command.
func TestGroupCommandOpcodeIsReadFromItsRow(t *testing.T) {
	for _, name := range []string{
		"pkg/ui/mapeditorscriptmodel.go",
		"pkg/ui/mapeditortrigger.go",
		"pkg/ui/mapeditorwizard.go",
	} {
		src, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		for _, bad := range []string{"scriptGroupCommand", "command = 3"} {
			if strings.Contains(string(src), bad) {
				t.Errorf("%s holds %q; read the operation row's Fallback or GroupRow", name, bad)
			}
		}
	}
}
