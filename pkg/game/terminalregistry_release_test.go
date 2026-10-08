package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"

	"againrom/pkg/formats/sav"
)

func terminalGroundKeys(doc sav.DocumentData) map[uint32]bool {
	keys := map[uint32]bool{}
	for _, object := range doc.DeadActors {
		r := &doc.Objects[object-1]
		stage, _ := savedStructureValue(r, "Stage")
		domain, _ := savedStructureValue(r, "U4A")
		if stage >= 2 && domain != 3 {
			key, _ := savedStructureValue(r, "Identity")
			keys[key] = true
		}
	}
	return keys
}

func registeredTerminalGround(doc sav.DocumentData, keys map[uint32]bool) int {
	last := map[uint16]sav.DocumentCellData{}
	for _, cell := range doc.World.Cells {
		last[cell.Cell] = cell
	}
	found := map[uint32]bool{}
	for _, cell := range last {
		if keys[cell.GroundActor] {
			found[cell.GroundActor] = true
		}
	}
	return len(found)
}

func TestReleaseTerminalRegistryCurrentSAV(t *testing.T) {
	path := os.Getenv("AGAINROM_TERMINAL_REGISTRY_SAV")
	if path == "" {
		t.Skip("set AGAINROM_TERMINAL_REGISTRY_SAV to the owner terminal-registry source")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const sourceSHA = "1ee3466b514ed90e69e05ad01eeceb49a20f2712cac0b64c7e88d1c06479b400"
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != sourceSHA {
		t.Fatal("terminal-registry source SHA differs")
	}
	source, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	keys := terminalGroundKeys(source)
	if len(keys) != 121 || registeredTerminalGround(source, keys) != 75 {
		t.Fatal("source no longer carries the observed 121 ground bones and 75 registered bones")
	}
	f := releaseFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "terminal-registry-source.sav")
	_, cells, _, _ := f.live.world.SavedActorMotions()
	for _, cell := range cells {
		if keys[cell.Ground.Key] {
			t.Fatalf("loaded current state retains terminal identity %#x at %04x", cell.Ground.Key, cell.Cell)
		}
	}
	store, name, saved := menuSAVE(t, f, app, OriginalStore{})
	out, err := sav.DecodeDocumentData(saved)
	if err != nil {
		t.Fatal(err)
	}
	if requireTerminalWire(t, out) != 121 || registeredTerminalGround(out, keys) != 0 {
		t.Fatal("ordinary save did not release every ground bone")
	}
	for _, r := range source.Objects {
		if r.Class != "Unit" && r.Class != "Human" && r.Class != "Humanoid" {
			continue
		}
		health, _ := savedStructureValue(&r, "Health")
		stage, _ := savedStructureValue(&r, "Stage")
		if int16(health) < -9 || stage > 1 {
			continue
		}
		identity, _ := savedStructureValue(&r, "Identity")
		matched := false
		for _, other := range out.Objects {
			key, _ := savedStructureValue(&other, "Identity")
			if other.Class != r.Class || key != identity {
				continue
			}
			matched = true
			for _, field := range []string{"Health", "Stage", "RuntimeID", "T08"} {
				want, _ := savedStructureValue(&r, field)
				got, _ := savedStructureValue(&other, field)
				if got != want {
					t.Fatalf("nonterminal identity%x %s changed %d -> %d", identity, field, want, got)
				}
			}
		}
		if !matched {
			t.Fatalf("nonterminal identity%x disappeared", identity)
		}
	}
	cold := loadLocalLegacySave(t, store, name)
	for tick := 0; tick < 3; tick++ {
		if f.live.world.Hash() != cold.live.world.Hash() {
			logCurrentCarrierDiff(t, f.live.world, cold.live.world)
			t.Fatalf("corrected current state differs after cold LOAD at tick%d", tick)
		}
		f.live.tick()
		cold.live.tick()
	}
	if output := os.Getenv("AGAINROM_TERMINAL_REGISTRY_OUT"); output != "" {
		file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_, writeErr := file.Write(saved)
		closeErr := file.Close()
		if writeErr != nil || closeErr != nil {
			t.Fatal(writeErr, closeErr)
		}
	}
	t.Logf("source=%s ground-bones=121 registered=75->0; ordinary menu SAVE, cold LOAD and 3 ticks agree; output-sha=%x", sourceSHA, sha256.Sum256(saved))
}
