package game

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

var documentPairs = []Document{{1, DocumentText}, {2, DocumentText}, {3, DocumentText}, {4, DocumentText}, {1, DocumentPicture}}

// The exact tuples come from REG-SCN-097, independently of the code granting
// them. DAT-DOC-021 identifies the physical item. Its fresh grant is owner
// direction (DIV-305), not a claim that ROM1 grants it during chargen.
func documentsState1176(f *FrontEnd, onMap bool, pages int) error {
	if got := f.Town.Documents(); !reflect.DeepEqual(got, documentPairs[:pages]) && !(pages == 0 && len(got) == 0) {
		return fmt.Errorf("collected pages=%v, want %v", got, documentPairs[:pages])
	}
	physical, primary := 0, 0
	party := f.Carried
	if onMap {
		party = f.live.mission.party
	}
	for i, member := range party {
		count := 0
		if onMap {
			stacks, ok := f.live.world.CarriedStacks(f.live.mission.ids[i])
			if !ok {
				return fmt.Errorf("starting hero lacks a live pack")
			}
			for _, item := range stacks {
				if item.Code == 0x0e1c {
					count += int(item.Count)
				}
			}
		} else {
			items := member.Carried
			if member.Carry != nil {
				items = member.Carry.Items
			}
			for _, code := range items {
				if code == 0x0e1c {
					count++
				}
			}
		}
		physical += count
		if member.StartingHero {
			primary += count
		}
	}
	if physical != 1 || primary != 1 {
		return fmt.Errorf("physical documents: party=%d primary=%d, want one on the primary", physical, primary)
	}
	return nil
}

func requireDocuments(t *testing.T, f *FrontEnd, onMap bool, pages int) {
	t.Helper()
	if err := documentsState1176(f, onMap, pages); err != nil {
		t.Fatal(err)
	}
}

