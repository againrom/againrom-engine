package game

import (
	"os"
	"testing"
	"time"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// abandonMissionSave is a SAV taken inside mission n under the mod, written to
// a store of its own; the party that entered is returned beside it.
func abandonMissionSave(t *testing.T, dir string, ids ...string) (store SaveStore, entry abandonTownState) {
	t.Helper()
	f, a, _, s := abandonTown(t, dir, ids...)
	if len(f.Carried) < 2 {
		t.Fatalf("the town holds %d members, the witness needs 2 or more", len(f.Carried))
	}
	entry = captureAbandonTown(f)
	abandonEnter(t, f, a, s, 30)
	abandonSteps(t, a, 40)
	store = SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, func() time.Time { return time.Date(2001, 1, 2, 0, 0, 0, 0, time.UTC) })
	if _, err := save(true); err != nil {
		t.Fatalf("mission SAVE: %v", err)
	}
	return store, entry
}

// abandonLoadFront is a fresh front end under the mods with the SAV directory
// of store behind its load window, accepting unmarked SAVs when asked.
func abandonLoadFront(t *testing.T, store SaveStore, orig OriginalStore, unmarked bool, dir string, ids ...string) (*FrontEnd, *ui.App, mod.ScreenData) {
	t.Helper()
	g, a, screens := abandonFront(t, dir, ids...)
	g.Table.Mods.AcceptUnmarked = unmarked
	save, list, load := g.SaveSeams(store, orig, nil)
	a.SetSaveSeams(save, list, load)
	return g, a, screens
}

// abandonLoadMission loads the newest SAV of the front end's load window and
// requires the map screen.
func abandonLoadMission(t *testing.T, g *FrontEnd, a *ui.App, row int) {
	t.Helper()
	keys := []string{"load"}
	for i := 0; i < row; i++ {
		keys = append(keys, "down")
	}
	keys = append(keys, "enter")
	for _, k := range keys {
		if err := a.HeadlessKey(k); err != nil {
			t.Fatalf("key %q: %v", k, err)
		}
	}
	if a.Screen() != ui.ScreenMap || g.live == nil || g.live.mission == nil {
		t.Fatalf("the mission LOAD left screen %s", a.Screen())
	}
}

// abandonThroughMenu chooses the action's menu entry and its confirming row.
func abandonThroughMenu(t *testing.T, g *FrontEnd, a *ui.App, screen mod.Screen) {
	t.Helper()
	if err := a.HeadlessKey("escape"); err != nil || a.Screen() != ui.ScreenGameMenu {
		t.Fatalf("Escape in the mission: %v on %s", err, a.Screen())
	}
	if err := a.HeadlessActivate(abandonLabel(g, screen.MenuLabel)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate(abandonLabel(g, screen.Title)); err != nil {
		t.Fatal(err)
	}
}

func abandonWalkHome(t *testing.T, g *FrontEnd, a *ui.App) *townScreen {
	t.Helper()
	s := g.TownScreen().(*townScreen)
	for i := 0; i < 20000 && !s.AtTownSquare(); i++ {
		if a.Screen() != ui.ScreenTown {
			t.Fatalf("homeward travel left screen %s", a.Screen())
		}
		abandonSteps(t, a, 1)
	}
	if !s.AtTownSquare() {
		t.Fatal("the party never reached the town square")
	}
	return s
}

// abandonSaveTown is a town SAVE through the production seam; an error is the
// refusal itself.
func abandonSaveTown(f *FrontEnd) ([]byte, error) {

	dir, err := os.MkdirTemp("", "abandon-town")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	store := SaveStore{Dir: dir}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		return nil, err
	}
	return store.Read(name)
}

// partyKey is the identity a town party is compared by: member ids and the
// worn and pack item codes.
func partyKey(party []mapload.PartyMember) []string {
	var out []string
	for _, p := range party {
		worn, pack := memberItemCodes(p)
		out = append(out, p.ID, formatCodes(worn[:]), formatCodes(pack))
	}
	return out
}

func formatCodes(codes []uint16) string {
	b := make([]byte, 0, len(codes)*5)
	for _, c := range codes {
		b = append(b, []byte{hexDigit(c >> 12), hexDigit(c >> 8), hexDigit(c >> 4), hexDigit(c), ','}...)
	}
	return string(b)
}

func hexDigit(v uint16) byte { return "0123456789abcdef"[v&15] }
