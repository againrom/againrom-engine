package game

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
)

// secondStream writes a second-game archive in file order: a class's first
// object introduces its name and takes an index for the class and one for
// itself; a later object of the class takes one (the codec's object table).
type secondStream struct {
	b       []byte
	next    uint16
	classes map[string]uint16
}

func (s *secondStream) obj(name string) {
	if idx, ok := s.classes[name]; ok {
		s.u16(0x8000 | idx)
	} else {
		s.u16(0xffff)
		s.u16(1)
		s.u16(uint16(len(name)))
		s.b = append(s.b, name...)
		s.classes[name] = s.next
		s.next++
	}
	s.next++
}

func (s *secondStream) u8(v uint8)   { s.b = append(s.b, v) }
func (s *secondStream) u16(v uint16) { s.b = binary.LittleEndian.AppendUint16(s.b, v) }
func (s *secondStream) u32(v uint32) { s.b = binary.LittleEndian.AppendUint32(s.b, v) }
func (s *secondStream) raw(n int)    { s.b = append(s.b, make([]byte, n)...) }
func (s *secondStream) null()        { s.u16(0) }
func (s *secondStream) cstr(v string) {
	s.u8(uint8(len(v)))
	s.b = append(s.b, v...)
}

// token writes the second game's 38-byte Token head: its +0x0c word is u16
// (R2-ENGINE-209). price is the head's +0x1c word, an item's price.
func (s *secondStream) token(runtimeID uint32, row uint16, mapUnitID uint16, price, key uint32) {
	s.u16(0)
	s.u16(0)
	s.u8(0x80)
	s.u8(0x80)
	s.u16(0)
	s.u32(0)
	s.u32(runtimeID)
	s.u16(row)
	s.u16(0)
	s.u32(uint32(mapUnitID))
	s.u16(0)
	s.u32(price)
	s.u32(key)
	s.u32(0) // no Reference
}

// secondPiece is one item a fixture hero wears or carries.
type secondPiece struct {
	class       string
	code, stack uint16
	row, kind   uint8
	price       uint32
}

func (s *secondStream) item(p secondPiece) {
	s.obj(p.class)
	s.token(0, uint16(p.row), 0, p.price, 0)
	s.u32(0) // no effects
	s.u16(p.code)
	s.u16(p.stack)
	s.u8(p.kind)
	s.u8(0)
	s.u8(0)
	s.u16(0)
	s.u16(0)
	s.u8(0)
	switch p.class {
	case "Shield":
		s.raw(22)
	case "Armor":
		s.raw(22)
		s.u8(p.row)
	case "Weapon":
		s.raw(24 + 22)
		s.u8(0)
		s.null() // no weapon spell
	}
}

// secondOriginalHero is the Player-1 Human of a fixture: the fields
// R2-SESSION-236 measures and the book of R2-SESSION-237.
type secondOriginalHero struct {
	name       string
	defRow     uint16
	mapUnitID  uint16
	key        uint32
	stats      [sav.UnitStatWords]uint16
	skills     [6]uint16
	mainSkill  uint32
	xp         [5]uint32
	experience uint32
	worn, bag  []secondPiece
	spells     []uint8
	bookSlots  int
}

