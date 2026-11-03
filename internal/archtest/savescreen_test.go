package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestCheckSaveScreenNamesPlantedShapes(t *testing.T) {
	const state = "package ui\ntype savePageState struct { page SavePage; list *Picker; ui loadWindow }\n"
	for _, tc := range []struct{ name, source, want string }{
		{"type", "type saveDialog struct{}", "saveDialog"},
		{"field", "type unrelated struct { saveDialog int }", "saveDialog"},
		{"variable", "func f() { saveDialog := 1; _ = saveDialog }", "saveDialog"},
		{"flow holder", "type flow struct { saveScreen, spare *savePageState }", "flow.spare"},
		{"page holder", "type pageSeams struct { spare *savePageState }", "pageSeams.spare"},
		{"alias", "type alias = savePageState; type flow struct { spare *alias }", "flow.spare"},
		{"container", "type flow struct { spare map[string][]*savePageState }", "flow.spare"},
		{"map key", "type flow struct { spare map[*savePageState]string }", "flow.spare"},
		{"channel", "type flow struct { spare chan *savePageState }", "flow.spare"},
		{"nested", "type nested struct { spare *savePageState }; type flow struct { pages nested }", "flow.pages.spare"},
		{"renamed request state", "type alternate struct { request SaveRequest; list *Picker }; type flow struct { spare *alternate }", "flow.spare"},
		{"renamed page state", "type alternate struct { page SavePage; list *Picker; ui loadWindow }; type flow struct { spare *alternate }", "flow.spare"},
		{"one holder and descriptions", "type flow struct { saveScreen *savePageState; pages pageSeams }; type pageSeams struct { loadSet, saveSet *SavePage }", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vs := CheckSaveScreen(map[string]string{"pkg/ui/screen.go": state + tc.source})
			if tc.want == "" {
				if len(vs) != 0 {
					t.Fatal(vs)
				}
				return
			}
			if len(vs) == 0 {
				t.Fatal("planted shape passed")
			}
			for _, v := range vs {
				if !strings.Contains(v.Reason, tc.want) || !strings.HasPrefix(v.From, "pkg/ui/screen.go:") {
					t.Fatal(v)
				}
			}
		})
	}
	if len(CheckSaveScreen(nil)) != 1 {
		t.Fatal("an empty source set passed")
	}
}

func TestSaveScreenHasOneBuilder(t *testing.T) {
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
	for _, v := range CheckSaveScreen(files) {
		t.Error(v)
	}
}
