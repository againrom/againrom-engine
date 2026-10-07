package mapload

import (
	"testing"

	"againrom/pkg/data"
)

type ghostResistanceEntry struct {
	name    string
	params  []int32
	strings []string
}

type ghostResistanceCollection []ghostResistanceEntry

func (c ghostResistanceCollection) Len() int                    { return len(c) }
func (c ghostResistanceCollection) EntryName(i int) string      { return c[i].name }
func (c ghostResistanceCollection) EntryParams(i int) []int32   { return c[i].params }
func (c ghostResistanceCollection) EntryStrings(i int) []string { return c[i].strings }

type ghostResistanceScale struct{}

func (ghostResistanceScale) Len() int             { return 1 }
func (ghostResistanceScale) EntryName(int) string { return "" }
func (ghostResistanceScale) EntryDoubles(int) []float64 {
	return []float64{1, 1, 1, 1, 1, 1, 1, 1, 1}
}

// TestGhostTemplateCarriesResistanceAndItsResolvedWeaponKind covers the one
// actor producer that does not pass through definitionFor. The synthetic Ghost
// row carries all five bytes and a long-reach melee kind 5 weapon; reach does
// not turn that weapon into the ranged arm, so the raised actor consults the
// Shooting byte.
func TestGhostTemplateCarriesResistanceAndItsResolvedWeaponKind(t *testing.T) {
	unit := make([]int32, 38)
	for i := range unit {
		unit[i] = -1
	}
	unit[4] = 30                // health maximum
	unit[24], unit[25] = 11, 22 // Blade, Axe
	unit[26], unit[27], unit[28] = 33, 44, 255
	unit[29], unit[30] = 61, 0 // type and face keys
	unit[34], unit[35] = 27, 9

	weapon := make([]int32, 14)
	for i := range weapon {
		weapon[i] = -1
	}
	weapon[5] = data.SkillShoot // supported melee kind despite long reach
	weapon[6], weapon[7] = 1, 1
	weapon[8], weapon[9] = 0, 0
	weapon[11], weapon[12], weapon[13] = 8, 1, 1

	units := ghostResistanceCollection{
		{},
		{name: ghostRowName, params: unit, strings: []string{"LongMelee"}},
	}
	weapons := ghostResistanceCollection{{}, {name: "LongMelee", params: weapon}}
	tbl := &Table{Units: units, Shapes: ghostResistanceScale{}, Materials: ghostResistanceScale{}, Weapons: weapons}

	got := ghostTemplate(tbl, DifficultyNormal)
	if got.Resistance != ([5]uint8{11, 22, 33, 44, 255}) {
		t.Errorf("Resistance = %v, want the Ghost row's five narrowed bytes", got.Resistance)
	}
	if got.XPSlot != uint8(data.SkillShoot) {
		t.Errorf("XPSlot = %d, want the equipped weapon kind %d", got.XPSlot, data.SkillShoot)
	}
	if got.Withdraw != 27 || got.Wimpy != 9 {
		t.Errorf("Withdraw/Wimpy = %d/%d, want the Ghost row's 27/9", got.Withdraw, got.Wimpy)
	}
	// This story resolves only the active kind for the dedicated Ghost path;
	// its pre-existing row-combat reach remains unchanged.
	if got.Reach != 1 {
		t.Errorf("Reach = %d, want the Ghost row's unchanged default 1", got.Reach)
	}
}