// human writes the second game's Human: Unit with its 184-byte sixth raw
// block, Humanoid's 24-byte block of the main skill and five experiences
// (R2-ENGINE-320), twelve worn references and no Diary (R2-ENGINE-208).
func (s *secondStream) human(h secondOriginalHero) {
	s.obj("Human")
	s.token(1, h.defRow, h.mapUnitID, 0, h.key)
	s.u32(0) // no effects
	s.u16(0) // two empty route lists
	s.u16(0)
	s.u16(0)
	for _, level := range h.skills {
		s.u16(level)
	}
	s.raw(10)
	s.raw(22 + 24 + 64 + 180 + 184)
	s.u16(0)
	s.raw(19)
	s.null() // weapon
	s.null() // shield
	s.cstr(h.name)
	for _, v := range h.stats {
		s.u16(v)
	}
	s.u8(0)
	s.u8(0)
	s.u16(0)
	s.u16(0x0800) // sight
	s.u8(0)
	s.u32(h.experience)
	s.raw(3)
	s.u32(0)
	s.u8(0) // stage
	s.u32(0)
	s.u32(0)
	s.null()
	if len(h.bag) == 0 {
		s.u8(0)
	} else {
		s.u8(1)
		s.u32(uint32(len(h.bag)))
		for _, p := range h.bag {
			s.item(p)
		}
		s.u32(0)
		s.u32(0)
	}
	if h.bookSlots == 0 {
		s.u8(0)
	} else {
		s.u8(1)
		s.u32(0)
		s.u32(uint32(h.bookSlots))
		for slot := 1; slot < h.bookSlots; slot++ {
			if !slices.Contains(h.spells, uint8(slot)) {
				s.null()
				continue
			}
			s.obj("Spell")
			s.u8(uint8(slot))
			s.u8(5)
			s.u8(0)
			s.u16(10)
			s.u32(0xb0000000 | uint32(slot))
		}
	}
	s.raw(16)
	s.u8(0)
	s.u32(h.mainSkill)
	for _, v := range h.xp {
		s.u32(v)
	}
	for i := range 12 {
		if i < len(h.worn) {
			s.item(h.worn[i])
			continue
		}
		s.null()
	}
}

// secondSavedBody is a second-game document in test code from the format
// contract: the head, one Player with its 2,560 and 36 raw bytes around one
// group holding the hero when there is one, an empty dead list, no world
// half and the trailer (R2-SESSION-234).
func secondSavedBody(mission uint32, money uint32, hero *secondOriginalHero) []byte {
	return secondSavedDocument(mission, money, hero, false)
}

// secondSavedDocument is secondSavedBody, with an empty world half when
// world is set: no Buildings, effects, terrain, Sacks or extra Tokens, and
// the second game's 10,374-byte session (R2-SESSION-235, R2-ENGINE-603).
func secondSavedDocument(mission uint32, money uint32, hero *secondOriginalHero, world bool) []byte {
	s := &secondStream{next: 1, classes: map[string]uint16{}}
	s.u32(0x50)
	s.u32(0x05)
	s.cstr("town.alm")
	for range 11 {
		s.u32(0)
	}
	s.u32(mission)
	s.u32(2)
	s.u32(6)
	s.u32(1)
	s.obj("Player")
	s.cstr("Hero")
	s.u16(1)
	s.u32(1)
	s.raw(8)
	s.u8(0x02)
	s.u32(0)
	s.u16(2)
	s.u32(money ^ 0x5c073f4d)
	s.u8(0)
	s.u8(0)
	s.u32(0 ^ 0x5c073f4d)
	s.u32(0)
	s.u16(0)
	s.u16(0)
	s.u32(0)
	if hero != nil {
		s.u32(hero.key)
	} else {
		s.u32(0)
	}
	s.u32(0x0a000001) // the Player's own key
	s.raw(2560)
	s.u32(1) // one group
	s.raw(2 + 80 + 2)
	if hero != nil {
		s.u32(1)
		s.human(*hero)
	} else {
		s.u32(0)
	}
	s.raw(12)
	s.raw(36)
	s.u32(0) // dead list
	if !world {
		s.u8(0)
		return savedTrailer(s.b)
	}
	s.u8(1)
	s.u32(0) // Buildings
	s.u32(0) // SpellEffects
	s.u16(0) // terrain blocks
	s.u16(0) // terrain cells
	s.raw(4 + 10374)
	s.u32(0) // Sacks
	s.u32(0) // extra Tokens
	return savedTrailer(s.b)
}

// secondSavedTail is a second-game tail: no pairs, a bank whose slot 768
// holds 10 and slot 769 1, as the first town's does after the inn's talk,
// then the given places, the current ordinal and nine empty buffers.
func secondSavedTail(current uint32, places ...[2]uint32) []byte {
	var b []byte
	le32 := func(v uint32) { b = binary.LittleEndian.AppendUint32(b, v) }
	le32(0)
	le32(0)
	le32(0)
	bank := make([]byte, 4096)
	binary.LittleEndian.PutUint32(bank[4*768:], 10)
	binary.LittleEndian.PutUint32(bank[4*769:], 1)
	b = append(b, bank...)
	b = append(b, shopGrid(secondshop.NewGameRecords)...)
	le32(uint32(len(places)))
	for _, p := range places {
		le32(p[0]) // ID
		le32(p[1]) // kind
		b = append(b, make([]byte, 16)...)
	}
	le32(current)
	for range 9 {
		b = append(b, make([]byte, 8)...)
	}
	return b
}

