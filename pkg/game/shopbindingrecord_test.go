package game

import (
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The shop page is bound three ways: the first game's chapter shop, the
// second game's bank shop and the Shop page a mission visit shows. This test
// records what each binding yields on synthetic fixtures, so a change to how
// the page is bound holds every one of these answers equal.

// shopBindingProbe is a probe item of the first game's chapter shop.
func shopBindingChapterProbes(table *mapload.Table) []sim.ItemInstance {
	priced := sim.PlainItem(0x0e05)
	priced.Price = 7
	return []sim.ItemInstance{
		mapload.ItemInstanceFromCode(0x0e01, table),
		mapload.ItemInstanceFromCode(0x0e02, table),
		mapload.ItemInstanceFromCode(0x0e03, table),
		priced,
		sim.PlainItem(0x0e05),
	}
}

func shopBindingHooks(s *townScreen) string {
	var out []string
	add := func(name string, set bool) {
		if set {
			out = append(out, name)
		}
	}
	h := s.shop
	add("shown", h.shown != nil)
	add("sound", h.sound != nil)
	add("scene", h.scene != nil)
	add("speak", h.speak != nil)
	add("fresh", h.fresh != nil)
	add("dress", h.dress != nil)
	add("stepped", h.stepped != nil)
	add("stats", h.stats != nil)
	add("leave", h.leave != nil)
	add("chose", h.chose != nil)
	add("page", h.page != nil)
	add("covered", h.covered != nil)
	add("faces", h.faces != nil)
	add("face", h.face != nil)
	add("pane", h.pane != nil)
	return strings.Join(out, ",")
}

// shopBindingRecord is what the bound page yields, one line per answer.
func shopBindingRecord(s *townScreen, probes []sim.ItemInstance, second *secondCampaign) string {
	var b strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }
	for i := -1; i <= 4; i++ {
		line("rackName %d %q", i, s.rackName(i))
	}
	for i, r := range s.rack() {
		line("rack %d shelf %d stocked %v name %q", i, r.shelf, r.stocked, r.name)
	}
	for i, p := range probes {
		cell := s.labelled(ui.ShopCell{}, p)
		line("probe %d code %#04x kind %d price %d text %q info %q value %d", i, p.Code, p.Kind, p.Price, cell.Text, s.itemInfo(p), s.shopItemValue(p))
	}
	switch d := s.deal(); {
	case d == nil:
		line("deal nil")
	case d == s.sess.Shop:
		line("deal session shop")
	case second != nil && d == &second.shop:
		line("deal second campaign shop")
	default:
		line("deal other")
	}
	line("cityGraph %v", s.hasCityShopTopology())
	line("packUnits %d", s.row.packUnits)
	line("events %q", s.shop.events)
	line("hooks %s", shopBindingHooks(s))
	line("shownRoom %v", s.shownRoom())
	line("wornFixed %v", s.row.wornFixed)
	line("rules %+v", s.pageRules())
	line("book %v", s.paneBook())
	// The bag the strip shows, and what putting items in it keeps.
	line("pack %x", s.shopPackItems())
	line("hasActorContainer %v", s.shopHasActorContainer())
	entered := []sim.ItemInstance{sim.PlainItem(0x0e05), sim.PlainItem(0x0e03)}
	if len(probes) > 2 {
		entered = []sim.ItemInstance{probes[0], probes[len(probes)-1]}
	}
	line("setPack %v", s.setShopPackItemInstances(entered))
	for i, in := range s.shopPackItemInstances() {
		line("packed %d code %#04x kind %d price %d fx %d", i, in.Code, in.Kind, in.Price, len(in.Effects))
	}
	// What one table gesture settles.
	calls := 0
	a := s.shopTableSounds(false, false, func() ui.TownAction { calls++; return ui.TownAction{Msg: "gesture"} })
	line("gesture calls %d msg %q", calls, a.Msg)
	// The presentation state.
	line("state chosen %d shelf %d shelfBase %d packBase %d tableBase %d member %d book %v",
		s.shopChosen, s.shopShelf, s.shelfBase, s.packBase, s.tableBase, s.shopMember, s.shopBook)
	return b.String()
}

