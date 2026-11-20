package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestScriptOperationReadersHaveNoGameBranch keeps the readers of the script
// operation rows free of a comparison on the game's operation vocabulary: a
// difference between the games is row data in pkg/scriptprofile.
func TestScriptOperationReadersHaveNoGameBranch(t *testing.T) {
	for _, name := range []string{
		"pkg/ui/mapeditortrigger.go",
		"pkg/ui/mapeditortriggerwords.go",
		"pkg/mapload/scriptvocabulary.go",
		"pkg/mapload/scriptsignature.go",
		"pkg/game/mapinspectiontriggers.go",
		"pkg/game/mapinspectionscriptauthor.go",
	} {
		src, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(src), "OpsSecond") {
			t.Errorf("%s compares on OpsSecond; read the operation row or the vocabulary instead", name)
		}
	}
}