// secondTailGrid is the shop grid's offset in secondSavedTail: the empty
// pair list and its two words, then the bank.
const secondTailGrid = 12 + 4096

// shopGrid is the tail's 320-byte grid of records: four locations of four
// 0x14-byte cells (R2-SESSION-156, R2-ENGINE-263).
func shopGrid(records [4]secondshop.Record) []byte {
	var b []byte
	for _, r := range records {
		for _, c := range r {
			for _, v := range []uint32{uint32(c.Min), uint32(c.Max), uint32(c.Draws), uint32(c.Bound), c.Mask} {
				b = binary.LittleEndian.AppendUint32(b, v)
			}
		}
	}
	return b
}

func shortcutsStore(cells ...int32) []byte {
	return synth.Reg(0x11, []synth.RegNode{
		{Name: "SpellBook", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Shortcuts", Kind: 0x06, Ints: cells},
		}},
	})
}

// originalContainer is the shared envelope: magic, the codec blob, the
// 256-byte label region and the tail (R2-SESSION-017).
func originalContainer(magic string, body, tail []byte) []byte {
	return labelledContainer(magic, "", body, tail)
}

func labelledContainer(magic, label string, body, tail []byte) []byte {
	blob := sav.Compress(body)
	out := make([]byte, 16)
	copy(out, magic)
	binary.LittleEndian.PutUint32(out[4:], uint32(16+len(blob)))
	binary.LittleEndian.PutUint32(out[8:], sav.MinVersion)
	binary.LittleEndian.PutUint32(out[12:], uint32(len(blob)))
	out = append(out, blob...)
	region := make([]byte, 0x100)
	copy(region, label)
	out = append(out, region...)
	return append(out, tail...)
}

func secondSavedFile(t *testing.T, current uint32, places ...[2]uint32) *sav.File {
	t.Helper()
	b := originalContainer(sav.SecondGame.Magic, secondSavedBody(0, 0, nil), append(shortcutsStore(), secondSavedTail(current, places...)...))
	sf, err := sav.Open(b)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return sf
}

// secondFixtureHero is a mage like the EN files' second hero: four
// attributes, two skills, the main skill and five experiences, a worn staff
// and robe, a bag of one staff, spells 24 and 27 in a 28-slot book.
func secondFixtureHero() secondOriginalHero {
	h := secondOriginalHero{
		name: "Gildarius", defRow: 28, mapUnitID: 13474, key: 0x0ff00001,
		skills: [6]uint16{0, 0, 0, 0, 20, 10}, mainSkill: 4, xp: [5]uint32{0, 0, 0, 5727, 1593},
		experience: 7320,
		worn: []secondPiece{
			{class: "Weapon", code: 0x812d, stack: 1, row: 13, kind: 2, price: 3819},
			{class: "Armor", code: 0xf72d, stack: 1, row: 13, kind: 1, price: 250},
		},
		bag:       []secondPiece{{class: "Weapon", code: 0x812d, stack: 1, row: 13, kind: 2, price: 3819}},
		spells:    []uint8{24, 27},
		bookSlots: 28,
	}
	h.stats = [sav.UnitStatWords]uint16{
		sav.StatBody: 19, sav.StatReaction: 15, sav.StatMind: 43, sav.StatSpirit: 25,
		sav.StatSpeed: 15, sav.StatCapacity: 191, sav.StatHealth: 29, sav.StatHealthMax: 29,
		sav.StatHealthRegen: 100, sav.StatMana: 75, sav.StatManaMax: 75, sav.StatManaRegen: 50,
	}
	return h
}

