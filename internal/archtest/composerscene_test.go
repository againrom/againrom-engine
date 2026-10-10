package archtest

import (
	"os"
	"strings"
	"testing"
)

func composerSceneTree() map[string]string {
	return map[string]string{
		"pkg/town/scene.go": "package town\ntype Host interface{ Art() *Art }\ntype Scene struct{}\n" +
			"func (s *Scene) Advance() {}\nfunc (s *Scene) Paint() {}\n",
		"pkg/town/art.go":           "package town\ntype Art struct{}\nfunc LoadArt() (*Art, error) { return nil, nil }\n",
		"pkg/game/townscenehost.go": "package game\ntype townSceneHost struct{}\nfunc (h townSceneHost) Art() *town.Art { return nil }\n",
	}
}

func TestCheckComposerSceneNamesEachSecondBuilder(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a second runtime", "pkg/town/view.go",
			"package town\ntype View struct{}\nfunc (v *View) Advance() {}\nfunc (v *View) Paint() {}\n",
			"pkg/town:View is a second composer-scene runtime"},
		{"a second host interface", "pkg/town/page.go",
			"package town\ntype PageHost interface{ Art() *Art }\n",
			"pkg/town:PageHost is a second composer-scene host interface"},
		{"a second manifest loader", "pkg/town/page.go",
			"package town\nfunc LoadSceneArt() (*Art, error) { return nil, nil }\n",
			"pkg/town:LoadSceneArt is a second composer-scene manifest loader"},
		{"a second adapter", "pkg/game/roompage.go",
			"package game\ntype roomPageHost struct{}\nfunc (h roomPageHost) Art() *town.Art { return nil }\n",
			"pkg/game:roomPageHost is a second composer-scene adapter"},
		{"a type with only one of the two runtime methods", "pkg/town/clock.go",
			"package town\ntype Clock struct{}\nfunc (c *Clock) Advance() {}\n", ""},
		{"a game screen that advances and paints", "pkg/game/screen.go",
			"package game\ntype screen struct{}\nfunc (s *screen) Advance() {}\nfunc (s *screen) Paint() {}\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := composerSceneTree()
			files[tc.file] = tc.src
			var got []string
			for _, v := range CheckComposerScene(files) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckComposerScene = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckComposerScene = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

func TestCheckComposerSceneRefusesAMissingBuilder(t *testing.T) {
	vs := CheckComposerScene(map[string]string{"pkg/ui/a.go": "package ui\n"})
	if len(vs) != len(composerSceneOne) {
		t.Fatalf("CheckComposerScene over an empty package = %v, want %d missing builders", vs, len(composerSceneOne))
	}
	if vs := CheckComposerScene(nil); len(vs) != 1 {
		t.Fatalf("CheckComposerScene(nil) = %v, want one violation", vs)
	}
}

func TestComposerScenesHaveOneRuntime(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckComposerScene(files) {
		t.Errorf("%s", v)
	}
}
