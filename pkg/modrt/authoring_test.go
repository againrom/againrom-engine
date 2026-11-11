package modrt

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/mod"
)

func TestAuthoringUsesTheLoaderWithoutActivatingManifestTargets(t *testing.T) {
	entries := makeMods(t, map[string]map[string]string{"project": {
		mod.ManifestName: "id = 'project'\ntitle = 'Project'\nversion = '1'\napplies-to = ['rom2-ru']\nrequires = ['rules>=2']\nconflicts = ['another']\nload-after = ['rules']\n",
		mod.SpellsFile:   "[[spell]]\ntarget = 'Bless'\nmana = 7\n",
	}})
	before, err := os.ReadFile(filepath.Join(entries[0].Dir, mod.ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Load(entries, "rom1-en", nil, Options{}); err == nil {
		t.Fatal("runtime admitted inactive requirements and target")
	}
	result, err := Load(entries, "rom1-en", nil, Options{Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(entries[0].Dir, mod.ManifestName))
	if !bytes.Equal(before, after) || !reflect.DeepEqual(result.Ordered[0].Manifest, entries[0].Manifest) {
		t.Fatal("authoring changed manifest")
	}
	if result.Spells.Empty() {
		t.Fatal("authoring skipped held data")
	}
	if _, err := Load(entries, "rom2-ru", nil, Options{}); err == nil {
		t.Fatal("runtime stopped checking dependencies")
	}
}
