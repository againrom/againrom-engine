package rules

import "strings"

// The spell catalog: one name per spell, each game's profile as data
// (MAGIC-SPELL-001, R2-ENGINE-019, DAT-NAMES-016, AI-SPELLIDENT-286,
// MAGIC-ICON-024, R2-ENGINE-444, SHOP-CONSUME-073, R2-ENGINE-024).

// GameSpell is one row of a game's profile; Slot counts from 1, 0 is none.
type GameSpell struct {
	Name         string
	Slot         uint8
	Book, Scroll bool
	Arm          uint16
}

const (
	SpellArmNone   uint16 = 0xffff
	SpellArmSummon uint16 = 0xfffe
)

// SpellIDs is the first game's spell count.
const SpellIDs = len(firstGameSpells)

var firstGameSpells = [...]GameSpell{
	{Name: "Fire_Arrow", Slot: 1, Scroll: true, Arm: 1},
	{Name: "Fire_Ball", Slot: 2, Book: true, Scroll: true, Arm: 2},
	{Name: "Wall_of_Fire", Slot: 3, Book: true, Scroll: true, Arm: 3},
	{Name: "Fire_Sacrifice", Slot: 4, Book: true, Scroll: true, Arm: 4},
	{Name: "Protection_from_Fire", Slot: 5, Book: true, Scroll: true, Arm: 5},
	{Name: "Heal", Slot: 13, Scroll: true, Arm: 6},
	{Name: "Freezing_Cloud", Slot: 14, Book: true, Scroll: true, Arm: 7},
	{Name: "Poison_Cloud", Slot: 15, Book: true, Scroll: true, Arm: 8},
	{Name: "Acid_Stream", Slot: 16, Book: true, Scroll: true, Arm: 9},
	{Name: "Protection_from_Water", Slot: 17, Book: true, Scroll: true, Arm: 10},
	{Name: "Drain_Life", Scroll: true, Arm: 11},
	{Name: "Light", Slot: 12, Scroll: true, Arm: 12},
	{Name: "Lightning", Slot: 11, Book: true, Scroll: true, Arm: 13},
	{Name: "Prismatic_Spray", Slot: 10, Book: true, Scroll: true, Arm: 14},
	{Name: "Invisibility", Slot: 9, Book: true, Scroll: true, Arm: 15},
	{Name: "Protection_from_Air", Slot: 8, Book: true, Scroll: true, Arm: 16},
	{Name: "Darkness", Scroll: true, Arm: 17},
	{Name: "Shield", Slot: 24, Scroll: true, Arm: 18},
	{Name: "Wall_of_Earth", Slot: 23, Book: true, Scroll: true, Arm: 19},
	{Name: "Stone_Curse", Slot: 22, Book: true, Scroll: true, Arm: 20},
	{Name: "Meteor_Storm", Slot: 21, Book: true, Scroll: true, Arm: 21},
	{Name: "Protection_from_Earth", Slot: 20, Book: true, Scroll: true, Arm: 22},
	{Name: "Bless", Slot: 6, Book: true, Scroll: true, Arm: 23},
	{Name: "Haste", Slot: 7, Book: true, Scroll: true, Arm: 24},
	{Name: "Control_Spirit", Slot: 18, Book: true, Scroll: true, Arm: 25},
	{Name: "Teleport", Slot: 19, Book: true, Scroll: true, Arm: 26},
	{Name: "Curse", Scroll: true, Arm: 27},
	{Name: "Slow", Arm: 28},
}

