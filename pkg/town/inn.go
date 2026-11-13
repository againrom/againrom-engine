package town

import (
	"fmt"
	"slices"
)

// OfferRow is one tavern or inn option: kind, topic and NPC record, and the
// predicates that admit it; an empty predicate admits. Kind 3 is a talk
// that makes mission Topic available.
type OfferRow struct {
	Kind  int `json:"kind"`
	Topic int `json:"topic"`
	NPC   int `json:"npc"`
	// Town is the town the row is offered in; 0 offers it in every town.
	Town int `json:"town,omitempty"`
	// Stages are the stages the row is offered at; empty offers it at
	// every stage.
	Stages []int `json:"stages,omitempty"`
	// After, when positive, offers the row only at a stage above it.
	After int `json:"after,omitempty"`
	// Zero, Nonzero and One are bank slots that must hold zero, a value
	// other than zero, and one; Any are slots of which at least one must
	// hold a value other than zero.
	Zero    []int `json:"zero,omitempty"`
	Nonzero []int `json:"nonzero,omitempty"`
	One     []int `json:"one,omitempty"`
	Any     []int `json:"any,omitempty"`
	// Class and Sex name the starting heroes an inn row is offered to; both
	// empty offers it to every hero. The game reads the words, and the
	// catalog's HeroSlots are where the campaign holds the hero's class and
	// sex.
	Class string `json:"class,omitempty"`
	Sex   string `json:"sex,omitempty"`
	// Place names the building of a chapter town that hands the row's
	// mission out: empty for the tavern's or inn's talk, else PlaceShop or
	// PlaceSchool, whose rows name no NPC record.
	Place string `json:"place,omitempty"`
	// From, when positive and no stage is stated, offers the row at every
	// stage from it on, until the row's mission is won.
	From int `json:"from,omitempty"`
	// Won are the missions that must all be won before the row is offered.
	Won []int `json:"won,omitempty"`
}

const (
	PlaceShop   = "shop"
	PlaceSchool = "school"
)

// HeroSlots are the bank slots that hold the starting hero's class and sex,
// nonzero for a mage and for a female hero.
type HeroSlots struct {
	Mage   int `json:"mage"`
	Female int `json:"female"`
}

// OfferTable names a catalog's row shape by its mod table: a tavern row is
// one chapter of a catalog town, its mission the topic; an inn row takes
// the whole shape.
type OfferTable string

const (
	TavernTable OfferTable = "tavern"
	InnTable    OfferTable = "inn"
	// ShopTable and SchoolTable are the mod tables of a tavern catalog's
	// rows that a shop and a school hand out.
	ShopTable   OfferTable = PlaceShop
	SchoolTable OfferTable = PlaceSchool
)

// Catalog is the catalog offer table whose rows mod table t writes.
func (t OfferTable) Catalog() OfferTable {
	if t == ShopTable || t == SchoolTable {
		return TavernTable
	}
	return t
}

// Of is the mod table that writes row r of catalog table t.
func (t OfferTable) Of(r OfferRow) OfferTable {
	if t == TavernTable && r.Place != "" {
		return OfferTable(r.Place)
	}
	return t
}

// Place is the building a row of mod table t names; empty for a talk.
func (t OfferTable) Place() string {
	if t == ShopTable || t == SchoolTable {
		return string(t)
	}
	return ""
}

const (
	OfferTalk    = 0
	OfferMission = 3
)

// OfferKey names a row: town, chapter and NPC in a tavern, town, chapter and
// mission in a shop or school, topic and NPC in an inn. Table is the mod
// table that writes the row.
type OfferKey struct {
	Table                     OfferTable
	Town, Chapter, Topic, NPC int
}

func (t OfferTable) Key(r OfferRow) OfferKey {
	switch t = t.Catalog().Of(r); t {
	case TavernTable:
		return OfferKey{Table: t, Town: r.Town, Chapter: r.Chapter(), NPC: r.NPC}
	case ShopTable, SchoolTable:
		return OfferKey{Table: t, Town: r.Town, Chapter: r.Chapter(), Topic: r.Topic}
	}
	return OfferKey{Table: t, Topic: r.Topic, NPC: r.NPC}
}

func (k OfferKey) String() string {
	switch k.Table {
	case TavernTable:
		return fmt.Sprintf("town %d chapter %d npc %d", k.Town, k.Chapter, k.NPC)
	case ShopTable, SchoolTable:
		return fmt.Sprintf("town %d chapter %d mission %d", k.Town, k.Chapter, k.Topic)
	}
	return fmt.Sprintf("topic %d npc %d", k.Topic, k.NPC)
}

// Chapter is the row's one stage, or its floor when it states no stage,
// else 0.
func (r OfferRow) Chapter() int {
	switch len(r.Stages) {
	case 0:
		return r.From
	case 1:
		return r.Stages[0]
	}
	return 0
}

