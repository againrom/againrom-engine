package main

import (
	"fmt"
	"sort"
	"strings"

	"againrom/pkg/game"
	"againrom/pkg/ui"
)

// driveTown reaches the town square with no live ui.App, through the same
// FrontEnd-level path cmd/shopdump and cmd/plaquescreens already use — see
// pkg/ui/app.go's composeTownRoom for why that path exists — and composes
// each room through ui.ComposeTownScreen, the exported half of the one seam
// this tool shares with Draw (main.go's own header).
func driveTown(f *game.FrontEnd) map[string]capturedFrame {
	out := map[string]capturedFrame{}

	n, err := openTown(f)
	if err != nil {
		return failAll(out, err, "town-square", "town-tavern", "town-tavern-selected", "town-tavern-next-frame", "town-tavern-talk", "town-shop", "town-shop-stats")
	}
	t := f.TownScreen()
	if t == nil {
		return failAll(out, fmt.Errorf("mission %d declared a town and TownScreen() returned nil", n),
			"town-square", "town-tavern", "town-tavern-selected", "town-tavern-next-frame", "town-tavern-talk", "town-shop", "town-shop-stats")
	}

	pix, err := ui.ComposeTownScreen(t, "")
	out["town-square"] = capturedFrame{pix: pix, err: err, tried: true}

	tavernOpen, err := enterRoom(t, "TAVERN")
	if err != nil || !tavernOpen {
		if err == nil {
			err = fmt.Errorf("this campaign's town room lists no TAVERN door")
		}
		for _, name := range []string{"town-tavern", "town-tavern-selected", "town-tavern-next-frame", "town-tavern-talk"} {
			out[name] = capturedFrame{err: err}
		}
	} else {
		if tips, ok := t.(ui.TipScreen); ok {
			tips.CloseTip()
		}
		pix, err = ui.ComposeTownScreen(t, "")
		out["town-tavern"] = capturedFrame{pix: pix, err: err, tried: true}
		captureSelectedTavern(out, t)
		for i := 0; i < 32; i++ {
			if d, ok := t.(ui.TownDialogueScreen); ok {
				if _, shown := d.TownDialogue(); shown {
					d.AdvanceTownDialogue()
					continue
				}
			}
			break
		}
		t.Back()
	}

	shopOpen, err := enterShop(t)
	if err != nil {
		out["town-shop"] = capturedFrame{err: fmt.Errorf("open the shop door: %w", err)}
		out["town-shop-stats"] = capturedFrame{err: fmt.Errorf("open the shop door: %w", err)}
		return out
	}
	if !shopOpen {
		out["town-shop"] = capturedFrame{err: fmt.Errorf("this campaign's first town room lists no SHOP door")}
		out["town-shop-stats"] = capturedFrame{err: fmt.Errorf("this campaign's first town room lists no SHOP door")}
		return out
	}
	pix, err = ui.ComposeTownScreen(t, "")
	out["town-shop"] = capturedFrame{pix: pix, err: err, tried: true}

	// town-shop-stats is the same room with the character pane's DOLL/STATS
	// toggle pressed, pressed through TownShopScreen.ShopClick, which is the
	// one mutation door ui.App itself uses for that control. A shop that
	// composes and a mode that does not is a difference worth a separate frame
	// rather than a silent reuse of the first one.
	s, ok := t.(ui.TownShopScreen)
	if !ok || !s.AtTownShop() {
		out["town-shop-stats"] = capturedFrame{err: fmt.Errorf("the shop room is open and it is not a ui.TownShopScreen")}
		return out
	}
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlCharacterMode})
	pix, err = ui.ComposeTownScreen(t, "")
	out["town-shop-stats"] = capturedFrame{pix: pix, err: err, tried: true}
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlCharacterMode})
	return out
}