func dirtyShopPresentation(s *townScreen) {
	s.shopChosen, s.shopShelf, s.shelfBase, s.packBase, s.tableBase, s.shopMember, s.shopBook = 3, ShelfBooks, 4, 5, 6, 0, true
}

func TestShopBindingRecord(t *testing.T) {
	var got strings.Builder

	f := missionShopFront(t, 35)
	chapter := f.TownScreen().(*townScreen)
	chapter.sess.Town.cityObjects = &cityObjectTopology{}
	dirtyShopPresentation(chapter)
	rebindChapterShop(chapter)
	got.WriteString("== chapter\n" + shopBindingRecord(chapter, shopBindingChapterProbes(chapter.in.Table), nil))

	c := secondShopSample()
	bank := secondShopPage(c, 500)
	bank.in.Table = secondShopCellTable()
	bank.sess.Town.cityObjects = &cityObjectTopology{}
	dirtyShopPresentation(bank)
	bank.row.wornFixed = true
	rebindBankShop(bank)
	var bankProbes []sim.ItemInstance
	for _, it := range []secondshop.Item{
		{Table: secondshop.Magic, Row: 6, Count: 1, Price: 100},
		{Table: secondshop.Magic, Row: 70, Count: 1, Price: 83},
		{Table: secondshop.Armor, Row: 3, Class: 1, Material: 2, Count: 1, Price: 125},
		{Table: secondshop.Book, Spell: 1, Count: 1, Price: 1000},
	} {
		units, _ := secondShopTestTables.Units(it)
		bankProbes = append(bankProbes, units...)
	}
	got.WriteString("== bank\n" + shopBindingRecord(bank, bankProbes, c))

	vf := missionShopFront(t, 35)
	missionShopOpened(t, vf)
	visit := vf.townUI
	if !visit.MissionShopOpen() {
		t.Fatal("the Shop page did not open over the mission")
	}
	visit.sess.Town.cityObjects = &cityObjectTopology{}
	got.WriteString("== visit\n" + shopBindingRecord(visit, shopBindingChapterProbes(visit.in.Table), nil))

	if got.String() != shopBindingGolden {
		t.Fatalf("shop binding record changed:\n%s", got.String())
	}
}

// rebindChapterShop and rebindBankShop bind the page again from the first and the second game's row.
func rebindChapterShop(s *townScreen) { s.bindShop(s.chapterShopRow()) }
func rebindBankShop(s *townScreen)    { s.bindShop(s.bankShopRow()) }

