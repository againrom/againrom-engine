package game

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/herokind"
	"againrom/pkg/mapload"
	"againrom/pkg/town"
)

// The second game's four-way inn records go through the one chooser by the
// starting hero: the catalog reads the hero in the slots the commit writes
// (R2-ENGINE-283), each family's rows name four kinds that do not overlap,
// and every value of slots 776 and 781 offers the record R2-ENGINE-217 and
// R2-ENGINE-424 name.
func TestSecondInnFamiliesChooseByTheStartingHero(t *testing.T) {
	slots := secondBook().catalog.HeroSlots
	g := secondGeneratorCampaigns["rom2"]
	if slots == nil || slots.Mage != g.MageSlot || slots.Female != g.FemaleSlot {
		t.Fatalf("catalog hero slots %+v, generator %d and %d", slots, g.MageSlot, g.FemaleSlot)
	}
	rows := secondBook().offerRows()
	row := func(topic int) town.OfferRow {
		i := slices.IndexFunc(rows, func(r town.OfferRow) bool { return r.Topic == topic })
		if i < 0 {
			t.Fatalf("no row of topic %d", topic)
		}
		return rows[i]
	}
	f := func(topic int) []innOffer { return []innOffer{{kind: 3, topic: topic, npc: 2022}} }
	// Each case is the value of 776 and 781 and its index in a family, in
	// R2-ENGINE-217's order zero/zero, zero/nonzero, nonzero/nonzero,
	// nonzero/zero.
	cases := []struct {
		mage, female int32
		at           int
	}{{0, 0, 0}, {0, 1, 1}, {0, -2, 1}, {1, 1, 2}, {-1, 7, 2}, {1, 0, 3}, {-1, 0, 3}}
	for _, fam := range []struct {
		gate   map[int]int32
		topics []int
	}{
		{map[int]int32{959: 1}, []int{74, 75, 76, 77}},
		{map[int]int32{970: 1}, []int{84, 85, 86, 87}},
		{map[int]int32{980: 1}, []int{93, 94, 95, 96}},
	} {
		var sets []herokind.Set
		for _, topic := range fam.topics {
			sets = append(sets, rowHeroes(row(topic)))
		}
		if i, j, ok := herokind.Overlap(sets); ok {
			t.Fatalf("family %v: records %d and %d overlap", fam.topics, fam.topics[i], fam.topics[j])
		}
		for _, c := range cases {
			bank := map[int]int32{776: c.mage, 781: c.female}
			for s, v := range fam.gate {
				bank[s] = v
			}
			in := innAt(2, 20, bank)
			if got := herokind.Choose(mapload.StartingHeroKind(mapload.HeroHold{Slots: slots, Bank: in.bank[:]}), sets, -1); got != c.at {
				t.Errorf("family %v, 776=%d 781=%d: chooser takes %d, want %d", fam.topics, c.mage, c.female, got, c.at)
			}
			if got, want := in.innOptions(), f(fam.topics[c.at]); !reflect.DeepEqual(got, want) {
				t.Errorf("family %v, 776=%d 781=%d: offered %v, want %v", fam.topics, c.mage, c.female, got, want)
			}
		}
	}
	// Records 81 and 82 split by class alone (R2-ENGINE-424).
	if i, j, ok := herokind.Overlap([]herokind.Set{rowHeroes(row(81)), rowHeroes(row(82))}); ok {
		t.Fatalf("records 81 and 82 overlap at %d and %d", i, j)
	}
	for _, c := range cases {
		want := 82
		if c.mage != 0 {
			want = 81
		}
		var topics []int
		for _, o := range innAt(2, 80, map[int]int32{776: c.mage, 781: c.female}).innOptions() {
			if o.topic == 81 || o.topic == 82 {
				topics = append(topics, o.topic)
			}
		}
		if !slices.Equal(topics, []int{want}) {
			t.Errorf("776=%d 781=%d at stage 80: records %v, want %d", c.mage, c.female, topics, want)
		}
	}
}

// rowHeroes is the set of starting heroes an inn row is offered to.
func rowHeroes(r town.OfferRow) herokind.Set { return herokind.Named(r.Class, r.Sex) }

// The bank's hero slots read the four kinds, no slots read every row's
// hero, and a row whose class or sex names no hero is refused by its key.
func TestInnRowHeroesAreReadByTheChooser(t *testing.T) {
	slots := &town.HeroSlots{Mage: 5, Female: 6}
	var bank [innBankSlots]int32
	for _, tc := range []struct {
		mage, female int32
		want         herokind.Kind
	}{{0, 0, herokind.MaleFighter}, {0, 3, herokind.FemaleFighter}, {-1, 1, herokind.FemaleMage}, {1, 0, herokind.MaleMage}} {
		bank[5], bank[6] = tc.mage, tc.female
		if got := mapload.StartingHeroKind(mapload.HeroHold{Slots: slots, Bank: bank[:]}); got != tc.want {
			t.Errorf("bank %d/%d reads %v, want %v", tc.mage, tc.female, got, tc.want)
		}
	}
	if got := mapload.StartingHeroKind(mapload.HeroHold{Bank: bank[:]}); got != herokind.MaleFighter {
		t.Errorf("no slots read %v", got)
	}
	mage := town.OfferRow{Kind: 3, Topic: 7, NPC: 40, Class: "mage", Sex: "female"}
	s := innState{bank: &bank, hero: herokind.FemaleMage}
	if !s.admits(mage) || !s.admits(town.OfferRow{Kind: 3, Topic: 8, NPC: 40}) {
		t.Fatal("a female mage is not offered her row or a row for every hero")
	}
	if s.hero = herokind.MaleMage; s.admits(mage) {
		t.Fatal("a male mage is offered the female mage's row")
	}
	for _, r := range []town.OfferRow{{Topic: 1, NPC: 1, Class: "rogue"}, {Topic: 1, NPC: 1, Sex: "Male"}} {
		err := checkOfferRows([]town.OfferRow{r}, town.InnTable)
		var oe *town.OfferError
		if err == nil || !errors.As(err, &oe) || oe.Row != town.InnTable.Key(r) || !strings.Contains(err.Error(), "is neither") {
			t.Errorf("%+v: %v", r, err)
		}
	}
	if err := checkOfferRows([]town.OfferRow{mage}, town.InnTable); err != nil {
		t.Fatal(err)
	}
}
