package game

import (
	"image"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/ui"
)

// A generated mage holds the staff his or her sex names, and the doll the
// generator draws is that staff's own sheet: the woman's plain staff and the
// man's Uncommon staff are different items with different pictures
// (MAGIC-STAFF-022).
func TestReleaseChargenMageStaffFollowsSex(t *testing.T) {
	f := releaseFront(t)
	const plainStaff, uncommonStaff = 0x810d, 0x812d

	dollOf := func(t *testing.T, res ui.ChargenResult, staff uint16) *image.RGBA {
		t.Helper()
		party := f.ChargenParty(res)
		if len(party) != 1 {
			t.Fatalf("party of %d, want 1", len(party))
		}
		worn := party[0].Worn
		worn[0] = staff
		doll, _ := composeUnitFigure(f.Archives.Containers, equipmentFromSlots(worn), memberFigureID(party[0]))
		if doll == nil {
			t.Fatal("no doll composed")
		}
		return doll
	}
	same := func(a, b image.Image) bool {
		ar, br := a.Bounds(), b.Bounds()
		if ar != br {
			return false
		}
		for y := ar.Min.Y; y < ar.Max.Y; y++ {
			for x := ar.Min.X; x < ar.Max.X; x++ {
				if a.At(x, y) != b.At(x, y) {
					return false
				}
			}
		}
		return true
	}

	for _, c := range []struct {
		name         string
		sex          int
		want, other  uint16
		wantDir      data.FigureDir
		wantSheetEnd string
	}{
		{"male", 0, uncommonStaff, plainStaff, data.FigureDirManMage, "0801113"},
		{"female", 1, plainStaff, uncommonStaff, data.FigureDirWomanMage, "0801013"},
	} {
		t.Run(c.name, func(t *testing.T) {
			for skill := 0; skill < 5; skill++ {
				res := ui.ChargenResult{Name: "Staff", Choices: []int{c.sex, 1, skill}, Stats: []int{25, 25, 25, 25}}
				party := f.ChargenParty(res)
				if len(party) != 1 || party[0].Worn[0] != c.want {
					t.Fatalf("skill %d: worn weapon = %+v, want %#04x", skill, party, c.want)
				}
				if data.FigureDir(party[0].FigureDir) != c.wantDir {
					t.Fatalf("figure dir %q, want %q", party[0].FigureDir, c.wantDir)
				}
			}
			res := ui.ChargenResult{Name: "Staff", Choices: []int{c.sex, 1, 0}, Stats: []int{25, 25, 25, 25}}
			got := f.ChargenPreview(res).Doll
			if got == nil {
				t.Fatal("generator preview has no doll")
			}
			if !same(got, dollOf(t, res, c.want)) {
				t.Errorf("generator doll is not the %s staff's own picture", c.wantSheetEnd)
			}
			// Loss control: the other sex's staff draws a different doll, so an
			// item that did not reach the picture would be seen here.
			if same(got, dollOf(t, res, c.other)) {
				t.Errorf("generator doll equals the other sex's staff picture: the staff does not reach the doll")
			}
		})
	}

	// The two staffs are the same weapon row and spell, and only the item differs.
	m := f.ChargenParty(ui.ChargenResult{Name: "A", Choices: []int{0, 1, 0}, Stats: []int{25, 25, 25, 25}})[0]
	w := f.ChargenParty(ui.ChargenResult{Name: "A", Choices: []int{1, 1, 0}, Stats: []int{25, 25, 25, 25}})[0]
	if m.Weapon == nil || w.Weapon == nil || m.Weapon.Row != w.Weapon.Row ||
		m.Weapon.SpellName != w.Weapon.SpellName || m.Weapon.SpellPower != w.Weapon.SpellPower {
		t.Errorf("male staff %+v and female staff %+v differ beyond the item code", m.Weapon, w.Weapon)
	}

	// Fighters and the hero's other worn items do not move with the staff.
	for sex := 0; sex < 2; sex++ {
		for skill := 0; skill < 5; skill++ {
			fighter := f.ChargenParty(ui.ChargenResult{Name: "F", Choices: []int{sex, 0, skill}, Stats: []int{25, 25, 25, 25}})
			if len(fighter) != 1 || fighter[0].Weapon == nil || fighter[0].Weapon.Code == plainStaff || fighter[0].Weapon.Code == uncommonStaff {
				t.Errorf("fighter sex %d skill %d holds %+v", sex, skill, fighter)
			}
		}
	}
}
