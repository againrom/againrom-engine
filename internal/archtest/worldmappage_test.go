package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestWorldMapPageGuardPlantsEachShape(t *testing.T) {
	methods := func(typ string) string {
		var s strings.Builder
		for _, m := range worldMapPageMethods {
			s.WriteString("func (x *" + typ + ") " + m + "() {}\n")
		}
		return s.String()
	}
	cases := []struct{ name, file, source, want string }{
		{"page", "other.go", "type other struct{}\n" + methods("other"), "second world-map page: other"},
		{"embedded page", "other.go", "type other struct{ *worldMapPage }\n" + methods("worldMapPage"), "second world-map page: other"},
		{"alias page", "other.go", "type other = worldMapPage\n" + methods("worldMapPage"), "second world-map page: other"},
		{"interface", "other.go", "type page interface{ WorldMapView() }\ntype other struct{}\nfunc (*other) WorldMapView(){}", "second implementation"},
		{"embedded interface", "other.go", "type view interface{ WorldMapView() }\ntype page interface{view}\ntype other struct{}\nfunc (*other) WorldMapView(){}", "interface page"},
		{"point", "other.go", "func f(){ townMapPoint() }", "townMapPoint"},
		{"point alias", "other.go", "func f(){ point := townMapPoint; _ = point }", "townMapPoint"},
		{"record", "other.go", "func f(){ _ = ui.WorldMapMission{} }", "WorldMapMission"},
		{"global record", "other.go", "var record = ui.WorldMapMission{}", "WorldMapMission"},
		{"record type alias", "other.go", "type row = ui.WorldMapMission\nvar record = row{}", "WorldMapMission"},
		{"global state", "other.go", "var state = worldMapState{}", "worldMapState"},
		{"import alias", "other.go", "func f(){ _ = renamed.WorldMapMission{} }", "WorldMapMission"},
		{"state", "other.go", "func f(){ _ = &worldMapState{} }", "worldMapState"},
		{"producer", "other.go", "func f(){ s.missions = append(s.missions, m) }", "second map record producer"},
		{"record assignment", "other.go", "func f(){ s.missions = records }", "second map record assignment"},
		{"constructor", "other.go", "func f(){ _ = &worldMapPage{} }", "constructed outside"},
		{"type switch", "other.go", "func f(v any){ switch v.(type){case *worldMapPage:} }", "type switch"},
		{"game branch", "other.go", "func (p *worldMapPage) atMap(){ if p.host.hostRules().Leave == \"room-map\" {} }", "policy rows"},
		{"identity branch", "other.go", "func (p *worldMapPage) atMap(){ if p.host.in.profile().IsSecond() {} }", "policy rows"},
		{"unnamed location entry", "secondworldmap.go", "var worldMapLocationEntries = map[string]any{\"another\": f}", "unnamed campaign location entry another"},
		{"changed location entry", "secondworldmap.go", "var worldMapLocationEntries = map[string]any{\"location\": f}", "unnamed campaign location entry location"},
		{"legal builder", "worldmap.go", "func (b worldMapEntryBuilder) record(){ townMapPoint(); _ = ui.WorldMapMission{} }\nfunc (b worldMapEntryBuilder) open(){ _ = worldMapState{}; s.missions = append(s.missions, m) }", ""},
		{"renamed builder receiver", "worldmap.go", "func (b other) record(){ townMapPoint() }", "townMapPoint"},
		{"test ignored", "other_test.go", "func f(){ townMapPoint(); _ = ui.WorldMapMission{} }", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file := "pkg/game/" + tc.file
			vs := CheckWorldMapPage(map[string]string{file: "package game\n" + tc.source})
			if tc.want == "" {
				if len(vs) != 0 {
					t.Fatalf("unexpected violations: %v", vs)
				}
				return
			}
			for _, v := range vs {
				if strings.Contains(v.Reason, tc.want) && strings.HasPrefix(v.From, file+":") {
					return
				}
			}
			t.Fatalf("wanted %q at %s: %v", tc.want, file, vs)
		})
	}
	if len(CheckWorldMapPage(nil)) != 1 {
		t.Fatal("empty scan accepted")
	}
}

func TestWorldMapPageHasOneBuilder(t *testing.T) {
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
	for _, v := range CheckWorldMapPage(files) {
		t.Error(v)
	}
}
