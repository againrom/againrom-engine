package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckTavernTalkTextNamesASecondChoice drives the scan with synthetic
// sources.
func TestCheckTavernTalkTextNamesASecondChoice(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a launch check choosing again", "pkg/game/a.go",
			"package game\nfunc check(npc, m, ch int) string { if m == 0 { m = ch }; p, _ := TownNPCTextPath(npc, m); return p }\n", "check calls TownNPCTextPath"},
		{"the chooser", "pkg/game/a.go",
			"package game\nfunc tavernTalkTextPath(npc, m, ch int) (string, bool) { return TownNPCTextPath(npc, m) }\n", ""},
		{"the leaf recogniser", "pkg/game/a.go",
			"package game\nfunc isTavernTalkLeaf(a string) bool { _, ok := TownNPCTextPath(1, 2); return ok }\n", ""},
		{"a test", "pkg/game/a_test.go",
			"package game\nfunc f() { TownNPCTextPath(1, 2) }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckTavernTalkText(map[string]string{tc.file: tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckTavernTalkText = %v, want none", got)
				}
				return
			}
			if len(got) == 0 || !strings.Contains(strings.Join(got, "\n"), tc.want) {
				t.Fatalf("CheckTavernTalkText = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

// TestTavernTalkTextHasOneChooser is the live-tree half.
func TestTavernTalkTextHasOneChooser(t *testing.T) {
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
	if !strings.Contains(files["pkg/game/townscreen.go"], "func tavernTalkTextPath(") {
		t.Fatal("pkg/game/townscreen.go holds no tavernTalkTextPath; the scan guards nothing")
	}
	for _, v := range CheckTavernTalkText(files) {
		t.Errorf("%s", v)
	}
}
