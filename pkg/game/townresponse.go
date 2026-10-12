package game

import (
	"fmt"

	"againrom/pkg/audio"
)

// Responses are optional presentation around an explicit item/skill choice.
// DIV-1263 records the inferred installed names and authored trigger/tier rules.
func (t *townScreen) startTownResponse(names []string) {
	if t == nil || t.sess == nil || (t.room != roomShop && t.room != roomSchool) {
		return
	}
	t.resetTownSpeech()
	t.speech.room, t.speech.pending = t.room, names
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

// schoolTeachSound is the sound posted after a paid training step (TOWN-502).
const schoolTeachSound = "town/school/teach.wav"

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
	t.requestSchoolSound(schoolTeachSound)
}

// shopItemResponse starts the one recording a merchant stack moved onto the
// table speaks (SHOP-109): its effect description, else its spell
// description, else one part of its class/row lines drawn from the shop's
// presentation seam. A recording the install lacks is passed over, and an
// item with none stops the previous one. DIV-2115 keeps a request on every
// selection instead of the original's draw gate.
func (t *townScreen) shopItemResponse(item ShopItem) {
	present := func(name string) bool {
		_, found := t.in.SpeechBank.Sample(name)
		return found
	}
	var names []string
	if name, ok := shopItemSpeech(item, present, func(n int) int { return t.sceneDraw("idle", n) }); ok {
		names = []string{name}
	}
	t.startTownResponse(names)
}

// shopItemSpeech names the one recording of an item, or none.
func shopItemSpeech(item ShopItem, present func(string) bool, draw func(int) int) (string, bool) {
	for _, effect := range item.Effects {
		if effect.Kind > 0 && effect.Kind < 41 {
			if name := fmt.Sprintf("speech/shop/effects/%02d.wav", effect.Kind); present(name) {
				return name, true
			}
		}
	}
	instance := item.Instance()
	spell, ok := instance.BookSpell()
	if !ok {
		var cast uint16
		if cast, _, ok = instance.CastSpell(); ok {
			spell, ok = cast, cast > 0 && cast < 32
		}
	}
	if ok {
		if name := fmt.Sprintf("speech/shop/books/%02d.wav", spell); present(name) {
			return name, true
		}
	}
	if item.Code.B() <= 0 || item.Code.B() >= 14 {
		return "", false
	}
	var parts []string
	for part := 1; part <= 3; part++ {
		if name := fmt.Sprintf("speech/shop/s%02di%02dp%d.wav", item.Code.B(), item.Code.D(), part); present(name) {
			parts = append(parts, name)
		}
	}
	if len(parts) == 0 {
		return "", false
	}
	return parts[draw(len(parts))%len(parts)], true
}
