package game

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"testing"

	"againrom/pkg/formats/sav"
)

func nativePartyCellCounts(doc sav.DocumentData) (actors, misplaced int) {
	cells := map[uint16]sav.DocumentCellData{}
	flags := map[uint16]byte{}
	for _, b := range doc.World.Blocks {
		flags[b.Cell] = b.Dyn
	}
	for _, c := range doc.World.Cells {
		cells[c.Cell] = c
	}
	for _, player := range doc.Players {
		p := &doc.Objects[player-1]
		slot, _ := savedStructureValue(p, "Slot")
		if slot != 1 {
			continue
		}
		for _, g := range p.Groups {
			for _, refs := range g.RefSlots {
				if refs.Name != "Actors" {
					continue
				}
				for _, index := range refs.Objects {
					r := &doc.Objects[index-1]
					hp, _ := savedStructureValue(r, "Health")
					if int16(hp) <= 0 {
						continue
					}
					position, err := savedMotionRaw(r, "Block12", 12)
					if err != nil {
						continue
					}
					actors++
					key, _ := savedStructureValue(r, "Identity")
					at := binary.LittleEndian.Uint16(position)
					count, here := 0, false
					for _, c := range cells {
						if c.GroundActor == key {
							count++
							here = here || c.Cell == at
						}
					}
					if count != 1 || !here || flags[at]&0x60 != 0x60 {
						misplaced++
					}
				}
			}
		}
	}
	return
}

func TestReleaseNativeActorRegistryCurrentSAV(t *testing.T) {
	path := os.Getenv("AGAINROM_TERMINAL_REGISTRY_SAV")
	if native := os.Getenv("AGAINROM_NATIVE_REGISTRY_SAV"); native != "" {
		path = native
	}
	if path == "" {
		t.Skip("set AGAINROM_TERMINAL_REGISTRY_SAV to the owner spatial-registry source")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sourceSHA := fmt.Sprintf("%x", sha256.Sum256(raw))
	if sourceSHA != "1ee3466b514ed90e69e05ad01eeceb49a20f2712cac0b64c7e88d1c06479b400" && sourceSHA != "9c0a62801d363d4a640c5bda4d34ac2064964713c021282cc6cd846e2160e52e" {
		t.Fatal("spatial-registry source SHA differs")
	}
	source, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if actors, misplaced := nativePartyCellCounts(source); actors != 6 || misplaced != 6 {
		t.Fatalf("source party=%d misplaced=%d, want6/6", actors, misplaced)
	}
	f := releaseFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "native-spatial-source.sav")
	store, name, saved := menuSAVE(t, f, app, OriginalStore{})
	out, err := sav.DecodeDocumentData(saved)
	if err != nil {
		t.Fatal(err)
	}
	if actors, misplaced := nativePartyCellCounts(out); actors != 6 || misplaced != 0 {
		t.Fatalf("saved party=%d misplaced=%d, want6/0", actors, misplaced)
	}
	terminal := requireTerminalWire(t, out)
	if terminal == 0 || sourceSHA == "1ee3466b514ed90e69e05ad01eeceb49a20f2712cac0b64c7e88d1c06479b400" && terminal != 121 {
		t.Fatal("terminal registry changed")
	}
	cold := loadLocalLegacySave(t, store, name)
	for tick := 0; tick < 3; tick++ {
		if f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatalf("spatial registry differs after cold LOAD at tick%d", tick)
		}
		f.live.tick()
		cold.live.tick()
	}
	if path := os.Getenv("AGAINROM_NATIVE_REGISTRY_OUT"); path != "" {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_, writeErr := file.Write(saved)
		closeErr := file.Close()
		if writeErr != nil || closeErr != nil {
			t.Fatal(writeErr, closeErr)
		}
	}
	t.Logf("source=%s party=6 misplaced=6->0 terminal=%d; menu SAVE, cold LOAD and3ticks agree; outputSHA=%x", sourceSHA, terminal, sha256.Sum256(saved))
}
