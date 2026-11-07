package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestTownStateHasOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckTownState(files) {
		t.Error(v)
	}
}

func TestTownStateGuardNamesPlantedShapes(t *testing.T) {
	for name, code := range map[string]string{
		"aliased scene read":     "import alias \"againrom/pkg/town\"; func planted() { alias.LoadArt(nil,nil) }",
		"room literal":           "func planted() { _ = townRoomArt{} }",
		"page read":              "func planted() { roomPageArt(nil, room, nil) }",
		"room scene read":        "func planted() { loadRoomSceneArt(nil, room, nil) }",
		"raw scene read":         "func planted() { town.LoadArt(nil,nil) }",
		"room manifest bind":     "func planted() { screenArtOf(nil).Bind(r.Name, r.Art, \"p\", 1) }",
		"room manifest load":     "func planted() { screenArtOf(nil).Load(room, r.Art) }",
		"state literal":          "func planted() { _ = townState{} }",
		"bank literal":           "func planted() { _ = secondCampaign{} }",
		"bundle literal":         "func planted() { _ = townArtLoads{} }",
		"state alias":            "type alias = townState; func planted() { _ = alias{} }",
		"bank defined type":      "type alias secondCampaign; func planted() { _ = alias{} }",
		"bundle alias chain":     "type a = townArtLoads; type b = a; func planted() { _ = b{} }",
		"state new":              "func planted() { _ = new(townState) }",
		"bank new":               "func planted() { _ = new(secondCampaign) }",
		"bundle new":             "func planted() { _ = new(townArtLoads) }",
		"state variable":         "func planted() { var s townState; s.id = 2 }",
		"square read":            "func planted() { LoadTownSquareArtFor(nil, nil) }",
		"tavern read":            "func planted() { LoadTownTavernArt(nil, nil) }",
		"school read":            "func planted() { LoadTownSchoolArt(nil, nil) }",
		"shop read":              "func planted() { loadShopArt(nil, nil) }",
		"read alias":             "func planted() { read := LoadTownSquareArtFor; read(nil, nil) }",
		"global read alias":      "var planted = LoadTownTavernArt",
		"renamed state storage":  "type planted struct { id int; taken map[int]bool; mercPool [16]int }",
		"renamed bank storage":   "type planted struct { bank [1024]int32; shop Shop; available []partyPlace; current partyPlace }",
		"renamed bundle storage": "type planted struct { square, tavern, school, shop any }",
		"anonymous bank storage": "func planted() { _ = struct { bank [1024]int32; shop Shop; available []partyPlace; current partyPlace }{} }",
	} {
		t.Run(name, func(t *testing.T) {
			files := map[string]string{"pkg/game/planted.go": "package game\n" + code}
			vs := CheckTownState(files)
			if len(vs) != 1 || !strings.Contains(vs[0].From, "pkg/game/planted.go") {
				t.Fatalf("planted shape was not named once: %v", vs)
			}
		})
	}
}

func TestTownStateGuardAdmitsBuilders(t *testing.T) {
	files := map[string]string{
		"pkg/game/town.go":    "package game; func newTownState() { _ = townState{}; _ = secondCampaign{} }",
		"pkg/game/townart.go": "package game; func newTownArtLoads() { _ = townArtLoads{}; LoadTownSquareArtFor(nil,nil); LoadTownTavernArt(nil,nil); LoadTownSchoolArt(nil,nil); loadShopArt(nil,nil) }",
	}
	if vs := CheckTownState(files); len(vs) != 0 {
		t.Fatal(vs)
	}
}
