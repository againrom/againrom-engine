package archtest

import (
	"os"
	"strings"
	"testing"
)

func innPageTree() map[string]string {
	return map[string]string{
		"pkg/game/inn.go": "package game\nvar hostInnProfiles = map[int]*innProfile{0: {rows: innRowProfile{}}}\n" +
			"func hostInnProfile(m int) innProfile { return *hostInnProfiles[m] }\n" +
			"func (t *townScreen) innRows() []innRow { return innPageRows() }\n",
		"pkg/game/installtext.go": "package game\nvar namedMainWords = []func(w *ui.Words) *string{func(w *ui.Words) *string { return &w.TavernHire }}\n",
	}
}

func TestCheckInnPageNamesEachPlantedShape(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{"a profile picked at a call site", "package game\nfunc f() innProfile { return *hostInnProfiles[1] }\n", "f reads hostInnProfiles"},
		{"a profile literal", "package game\nfunc f() innProfile { return innProfile{keepSpeakers: true} }\n", "f builds an innProfile literal"},
		{"a row profile literal", "package game\nvar rows = innRowProfile{sleep: true}\n", "rows builds an innRowProfile literal"},
		{"a second row producer", "package game\nfunc (c *secondCampaign) tavernRows() []innRow { return innPageRows() }\n", "tavernRows calls innPageRows"},
		{"a caption built in Go", "package game\nfunc f(t *townScreen) string { return t.in.Words.TavernFire }\n", "f reads the named word TavernFire"},
		{"a page compared with an inn name", "package game\nfunc f(r *town.RoomSpec) bool { return \"inn\" == r.Page }\n", "f compares a page"},
		{"a page arm on an inn name", "package game\nfunc f(r *town.RoomSpec) { switch r.Page { case \"school\", \"tavern\": } }\n", "f switches a page"},
		{"the retired square entry", "package game\nfunc AtTownSquare(r *town.RoomSpec) bool { return r.Page == \"inn\" }\n", "AtTownSquare compares a page"},
		{"a page that is not an inn", "package game\nfunc f(r *town.RoomSpec) bool { return r.Page == \"school\" }\n", ""},
		{"a room name", "package game\nfunc f(t *townScreen) bool { return t.roomName() == \"inn\" }\n", ""},
		{"the picker", "package game\nfunc g() innProfile { return hostInnProfile(0) }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := innPageTree()
			files["pkg/game/a.go"] = tc.src
			var got []string
			for _, v := range CheckInnPage(files) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckInnPage = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckInnPage = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

func TestCheckInnPageRefusesAnEmptyTree(t *testing.T) {
	if vs := CheckInnPage(map[string]string{"pkg/ui/a.go": "package ui\n"}); len(vs) != 1 {
		t.Fatalf("CheckInnPage without pkg/game = %v, want one violation", vs)
	}
}

func TestInnPageHasOneBuilderPerPart(t *testing.T) {
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
	for _, v := range CheckInnPage(files) {
		t.Errorf("%s", v)
	}
}
