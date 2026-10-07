package game_test

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/game"
)

// The class name's first hop: the registry's own bytes onto the render
// tier's bundle entry (AC-1).

const (
	nameASCII    = 41
	nameHighByte = 42
	nameAbsent   = 43
)

// nameHighBytes is a name every byte of which is at or above 0x80, written as
// escapes and never as literal text. What it is FOR is that a loader applying
// any character encoding would move these bytes; carried verbatim, they arrive
// as they left.
var nameHighBytes = "\xca\xe0\xe2\xe0\xeb\xe5\xf0\xe8\xff"

// nameRegistry is a units.reg of three classes: one ASCII name, one whose name
// is entirely high bytes, and one carrying no DescText key at all.
//
// NO SHEET IS SHIPPED for any of them, deliberately. Every class loads
// frameless, so nothing here can pass by way of the art, and the name is shown
// to ride independently of whether the class draws.
func nameRegistry() []byte {
	i := func(name string, v int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x02, Int: v}
	}
	s := func(name, v string) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x00, Str: v}
	}
	return synth.UnitsReg([]string{"units/absent/none"},
		[]synth.RegNode{
			i("ID", nameASCII), i("File", 0), s("DescText", "Warrior"),
			i("Width", 16), i("Height", 16), i("CenterX", 8), i("CenterY", 14),
		},
		[]synth.RegNode{
			i("ID", nameHighByte), i("File", 0), s("DescText", nameHighBytes),
			i("Width", 16), i("Height", 16), i("CenterX", 8), i("CenterY", 14),
		},
		// No DescText key on the chain and no Parent to inherit one from.
		[]synth.RegNode{
			i("ID", nameAbsent), i("File", 0),
			i("Width", 16), i("Height", 16), i("CenterX", 8), i("CenterY", 14),
		},
	)
}

func TestLoadUnitsCarriesTheClassName(t *testing.T) {
	set, err := game.LoadUnits(openContainers(t, synth.Archive([]synth.File{
		{Path: graphicsEntry(t, game.UnitRegistry), Data: nameRegistry()},
	})))
	if err != nil {
		t.Fatalf("LoadUnits: %v", err)
	}

	for _, tc := range []struct {
		id   int32
		want string
		why  string
	}{
		{nameASCII, "Warrior", "an ASCII name is its own bytes"},
		{nameHighByte, nameHighBytes, "every byte at or above 0x80 survives unconverted"},
		{nameAbsent, "", "a class carrying no DescText carries the empty name"},
	} {
		c := set.Classes[tc.id]
		if c == nil {
			t.Fatalf("Classes[%d] is nil", tc.id)
		}
		if c.Name != tc.want {
			t.Errorf("Classes[%d].Name = %q, want %q — %s", tc.id, c.Name, tc.want, tc.why)
		}
		// The bundle entry is frameless, so the name above cannot have
		// arrived by way of a decoded sheet.
		if len(c.Frames) != 0 {
			t.Errorf("Classes[%d] resolved %d frame(s); the fixture ships no sheet", tc.id, len(c.Frames))
		}
	}

	// Byte for byte, not merely equal as text: a conversion that happened to
	// round-trip the string's length would still move its bytes.
	got := []byte(set.Classes[nameHighByte].Name)
	if len(got) != len(nameHighBytes) {
		t.Fatalf("the high-byte name is %d byte(s), want %d — the loader converted it",
			len(got), len(nameHighBytes))
	}
	for i := range got {
		if got[i] != nameHighBytes[i] {
			t.Fatalf("byte %d is %#02x, want %#02x", i, got[i], nameHighBytes[i])
		}
	}
}
