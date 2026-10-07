package game

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// WitnessTownMedia checks installed speech through the production dialogue
// controller and retained device. Only rendered evidence is written; no save
// or install file is changed. Output was validated by WitnessCutscene.
func (f *FrontEnd) WitnessTownMedia(output string, report io.Writer) error {
	if output != "" && !filepath.IsAbs(output) {
		return fmt.Errorf("speech witness: absolute output required")
	}
	t := f.TownScreen().(*townScreen)
	defer t.atSquare()
	deferredBank := f.SpeechBank
	if deferredBank == nil {
		return fmt.Errorf("speech witness: no speech bank")
	}
	for _, topic := range []struct {
		name                    string
		building                TownBuilding
		npc, mission, mercenary int
	}{
		{"tavern", TownTavern, 25, 100, 0},
		{"mercenary", TownTavern, 0, 0, 3},
		{"merchant", TownShop, 0, 31, 0},
		{"teacher", TownSchool, 0, 61, 0},
	} {
		t.atSquare()
		if topic.building == TownShop {
			t.Choose(1)
		}
		if topic.building == TownSchool {
			t.Choose(2)
		}
		if topic.mercenary != 0 {
			t.openMercenaryDialogue(topic.mercenary)
		} else if !t.openTownDialogue(topic.building, TownOffer{Mission: topic.mission}, topic.npc) {
			return fmt.Errorf("speech witness: %s text absent", topic.name)
		}
		name, ok := t.townSpeechPath()
		if !ok {
			return fmt.Errorf("speech witness: %s has no voice address", topic.name)
		}
		sample, ok := deferredBank.Sample(name)
		if !ok {
			return fmt.Errorf("speech witness: %s missing or invalid", name)
		}
		nonzero := false
		for _, value := range sample.PCM {
			if value != 0 {
				nonzero = true
				break
			}
		}
		if !nonzero {
			return fmt.Errorf("speech witness: %s has only zero samples", name)
		}
		t.TownDialogueActive(true)
		voice := t.speech.voice
		if voice == nil || !voice.Playing() {
			return fmt.Errorf("speech witness: %s device did not start (enabled=%v volume=%d)", topic.name, f.Sound.Enabled, f.Sound.Volume)
		}
		pix, err := ui.ComposeTownScreen(t, "")
		if err != nil {
			return err
		}
		if err := writeMediaFrame(output, topic.name, pix); err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
		t.atSquare()
		if voice.Playing() {
			return fmt.Errorf("speech witness: %s voice survived dialogue exit", topic.name)
		}
		fmt.Fprintf(report, "speech: topic=%s file=%s rate=%d samples=%d nonzero=yes device=playing stop=closed\n", topic.name, name, sample.Rate, len(sample.PCM))
	}
	if err := f.witnessTownResponses(report); err != nil {
		return err
	}
	// Decode every recording as a separate inventory check.
	entries := deferredBank.src.Entries()
	count := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Address, ".wav") {
			continue
		}
		if _, ok := deferredBank.Sample(entry.Address); !ok {
			return fmt.Errorf("speech witness: cannot decode %s", entry.Address)
		}
		count++
	}
	fmt.Fprintf(report, "speech: installed-wave-decode=%d/%d\n", count, count)
	return nil
}