func documentsFrame1176(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	if _, kind, up := f.LiveNotice(); up && kind != ui.NoticeSuccess {
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
}

// Reuse 1060's objective actions against the actual App's World. Those actions
// accelerate placement/death conditions; App frames still execute the installed
// script and show its notice. This is not an unassisted campaign playthrough.
func documentsObjective(t *testing.T, f *FrontEnd, app *ui.App, mission int) {
	t.Helper()
	scenario, err := ReadHeadlessScenario(filepath.Join("..", "..", "scenarios", fmt.Sprintf("1060-campaign-%03d-reachable.json", mission)))
	if err != nil {
		t.Fatal(err)
	}
	p := PlayWorld{World: f.live.world, Script: mapload.ScriptUnits(f.live.mission.state.Map, f.live.mission.party), Party: f.live.mission.ids}
	for _, step := range scenario.Steps {
		switch step.Command {
		case "kill", "place":
			id, err := p.Resolve(step.Unit)
			if err != nil {
				t.Fatal(err)
			}
			if step.Command == "kill" {
				err = p.World.HeadlessKill(id)
			} else {
				err = p.World.HeadlessPlace(id, *step.X, *step.Y)
			}
			if err != nil {
				t.Fatal(err)
			}
			documentsFrame1176(t, f, app)
		case "wait_ticks", "wait_until":
			for i := 0; i < step.Ticks; i++ {
				if step.Command == "wait_until" && p.World.Outcome() == sim.OutcomeWon {
					break
				}
				documentsFrame1176(t, f, app)
			}
		case "assert_world":
			if p.World.Outcome() != sim.OutcomeWon {
				t.Fatalf("mission%d objective script did not win: %v", mission, p.World.Outcome())
			}
		default:
			t.Fatalf("objective witness must handle the existing action %q", step.Command)
		}
	}
	for i := 0; i < 300; i++ {
		if _, kind, up := f.LiveNotice(); up && kind == ui.NoticeSuccess {
			return
		}
		documentsFrame1176(t, f, app)
	}
	t.Fatal("objective script won without a usable Victory notice")
}

func documentsPanel1176(t *testing.T, f *FrontEnd, app *ui.App, pages int) {
	t.Helper()
	for i := 0; i < 10; i++ {
		documentsFrame1176(t, f, app)
	}
	for i := 0; i < 128; i++ {
		if _, _, up := f.LiveNotice(); !up {
			break
		}
		documentsFrame1176(t, f, app)
	}
	var hero sim.EntityID
	for i, p := range f.live.mission.party {
		if p.StartingHero {
			hero = f.live.mission.ids[i]
		}
	}
	if hero == 0 {
		t.Fatal("documents panel lacks its starting hero")
	}
	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	for _, entity := range f.live.world.Entities() {
		if entity.ID == hero {
			inspectionCentre(f.live, int(entity.X), int(entity.Y))
		}
	}
	if err := app.HeadlessSelectEntity(uint32(hero)); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	code := uint16(0x0e1c)
	for _, action := range []string{"press", "release", "press", "release"} {
		if err := headlessPointer(f, app, HeadlessStep{Action: action, At: &HeadlessPoint{PackCode: &code}}); err != nil {
			t.Fatal("ordinary document equip gesture", err)
		}
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	state, open := app.HeadlessDocumentState()
	if !open || state.Elements != pages {
		t.Fatal("physical item's equip gesture did not open the collected pages", state, open)
	}
	if pages == 5 {
		for i := 0; i < 32 && !state.Picture; i++ {
			x, y, err := app.HeadlessDocumentPoint("right")
			if err != nil {
				t.Fatal(err)
			}
			for _, action := range []string{"press", "release"} {
				if err := app.HeadlessPointer(action, x, y); err != nil {
					t.Fatal(err)
				}
			}
			state, _ = app.HeadlessDocumentState()
		}
		if state.Element != 4 || !state.Picture {
			t.Fatal("collected picture does not resolve after the four texts", state)
		}
	}
	if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenMap {
		t.Fatal("document panel does not close to mission", app.Screen(), err)
	}
	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	requireDocuments(t, f, true, pages)
}

func documentsUISave(t *testing.T, f *FrontEnd, app *ui.App, pages int) (*FrontEnd, *ui.App, []byte) {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenGameMenu {
		t.Fatal("open ordinary save menu", app.Screen(), err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveEdit(store.Dir, "Documents checkpoint", ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(store.Dir, "Documents checkpoint.sav"))
	if err != nil {
		t.Fatal("player SAVE did not publish SAV", err)
	}
	g, a := campaignCold(t, store)
	requireDocuments(t, g, a.Screen() == ui.ScreenMap, pages)
	return g, a, raw
}

func documentsShop1176(t *testing.T, f *FrontEnd, app *ui.App, pages int) {
	t.Helper()
	s := f.TownScreen().(*townScreen)
	for i, door := range townDoors {
		if door.room == roomShop {
			if err := app.HeadlessActivate(s.Rows()[i].Text); err != nil {
				t.Fatal("ordinary shop door", err)
			}
			break
		}
	}
	if !s.AtTownShop() {
		t.Fatal("shop did not open")
	}
	for i := 0; i < 64; i++ {
		if _, up := s.TownDialogue(); !up {
			break
		}
		if err := app.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
	}
	if s.room != roomShop {
		t.Fatal("shop conversation did not return to its room")
	}
	// Use the same control dispatch as App after its hit test. The weapon
	// goes through actual unequip/equip before the mixed sale control.
	worn := *s.shopWornSlots(s.shopMemberIndex())
	slot := 0
	for i, code := range worn {
		if code != 0 {
			slot = i + 1
			break
		}
	}
	if slot == 0 {
		t.Fatal("fresh hero has no wearable control item")
	}
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlDoll, Index: slot - 1})
	findPack := func(code uint16) int {
		for i, item := range s.shopPackStacks() {
			if item.Code == code {
				return i + 1
			}
		}
		t.Fatalf("missing shop pack code %04x", code)
		return -1
	}
	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: findPack(worn[slot-1])}, ui.ShopControl{Kind: ui.ShopControlDoll})
	if *s.shopWornSlots(s.shopMemberIndex()) != worn {
		t.Fatal("normal equipment round trip changed the hero")
	}
	requireDocuments(t, f, false, pages)
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: findPack(0x0e1c)})
	single, singleGold := f.Shop.Table(), f.Town.Gold()
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 2})
	if !reflect.DeepEqual(single, f.Shop.Table()) || f.Town.Gold() != singleGold {
		t.Fatal("standalone document sale changed the source item or purse")
	}
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlTableCell, Index: 0})
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlDoll, Index: slot - 1})
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: findPack(worn[slot-1])})
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: findPack(0x0e1c)})
	table := f.Shop.Table()
	if len(table) != 2 || table[0].Price <= 0 || table[1].Code != data.QuestDocumentCode || table[1].Price != -1 {
		t.Fatal("mixed sale needs a real priced item beside the unpriced physical document", table)
	}
	gold := f.Town.Gold()
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 2})
	left := f.Shop.Table()
	if len(left) != 1 || left[0].Code != data.QuestDocumentCode || left[0].Price != -1 || !left[0].Mine || f.Town.Gold() != gold+int((table[0].Count*table[0].Price+1)/2) {
		t.Fatal("mixed sale lost the document or failed to sell its priced control", left, gold, f.Town.Gold())
	}
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlTableCell, Index: 0})
	requireDocuments(t, f, false, pages)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if !s.AtTownSquare() {
		t.Fatal("shop cannot return to square")
	}
}

