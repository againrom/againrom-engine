package game

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
	"againrom/pkg/vfs"
)

// previousAssetRecords is the world-map assets' former registry read, the oracle.
func previousAssetRecords(src entrySource) (*globalMapData, string) {
	raw, err := src.ReadFile(globalMapRegistry)
	if err != nil {
		return nil, err.Error()
	}
	parsed, err := reg.Parse(raw)
	if err != nil {
		return nil, fmt.Sprintf("%s: %v", globalMapRegistry, err)
	}
	data, err := ReadGlobalMap(parsed)
	if err != nil {
		return nil, err.Error()
	}
	return data, ""
}

// previousMapObjects is LoadCampaign's former map-object read, the oracle.
func previousMapObjects(src entrySource) map[int]int {
	var out map[int]int
	if raw, err := src.ReadFile(globalMapRegistry); err == nil {
		if parsed, err := reg.Parse(raw); err == nil {
			if data, err := ReadGlobalMap(parsed); err == nil {
				for mission, object := range data.Missions {
					if object >= 0 && object < len(data.Objects) {
						if out == nil {
							out = make(map[int]int)
						}
						out[mission] = object
					}
				}
			}
		}
	}
	return out
}

func worldMapCatalogRegistries() map[string][]byte {
	second := synth.Reg(kindRoot, []synth.RegNode{
		{Name: "General", Kind: kindDir, Children: []synth.RegNode{{Name: "ObjectCount", Kind: kindInt, Int: 2}}},
		{Name: "MapObject1", Kind: kindDir, Children: []synth.RegNode{
			{Name: "MapPoint", Kind: kindIntArray, Ints: []int32{40, 50}},
			{Name: "MapRect", Kind: kindIntArray, Ints: []int32{30, 40, 20, 20}},
		}},
		{Name: "MapObject2", Kind: kindDir, Children: []synth.RegNode{
			{Name: "MapPoint", Kind: kindIntArray, Ints: []int32{7}},
		}},
		{Name: "MissionObjects", Kind: kindDir, Children: []synth.RegNode{
			{Name: "Mission10", Kind: kindInt, Int: 1},
			{Name: "Mission20", Kind: kindInt, Int: 2},
			{Name: "Mission0", Kind: kindInt, Int: 1},
			{Name: "Missionx", Kind: kindInt, Int: 1},
		}},
	})
	negative := synth.Reg(kindRoot, []synth.RegNode{
		{Name: "General", Kind: kindDir, Children: []synth.RegNode{{Name: "ObjectCount", Kind: kindInt, Int: -1}}},
	})
	return map[string][]byte{
		"first": worldMapRegistryFixture(), "second": second, "negative": negative,
		"unparsed": []byte("not a registry"), "absent": nil,
	}
}

func worldMapCatalogFS(t *testing.T, registry []byte) *vfs.FS {
	t.Helper()
	files := []synth.File{{Path: "scenario.reg", Data: townFixture()}}
	if registry != nil {
		files = append(files, synth.File{Path: "globalmap.reg", Data: registry})
	}
	path := filepath.Join(t.TempDir(), ScenarioArchive)
	if err := os.WriteFile(path, synth.Archive(files), 0o600); err != nil {
		t.Fatal(err)
	}
	fsys, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return fsys
}

func TestWorldMapCatalogGivesTodaysRecords(t *testing.T) {
	for name, registry := range worldMapCatalogRegistries() {
		t.Run(name, func(t *testing.T) {
			fsys := worldMapCatalogFS(t, registry)
			wantData, wantProblem := previousAssetRecords(fsys)
			catalog := readWorldMapCatalog(fsys)
			if !reflect.DeepEqual(catalog.data, wantData) || catalog.problem != wantProblem {
				t.Fatalf("catalog = %+v %q, want %+v %q", catalog.data, catalog.problem, wantData, wantProblem)
			}
			c, err := LoadCampaign(fsys)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(c.MapObjects, previousMapObjects(fsys)) {
				t.Fatalf("campaign map objects = %v, want %v", c.MapObjects, previousMapObjects(fsys))
			}
			if !reflect.DeepEqual(c.worldMap, catalog) {
				t.Fatalf("campaign catalog = %+v, want %+v", c.worldMap, catalog)
			}
			archives := &Archives{Containers: fsys}
			own := loadWorldMapAssets(archives, nil)
			taken := loadWorldMapAssets(archives, c.worldMap)
			if !reflect.DeepEqual(own.data, wantData) || own.problem != wantProblem ||
				!reflect.DeepEqual(taken.data, wantData) || taken.problem != wantProblem {
				t.Fatalf("assets records: own %+v %q, taken %+v %q, want %+v %q", own.data, own.problem, taken.data, taken.problem, wantData, wantProblem)
			}
			if own.reads[globalMapRegistry] != 1 || taken.reads[globalMapRegistry] != 0 {
				t.Fatalf("registry reads: own %d, taken %d; want 1 and 0", own.reads[globalMapRegistry], taken.reads[globalMapRegistry])
			}
		})
	}
}
