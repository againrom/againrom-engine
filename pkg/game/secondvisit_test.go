package game

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

var secondVisitTowns = []struct {
	name       string
	shop, inn  uint32
	town       int
	innOffered int
}{
	{"town", 34, 67, 1, 207},
	{"Kaarg", 93, 100, 2, 207},
	{"Druid", 107, 113, 3, 207},
}

func TestSessionSecondShopOpensTheKindsTownPage(t *testing.T) {
	for _, tc := range secondVisitTowns {
		t.Run(tc.name, func(t *testing.T) {
			v := visitEnter(t, visitCasesOf(tc.shop)[1])
			s, c := v.f.townUI, v.f.townUI.state()
			if !s.MissionShopOpen() || s.roomName() != "shop" || c.current.id != tc.town {
				t.Fatalf("page open %v room %q town %d, want the shop of town %d", s.MissionShopOpen(), s.roomName(), c.current.id, tc.town)
			}
			deal := s.deal()
			stocked := 0
			for _, shelf := range deal.shelves {
				stocked += len(shelf)
			}
			if deal.location != tc.town || deal.filled != tc.town || *deal.records[tc.town] != secondshop.NewGameRecords[tc.town] {
				t.Fatalf("stock for town %d filled %d from record %+v", deal.location, deal.filled, *deal.records[tc.town])
			}
			if tc.town < 3 && stocked == 0 {
				t.Fatalf("the shop of town %d holds nothing", tc.town)
			}
			if e := v.leave(t); e.X != v.cell.X || e.Y != v.cell.Y {
				t.Fatalf("left at %d,%d, want his own cell %v", e.X, e.Y, v.cell)
			}
		})
	}
}

func TestSessionSecondShopBuyMovesTheUnitAndThePriceAndWrites(t *testing.T) {
	now := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	sessionNow = func() time.Time { return now }
	t.Cleanup(func() { sessionNow = nil })
	c := visitCasesOf(34)[1]
	v := visitEnter(t, c)
	s, f := v.f.townUI, v.f
	tables, ok := secondShopTables(f.Table)
	if !ok {
		t.Fatal("the fixture's item tables do not make shop tables")
	}
	s.shop.shown = nil
	deal := s.deal()
	deal.shelves[0] = secondShopItems(tables, []secondshop.Item{{Table: secondshop.Magic, Row: 70, Count: 4, Price: 83}})
	path := filepath.Join(v.dir, c.file)
	old := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	written := func() bool {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		return !fi.ModTime().Equal(old)
	}
	s.click(ui.ShopControlShelfPick, 0)
	for i, tc := range []struct {
		after time.Duration
		write bool
	}{{0, true}, {10 * time.Second, false}, {16 * time.Second, true}} {
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
		now = now.Add(tc.after)
		packBefore, purse := missionShopPack(f, v.hero)
		gold := s.sess.Town.gold
		missionShopBuy(s)
		pack, after := missionShopPack(f, v.hero)
		if len(*s.visit.pack) != len(pack) {
			t.Fatalf("buy %d: the page's bag holds %d units, the world's pack %d", i+1, len(*s.visit.pack), len(pack))
		}
		if len(pack) != len(packBefore)+1 || pack[len(pack)-1] != 0x0e00|70 {
			t.Fatalf("buy %d: pack %d -> %d, purse %d -> %d", i+1, len(packBefore), len(pack), purse, after)
		}
		if got := purse - after; got == 0 || int(got) != gold-s.sess.Town.gold {
			t.Fatalf("buy %d: purse fell %d, the page's gold fell %d", i+1, got, gold-s.sess.Town.gold)
		}
		if got := written(); got != tc.write {
			t.Fatalf("deal %d at +%v: file written %v, want %v", i+1, tc.after, got, tc.write)
		}
	}
}

func TestSessionSecondInnListsTheSinglePlayerOffers(t *testing.T) {
	for _, tc := range secondVisitTowns {
		t.Run(tc.name, func(t *testing.T) {
			v := visitEnter(t, visitCasesOf(tc.inn)[1])
			s, c := v.f.townUI, v.f.townUI.state()
			if !s.MissionShopOpen() || s.roomName() != "inn" || c.current.id != tc.town {
				t.Fatalf("page open %v room %q town %d, want the inn of town %d", s.MissionShopOpen(), s.roomName(), c.current.id, tc.town)
			}
			want := fmt.Sprint(innAt(tc.town, c.bank[768], nil).innOptions())
			if got := fmt.Sprint(c.innOptions()); got != want || len(c.innOptions()) == 0 {
				t.Fatalf("offers %s, want the single-player offers %s", got, want)
			}
			listed := false
			for _, r := range s.Rows() {
				listed = listed || strings.Contains(r.Text, fmt.Sprint(tc.innOffered))
			}
			if !listed {
				t.Fatalf("the page's rows %v list no offer of NPC %d", s.Rows(), tc.innOffered)
			}
			var o innOffer
			for _, offer := range c.innOptions() {
				if offer.kind == town.OfferMission && o.kind == 0 {
					o = offer
				}
			}
			if o.kind != town.OfferMission {
				t.Fatalf("town %d offers no mission", tc.town)
			}
			ref := *c
			ref.available = slices.Clone(c.available)
			ref.talkTo(o)
			if ref.bank == c.bank && len(ref.available) == len(c.available) {
				t.Fatalf("offer %+v changes nothing in the single-player rule", o)
			}
			s.startInnTalk(secondInnProfile(), innTalkRequest{building: TownTavern, npc: o.npc, word: &o, key: secondTalkKey(o.npc, o.topic),
				payload: []byte("<NPC=1,PART=1>\r\nWord.\r\n"), loaded: true})
			if c.bank != ref.bank || !slices.Equal(c.available, ref.available) {
				t.Fatalf("taking offer %+v: available %v, the single-player rule gives %v", o, c.available, ref.available)
			}
			for range 4 {
				s.AdvanceTownDialogue()
			}
			if e := v.leave(t); e.X != v.cell.X || e.Y != v.cell.Y {
				t.Fatalf("left at %d,%d, want his own cell %v", e.X, e.Y, v.cell)
			}
			if kept := secondOf(v.f.townUI.sess); kept.bank != ref.bank || !slices.Equal(kept.available, ref.available) {
				t.Fatalf("the session kept available %v, want %v", kept.available, ref.available)
			}
		})
	}
}

func TestSecondGameCampaignShopUseOpensNoMissionPage(t *testing.T) {
	v := visitEnter(t, visitCasesOf(34)[1])
	mw, s := v.f.live, v.f.townUI
	v.leave(t)
	mw.mission.buildings.visits = false
	mw.shop.customer, mw.shop.waiting, mw.shop.class = v.hero, true, sim.VisitShop
	if s.OpenMissionShop() || s.MissionShopOpen() {
		t.Fatal("a map without the session's building rule opened the second game's Shop page")
	}
}

func TestRefusedOpenLeavesNoCustomerWaiting(t *testing.T) {
	v := visitEnter(t, visitCasesOf(34)[1])
	mw, s := v.f.live, v.f.townUI
	v.leave(t)
	town := s.sess.Town
	s.sess.Town = nil
	mw.shop.customer, mw.shop.waiting, mw.shop.class = v.hero, true, sim.VisitShop
	opened := s.OpenMissionShop()
	s.sess.Town = town
	if opened || mw.shop.waiting {
		t.Fatalf("opened %v, customer still waiting %v", opened, mw.shop.waiting)
	}
}
