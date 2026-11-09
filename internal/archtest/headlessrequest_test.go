package archtest

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func headlessRequestFixture(named int) map[string]string {
	var b strings.Builder
	b.WriteString("package ui\nfunc (a *App) HeadlessPicture(req HeadlessRequest) (*image.RGBA, HeadlessExtras, error) { return nil, HeadlessExtras{}, nil }\nfunc ParseWindowSize(value string, empty image.Point) (image.Point, error) { return empty, nil }\n")
	for i := range named {
		fmt.Fprintf(&b, "func (e *MapEditor) Named%cRect() (image.Rectangle, bool) { return image.Rectangle{}, false }\n", rune('A'+i))
	}
	return map[string]string{"pkg/ui/a.go": b.String()}
}

func TestCheckHeadlessRequestNamesEachFault(t *testing.T) {
	if v := CheckHeadlessRequest(headlessRequestFixture(namedControlAccessorsRemaining)); len(v) != 0 {
		t.Fatalf("clean fixture: %v", v)
	}
	cases := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"second picture", func() map[string]string {
			f := headlessRequestFixture(namedControlAccessorsRemaining)
			f["pkg/ui/a.go"] += "func (a *App) HeadlessExtraFrame() (*image.RGBA, error) { return nil, nil }\n"
			return f
		}(), "HeadlessPicture only"},
		{"second parser", func() map[string]string {
			f := headlessRequestFixture(namedControlAccessorsRemaining)
			f["cmd/x/main.go"] = "package main\nvar _, _ = fmt.Sscanf(v, \"%dx%d\", &a, &b)\n"
			return f
		}(), "window size parser"},
		{"rose", headlessRequestFixture(namedControlAccessorsRemaining + 1), "rose to"},
		{"fell", headlessRequestFixture(namedControlAccessorsRemaining - 1), "fell to"},
	}
	for _, tc := range cases {
		found := false
		for _, v := range CheckHeadlessRequest(tc.files) {
			found = found || strings.Contains(v.Reason, tc.want)
		}
		if !found {
			t.Errorf("%s: no violation naming %q", tc.name, tc.want)
		}
	}
}

// TestHeadlessRequestHasOneBuilder is the live-tree half.
func TestHeadlessRequestHasOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadHeadlessRequestSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckHeadlessRequest(files) {
		t.Error(v)
	}
}
