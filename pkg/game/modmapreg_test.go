package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/sim"
)

func routeValuesMod(t *testing.T, registry []byte, rel string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "route-values")
	b, err := os.ReadFile(filepath.Join("..", "modrt", "testdata", "mods", "route-values", "mod.toml"))
	if err != nil {
		t.Fatal(err)
	}
	writeFileAll(t, filepath.Join(dir, "mod.toml"), b)
	writeFileAll(t, filepath.Join(dir, filepath.FromSlash(rel)), registry)
	return dir
}

func pathFindingReg(refresh int32) []byte {
	return synth.Reg(0, []synth.RegNode{{Name: "Path Finding", Kind: 1, Children: []synth.RegNode{
		{Name: "StaticRefreshRa", Kind: 2, Int: refresh},
	}}})
}

func TestModMapRegistryReachesBothGames(t *testing.T) {
	for _, tc := range []struct {
		install string
		game    base.Game
	}{{"rom1-en", base.GameROM1}, {"rom2-en", base.GameROM2}} {
		dir := routeValuesMod(t, pathFindingReg(77), "files/common/world/data/map.reg")
		res, err := modrt.Load([]mod.Entry{mod.Load(dir)}, tc.install, nil, modrt.Options{})
		if err != nil {
			t.Fatalf("%s: %v", tc.install, err)
		}
		if res.Set.Mods[0].StateFree {
			t.Errorf("%s: a mod carrying map.reg is state-free", tc.install)
		}
		fs := worldFS(t, synth.File{Path: "data/map.reg", Data: pathFindingReg(1)})
		b := &modFiles{fs: fs, order: res.Files.Order, owners: map[string]modFileEntry{}}
		if err := b.serveInstallFiles(&Archives{Containers: fs, mods: b}, res.Files.Files); err != nil {
			t.Fatalf("%s: %v", tc.install, err)
		}
		registry, ok, err := mapRegistry(fs, filesOf(tc.game).planeCosts)
		if err != nil || !ok {
			t.Fatalf("%s: registry %v %v", tc.install, ok, err)
		}
		want := sim.ShippedPathFinding()
		want.StaticRefreshRate = 77
		if registry.Path != want {
			t.Errorf("%s: route values %+v, want %+v", tc.install, registry.Path, want)
		}
	}
}

func TestMapRegistryCarriesSpeedMultiplierAndScanShiftInBothGames(t *testing.T) {
	custom := synth.Reg(0, []synth.RegNode{
		{Name: "Path Finding", Kind: 1, Children: []synth.RegNode{{Name: "SpeedMultiplier", Kind: 2, Int: 4}}},
		{Name: "Scanning", Kind: 1, Children: []synth.RegNode{{Name: "ScanShift", Kind: 2, Int: 6}}},
	})
	for _, g := range []base.Game{base.GameROM1, base.GameROM2} {
		for _, tc := range []struct {
			name        string
			reg         []byte
			speed, shft int32
		}{{"custom", custom, 4, 6}, {"keys missing", pathFindingReg(1), 8, 7}} {
			fs := worldFS(t, synth.File{Path: "data/map.reg", Data: tc.reg})
			registry, ok, err := mapRegistry(fs, filesOf(g).planeCosts)
			if err != nil || !ok {
				t.Fatalf("%v %s: registry %v %v", g, tc.name, ok, err)
			}
			if registry.Path.SpeedMultiplier != tc.speed || registry.Scan.ScanShift != tc.shft {
				t.Errorf("%v %s: %d and %d, want %d and %d", g, tc.name, registry.Path.SpeedMultiplier, registry.Scan.ScanShift, tc.speed, tc.shft)
			}
		}
		fs := worldFS(t)
		if _, ok, err := mapRegistry(fs, filesOf(g).planeCosts); err != nil || ok {
			t.Errorf("%v: a missing file gave %v %v", g, ok, err)
		}
	}
}

func TestModMapRegistryRefusals(t *testing.T) {
	for _, tc := range []struct {
		name  string
		rel   string
		bytes []byte
		want  string
	}{
		{"garbage", "files/common/world/data/map.reg", []byte("not a registry"), "cannot read this registry"},
		{"zero multiplier", "files/common/world/data/map.reg", synth.Reg(0, []synth.RegNode{{Name: "Path Finding", Kind: 1, Children: []synth.RegNode{{Name: "SpeedMultiplier", Kind: 2, Int: 0}}}}), "SpeedMultiplier 0"},
		{"shift past the tables", "files/common/world/data/map.reg", synth.Reg(0, []synth.RegNode{{Name: "Scanning", Kind: 1, Children: []synth.RegNode{{Name: "ScanShift", Kind: 2, Int: 12}}}}), "ScanShift 12"},
		{"elsewhere", "files/common/main/data/map.reg", pathFindingReg(5), "holds no file"},
	} {
		dir := routeValuesMod(t, tc.bytes, tc.rel)
		res, err := modrt.Load([]mod.Entry{mod.Load(dir)}, "rom1-en", nil, modrt.Options{})
		if err == nil {
			fs := worldFS(t, synth.File{Path: "data/map.reg", Data: pathFindingReg(1)})
			b := &modFiles{fs: fs, order: res.Files.Order, owners: map[string]modFileEntry{}}
			err = b.serveInstallFiles(&Archives{Containers: fs, mods: b}, res.Files.Files)
		}
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v, want a refusal naming %q", tc.name, err, tc.want)
		}
	}
}
