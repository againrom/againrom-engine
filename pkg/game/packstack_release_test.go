package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// packStackCells lists the cells holding one of codes in pack order.
func packStackCells(stacks []sim.ItemStack, codes ...uint16) string {
	var out []string
	for _, st := range stacks {
		if slices.Contains(codes, st.Code) {
			out = append(out, fmt.Sprintf("%#x x%d", st.Code, st.Count))
		}
	}
	return fmt.Sprint(out)
}

// TestReleaseEqualBowsShareOnePackCell: an archer takes his Wood Short Bow
// 0x8134 off in the town shop after mission 10 and in mission 20 picks up
// the 0x8134 lying in the Sack at (15,11): one cell of two through a mission
// SAVE and cold LOAD. He then buys 0x8134 and the Wood Short Bow 0x8114 from
// the weapons shelf: 0x8134 x3, and 0x8114, another code, in its own cell
// through a town SAVE and cold LOAD (ITEM-MERGE-129).
func TestReleaseEqualBowsShareOnePackCell(t *testing.T) {
	const bow, cheap = 0x8134, 0x8114
	f := releaseFront(t)
	party := f.ChargenParty(ui.ChargenResult{Name: "Archer", Choices: []int{0, 0, 4}, Stats: []int{31, 27, 24, 29}})
	app := f.App("equal bows")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatalf("open mission 10: %v", err)
	}
	if _, _, err := f.LiveCompleteCampaign(); err != nil {
		t.Fatalf("finish mission 10: %v", err)
	}
	if a := currentTownShop(f, "hero").ShopClick(ui.ShopControl{Kind: ui.ShopControlDoll, Index: 0}); a.Msg != "off, into the pack" {
		t.Fatalf("take the worn bow off: %q", a.Msg)
	}
	if err := app.OpenMission(f.MissionOpenerWith(20, f.NextParty())); err != nil {
		t.Fatalf("open mission 20: %v", err)
	}
	hero := equipmentReturnHero(t, f)
	sackAt := func() bool {
		return slices.ContainsFunc(f.live.world.Sacks(), func(s sim.Sack) bool { return s.X == 15 && s.Y == 11 })
	}
	if !sackAt() {
		t.Fatal("mission 20 has no Sack at (15,11)")
	}
	f.live.grab(uint32(hero), 15, 11, true)
	for tick := 0; tick < 20000 && sackAt(); tick++ {
		f.live.tick()
	}
	if sackAt() {
		t.Fatal("the hero did not pick up the Sack at (15,11)")
	}
	stacks, _ := f.live.world.CarriedStacks(hero)
	if got := packStackCells(stacks, bow); got != "[0x8134 x2]" {
		t.Fatalf("mission 20 pack bows %s after the pickup; want one cell of two", got)
	}

	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("mission SAVE wrote %q: %v", name, err)
	}
	g := releaseFront(t)
	_, _, load := g.SaveSeams(store, OriginalStore{}, nil)
	open, town, err := load(localOriginalSaveToken(name))
	if err != nil || town {
		t.Fatalf("cold LOAD of the mission SAVE: town=%v err=%v", town, err)
	}
	if err := g.App("equal bows loaded").OpenMission(open); err != nil {
		t.Fatalf("reopen mission 20: %v", err)
	}
	stacks, _ = g.live.world.CarriedStacks(equipmentReturnHero(t, g))
	if got := packStackCells(stacks, bow); got != "[0x8134 x2]" {
		t.Fatalf("loaded mission 20 pack bows %s; want one cell of two", got)
	}

	if _, _, err := g.LiveCompleteCampaign(); err != nil {
		t.Fatalf("finish mission 20: %v", err)
	}
	s := currentTownShop(g, "hero")
	currentTownBuy(t, s, roomWeapons, bow)
	currentTownBuy(t, s, roomWeapons, cheap)
	if got := packStackCells(s.shopPackStacks(), bow, cheap); got != "[0x8134 x3 0x8114 x1]" {
		t.Fatalf("town pack bows %s after two purchases; want 0x8134 x3 and 0x8114 x1", got)
	}
	h := currentTownReload(t, currentTownSave(t, g))
	if got := packStackCells(currentTownShop(h, "hero").shopPackStacks(), bow, cheap); got != "[0x8134 x3 0x8114 x1]" {
		t.Fatalf("cold town LOAD pack bows %s; want 0x8134 x3 and 0x8114 x1", got)
	}
}