func documentsRawSAV1176(t *testing.T, raw []byte, pages int) {
	t.Helper()
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	campaign, _, err := source.Campaign()
	if err != nil || len(campaign.Documents) != pages {
		t.Fatal("SAV source lost collected pages", campaign.Documents, err)
	}
	for i, pair := range documentPairs[:pages] {
		if campaign.Documents[i].Value != uint32(pair.Value) || campaign.Documents[i].Kind != uint32(pair.Kind) {
			t.Fatal("SAV source reordered or changed page", i, campaign.Documents[i], pair)
		}
	}
	party, err := source.Party()
	if err != nil {
		t.Fatal(err)
	}
	physical := 0
	for _, p := range party {
		for _, item := range p.Items {
			if item.Code == 0x0e1c {
				if item.Price != -1 {
					t.Fatal("SAV rewrote the source document price", item.Price)
				}
				physical += int(item.Stack)
			}
		}
	}
	if physical != 1 {
		t.Fatal("SAV source confused collected pages with physical items", physical)
	}
}

// Each source loss changes one independent carrier in the emitted SAV. LOAD
// and subsequent mission entry must retain that deliberate absence; neither
// the page collection nor fresh-party defaults may reconstruct the other.
func documentsLossControls1176(t *testing.T, raw []byte) {
	t.Helper()
	for _, loss := range []string{"missing_page", "missing_physical_item"} {
		t.Run(loss, func(t *testing.T) {
			source, err := sav.Open(raw)
			if err != nil {
				t.Fatal(err)
			}
			var changed []byte
			if loss == "missing_page" {
				campaign, _, err := source.Campaign()
				if err != nil || len(campaign.Documents) != 5 {
					t.Fatal("page control lacks the five source tuples", err)
				}
				if err := source.SetCampaignDocuments(campaign.Documents[:4]); err != nil {
					t.Fatal(err)
				}
				changed = source.Marshal()
			} else {
				provenance, err := source.CityProvenance()
				if err != nil {
					t.Fatal(err)
				}
				update := sav.CityUpdate{Money: source.Players[0].Money}
				removed := 0
				for _, actor := range provenance.Roster() {
					v := sav.CityCharacterUpdate{Identity: actor.Identity, Name: actor.Name, Stats: actor.Stats,
						SkillLevels: actor.SkillLevels, SkillXP: actor.SkillXP, Experience: actor.Experience}
					items, err := provenance.Inventory(actor.Identity)
					if err != nil {
						t.Fatal(err)
					}
					for i, item := range items {
						if item.Piece.Code == 0x0e1c {
							v.Sales = append(v.Sales, sav.CityItemSale{Position: uint32(i), Quantity: 1})
							removed++
						}
					}
					update.Characters = append(update.Characters, v)
				}
				if removed != 1 {
					t.Fatal("physical loss control did not target exactly one source item", removed)
				}
				changed, err = provenance.Marshal(update)
				if err != nil {
					t.Fatal("remove physical item from detached source control", err)
				}
			}
			store := SaveStore{Dir: t.TempDir()}
			if err := os.WriteFile(filepath.Join(store.Dir, "Loss control.sav"), changed, 0600); err != nil {
				t.Fatal(err)
			}
			f, app := campaignCold(t, store)
			for _, onMap := range []bool{false, true} {
				if onMap {
					takeCampaignOffer(t, f, 70)
					if err := app.OpenMission(f.MissionOpenerWith(70, f.NextParty())); err != nil {
						t.Fatal("loss control next mission", err)
					}
				}
				if documentsState1176(f, onMap, 5) == nil {
					t.Fatal("source loss was silently reconstructed on LOAD or entry", loss, onMap)
				}
				if loss == "missing_page" {
					requireDocuments(t, f, onMap, 4)
				} else {
					if !reflect.DeepEqual(f.Town.Documents(), documentPairs) || f.HeadlessSnapshot(app.Screen()).Documents != 0 {
						t.Fatal("physical loss changed the separate collection or reminted the item", onMap)
					}
				}
			}
		})
	}
}