// meets reports whether r and o, rows of one tavern catalog, stand in a
// common chapter.
func (r OfferRow) meets(o OfferRow) bool {
	switch {
	case len(r.Stages) == 0 && len(o.Stages) == 0:
		return true
	case len(r.Stages) == 0:
		return o.Chapter() >= r.From
	case len(o.Stages) == 0:
		return r.Chapter() >= o.From
	}
	return r.Chapter() == o.Chapter()
}

func MissionKind(mission int) int {
	if mission != 0 {
		return OfferMission
	}
	return OfferTalk
}

// ChapterOffer is the tavern row of npc in chapter of town id.
func ChapterOffer(id, chapter, npc, mission int) OfferRow {
	return OfferRow{Kind: MissionKind(mission), Topic: mission, NPC: npc, Town: id, Stages: []int{chapter}}
}

func (r OfferRow) Equal(o OfferRow) bool {
	return r.Kind == o.Kind && r.Topic == o.Topic && r.NPC == o.NPC && r.Town == o.Town && r.After == o.After &&
		slices.Equal(r.Stages, o.Stages) && slices.Equal(r.Zero, o.Zero) && slices.Equal(r.Nonzero, o.Nonzero) &&
		slices.Equal(r.One, o.One) && slices.Equal(r.Any, o.Any) && r.Class == o.Class && r.Sex == o.Sex &&
		r.Place == o.Place && r.From == o.From && slices.Equal(r.Won, o.Won)
}

func (r OfferRow) Gates() [][]int { return [][]int{r.Zero, r.Nonzero, r.One, r.Any} }

// OfferError is a refusal of row Row.
type OfferError struct {
	Row OfferKey
	Err error
}

func (e *OfferError) Error() string { return e.Err.Error() }
func (e *OfferError) Unwrap() error { return e.Err }

func OfferErrorf(key OfferKey, format string, args ...any) error {
	return &OfferError{Row: key, Err: fmt.Errorf(format, args...)}
}

// validateOffers checks each row's bounds and key against its table; a
// chapter names an NPC record once. A refusal is an OfferError.
func (c *Catalog) validateOffers(towns map[int]bool) error {
	t := c.OfferTable
	switch {
	case t == "" && c.Offers == nil:
		return nil
	case t != TavernTable && t != InnTable:
		return fmt.Errorf("town catalog %s: offer table %q is neither %q nor %q", c.Catalog, t, TavernTable, InnTable)
	}
	tavern := t == TavernTable
	minNPC, topic := 1, "topic"
	if tavern {
		minNPC, topic = 0, "mission"
	}
	seen := map[OfferKey]bool{}
	speaks := map[int][]OfferRow{}
	hands := map[string][]OfferRow{}
	for _, r := range c.Offers {
		k := t.Key(r)
		talk := r.Place == ""
		bad := func(format string, args ...any) error {
			return OfferErrorf(k, "town catalog %s: %s row %v: "+format, append([]any{c.Catalog, k.Table, k}, args...)...)
		}
		switch {
		case (tavern || r.Town != 0) && !towns[r.Town]:
			return bad("town %d is not a town of the catalog", r.Town)
		case !tavern && (r.Place != "" || r.From != 0 || len(r.Won) > 0):
			return bad("an inn row states no building, chapter floor or won mission")
		case !talk && r.Place != PlaceShop && r.Place != PlaceSchool:
			return bad("place %q is neither %q nor %q", r.Place, PlaceShop, PlaceSchool)
		case tavern && len(r.Stages) == 0 && r.From == 0:
			return bad("%d chapters are stated; a tavern row stands in one", len(r.Stages))
		case tavern && len(r.Stages) > 1:
			return bad("%d chapters are stated; a tavern row stands in one", len(r.Stages))
		case tavern && len(r.Stages) == 1 && r.From != 0:
			return bad("chapter %d and from %d are both stated; a row stands in one chapter or from one on", r.Stages[0], r.From)
		case tavern && r.Chapter() <= 0:
			return bad("chapter %d is not positive", r.Chapter())
		case !talk && r.NPC != 0:
			return bad("npc %d is stated; a %s row names no NPC record", r.NPC, k.Table)
		case !talk && r.Topic <= 0:
			return bad("mission %d is not positive; a %s row hands out a mission", r.Topic, k.Table)
		case r.NPC < minNPC || r.NPC > maxTavernValue:
			return bad("npc %d is outside the range %d to %d", r.NPC, minNPC, maxTavernValue)
		case r.Topic < 0 || r.Topic > maxTavernValue:
			return bad("%s %d is outside the range 0 to %d", topic, r.Topic, maxTavernValue)
		case r.Kind != OfferTalk && r.Kind != OfferMission:
			return bad("kind %d is neither a talk (%d) nor a mission talk (%d)", r.Kind, OfferTalk, OfferMission)
		case tavern && r.Kind != MissionKind(r.Topic):
			return bad("kind %d does not follow mission %d; a tavern row is a mission talk exactly when it gives a mission", r.Kind, r.Topic)
		case tavern && (r.After != 0 || slices.ContainsFunc(r.Gates(), func(l []int) bool { return len(l) > 0 })):
			return bad("a tavern row states no stage floor or bank gate")
		case r.After < 0:
			return bad("after %d is negative", r.After)
		case tavern && (r.Class != "" || r.Sex != ""):
			return bad("a tavern row is offered to every starting hero; it states no class or sex")
		case (r.Class != "" || r.Sex != "") && c.HeroSlots == nil:
			return bad("class %q and sex %q are stated, but the catalog names no hero-slots", r.Class, r.Sex)
		case (!tavern || !talk) && seen[k]:
			return bad("the row is named twice")
		}
		for i, m := range r.Won {
			switch {
			case m <= 0 || m > maxTavernValue:
				return bad("won mission %d is outside the range 1 to %d", m, maxTavernValue)
			case m == r.Topic:
				return bad("won names mission %d, the mission the row itself gives", m)
			case slices.Contains(r.Won[:i], m):
				return bad("won names mission %d twice", m)
			}
		}
		for _, s := range r.Stages {
			if s <= 0 {
				return bad("stage %d is not positive", s)
			}
		}
		for _, slots := range r.Gates() {
			for _, s := range slots {
				if s < 0 {
					return bad("bank slot %d is negative", s)
				}
			}
		}
		if tavern && talk {
			for _, prior := range speaks[r.NPC] {
				if r.meets(prior) {
					return bad("npc %d already speaks in chapter %d, in the row %v; a chapter names an NPC record once",
						r.NPC, max(r.Chapter(), prior.Chapter()), t.Key(prior))
				}
			}
			speaks[r.NPC] = append(speaks[r.NPC], r)
		}
		if tavern && !talk {
			for _, prior := range hands[r.Place] {
				if prior.Town == r.Town && prior.Topic == r.Topic && r.meets(prior) {
					return bad("the %s of town %d already hands out mission %d in chapter %d, in the row %v; a place hands out a mission once in a chapter",
						k.Table, r.Town, r.Topic, max(r.Chapter(), prior.Chapter()), t.Key(prior))
				}
			}
			hands[r.Place] = append(hands[r.Place], r)
		}
		seen[k] = true
	}
	return nil
}

