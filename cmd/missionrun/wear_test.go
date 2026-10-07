package main

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestAWearIsREFXY(t *testing.T) {
	got, err := parseWear("p0:66:16")
	if err != nil {
		t.Fatalf("parseWear: %v", err)
	}
	want := wear{ref: waypoint{kind: refParty, index: 0}, x: 66, y: 16}
	if got != want {
		t.Errorf("parseWear = %+v, want %+v", got, want)
	}
}

func TestAMalformedWearIsRefused(t *testing.T) {
	for _, s := range []string{
		"", "p0", "p0:66", "p0:66:16:3",
		"x0:66:16", // neither naming
		"p:66:16",  // no index
		"p-1:66:16", "p66:x:16", "p66:16:x",
	} {
		if _, err := parseWear(s); err == nil {
			t.Errorf("parseWear(%q) was accepted", s)
		}
	}
}

func TestAWearNamingAScriptUnitNamesWhyItIsRefused(t *testing.T) {
	_, err := parseWear("u21:66:16")
	if err == nil {
		t.Fatal("parseWear(a script-unit reference) was accepted")
	}
	if !strings.Contains(err.Error(), "party member") {
		t.Errorf("parseWear(u21:66:16) = %v, want an error naming the party-member requirement", err)
	}
}

func TestFormatItemNamesAnArmourWhenNoWeaponResolves(t *testing.T) {
	shapes := fakeScaleTable{names: []string{""}}
	materials := fakeScaleTable{names: []string{""}}
	armors := fakeCollection{
		names: []string{"", "Boots"},
		// Row 1's 11 cells: param 4 the Slot, param 9 the raw defence, param
		// 10 the raw absorption — wear.go's own armorSlotColumn,
		// armorDefenceColumn, armorAbsorptionColumn. shapes/materials above
		// answer EntryDoubles(nil) for every index, which scale()'s own
		// "absent" arm reads as the identity factor 1, so the stored 7 and 3
		// come straight through: defence ftol(7*1+0.5)=7, absorption
		// ftol(3*1)=3.
		params: [][]int32{nil, {-1, -1, -1, -1, 12, -1, -1, -1, -1, 7, 3}},
	}
	table := &mapload.Table{Shapes: shapes, Materials: materials, Armors: armors}

	// Field A=0 (material 0, unnamed), B=5 (a worn class, neither the
	// weapon's 1 nor the shield's 2 — and deliberately not 12 either, so a
	// slot printed off this code's own B() cannot be mistaken for the row's
	// own Slot column, exactly as rearm_test.go's gaBootsCode is built to
	// show for EquipTarget), C=0 (shape 0), D=1 (the Boots row).
	const armorCode = uint16(5)<<8 | uint16(1)
	got := formatItem(armorCode, table)
	if !strings.HasPrefix(got, "0005001(slot 5)") {
		t.Errorf("formatItem(%04x, table) = %q, want it to start with the seven-digit name and slot 5", armorCode, got)
	}
	if !strings.Contains(got, `"Boots"`) {
		t.Errorf("formatItem(%04x, table) = %q, want the resolved armour's own row name", armorCode, got)
	}
	if !strings.Contains(got, "defence 7") || !strings.Contains(got, "absorption 3") {
		t.Errorf("formatItem(%04x, table) = %q, want the armour's defence and absorption", armorCode, got)
	}

	// A table with no Armors collection at all takes the "tables named
	// nothing" arm, on formatItem's own doc, exactly as an unresolved weapon
	// code already does in main_test.go.
	noArmors := &mapload.Table{Shapes: shapes, Materials: materials}
	if got := formatItem(armorCode, noArmors); strings.Contains(got, `"`) {
		t.Errorf("formatItem(%04x, no Armors) = %q, want no name at all", armorCode, got)
	}
}

func TestDriveTakesWearsAndReDerivesFromASyntheticWorld(t *testing.T) {
	shapes := fakeScaleTable{names: []string{""}}
	materials := fakeScaleTable{names: []string{""}}
	// EquipTarget refuses outright unless all FOUR collections are present
	// (rearm.go's own doc) — so this fixture carries an otherwise-empty
	// Weapons collection too, on rearm_test.go's gaTable precedent, even
	// though nothing here is meant to resolve as a weapon.
	weapons := fakeCollection{names: []string{""}, params: [][]int32{nil}}
	armors := fakeCollection{
		names:  []string{"", "Boots"},
		params: [][]int32{nil, {-1, -1, -1, -1, 12, -1, -1, -1, -1, 7, 3}},
	}
	table := &mapload.Table{Shapes: shapes, Materials: materials, Weapons: weapons, Armors: armors}
	const bootsCode = uint16(5)<<8 | uint16(1)

	// The map carries five script units and no others, so party slot 0's
	// entity id is mapload.PartyEntity's own arithmetic: len(Units) + 0.
	m := &alm.Map{Units: make([]alm.Unit, 5)}
	const partyID = sim.EntityID(5)

	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: partyID, X: 2, Y: 2, HP: 10, MaxHP: 10}}, nil, sim.Relations{},
		[]sim.Sack{{X: 2, Y: 2, Items: []uint16{bootsCode}}}, nil)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}

	hero := data.Hero{Reaction: 5}
	ms := &game.Mission{
		Map:   m,
		World: w,
		Party: []mapload.PartyMember{{Hero: hero}},
	}

	wr, err := parseWear("p0:2:2")
	if err != nil {
		t.Fatalf("parseWear: %v", err)
	}

	var buf bytes.Buffer
	// The tail is the flag's own default (0135-skill-moves T4): this drive
	// has no waypoint and no blow to outlive, so it changes nothing here.
	if err := drive(ms, table, nil, nil, nil, []wear{wr}, nil, 100, 64, false, false, &buf); err != nil {
		t.Fatalf("drive: %v", err)
	}
	out := buf.String()
	t.Log("\n" + out)

	if !strings.Contains(out, `"Boots"`) {
		t.Errorf("drive output does not name the boots:\n%s", out)
	}
	if !strings.Contains(out, "slot 12") {
		t.Errorf("drive output does not show the boots landing in slot 12 (its row's own, not its code's field B):\n%s", out)
	}
	if strings.Contains(out, "refused") {
		t.Errorf("drive output refused a code the gate should have accepted:\n%s", out)
	}

	bare := hero.Recompute(data.Profile{}, data.Loadout{}).Combat
	wantDefence := bare.Defence + 7
	wantAbsorption := bare.Absorption + 3
	wantAfter := "after: defence " + strconv.Itoa(int(wantDefence)) + " absorption " + strconv.Itoa(int(wantAbsorption))
	if !strings.Contains(out, wantAfter) {
		t.Errorf("drive output's after-line does not show defence %d absorption %d (bare %d/%d + the boots' 7/3):\n%s",
			wantDefence, wantAbsorption, bare.Defence, bare.Absorption, out)
	}

	final := entityAt(w, partyID)
	if final.Defence != wantDefence {
		t.Errorf("entity Defence = %d, want %d", final.Defence, wantDefence)
	}
	if final.Absorption != wantAbsorption {
		t.Errorf("entity Absorption = %d, want %d", final.Absorption, wantAbsorption)
	}
	if codes, _ := w.Carried(partyID); len(codes) != 0 {
		t.Errorf("Carried(partyID) = %v after the wear, want empty — the code moved into equipment", codes)
	}
	slots, _ := w.Equipped(partyID)
	if slots[11] != bootsCode {
		t.Errorf("Equipped(partyID)[11] (slot 12) = %#x, want %#x (the boots)", slots[11], bootsCode)
	}
}
