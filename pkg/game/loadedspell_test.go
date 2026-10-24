package game

import (
	"fmt"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// loadedSpellSave is an original SAV whose first mage knows spells 6 and 26.
func loadedSpellSave() []byte {
	spells := make([]byte, 26)
	spells[5], spells[25] = 6, 26
	a := &poolFixtureActor{mapID: 91, cell: 0x0a0a, hp: 10, maxHP: 30, mana: 50, maxMana: 50, human: true, name: "First", spells: spells}
	b := &poolFixtureActor{mapID: 92, cell: 0x0a0b, hp: 10, maxHP: 30, mana: 50, maxMana: 50, human: true, name: "Second", spells: []byte{}}
	return savedContainer(poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{a, b}}}}, nil))
}

func loadedSpellOpen(t *testing.T, raw []byte) (*FrontEnd, *ui.App) {
	t.Helper()
	f := poolFixtureFront(t, 91, 92)
	row := make([]int32, data.MinHumanRow)
	row[16], row[17], row[18], row[25] = 0x17, 4, 1, 1<<1
	rows := dbCollection{{}}
	for i := 1; i <= 28; i++ {
		p := make([]int32, 19)
		p[1], p[4], p[6], p[8], p[16], p[17], p[18] = 1, 1, 10, 1, 1, 1, 1
		rows = append(rows, dbEntry{name: "spell", params: p})
	}
	f.Table = &mapload.Table{Humans: dbCollection{{}, {name: "PC_fixture", params: row}}, Spells: rows}
	f.Campaign = resolved(Campaign{Main: []int{10}, Offered: []int{10}, Chapters: map[int]Chapter{10: {Mission: 10}}}, nil)
	f.Font = resolved(missionFont(), nil)
	opener, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("RestoreOriginal: town=%v err=%v", town, err)
	}
	a := f.App("loaded spell")
	a.Layout(1024, 768)
	if err := a.OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	a.Layout(1024, 768)
	// The fixture's mage class, which C and the Cast cell require.
	for i := range f.live.mission.party {
		f.live.mission.party[i].Class = 0x18
	}
	f.live.push()
	return f, a
}

func loadedSpellKey(t *testing.T, a *ui.App, key string) {
	t.Helper()
	if err := a.HeadlessKey(key); err != nil {
		t.Fatal(err)
	}
}

func loadedSpellTap(t *testing.T, a *ui.App, x, y int, edges ...string) {
	t.Helper()
	for _, edge := range edges {
		if err := a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
}

func loadedSpellBookShown(a *ui.App) bool {
	_, _, err := a.HeadlessSpellPoint(6)
	return err == nil
}

// loadedSpellClick taps the mage and reports whether it queued a cast.
func loadedSpellClick(t *testing.T, f *FrontEnd, a *ui.App) bool {
	t.Helper()
	before := len(f.live.pending)
	x, y, err := a.HeadlessEntityPoint(uint32(poolEntity(t, f.live.world, 91).ID))
	if err != nil {
		t.Fatal(err)
	}
	loadedSpellTap(t, a, x, y, "press", "release")
	for _, c := range f.live.pending[before:] {
		if c.Kind == sim.KindCast {
			return true
		}
	}
	return false
}

// loadedSpellTrace plays a map click, C, a second click and a right click,
// and states what each did.
func loadedSpellTrace(t *testing.T, f *FrontEnd, a *ui.App) string {
	t.Helper()
	_, current, armed := f.live.view.QuickSpellState()
	out := fmt.Sprintf("spell %d armed %v book %v; click cast %v", current, armed, loadedSpellBookShown(a), loadedSpellClick(t, f, a))
	loadedSpellKey(t, a, "c")
	_, _, armed = f.live.view.QuickSpellState()
	out += fmt.Sprintf("; C armed %v book %v", armed, loadedSpellBookShown(a))
	out += fmt.Sprintf("; click cast %v book %v", loadedSpellClick(t, f, a), loadedSpellBookShown(a))
	x, y, err := a.HeadlessGroundPoint()
	if err != nil {
		t.Fatal(err)
	}
	loadedSpellTap(t, a, x, y, "right-press", "right-release")
	_, current, _ = f.live.view.QuickSpellState()
	return out + fmt.Sprintf("; right click selection %d spell %d", len(f.live.view.SelectedUnits()), current)
}

// A spell chosen in the book behaves after LOAD as it did before SAVE, with
// the book closed or open: the saved state is the book-chosen arm.
func TestLoadedBookSpellBehavesAsBeforeSave(t *testing.T) {
	for _, open := range []bool{false, true} {
		name := map[bool]string{false: "book closed", true: "book open"}[open]
		t.Run(name, func(t *testing.T) {
			f, a := loadedSpellOpen(t, loadedSpellSave())
			if err := a.HeadlessSelectEntity(uint32(poolEntity(t, f.live.world, 91).ID)); err != nil {
				t.Fatal(err)
			}
			f.live.push()
			if !loadedSpellBookShown(a) {
				loadedSpellKey(t, a, "book")
			}
			x, y, err := a.HeadlessSpellPoint(6)
			if err != nil {
				t.Fatal(err)
			}
			loadedSpellTap(t, a, x, y, "press", "release")
			if !open {
				loadedSpellKey(t, a, "book")
			}
			snapshot, label, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := f.ExportCurrentSave(snapshot, label)
			if err != nil {
				t.Fatal(err)
			}
			want := loadedSpellTrace(t, f, a)
			t.Log("before SAVE:", want)
			for cycle := 0; cycle < 2; cycle++ {
				g, b := loadedSpellOpen(t, raw)
				if got := loadedSpellTrace(t, g, b); got != want {
					t.Errorf("LOAD %d: %s\nbefore SAVE: %s", cycle, got, want)
				}
			}
		})
	}
}
