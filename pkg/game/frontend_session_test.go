package game

import (
	"errors"
	"image"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"againrom/pkg/audio"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/menu"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// This file is 1019's own guard, on TestResetForNewGameDropsExactlyTheGamePopulation's
// precedent (townscreen_test.go): the owner reported that leaving a running
// game for the main menu does not end the session, so a mission picked
// afterwards opens against the previous game's own town — its gold, chapter,
// finished/available/taken state and, on death, the previous game's own
// carried party. resetSessionForNewGame (frontend.go) is the fix.
//
// TestResetSessionForNewGameDropsExactlyTheSessionPopulation enumerates
// FrontEnd's whole field set with reflect, so a field added to the struct
// later fails this test until it is classified as session state (reset) or
// install state (kept, with a reason) — the same shape townscreen_test.go
// already enforces for townScreen, applied here to the struct that actually
// held the reported bug.

// fakeFaceSource, fakeCollection and fakePlayer give the three interface
// fields (Faces, Humans, SoundPlayer) a non-nil, non-zero value for the
// reflect walk below. None of their methods is exercised by this test.
type fakeFaceSource struct{}

func (fakeFaceSource) SpeakerFace(int) (*image.RGBA, image.Rectangle, bool) {
	return nil, image.Rectangle{}, false
}

type fakeCollection struct{}

func (fakeCollection) Len() int                  { return 0 }
func (fakeCollection) EntryName(int) string      { return "" }
func (fakeCollection) EntryParams(int) []int32   { return nil }
func (fakeCollection) EntryStrings(int) []string { return nil }

type fakePlayer struct{}

func (fakePlayer) Play(audio.Sample, audio.Placement)                    {}
func (fakePlayer) RequestSample(audio.Sample, audio.Request) audio.Voice { return nil }
func (fakePlayer) RequestLoop(ui.AmbientLoop, audio.Sample, audio.Request) audio.Voice {
	return nil
}
func (fakePlayer) Start(audio.Track)                                       {}
func (fakePlayer) Stop()                                                   {}
func (fakePlayer) Ended() bool                                             { return false }
func (fakePlayer) SetSettings(audio.Settings)                              {}
func (fakePlayer) StartLoop(ui.AmbientLoop, audio.Sample, audio.Placement) {}
func (fakePlayer) MoveLoop(ui.AmbientLoop, audio.Placement)                {}
func (fakePlayer) StopLoop(ui.AmbientLoop)                                 {}

// fakeCutsceneAudioPlayer gives the CutsceneAudioPlayer field a non-nil,
// non-zero value for the reflect walk below. None of its methods is
// exercised by this test.
type fakeCutsceneAudioPlayer struct{}

func (fakeCutsceneAudioPlayer) Start(int, int)             {}
func (fakeCutsceneAudioPlayer) Push([]byte)                {}
func (fakeCutsceneAudioPlayer) Stop()                      {}
func (fakeCutsceneAudioPlayer) SetSettings(audio.Settings) {}

// nonZeroFrontEnd returns a FrontEnd with every field set to a value that
// differs from that field's own Go zero value, built by direct field
// assignment (this file is package game, so every field is reachable by
// name). Every pointer field is a bare &T{}: this test only requires each
// one to come back at its OWN expected value (nil for a reset field,
// unchanged for a kept one), never to hold any particular payload.
func nonZeroFrontEnd() *FrontEnd {
	return &FrontEnd{InstallResources: InstallResources{endingAssets: &ui.EndingView{Hall: []ui.EndingHallRow{{Name: "installed record"}}}, Cutscenes: &CutsceneBank{}, Archives: &Archives{}, Assets: &menu.Assets{}, Tiles: &terrain.Tileset{}, Maps: []MapEntry{{Source: "a"}}, Faces: fakeFaceSource{}, NPCFaces: map[int32]data.NPCFace{1: {}}, Statics: &terrain.StaticSet{}, Units: &terrain.UnitSet{}, Structures: &terrain.StructureSet{}, Projectiles: &terrain.EffectSet{}, Font: resolved(&text.Font{}, errors.New("a previous game's own font error")), Words: ui.Words{MenuSave: "a previous game's own word"}, ChargenAssets: &ChargenAssets{}, TownSchoolArt: resolved(&ui.TownSchoolArt{}, errors.New("a previous game's own school art error")), TownTavernArt: resolved(&ui.TownTavernArt{}, errors.New("a previous game's own tavern art error")), TownSquareArt: resolved(&ui.TownSquareArt{}, errors.New("a previous game's own square art error")), AttackPointer: resolved(&image.RGBA{}, errors.New("a previous game's own pointer error")), CursorRegistry: resolved(&ui.CursorRegistry{}, errors.New("a previous game's own cursor registry error")), BottomHUDArt: resolved(&ui.BottomHUDArt{}, errors.New("bottom HUD art unavailable")), CommandPanelArt: resolved(&ui.CommandPanelArt{}, errors.New("a previous game's own command panel art error")), Table: &mapload.Table{}, StartWeapon: resolved(&data.Weapon{}, errors.New("a previous game's own weapon error")), SackFrames: []*terrain.StaticFrame{{}}, SackBoundaries: []*terrain.StaticFrame{{}}, Bodies: data.BodyList{data.HeroBody("a")}, SoundBank: &SoundBank{}, SpeechBank: &SpeechBank{}, MusicBank: &MusicBank{}, SoundClasses: map[int32]UnitSound{1: {}}, Humans: fakeCollection{}, Campaign: resolved(Campaign{Declared: 15}, errors.New("a previous game's own campaign error"))}, RuntimeServices: RuntimeServices{TownAnimationNow: func() time.Time {
		return time.Unix(1116, 83)
	}, TownAnimationRandom: func(n int) int {
		return n - 1
	}, TownAmbientRandom: func(n int) int {
		return n / 4
	}, TavernRandom: func(n int) int {
		return n / 2
	}, ShopRandom: func(n int) int {
		return n / 3
	}, SchoolRandom: func(n int) int {
		return n / 4
	}, Sound: SoundOptions{Enabled: true, Volume: 50}, SoundChannels: audio.ChannelVolumes{25, 50, 75}, SpeechPlayer: fakePlayer{}, SoundPlayer: fakePlayer{}, acknowledgmentsOff: true, MusicPlayer: fakePlayer{}, MusicSeed: 1, AmbientPlayer: fakePlayer{}, AmbientSeed: 2, CutsceneAudioPlayer: fakeCutsceneAudioPlayer{}, runtime: runtimeSwitches{deterministicFrames: true, chicken: true}}, CampaignSession: CampaignSession{fame: SnapshotFame{Known: true, Time: 31, Events: 7, Result: &FameResult{Name: "old hero", Score: 19}}, quickSpells: [4]uint32{17, 5, 6, 89}, Difficulty: mapload.DifficultyHard, Carried: []mapload.PartyMember{{}}, Offered: 30, Town: &Town{gold: 999}, originalCity: &originalCitySaveState{}, live: &mapWorld{}, liveMission: 10, liveParty: []mapload.PartyMember{{}}, Shop: &Shop{}}, Presentation: Presentation{Markers: Markers{Objects: true, Units: true, Statics: true}, townLatches: townAmbientLatches{birdDelayReady: true, birdDelay: 1777 * time.Millisecond, starTerminalCnt: 6}, graphics: ui.GraphicsOptions{Smoothing: true, HideShadows: true, DisableLighting: true, StaticObjects: true}, showPathfinding: true, townUI: &townScreen{}, shopArtCache: resolved(&ui.ShopScreenArt{}, nil), tipArtCache: resolved(&ui.TipPanelArt{}, nil), dialogArt: resolved(&ui.DialogFrame{}, nil), docArtCache: resolved(&ui.DocumentPanelArt{}, nil), docFontCache: resolved(&text.Font{}, nil), characterPaneCache: resolved(TownCharacterPaneArt{Figure: ui.TownPane{Body: &image.RGBA{}}, Stats: ui.TownPane{Body: &image.RGBA{}}}, nil), characterCornerCache: resolved(&ui.CharacterPaneCornerArt{BackpackOpen: &image.RGBA{}, BackpackClosed: &image.RGBA{}, BookOpened: &image.RGBA{}, BookClosed: &image.RGBA{}, HumanMode: &image.RGBA{}, TextMode: &image.RGBA{}, Diskette: &image.RGBA{}, Ar1: &image.RGBA{}, Ar2: &image.RGBA{}}, nil), characterFillerCache: resolved(ui.TownPane{Body: &image.RGBA{}}, nil), worldMapCache: resolved(&worldMapAssets{}, nil)}, PersistenceContext: PersistenceContext{hallStore: fameStore{Dir: "local hall store"}, Options: OptionsStore{Path: "a previous game's own options path"}, tipsOff: true, smoothingOff: smoothingSwitches{text: true, frame: true}, wimpyMode: wimpyModeHigh}}
}

// settable returns an addressable, interfaceable reflect.Value for a struct
// field reflect would otherwise refuse to read or write, because several of
// FrontEnd's fields are unexported (live, liveMission, liveParty,
// deterministicFrames, townUI, shopArtCache, tipArtCache, tipsOff,
// worldMapCache). Same escape hatch townscreen_test.go's settable already
// uses. It is also what reads a lazy field, whose own three fields are
// unexported whatever the field holding it is called.
func settableFE(v reflect.Value) reflect.Value {
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}

// frontEndField is one leaf field of a FrontEnd and the owned component that
// declares it.
type frontEndField struct {
	component string
	name      string
	value     reflect.Value
}

// frontEndFields flattens a FrontEnd into its leaf fields, one level down
// through the five embedded components, so the walk below still enumerates
// every field a FrontEnd holds rather than the five structs holding them.
//
// It also REQUIRES that FrontEnd declares nothing but those components: a
// field added straight to FrontEnd has no owner, and the composition root is
// exactly the claim that every field has one.
func frontEndFields(t *testing.T, rv reflect.Value) []frontEndField {
	t.Helper()
	var out []frontEndField
	typ := rv.Type()
	for i := 0; i < typ.NumField(); i++ {
		comp := typ.Field(i)
		cv := rv.Field(i)
		if !comp.Anonymous || cv.Kind() != reflect.Struct {
			t.Fatalf("FrontEnd field %s is not an embedded owned component; every field belongs "+
				"to one of the five, and a field declared straight on FrontEnd belongs to none", comp.Name)
		}
		ct := cv.Type()
		for j := 0; j < ct.NumField(); j++ {
			out = append(out, frontEndField{
				component: comp.Name,
				name:      ct.Field(j).Name,
				value:     settableFE(cv.Field(j)),
			})
		}
	}
	return out
}

// TestResetSessionForNewGameDropsExactlyTheSessionPopulation walks every
// field of FrontEnd on a value seeded fully non-zero, calls
// resetSessionForNewGame, and requires every field back at its own expected
// value: nil (or the Go zero value) for the five session fields
// resetSessionForNewGame documents resetting, a freshly built *Town for Town
// specifically (NewTown(f.Campaign), not merely nil), and UNCHANGED for
// every other field, because each of those is the INSTALL's state rather
// than the GAME's — read once by NewFrontEnd out of the archive and handed
// to every game this process opens.
func TestResetSessionForNewGameDropsExactlyTheSessionPopulation(t *testing.T) {
	keepReason := "install-scoped: read once by NewFrontEnd out of the archive and handed to every game this process opens"
	keepUnchanged := map[string]string{
		"hallStore":           "local hall persistence configuration shared by campaigns",
		"Cutscenes":           "optional install-scoped video bank and helper selection, never campaign state",
		"Archives":            keepReason,
		"Assets":              keepReason,
		"Tiles":               keepReason,
		"Maps":                keepReason,
		"Faces":               keepReason,
		"NPCFaces":            keepReason,
		"Statics":             keepReason,
		"Units":               keepReason,
		"Structures":          keepReason,
		"Projectiles":         keepReason,
		"Markers":             "a diagnostic display option the -markers flag writes, not game progress",
		"Font":                keepReason,
		"Words":               keepReason,
		"ChargenAssets":       keepReason,
		"TownSchoolArt":       keepReason,
		"TownAnimationNow":    "the process-level presentation clock service; animation phase belongs to townScreen and resets there",
		"TownAnimationRandom": "the process-level presentation random service; per-visit generator state belongs to townScreen and resets there",
		"TownAmbientRandom":   "the separate process-level bird random service; it cannot perturb the established sign/fluger stream",
		"townLatches":         "process-level presentation latches and counter, retained independently of a campaign",
		"TavernRandom":        "the separate process-level tavern presentation random service; per-visit generator state belongs to townScreen and resets there",
		"ShopRandom":          "the separate process-level shop presentation random service; per-visit generator state belongs to townScreen and resets there",
		"SchoolRandom":        "the separate process-level school presentation random service; object-local generator state belongs to townScreen and resets there",
		"TownTavernArt":       keepReason,
		"TownSquareArt":       keepReason,
		"AttackPointer":       keepReason,
		"CursorRegistry":      keepReason,
		"BottomHUDArt":        keepReason,
		"CommandPanelArt":     keepReason,
		"Table":               keepReason,
		"StartWeapon":         keepReason,
		"SackFrames":          keepReason,
		"SackBoundaries":      keepReason,
		"graphics":            "process-local rendering preferences survive new campaigns without entering game progress",
		"Bodies":              keepReason,
		"Sound":               "a snapshot of package sound state taken at construction (spec FR-8, FR-16), not game progress",
		"SoundPlayer":         "the opened device itself, reused for every game this process opens",
		"SoundChannels":       "process-local playback preferences remain outside game progress",
		"SpeechPlayer":        "the independent speech device is reused across games",
		"acknowledgmentsOff":  "process-local voice preference remains outside game progress",
		"showPathfinding":     "display preference remains outside game progress",
		"SoundBank":           keepReason,
		"SpeechBank":          "the lazy install-backed speech index, reused across games; the active voice belongs to townScreen",
		"MusicBank":           "the lazy install-backed music index, reused for every game this process opens",
		"MusicPlayer":         "the retained music device, reused for every game this process opens",
		"MusicSeed":           "the process-local presentation shuffle seed, not game progress",
		"AmbientPlayer":       "the retained mission-ambience device, reused for every game this process opens",
		"AmbientSeed":         "the process-local ambience seed, not game progress",
		"CutsceneAudioPlayer": "the retained cutscene-audio device (story1178), reused for every game this process opens",
		"SoundClasses":        keepReason,
		"Humans":              keepReason,
		"Campaign":            "the scenario registry's own declared mission set, read once; Town is built OVER it and Town is what resets",

		"runtime": "the launch and scenario-runner's own frame adapter selection, a process option and not game progress",
		"townUI":  "the one town screen App installed, held for the life of the process (TownScreen's own doc); resetForNewGame already clears its per-game fields",

		"shopArtCache": "the shop's own art, resolved once from the install and cached across every game",
		"tipArtCache":  "the tip panel's own art, cached on shopArtCache's own precedent",
		"dialogArt":    "shared installed window, portrait and minimap artwork",
		"docArtCache":  "the documents panel's own art, cached on tipArtCache's own precedent (story 1035)",
		"docFontCache": "font4, the documents panel's own text font, cached on tipArtCache's own precedent (story 1035)",

		"characterPaneCache":   "the shared character panel's own two mode-switched bodies and seams (1021 spec B3, round-1 adversarial review P-1), cached on tipArtCache's own precedent",
		"characterCornerCache": "the character pane's nine shipped corner bitmaps (story 1036), cached on characterPaneCache's own precedent",
		"characterFillerCache": "the mission column's fourth child's own background (story 1036 round 3, P-B), cached on characterCornerCache's own precedent",

		"Options":      "the persisted preference store's path (tipstore.go), a process option and not game progress",
		"tipsOff":      "mirrors Options.TipsMode(), read once by LoadOptions and not from disk per frame",
		"smoothingOff": "mirrors Options.TextSmoothing() (DIV-1385) and Options.FrameSmoothing(), read once by LoadOptions and not from disk per frame",
		"wimpyMode":    "mirrors Options.WimpyMode(), the process-wide next-cycle label and not one campaign's progress",

		"worldMapCache": "the install-backed campaign-map manifest, loaded once and retained for the process",
		"endingAssets":  "read-only installed credits, hall table and background; no campaign result",
	}

	fe := nonZeroFrontEnd()
	fields := frontEndFields(t, reflect.ValueOf(fe).Elem())

	seeded := make(map[string]interface{}, len(fields))
	for _, f := range fields {
		val := f.value.Interface()
		seeded[f.name] = val
		if reflect.DeepEqual(val, reflect.Zero(f.value.Type()).Interface()) {
			t.Fatalf("fixture field %s.%s is still its own Go zero value; nonZeroFrontEnd must "+
				"seed it non-zero, or this test cannot tell a real reset from a field it never "+
				"touched", f.component, f.name)
		}
	}

	fe.resetSessionForNewGame()

	for _, f := range frontEndFields(t, reflect.ValueOf(fe).Elem()) {
		name := f.name
		got := f.value.Interface()

		if reason, kept := keepUnchanged[name]; kept {
			if f.component == "CampaignSession" {
				t.Errorf("field %s is kept across a new game but is declared in CampaignSession, "+
					"whose whole definition is what resetSessionForNewGame drops", name)
			}
			if name == "TownAnimationRandom" || name == "TownAmbientRandom" || name == "TavernRandom" || name == "ShopRandom" || name == "SchoolRandom" {
				roll, original := got.(func(int) int), seeded[name].(func(int) int)
				if roll == nil || roll(100) != original(100) || roll(7) != original(7) {
					t.Errorf("presentation random service changed across reset")
				}
				continue
			}
			if name == "TownAnimationNow" {
				// DeepEqual rejects every nonnil function. Verify this named
				// service's identity AND its reading, without exempting any
				// unclassified field from the population check.
				clock := got.(func() time.Time)
				original := seeded[name].(func() time.Time)
				if clock == nil || reflect.ValueOf(clock).Pointer() != reflect.ValueOf(original).Pointer() || clock() != original() {
					t.Errorf("field %s changed by resetSessionForNewGame, but is kept (%s)", name, reason)
				}
				continue
			}
			if !reflect.DeepEqual(got, seeded[name]) {
				t.Errorf("field %s changed by resetSessionForNewGame, but is kept (%s): got %#v, want unchanged %#v",
					name, reason, got, seeded[name])
			}
			continue
		}

		if f.component != "CampaignSession" {
			t.Errorf("field %s.%s is reset for a new game but is not declared in CampaignSession; "+
				"move it there or classify it in keepUnchanged", f.component, name)
		}

		// Town is rebuilt, not zeroed: the reset method's own doc says it
		// starts a second game exactly as NewFrontEnd starts the first, over
		// the SAME Campaign the fixture seeded, rather than leaving a nil
		// pointer every reader would have to guard for. Compare it against a
		// like-built Town instead of the Go zero value.
		if name == "Town" {
			want := NewTown(fe.Campaign.Value())
			if !reflect.DeepEqual(got, want) {
				t.Errorf("field Town = %#v after resetSessionForNewGame, want a fresh NewTown(f.Campaign): %#v", got, want)
			}
			continue
		}

		want := reflect.Zero(f.value.Type()).Interface()
		if !reflect.DeepEqual(got, want) {
			t.Errorf("field %s = %#v after resetSessionForNewGame, want %#v — classify this field in "+
				"resetSessionForNewGame (reset it) or in this test's keepUnchanged (name why it is the "+
				"install's, not the game's) if it is new", name, got, want)
		}
	}
}
