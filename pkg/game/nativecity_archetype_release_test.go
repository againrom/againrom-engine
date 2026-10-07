package game

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// The old game re-derived Mage from DefRow when reading its own output, so
// its cold-load appearance test passed a wire type that crashes ROM1. Read
// the written Human type independently of restoredDefinition and check the
// mage drawable path against the actual installed archive (HERO-APPEAR-041/043).
func TestReleaseNativeCityOriginalArchetypeAndMageSprites(t *testing.T) {
	for _, tc := range []struct {
		name            string
		sex, class      int
		hero, companion uint16
	}{
		{"man-fighter", 0, 0, 0x21, 0x24},
		{"woman-fighter", 1, 0, 0x22, 0x23},
		{"man-mage", 0, 1, 0x23, 0x24},
		{"woman-mage", 1, 1, 0x24, 0x23},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := releaseFront(t)
			f.Carried = f.ChargenParty(ui.ChargenResult{Name: tc.name, Choices: []int{tc.sex, tc.class, 3}, Stats: []int{31, 27, 24, 29}})
			f.arriveInTown()
			f.Town.mercEnabled[14] = true
			f.addChapterCompanions(f.Town.Chapter())
			if len(f.Carried) != 2 {
				t.Fatalf("expected hero plus chapter companion, got %d", len(f.Carried))
			}
			dir := t.TempDir()
			save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
			name, err := save(false)
			if err != nil || !IsOriginal(name) {
				t.Fatalf("ordinary SAVE = %q, %v; want original SAV", name, err)
			}
			files, err := filepath.Glob(filepath.Join(dir, "*.sav"))
			if err != nil || len(files) != 1 {
				t.Fatalf("ordinary SAV files = %v, %v", files, err)
			}
			b, err := os.ReadFile(files[0])
			if err != nil {
				t.Fatal(err)
			}
			file, err := sav.Open(b)
			if err != nil {
				t.Fatal(err)
			}
			p, err := file.CityProvenance()
			if err != nil {
				t.Fatal(err)
			}
			if len(p.Roster()) != 2 {
				t.Fatal("SAV lost a roster member")
			}
			checkNativeCityKnownFields(t, f, p)
			for _, character := range p.Roster() {
				want := tc.companion
				if character.Hero {
					want = tc.hero
				}
				human, err := p.Human(character.Identity)
				if err != nil {
					t.Fatal(err)
				}
				if human.TypeID != want {
					t.Fatalf("%q wire type = %#x, want %#x", character.Name, human.TypeID, want)
				}
				// ROM1 banks bit1 of typeID-0x21 as the drawable mage flag.
				if (want-0x21)&2 != 0 {
					directory := "heroes_l"
					if (human.TypeID-0x21)&2 != 0 {
						directory = "heroes"
					}
					for _, body := range []string{"mage", "mage_st"} {
						path := fmt.Sprintf("graphics/units/%s/%s/sprites.256", directory, body)
						if pixels, err := f.Archives.Containers.ReadFile(path); err != nil || len(pixels) == 0 {
							t.Fatalf("original mage selector reaches missing sprite %q: %v", path, err)
						}
					}
				}
			}
			// An independent negative control prevents a loose/misconfigured
			// install from hiding the owner's exact missing-resource failure.
			if _, err := f.Archives.Containers.ReadFile("graphics/units/heroes_l/mage_st/sprites.256"); err == nil {
				t.Fatal("the original crash path unexpectedly exists")
			}
		})
	}
}