// secondFixtureTown is a town-like second-game file in the first town with
// mission 10 open, the hero above and money.
func secondFixtureTown(label string, hero secondOriginalHero, money uint32) []byte {
	return labelledContainer(sav.SecondGame.Magic, label, secondSavedBody(0, money, &hero),
		append(shortcutsStore(), secondSavedTail(0, [2]uint32{1, placeTown}, [2]uint32{10, placeMission})...))
}

func TestSecondFixtureHeroWalks(t *testing.T) {
	sf, err := sav.Open(secondFixtureTown("town", secondFixtureHero(), 44))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	chars, err := sf.Party()
	if err != nil || len(chars) != 1 || !chars[0].Hero {
		t.Fatalf("party %d %v", len(chars), err)
	}
	c := chars[0]
	if c.Name != "Gildarius" || c.Stat(sav.StatMind) != 43 || c.MainSkill != 4 || c.SkillXP[4] != 5727 || c.SkillXP[5] != 1593 {
		t.Fatalf("hero %+v", c)
	}
	if len(c.Worn) != 2 || len(c.Items) != 1 || len(c.Spells) != 2 || c.Spells[1].ID != 27 {
		t.Fatalf("worn %d bag %d book %+v", len(c.Worn), len(c.Items), c.Spells)
	}
}

func TestSecondFileSessionReadsTheCampaignTail(t *testing.T) {
	sf := secondSavedFile(t, 0, [2]uint32{1, placeTown}, [2]uint32{10, placeMission})
	stated, err := fileCampaign(sf).fileState(sf)
	if err != nil {
		t.Fatalf("fileState: %v", err)
	}
	s := stated.session
	if s == nil || s.Second == nil || s.Game != base.GameROM2 {
		t.Fatalf("session %+v, want the second game's", s)
	}
	c := s.Second
	if c.Shop != nil || c.restore().shop.records != secondshop.NewGameRecords {
		t.Fatalf("a grid of the new game's records restores %+v", c.Shop)
	}
	if c.Bank[768] != 10 || c.Current != (currentSecondLocation{placeTown, 1}) {
		t.Fatalf("bank slot 768 %d current %+v", c.Bank[768], c.Current)
	}
	if len(c.Available) != 2 || c.Available[1] != (currentSecondLocation{placeMission, 10}) {
		t.Fatalf("available %+v", c.Available)
	}
	if c.Room == nil || *c.Room != secondTownSquare {
		t.Fatal("a town opens elsewhere than at its square (DIV-4153)")
	}
	if len(c.Party) != 0 {
		t.Fatal("an empty bag became a party")
	}
	mission, err := secondFileSession(secondSavedFile(t, 0, [2]uint32{10, placeMission}))
	if err != nil || mission.Second.Room != nil {
		t.Fatalf("a mission place has a town room: %+v %v", mission, err)
	}
	if _, err := secondFileSession(secondSavedFile(t, 2, [2]uint32{1, placeTown})); err == nil {
		t.Fatal("a current ordinal past the list is accepted")
	}
	withBag, err := sav.Open(secondFixtureTown("town", secondFixtureHero(), 44))
	if err != nil {
		t.Fatal(err)
	}
	s, err = secondFileSession(withBag)
	if err != nil || len(s.Second.Party) != 1 || len(s.Second.Party[0].Member.CarriedItems) != 1 || s.Second.Party[0].Member.CarriedItems[0].Code != 0x812d {
		t.Fatalf("the hero's bag is not the campaign's: %+v %v", s, err)
	}
}

func TestSecondHeroBagExpandsStacksAndRefusesForeignItems(t *testing.T) {
	shield := sav.Piece{Code: 0x0205, Stack: 3, Price: 40}
	hero := sav.Character{Hero: true, Items: []sav.Piece{shield}}
	other := sav.Character{Items: []sav.Piece{shield}}
	bag, err := secondHeroBag([]sav.Character{other, hero})
	if err != nil {
		t.Fatalf("secondHeroBag: %v", err)
	}
	if len(bag) != 3 || bag[0].Code != 0x0205 || bag[2].Price != 40 {
		t.Fatalf("bag %+v, want three units of the hero's shield", bag)
	}
	hero.Items = []sav.Piece{{Code: 0x0f01}}
	if _, err := secondHeroBag([]sav.Character{hero}); err == nil || !strings.Contains(err.Error(), "no bag form") {
		t.Fatalf("an item outside the bag's forms: %v", err)
	}
	hero.Items = []sav.Piece{{Code: 0x0205, Stack: maxSecondBagUnits}, {Code: 0x0205}}
	if _, err := secondHeroBag([]sav.Character{hero}); err == nil || !strings.Contains(err.Error(), "more than") {
		t.Fatalf("a bag past the unit bound: %v", err)
	}
}

