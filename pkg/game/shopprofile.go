package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/shopstock"
	"againrom/pkg/town"
)

type shopProfile struct {
	shown                                              func(*townScreen) bool
	missionVisits                                      bool
	fillOnArrival                                      bool
	resetCategoryOnFill                                bool
	binding                                            func(*townScreen) shopBinding
	wholeBasket, unpricedBack, keepEmptied             bool
	fallback                                           ShopShelf
	split                                              func(ShopItem, int64) (ShopItem, ShopItem)
	home                                               func(*Shop, ShopPlace) ShopShelf
	shelve                                             func(*Shop, ShopItem) ShopShelf
	put                                                func(*Shop, []ShopItem, ShopItem) []ShopItem
	order                                              func(*Shop) func([]ShopItem) []int
	join                                               func(ShopItem, ShopItem) bool
	fill                                               func(*Shop, *mapload.Table, shopDrawSource) [shopShelfCount][]ShopItem
	rawStream, ordinalSeed, clickEpoch, derivedRecords bool
	blockRestock, refillOnEntry, undoOnEntry           bool
	departures, stages, unlocks                        []shopRecordPatch
}

var chapterShopProfile = shopProfile{
	shown: func(t *townScreen) bool { return t.displayRoom() == "shop" }, missionVisits: true,
	fillOnArrival: true,
	binding:       (*townScreen).chapterShopRow,
	wholeBasket:   true, fallback: ShelfMagic,
	ordinalSeed: true, clickEpoch: true, derivedRecords: true, blockRestock: true,
	split: func(i ShopItem, n int64) (ShopItem, ShopItem) {
		moved, rest := i, i
		moved.Count, rest.Count = int32(n), i.Count-int32(n)
		moved.Owners, rest.Owners = takeShopOwners(i.Owners, int32(n))
		return moved, rest
	},
	home:   func(_ *Shop, p ShopPlace) ShopShelf { return p.From },
	shelve: func(s *Shop, i ShopItem) ShopShelf { return shopShelfForItem(i, s.tbl) },
	put: func(s *Shop, items []ShopItem, i ShopItem) []ShopItem {
		i.Owners = nil
		if i.Count <= 0 {
			return items
		}
		return shopstock.Insert(items, shopShelfStack(s.tbl), i)
	},
	order: func(*Shop) func([]ShopItem) []int { return nil },
	fill: func(s *Shop, t *mapload.Table, r shopDrawSource) (out [shopShelfCount][]ShopItem) {
		copy(out[:], shopstock.Generate(s.stockProfile(t, r), r))
		return
	},
}

var bankShopProfile = shopProfile{
	shown:        (*townScreen).bankShopShown,
	blockRestock: true, resetCategoryOnFill: true,
	binding:      (*townScreen).bankShopRow,
	wholeBasket:  true,
	unpricedBack: true, keepEmptied: true, fallback: ShelfBooks,
	rawStream: true, refillOnEntry: true, undoOnEntry: true,
	split: func(i ShopItem, n int64) (ShopItem, ShopItem) {
		moved, rest := i, i
		moved.Count, rest.Count = int32(n), i.Count-int32(n)
		return moved, rest
	},
	home:   func(s *Shop, p ShopPlace) ShopShelf { return bankShopPlace(s, p.ShopItem) },
	shelve: bankShopPlace,
	put: func(s *Shop, items []ShopItem, i ShopItem) []ShopItem {
		var shelf []ShopItem
		for _, e := range items {
			if _, ok := secondShopItem(s.tables, secondItem(e)); ok {
				shelf = append(shelf, e.Clone())
			}
		}
		if it, ok := secondShopItem(s.tables, secondItem(i)); ok {
			shelf = shopstock.Insert(shelf, secondshop.Stacking, it)
		}
		return shelf
	},
	order: func(s *Shop) func([]ShopItem) []int {
		return func(items []ShopItem) []int { return s.tables.Order(secondItems(items)) }
	},
	join: func(a, b ShopItem) bool { return secondshop.Stacking.Merges(secondItem(a), secondItem(b)) },
	fill: func(s *Shop, _ *mapload.Table, src shopDrawSource) [shopShelfCount][]ShopItem {
		return stockOf(s.tables, s.tables.Fill(*s.records[s.location], secondshop.Rand(src.(shopRawDraw))))
	},
	departures: bankShopDepartures(), stages: bankShopStages(),
	unlocks: []shopRecordPatch{{record: 3, category: 3, field: shopRecordMaskOr, value: secondShopClass4}},
}

func (s *Shop) profile() *shopProfile {
	if s == nil || s.policy == nil {
		return &chapterShopProfile
	}
	return s.policy
}

// hostShopProfiles are the profiles the town host's shop mode names.
var hostShopProfiles = map[town.ShopMode]*shopProfile{town.ShopChapter: &chapterShopProfile, town.ShopBank: &bankShopProfile}

// townShopProfile is the profile the town's shop room shows and trades by:
// the campaign's town shop's own, and while the campaign holds no shop yet,
// as a town screen bound before NEW GAME or LOAD does, the one the town
// host's shop mode names.
func (t *townScreen) townShopProfile() *shopProfile {
	if s := t.campaignTown().townShop(); s != nil {
		return s.profile()
	}
	if p := hostShopProfiles[t.hostRules().Shop]; p != nil {
		return p
	}
	return &chapterShopProfile
}

