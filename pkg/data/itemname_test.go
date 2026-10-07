package data

import "testing"

func TestItemNamesNameForFindsAStoredCode(t *testing.T) {
	n := ItemNames{ItemCode(0x0101): "Sword"}
	got, ok := n.NameFor(ItemCode(0x0101))
	if !ok || got != "Sword" {
		t.Errorf("NameFor(0x0101) = (%q, %v), want (%q, true)", got, ok, "Sword")
	}
}

func TestItemNamesNameForMissesAnUnstoredCode(t *testing.T) {
	n := ItemNames{ItemCode(0x0101): "Sword"}
	if _, ok := n.NameFor(ItemCode(0x0202)); ok {
		t.Errorf("NameFor(0x0202) reported ok, want false: 0x0202 was never stored")
	}
}

func TestItemNamesNameForOnANilMapAnswersFalse(t *testing.T) {
	var n ItemNames
	if got, ok := n.NameFor(ItemCode(0x0101)); ok || got != "" {
		t.Errorf("NameFor on a nil ItemNames = (%q, %v), want (\"\", false)", got, ok)
	}
}
