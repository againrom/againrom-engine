package game

import (
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sync"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

type decodedInstallSlot struct {
	once      sync.Once
	resources InstallResources
	err       error
}

var decodedInstalls struct {
	sync.Mutex
	slots map[string]*decodedInstallSlot
}

// Save witnesses share read-only decoded payloads. Descriptors, banks,
// devices and session state remain private.
func decodedInstallFront(root string) (*FrontEnd, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	key, ok := installShareKey(abs)
	if !ok {
		return NewFrontEnd(root)
	}
	decodedInstalls.Lock()
	if decodedInstalls.slots == nil {
		decodedInstalls.slots = map[string]*decodedInstallSlot{}
	}
	slot := decodedInstalls.slots[key]
	if slot == nil {
		slot = &decodedInstallSlot{}
		decodedInstalls.slots[key] = slot
	}
	decodedInstalls.Unlock()
	slot.once.Do(func() {
		var f *FrontEnd
		f, slot.err = NewFrontEnd(abs)
		if slot.err != nil {
			return
		}
		slot.resources = f.InstallResources
		slot.resources.SoundBank, slot.resources.MusicBank, slot.resources.SpeechBank = nil, nil, nil
		for _, device := range []interface{ Stop() }{f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer} {
			if device != nil {
				device.Stop()
			}
		}
		if owner := ui.DeliveryOwner(f.SoundPlayer); owner != nil && owner.Service != nil {
			owner.Service.Close()
		}
	})
	if slot.err != nil {
		return nil, slot.err
	}
	in := slot.resources
	in.Units = cloneCandidateUnits(in.Units)
	in.Statics = cloneCandidateStatics(in.Statics)
	in.Structures = cloneCandidateStructures(in.Structures)
	in.Maps = slices.Clone(in.Maps)
	in.NPCFaces = maps.Clone(in.NPCFaces)
	if in.Table != nil {
		table := *in.Table
		in.Table = &table
	}
	in.SoundBank, in.MusicBank, in.SpeechBank = OpenSounds(abs), OpenMusic(abs), OpenSpeech(abs)
	settings := audio.Settings{Master: soundOptions.Volume, Muted: !soundOptions.Enabled}
	shared, _ := ui.OpenSharedAudio(soundChannelVolumes.Settings(audio.EffectsChannel, settings),
		soundChannelVolumes.Settings(audio.SpeechChannel, settings))
	scope := shared.NewScope()
	music, _ := ui.OpenMusic(soundChannelVolumes.Settings(audio.MusicChannel, settings))
	cutscene, _ := ui.OpenCutsceneAudio(settings)
	return &FrontEnd{
		InstallResources: in,
		RuntimeServices: RuntimeServices{
			Sound: soundOptions, SoundChannels: soundChannelVolumes,
			SoundPlayer: scope.Player(audio.EffectsChannel), SpeechPlayer: scope.Player(audio.SpeechChannel),
			MusicPlayer: music, AmbientPlayer: scope.Ambient(), CutsceneAudioPlayer: cutscene,
			Random: random.NewService(random.Session{}),
		},
		CampaignSession: CampaignSession{fame: SnapshotFame{Known: true}, Town: NewTown(in.Campaign.Value())},
		Presentation:    Presentation{Markers: Markers{Objects: true, Units: true, Statics: true}},
	}, nil
}

func viewTable(t *mapload.Table) *mapload.Table {
	if t == nil {
		return nil
	}
	v := *t
	v.SpellArms, v.FreshPlayers = nil, nil
	if t.Edition != nil {
		e := *t.Edition
		e.TextCodePage, e.MissionTip = nil, nil
		v.Edition = &e
	}
	return &v
}

func tableFuncCodes(t *mapload.Table) []uintptr {
	code := func(fn any) uintptr {
		if v := reflect.ValueOf(fn); v.IsNil() {
			return 0
		} else {
			return v.Pointer()
		}
	}
	out := []uintptr{code(t.SpellArms), code(t.FreshPlayers)}
	if t.Edition != nil {
		out = append(out, code(t.Edition.TextCodePage), code(t.Edition.MissionTip))
	}
	return out
}

func TestReleaseDecodedInstallFixtureIsolation(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: decoded install fixture needs a lawful install")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	front := func(build func(string) (*FrontEnd, error)) *FrontEnd {
		f, err := build(abs)
		if err != nil {
			t.Fatal(err)
		}
		cleanupFrontAudio(t, f)
		return f
	}
	fresh, a, b := front(NewFrontEnd), front(decodedInstallFront), front(decodedInstallFront)
	resources := func(f *FrontEnd) InstallResources {
		in := f.InstallResources
		in.SoundBank, in.MusicBank, in.SpeechBank = nil, nil, nil
		in.Table = viewTable(in.Table)
		return in
	}
	codes := func(f *FrontEnd) []uintptr { return tableFuncCodes(f.Table) }
	runtime := func(f *FrontEnd) RuntimeServices {
		r := f.RuntimeServices
		r.SoundPlayer, r.SpeechPlayer, r.MusicPlayer, r.AmbientPlayer, r.CutsceneAudioPlayer = nil, nil, nil, nil, nil
		r.Random = nil
		return r
	}
	defaults := func(f *FrontEnd) {
		if !reflect.DeepEqual(resources(f), resources(fresh)) || !reflect.DeepEqual(runtime(f), runtime(fresh)) ||
			!reflect.DeepEqual(f.CampaignSession, fresh.CampaignSession) || !reflect.DeepEqual(f.Presentation, fresh.Presentation) ||
			!reflect.DeepEqual(f.PersistenceContext, fresh.PersistenceContext) ||
			!reflect.DeepEqual(codes(f), codes(fresh)) {
			t.Fatal("decoded install fixture differs from a fresh front end")
		}
	}
	defaults(a)
	defaults(b)
	if a.Table == b.Table || a.Town == b.Town || a.Units == b.Units || a.Statics == b.Statics || a.Structures == b.Structures {
		t.Fatal("decoded install fixture shared mutable descriptors or town state")
	}
	for _, class := range a.Statics.Classes {
		if class != nil {
			class.Width++
			if len(class.Timeline) != 0 {
				class.Timeline[0]++
			}
			if class.Dead != nil {
				class.Dead.Width++
			}
			break
		}
	}
	for _, class := range a.Structures.Classes {
		if class != nil {
			class.Name += " edited"
			if len(class.Rank) != 0 {
				class.Rank[0]++
			}
			break
		}
	}
	a.Units.Bodies["fixture-only"] = &terrain.UnitClass{}
	a.Table.Humans = nil
	a.Town.gold++
	a.Town.won[10] = true
	a.fame.Known = false
	a.live = &mapWorld{}
	a.Markers.Objects = false
	a.tipsOff = true
	a.SetDeterministicFrames(true)
	if a.SoundBank != nil {
		if a.SoundBank == b.SoundBank {
			t.Fatal("decoded install fixture shared sound caches")
		}
		a.SoundBank.cache[-1] = soundCacheEntry{}
		if _, ok := b.SoundBank.cache[-1]; ok {
			t.Fatal("editing a sound cache changed another front end")
		}
	}
	if a.SpeechBank == b.SpeechBank || a.MusicBank == b.MusicBank || ui.DeliveryOwner(a.SoundPlayer) == ui.DeliveryOwner(b.SoundPlayer) {
		t.Fatal("decoded install fixture shared media or delivery state")
	}
	a.SpeechBank.last = "fixture-only"
	defaults(b)
	c := front(decodedInstallFront)
	defaults(c)
	a.resetSessionForNewGame()
	fresh.resetSessionForNewGame()
	if !reflect.DeepEqual(a.CampaignSession, fresh.CampaignSession) {
		t.Fatal("decoded install fixture reset differs from a fresh front end")
	}
	previousSound, previousChannels := soundOptions, soundChannelVolumes
	defer func() { soundOptions, soundChannelVolumes = previousSound, previousChannels }()
	soundOptions.Enabled = !soundOptions.Enabled
	soundOptions.Volume /= 2
	soundChannelVolumes[audio.MusicChannel] /= 2
	d := front(decodedInstallFront)
	if d.Sound != soundOptions || d.SoundChannels != soundChannelVolumes {
		t.Fatal("decoded install fixture retained earlier audio settings")
	}
}
