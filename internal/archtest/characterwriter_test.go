package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModCharactersUseTheKindAndRowWriter(t *testing.T) {
	writer, err := os.ReadFile("../../pkg/mod/characterwrite.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(writer), "writeRows(") {
		t.Fatal("characters must use the shared row writer")
	}
	for _, dir := range []string{"../../pkg/mod", "../../pkg/game"} {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			src, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(src), `[[character]]\n`) {
				t.Errorf("%s emits character tables outside the shared row writer", file)
			}
		}
	}
}
