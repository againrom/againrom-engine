package archtest

import (
	"os"
	"strings"
	"testing"
)

func hostModesTree() map[string]string {
	return map[string]string{
		"pkg/game/town.go": "package game\ntype Town struct{ talk talkCursor }\ntype talkCursor struct{}\n" +
			"func f(t *townScreen) bool { return t.hostRules().Pages == town.PagesEntry }\n",
		"pkg/town/validate.go": "package town\nfunc g(d *Description) bool { h := d.Host; return h.Shop == ShopBank }\n",
	}
}

func TestCheckHostModesNamesEachPlantedShape(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a literal on the right", "pkg/game/a.go",
			"package game\nfunc f(t *townScreen) bool { return t.hostRules().Pages == \"entry\" }\n", "host mode Pages is compared"},
		{"a literal on the left", "pkg/game/a.go",
			"package game\nfunc f(t *townScreen) bool { return \"bank\" != t.hostRules().Shop }\n", "host mode Shop is compared"},
		{"a local read from hostRules", "pkg/game/a.go",
			"package game\nfunc f(t *townScreen) bool { r := t.hostRules(); return r.Talk == \"section\" }\n", "host mode Talk is compared"},
		{"a description's host", "pkg/town/a.go",
			"package town\nfunc f(d *Description) bool { return d.Host.Rows == \"hotspots\" }\n", "host mode Rows is compared"},
		{"a pointer read of the host", "pkg/game/a.go",
			"package game\nfunc f(d *town.Description) bool { h := *d.Host; return h.Art == \"pages\" }\n", "host mode Art is compared"},
		{"a switch on a mode", "pkg/game/a.go",
			"package game\nfunc f(t *townScreen) { switch t.hostRules().Footer { case \"party\": } }\n", "host mode Footer is compared"},
		{"a constant comparison", "pkg/game/a.go",
			"package game\nfunc f(t *townScreen) bool { return t.hostRules().Pages == town.PagesRetained }\n", ""},
		{"a field of another value", "pkg/game/a.go",
			"package game\nfunc f(p *page) bool { return p.policy().Leave == \"square\" }\n", ""},
		{"a text that is not a mode", "pkg/game/a.go",
			"package game\nfunc f(t *townScreen) bool { return t.hostRules().HeaderPrefix == \"x\" }\n", ""},
		{"a second talk cursor field", "pkg/game/a.go",
			"package game\ntype secondCampaign struct{ talk talkCursor }\n", "secondCampaign holds a talkCursor field"},
		{"a second talk cursor pointer field", "pkg/game/a.go",
			"package game\ntype screen struct{ page *talkCursor }\n", "screen holds a talkCursor field"},
		{"a talkCursor method", "pkg/game/a.go",
			"package game\nfunc (c campaign) talkCursor() *talkCursor { return nil }\n", "method talkCursor is a second talk cursor home"},
		{"an accessor of the home", "pkg/game/a.go",
			"package game\nfunc (t *townScreen) talkPage() *talkCursor { return nil }\n", ""},
		{"a local cursor", "pkg/game/a.go",
			"package game\nfunc f() { var c talkCursor; _ = c }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := hostModesTree()
			files[tc.file] = tc.src
			var got []string
			for _, v := range CheckHostModes(files) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckHostModes = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckHostModes = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

func TestCheckHostModesRefusesAMissingHome(t *testing.T) {
	if vs := CheckHostModes(map[string]string{"pkg/game/a.go": "package game\n"}); len(vs) != 1 {
		t.Fatalf("CheckHostModes without a home = %v, want one violation", vs)
	}
	if vs := CheckHostModes(nil); len(vs) != 1 {
		t.Fatalf("CheckHostModes(nil) = %v, want one violation", vs)
	}
}

func TestHostModesAreTypedAndTheTalkCursorHasOneHome(t *testing.T) {
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
	for _, v := range CheckHostModes(files) {
		t.Errorf("%s", v)
	}
}
