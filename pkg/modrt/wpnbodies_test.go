package modrt

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/mod"
)

func weaponNameMod(file string) map[string]map[string]string {
	return map[string]map[string]string{"wb": {
		"main.star": "def init(game, settings):\n    game.data.add(\"" + file + "\")\n",
		file:        weaponSrc,
	}}
}

// A mod with data/wpnbodies.toml packs, and its packed file gives the game the
// weapon bodies its folder gives. The long name still reads from a folder and
// is refused in a pack.
func TestShortWeaponBodiesNamePacksAndReadsAsItsFolder(t *testing.T) {
	folder := makeMods(t, weaponNameMod(mod.WeaponBodiesFile))
	_, b, err := mod.Pack(folder[0].Dir)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, mod.PackedName("wb")), b, 0o644); err != nil {
		t.Fatal(err)
	}
	packed, err := mod.Resolve(dir, []string{"wb"})
	if err != nil {
		t.Fatal(err)
	}
	want, err := Load(folder, "rom1-en", nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Load(packed, "rom1-en", nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Bodies.Weapons) != 1 || got.Bodies.Weapons[0].File != mod.WeaponBodiesFile || got.Bodies.Weapons[0].Body != "hammer2h" {
		t.Fatalf("packed rows: %+v", got.Bodies.Weapons)
	}
	if !reflect.DeepEqual(want.Bodies.Weapons, got.Bodies.Weapons) {
		t.Fatalf("folder %+v, packed %+v", want.Bodies.Weapons, got.Bodies.Weapons)
	}

	long := makeMods(t, weaponNameMod(mod.WeaponBodiesLongFile))
	res, err := Load(long, "rom1-en", nil, Options{})
	if err != nil || len(res.Bodies.Weapons) != 1 || res.Bodies.Weapons[0].File != mod.WeaponBodiesLongFile {
		t.Fatalf("the long name in a folder: %v %+v", err, res.Bodies.Weapons)
	}
	if _, _, err := mod.Pack(long[0].Dir); err == nil {
		t.Fatal("the long name was packed")
	}
}
