package rules

import (
	"fmt"
	"slices"
	"strings"
)

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
func SpellBookRows(g SpellGame) []uint16 { return GameProfile(g).BookRows() }

// SpellShopBook reports that a shop of game sells the book of spell id.
func SpellShopBook(g SpellGame, id int) bool { return GameProfile(g).ShopBook(id) }

// SpellShopScroll reports that a shop of game sells the scroll of spell id.
func SpellShopScroll(g SpellGame, id int) bool { return GameProfile(g).ShopScroll(id) }

// SpellBookMax is the most spells the SAV book holds; a spell past it has no
// book position.
const SpellBookMax = 28

// SpellGameProfile is a game's profile and its book's number of positions.
type SpellGameProfile struct {
	Spells   []GameSpell
	BookSize int
}

// GameProfile is game's own profile; its book is the positions from 1 that
// hold a spell, the positions the book draws.
func GameProfile(g SpellGame) SpellGameProfile {
	p := SpellGameProfile{Spells: SpellProfile(g)}
	for p.BookSize < SpellBookMax && slices.ContainsFunc(p.Spells, func(s GameSpell) bool { return int(s.Slot) == p.BookSize+1 }) {
		p.BookSize++
	}
	return p
}

// SpellBookPositions is the most book positions a mod may give any game.
func SpellBookPositions() int {
	return max(GameProfile(SpellGameFirst).BookSize, GameProfile(SpellGameSecond).BookSize)
}

// BookRows is the spell number at each book position, 0 at an empty one.
func (p SpellGameProfile) BookRows() []uint16 {
	out := make([]uint16, p.BookSize)
	for i, s := range p.Spells {
		if s.Slot != 0 && int(s.Slot) <= p.BookSize {
			out[s.Slot-1] = uint16(i + 1)
		}
	}
	return out
}

// ShopBook reports that a shop sells the book of spell id.
func (p SpellGameProfile) ShopBook(id int) bool {
	return id >= 1 && id <= len(p.Spells) && p.Spells[id-1].Book
}

// ShopScroll reports that a shop sells the scroll of spell id.
func (p SpellGameProfile) ShopScroll(id int) bool {
	return id >= 1 && id <= len(p.Spells) && p.Spells[id-1].Scroll
}

// SpellEdit is a mod's switch and book position of one spell by catalog
// name; On is an explicit on, Position 0 keeps the profile's.
type SpellEdit struct {
	Name     string
	Off, On  bool
	Position int
}

// SpellEditError is the edit Edited refuses; Edit -1 is the book size.
type SpellEditError struct {
	Edit int
	Msg  string
}

func (e *SpellEditError) Error() string { return e.Msg }

// Edited is p with size book positions (0 keeps p's) and edits applied. An
// off spell, or one whose position falls past the size, leaves the book and
// the shops' books and scrolls; its arm and number stay. Only a spell p's
// book draws may be on or take a position.
func (p SpellGameProfile) Edited(size int, edits []SpellEdit) (SpellGameProfile, error) {
	if size == 0 {
		size = p.BookSize
	}
	if size < 1 || size > p.BookSize {
		return p, &SpellEditError{-1, fmt.Sprintf("book_size is %d, outside 1..%d", size, p.BookSize)}
	}
	out := SpellGameProfile{Spells: slices.Clone(p.Spells), BookSize: size}
	moved, off, asked := map[int]int{}, map[int]bool{}, map[int]int{}
	for k, e := range edits {
		i := slices.IndexFunc(out.Spells, func(s GameSpell) bool { return s.Name == SpellKey(e.Name) })
		_, hasPosition := moved[i]
		bad := func(format string, args ...any) (SpellGameProfile, error) {
			return p, &SpellEditError{k, fmt.Sprintf(format, args...)}
		}
		switch {
		case i < 0:
			return bad("%s is not a spell of this game's catalog", e.Name)
		case e.Off && (e.Position != 0 || hasPosition), e.Position != 0 && off[i]:
			return bad("%s is off, so book_position does not apply to it", e.Name)
		case (e.On || e.Position != 0) && p.Spells[i].Slot == 0:
			return bad("%s has no place in this game's book, so it cannot be on or take a book_position", e.Name)
		case e.Position < 0 || e.Position > size:
			return bad("book_position of %s is %d, outside 1..%d", e.Name, e.Position, size)
		}
		if e.On && e.Position == 0 {
			asked[k] = i
		}
		s := &out.Spells[i]
		if e.Off {
			s.Slot, s.Book, s.Scroll, off[i] = 0, false, false, true
		}
		if e.Position != 0 {
			s.Slot, moved[i] = uint8(e.Position), k
		}
	}
	// An explicit on is tested against the cell the edits leave.
	for k, e := range edits {
		if i, ok := asked[k]; ok && !movedAt(moved, i) && int(p.Spells[i].Slot) > size {
			return p, &SpellEditError{k, fmt.Sprintf("%s is on, but its book position %d is past book_size %d", e.Name, p.Spells[i].Slot, size)}
		}
	}
	held := map[uint8]int{}
	for i := range out.Spells {
		s := &out.Spells[i]
		if int(s.Slot) > size {
			s.Slot, s.Book, s.Scroll = 0, false, false
		}
		if s.Slot == 0 {
			continue
		}
		if j, dup := held[s.Slot]; dup {
			k, ok := moved[i]
			if !ok {
				k = moved[j]
			}
			return p, &SpellEditError{k, fmt.Sprintf("book_position %d holds both %s and %s", s.Slot, out.Spells[j].Name, s.Name)}
		}
		held[s.Slot] = i
	}
	return out, nil
}

func movedAt(m map[int]int, k int) bool {
	_, ok := m[k]
	return ok
}

// SpellArm is the arm spell id runs in game.
func SpellArm(g SpellGame, id int) uint16 {
	if s, ok := GameSpellOf(g, id); ok {
		return s.Arm
	}
	return SpellArmNone
}