const shopBindingGolden = `== chapter
rackName -1 ""
rackName 0 "armour"
rackName 1 "weapons"
rackName 2 "magic"
rackName 3 ""
rackName 4 ""
rack 0 shelf 0 stocked true name "armour"
rack 1 shelf 1 stocked true name "weapons"
rack 2 shelf 2 stocked true name "magic"
rack 3 shelf 3 stocked true name ""
probe 0 code 0x0e01 kind 0 price 0 text [] info ["0014001"] value 0
probe 1 code 0x0e02 kind 0 price 0 text [] info ["0014002"] value 0
probe 2 code 0x0e03 kind 0 price 0 text [] info ["0014003"] value 0
probe 3 code 0x0e05 kind 0 price 7 text [] info ["0014005"] value 7
probe 4 code 0x0e05 kind 0 price 0 text [] info ["0014005"] value 0
deal session shop
cityGraph true
packUnits 0
events ["merchant-yes" "merchant-no" "merchant-yes" "rack"]
hooks shown,sound,scene,speak,fresh,dress,stepped,stats,leave,page,covered,faces,face,pane
shownRoom false
wornFixed false
rules {SpeechDir: MoneyLast:true ShelfGems:true TableShelfBack:false LabelOffset:[[0 0] [0 0] [0 0] [0 0]] Cite:[SHOP-SCREEN-036 SHOP-SCREEN-033 owner]}
book true
pack [e1c]
hasActorContainer true
setPack true
packed 0 code 0x0e01 kind 0 price 0 fx 0
packed 1 code 0x0e05 kind 0 price 0 fx 0
gesture calls 1 msg "gesture"
state chosen 3 shelf 3 shelfBase 4 packBase 5 tableBase 6 member 0 book true
== bank
rackName -1 ""
rackName 0 ""
rackName 1 ""
rackName 2 ""
rackName 3 ""
rackName 4 ""
rack 0 shelf 0 stocked true name ""
rack 1 shelf 1 stocked true name ""
rack 2 shelf 2 stocked true name ""
rack 3 shelf 3 stocked true name ""
probe 0 code 0x0e06 kind 0 price 100 text ["Scroll_Fire_Arrow"] info ["Scroll of Fire Arrow"] value 100
probe 1 code 0x0e46 kind 0 price 83 text [""] info ["0014070"] value 83
probe 2 code 0x2623 kind 1 price 125 text ["" "" ""] info ["0206103"] value 125
probe 3 code 0x0e81 kind 0 price 1000 text ["Fire Arrow"] info ["Book of Fire Arrow"] value 1000
deal second campaign shop
cityGraph false
packUnits 8192
events ["bought" "refused" "sold" "category-changed"]
hooks shown,sound,scene,speak,dress,stats,leave,chose,page,faces,face,pane
shownRoom false
wornFixed false
rules {SpeechDir: MoneyLast:false ShelfGems:false TableShelfBack:false LabelOffset:[[0 0] [0 0] [0 0] [0 0]] Cite:[]}
book false
pack [e46 e46 e46 e46]
hasActorContainer true
setPack true
packed 0 code 0x0e06 kind 0 price 100 fx 0
packed 1 code 0x0e81 kind 0 price 1000 fx 1
gesture calls 1 msg "gesture"
state chosen 0 shelf 0 shelfBase 0 packBase 0 tableBase 0 member 0 book false
== visit
rackName -1 ""
rackName 0 "armour"
rackName 1 "weapons"
rackName 2 "magic"
rackName 3 ""
rackName 4 ""
rack 0 shelf 0 stocked true name "armour"
rack 1 shelf 1 stocked true name "weapons"
rack 2 shelf 2 stocked true name "magic"
rack 3 shelf 3 stocked true name ""
probe 0 code 0x0e01 kind 0 price 0 text [] info ["0014001"] value 0
probe 1 code 0x0e02 kind 0 price 0 text [] info ["0014002"] value 0
probe 2 code 0x0e03 kind 0 price 0 text [] info ["0014003"] value 0
probe 3 code 0x0e05 kind 0 price 7 text [] info ["0014005"] value 7
probe 4 code 0x0e05 kind 0 price 0 text [] info ["0014005"] value 0
deal session shop
cityGraph true
packUnits 0
events ["merchant-yes" "merchant-no" "merchant-yes" "rack"]
hooks shown,sound,scene,speak,fresh,dress,stepped,stats,leave,page,covered,faces,face,pane
shownRoom true
wornFixed true
rules {SpeechDir: MoneyLast:true ShelfGems:true TableShelfBack:false LabelOffset:[[0 0] [0 0] [0 0] [0 0]] Cite:[SHOP-SCREEN-036 SHOP-SCREEN-033 owner]}
book true
pack [e1c e03]
hasActorContainer true
setPack true
packed 0 code 0x0e01 kind 0 price 0 fx 0
packed 1 code 0x0e05 kind 0 price 0 fx 0
gesture calls 1 msg "gesture"
state chosen 2 shelf 2 shelfBase 0 packBase 0 tableBase 0 member 0 book false
`
