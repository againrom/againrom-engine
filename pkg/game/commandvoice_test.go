package game

import (
	"io"
	"slices"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestCommandVoiceObserverForwardsTypedAdmissionAndRefusal(t *testing.T) {
	restore := ui.SetDeliveryPlayerFactory(func(reader io.ReadSeeker) (ui.DeliveryDevicePlayer, error) {
		return &deliveryWitnessPlayer{reader: reader}, nil
	})
	t.Cleanup(restore)
	owner, err := ui.OpenSharedAudio(audio.Settings{Master: 100}, audio.Settings{Master: 100})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(owner.Service.Close)
	if err := audioWitnessClosed(owner); err == nil {
		t.Fatal("shutdown instrument accepted an open service")
	}
	observer := &commandVoiceObserver{Player: owner.NewScope().Player(audio.SpeechChannel)}
	sample := audio.Sample{Rate: audio.DeviceRate, PCM: []int16{11, 11}}
	request := audio.FixedRequest("unit-selection", "voice:mf_hero/select1.wav", audio.SpeechChannel, 128, false,
		audio.Placement{Left: 10000, Right: 10000})
	if voice := audio.Dispatch(observer, sample, request); voice == nil || !voice.Playing() {
		t.Fatal("typed observer failed to reach the actual shared service/backend")
	}
	if observer.plays != 1 || observer.legacy != 0 || len(observer.requests) != 1 || observer.requests[0] != request ||
		owner.Service.Snapshot().Counters.Admitted != 1 || owner.BackendState().Created != 1 {
		t.Fatal("observer lost semantic request or shared admission", observer, owner.Service.Snapshot())
	}
	owner.Service.Close()
	if err := audioWitnessClosed(owner); err != nil {
		t.Fatal(err)
	}
	if audio.Dispatch(observer, sample, request) != nil || observer.plays != 1 || len(observer.requests) != 2 ||
		owner.Service.Snapshot().Counters.Refused != 1 || observer.legacy != 0 {
		t.Fatal("closed-service refusal was counted as admitted or took a legacy fallback", observer)
	}
}

// A fixture recording carries 100*(bank place+1)+leaf code.
var voiceLeafCodes = map[string]int16{
	"select1": 1, "select2": 2, "command1": 3, "command2": 4, "command3": 5,
	"retreat": 6, "defend": 7, "idle": 8,
}

func voiceBankFixture(skip ...string) *SoundBank {
	named := map[string]soundCacheEntry{}
	for i, bank := range hurtVoiceBanks {
		for leaf, code := range voiceLeafCodes {
			name := bank + "/" + leaf + ".wav"
			if slices.Contains(skip, name) {
				continue
			}
			named[name] = soundCacheEntry{sample: audio.Sample{PCM: []int16{int16(100*(i+1)) + code}}, ok: true}
		}
	}
	return &SoundBank{named: named}
}

// voiceSample is the fixture value of a bank's leaf.
func voiceSample(bank, leaf string) int16 {
	return int16(100*(slices.Index(hurtVoiceBanks, bank)+1)) + voiceLeafCodes[leaf]
}

// scriptedDraw answers the listed draws in order and then 0, and counts them.
type scriptedDraw struct {
	values []int
	used   int
}

func (s *scriptedDraw) draw() int {
	s.used++
	if s.used > len(s.values) {
		return 0
	}
	return s.values[s.used-1]
}

func (s *scriptedDraw) set(values ...int) { s.values, s.used = values, 0 }

type voicePerson struct {
	typeID int32
	dir    data.FigureDir
	weapon bool
	owner  uint32
	hp     int32
}

// voiceFixture builds a world holding the people as entities 1..n.
func voiceFixture(t *testing.T, people ...voicePerson) *mapWorld {
	t.Helper()
	var ents []sim.Entity
	var stocks []sim.Stock
	figures := map[sim.EntityID]figureID{}
	for i, p := range people {
		id := sim.EntityID(i + 1)
		ents = append(ents, sim.Entity{ID: id, X: int32(i % swingW), Y: int32(i / swingW), Owner: p.owner,
			TypeID: p.typeID, Class: 1, Humanoid: p.dir != "", HP: p.hp, MaxHP: 10})
		if p.weapon {
			stocks = append(stocks, sim.Stock{ID: id, Equipped: [sim.EquipSlots]uint16{0: eqSwordCode}})
		}
		if p.dir != "" {
			figures[id] = figureID{Dir: p.dir, Hero: data.FigureIsHero(p.typeID)}
		}
	}
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: swingW, Height: swingH}, sim.ModeCanonical, sim.Terrain{},
		ents, nil, sim.Relations{}, nil, stocks)
	if err != nil {
		t.Fatal(err)
	}
	v, err := ui.NewViewer("voices", terrain.Grid{
		Width: swingW, Height: swingH, Tiles: make([]uint16, swingW*swingH),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	v.Layout(640, 480)
	mw := newMapWorld(w, nil, &terrain.UnitSet{}, v)
	mw.figures = figures
	return mw
}

var (
	heroMan   = voicePerson{typeID: sim.HeroTypeID(false, false), dir: data.FigureDirManFighter, owner: sim.SelfSlot, hp: 10}
	heroWoman = voicePerson{typeID: sim.HeroTypeID(false, true), dir: data.FigureDirWomanFighter, owner: sim.SelfSlot, hp: 10}
	armedMan  = voicePerson{typeID: 14, dir: data.FigureDirManFighter, weapon: true, owner: sim.SelfSlot, hp: 10}
	bareMan   = voicePerson{typeID: 1, dir: data.FigureDirManFighter, owner: sim.SelfSlot, hp: 10}
	bareWoman = voicePerson{typeID: 1, dir: data.FigureDirWomanFighter, owner: sim.SelfSlot, hp: 10}
	creature  = voicePerson{typeID: 0x40, owner: sim.SelfSlot, hp: 10}
)

func voiceFront(mw *mapWorld, bank *SoundBank) (*FrontEnd, *acknowledgmentRecorder) {
	voices := &acknowledgmentRecorder{}
	return &FrontEnd{InstallResources: InstallResources{SoundBank: bank}, RuntimeServices: RuntimeServices{SpeechPlayer: voices}}, voices
}

// TestSpeakerIsOneMemberOfTheFirstNonEmptyTier checks tier order, member
// tests, the index formula and that nobody qualifying takes no draw.
func TestSpeakerIsOneMemberOfTheFirstNonEmptyTier(t *testing.T) {
	dead := bareMan
	dead.hp = 0
	foreignHero := heroWoman
	foreignHero.owner = 2
	mw := voiceFixture(t, heroMan, armedMan, bareMan, creature, dead, heroWoman, foreignHero)
	const hero, armed, bare, beast, fallen, hero2, foreign = 1, 2, 3, 4, 5, 6, 7
	script := &scriptedDraw{}
	pick := func(draw int, ids ...uint32) (uint32, bool) {
		script.set(draw)
		e, _, ok := mw.speakerOf(ids, script.draw)
		return uint32(e.ID), ok
	}
	for _, tc := range []struct {
		name string
		ids  []uint32
		draw int
		want uint32
		ok   bool
	}{
		{"a hero before an armed and an unarmed human", []uint32{armed, bare, hero}, 0, hero, true},
		{"a hero after them", []uint32{hero, armed, bare}, 32766, hero, true},
		{"armed before unarmed", []uint32{bare, armed}, 0, armed, true},
		{"unarmed alone", []uint32{bare, beast}, 0, bare, true},
		{"a dead hero is no candidate", []uint32{fallen, bare}, 0, bare, true},
		{"two heroes, low draw", []uint32{hero, hero2}, 0, hero, true},
		{"two heroes, high draw", []uint32{hero, hero2}, 20000, hero2, true},
		{"a foreign hero is a candidate", []uint32{hero, foreign}, 32766, foreign, true},
		{"the slot one past the last member", []uint32{hero, hero2}, 32767, 0, false},
		{"a creature alone", []uint32{beast}, 0, 0, false},
	} {
		if got, ok := pick(tc.draw, tc.ids...); got != tc.want || ok != tc.ok {
			t.Errorf("%s: speaker %d (%v), want %d (%v)", tc.name, got, ok, tc.want, tc.ok)
		}
	}
	script.set()
	if _, _, ok := mw.speakerOf([]uint32{beast, fallen}, script.draw); ok || script.used != 0 {
		t.Errorf("nobody qualifies: ok %v after %d draws, want none and no draw", ok, script.used)
	}
	// Three heroes split at 10922|10923 and, by the formula, 21844|21845; the
	// claim's executed list puts the second boundary one draw higher.
	three := voiceFixture(t, heroMan, heroWoman, heroMan)
	for draw, want := range map[int]uint32{0: 1, 10922: 1, 10923: 2, 21844: 2, 21846: 3, 32766: 3} {
		script.set(draw)
		if e, _, _ := three.speakerOf([]uint32{1, 2, 3}, script.draw); uint32(e.ID) != want {
			t.Errorf("three heroes, draw %d: speaker %d, want %d", draw, e.ID, want)
		}
	}
}

// TestCommandRecordingByGesture checks each gesture's recording and draws.
func TestCommandRecordingByGesture(t *testing.T) {
	script := &scriptedDraw{}
	for _, tc := range []struct {
		bank string
		draw int
		want string
	}{
		{"mf_hero", 0, "command1"}, {"mf_hero", 8191, "command1"},
		{"mf_hero", 8192, "command2"}, {"mf_hero", 16383, "command2"},
		{"mf_hero", 16384, "command3"}, {"mf_hero", 24575, "command3"},
		{"mf_hero", 24576, "defend"}, {"mf_hero", 32767, "defend"},
		{"m_peasant", 24576, "command1"}, {"f_peasant", 32767, "command1"},
		{"f_peasant", 8192, "command2"}, {"ff_merc", 32767, "defend"},
	} {
		script.set(tc.draw)
		if got := commandRecording(ui.VoiceMove, tc.bank, script.draw); got != tc.want || script.used != 1 {
			t.Errorf("%s at draw %d: %s after %d draws, want %s after 1", tc.bank, tc.draw, got, script.used, tc.want)
		}
	}
	for gesture, want := range map[ui.VoiceGesture]string{
		ui.VoiceGuard: "defend", ui.VoiceStandGround: "defend", ui.VoiceDefend: "defend",
		ui.VoiceRetreat: "retreat", ui.VoicePickup: "idle",
	} {
		script.set(24576)
		if got := commandRecording(gesture, "mf_hero", script.draw); got != want || script.used != 0 {
			t.Errorf("gesture %d: %s after %d draws, want %s and no draw", gesture, got, script.used, want)
		}
	}
	for _, gesture := range []ui.VoiceGesture{ui.VoiceMove, ui.VoiceAttack, ui.VoiceSwarm, ui.VoicePatrol, ui.VoiceTown} {
		script.set(8192)
		if got := commandRecording(gesture, "mf_hero", script.draw); got != "command2" || script.used != 1 {
			t.Errorf("gesture %d: %s after %d draws, want command2 after 1", gesture, got, script.used)
		}
	}
}

// TestCommandReplyPlaysTheBankRecordingOfTheGesture runs each gesture through
// the reply and requires the speaker's bank recording.
func TestCommandReplyPlaysTheBankRecordingOfTheGesture(t *testing.T) {
	mw := voiceFixture(t, heroMan, bareWoman)
	f, voices := voiceFront(mw, voiceBankFixture())
	script := &scriptedDraw{}
	command, _ := f.runtimeAudio().unitReplies(mw, script.draw)
	now := time.Unix(100, 0)
	for _, tc := range []struct {
		gesture ui.VoiceGesture
		ids     []uint32
		draws   []int
		bank    string
		leaf    string
		drawn   int
	}{
		{ui.VoiceMove, []uint32{1}, []int{0, 16384}, "mf_hero", "command3", 2},
		{ui.VoiceAttack, []uint32{1}, []int{0, 24576}, "mf_hero", "defend", 2},
		{ui.VoiceSwarm, []uint32{2}, []int{0, 24576}, "f_peasant", "command1", 2},
		{ui.VoicePatrol, []uint32{1}, []int{0, 8192}, "mf_hero", "command2", 2},
		{ui.VoiceTown, []uint32{1}, []int{0, 0}, "mf_hero", "command1", 2},
		{ui.VoiceGuard, []uint32{1}, []int{0}, "mf_hero", "defend", 1},
		{ui.VoiceStandGround, []uint32{2}, []int{0}, "f_peasant", "defend", 1},
		{ui.VoiceDefend, []uint32{1}, []int{0}, "mf_hero", "defend", 1},
		{ui.VoiceRetreat, []uint32{2}, []int{0}, "f_peasant", "retreat", 1},
		{ui.VoicePickup, []uint32{1}, []int{0}, "mf_hero", "idle", 1},
	} {
		now = now.Add(time.Minute)
		mark := len(voices.samples)
		script.set(tc.draws...)
		command(tc.gesture, tc.ids, now)
		if got, want := heard(voices, mark), []int16{voiceSample(tc.bank, tc.leaf)}; !slices.Equal(got, want) || script.used != tc.drawn {
			t.Errorf("gesture %d: requested %v after %d draws, want %v (%s/%s) after %d", tc.gesture, got, script.used, want, tc.bank, tc.leaf, tc.drawn)
		}
	}
}

// TestReplyWaitsOnTheSpeakersOneStamp checks the thresholds on the shared
// stamp, refusal without a second speaker, and a missing recording's stamp.
func TestReplyWaitsOnTheSpeakersOneStamp(t *testing.T) {
	mw := voiceFixture(t, heroMan, heroWoman, bareMan, heroMan)
	f, voices := voiceFront(mw, voiceBankFixture("m_peasant/idle.wav"))
	script := &scriptedDraw{}
	command, selection := f.runtimeAudio().unitReplies(mw, script.draw)
	t0 := time.Unix(100, 0)
	at := func(ms int) time.Time { return t0.Add(time.Duration(ms) * time.Millisecond) }
	plays := func(do func()) int {
		mark := len(voices.samples)
		script.set()
		do()
		return len(voices.samples) - mark
	}
	attack := func(ms int, ids ...uint32) func() { return func() { command(ui.VoiceAttack, ids, at(ms)) } }
	choose := func(ms int, ids ...uint32) func() { return func() { selection(ids, at(ms)) } }
	for _, step := range []struct {
		name string
		do   func()
		want int
	}{
		{"first command reply", attack(0, 1), 1},
		{"same speaker at 2999 ms", attack(2999, 1), 0},
		{"same speaker at 3000 ms", attack(3000, 1), 1},
		{"a selection reply 1999 ms after a command reply", choose(4999, 1), 0},
		{"a selection reply 2000 ms after it", choose(5000, 1), 1},
		{"a command reply 2999 ms after a selection reply", attack(7999, 1), 0},
		{"a command reply 3000 ms after it", attack(8000, 1), 1},
		{"another speaker has its own stamp", attack(8001, 2), 1},
	} {
		if got := plays(step.do); got != step.want {
			t.Errorf("%s: %d recordings, want %d", step.name, got, step.want)
		}
	}
	// The draw names the hero on cooldown: no reply, no reader draw, and the
	// free hero is not tried.
	script.set(0)
	mark := len(voices.samples)
	command(ui.VoiceAttack, []uint32{1, 4}, at(8002))
	if len(voices.samples) != mark || script.used != 1 {
		t.Errorf("a speaker on cooldown: %d recordings after %d draws, want none after 1", len(voices.samples)-mark, script.used)
	}
	// A missing recording stores the stamp and plays nothing.
	mark = len(voices.samples)
	command(ui.VoicePickup, []uint32{3}, at(30000))
	command(ui.VoiceAttack, []uint32{3}, at(32999))
	command(ui.VoiceAttack, []uint32{3}, at(33000))
	if got := heard(voices, mark); len(got) != 1 {
		t.Errorf("a missing idle recording: %v, want the later reply only", got)
	}
}

// TestMoveAndSwarmAreSilentWhileTheSpeakerIsDrawnMoving checks the gate: the
// speaker is drawn first, nobody else is tried, no stamp is stored.
func TestMoveAndSwarmAreSilentWhileTheSpeakerIsDrawnMoving(t *testing.T) {
	mw := voiceFixture(t, heroMan, heroWoman)
	f, voices := voiceFront(mw, voiceBankFixture())
	script := &scriptedDraw{}
	command, _ := f.runtimeAudio().unitReplies(mw, script.draw)
	now := time.Unix(100, 0)
	mw.drawnMoving = map[sim.EntityID]bool{1: true}
	for _, gesture := range []ui.VoiceGesture{ui.VoiceMove, ui.VoiceSwarm} {
		script.set(0)
		command(gesture, []uint32{1, 2}, now)
		if len(voices.samples) != 0 || script.used != 1 {
			t.Fatalf("gesture %d for a moving speaker: %d recordings after %d draws, want none after 1", gesture, len(voices.samples), script.used)
		}
	}
	for i, gesture := range []ui.VoiceGesture{ui.VoiceAttack, ui.VoicePatrol, ui.VoiceTown, ui.VoiceDefend, ui.VoiceGuard, ui.VoiceRetreat, ui.VoicePickup} {
		now = now.Add(time.Minute)
		command(gesture, []uint32{1, 2}, now)
		if len(voices.samples) != i+1 {
			t.Fatalf("gesture %d for a moving speaker: %d recordings, want %d", gesture, len(voices.samples), i+1)
		}
	}
}

// TestSelectionReplyNeedsAnOwnedPrimaryAndTheOption checks the primary gate
// and the preference.
func TestSelectionReplyNeedsAnOwnedPrimaryAndTheOption(t *testing.T) {
	foe := heroMan
	foe.owner = 2
	mw := voiceFixture(t, heroMan, foe, bareMan)
	f, voices := voiceFront(mw, voiceBankFixture())
	script := &scriptedDraw{}
	_, selection := f.runtimeAudio().unitReplies(mw, script.draw)
	hash := mw.world.Hash()
	now := time.Unix(100, 0)
	for _, tc := range []struct {
		name string
		ids  []uint32
		want int
	}{
		{"a foreign primary", []uint32{2}, 0},
		{"a foreign primary before an own unit", []uint32{2, 1}, 0},
		{"no selection", nil, 0},
		{"an unknown unit", []uint32{9000}, 0},
		{"an own primary", []uint32{1}, 1},
	} {
		now = now.Add(time.Minute)
		mark := len(voices.samples)
		script.set(0, 16384)
		selection(tc.ids, now)
		if got := len(voices.samples) - mark; got != tc.want || tc.want == 0 && script.used != 0 {
			t.Errorf("%s: %d recordings after %d draws, want %d", tc.name, got, script.used, tc.want)
		}
	}
	if got := heard(voices, 0); !slices.Equal(got, []int16{voiceSample("mf_hero", "select2")}) {
		t.Errorf("select2 at draw 16384: %v", got)
	}
	f.acknowledgmentsOff = true
	script.set()
	selection([]uint32{1}, now.Add(time.Hour))
	if len(voices.samples) != 1 || script.used != 0 {
		t.Error("the option off spoke or drew")
	}
	if mw.world.Hash() != hash {
		t.Error("a reply changed the World")
	}
}

// TestViewerVoiceDrawStaysInRange checks the generator spans 0..32767.
func TestViewerVoiceDrawStaysInRange(t *testing.T) {
	draw := newVoiceDraw()
	lo, hi := voiceDrawRange, 0
	for range 20000 {
		d := draw()
		lo, hi = min(lo, d), max(hi, d)
		if d < 0 || d > voiceDrawRange {
			t.Fatalf("draw %d is outside 0..%d", d, voiceDrawRange)
		}
	}
	if lo > 200 || hi < voiceDrawRange-200 {
		t.Fatalf("20000 draws span %d..%d", lo, hi)
	}
}
