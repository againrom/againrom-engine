package game

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"reflect"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/render/text"
	"againrom/pkg/screenart"
	"againrom/pkg/secondshop"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

type sleepCollection struct {
	secondShopTestCollection
	size int
}

func (c sleepCollection) Len() int { return c.size }

func (sleepCollection) EntryName(int) string      { return "Potion" }
func (sleepCollection) EntryStrings(int) []string { return nil }

func sleepTown(id int) (*FrontEnd, *townScreen) {
	rows := sleepCollection{secondShopTestCollection{}, 128}
	for i := 69; i <= 74; i++ {
		rows.secondShopTestCollection[i] = []int32{10}
	}
	// R2-ENGINE-941 requires a candidate before the potion lead.
	for _, i := range []int{6, 35} {
		rows.secondShopTestCollection[i] = []int32{10}
	}
	tab := &mapload.Table{Shapes: rows, Materials: rows, Armors: rows, Shields: rows,
		Weapons: rows, MagicItems: rows, Spells: sleepCollection{secondShopTestCollection{}, 1}, Magic: rows}
	c := newSecondCampaign()
	c.current, c.available, c.room = partyPlace{2, id}, []partyPlace{{2, id}}, secondTownInn
	c.party = []mapload.PartyMember{{ID: "21", Name: "Sleeper", StartingHero: true, PlayerCharacter: true, Hero: data.NewCampaignHero(1)}}
	record := secondshop.Record{3: {Min: 0, Max: 10000, Draws: 5, Bound: 2, Mask: 1 << 26}}
	c.shop.records[id] = &record
	f := &FrontEnd{InstallResources: InstallResources{Table: tab, Words: ui.AuthoredWords(), Archives: &Archives{Base: base.Match{Profile: base.Profile{Game: base.GameROM2}}}},
		CampaignSession: CampaignSession{Town: &Town{second: c, gold: 1234}, Carried: mapload.CloneParty(c.party)},
		RuntimeServices: RuntimeServices{Random: random.NewService(random.Session{Seed: 71})}}
	s := &townScreen{shopController: shopController{in: &f.InstallResources, sess: &f.CampaignSession},
		draws: runtimeTownDraws{&f.RuntimeServices}}
	s.adapter = destinationTownCampaign{s}
	f.townUI = s
	tables, _ := secondShopTables(tab)
	c.shop.drawStock(tables, id, f.randomService().Stream(random.ShopStock).Raw)
	return f, s
}

func TestSecondInnSleepButtonAndCardProfiles(t *testing.T) {
	for _, id := range []int{1, 2, 3} {
		f, s := sleepTown(id)
		d := s.townDescription()
		r := d.Room("inn")
		if len(r.Buttons) != 4 || r.Buttons[0].Name != "sleep" || r.Buttons[0].Word != "tavern.sleep" || r.Buttons[0].Rect.Rectangle() != ui.ShopButtonRect(0) {
			t.Fatal("inn has no data-driven Sleep plaque", id, r.Buttons)
		}
		for _, lang := range []string{"english", "russian"} {
			f.Words.Engine = words.For(lang)
			f.Words.TavernSleep = f.Words.Engine.Text("tavern.sleep")
			chooseSleep(t, s)
		}
		f.ChargenAssets = &ChargenAssets{Presentation: &ui.ChargenPresentation{Layout: GeneratorDescription(base.Profile{Game: base.GameROM2})}}
		seats, _ := s.state().innSeats()
		s.composed = &townSceneState{seat: 0, patrons: map[secondInnPatronKey]*secondInnPatron{
			{room: r, npc: seats[0].npc}: {known: true, subject: ui.PanelSubject{Name: "Sleeper"}},
		}}
		person := (roomPageView{t: s, d: d, room: "inn"}).patron(r, &screenart.Set{})
		hero, _, _ := s.heroPane(r.Pane)
		want := &ui.TextShadow{Ink: color.RGBA{8, 8, 8, 255}, At: image.Pt(1, 1)}
		if person.Card == nil || !reflect.DeepEqual(person.Card.CardShadow, want) || !reflect.DeepEqual(hero.CardShadow, want) {
			t.Fatal("patron and hero cards do not share the profile shadow", id)
		}
	}
}

