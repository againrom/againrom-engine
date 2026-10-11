package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestLiveScreensReadArtThroughTheOneLoader measures the real tree: no
// production file under pkg/ outside the one screen art loader names or
// opens an install picture, except the files of another kind and the
// falling debt list, each entry of which must still hold a finding.
func TestLiveScreensReadArtThroughTheOneLoader(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil || len(files) == 0 {
		t.Fatalf("sources: %v (%d files)", err, len(files))
	}
	found, err := ScreenArtFindings(files)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckScreenArt(found, ScreenArtDebt) {
		t.Error(v)
	}
	for name := range screenArtOtherKinds {
		if _, ok := files[name]; !ok {
			t.Errorf("%s is listed as another kind of picture reader and no longer exists", name)
		}
	}
}

// TestScreenArtFindingsSeeEachShape holds the scan against synthetic
// sources, so a scan that went blind fails here rather than passing a dirty
// tree.
func TestScreenArtFindingsSeeEachShape(t *testing.T) {
	files := map[string]string{
		"pkg/game/panel.go": `package game

const path = "graphics/interface/panel.bmp"

func sheet() string { return fmt.Sprintf("%s/sprites.16a", dir) }

func load(src entrySource) { readInstallBMP(src, path); bmp.DecodeRGBA(nil); spr256.Decode(nil) }
`,
		"pkg/game/screenart.go": `package game

const loader = "x.bmp"

func read() { readInstallBMP(nil, loader) }
`,
		"pkg/screenart/load.go": "package screenart\n\nconst k = \"a.256\"\n",
		"pkg/mapload/range.go":  "package mapload\n\nconst r = \"dimensions in 8..256\"\n\nfunc f() { bmp.Encode(nil) }\n",
	}
	found, err := ScreenArtFindings(files)
	if err != nil {
		t.Fatal(err)
	}
	var shapes []string
	for _, f := range found {
		if f.File != "pkg/game/panel.go" {
			t.Errorf("finding in the loader: %v", f)
		}
		shapes = append(shapes, f.Shape)
	}
	want := []string{`names an install picture "graphics/interface/panel.bmp"`, `names an install picture "%s/sprites.16a"`,
		"opens a picture through readInstallBMP", "opens a picture through bmp.DecodeRGBA", "opens a picture through spr256.Decode"}
	if strings.Join(shapes, "|") != strings.Join(want, "|") {
		t.Fatalf("shapes = %q, want %q", shapes, want)
	}
}

// TestCheckScreenArtHoldsTheDebtListFalling: a finding outside the lists
// fails, a debt entry keeps its place only while it still holds a finding.
func TestCheckScreenArtHoldsTheDebtListFalling(t *testing.T) {
	debt := map[string]bool{"a.go": true, "b.go": true}
	got := CheckScreenArt([]ScreenArtFinding{{File: "a.go", Line: 1, Shape: "x"}, {File: "c.go", Line: 2, Shape: "y"}}, debt)
	if len(got) != 2 || !strings.HasPrefix(got[0], "c.go:2 y") || !strings.HasPrefix(got[1], "b.go no longer") {
		t.Fatalf("violations = %v", got)
	}
	if got := CheckScreenArt([]ScreenArtFinding{{File: "a.go"}, {File: "b.go"}}, debt); len(got) != 0 {
		t.Fatalf("a clean tree = %v", got)
	}
}
