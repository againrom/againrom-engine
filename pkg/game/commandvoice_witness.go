package game

import (
	"fmt"
	"io"
	"strings"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/random"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// replyClockFrames is one response clock of 20 ms headless frames.
const replyClockFrames = int(commandVoiceDelay / (20 * time.Millisecond))

// headlessReplyClock runs one response clock of ordinary frames and answers
// each mission notice that opens with Enter, as a player would, so the map
// takes the next click.
func headlessReplyClock(a *ui.App) error {
	for n := 0; n < replyClockFrames || a.HeadlessNoticeOpen() && n < 4*replyClockFrames; n++ {
		var err error
		if a.HeadlessNoticeOpen() {
			err = a.HeadlessKey("enter")
		} else {
			err = a.HeadlessStep()
		}
		if err != nil {
			return err
		}
	}
	if a.HeadlessNoticeOpen() {
		return fmt.Errorf("reply clock: a notice stayed open")
	}
	return nil
}

type commandVoiceObserver struct {
	audio.Player
	plays    int
	legacy   int
	requests []audio.Request
}

func (p *commandVoiceObserver) Play(sample audio.Sample, placement audio.Placement) {
	p.legacy++
	p.Player.Play(sample, placement)
}

func (p *commandVoiceObserver) RequestSample(sample audio.Sample, request audio.Request) audio.Voice {
	p.requests = append(p.requests, request)
	voice := audio.Dispatch(p.Player, sample, request)
	if voice != nil {
		p.plays++
	}
	return voice
}

func newAudioWitnessFront(root string, options OptionsStore, sound SoundOptions, channels audio.ChannelVolumes, deterministic bool) (*FrontEnd, error) {
	g, err := NewFrontEnd(root)
	if err != nil {
		return nil, err
	}
	g.Options = options
	g.Sound, g.SoundChannels = sound, channels
	g.SetDeterministicFrames(deterministic)
	g.applySoundSettings()
	return g, nil
}

func audioWitnessClosed(owner *ui.SharedAudio) error {
	if owner == nil {
		return fmt.Errorf("audio witness: shared owner absent")
	}
	snapshot := owner.Service.Snapshot()
	backend := owner.BackendState()
	if !snapshot.Closed || len(snapshot.Samples) != 0 || snapshot.Counters.BuffersCreated != snapshot.Counters.BuffersDestroyed || len(backend.Buffers) != 0 || backend.Created != backend.Destroyed {
		return fmt.Errorf("audio witness: shutdown retained delivery state: service=%+v backend=%+v", snapshot, backend)
	}
	for _, channel := range snapshot.Channels {
		if channel.Playing || channel.Voice.Sample != 0 {
			return fmt.Errorf("audio witness: shutdown retained channel: %+v", channel)
		}
	}
	return nil
}

func (f *FrontEnd) witnessCommandAcknowledgments(report io.Writer) error {
	return f.witnessCommandAcknowledgmentsWithClockControl(report, nil)
}

func (f *FrontEnd) witnessCommandAcknowledgmentsWithClockControl(report io.Writer, beforeReply func(*FrontEnd, bool)) error {
	for _, banks := range humanVoiceBanks {
		for _, bank := range banks {
			for n := 1; n <= 3; n++ {
				name := fmt.Sprintf("%s/command%d.wav", bank, n)
				sample, ok := f.SoundBank.namedSample(name)
				nonzero := false
				for _, value := range sample.PCM {
					if value != 0 {
						nonzero = true
						break
					}
				}
				if !ok || !nonzero {
					return fmt.Errorf("command voice witness: %s absent or silent", name)
				}
			}
		}
	}
	// A fixed draw keeps the witness off the one-in-32768 draw that names no
	// speaker.
	previous := newViewerVoiceDraw
	newViewerVoiceDraw = func(*random.Stream) voiceDraw { return func() int { return 0 } }
	defer func() { newViewerVoiceDraw = previous }()
	var hashes [2]uint64
	var ticks [2]uint64
	for run, enabled := range []bool{true, false} {
		probe, err := newAudioWitnessFront(f.Archives.Root, f.Options, f.Sound, f.SoundChannels, true)
		if err != nil {
			return err
		}
		owner := ui.DeliveryOwner(probe.SoundPlayer)
		player := probe.SpeechPlayer
		if owner == nil || owner != ui.DeliveryOwner(player) {
			return fmt.Errorf("command voice witness: effects and speech have no shared owner")
		}
		observed := &commandVoiceObserver{Player: player}
		probe.SpeechPlayer = observed
		a := probe.App("command voice witness")
		defer a.StopAudio()
		a.SetCutscenes(nil)
		if err := a.OpenMission(probe.MissionOpener(20)); err != nil {
			return err
		}
		a.Layout(640, 480)
		for n := 0; a.HeadlessNoticeOpen() && n < 16; n++ {
			if err := a.HeadlessKey("enter"); err != nil {
				return err
			}
		}
		if a.HeadlessNoticeOpen() {
			return fmt.Errorf("command voice witness: notice did not close")
		}
		if err := a.HeadlessKey("escape"); err != nil {
			return err
		}
		if err := a.HeadlessGameMenuAction("sound-options"); err != nil {
			return err
		}
		if probe.acknowledgmentsOff == enabled {
			if err := a.HeadlessGameMenuAction("acknowledgments"); err != nil {
				return err
			}
		}
		if err := a.HeadlessGameMenuAction("page-return"); err != nil {
			return err
		}
		if err := a.HeadlessGameMenuAction("return"); err != nil {
			return err
		}
		selected := false
		for _, e := range probe.live.world.Entities() {
			if e.Owner == sim.SelfSlot && e.HP > 0 && probe.live.voiceBank(e) != "" {
				if err := a.HeadlessSelectEntity(uint32(e.ID)); err == nil {
					selected = true
					break
				}
			}
		}
		if !selected {
			return fmt.Errorf("command voice witness: no visible human to command")
		}
		selection := observed.plays
		if selection != boolOption(enabled) {
			return fmt.Errorf("command voice witness: enabled=%v selection requests=%d", enabled, selection)
		}
		// The selection reply started the speaker's response clock; the order
		// follows once it has run out.
		if beforeReply != nil {
			beforeReply(probe, enabled)
		}
		if err := headlessReplyClock(a); err != nil {
			return err
		}
		x, y, err := a.HeadlessGroundPoint()
		if err != nil {
			return err
		}
		for _, edge := range []string{"press", "release"} {
			if err := a.HeadlessPointer(edge, x, y); err != nil {
				return err
			}
		}
		if observed.plays-selection != boolOption(enabled) {
			return fmt.Errorf("command voice witness: enabled=%v command requests=%d", enabled, observed.plays-selection)
		}
		if observed.legacy != 0 || len(observed.requests) != 2*boolOption(enabled) {
			return fmt.Errorf("command voice witness: typed requests=%d legacy plays=%d", len(observed.requests), observed.legacy)
		}
		for i, request := range observed.requests {
			want := []string{"unit-selection", "unit-command"}[i]
			if request.Source != want || request.Group != audio.SpeechChannel || !strings.HasPrefix(request.Selector, "voice:") {
				return fmt.Errorf("command voice witness: wrong semantic request %+v, want %s on speech", request, want)
			}
		}
		if _, gains, concrete := ui.AudioDeviceState(player); enabled {
			if !concrete || len(gains) == 0 {
				return fmt.Errorf("command voice witness: concrete player did not start")
			}
			fmt.Fprintf(report, "command voice: concrete speech players=%d gains=%v\n", len(gains), gains)
		}
		fmt.Fprintf(report, "command voice: enabled=%v selection-requests=%d command-requests=%d fresh-viewer=yes pointer-command=yes\n", enabled, selection, observed.plays-selection)
		hashes[run], ticks[run] = probe.live.world.Hash(), probe.live.world.Tick()
		f.acknowledgmentsOff = probe.acknowledgmentsOff
		a.StopAudio()
		if err := audioWitnessClosed(owner); err != nil {
			return err
		}
	}
	if hashes[0] != hashes[1] || ticks[0] != ticks[1] {
		return fmt.Errorf("command voice witness: Acknowledgement changed the World: tick %d hash %#x against tick %d hash %#x", ticks[0], hashes[0], ticks[1], hashes[1])
	}
	fmt.Fprintf(report, "command voice: World digest %#x at tick %d with Acknowledgement on and off\n", hashes[0], ticks[0])
	fmt.Fprintln(report, "command voice: 18 installed human recordings contain nonzero PCM; disabled preference committed")
	return nil
}
