package game

import (
	"fmt"
	"slices"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// hireRow is one hire of a game's hire rows: the identity a hired member
// carries, as its tavern type when tavern holds and as its campaign hire
// number otherwise, the row arm its actor resolves through, and the bank
// slots of its selection, availability and town. A game without a hire bank
// holds the slots 0.
type hireRow struct {
	identity                      int
	tavern                        bool
	arm                           mapload.CampaignActorProfile
	selection, availability, town int
}

// holds reports whether m is a member hired by r.
func (r hireRow) holds(m mapload.PartyMember) bool {
	if r.tavern {
		return m.Membership().Matches(mapload.TavernMember, r.identity)
	}
	return m.Membership().Matches(mapload.CampaignMember, r.identity)
}

// hireMoment is when the hire producer builds a hired member.
type hireMoment int

const (
	// hireAtOffer builds the members when the hire is accepted.
	hireAtOffer hireMoment = iota
	// hireAtEntry builds them at mission entry from the rows the bank selects.
	hireAtEntry
)

// hireCost is what a hire is paid with.
type hireCost int

const (
	// hireGold charges a price from the purse and refunds it at dismissal.
	hireGold hireCost = iota
	// hireBankBit is the row's selection word, which the word's TalkTo toggles.
	hireBankBit
)

// hireState is the campaign state hire rows are read over: the town whose
// pools, flags and chapter the first game's rows read, the hire bank the
// second game's rows read and the town the party stands in, and the actor
// table.
type hireState struct {
	town  *Town
	bank  *[innBankSlots]int32
	here  int
	table *mapload.Table
}

func (c *secondCampaign) hireState() hireState {
	return hireState{bank: &c.bank, here: c.current.id}
}

// hireProfile is one game's hire as data: its rows, the moment the producer
// builds a member, the cost source, the walk the offers take, the stage a
// row's arm reads, what offers a row and what selects it, which rows march
// at mission entry, and for a squad game the squad size and price.
type hireProfile struct {
	rows       []hireRow
	moment     hireMoment
	cost       hireCost
	wornSpells bool
	walk       func(s hireState) []int
	stage      func(s hireState) int
	offered    func(s hireState, r hireRow) int
	selected   func(s hireState, r hireRow) bool
	marches    func(s hireState, r hireRow) bool
	squad      func(s hireState, r hireRow) (count, capacity int)
	price      func(s hireState, r hireRow, count int) (int, bool)
}

// row is the hire row of identity id.
func (p *hireProfile) row(id int) (hireRow, bool) {
	if id < 1 || id > len(p.rows) {
		return hireRow{}, false
	}
	return p.rows[id-1], true
}

// tavernHires is the first game's hire: mercenary types 1 and 2 are the
// Units rows of the Catapult and the Ballista, types 3..15 the Humans rows
// NPC<type>_<level>. A hire takes the type's whole pool at the squad price
// from the purse, and the tavern builds the squad as it is hired.
var tavernHires = hireProfile{
	rows: func() []hireRow {
		profile := mapload.TableEdition(nil)
		types := profile.TavernTypes()
		rows := make([]hireRow, len(types))
		for i := range rows {
			typ := types[i]
			arm := mapload.CampaignActorProfile{HumanName: "NPC%02d_%d", Temporary: true, RowFigure: true, RowRotation: true, RowDefinition: true}
			if profile.TavernUnit(typ) {
				arm = mapload.CampaignActorProfile{UnitName: siegeHireName(typ), Temporary: true}
			}
			rows[i] = hireRow{identity: typ, tavern: true, arm: arm}
		}
		return rows
	}(),
	moment: hireAtOffer, cost: hireGold, wornSpells: true,
	walk:  func(s hireState) []int { return tavernStockWalk(s.town.tavernStockIDs) },
	stage: func(s hireState) int { return mercenaryLevel(s.town.Chapter()) },
	offered: func(s hireState, r hireRow) int {
		t, typ := s.town, r.identity
		if !slices.Contains(t.ChapterData().Mercenaries, typ) || !t.MercenaryEnabled(typ) || t.mercenaryHiredElsewhere(typ) || t.MercenaryPool(typ) <= 0 {
			return 0
		}
		return 1
	},
	selected: func(s hireState, r hireRow) bool { return s.town.MercenaryHired(r.identity) },
	squad: func(s hireState, r hireRow) (int, int) {
		count := s.town.MercenaryPool(r.identity)
		return count, max(s.town.MercenaryCapacity(r.identity), count)
	},
	price: func(s hireState, r hireRow, count int) (int, bool) {
		return mercenarySquadPrice(s.table, s.town.Chapter(), r.identity, count)
	},
}

// The hire bank: hire h (1..20) has selection slot 511+h, availability slot
// 531+h and town slot 551+h (R2-ENGINE-221, R2-ENGINE-380).
const (
	secondHireSlots        = 20
	secondHireSelection    = 512
	secondHireAvailability = 532
	secondHireTown         = 552
)

// bankHires is the second game's hire: hire h resolves through the Humans
// or Units row keyed 10000+10h and its stage variant. A hire toggles the
// row's selection word, and mission entry builds each hire whose selection
// and availability are set. An offer is a hire standing in the party's town
// whose availability is 1 (a hire) or 2 (a refusal).
var bankHires = hireProfile{
	rows: func() []hireRow {
		rows := make([]hireRow, secondHireSlots)
		for i := range rows {
			rows[i] = hireRow{identity: i + 1, selection: secondHireSelection + i, availability: secondHireAvailability + i, town: secondHireTown + i,
				arm: mapload.CampaignActorProfile{ServerBase: 10000, ServerStride: 10, StageDivisor: 10, StageBias: -2,
					Temporary: true, RowFigure: true, RowRotation: true, RowDefinition: true}}
		}
		return rows
	}(),
	moment: hireAtEntry, cost: hireBankBit,
	stage: func(s hireState) int { return int(s.bank[768]) },
	offered: func(s hireState, r hireRow) int {
		if s.bank == nil {
			return 0
		}
		kind := s.bank[r.availability]
		if int(s.bank[r.town]) != s.here || kind != 1 && kind != 2 {
			return 0
		}
		return int(kind)
	},
	selected: func(s hireState, r hireRow) bool { return s.bank[r.selection] > 0 },
	marches:  func(s hireState, r hireRow) bool { return s.bank[r.selection] != 0 && s.bank[r.availability] != 0 },
}

// hireMembers is the one hire producer: count members of profile p's row
// id, each resolved through the row's arm at the stage p reads over s.
func hireMembers(p *hireProfile, s hireState, id, count int) ([]mapload.PartyMember, bool) {
	r, ok := p.row(id)
	if !ok || s.table == nil || count <= 0 {
		return nil, false
	}
	spec := mapload.CampaignActorSpec{Profile: r.arm, Identity: r.identity, Stage: p.stage(s), Hire: !r.tavern}
	if r.tavern {
		spec.MercenaryType = uint8(r.identity)
	}
	base, ok := mapload.CampaignActor(s.table, spec, nil)
	if !ok {
		return nil, false
	}
	if p.wornSpells {
		joinedWornSpells(base.WornItems[:], s.table)
	}
	out := make([]mapload.PartyMember, count)
	for i := range out {
		out[i] = base
	}
	return out, true
}

// hireOffers is the hire options of the one offer list: each row p offers
// over s, in p's walk, as an option of the kind p offers it with the row's
// identity as NPC key and its selection as the high bit; a squad option
// carries its size, capacity, price and whether the purse affords it.
func hireOffers(p *hireProfile, s hireState) []innOffer {
	walk := p.rows
	if p.walk != nil {
		walk = nil
		for _, id := range p.walk(s) {
			if r, ok := p.row(id); ok {
				walk = append(walk, r)
			}
		}
	}
	var out []innOffer
	for _, r := range walk {
		kind := p.offered(s, r)
		if kind == 0 {
			continue
		}
		o := innOffer{kind: kind, npc: r.identity, high: p.selected(s, r)}
		if p.squad != nil {
			o.count, o.capacity = p.squad(s, r)
			price, ok := p.price(s, r, o.count)
			if !ok {
				continue
			}
			o.price, o.affordable = price, o.high || price <= s.town.Gold()
		}
		out = append(out, o)
	}
	return out
}

// enterHires is mission entry's hire moment under campaign service rules,
// which name the hire profile and read town's hire state. Under a profile
// whose moment is mission entry, the party keeps each hired member whose
// row marches, once, and gains a member of each marching row it lacks, in
// row order; a town without a hire bank, or another profile, enters
// unchanged.
func enterHires(rules campaignService, party []mapload.PartyMember, town *Town, table *mapload.Table) ([]mapload.PartyMember, error) {
	p, s := rules.files().hire, rules.hires(town, table)
	if p == nil || p.moment != hireAtEntry || s.bank == nil {
		return party, nil
	}
	out := make([]mapload.PartyMember, 0, len(party)+len(p.rows))
	seen := make([]bool, len(p.rows)+1)
	for _, m := range party {
		if i := slices.IndexFunc(p.rows, func(r hireRow) bool { return r.holds(m) }); i >= 0 {
			if seen[i+1] || !p.marches(s, p.rows[i]) {
				continue
			}
			seen[i+1] = true
		}
		out = append(out, m)
	}
	for _, r := range p.rows {
		if seen[r.identity] || !p.marches(s, r) {
			continue
		}
		members, ok := hireMembers(p, s, r.identity, 1)
		if !ok {
			return nil, fmt.Errorf("campaign hire %d at stage %d has no actor template", r.identity, p.stage(s))
		}
		out = append(out, members[0])
	}
	return mapload.OwnParty(out), nil
}

// fallenHires is the identities of the hires in party whose actor, ids[i]
// in w, fell below -10 health or has left the world (R2-SESSION-195).
func fallenHires(party []mapload.PartyMember, ids []sim.EntityID, w *sim.World) []int {
	var out []int
	for i, m := range party {
		r := slices.IndexFunc(bankHires.rows, func(r hireRow) bool { return r.holds(m) })
		if r < 0 || i >= len(ids) || w == nil {
			continue
		}
		if e, ok := w.Entity(ids[i]); !ok || e.InLife(sim.LifeFinished) {
			out = append(out, bankHires.rows[r].identity)
		}
	}
	return out
}

// departHires is the hire part of a won mission's ordinary departure over
// the hire bank: a fallen hire's availability clears first (DIV-4645), then
// a hire left available becomes 1, a selected one 2, and every selection
// clears.
func departHires(bank *[innBankSlots]int32, fallen []int) {
	for _, r := range bankHires.rows {
		if slices.Contains(fallen, r.identity) {
			sim.SetScenario(bank, int64(r.availability), 0)
		}
		if bank[r.availability] != 0 {
			sim.SetScenario(bank, int64(r.availability), 1)
			if bank[r.selection] != 0 {
				sim.SetScenario(bank, int64(r.availability), 2)
			}
		}
		sim.SetScenario(bank, int64(r.selection), 0)
	}
}

// hireDeal is one hire or dismissal of either game. Its cost source is a
// gold price from purse, or the bank bit word's TalkTo on talk toggles; held
// is the pool's hired flag; party is the list a hire adds members to and a
// dismissal returns row's members from; seat is the page's copy of the
// word's high bit by low ID npc. A hire answers caption, or refusal and no
// change when refusal is set.
type hireDeal struct {
	fire    bool
	cost    hireCost
	purse   *int
	price   int
	talk    *secondCampaign
	word    innOffer
	held    *bool
	party   *[]mapload.PartyMember
	members []mapload.PartyMember
	row     hireRow
	seat    *map[int]bool
	npc     int
	caption string
	refusal string
}

// run is the one hire and dismissal transaction. It answers the caption,
// whether the deal ran, and the IDs of the members a dismissal returned. A
// deal holding a flag runs only when the flag stands opposite the deal and,
// for a dismissal, the party holds the row's members; a gold hire only when
// the purse holds the price.
func (d hireDeal) run() (string, bool, []string) {
	if d.seat != nil && *d.seat == nil {
		*d.seat = map[int]bool{}
	}
	if !d.fire && d.refusal != "" {
		return d.refusal, false, nil
	}
	if d.cost == hireGold && (d.purse == nil || !d.fire && (d.price < 0 || *d.purse < d.price)) {
		return "", false, nil
	}
	if d.held != nil && *d.held != d.fire {
		return "", false, nil
	}
	var returned []string
	if d.party != nil && d.fire {
		for _, m := range *d.party {
			if d.row.holds(m) {
				returned = append(returned, m.ID)
			}
		}
	}
	if d.held != nil && d.fire && len(returned) == 0 {
		return "", false, nil
	}
	if d.held != nil {
		*d.held = !d.fire
	}
	if d.seat != nil {
		d.word.high = !d.fire
		(*d.seat)[d.npc] = !d.fire
	}
	switch {
	case d.cost == hireGold && d.fire:
		*d.purse += d.price
	case d.cost == hireGold:
		*d.purse -= d.price
	case d.talk != nil:
		d.talk.talkTo(d.word)
	}
	if d.party != nil {
		if d.fire {
			kept := (*d.party)[:0]
			for _, m := range *d.party {
				if !d.row.holds(m) {
					kept = append(kept, m)
				}
			}
			*d.party = kept
		} else {
			*d.party = mapload.OwnParty(append(*d.party, d.members...))
		}
	}
	if d.fire {
		return "", true, returned
	}
	return d.caption, true, returned
}