var secondGameSpells = [...]GameSpell{
	{Name: "Fire_Arrow", Slot: 1, Scroll: true, Arm: 1},
	{Name: "Fire_Ball", Slot: 2, Book: true, Scroll: true, Arm: 2},
	{Name: "Wall_of_Fire", Slot: 3, Book: true, Scroll: true, Arm: 3},
	{Name: "Protection_from_Fire", Slot: 4, Book: true, Scroll: true, Arm: 5},
	{Name: "Ice_Missile", Slot: 13, Book: true, Scroll: true, Arm: SpellArmNone},
	{Name: "Poison_Cloud", Slot: 14, Scroll: true, Arm: 8},
	{Name: "Blizzard", Slot: 15, Book: true, Scroll: true, Arm: 21},
	{Name: "Protection_from_Water", Slot: 16, Book: true, Scroll: true, Arm: 10},
	{Name: "Acid_Stream", Book: true, Scroll: true, Arm: 9},
	{Name: "Lightning", Slot: 12, Book: true, Scroll: true, Arm: 13},
	{Name: "Prismatic_Spray", Slot: 11, Scroll: true, Arm: 14},
	{Name: "Invisibility", Slot: 10, Scroll: true, Arm: 15},
	{Name: "Protection_from_Air", Slot: 9, Book: true, Scroll: true, Arm: 16},
	{Name: "Darkness", Book: true, Scroll: true, Arm: 17},
	{Name: "Light", Book: true, Scroll: true, Arm: 12},
	{Name: "Diamond_Dust", Slot: 24, Book: true, Scroll: true, Arm: SpellArmNone},
	{Name: "Wall_of_Earth", Slot: 23, Scroll: true, Arm: 19},
	{Name: "Stone_Curse", Slot: 22, Scroll: true, Arm: 20},
	{Name: "Protection_from_Earth", Slot: 21, Book: true, Scroll: true, Arm: 22},
	{Name: "Bless", Slot: 6, Book: true, Scroll: true, Arm: 23},
	{Name: "Haste", Slot: 7, Book: true, Scroll: true, Arm: 24},
	{Name: "Control_Spirit", Slot: 18, Book: true, Scroll: true, Arm: 25},
	{Name: "Teleport", Slot: 19, Book: true, Scroll: true, Arm: 26},
	{Name: "Heal", Slot: 5, Book: true, Scroll: true, Arm: 6},
	{Name: "Summon", Slot: 17, Book: true, Scroll: true, Arm: SpellArmSummon},
	{Name: "Drain_Life", Slot: 8, Book: true, Scroll: true, Arm: 11},
	{Name: "Shield", Slot: 20, Scroll: true, Arm: 18},
	{Name: "Curse", Arm: 27},
	{Name: "Slow", Arm: 28},
}

var spellProfiles = [...][]GameSpell{SpellGameFirst: firstGameSpells[:], SpellGameSecond: secondGameSpells[:]}

// SpellProfile is game's profile, row n-1 being spell number n.
func SpellProfile(g SpellGame) []GameSpell {
	if int(g) >= len(spellProfiles) {
		return nil
	}
	return spellProfiles[g]
}

// GameSpellOf is the row of spell number id in game's profile.
func GameSpellOf(g SpellGame, id int) (GameSpell, bool) {
	p := SpellProfile(g)
	if id < 1 || id > len(p) {
		return GameSpell{}, false
	}
	return p[id-1], true
}

// SpellCatalog is every catalog name once.
func SpellCatalog() []string {
	var out []string
	seen := map[string]bool{}
	for _, p := range spellProfiles {
		for _, s := range p {
			if !seen[s.Name] {
				seen[s.Name] = true
				out = append(out, s.Name)
			}
		}
	}
	return out
}

// SpellKey is the catalog form of a name: "_" for a space.
func SpellKey(name string) string { return strings.ReplaceAll(name, " ", "_") }

// SpellNumber is the number game gives name.
func SpellNumber(g SpellGame, name string) (int, bool) {
	key := SpellKey(name)
	for i, s := range SpellProfile(g) {
		if s.Name == key {
			return i + 1, true
		}
	}
	return 0, false
}

// SpellBookRows is game's book order by spell number.
func SpellBookRows(g SpellGame) []uint16 {
	p := SpellProfile(g)
	var out []uint16
	for slot := 1; ; slot++ {
		n := len(out)
		for i, s := range p {
			if int(s.Slot) == slot {
				out = append(out, uint16(i+1))
			}
		}
		if len(out) == n {
			return out
		}
	}
}

// SpellShopBook reports that a shop of game sells the book of spell id.
func SpellShopBook(g SpellGame, id int) bool {
	s, ok := GameSpellOf(g, id)
	return ok && s.Book
}

// SpellShopScroll reports that a shop of game sells the scroll of spell id.
func SpellShopScroll(g SpellGame, id int) bool {
	s, ok := GameSpellOf(g, id)
	return ok && s.Scroll
}

// SpellArm is the arm spell id runs in game.
func SpellArm(g SpellGame, id int) uint16 {
	if s, ok := GameSpellOf(g, id); ok {
		return s.Arm
	}
	return SpellArmNone
}
