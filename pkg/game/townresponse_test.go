package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/audio"
	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func responseBank(t *testing.T, files []synth.File) *SpeechBank {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "speech.res"), synth.Archive(files), 0600); err != nil {
		t.Fatal(err)
	}
	return OpenSpeech(root)
}

// TestSchoolLatchesShareDwordsAcrossClasses pins TOWN-502's latch arithmetic:
// a mage's tier t and a fighter's tier t+1 of one slot share a dword, and the
// next slot's first fighter tier follows the last mage tier.
func TestSchoolLatchesShareDwordsAcrossClasses(t *testing.T) {
	for slot := 0; slot < 5; slot++ {
		for tier := 0; tier < 2; tier++ {
			if schoolLatchIndex(true, slot, tier) != schoolLatchIndex(false, slot, tier+1) {
				t.Fatalf("slot %d: mage tier %d and fighter tier %d use different latches", slot, tier, tier+1)
			}
		}
	}
	if schoolLatchIndex(true, 0, 2) != schoolLatchIndex(false, 1, 0) || schoolLatchIndex(true, 4, 2) != schoolLatchCount-1 {
		t.Fatal("the last mage tier does not meet the next slot's first fighter tier, or the table is the wrong size")
	}
}

// TestSchoolTeacherSpeaksAfterAPaidStepOncePerLatch is TOWN-502: selecting a
// skill requests no speech; a paid training step requests the speech of the
// trained skill's tier (up to 20, up to 50, above) once per latch, then posts
// the teaching sound; an unpaid step requests nothing; entering the room sets
// every latch again.
func TestSchoolTeacherSpeaksAfterAPaidStepOncePerLatch(t *testing.T) {
	f := shellFrontEnd()
	f.Town.gold = 1 << 20
	f.SpeechBank = responseBank(t, []synth.File{
		{Path: "training/npc33s3l1.wav", Data: speechWAV(1001)},
		{Path: "training/npc33s3l2.wav", Data: speechWAV(1002)},
		{Path: "training/npc33s3l3.wav", Data: speechWAV(1003)},
		{Path: "training/npc34s4l1.wav", Data: speechWAV(2001)},
		{Path: "training/npc34s4l3.wav", Data: speechWAV(2003)},
	})
	voice, sound := &tavernInteriorRecorder{}, &schoolRotateRecorder{}
	f.SpeechPlayer, f.SoundPlayer = voice, sound
	f.SoundBank = &SoundBank{named: map[string]soundCacheEntry{
		schoolTeachSound: {sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{77}}, ok: true},
	}}
	s := f.townUI
	s.atSquare()
	s.Choose(2)
	cell := func(i int) {
		s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: i}, false)
	}
	train := func() {
		s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 0}, false)
	}
	spoke := func(step string, want ...int16) {
		t.Helper()
		if len(voice.samples) != len(want) {
			t.Fatalf("%s: %d speech requests, want %d", step, len(voice.samples), len(want))
		}
		for i, pcm := range want {
			if voice.samples[i].PCM[0] != pcm {
				t.Fatalf("%s: speech request %d played %d, want %d", step, i, voice.samples[i].PCM[0], pcm)
			}
		}
	}
	taught := func(step string, want int) {
		t.Helper()
		if len(sound.samples) != want {
			t.Fatalf("%s: %d teaching sounds, want %d", step, len(sound.samples), want)
		}
	}

	gold := f.Town.Gold()
	cell(2)
	spoke("selecting the club skill")
	taught("selecting the club skill", 0)
	if f.Town.Gold() != gold {
		t.Fatal("selecting a skill spent gold")
	}
	f.Carried[0].Hero.Skill[3] = 5
	train()
	spoke("first step, level 6", 1001)
	taught("first step", 1)
	train()
	spoke("second step in the same tier", 1001)
	taught("second step", 2)
	f.Carried[0].Hero.Skill[3] = 20
	train()
	spoke("level 21 is the second tier", 1001, 1002)
	f.Carried[0].Hero.Skill[3] = 50
	train()
	spoke("level 51 is the third tier", 1001, 1002, 1003)
	taught("four paid steps", 4)

	// A refused step requests neither.
	f.Town.gold = 0
	train()
	spoke("an unpaid step", 1001, 1002, 1003)
	taught("an unpaid step", 4)
	f.Town.gold = 1 << 20

	// Entering the room sets the latches again.
	s.Back()
	s.Choose(2)
	cell(2)
	f.Carried[0].Hero.Skill[3] = 5
	train()
	spoke("a second visit", 1001, 1002, 1003, 1001)

	// The mage's key is npc34, with its own latch.
	f.Carried[0].Mage = true
	s.resetSchoolColumn()
	cell(8)
	f.Carried[0].Hero.Skill[4] = 70
	train()
	spoke("mage level 71", 1001, 1002, 1003, 1001, 2003)
	f.Carried[0].Hero.Skill[4] = 3
	train()
	spoke("mage level 4", 1001, 1002, 1003, 1001, 2003, 2001)
}