// The Sleep plaque's caption is the Sleep row's text byte for byte, in each
// language and under each selector.
func TestSecondInnSleepPlaqueCaptionEqualsRowText(t *testing.T) {
	for _, id := range []int{1, 2, 3} {
		f, s := sleepTown(id)
		var word string
		for _, b := range s.townDescription().Room("inn").Buttons {
			if b.Name == "sleep" {
				word = b.Word
			}
		}
		for _, lang := range []string{"english", "russian"} {
			for sel := 0; sel <= 2; sel++ {
				f.InstallResources.Words = (&InstallWords{Language: lang, Selector: sel}).Words()
				f.InstallResources.Font = lazy[*text.Font]{value: &text.Font{Selector: sel}}
				if got, want := f.InstallResources.engineWord(word), f.Words.TavernSleep; got != want {
					t.Errorf("town %d %s selector %d: plaque % x, row % x", id, lang, sel, got, want)
				}
			}
		}
	}
}

func chooseSleep(t *testing.T, s *townScreen) {
	t.Helper()
	for i, row := range s.Rows() {
		if row.Text == s.in.Words.TavernSleep {
			if !row.Choosable {
				t.Fatal("Sleep is disabled")
			}
			if act := s.Choose(i); act.Msg != "" || act.Open != nil || act.Info {
				t.Fatalf("Sleep posts an action: %+v", act)
			}
			return
		}
	}
	t.Fatal("the inn lists no Sleep control")
}

func TestSecondInnSleepUsesTheStockStreamAndChangesOnlyShelves(t *testing.T) {
	for _, id := range []int{1, 2, 3} {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			f, s := sleepTown(id)
			c := f.Town.second
			control, _ := sleepTown(id)
			before := captureSecondCampaign(c)
			records := c.shop.records.clone()
			party, gold, selection := mapload.CloneParty(f.Carried), f.Town.gold, s.shopMember
			old := c.shop.stock()
			for n := 0; n < 2; n++ {
				tables, _ := secondShopTables(f.Table)
				want := tables.Fill(*c.shop.records[id], control.randomService().Stream(random.ShopStock).Raw)
				for k, shelf := range stockOf(tables, want) {
					want[k] = secondItems(shelf)
				}
				chooseSleep(t, s)
				if got := c.shop.stock(); !reflect.DeepEqual(got, want) || reflect.DeepEqual(got, old) {
					t.Fatalf("Sleep %d: shelves do not equal the next stock draw or did not change", n+1)
				}
				old = c.shop.stock()
				after := captureSecondCampaign(c)
				after.Shop, before.Shop = nil, nil
				if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(party, f.Carried) || gold != f.Town.gold || selection != s.shopMember || !reflect.DeepEqual(c.shop.records, records) {
					t.Fatal("Sleep changed party, gold, bank, destination, room or selected hero")
				}
			}
			if a, b := f.randomService().Stream(random.ShopStock).Raw(), control.randomService().Stream(random.ShopStock).Raw(); a != b {
				t.Fatal("Sleep consumed another number of stock draws", a, b)
			}
		})
	}
}

func TestSecondInnSleepSAVCarriesNoStock(t *testing.T) {
	loaded := func(slept bool) (*FrontEnd, secondshop.Stock) {
		f, s := sleepTown(1)
		if slept {
			chooseSleep(t, s)
		}
		live := f.Town.second.shop.stock()
		snapshot, _, err := f.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(snapshot, "Inn")
		if err != nil {
			t.Fatal(err)
		}
		doc := engineSaveDocument(t, raw)
		actions, err := readCurrentActions(&doc)
		if err != nil {
			t.Fatal(err)
		}
		shop, err := json.Marshal(actions.Session.Second.Shop)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(shop, &fields); err != nil || len(fields) != 1 || fields["Records"] == nil {
			t.Fatalf("shop record is not records alone: %s", shop)
		}
		cold, _ := sleepTown(1)
		cold.Town, cold.Carried = nil, nil
		cold.Random = random.NewService(random.Session{Seed: 999})
		if open, town, err := cold.RestoreOriginal(raw); err != nil || !town || open != nil {
			t.Fatal("cold LOAD", err, town, open != nil)
		}
		return cold, live
	}
	slept, liveSlept := loaded(true)
	plain, livePlain := loaded(false)
	got := slept.Town.second.shop.stock()
	if !reflect.DeepEqual(got, plain.Town.second.shop.stock()) {
		t.Fatal("cold LOAD after Sleep drew other than a LOAD without Sleep")
	}
	if reflect.DeepEqual(got, liveSlept) || !reflect.DeepEqual(got, livePlain) {
		t.Fatal("cold LOAD kept the slept stock instead of drawing the shop stream anew")
	}
}