// Exercise the ordinary click handlers with controlled in-memory choices and
// real installed recordings. These are playback witnesses, not ROM1 triggers.
func (f *FrontEnd) witnessTownResponses(report io.Writer) error {
	t := f.TownScreen().(*townScreen)
	party, shop, town := f.Carried, f.Shop, f.Town
	defer func() {
		t.atSquare()
		f.Carried, f.Shop, f.Town = party, shop, town
	}()
	f.Town = NewTown(Campaign{})
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Speech witness", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 30, 30}})
	f.arriveInTown()
	check := func(action, expected string) error {
		voice := t.speech.voice
		if f.SpeechBank.last != expected || voice == nil || !voice.Playing() {
			return fmt.Errorf("speech response: %s did not start %s", action, expected)
		}
		sample, ok := f.SpeechBank.Sample(expected)
		nonzero := false
		for _, value := range sample.PCM {
			nonzero = nonzero || value != 0
		}
		if !ok || !nonzero {
			return fmt.Errorf("speech response: %s has no decoded nonzero PCM", expected)
		}
		t.TownDialogueActive(true)
		time.Sleep(20 * time.Millisecond)
		if t.speech.voice != voice || !voice.Playing() {
			return fmt.Errorf("speech response: %s stopped on ordinary update", action)
		}
		if err := f.witnessSpeechFocus(t, voice, report); err != nil {
			return fmt.Errorf("speech response: %s: %w", action, err)
		}
		fmt.Fprintf(report, "response: action=%s file=%s samples=%d nonzero=yes device=playing focus=retained/resumed current-SAV=unchanged room-exit=destroyed-once\n", action, expected, len(sample.PCM))
		if action == "train-skill" {
			t.Choose(2)
		}
		return nil
	}
	// A paid training step requests the teacher speech of the trained level's
	// tier (TOWN-502): the level before the step is 5, 30 or 60.
	f.Town.gold = 1 << 30
	for class := 0; class < 2; class++ {
		f.Carried[0].Mage = class == 1
		t.atSquare()
		t.Choose(2)
		for tier, before := range []int32{5, 30, 60} {
			for slot := 1; slot <= 5; slot++ {
				f.Carried[0].Hero.Skill[slot] = before
				t.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: class*5 + slot - 1}, false)
				t.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 0}, false)
				name := fmt.Sprintf("speech/training/npc%ds%dl%d.wav", 33+class, slot, tier+1)
				if err := check("train-skill", name); err != nil {
					return err
				}
			}
		}
	}
	for _, item := range []struct {
		value ShopItem
		name  string
	}{
		{ShopItem{Code: 0x0102}, "speech/shop/s01i02p1.wav"},
		{ShopItem{Code: 0x0606}, "speech/shop/s06i06p1.wav"},
		{ShopItem{Code: 0x0e15, Kind: 5, Effects: []sim.ItemEffect{{Kind: 42, Operand: 3}}}, "speech/shop/books/03.wav"},
		{ShopItem{Code: 0x0e01, Effects: []sim.ItemEffect{{Kind: 2}}}, "speech/shop/effects/02.wav"},
	} {
		t.atSquare()
		f.Shop = NewShop(1000)
		value := item.value
		value.Price, value.Count = 100, 1
		f.Shop.shelves[ShelfArmour] = []ShopItem{value}
		t.Choose(1)
		t.ShopClick(ui.ShopControl{Kind: ui.ShopControlShelfCell, Index: 0})
		if len(f.Shop.Table()) != 1 {
			return fmt.Errorf("speech response: item selection did not reach trade table")
		}
		if err := check("select-item", item.name); err != nil {
			return err
		}
	}
	return nil
}