// TestReleaseBoughtItemsJoinTheRestoredCells: a LOAD of the owner's original
// town SAV game0010.sav, pinned by its SHA-256, restores Danath's pack with
// the Wood Short Bow 0x8114 x9 in its saved Weapon blocks and the potion
// 0x0e06 x2 at weight 1, which the tables do not give a potion. A 0x8114
// bought from the weapons shelf and a 0x0e06 bought from the fourth shelf
// hold no saved operand and join those cells: 0x8114 x10 in the restored
// blocks and 0x0e06 x3 at weight 1. One of those potions sold back joins the
// shelf's own 0x0e06 cell. Both hold through a town SAVE and cold LOAD
// (ITEM-STACK-003, ITEM-MERGE-129).
func TestReleaseBoughtItemsJoinTheRestoredCells(t *testing.T) {
	const cheap, potion = 0x8114, 0x0e06
	root := os.Getenv("AGAINROM_SAVE_CORPUS")
	if os.Getenv("AGAINROM_ASSETS") == "" || root == "" {
		t.Skip("AGAINROM_ASSETS and AGAINROM_SAVE_CORPUS required")
	}
	raw, err := os.ReadFile(filepath.Join(root, "2026-08-02", "game0010.sav"))
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%X", sha256.Sum256(raw)); got != "B4E5CEB73018D0C4656515CE643712FB11D5BFA57FE9065361DE5BA8156BC0AC" {
		t.Fatalf("owner SAV SHA256=%s", got)
	}
	f := currentTownReload(t, raw)
	s := currentTownShop(f, "hero")
	cell := func(stacks []sim.ItemStack, code uint16) int {
		return slices.IndexFunc(stacks, func(st sim.ItemStack) bool { return st.Code == code })
	}
	stacks := s.shopPackStacks()
	i, k := cell(stacks, cheap), cell(stacks, potion)
	if got := packStackCells(stacks, cheap, potion); i < 0 || k < 0 || got != "[0x8114 x9 0xe06 x2]" || !stacks[i].WeightPresent || stacks[k].Weight != 1 {
		t.Fatalf("restored pack %s; want the saved 0x8114 x9 and 0x0e06 x2 at weight 1", got)
	}
	blocks := stacks[i].SourceEquipment
	currentTownBuy(t, s, roomWeapons, cheap)
	currentTownBuy(t, s, roomBooks, potion)
	check := func(label, want string, stacks []sim.ItemStack) {
		t.Helper()
		if got := packStackCells(stacks, cheap, potion); got != want || cell(stacks, cheap) != i || cell(stacks, potion) != k ||
			stacks[i].SourceEquipment != blocks || !stacks[k].WeightPresent || stacks[k].Weight != 1 {
			t.Fatalf("%s %s at cells %d and %d; want %s in the restored cells %d and %d, the potion at weight 1",
				label, got, cell(stacks, cheap), cell(stacks, potion), want, i, k)
		}
	}
	check("town pack after the purchases", "[0x8114 x10 0xe06 x3]", s.shopPackStacks())
	shelf := func(f *FrontEnd) string {
		var out []string
		for _, item := range f.Shop.Shelf(ShelfBooks) {
			if uint16(item.Code) == potion {
				out = append(out, fmt.Sprintf("x%d", item.Count))
			}
		}
		return fmt.Sprint(out)
	}
	before := shelf(f)
	var n int
	if _, err := fmt.Sscanf(before, "[x%d]", &n); err != nil {
		t.Fatalf("fourth shelf 0x0e06 cells %s; want one cell", before)
	}
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: currentTownPackCell(t, s, potion)})
	if action := s.shopSell(); !strings.HasPrefix(action.Msg, "he pays") {
		t.Fatalf("sell one 0x0e06: %q", action.Msg)
	}
	want := fmt.Sprintf("[x%d]", n+1)
	if got := shelf(f); got != want {
		t.Fatalf("fourth shelf 0x0e06 cells %s after the sale; want %s", got, want)
	}
	check("town pack after the sale", "[0x8114 x10 0xe06 x2]", s.shopPackStacks())
	h := currentTownReload(t, currentTownSave(t, f))
	check("cold town LOAD pack", "[0x8114 x10 0xe06 x2]", currentTownShop(h, "hero").shopPackStacks())
	if got := shelf(h); got != want {
		t.Fatalf("cold town LOAD fourth shelf 0x0e06 cells %s; want %s", got, want)
	}
}
