package game

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func roodStageHealSource(t *testing.T) ([]byte, sav.DocumentData) {
	t.Helper()
	path := os.Getenv("AGAINROM_ROOD_SAV")
	if path == "" {
		t.Skip("no AGAINROM_ROOD_SAV: exact Rood SAV is required")
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%X", sha256.Sum256(source)); got != "28E705DA3AE24A6537082C110D3C99A6715E79B2E0F1B005B2EAD8F553CD2DC1" {
		t.Fatalf("Rood source hash=%s", got)
	}
	sourceDoc, err := sav.DecodeDocumentData(source)
	if err != nil {
		t.Fatal(err)
	}
	return source, sourceDoc
}

func TestReleaseZeroHealthStageOneRemainsHealable(t *testing.T) {
	source, sourceDoc := roodStageHealSource(t)
	requireRoodPlacement(t, sourceDoc, 0, 1, false)
	f := loadRoodMission(t, source)
	rood := worldEntityByRuntimeID(t, f, 258)
	if rood.HP != 0 || rood.Decay != sim.DecayFallen {
		t.Fatal("source Rood is not fallen", rood.HP, rood.Decay)
	}
	if rood.ID != 124 || rood.Defence != 26 || rood.Owner != 5 || rood.Group != 16 {
		t.Fatal("source Rood is not the fallen Group actor", rood.ID, rood.Defence, rood.Owner, rood.Group)
	}
	before := rood
	for _, terminal := range f.live.world.CurrentTerminalActors() {
		if terminal.ID == rood.ID {
			t.Fatal("HP-zero Stage-1 Rood became terminal")
		}
	}
	sim.Step(f.live.world, []sim.Command{sim.MoveTo(rood.ID, sim.CellPoint{X: rood.X + 1, Y: rood.Y})})
	rood = worldEntityByRuntimeID(t, f, 258)
	if rood.X != before.X || rood.Y != before.Y || rood.HasTarget || rood.Defence != 26 || rood.Owner != 5 || rood.Group != 16 {
		t.Fatal("fallen Rood accepted a move or changed defence or Group", rood.X, rood.Y, rood.HasTarget, rood.Defence, rood.Owner, rood.Group)
	}
	if err := f.live.world.HeadlessHeal(rood.ID); err != nil {
		t.Fatal(err)
	}
	rood = worldEntityByRuntimeID(t, f, 258)
	if rood.HP != 86 || rood.MaxHP != 86 || rood.Decay != sim.DecayNone || rood.Defence != 52 || rood.Owner != 5 || rood.Group != 16 {
		t.Fatal("ordinary health write did not revive Rood in his Group", rood.HP, rood.MaxHP, rood.Decay, rood.Defence, rood.Owner, rood.Group)
	}
}

func TestReleaseHealedActorSAVUsesCurrentStage(t *testing.T) {
	source, sourceDoc := roodStageHealSource(t)
	_, identity := roodDocumentActor(t, sourceDoc)
	f := loadRoodMission(t, source)
	rood := worldEntityByRuntimeID(t, f, 258)
	if rood.HP != 0 || rood.Decay != sim.DecayFallen {
		t.Fatal("source Rood is not fallen", rood.HP, rood.Decay)
	}
	if rood.ID != 124 || rood.Defence != 26 || rood.Owner != 5 || rood.Group != 16 {
		t.Fatal("source Rood is not the fallen Group actor", rood.ID, rood.Defence, rood.Owner, rood.Group)
	}
	if err := f.live.world.HeadlessHeal(rood.ID); err != nil {
		t.Fatal(err)
	}
	rood = worldEntityByRuntimeID(t, f, 258)
	if rood.HP != 86 || rood.MaxHP != 86 || rood.Decay != sim.DecayNone || rood.Defence != 52 || rood.Owner != 5 || rood.Group != 16 {
		t.Fatal("ordinary health write did not revive Rood in his Group", rood.HP, rood.MaxHP, rood.Decay, rood.Defence, rood.Owner, rood.Group)
	}
	written, err := sav.DecodeDocumentData(saveRoodMission(t, f))
	if err != nil {
		t.Fatal(err)
	}
	object, writtenIdentity := roodDocumentActor(t, written)
	if writtenIdentity != identity || roodComparable(sourceDoc, roodGroupRecord(t, sourceDoc)) != roodComparable(written, roodGroupRecord(t, written)) {
		t.Fatal("SAVE changed Rood's actor identity or Group")
	}
	requireRoodPlacement(t, written, int16(rood.HP), 0, false)
	defence := savedRecordRawForTest(t, written.Objects[object-1], "UBE")
	if len(defence) < 2 || int32(int16(binary.LittleEndian.Uint16(defence))) != rood.Defence {
		t.Fatal("ordinary SAV lost revived defence", defence, rood.Defence)
	}
}
