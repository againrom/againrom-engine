package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMessageMarkupUsesOneCodec(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"eventtext.go", "mapmessagepages.go"} {
		b, err := os.ReadFile(filepath.Join(root, "pkg/game", name))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "decodeMessageMarkup(") || strings.Contains(string(b), "regexp.") || strings.Contains(string(b), "strings.IndexByte(s[i:], '<')") {
			t.Fatal(name, "must read message tags through the shared codec")
		}
	}
}
