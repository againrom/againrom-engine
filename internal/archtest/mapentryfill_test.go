package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMapEntryFilesAskNoSessionIdentity keeps the map entry one body over the
// data and steps its filler supplies: no nil test on a session in the entry
// files.
func TestMapEntryFilesAskNoSessionIdentity(t *testing.T) {
	for _, name := range []string{"missionentry.go", "difficulty.go"} {
		src, err := os.ReadFile(filepath.Join("..", "..", "pkg", "game", name))
		if err != nil {
			t.Fatal(err)
		}
		for _, bad := range []string{".session == nil", ".session != nil"} {
			if strings.Contains(string(src), bad) {
				t.Errorf("%s tests %q: the filler supplies the difference", name, bad)
			}
		}
	}
}
