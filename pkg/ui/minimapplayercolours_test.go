package ui

import "testing"

func TestMinimapAuthoredPlayerColoursKeepTheRosterAcrossArtChanges(t *testing.T) {
	v := originalViewer(t, 32, 32)
	shades := originalShades()
	if got, ok := v.minimapOwnerColour(2); !ok || got != shades[2] {
		t.Fatal("campaign mark lost ordinal colour", got, ok)
	}
	colours := []uint32{9, 7}
	v.SetPlayerColours(colours)
	colours[1] = 12
	if got, ok := v.minimapOwnerColour(2); !ok || got != shades[7] {
		t.Fatal("authored mark has no independent roster colour", got, ok)
	}
	v.SetMinimapPaint(&MinimapPaint{Colours: originalLookup(), OwnerShades: shades})
	if got, ok := v.minimapOwnerColour(2); !ok || got != shades[7] {
		t.Fatal("replacing installed paint lost roster colour", got, ok)
	}
	if got, ok := v.minimapOwnerColour(18); !ok || got != shades[2] {
		t.Fatal("unknown owner lost ordinal fallback", got, ok)
	}
	if _, ok := v.minimapOwnerColour(0); ok {
		t.Fatal("ownerless mark acquired a colour")
	}
	v.SetPlayerColours(nil)
	if got, _ := v.minimapOwnerColour(2); got != shades[2] {
		t.Fatal("clearing roster colours retained the authored mark", got)
	}
}
