package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckInnOffersNamesAnOptionTable drives the scan with synthetic
// sources.
func TestCheckInnOffersNamesAnOptionTable(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a stage table in Go", "pkg/game/a.go",
			"package game\nvar stages = map[int32][]innOffer{10: {{kind: 0, topic: 9, npc: 207}}}\n", "package scope writes an inn option"},
		{"a second producer", "pkg/game/a.go",
			"package game\nfunc tail() []innOffer { return []innOffer{{kind: 3, topic: 62, npc: 22}} }\n", "tail writes an inn option"},
		{"a row literal", "pkg/game/a.go",
			"package game\nfunc rows() []town.OfferRow { return []town.OfferRow{{Topic: 1, NPC: 2}} }\n", "rows writes an inn option or row literal"},
		{"the chapter row statement", "pkg/town/inn.go",
			"package town\nfunc ChapterOffer() OfferRow { return OfferRow{NPC: 1} }\n", ""},
		{"a second tavern row type", "pkg/town/a.go",
			"package town\ntype TavernRow struct{ Town, Chapter, NPC, Mission int }\n", "type TavernRow is a second offer row type"},
		{"a second inn row type", "pkg/mod/a.go",
			"package mod\nfunc f() { type InnRow struct{ Topic int }; _ = InnRow{} }\n", "type InnRow is a second offer row type"},
		{"the builder", "pkg/game/innoffers.go",
			"package game\nfunc innOffers() []innOffer { return []innOffer{{kind: 1}} }\n", ""},
		{"the builder's name in another file", "pkg/game/a.go",
			"package game\nfunc innOffers() []innOffer { return []innOffer{{kind: 1}} }\n", "innOffers writes an inn option"},
		{"an empty value", "pkg/game/a.go",
			"package game\nfunc word() (innOffer, bool) { return innOffer{}, false }\n", ""},
		{"a test", "pkg/game/a_test.go",
			"package game\nvar want = []innOffer{{kind: 3}}\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckInnOffers(map[string]string{tc.file: tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckInnOffers = %v, want none", got)
				}
				return
			}
			if len(got) == 0 || !strings.Contains(strings.Join(got, "\n"), tc.want) {
				t.Fatalf("CheckInnOffers = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

// TestInnOffersHaveOneBuilder is the live-tree half.
func TestInnOffersHaveOneBuilder(t *testing.T) {
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
	if !strings.Contains(files["pkg/game/innoffers.go"], "func innOffers(") {
		t.Fatal("pkg/game/innoffers.go holds no innOffers; the scan guards nothing")
	}
	for _, v := range CheckInnOffers(files) {
		t.Errorf("%s: %s", v.From, v.Reason)
	}
}
