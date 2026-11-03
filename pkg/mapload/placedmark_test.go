package mapload_test

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

func placedMarkTable(keys mapload.UnitKeys) *mapload.Table {
	row := wideUnitRow(2000)
	row[slotUnitType], row[slotUnitFace], row[slotHealthMax] = 0x40, 1, 50
	return &mapload.Table{Units: defCollection{{}, {name: "u", params: row}}, Humans: defCollection{}, UnitKeys: keys}
}

func placedMarkMap(rows ...uint32) *alm.Map {
	m := &alm.Map{Width: 40, Height: 40}
	for i, bits := range rows {
		// The second game's row dword +0x10 holds the placement bits (1, 2, 4, 8).
		m.Units = append(m.Units, alm.Unit{X: 0x0C80 + uint32(i)*0x100, Y: 0x0C80, ServerID: 2000, DefID: bits})
	}
	return m
}

func TestSecondGamePlacedUnitCarriesRowBitEightAsTheRewardMark(t *testing.T) {
	m := placedMarkMap(8, 0, 1|2|4, 8|1)
	w, err := mapload.FromALMWith(m, placedMarkTable(mapload.ServerUnitKeys), mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []bool{true, false, false, true} {
		if got := w.Entities()[i].SuppressRewards; got != want {
			t.Errorf("row %d (bits %#x): mark %t, want %t", i, m.Units[i].DefID, got, want)
		}
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	cold, err := mapload.FromALMWith(placedMarkMap(0, 0, 0, 0), placedMarkTable(mapload.ServerUnitKeys), mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	for i, want := range []bool{true, false, false, true} {
		if got := cold.Entities()[i].SuppressRewards; got != want {
			t.Errorf("after a round trip row %d: mark %t, want %t", i, got, want)
		}
	}
	if cold.Hash() != w.Hash() {
		t.Error("the round trip changed the World hash")
	}
}

func TestFirstGamePlacementNeverCarriesTheRewardMark(t *testing.T) {
	// The first game's dword +0x10 is a definition id; 8 is an id there.
	tbl := placedMarkTable(mapload.ClassUnitKeys)
	m := &alm.Map{Width: 40, Height: 40, Units: []alm.Unit{{X: 0x0C80, Y: 0x0C80, ClassID: 0x40, ClassSubID: 1, DefID: 8}}}
	w, err := mapload.FromALMWith(m, tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range w.Entities() {
		if e.SuppressRewards {
			t.Fatal("a first-game placement carries the reward mark")
		}
	}
}