func (f *FrontEnd) witnessSpeechFocus(t *townScreen, voice audio.Voice, report io.Writer) error {
	owner := ui.DeliveryOwner(f.SoundPlayer)
	current, ok := voice.(interface {
		ID() audio.VoiceID
		Phase() int64
	})
	if owner == nil || !ok {
		return fmt.Errorf("shared speech owner or concrete voice absent")
	}
	id := current.ID()
	before := owner.BackendState()
	var buffer, sequence uint64
	physical := false
	for _, b := range before.Buffers {
		if b.Group == audio.SpeechChannel && b.Playing {
			if buffer != 0 {
				return fmt.Errorf("ambiguous active speech buffers")
			}
			buffer = b.ID
			physical = b.Physical
		}
	}
	for _, r := range before.Receipts {
		sequence = max(sequence, r.Sequence)
	}
	if buffer == 0 {
		return fmt.Errorf("concrete speech buffer absent")
	}
	deadline := time.Now().Add(750 * time.Millisecond)
	for current.Phase() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if current.Phase() == 0 {
		return fmt.Errorf("speech buffer did not establish a nonzero phase before focus loss")
	}
	save := func() ([]byte, error) {
		snapshot, _, err := f.Snapshot(false)
		if err != nil {
			return nil, err
		}
		return f.ExportCurrentSave(snapshot, "Speech focus witness")
	}
	raw, err := save()
	if err != nil {
		return err
	}
	phaseBefore := current.Phase()
	if phaseBefore <= 0 {
		return fmt.Errorf("speech phase returned to zero before focus loss")
	}
	t.TownDialogueActive(false)
	phase := current.Phase()
	immediate := phase
	if t.speech.voice != voice || !t.speech.paused || voice.Playing() || current.ID() != id {
		return fmt.Errorf("focus loss replaced or failed to pause the speech voice")
	}
	if physical {
		deadline := time.Now().Add(750 * time.Millisecond)
		stable := 0
		for stable < 3 && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
			next := current.Phase()
			if next == phase {
				stable++
			} else {
				phase, stable = next, 0
			}
		}
		if stable < 3 || phase <= 0 {
			return fmt.Errorf("native paused phase cache did not settle above zero")
		}
	} else if phase != phaseBefore {
		return fmt.Errorf("controlled pause changed speech phase: %d -> %d", phaseBefore, phase)
	}
	retained := false
	for _, sample := range owner.Service.Snapshot().Samples {
		if sample.ID == id.Sample && id.Duplicate < len(sample.Duplicates) {
			d := sample.Duplicates[id.Duplicate]
			retained = d.Retained && d.Generation == id.Generation && d.Paused && !d.Playing && d.Phase == phase
		}
	}
	if !retained {
		return fmt.Errorf("focus loss discarded the reserved sample generation or phase")
	}
	paused := false
	for _, b := range owner.BackendState().Buffers {
		if b.ID == buffer {
			paused = !b.Playing && b.Phase == phase
		}
	}
	if !paused {
		return fmt.Errorf("focus loss discarded the lowest buffer or its phase")
	}
	time.Sleep(20 * time.Millisecond)
	if current.Phase() != phase {
		return fmt.Errorf("paused speech advanced its phase")
	}
	t.TownDialogueActive(true)
	resumed := current.Phase()
	if t.speech.voice != voice || t.speech.paused || !voice.Playing() || current.ID() != id || resumed <= 0 || !physical && resumed != phase {
		return fmt.Errorf("focus return replaced, rewound or failed to resume the speech voice")
	}
	after, err := save()
	if err != nil || !bytes.Equal(raw, after) {
		return fmt.Errorf("focus changed current SAV: %v", err)
	}
	t.atSquare()
	t.atSquare()
	if voice.Playing() || current.Phase() != 0 || t.speech.voice != nil {
		return fmt.Errorf("room exit retained the speech voice")
	}
	for _, sample := range owner.Service.Snapshot().Samples {
		if sample.ID == id.Sample {
			return fmt.Errorf("room exit retained the speech sample")
		}
	}
	final := owner.BackendState()
	for _, b := range final.Buffers {
		if b.ID == buffer {
			return fmt.Errorf("room exit retained the lowest speech buffer")
		}
	}
	var actions []string
	for _, r := range final.Receipts {
		if r.Sequence > sequence && r.Buffer == buffer {
			actions = append(actions, r.Action)
		}
	}
	if strings.Join(actions, ",") != "stop,play,stop,destroy" {
		return fmt.Errorf("focus/room-exit buffer operations %v, want pause/resume/terminal-stop/destruction once", actions)
	}
	backend := "controlled-exact"
	if physical {
		backend = "native-cache-cuts"
	}
	fmt.Fprintf(report, "focus: backend=%s before=%d immediate=%d paused=%d resumed=%d buffer=%d generation=%d owner=retained SAV=unchanged operations=stop,play,stop,destroy\n", backend, phaseBefore, immediate, phase, resumed, buffer, id.Generation)
	return nil
}