func TestReleaseDocuments1176ActualGrantsReturnShopColdSave(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Document traveler", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	requireDocuments(t, f, false, 0)
	app := f.App("1176 fresh document grant")
	if err := app.OpenMission(f.MissionOpenerWith(10, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	requireDocuments(t, f, true, 3)
	documentsPanel1176(t, f, app, 3)
	f, app, _ = documentsUISave(t, f, app, 3)
	documentsObjective(t, f, app, 10)
	if err := app.HeadlessActivate("notice"); err != nil || f.liveMission != 20 {
		t.Fatal("mission10 Victory did not auto-enter20", err)
	}
	requireDocuments(t, f, true, 3)
	documentsObjective(t, f, app, 20)
	campaignReturn(t, f, app)
	requireDocuments(t, f, false, 3)
	shopStore, _, _ := campaignSave(t, f, false)
	t.Run("mixed_sale_and_cold_save", func(t *testing.T) {
		g, a := campaignCold(t, shopStore)
		documentsShop1176(t, g, a, 3)
		g, a, _ = documentsUISave(t, g, a, 3)
		_, _, raw := documentsUISave(t, g, a, 3)
		documentsRawSAV1176(t, raw, 3)
	})
	f, app, _ = documentsUISave(t, f, app, 3)
	// Chapters30/40 are controlled prior-victory inputs only. No helper
	// grants pages; each later page must come from its actual mission entry.
	f.Town.Won(30)
	f.Town.Won(40)
	for _, mission := range []int{50, 60} {
		takeCampaignOffer(t, f, mission)
		if err := app.OpenMission(f.MissionOpenerWith(mission, f.NextParty())); err != nil {
			t.Fatal(err)
		}
		pages := 4
		if mission == 60 {
			pages = 5
		}
		requireDocuments(t, f, true, pages)
		documentsPanel1176(t, f, app, pages)
		f, app, _ = documentsUISave(t, f, app, pages)
		documentsObjective(t, f, app, mission)
		campaignReturn(t, f, app)
		requireDocuments(t, f, false, pages)
		f, app, _ = documentsUISave(t, f, app, pages)
	}
	f, app, raw := documentsUISave(t, f, app, 5)
	documentsRawSAV1176(t, raw, 5)
	// A second SAV from the cold imported city must preserve both independent
	// representations, then actual mission entry must keep the same collection.
	f, app, raw = documentsUISave(t, f, app, 5)
	documentsRawSAV1176(t, raw, 5)
	documentsLossControls1176(t, raw)
	takeCampaignOffer(t, f, 70)
	if err := app.OpenMission(f.MissionOpenerWith(70, f.NextParty())); err != nil {
		t.Fatal("cold imported city next mission", err)
	}
	documentsPanel1176(t, f, app, 5)
	requireDocuments(t, f, true, 5)
	t.Log("grant route: one physical item, collected tuples3/4/5, actual script victories10/20/50/60, cold SAV, two cold city SAVs and next mission70")
}