func TestShopSelectionSpeaksItemThenEffectsAndNewSelectionReplacesIt(t *testing.T) {
	f := shellFrontEnd()
	f.SpeechBank = responseBank(t, []synth.File{
		{Path: "shop/s01i02p1.wav", Data: speechWAV(1000)},
		{Path: "shop/s01i02p2.wav", Data: speechWAV(2000)},
		{Path: "shop/effects/02.wav", Data: speechWAV(3000)},
		{Path: "shop/books/03.wav", Data: speechWAV(4000)},
	})
	r := &tavernInteriorRecorder{}
	f.SpeechPlayer = r
	f.Shop = NewShop(1000)
	f.Shop.shelves[ShelfWeapons] = []ShopItem{
		{Code: data.ComposeItemCode(2, 1, 3, 2), Count: 1, Price: 100, Effects: []sim.ItemEffect{{Kind: 2}, {Kind: 2}}},
		{Code: 0x0e15, Kind: 5, Count: 1, Price: 100, Effects: []sim.ItemEffect{{Kind: 42, Operand: 3}}},
		{Code: 0x0101, Count: 1, Price: 100},
	}
	s := f.townUI
	s.atSquare()
	s.Choose(1)
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlShelfCell, Index: 0})
	if len(r.samples) != 1 || r.samples[0].PCM[0] != 1000 || len(f.Shop.Table()) != 1 {
		t.Fatal("shelf click did not move the selected item and speak its description")
	}
	s.TownDialogueActive(true)
	if !r.voices[0].Playing() || len(r.samples) != 1 {
		t.Fatal("ordinary town update interrupted or replayed the shop response")
	}
	r.voices[0].playing = false
	s.TownDialogueActive(true)
	if len(r.samples) != 2 || r.samples[1].PCM[0] != 2000 {
		t.Fatal("description did not continue when its first recording ended")
	}
	r.voices[1].playing = false
	s.TownDialogueActive(true)
	if len(r.samples) != 3 || r.samples[2].PCM[0] != 3000 {
		t.Fatal("missing optional part blocked the item's effect description")
	}
	r.voices[2].playing = false
	s.TownDialogueActive(true)
	if len(r.samples) != 3 {
		t.Fatal("duplicate effect replayed its description")
	}
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlShelfCell, Index: 0})
	if len(r.samples) != 4 || r.samples[3].PCM[0] != 4000 {
		t.Fatal("book selection did not describe its actual spell")
	}
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlMerchant})
	if !r.voices[3].Playing() || len(r.samples) != 4 {
		t.Fatal("a press on the merchant restarted or stopped the current item's description")
	}
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlShelfCell, Index: 0})
	if r.voices[3].Playing() || len(r.samples) != 4 || len(f.Shop.Table()) != 3 {
		t.Fatal("an unvoiced item kept the previous description or prevented trading")
	}
}