func TestValidateOriginalGameNamesABsgFileTheSecondGames(t *testing.T) {
	b := secondFixtureTown("town", secondFixtureHero(), 44)
	if err := validateOriginalGame(b, base.GameROM2); err != nil {
		t.Fatalf("second game: %v", err)
	}
	if err := validateOriginalGame(b, base.GameROM1); err == nil {
		t.Fatal("the first game accepts a Bsg& file")
	}
}

// An empty shortcut array binds no cell in a second-game file and stays a
// refusal in a first-game file (DIV-4154).
func TestOriginalQuickSpellsEmptyArrayIsTheSecondGamesOnly(t *testing.T) {
	second := secondSavedFile(t, 0, [2]uint32{1, placeTown})
	if got, err := fileCampaign(second).fileState(second); err != nil || got.quickSpells != ([4]uint32{}) {
		t.Fatalf("second game: %v %v", got, err)
	}
	first, err := sav.Open(originalContainer(sav.Magic, savedTrailer(append(savedBody(0, tenActors()), 0)), shortcutsStore()))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := fileCampaign(first).fileState(first); err == nil {
		t.Fatal("a first-game file with an empty shortcut array is accepted")
	}
}

// A mission-like file and a book past the engine's 28 slots refuse before
// any state changes (DIV-4155, DIV-4156); the town above passes.
func TestRefuseSecondFileDebt(t *testing.T) {
	open := func(b []byte) *sav.File {
		t.Helper()
		sf, err := sav.Open(b)
		if err != nil {
			t.Fatal(err)
		}
		return sf
	}
	if err := refuseSecondFileDebt(open(secondFixtureTown("town", secondFixtureHero(), 44))); err != nil {
		t.Fatalf("town: %v", err)
	}
	hero := secondFixtureHero()
	mission := labelledContainer(sav.SecondGame.Magic, "mission", secondSavedDocument(10, 44, &hero, true),
		append(shortcutsStore(), secondSavedTail(0, [2]uint32{10, placeMission})...))
	if err := refuseSecondFileDebt(open(mission)); err == nil || !strings.Contains(err.Error(), "mission 10") {
		t.Fatalf("mission: %v", err)
	}
	hero.spells, hero.bookSlots = []uint8{24, 29}, 30
	if err := refuseSecondFileDebt(open(secondFixtureTown("town", hero, 44))); err == nil || !strings.Contains(err.Error(), "spell 29") {
		t.Fatalf("spell 29: %v", err)
	}
}

func TestLeaveBagToCampaignEmptiesEveryPack(t *testing.T) {
	party := []mapload.PartyMember{{
		ID: "hero", Carried: []uint16{0x0205}, CarriedItems: []sim.ItemInstance{{Code: 0x0205}},
		Carry: &mapload.Carry{Items: []uint16{0x0205}},
	}}
	leaveBagToCampaign(party)
	m := party[0]
	if m.Carried != nil || m.CarriedItems != nil || m.Carry.Items != nil || m.ID != "hero" {
		t.Fatalf("member %+v keeps a pack", m)
	}
}

// laterTownFile is a Bsg& town at town 2 after mission 30's departure:
// mission 40 open, bank slot 768 at 30, and the tail's grid holding grid.
func laterTownFile(grid [4]secondshop.Record) ([]byte, [1024]int32) {
	tail := secondSavedTail(0, [2]uint32{2, placeTown}, [2]uint32{40, placeMission})
	var bank [1024]int32
	bank[768], bank[769], bank[896+30] = 30, 1, 1
	for i, v := range bank {
		binary.LittleEndian.PutUint32(tail[12+4*i:], uint32(v))
	}
	copy(tail[secondTailGrid:], shopGrid(grid))
	hero := secondFixtureHero()
	return labelledContainer(sav.SecondGame.Magic, "later", secondSavedBody(0, 44, &hero), append(shortcutsStore(), tail...)), bank
}

