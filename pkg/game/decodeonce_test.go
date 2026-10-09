package game

import (
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/render/terrain"
)

// addressCounter counts the reads of each address.
type addressCounter struct {
	src   terrain.EntrySource
	reads map[string]int
}

func (c *addressCounter) ReadFile(name string) ([]byte, error) {
	c.reads[name]++
	return c.src.ReadFile(name)
}

// The attack pointer and the registry's attack slot are one decoded sheet: the
// sheet is read once and the pointer is the slot's own frame 0.
func TestCursorArtReadsTheAttackSheetOnce(t *testing.T) {
	if graphicsPrefix+attackRegistration.path != AttackCursorPath || attackRegistration.name != "attack" {
		t.Fatalf("attack slot %d is %q at %q, not the attack pointer's sheet", attackSlot, attackRegistration.name, attackRegistration.path)
	}
	src := &addressCounter{src: regGraphicsContainers(t, regAllSlotsArchive(t)), reads: map[string]int{}}
	art := loadCursorArt(src)
	if art.pointerErr != nil || art.registryErr != nil {
		t.Fatalf("pointer %v, registry %v", art.pointerErr, art.registryErr)
	}
	if n := src.reads[AttackCursorPath]; n != 1 {
		t.Fatalf("the attack sheet was read %d times, want 1", n)
	}
	slot, ok := art.registry.Slot("attack")
	if !ok || slot.Frames[0] != art.pointer {
		t.Fatal("the pointer is not the registry slot's own frame 0")
	}
}

// A missing slot fails the registry and leaves the pointer, which needs only
// its own sheet.
func TestCursorArtKeepsThePointerWhenAnotherSlotIsMissing(t *testing.T) {
	files := regAllSlotsArchive(t)[attackSlot : attackSlot+1]
	art := loadCursorArt(regGraphicsContainers(t, files))
	if art.pointer == nil || art.pointerErr != nil {
		t.Fatalf("pointer %v, %v", art.pointer, art.pointerErr)
	}
	if art.registry != nil || art.registryErr == nil {
		t.Fatal("a registry with 27 missing slots assembled")
	}
}

// The unit set and the sound table come from one read of units.reg.
func TestUnitRegistryIsParsedOnce(t *testing.T) {
	unitsReg := synth.UnitsReg(nil,
		[]synth.RegNode{{Name: "ID", Kind: 0x02, Int: 1}, {Name: "Sound", Kind: 0x06, Ints: []int32{4, 0, 5}}, {Name: "AttackDelay", Kind: 0x02, Int: 7}},
		[]synth.RegNode{{Name: "ID", Kind: 0x02, Int: 2}, {Name: "AttackDelay", Kind: 0x02, Int: 0}},
	)
	files := []synth.File{{Path: UnitRegistry[len(graphicsPrefix):], Data: unitsReg}}
	src := &addressCounter{src: regGraphicsContainers(t, files), reads: map[string]int{}}
	set, sounds, err := loadUnitRegistry(src)
	if err != nil {
		t.Fatal(err)
	}
	if n := src.reads[UnitRegistry]; n != 1 {
		t.Fatalf("units.reg was read %d times, want 1", n)
	}
	if len(set.Classes) != 2 || len(sounds) != 2 {
		t.Fatalf("%d classes, %d sound rows", len(set.Classes), len(sounds))
	}
	if s := sounds[1]; !slices.Equal(s.Slots, []int32{4, 0, 5}) || s.AttackDelay != 7 {
		t.Fatalf("class 1 sound = %+v", s)
	}
	if _, _, err := loadUnitRegistry(nil); err == nil {
		t.Fatal("a nil source answered a registry")
	}
}
