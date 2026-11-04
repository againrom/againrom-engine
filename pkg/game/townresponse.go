package game

import (
	"fmt"
	"slices"

	"againrom/pkg/audio"
	"againrom/pkg/random"
	"againrom/pkg/rules"
)

// Responses are optional presentation around an explicit item/skill choice.
// DIV-1263 records the inferred installed names and authored trigger/tier rules.
func (t *townScreen) startTownResponse(names []string) {
	if t == nil || t.sess == nil {
		return
	}
	room := townRoomNames[t.roomName()]
	if room != roomShop && room != roomSchool {
		return
	}
	t.resetTownSpeech()
	t.speech.room, t.speech.pending = room, names
	t.advanceTownResponse()
}

func (t *townScreen) advanceTownResponse() {
	if t.speech.voice != nil {
		if t.speech.voice.Playing() {
			return
		}
		audio.DestroyVoice(t.speech.voice)
		t.speech.voice = nil
	}
	for len(t.speech.pending) > 0 {
		name := t.speech.pending[0]
		t.speech.pending = t.speech.pending[1:]
		if t.startTownVoice(name) {
			return
		}
	}
}

// schoolLatchCount is the size of the teacher speech latch table: the index
// class + 6*slot + 2*tier reaches 30 for the last mage tier (TOWN-502).
const schoolLatchCount = 31

// schoolSpeechTier is the tier of a skill level: 0 up to 20, 1 up to 50,
// otherwise 2. The speech key carries the tier plus one (TOWN-502).
func schoolSpeechTier(level int32) int {
	switch {
	case level <= 20:
		return 0
	case level <= 50:
		return 1
	}
	return 2
}

// schoolLatchIndex is the latch dword a training step reads: class is 0 for a
// fighter and 2 for a mage, slot is the 0-based skill slot and tier the speech
// tier. A mage's tier t and a fighter's tier t+1 of one slot share a dword
// (TOWN-502, Medium).
func schoolLatchIndex(mage bool, slot0, tier int) int {
	class := 0
	if mage {
		class = 2
	}
	return class + 6*slot0 + 2*tier
}

// speakSchoolTraining is the mouse-up handler's request after a paid training
// step (TOWN-502): the teacher speech of the trained skill's tier, once per
// latch, then the teaching sound. Selecting a skill requests no speech.
func (t *townScreen) speakSchoolTraining(slot int) {
	if t == nil || t.sess == nil {
		return
	}
	party, member := t.shopParty(), t.shopMemberIndex()
	if member < 0 || member >= len(party) || slot < 1 || slot > 5 {
		return
	}
	m := party[member]
	tier := schoolSpeechTier(m.Hero.Skill[slot])
	if latch := schoolLatchIndex(m.Mage, slot-1, tier); !t.schoolSpent[latch] {
		t.schoolSpent[latch] = true
		npc := 33
		if m.Mage {
			npc = 34
		}
		t.startTownResponse([]string{fmt.Sprintf("speech/training/npc%ds%dl%d.wav", npc, slot, tier+1)})
	}
	t.raiseRoomEvent(roomSchool, roomEventTeach)
}

// shopItemSpeechPercent bounds the tray draw (SHOP-109).
const shopItemSpeechPercent = 30

type shopSpeechRules struct {
	dir           string
	effects       [][2]uint8
	bookTag       uint8
	kindMask      int
	numberMask    int
	kinds         [2]int
	filterMissing bool
	rawScaled     bool
	tableRelease  bool
}

var firstShopSpeech = shopSpeechRules{
	dir: "shop", effects: [][2]uint8{{1, 40}}, kindMask: 255, numberMask: 255,
	kinds: [2]int{1, 13}, filterMissing: true,
}

var secondShopSpeech = shopSpeechRules{
	dir: "shop", effects: [][2]uint8{{2, 11}, {21, 25}, {27, 31}, {33, 37}},
	bookTag: 42, kindMask: 15, numberMask: 31, kinds: [2]int{0, 15}, rawScaled: true, tableRelease: true,
}

// shopItemResponse requests one table item's recording (SHOP-109, R2-ENGINE-862).
func (t *townScreen) shopItemResponse(item ShopItem) {
	if t.draws == nil {
		return
	}
	rules := t.row.speech
	if rules.effects == nil {
		rules = firstShopSpeech
	}
	if d := t.townDescription(); d != nil {
		if room := d.Room("shop"); room != nil && room.Shop != nil && room.Shop.SpeechDir != "" {
			rules.dir = room.Shop.SpeechDir
		}
	}
	draw := func(n int) int {
		if rules.rawScaled {
			return sceneRawFrom(t.draws, "idle") * n / random.RandMax % n
		}
		return t.sceneDraw("idle", n)
	}
	if draw(100) >= shopItemSpeechPercent {
		return
	}
	present := func(name string) bool {
		_, found := t.in.SpeechBank.Sample(name)
		return found
	}
	var names []string
	if name, ok := shopItemSpeech(item, spellGame(t.in.Table), rules, present, draw); ok {
		names = []string{name}
	}
	t.startTownResponse(names)
}

// shopItemSpeech names the one recording of an item of game, or none.
func shopItemSpeech(item ShopItem, game rules.SpellGame, names shopSpeechRules, present func(string) bool, draw func(int) int) (string, bool) {
	accept := func(name string) bool { return !names.filterMissing || present(name) }
	for _, effect := range item.Effects {
		if slices.ContainsFunc(names.effects, func(span [2]uint8) bool { return effect.Kind >= span[0] && effect.Kind <= span[1] }) {
			if name := fmt.Sprintf("speech/%s/effects/%02d.wav", names.dir, effect.Kind); accept(name) {
				return name, true
			}
		}
	}
	spell, ok := 0, false
	if names.bookTag != 0 {
		for _, effect := range item.Effects {
			if effect.Kind == names.bookTag {
				spell = int(effect.Operand)
			}
		}
		ok = spell != 0
	} else {
		instance := item.Instance()
		book, found := instance.BookSpell(game)
		spell, ok = int(book), found
		if !ok {
			var cast uint16
			if cast, _, ok = instance.CastSpell(); ok {
				spell, ok = int(cast), cast > 0 && cast < 32
			}
		}
	}
	if ok {
		if name := fmt.Sprintf("speech/%s/books/%02d.wav", names.dir, spell); accept(name) {
			return name, true
		}
	}
	kind, number := int(item.Code.B())&names.kindMask, int(item.Code.D())&names.numberMask
	if kind < names.kinds[0] || kind > names.kinds[1] {
		return "", false
	}
	var parts []string
	for part := 1; part <= 3; part++ {
		if name := fmt.Sprintf("speech/%s/s%02di%02dp%d.wav", names.dir, kind, number, part); accept(name) {
			parts = append(parts, name)
		}
	}
	if len(parts) == 0 {
		return "", false
	}
	return parts[draw(len(parts))%len(parts)], true
}