func newShopState(p *shopProfile, records shopRecords) Shop {
	return Shop{policy: p, records: records.clone()}
}

type shopRecords map[int]*secondshop.Record

func shopRecordRows(native [4]secondshop.Record) shopRecords {
	out := make(shopRecords, len(native))
	for id, record := range native {
		out[id] = &record
	}
	return out
}

func (r shopRecords) clone() shopRecords {
	if r == nil {
		return nil
	}
	out := make(shopRecords, len(r))
	for id, record := range r {
		value := *record
		out[id] = &value
	}
	return out
}

func (s *Shop) nativeRecords() (out [4]secondshop.Record) {
	for id := range out {
		if record := s.records[id]; record != nil {
			out[id] = *record
		}
	}
	return
}

func (s *Shop) deriveRecords(ids []int) {
	if s.records == nil {
		s.records = make(shopRecords)
	}
	for _, id := range ids {
		if s.records[id] == nil {
			s.records[id] = new(secondshop.Record)
		}
	}
	for id := range s.records {
		for category, draws := range shopShelfDraws {
			s.records[id][category] = secondshop.Category{Max: s.ceiling, Draws: int32(draws)}
		}
	}
}

func bankShopPlace(s *Shop, i ShopItem) ShopShelf {
	var record secondshop.Record
	if current := s.records[s.filled]; current != nil {
		record = *current
	}
	return ShopShelf(secondshop.Place(record, secondItem(i)))
}

type shopRawDraw secondshop.Rand

func (r shopRawDraw) Intn(n int) int { return (r() * n) >> 15 }

type shopRecordField uint8

const (
	shopRecordMin shopRecordField = iota
	shopRecordMax
	shopRecordDraws
	shopRecordBound
	shopRecordMask
	shopRecordMaskOr
)

type shopRecordPatch struct {
	event            int32
	record, category int
	field            shopRecordField
	value            int32
}

func (s *Shop) patch(rows []shopRecordPatch, event int32) {
	var before secondshop.Record
	if current := s.records[s.filled]; current != nil {
		before = *current
	}
	for _, row := range rows {
		if row.event != event || row.record < 0 || s.records[row.record] == nil {
			continue
		}
		c := &s.records[row.record][row.category]
		switch row.field {
		case shopRecordMin:
			c.Min = row.value
		case shopRecordMax:
			c.Max = row.value
		case shopRecordDraws:
			c.Draws = row.value
		case shopRecordBound:
			c.Bound = row.value
		case shopRecordMask:
			c.Mask = uint32(row.value)
		case shopRecordMaskOr:
			c.Mask |= uint32(row.value)
		}
	}
	if s.filled > 0 && s.records[s.filled] != nil && before != *s.records[s.filled] {
		s.filled = 0
	}
}

func bankShopDepartures() []shopRecordPatch {
	var out []shopRecordPatch
	for _, row := range []struct {
		mission int32
		record  int
		masks   [4]uint32
	}{
		{30, 2, [4]uint32{0x1383c22c, 0x2bc3c324, 0, 0x10438324}},
		{40, 2, [4]uint32{0x1387c26c, 0x2bc7c324, 0, 0x10478264}},
		{50, 3, [4]uint32{0x13c7d318, 0x2bc7d318, 0x04000000, 0x2bc042f0}},
		{60, 2, [4]uint32{0x1387c268, 0x2bc7c268, 0, 0x10478260}},
		{80, 2, [4]uint32{0x138752e0, 0x2bc752e0, 0, 0x104702e0}},
		{90, 2, [4]uint32{0x138772c0, 0x2bc772c0, 0, 0x104722c0}},
	} {
		for category, mask := range row.masks {
			if mask != 0 {
				out = append(out, shopRecordPatch{row.mission, row.record, category, shopRecordMask, int32(mask)})
			}
		}
	}
	for c := 0; c < 4; c++ {
		least, draws, bound := int32(499), int32(100), int32(2)
		if c == 1 || c == 2 {
			draws, bound = 20, 1
		}
		if c == 2 {
			least = 0
		}
		out = append(out, shopRecordPatch{50, 3, c, shopRecordMin, least}, shopRecordPatch{50, 3, c, shopRecordDraws, draws}, shopRecordPatch{50, 3, c, shopRecordBound, bound})
	}
	return out
}

func bankShopStages() []shopRecordPatch {
	var out []shopRecordPatch
	for _, row := range []struct {
		stage, maximum int32
		entries        int
	}{
		{30, 10000, 4}, {40, 22000, 4}, {50, 60000, 8}, {60, 150000, 8}, {70, 400000, 8},
		{80, 800000, 8}, {90, 1500000, 8}, {100, 5000000, 8}, {110, 10000000, 8},
	} {
		for i := 0; i < row.entries; i++ {
			out = append(out, shopRecordPatch{row.stage, 2 + i/4, i % 4, shopRecordMax, row.maximum})
		}
	}
	for _, row := range []struct{ stage, least int32 }{{90, 12000}, {100, 40000}} {
		for _, category := range []int{0, 1, 3} {
			out = append(out, shopRecordPatch{row.stage, 2, category, shopRecordMin, row.least})
		}
	}
	return out
}