// A later town's LOAD installs the four shop records of the tail's grid
// (R2-SESSION-156, R2-SESSION-187), not the new game's; the SAVE after it
// writes them and its cold LOAD keeps them.
func TestBsgTownShopRecordsAreTheGrids(t *testing.T) {
	_, bank := laterTownFile(secondshop.NewGameRecords)
	grid := replayRecords(bank)
	grid[3][0].Max = 7777 // a value only the grid holds
	b, _ := laterTownFile(grid)
	src, err := decodeOriginalSource(b, mapload.ModContext{}, Campaign{})
	if err != nil {
		t.Fatalf("decodeOriginalSource: %v", err)
	}
	second := src.campaign.town.second
	if second == nil || second.current != (partyPlace{placeTown, 2}) {
		t.Fatalf("town %+v, want town 2", second)
	}
	if got := second.shop.records; got != grid || got == secondshop.NewGameRecords {
		t.Fatalf("records %+v, want the grid's %+v", got, grid)
	}
	saved := captureSecondCampaign(second)
	if saved.Shop == nil || saved.Shop.Records != grid {
		t.Fatalf("the SAVE writes %+v, want the grid's records", saved.Shop)
	}
	raw, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	var cold currentSecondCampaign
	if err := json.Unmarshal(raw, &cold); err != nil {
		t.Fatalf("cold LOAD: %v", err)
	}
	if got := cold.restore().shop.records; got != grid {
		t.Fatalf("the cold LOAD restores %+v, want the grid's records", got)
	}
}

// A grid cell whose draws or quantity bound the stock fill cannot run
// refuses before any state changes.
func TestBsgTownShopGridIsBounded(t *testing.T) {
	for _, bad := range []secondshop.Category{{Draws: maxSecondShopDraws + 1}, {Draws: -1}, {Bound: -1}, {Bound: maxSecondShopBound + 1}} {
		grid := secondshop.NewGameRecords
		grid[2][1] = bad
		b, _ := laterTownFile(grid)
		sf, err := sav.Open(b)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fileCampaign(sf).fileState(sf); err == nil || !strings.Contains(err.Error(), "shop record") {
			t.Fatalf("cell %+v: %v", bad, err)
		}
		if _, err := decodeOriginalSource(b, mapload.ModContext{}, Campaign{}); err == nil {
			t.Fatalf("cell %+v decodes", bad)
		}
	}
}

// The first game's Load list leaves out a Bsg& file in a shared save
// directory and keeps its own original; the second game's lists the Bsg&
// file alone (DIV-2404).
func TestFirstGameLoadListLeavesOutABsgFile(t *testing.T) {
	dir := t.TempDir()
	first := originalContainer(sav.Magic, savedTrailer(append(savedBody(0, tenActors()), 0)), shortcutsStore())
	for name, b := range map[string][]byte{"game0000.sav": secondFixtureTown("Second town", secondFixtureHero(), 44), "game0001.sav": first} {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var f FrontEnd
	rows := f.saveRows(SaveStore{Dir: dir}, OriginalStore{Dir: dir}, true)
	for _, r := range rows {
		if strings.Contains(r.Name, "game0000") {
			t.Fatalf("the first game lists the Bsg& file: %+v", rows)
		}
	}
	if len(rows) != 2 {
		t.Fatalf("first-game rows %+v, want the first-game original from both stores", rows)
	}
	empty := t.TempDir()
	if err := os.WriteFile(filepath.Join(empty, "game0000.sav"), secondFixtureTown("Second town", secondFixtureHero(), 44), 0o644); err != nil {
		t.Fatal(err)
	}
	if f.anyLoadableSaveRow(SaveStore{Dir: empty}, OriginalStore{Dir: empty}) {
		t.Fatal("the first game's Load counts the Bsg& file")
	}
	if got := ownGameSaves(dir, 0, base.GameROM2); len(got) != 1 || got[0].Name != "game0000.sav" {
		t.Fatalf("second-game rows %+v, want the Bsg& file", got)
	}
}
