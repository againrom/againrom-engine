package game

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/rules"
)

// modTownStateHash is the loaded town state a LOAD leaves: the party, the
// city object graph and the purse.
func modTownStateHash(t *testing.T, f *FrontEnd) string {
	t.Helper()
	b, err := json.Marshal(struct {
		Party any
		Graph any
		Gold  int
	}{f.Carried, f.Town.cityObjects, f.Town.Gold()})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// modTownStateReload saves f and loads the file in a new front end over the
// same install.
func modTownStateReload(t *testing.T, f *FrontEnd) *FrontEnd {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(store.Dir, name))
	if err != nil {
		t.Fatal(err)
	}
	next := &FrontEnd{InstallResources: f.InstallResources, RuntimeServices: f.RuntimeServices}
	if _, town, err := next.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("LOAD", town, err)
	}
	return next
}

// modTownStateFronts are the fixture town without mods and under one mod, the
// hero holding one bought item in each.
func modTownStateFronts(t *testing.T) (plain, modded *FrontEnd) {
	t.Helper()
	plain = currentTrainingCity(t)
	buyTrainingItem(t, plain)
	modded = currentTrainingCity(t)
	modded.Table.Mods.Items = []mapload.ModItem{{Mod: "skill-cap", Key: "x", Code: 0x0e01, StandInCode: 0x0e02}}
	if err := modded.SetMods(rules.Default(), modTestSet(150), false); err != nil {
		t.Fatal(err)
	}
	buyTrainingItem(t, modded)
	return plain, modded
}

// The next SAVE after a town LOAD under no mod is the bytes the engine wrote
// before marks named their items. Under an unchanged mod it differs from them
// only by "mod":"skill-cap","key":"x" on the one stamped item: without the
// two its bytes are that engine's. The hashes were taken from that engine.
func TestAnUnchangedModTownSaveAddsOnlyTheItemKeys(t *testing.T) {
	plain, modded := modTownStateFronts(t)
	sum := func(b []byte) string {
		s := sha256.Sum256(b)
		return hex.EncodeToString(s[:])
	}
	if got := sum(modTownSave(t, modTownStateReload(t, plain))); got != modTownSaveNoMod {
		t.Errorf("no mod: next SAVE %s, want %s", got, modTownSaveNoMod)
	}
	raw := modTownSave(t, modTownStateReload(t, modded))
	mark := modTownMark(t, raw)
	if len(mark.Items) != 1 || mark.Items[0].Mod != "skill-cap" || mark.Items[0].Key != "x" || len(mark.Layers) != 0 {
		t.Fatalf("the mark stamps %+v %+v", mark.Items, mark.Layers)
	}
	keyless := modTownKeyless(t, raw)
	if got := sum(keyless); got != modTownSaveUnchanged {
		t.Errorf("unchanged mod: next SAVE without the keys %s, want %s", got, modTownSaveUnchanged)
	}
	if extra := len(raw) - len(keyless); extra != len(`,"mod":"skill-cap","key":"x"`) {
		t.Errorf("the keys add %d bytes", extra)
	}
}

// Under an unchanged mod and under no mod, a town LOAD leaves the state the
// engine left before changed mods were admitted; the hashes were taken from
// that engine on the same fixture.
func TestAnUnchangedModTownLoadLeavesTheSameState(t *testing.T) {
	plain, modded := modTownStateFronts(t)
	for _, c := range []struct {
		name string
		f    *FrontEnd
		want string
	}{
		{"no mod", plain, modTownStateNoMod},
		{"unchanged mod", modded, modTownStateUnchanged},
	} {
		if got := modTownStateHash(t, modTownStateReload(t, c.f)); got != c.want {
			t.Errorf("%s: loaded state %s, want %s", c.name, got, c.want)
		}
	}
}

const (
	modTownStateNoMod     = "9e7d8e45294a1e0eb1173ea9b226b5accb2c196701e8edb3437bb340859e9640"
	modTownStateUnchanged = "9e7d8e45294a1e0eb1173ea9b226b5accb2c196701e8edb3437bb340859e9640"
	modTownSaveNoMod      = "7e2c208ef2c482d9b594e3ab98c0de6ffcbeeed39c2f69f8f74c2e9ae26e5018"
	modTownSaveUnchanged  = "cbaa0889966244cd3cfdb8b41fd483440cbd845862661d5e5ce25026285665ea"
)
