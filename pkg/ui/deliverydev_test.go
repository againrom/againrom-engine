package ui

import (
	"io"
	"testing"
	"time"

	"againrom/pkg/audio"
)

type controlledDeliveryPlayer struct {
	reader   io.ReadSeeker
	playing  bool
	position time.Duration
	gain     float64
	closed   int
	seeks    int
}

func (p *controlledDeliveryPlayer) Play()           { p.playing = true }
func (p *controlledDeliveryPlayer) IsPlaying() bool { return p.playing }
func (p *controlledDeliveryPlayer) Pause()          { p.playing = false }
func (p *controlledDeliveryPlayer) Seek(at time.Duration) error {
	p.position = at
	p.seeks++
	_, err := p.reader.Seek(int64(at)*audio.DeviceRate*4/int64(time.Second), io.SeekStart)
	return err
}
func (p *controlledDeliveryPlayer) Position() time.Duration { return p.position }
func (p *controlledDeliveryPlayer) SetVolume(gain float64)  { p.gain = gain }
func (p *controlledDeliveryPlayer) Volume() float64         { return p.gain }
func (p *controlledDeliveryPlayer) Close() error            { p.closed++; p.playing = false; return nil }

func TestSharedAudioRetainsBufferAndLoopPhase(t *testing.T) {
	var players []*controlledDeliveryPlayer
	restore := SetDeliveryPlayerFactory(func(reader io.ReadSeeker) (DeliveryDevicePlayer, error) {
		p := &controlledDeliveryPlayer{reader: reader}
		players = append(players, p)
		return p, nil
	})
	defer restore()
	owner, err := OpenSharedAudio(audio.DefaultSettings, audio.DefaultSettings)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Service.Close()
	scope := owner.NewScope()
	sample := audio.Sample{Rate: audio.DeviceRate, PCM: make([]int16, audio.DeviceRate)}
	request := audio.FixedRequest("water", "water", audio.EffectsChannel, 128, true,
		audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit})
	voice := audio.Dispatch(scope.Player(audio.EffectsChannel), sample, request).(*deliveryVoice)
	players[0].position = 250 * time.Millisecond
	phase := voice.Phase()
	voice.Move(audio.Placement{Left: audio.GainUnit / 2, Right: audio.GainUnit})
	owner.SetSettings(audio.EffectsChannel, audio.Settings{Master: audio.MasterUnit / 2})
	if voice.Phase() != phase || players[0].seeks != 0 || len(players) != 1 {
		t.Fatal("Move/settings replaced or rewound the live loop")
	}
	voice.Stop()
	if voice.Phase() != phase || players[0].closed != 0 {
		t.Fatal("stop destroyed or rewound buffer")
	}
	voice.StopReset()
	if voice.Phase() != 0 || players[0].closed != 0 {
		t.Fatal("stop/reset did not retain rewound buffer")
	}
	voice = audio.Dispatch(scope.Player(audio.EffectsChannel), sample, request).(*deliveryVoice)
	if len(players) != 1 || !voice.Playing() {
		t.Fatal("request did not reuse retained duplicate")
	}
	scope.Destroy()
	scope.Destroy()
	if players[0].closed != 1 {
		t.Fatalf("scope destruction closed buffer %d times", players[0].closed)
	}
	if state := owner.BackendState(); len(state.Buffers) != 0 || state.Created != state.Destroyed {
		t.Fatalf("backend buffers leaked: %+v", state)
	}
}

func TestSharedAudioOneFactoryForEveryGroupAndLoop(t *testing.T) {
	var players []*controlledDeliveryPlayer
	restore := SetDeliveryPlayerFactory(func(reader io.ReadSeeker) (DeliveryDevicePlayer, error) {
		p := &controlledDeliveryPlayer{reader: reader}
		players = append(players, p)
		return p, nil
	})
	defer restore()
	owner, err := OpenSharedAudio(audio.DefaultSettings, audio.DefaultSettings)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Service.Close()
	scope := owner.NewScope()
	sample := audio.Sample{Rate: audio.DeviceRate, PCM: []int16{1, 2, 3}}
	centre := audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}
	for _, group := range []audio.Channel{audio.EffectsChannel, audio.SpeechChannel} {
		if audio.Dispatch(scope.Player(group), sample, audio.FixedRequest("fixed", "same-object", group, 128, false, centre)) == nil {
			t.Fatal("group refused")
		}
	}
	loop := RequestAmbient(scope.Ambient(), AmbientRiver, sample,
		audio.FixedRequest("river", "river-object", audio.EffectsChannel, 220, true, centre))
	if loop == nil || len(players) != 3 {
		t.Fatal("loop bypassed common factory")
	}
	snapshot := owner.Service.Snapshot()
	if len(snapshot.Samples) != 2 || snapshot.Samples[0].Duplicates[0].Request.Group != audio.EffectsChannel ||
		snapshot.Samples[0].Duplicates[1].Request.Group != audio.SpeechChannel {
		t.Fatal("same sample object did not share duplicate ownership across groups")
	}
	buf := make([]byte, 32)
	if n, err := players[2].reader.Read(buf); n != len(buf) || err != nil {
		t.Fatal("repeat reader reached EOF")
	}
	if _, err := players[0].reader.Read(buf); err != nil {
		t.Fatal(err)
	}
	if _, err := players[0].reader.Read(buf); err != io.EOF {
		t.Fatal("one-shot reader repeated")
	}
	if _, gains, ok := SoundDeviceState(scope.Player(audio.SpeechChannel)); !ok || len(gains) != 1 {
		t.Fatal("speech witness could not observe actual common-factory player")
	}
}

func TestSharedAudioRegistryObjectCannotBypassDuplicatesAcrossScopes(t *testing.T) {
	var players []*controlledDeliveryPlayer
	restore := SetDeliveryPlayerFactory(func(reader io.ReadSeeker) (DeliveryDevicePlayer, error) {
		p := &controlledDeliveryPlayer{reader: reader}
		players = append(players, p)
		return p, nil
	})
	defer restore()
	owner, err := OpenSharedAudio(audio.DefaultSettings, audio.DefaultSettings)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Service.Close()
	sample := audio.Sample{Rate: audio.DeviceRate, PCM: []int16{1, 2}}
	request := audio.FixedRequest("fixed-interface", "registry:100", audio.EffectsChannel, 220, false,
		audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit})
	var scopes []*AudioScope
	for range 16 {
		scope := owner.NewScope()
		scopes = append(scopes, scope)
		if audio.Dispatch(scope.Player(audio.EffectsChannel), sample, request) == nil {
			t.Fatal("request refused before duplicate cap")
		}
	}
	if audio.Dispatch(owner.NewScope().Player(audio.EffectsChannel), sample, request) != nil {
		t.Fatal("new caller scope bypassed sample duplicate cap")
	}
	snapshot := owner.Service.Snapshot()
	if len(snapshot.Samples) != 1 || snapshot.Receipts[len(snapshot.Receipts)-1].Reason != audio.DeliverySampleBusy {
		t.Fatal("registry sample identity split across scopes")
	}
	scopes[0].Destroy()
	if players[0].closed != 1 || players[1].closed != 0 || len(owner.Service.Snapshot().Samples) != 1 {
		t.Fatal("caller scope destroyed unrelated duplicate or persistent sample")
	}
	if audio.Dispatch(scopes[1].Player(audio.EffectsChannel), sample, request) == nil {
		t.Fatal("remaining caller could not reuse freed duplicate")
	}
	if len(players) != 17 {
		t.Fatal("destroyed duplicate buffer was reused")
	}
}
