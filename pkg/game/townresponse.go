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

func shopItemSpeech(item ShopItem) []string {
	var names []string
	add := func(name string) {
		for _, old := range names {
			if old == name {
				return
			}
		}
		if len(names) < 16 {
			names = append(names, name)
		}
	}
	if item.Code.B() > 0 && item.Code.B() < 14 {
		for part := 1; part <= 3; part++ {
			add(fmt.Sprintf("speech/shop/s%02di%02dp%d.wav", item.Code.B(), item.Code.D(), part))
		}
	}
	instance := item.Instance()
	if spell, ok := instance.BookSpell(); ok {
		add(fmt.Sprintf("speech/shop/books/%02d.wav", spell))
	} else if spell, _, ok := instance.CastSpell(); ok && spell > 0 && spell < 32 {
		add(fmt.Sprintf("speech/shop/books/%02d.wav", spell))
	}
	for _, effect := range item.Effects {
		if effect.Kind > 0 && effect.Kind < 41 {
			add(fmt.Sprintf("speech/shop/effects/%02d.wav", effect.Kind))
		}
	}
	return names
}