func captureSelectedTavern(out map[string]capturedFrame, t ui.TownScreen) {
	s, ok := t.(ui.TownSurfaceScreen)
	if !ok || !s.AtTownSurface() {
		err := fmt.Errorf("the tavern room is open and it is not a ui.TownSurfaceScreen")
		out["town-tavern-selected"], out["town-tavern-next-frame"], out["town-tavern-talk"] = capturedFrame{err: err}, capturedFrame{err: err}, capturedFrame{err: err}
		return
	}
	v := s.TownSurface()
	merc := -1
	for i, cell := range v.Cells {
		if cell.Portrait && !cell.TalkOnly {
			merc = i
			break
		}
	}
	if merc < 0 {
		err := fmt.Errorf("the reached tavern has no available mercenary card")
		out["town-tavern-selected"], out["town-tavern-next-frame"], out["town-tavern-talk"] = capturedFrame{err: err}, capturedFrame{err: err}, capturedFrame{err: err}
		return
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: merc}, false)
	pix, err := ui.ComposeTownScreenFrame(t, "", 0)
	out["town-tavern-selected"] = capturedFrame{pix: pix, err: err, tried: true}
	pix, err = ui.ComposeTownScreenFrame(t, "", 1)
	out["town-tavern-next-frame"] = capturedFrame{pix: pix, err: err, tried: true}
	v = s.TownSurface()
	for i, button := range v.Buttons {
		if strings.EqualFold(strings.TrimSpace(button.Label), "Talk") && button.Enabled {
			s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: i}, false)
			pix, err = ui.ComposeTownScreen(t, "")
			out["town-tavern-talk"] = capturedFrame{pix: pix, err: err, tried: true}
			return
		}
	}
	out["town-tavern-talk"] = capturedFrame{err: fmt.Errorf("the selected mercenary has no enabled Talk button")}
}

// openTown finishes the first campaign mission the registry marks as
// offering a town, using this front end's own default new-game party — the
// same derivation cmd/shopdump and cmd/plaquescreens use (openTown there).
// It never plays the mission through the sim: FinishMission applies its
// declared outcome directly, which is the production shortcut those two
// tools already established for reaching a town with no full mission run.
func openTown(f *game.FrontEnd) (int, error) {
	party := f.NextParty()
	if target, ok := mercenaryTownTarget(f.Campaign.Value()); ok {
		var prior []int
		for mission := range f.Campaign.Value().Chapters {
			if mission < target {
				prior = append(prior, mission)
			}
		}
		sort.Ints(prior)
		for _, mission := range prior {
			f.FinishMission(mission, party, nil, nil)
			party = f.NextParty()
		}
		if f.Town != nil && f.Town.Open() && f.Town.Chapter() == target {
			return target, nil
		}
	}
	for n := 1; n < 200; n++ {
		offer, ok := f.Campaign.Value().NextMission(n)
		if !ok || !offer.Town {
			continue
		}
		f.FinishMission(n, party, nil, nil)
		if f.Town == nil {
			return 0, fmt.Errorf("mission %d declares a town and none opened", n)
		}
		return n, nil
	}
	return 0, fmt.Errorf("this install's campaign declares no town")
}

func mercenaryTownTarget(c game.Campaign) (int, bool) {
	for _, target := range c.Main {
		i := sort.SearchInts(c.Offered, target)
		if i >= len(c.Offered) || c.Offered[i] != target {
			continue
		}
		unlocked := make(map[int]bool)
		for mission, chapter := range c.Chapters {
			if mission >= target {
				continue
			}
			for _, typ := range chapter.EnableMercenary {
				unlocked[typ] = true
			}
		}
		for _, typ := range c.Chapters[target].Mercenaries {
			if unlocked[typ] && typ > 0 && typ <= len(c.MercenaryCount) && c.MercenaryCount[typ-1] > 0 {
				return target, true
			}
		}
	}
	return 0, false
}

// enterShop presses the square's own SHOP door, exactly as ui.App's own hit
// test would (TownScreen.Choose), and pages the merchant's greeting to its
// end if entering the shop opened one. It reports false, not an error, when
// the room lists no SHOP door at all — that is a fact about this campaign's
// first town room, not a drive failure.
func enterShop(t ui.TownScreen) (bool, error) {
	pressed, err := enterRoom(t, "SHOP")
	if err != nil {
		return false, err
	}
	if !pressed {
		return false, nil
	}
	d, ok := t.(ui.TownDialogueScreen)
	if !ok {
		return true, nil
	}
	for i := 0; i < 32; i++ {
		if _, shown := d.TownDialogue(); !shown {
			return true, nil
		}
		d.AdvanceTownDialogue()
	}
	return false, fmt.Errorf("the merchant's greeting did not end")
}

func enterRoom(t ui.TownScreen, name string) (bool, error) {
	for i, row := range t.Rows() {
		if !row.Choosable || !strings.HasPrefix(strings.TrimSpace(row.Text), name) {
			continue
		}
		t.Choose(i)
		return true, nil
	}
	return false, nil
}
