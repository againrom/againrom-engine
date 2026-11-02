package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckTextAddressNamesEachSecondForm drives the scan with a planted copy
// of each form it refuses and the forms it admits.
func TestCheckTextAddressNamesEachSecondForm(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a formatted address", "pkg/game/a.go",
			"package game\nimport \"fmt\"\nfunc f(n int) string { return fmt.Sprintf(\"%stext/inn/npc%02d.txt\", mainPrefix, n) }\n", "formats an install text address"},
		{"a pattern constant", "pkg/game/a.go",
			"package game\nconst pattern = \"main/text/battle/m%d/\"\n", "formats an install text address"},
		{"a parse form", "pkg/game/a.go",
			"package game\nimport \"fmt\"\nfunc f(s string) { var n int; fmt.Sscanf(s, \"main/text/inn/npc%d.txt\", &n) }\n", "formats an install text address"},
		{"a concatenation with a number", "pkg/game/a.go",
			"package game\nimport \"strconv\"\nfunc f(n int) string { return mainPrefix + \"text/battle/m\" + strconv.Itoa(n) + \"/\" }\n", "concatenates an install text address with a number"},
		{"ReadTexts outside the resolver", "pkg/game/a.go",
			"package game\nfunc f(m mission) { m.ReadTexts(\"d\", \"en\") }\n", "calls ReadTexts"},
		{"ReadTexts in another mod file", "pkg/mod/missions.go",
			"package mod\nfunc f(m Mission) { m.ReadTexts(\"d\", \"en\") }\n", "calls ReadTexts"},
		{"the rows", "pkg/game/installtext.go",
			"package game\nvar row = &textSource{address: mainPrefix + \"text/battle/m%d/briefing.txt\"}\nconst c = mainPrefix + \"text/\" + strconv.Itoa(1)\n", ""},
		{"the resolver", "pkg/mod/pageresolve.go",
			"package mod\nfunc f(m Mission) { m.ReadTexts(\"d\", \"en\") }\n", ""},
		{"a plain constant", "pkg/game/a.go",
			"package game\nconst town = mainPrefix + \"text/town.txt\"\nfunc f(s string) string { return s + \"text/\" }\n", ""},
		{"a test", "pkg/game/a_test.go",
			"package game\nfunc f(n int) string { return fmt.Sprintf(\"text/%d\", n) }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckTextAddress(map[string]string{tc.file: tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckTextAddress = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckTextAddress = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

// TestTextAddressHasOneTable is the live-tree half.
func TestTextAddressHasOneTable(t *testing.T) {
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
	if !strings.Contains(files[textAddressRowsFile], "type textSource struct") || !strings.Contains(files[textResolverFile], "func ResolvePages(") {
		t.Fatal("the rows file or the resolver file moved; the scan guards nothing")
	}
	for _, v := range CheckTextAddress(files) {
		t.Errorf("%s", v)
	}
}
