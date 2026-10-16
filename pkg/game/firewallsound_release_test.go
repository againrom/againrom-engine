package game

import (
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// fireWallVoice is a recorded voice that keeps playing until stopped.
type fireWallVoice struct{ stopped bool }

func (v *fireWallVoice) Playing() bool { return !v.stopped }
func (v *fireWallVoice) Stop()         { v.stopped = true }

// fireWallSounds records every effects request and every ambient loop
// request and stop, in order.
type fireWallSounds struct{ log []string }

func (r *fireWallSounds) Play(audio.Sample, audio.Placement) {}
func (r *fireWallSounds) RequestSample(_ audio.Sample, q audio.Request) audio.Voice {
	r.log = append(r.log, "effect "+q.Source+" "+q.Selector)
	return &fireWallVoice{}
}
func (r *fireWallSounds) RequestLoop(k ui.AmbientLoop, _ audio.Sample, q audio.Request) audio.Voice {
	r.log = append(r.log, fmt.Sprintf("loop %d %s", k, q.Selector))
	return &fireWallVoice{}
}
func (r *fireWallSounds) StartLoop(ui.AmbientLoop, audio.Sample, audio.Placement) {}
func (r *fireWallSounds) MoveLoop(ui.AmbientLoop, audio.Placement)                {}
func (r *fireWallSounds) StopLoop(k ui.AmbientLoop) {
	r.log = append(r.log, fmt.Sprintf("stop %d", k))
}
func (r *fireWallSounds) Stop()                      {}
func (r *fireWallSounds) SetSettings(audio.Settings) {}

// A mage's Fire Wall cast through App input in mission 41, with the camera
// still, starts the wall's fire loop (sfx.reg slot 90, VIDEO-SFX-085) while
// the wall stands and stops it when the wall burns out (DIV-3153). Loss
// control: before the fix this route requested no loop at all until a camera
// move or the ambient deadline, which falls after the wall's end here; the
// same cast at 0.113.1 (93d12014) and before the spell presentation table
// (c3835e74) requested none either.
func TestReleaseFireWallSoundsWithTheCameraStill(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	r := &fireWallSounds{}
	f.SoundPlayer, f.AmbientPlayer = r, r
	f.speech.player, f.MusicPlayer, f.CutsceneAudioPlayer = nil, nil, nil
	party := MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
	a := f.App("fire wall sound")
	a.SetCutscenes(nil)
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpenerWith(41, party)); err != nil {
		t.Fatal(err)
	}
	for n := 0; a.HeadlessNoticeOpen() && n < 32; n++ {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	releasePauseMission(t, f, a)
	id := f.live.mission.ids[0]
	e, _ := f.live.entity(id)
	book := e.Book
	book.State = sim.BookPresent
	rule, ok := f.live.world.Spell(3)
	if !ok {
		t.Fatal("the install has no Fire Wall")
	}
	book.Slots[2] = sim.BookSpell{Range: rule.MaxRange, ManaCost: uint16(rule.ManaCost)}
	if err := f.live.world.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{ID: id, KnownSpells: e.KnownSpells | 1<<3, Book: book}}); err != nil {
		t.Fatal(err)
	}
	f.live.push()
	at, ok := cityMageNearest(cityMageCastCells(t, f, a, id, 3))
	if !ok {
		t.Fatal("no visible cell admits Fire Wall")
	}
	cityMageCastInput(t, f, a, id, 3, at)
	for _, edge := range []string{"right-press", "right-release"} {
		if err := a.HeadlessPointer(edge, 512, 200); err != nil {
			t.Fatal(err)
		}
	}
	activePauseResume(t, a)
	camera := *f.live.view.Camera()
	r.log = nil
	stood, burnt := -1, -1
	standsAt, burntAt := -1, -1
	for n := 0; n < 300 && burnt < 0; n++ {
		before := len(r.log)
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessPointer("hover", 512, 200); err != nil {
			t.Fatal(err)
		}
		cells := len(wallFireAmbientCells(f.live.cellEffectsByArm()))
		switch {
		case stood < 0 && cells != 0:
			stood, standsAt = n, before
		case stood >= 0 && cells == 0:
			burnt, burntAt = n, before
		}
	}
	if c := f.live.view.Camera(); c.X != camera.X || c.Y != camera.Y {
		t.Fatal("the camera moved during the witness")
	}
	loop := fmt.Sprintf("loop %d registry:90", ui.AmbientWallOfFire)
	stop := fmt.Sprintf("stop %d", ui.AmbientWallOfFire)
	loopAt, stopAt, loops := -1, -1, 0
	for i, line := range r.log {
		switch line {
		case loop:
			loops++
			if loopAt < 0 {
				loopAt = i
			}
		case stop:
			stopAt = i
		}
	}
	log := strings.Join(r.log, "; ")
	if stood < 0 || burnt < 0 || loops != 1 || loopAt < standsAt || loopAt >= burntAt || stopAt < burntAt {
		t.Fatalf("Fire Wall stood at frame %d and burnt out at %d; sounds: %s", stood, burnt, log)
	}
	t.Logf("Fire Wall stood frames %d..%d; sounds: %s", stood, burnt, log)
}

// Every first-game spell's cast and cell sound slots are those before the
// spell presentation table (c3835e74): the cast slot 500 plus the even cast
// picture and the cell slot 500 plus the odd burst picture, slot 551 for the
// waiting storm burst; each slot's installed sound is named. Fire Wall has
// neither in either root; its sound is the fire loop.
func TestReleaseFirstGameSpellSoundsKeepTheirSlots(t *testing.T) {
	f := releaseFront(t)
	bank := f.SoundBank
	if bank == nil {
		t.Fatal("the install has no sound bank")
	}
	set := data.FirstSpellPresentation()
	casts, cells := 0, 0
	var named []string
	for spell := uint16(1); spell <= 28; spell++ {
		wantCast, wantCell := 500+data.CastPicture(int(spell)), 500+data.BurstPicture(int(spell))
		if data.BurstPicture(int(spell)) == 51 {
			wantCell = 551
		}
		cast, cell := castSpellSoundSlot(set, spell), effectSpellSoundSlot(set, spell)
		if cast != wantCast || cell != wantCell {
			t.Fatalf("spell %d sounds at slots %d and %d, want %d and %d", spell, cast, cell, wantCast, wantCell)
		}
		castName, cellName := "-", "-"
		if _, ok := bank.Sample(cast); ok {
			casts++
			castName = bank.entries[cast]
		}
		if _, ok := bank.Sample(cell); ok {
			cells++
			cellName = bank.entries[cell]
		}
		if spell == 3 && (castName != "-" || cellName != "-") {
			t.Fatalf("Fire Wall cast %s cell %s; the install was read to hold neither", castName, cellName)
		}
		named = append(named, fmt.Sprintf("%d:%s/%s", spell, castName, cellName))
	}
	t.Logf("%d of 28 spells have a cast sound, %d a cell sound: %s", casts, cells, strings.Join(named, " "))
}