// MercenaryRule is a tavern's mercenary stock walk and squad price rule as
// data. The roster walks the stock units of types 1 to StockTypes; with no
// unit ids from a LOAD, type t's unit id is StockIDPhase + t - 1. A squad's
// unit price factor at mission m is PriceLadder[(m-PriceFrom)/PriceStep],
// 0 below PriceFrom and past the ladder; its level is the Level of the last
// of Levels whose From m reaches, 1 below them all.
type MercenaryRule struct {
	StockTypes   int              `json:"stock-types"`
	StockIDPhase int              `json:"stock-id-phase"`
	PriceFrom    int              `json:"price-from"`
	PriceStep    int              `json:"price-step"`
	PriceLadder  []int            `json:"price-ladder"`
	Levels       []MercenaryLevel `json:"levels"`
}

// MercenaryLevel is the squad level from mission From.
type MercenaryLevel struct {
	From  int `json:"from"`
	Level int `json:"level"`
}

// validate checks the rule's own bounds: positive stock types, price step
// and levels, no negative phase, start or factor, and levels in rising
// order of From. A nil rule states none.
func (r *MercenaryRule) validate(catalog string) error {
	if r == nil {
		return nil
	}
	bad := func(format string, args ...any) error {
		return fmt.Errorf("town catalog %s: mercenaries: "+format, append([]any{catalog}, args...)...)
	}
	switch {
	case r.StockTypes <= 0:
		return bad("stock-types %d is not positive", r.StockTypes)
	case r.StockIDPhase < 0 || r.PriceFrom < 0:
		return bad("stock-id-phase %d or price-from %d is negative", r.StockIDPhase, r.PriceFrom)
	case r.PriceStep <= 0:
		return bad("price-step %d is not positive", r.PriceStep)
	}
	for _, f := range r.PriceLadder {
		if f < 0 {
			return bad("price factor %d is negative", f)
		}
	}
	for i, l := range r.Levels {
		if l.Level <= 0 || i > 0 && l.From <= r.Levels[i-1].From {
			return bad("level %+v is not positive or does not rise above the one before", l)
		}
	}
	return nil
}

// UnitPrice is the squad unit price factor at mission.
func (r *MercenaryRule) UnitPrice(mission int) int {
	if r == nil || mission < r.PriceFrom {
		return 0
	}
	if i := (mission - r.PriceFrom) / r.PriceStep; i < len(r.PriceLadder) {
		return r.PriceLadder[i]
	}
	return 0
}

// Level is the squad level at mission.
func (r *MercenaryRule) Level(mission int) int {
	level := 1
	if r != nil {
		for _, l := range r.Levels {
			if mission >= l.From {
				level = l.Level
			}
		}
	}
	return level
}
