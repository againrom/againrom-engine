package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCapturingSurfacesPassInputThroughOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(root, "pkg", "ui", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	for _, route := range []struct{ file, call string }{
		{"cheatinput.go", "in.passing(passCheatChat)"},
		{"app.go", "in.passing(passKeysCapture)"},
	} {
		src := read(route.file)
		if !strings.Contains(src, route.call) {
			t.Errorf("%s no longer builds its captured tick with %s", route.file, route.call)
		}
	}
	for _, file := range []string{"cheatinput.go", "keyspage.go", "app.go"} {
		src := read(file)
		if strings.Contains(src, "capturedByKeysPage") {
			t.Errorf("%s: the hand field list capturedByKeysPage is back", file)
		}
		if strings.Contains(src, "return appInput{") {
			t.Errorf("%s returns a hand-built appInput; use passing", file)
		}
	}
}
