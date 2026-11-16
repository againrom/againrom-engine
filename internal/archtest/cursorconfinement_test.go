package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCursorConfinementUsesTheAppPointerRoute(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(root, "pkg", "ui", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	nativeFiles, err := filepath.Glob(filepath.Join(root, "pkg", "ui", "cursorclip", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, nativeFiles...)
	clips, applies := 0, 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		source := strings.ReplaceAll(string(data), "\r\n", "\n")
		clips += strings.Count(source, `"ClipCursor"`)
		applies += strings.Count(source, ".pointerClip.apply()")
		if filepath.Base(path) == "app.go" {
			if !strings.Contains(source, "func (a *App) applyPointerMode(draw bool) {\n\tif !draw {\n\t\ta.pointerClip.apply()") ||
				!strings.Contains(source, "defer a.pointerClip.release()") {
				t.Error("App must own cursor confinement and release it at exit")
			}
		}
	}
	if clips != 1 || applies != 1 {
		t.Fatalf("cursor confinement has %d native bindings and %d App routes, want one each", clips, applies)
	}
}

func TestCursorConfinementNativeImportsStayInLeaf(t *testing.T) {
	for _, imp := range []string{"github.com/ebitengine/purego", "github.com/ebitengine/purego/objc", "golang.org/x/sys/windows"} {
		if vs := Check(map[string][]string{"pkg/ui/cursorclip": {imp}}); len(vs) != 0 {
			t.Errorf("native leaf cannot import %s: %v", imp, vs)
		}
		if vs := Check(map[string][]string{"pkg/ui": {imp}}); len(vs) != 1 {
			t.Errorf("UI must keep %s in the native leaf: %v", imp, vs)
		}
	}
	if vs := Check(map[string][]string{"pkg/ui/cursorclip": {"againrom/pkg/sim"}}); len(vs) != 1 {
		t.Errorf("cursor confinement must not reach simulation: %v", vs)
	}
}
