package game

import (
	"slices"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondVoiceBank holds the named second-game recordings; recording i of the
// list carries code i+1.
func secondVoiceBank(paths ...string) *SoundBank {
	named := map[string]soundCacheEntry{}
	for i, p := range paths {
		named[p] = soundCacheEntry{sample: audio.Sample{PCM: []int16{int16(i + 1)}}, ok: true}
	}
	return &SoundBank{named: named}
}

// secondVoiceMission makes mw a second-game mission whose party is the given
// members, entity i+1 for member i, and whose placements carry words.
func secondVoiceMission(mw *mapWorld, party []mapload.PartyMember, words map[sim.EntityID]uint16) {
	second := base.GameROM2.Edition()
	ids := make([]sim.EntityID, len(party))
	for i := range party {
		ids[i] = sim.EntityID(i + 1)
	}
	mw.mission = &missionNotices{table: &mapload.Table{Edition: &second}, party: party, ids: ids, npcKeys: words}
}

var (
	voiceHeroMember      = mapload.PartyMember{ID: "hero", StartingHero: true, PlayerCharacter: true}
	voiceCompanionMember = func(npc int) mapload.PartyMember {
		return mapload.PartyMember{ID: "npc", PlayerCharacter: true, CompanionNPC: npc}
	}
	voiceHireMember = func(id uint8, unit bool) mapload.PartyMember {
		return mapload.PartyMember{ID: "hire", Temporary: true, CampaignHire: id, CampaignUnit: unit}
	}
)

// TestSecondGameSelectionSpeaksTheIdentityVoice checks the voice each
// identity word selects (R2-ENGINE-718) through the selection reply: the hero
// by sex and class, the two companions, the hire classes and the generic
// voices of a placed person.
func TestSecondGameSelectionSpeaksTheIdentityVoice(t *testing.T) {
	troll := voicePerson{typeID: 0x40, owner: sim.SelfSlot, hp: 10}
	xbow := voicePerson{typeID: 14, dir: data.FigureDirManFighter, weapon: true, owner: sim.SelfSlot, hp: 10}
	womanMage := voicePerson{typeID: 0x17, dir: data.FigureDirWomanMage, owner: sim.SelfSlot, hp: 10}
	for _, tc := range []struct {
		name   string
		person voicePerson
		member *mapload.PartyMember
		word   uint16
		voice  string
	}{
		{"a male fighter hero", heroMan, &voiceHeroMember, 0, "aldor"},
		{"a female fighter hero", heroWoman, &voiceHeroMember, 0, "sagita"},
		{"a male mage hero", voicePerson{typeID: sim.HeroTypeID(true, false), dir: data.FigureDirManMage, owner: sim.SelfSlot, hp: 10}, &voiceHeroMember, 0, "hildarius"},
		{"a female mage hero", voicePerson{typeID: sim.HeroTypeID(true, true), dir: data.FigureDirWomanMage, owner: sim.SelfSlot, hp: 10}, &voiceHeroMember, 0, "galinel"},
		{"companion 22", heroMan, ptr(voiceCompanionMember(22)), 0, "igles"},
		{"companion 23", heroWoman, ptr(voiceCompanionMember(23)), 0, "dina"},
		{"hire 1, a person", xbow, ptr(voiceHireMember(1, false)), 0, "xbow"},
		{"hire 4, a creature", troll, ptr(voiceHireMember(4, true)), 0, "troll"},
		{"hire 6", womanMage, ptr(voiceHireMember(6, false)), 0, "mage"},
		{"hire 9, a generic creature", troll, ptr(voiceHireMember(9, true)), 0, "m_peasant"},
		{"a placed armed man", armedMan, nil, 100, "m_fighter"},
		{"a placed unarmed woman", bareWoman, nil, 100, "f_peasant"},
		{"a placed woman mage", womanMage, nil, 100, "f_mage"},
		{"a placed person with no word", armedMan, nil, 0, "m_fighter"},
	} {
		mw := voiceFixture(t, tc.person)
		var party []mapload.PartyMember
		var words map[sim.EntityID]uint16
		if tc.member != nil {
			party = []mapload.PartyMember{*tc.member}
		} else if tc.word != 0 {
			words = map[sim.EntityID]uint16{1: tc.word}
		}
		secondVoiceMission(mw, party, words)
		want := "characters/" + tc.voice + "/select/1.wav"
		f, voices := voiceFront(mw, secondVoiceBank(want))
		script := &scriptedDraw{}
		_, selection := f.runtimeAudio().unitReplies(mw, script.draw)
		script.set(0, 0)
		selection([]uint32{1}, time.Unix(100, 0))
		if got := heard(voices, 0); !slices.Equal(got, []int16{1}) || script.used != 2 {
			t.Errorf("%s: heard %v after %d draws, want %s after 2", tc.name, got, script.used, want)
		}
	}
	// A creature with no hire word never speaks and takes no draw.
	mw := voiceFixture(t, troll)
	secondVoiceMission(mw, nil, map[sim.EntityID]uint16{1: 21})
	f, voices := voiceFront(mw, secondVoiceBank("characters/aldor/select/1.wav", "characters/m_peasant/select/1.wav"))
	script := &scriptedDraw{}
	_, selection := f.runtimeAudio().unitReplies(mw, script.draw)
	selection([]uint32{1}, time.Unix(100, 0))
	if len(voices.samples) != 0 || script.used != 0 {
		t.Errorf("a creature with word 21: %d recordings after %d draws, want none", len(voices.samples), script.used)
	}
}

func ptr[T any](v T) *T { return &v }

// TestSecondGameSpeakerIsDrawnAmongTheTiers checks the picker's lists
// (R2-ENGINE-719): heroes first, then armed persons and hires 1..20, then
// unarmed persons, by draw × n / 32767.
func TestSecondGameSpeakerIsDrawnAmongTheTiers(t *testing.T) {
	troll := voicePerson{typeID: 0x40, owner: sim.SelfSlot, hp: 10}
	dead := heroWoman
	dead.hp = 0
	mw := voiceFixture(t, heroMan, troll, bareMan, armedMan, troll, dead)
	secondVoiceMission(mw, []mapload.PartyMember{voiceHeroMember, voiceHireMember(4, true), voiceHireMember(2, false), voiceHireMember(3, false), voiceHireMember(25, true), voiceCompanionMember(22)}, nil)
	const hero, hire, bare, armed, outside, fallen = 1, 2, 3, 4, 5, 6
	script := &scriptedDraw{}
	for _, tc := range []struct {
		name string
		ids  []uint32
		draw int
		want uint32
		ok   bool
	}{
		{"the hero before every other", []uint32{bare, armed, hire, hero}, 0, hero, true},
		{"a dead companion is no candidate", []uint32{fallen, bare}, 0, bare, true},
		{"a creature hire beside an armed person, low draw", []uint32{bare, hire, armed}, 0, hire, true},
		{"a creature hire beside an armed person, high draw", []uint32{bare, hire, armed}, 32766, armed, true},
		{"an unarmed person alone", []uint32{bare, outside}, 0, bare, true},
		{"a creature with word 25 never speaks", []uint32{outside}, 0, 0, false},
		{"the slot one past the last member", []uint32{hire, armed}, 32767, 0, false},
	} {
		script.set(tc.draw)
		e, _, ok := mw.speakerOf(tc.ids, script.draw)
		if uint32(e.ID) != tc.want || ok != tc.ok {
			t.Errorf("%s: speaker %d (%v), want %d (%v)", tc.name, e.ID, ok, tc.want, tc.ok)
		}
	}
}

// TestSecondGameGesturesPlayTheirClaimedRecording runs the selection and each
// order gesture of a hero whose archive holds three files of each numbered
// gesture (R2-ENGINE-717, R2-ENGINE-718, R2-ENGINE-719, R2-ASSET-185).
func TestSecondGameGesturesPlayTheirClaimedRecording(t *testing.T) {
	var paths []string
	for _, g := range []string{"select", "move", "attack", "swarm"} {
		for n := 1; n <= 3; n++ {
			paths = append(paths, "characters/galinel/"+g+"/"+string(rune('0'+n))+".wav")
		}
	}
	paths = append(paths, "characters/galinel/defend/1.wav", "characters/galinel/retreat/1.wav", "characters/galinel/pickup/1.wav",
		"characters/galinel/defend/2.wav", "characters/galinel/pickup/2.wav")
	code := func(p string) int16 { return int16(slices.Index(paths, p) + 1) }
	mw := voiceFixture(t, voicePerson{typeID: sim.HeroTypeID(true, true), dir: data.FigureDirWomanMage, owner: sim.SelfSlot, hp: 10})
	secondVoiceMission(mw, []mapload.PartyMember{voiceHeroMember}, nil)
	f, voices := voiceFront(mw, secondVoiceBank(paths...))
	script := &scriptedDraw{}
	command, selection := f.runtimeAudio().unitReplies(mw, script.draw)
	now := time.Unix(100, 0)
	for _, tc := range []struct {
		gesture ui.VoiceGesture
		draws   []int
		want    string
		drawn   int
	}{
		{ui.VoiceMove, []int{0, 0}, "move/1", 2},
		{ui.VoiceMove, []int{0, 10923}, "move/2", 2},
		{ui.VoiceAttack, []int{0, 21845}, "attack/2", 2},
		{ui.VoiceAttack, []int{0, 21846}, "attack/3", 2},
		{ui.VoiceSwarm, []int{0, 32767}, "swarm/3", 2},
		{ui.VoiceGuard, []int{0}, "defend/1", 1},
		{ui.VoiceStandGround, []int{0}, "defend/1", 1},
		{ui.VoiceDefend, []int{0}, "defend/1", 1},
		{ui.VoiceRetreat, []int{0}, "retreat/1", 1},
		{ui.VoicePickup, []int{0}, "pickup/1", 1},
		{ui.VoicePatrol, nil, "", 0},
		{ui.VoiceTown, nil, "", 0},
	} {
		now = now.Add(time.Minute)
		mark := len(voices.samples)
		script.set(tc.draws...)
		command(tc.gesture, []uint32{1}, now)
		var want []int16
		if tc.want != "" {
			want = []int16{code("characters/galinel/" + tc.want + ".wav")}
		}
		if got := heard(voices, mark); !slices.Equal(got, want) || script.used != tc.drawn {
			t.Errorf("gesture %d: heard %v after %d draws, want %v (%s) after %d", tc.gesture, got, script.used, want, tc.want, tc.drawn)
		}
	}
	for _, tc := range []struct {
		draw int
		want string
	}{{0, "select/1"}, {10922, "select/1"}, {10923, "select/2"}, {32767, "select/3"}} {
		now = now.Add(time.Minute)
		mark := len(voices.samples)
		script.set(0, tc.draw)
		selection([]uint32{1}, now)
		if got := heard(voices, mark); !slices.Equal(got, []int16{code("characters/galinel/" + tc.want + ".wav")}) || script.used != 2 {
			t.Errorf("selection at draw %d: heard %v after %d draws, want %s", tc.draw, got, script.used, tc.want)
		}
	}
	// The option off is silent and draws nothing.
	f.speech.acknowledgmentsOff = true
	script.set()
	selection([]uint32{1}, now.Add(time.Hour))
	command(ui.VoiceAttack, []uint32{1}, now.Add(2*time.Hour))
	if got := len(voices.samples); got != 14 || script.used != 0 {
		t.Errorf("the option off: %d recordings after %d draws, want 14 and none", got, script.used)
	}
}

// TestSecondGameCountedGestureWithNoFileStoresTheStamp checks a numbered
// gesture with no file held: no recording, no reader draw, and the speaker's
// stamp stored (R2-ENGINE-718, R2-ENGINE-719).
func TestSecondGameCountedGestureWithNoFileStoresTheStamp(t *testing.T) {
	mw := voiceFixture(t, armedMan)
	secondVoiceMission(mw, nil, map[sim.EntityID]uint16{1: 100})
	f, voices := voiceFront(mw, secondVoiceBank("characters/m_fighter/defend/1.wav", "characters/m_fighter/move/1.wav"))
	script := &scriptedDraw{}
	command, _ := f.runtimeAudio().unitReplies(mw, script.draw)
	t0 := time.Unix(100, 0)
	script.set(0)
	command(ui.VoiceAttack, []uint32{1}, t0)
	if len(voices.samples) != 0 || script.used != 1 {
		t.Fatalf("attack with no file: %d recordings after %d draws, want none after 1", len(voices.samples), script.used)
	}
	script.set(0, 0)
	command(ui.VoiceDefend, []uint32{1}, t0.Add(2999*time.Millisecond))
	if len(voices.samples) != 0 {
		t.Fatal("the stamp of the silent attack was not stored")
	}
	script.set(0, 0)
	command(ui.VoiceMove, []uint32{1}, t0.Add(3000*time.Millisecond))
	if got := heard(voices, 0); !slices.Equal(got, []int16{2}) || script.used != 2 {
		t.Fatalf("move with one file: heard %v after %d draws, want [2] after 2", got, script.used)
	}
}

// TestFirstGameVoicesAreUnchangedByTheRules runs every first-game gesture and
// the selection through the reply with a second-game recording beside each
// first-game one: the first game plays its own bank and leaf.
func TestFirstGameVoicesAreUnchangedByTheRules(t *testing.T) {
	bank := voiceBankFixture()
	for _, name := range []string{"characters/aldor/select/1.wav", "characters/aldor/move/1.wav", "characters/aldor/defend/1.wav"} {
		bank.named[name] = soundCacheEntry{sample: audio.Sample{PCM: []int16{-1}}, ok: true}
	}
	mw := voiceFixture(t, heroMan)
	mw.mission = &missionNotices{party: []mapload.PartyMember{voiceHeroMember}, ids: []sim.EntityID{1}}
	f, voices := voiceFront(mw, bank)
	script := &scriptedDraw{}
	command, selection := f.runtimeAudio().unitReplies(mw, script.draw)
	now := time.Unix(100, 0)
	for gesture, leaf := range map[ui.VoiceGesture]string{
		ui.VoiceMove: "command2", ui.VoiceAttack: "command2", ui.VoiceSwarm: "command2", ui.VoicePatrol: "command2",
		ui.VoiceTown: "command2", ui.VoiceGuard: "defend", ui.VoiceStandGround: "defend", ui.VoiceDefend: "defend",
		ui.VoiceRetreat: "retreat", ui.VoicePickup: "idle",
	} {
		now = now.Add(time.Minute)
		mark := len(voices.samples)
		script.set(0, 8192)
		command(gesture, []uint32{1}, now)
		if got := heard(voices, mark); !slices.Equal(got, []int16{voiceSample("mf_hero", leaf)}) {
			t.Errorf("gesture %d: heard %v, want mf_hero/%s", gesture, got, leaf)
		}
	}
	mark := len(voices.samples)
	script.set(0, 16384)
	selection([]uint32{1}, now.Add(time.Minute))
	if got := heard(voices, mark); !slices.Equal(got, []int16{voiceSample("mf_hero", "select2")}) || script.used != 2 {
		t.Errorf("selection: heard %v after %d draws, want mf_hero/select2 after 2", got, script.used)
	}
}
