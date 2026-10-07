package game

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// cityHireGoldReads is every place the town and the mission entry read the
// purse from, taken on one loaded front end.
func cityHireGoldReads(t *testing.T, f *FrontEnd) map[string]int {
	t.Helper()
	reads := map[string]int{"town": f.Town.Gold()}
	screen := f.TownScreen().(*townScreen)
	var footer int
	if _, err := fmt.Sscanf(strings.Join(screen.Footer(), " "), "gold %d", &footer); err != nil {
		t.Fatal("town footer", err)
	}
	reads["footer"] = footer
	affordable := f.Town.Gold()
	for _, offer := range screen.tavernMercenaries() {
		if !offer.Hired && offer.Affordable != (offer.Price <= affordable) {
			t.Fatalf("tavern offer %d affordability does not follow the purse %d", offer.Type, affordable)
		}
	}
	reads["tavern"] = f.Town.Gold()
	app := f.App("city hire gold")
	if err := app.OpenMission(f.MissionOpener(f.Town.Chapter())); err != nil {
		t.Fatal(err)
	}
	reads["mission"] = int(f.live.world.Purse(sim.SelfSlot))
	return reads
}

// A city with paid hires keeps one purse across SAVE and cold LOAD: the hires
// are charged once, at the hire, and the loaded town, its tavern and the
// mission entry all read that same purse.
func TestReleaseCityHireGoldIsOnePurseAcrossSaveAndLoad(t *testing.T) {
	f, screen := hireForOrderTest(t, "City hire gold")
	before := f.Town.Gold()
	paid := 0
	for _, typ := range []int{6, 14} {
		price, ok := screen.mercenaryPrice(typ, f.Town.MercenaryPool(typ))
		if !ok {
			t.Fatal("price of type", typ)
		}
		if _, ok := screen.toggleMercenary(typ); !ok {
			t.Fatal("hire", typ)
		}
		paid += price
	}
	if paid <= 0 || f.Town.Gold() != before-paid {
		t.Fatalf("hires paid %d, purse %d -> %d", paid, before, f.Town.Gold())
	}
	want := f.Town.Gold()
	hired := f.Town.mercHired
	for cycle := 0; cycle < 2; cycle++ {
		raw := currentTownSave(t, f)
		if got := cityGlobalDWord(t, raw); got != uint32(paid) {
			t.Fatalf("cycle %d: SAV hire cost word %d, want the %d the hires cost", cycle, got, paid)
		}
		f = currentTownReload(t, raw)
		if f.Town.Gold() != want || f.Town.mercHired != hired {
			t.Fatalf("cycle %d: LOAD purse %d hired %v, want %d %v", cycle, f.Town.Gold(), f.Town.mercHired, want, hired)
		}
		if again := currentTownSave(t, f); len(again) == 0 {
			t.Fatal("empty SAVE")
		}
		for room, got := range cityHireGoldReads(t, f) {
			if got != want {
				t.Fatalf("cycle %d: %s reads purse %d, want %d", cycle, room, got, want)
			}
		}
		// The mission entry consumed the town; reload the saved city for the next cycle.
		f = currentTownReload(t, raw)
	}
}

func cityGlobalDWord(t *testing.T, raw []byte) uint32 {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	return doc.GlobalDWord
}

// The original keeps the accumulated hire cost in the word after the document
// marker: 520 in an original city that holds the one type 14 squad of chapter
// 30 and 0 in one that holds none. The engine writes the same word from the
// town's hire flags, so an original city and the engine's SAVE of it agree.
func TestReleaseCityHireCostWordMatchesOriginalCity(t *testing.T) {
	root := os.Getenv("AGAINROM_SAVE_CORPUS")
	if root == "" {
		t.Skip("AGAINROM_SAVE_CORPUS is not set")
	}
	for _, c := range []struct {
		file string
		want uint32
	}{{"game0056.sav", 520}, {"game0057.sav", 0}} {
		raw, err := os.ReadFile(filepath.Join(root, "2027-09-07", c.file))
		if err != nil {
			t.Skip("corpus file missing:", err)
		}
		if got := cityGlobalDWord(t, raw); got != c.want {
			t.Fatalf("%s: original hire cost word %d, want %d", c.file, got, c.want)
		}
		f := currentTownReload(t, raw)
		if got := cityGlobalDWord(t, currentTownSave(t, f)); got != c.want {
			t.Fatalf("%s: engine SAVE hire cost word %d, original %d", c.file, got, c.want)
		}
	}
}
