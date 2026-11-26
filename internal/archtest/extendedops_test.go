package archtest

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/scriptprofile"
)

// TestExtendedOpsHaveOneTable pins the one table of extended-record
// operations: an Againrom opcode of a row is written as a number only in the
// table's file, and the one refusal check is asked only by the library fold
// and the game's refusal of the original record.
func TestExtendedOpsHaveOneTable(t *testing.T) {
	root := filepath.Join("..", "..")
	table := filepath.Join(root, "pkg", "scriptprofile", "extendedops.go")
	var literals []string
	for _, o := range scriptprofile.ExtendedOps {
		for _, op := range o.Ops {
			if row, ok := scriptprofile.OpRow(scriptprofile.OpsFirst, op.Condition, op.ID); ok && row.Again {
				literals = append(literals, fmt.Sprintf("0x%x", op.ID))
			}
		}
	}
	if len(literals) < 5 {
		t.Fatalf("the table names %d Againrom opcodes, want at least 5", len(literals))
	}
	asks := map[string]bool{}
	for _, dir := range []string{"pkg", "cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := strings.ToLower(string(src))
			if path != table {
				for _, lit := range literals {
					if strings.Contains(text, lit) {
						t.Errorf("%s writes opcode %s; name it from scriptprofile's extended-op table", path, lit)
					}
				}
			}
			if strings.Contains(string(src), "ExtendedOnly(") {
				asks[filepath.ToSlash(path)] = true
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	want := map[string]bool{"../../pkg/mapedit/scriptform.go": true, "../../pkg/game/mapinspectionscriptform.go": true}
	for path := range asks {
		if !want[path] {
			t.Errorf("%s asks mapedit.ExtendedOnly; the refusal of the original record has one path (originalRefusal)", path)
		}
	}
	for path := range want {
		if !asks[path] {
			t.Errorf("%s no longer asks mapedit.ExtendedOnly", path)
		}
	}
}
