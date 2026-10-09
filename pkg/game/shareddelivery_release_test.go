package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

type deliveryWitnessPlayer struct {
	reader        io.ReadSeeker
	playing       bool
	position      time.Duration
	gain          float64
	closes, seeks int
}

func (p *deliveryWitnessPlayer) Play()           { p.playing = true }
func (p *deliveryWitnessPlayer) IsPlaying() bool { return p.playing }
func (p *deliveryWitnessPlayer) Pause()          { p.playing = false }
func (p *deliveryWitnessPlayer) Seek(at time.Duration) error {
	p.position = at
	p.seeks++
	_, err := p.reader.Seek(int64(at)*audio.DeviceRate*4/int64(time.Second), io.SeekStart)
	return err
}
func (p *deliveryWitnessPlayer) Position() time.Duration { return p.position }
func (p *deliveryWitnessPlayer) SetVolume(gain float64)  { p.gain = gain }
func (p *deliveryWitnessPlayer) Volume() float64         { return p.gain }
func (p *deliveryWitnessPlayer) Close() error            { p.closes++; p.playing = false; return nil }

type deliveryWitness struct {
	t            *testing.T
	front        *FrontEnd
	app          *ui.App
	owner        *ui.SharedAudio
	dir          string
	players      []*deliveryWitnessPlayer
	restore      func()
	now          time.Time
	holdOneShots bool
}

func newDeliveryWitness(t *testing.T, dir string, controlled bool) *deliveryWitness {
	t.Helper()
	w := &deliveryWitness{t: t, dir: dir, now: time.Unix(100, 0)}
	if controlled {
		w.restore = ui.SetDeliveryPlayerFactory(func(reader io.ReadSeeker) (ui.DeliveryDevicePlayer, error) {
			p := &deliveryWitnessPlayer{reader: reader}
			w.players = append(w.players, p)
			return p, nil
		})
		t.Cleanup(w.restore)
	}
	w.front = releaseFront(t)
	w.front.Options = OptionsStore{}
	w.front.SetDeterministicFrames(true)
	w.front.TownAnimationNow = func() time.Time { return w.now }
	w.front.TownAnimationRandom = func(int) int { return 0 }
	w.front.TownAmbientRandom = func(int) int { return 0 }
	w.front.TavernRandom = func(int) int { return 0 }
	w.front.SchoolRandom = func(int) int { return 0 }
	w.owner = ui.DeliveryOwner(w.front.SoundPlayer)
	if w.owner == nil || w.owner != ui.DeliveryOwner(w.front.SpeechPlayer) {
		t.Fatal("instrument: production has no one effects/speech owner")
	}
	w.app = w.front.App("shared audio delivery")
	w.app.SetCutscenes(nil)
	w.app.Layout(640, 480)
	t.Cleanup(w.app.StopAudio)
	return w
}
func (w *deliveryWitness) check(err error) {
	w.t.Helper()
	if err != nil {
		w.t.Fatalf("instrument: %v", err)
	}
}
func (w *deliveryWitness) finishOneShots() {
	if w.holdOneShots {
		return
	}
	for _, b := range w.owner.BackendState().Buffers {
		if !b.Repeat && int(b.ID) <= len(w.players) {
			w.players[b.ID-1].playing = false
		}
	}
}
func (w *deliveryWitness) paint(dt time.Duration) {
	w.t.Helper()
	w.now = w.now.Add(dt)
	w.app.Draw(ebiten.NewImage(640, 480))
	w.finishOneShots()
}
func (w *deliveryWitness) click(point image.Point) {
	w.t.Helper()
	w.check(w.app.HeadlessPointer("press", point.X, point.Y))
	w.check(w.app.HeadlessPointer("release", point.X, point.Y))
	w.finishOneShots()
}
func (w *deliveryWitness) sources() map[string]int {
	result := map[string]int{}
	for _, r := range w.owner.Service.Snapshot().Receipts {
		if r.Reason == audio.DeliveryAdmitted || r.Reason == audio.DeliverySampleBusy || r.Reason == audio.DeliveryChannelsBusy {
			result[r.Source]++
		}
	}
	return result
}
func (w *deliveryWitness) requireSource(source string) {
	w.t.Helper()
	if w.sources()[source] == 0 {
		w.t.Fatalf("instrument: App never reached %s; sources=%v", source, w.sources())
	}
}
func (w *deliveryWitness) receipt(name string) {
	w.t.Helper()
	out := struct {
		Service audio.DeliverySnapshot
		Backend ui.DeliveryBackendState
	}{w.owner.Service.Snapshot(), w.owner.BackendState()}
	raw, err := json.MarshalIndent(out, "", "  ")
	w.check(err)
	w.check(os.WriteFile(filepath.Join(w.dir, name+".json"), raw, 0600))
}
func (w *deliveryWitness) openTown() {
	w.t.Helper()
	f := w.front
	f.Carried = f.NextParty()
	mage := f.ChargenParty(ui.ChargenResult{Name: "Delivery mage", Choices: []int{1, 1, 0}, Stats: []int{20, 20, 30, 30}})
	if len(mage) != 1 {
		w.t.Fatal("instrument: installed chargen did not create mage")
	}
	mage[0].ID, mage[0].StartingHero = "delivery-mage", false
	f.Carried = append(f.Carried, mage...)
	f.arriveInTown()
	f.Town.gold = 100000
	f.Town.announceMission(f.Town.currentMain())
	snap, label, err := f.Snapshot(false)
	w.check(err)
	raw, err := f.ExportCurrentSave(snap, label)
	w.check(err)
	store := SaveStore{Dir: filepath.Join(w.dir, "town-saves")}
	w.check(os.MkdirAll(store.Dir, 0700))
	_, err = store.WriteOriginal(f.Archives.Root, raw)
	w.check(err)
	f.ConfigureSaveSeams(w.app, store, OriginalStore{}, nil)
	for n := 0; w.app.Screen() == ui.ScreenChargen && n < 4; n++ {
		w.check(w.app.HeadlessKey("escape"))
	}
	w.check(w.app.HeadlessKey("load"))
	w.check(w.app.HeadlessActivate("@first"))
	w.check(w.app.HeadlessStep())
	if w.app.Screen() != ui.ScreenTown {
		w.t.Fatalf("instrument: ordinary town LOAD returned %v", w.app.Screen())
	}
	f.townUI.CloseTip()
	w.finishOneShots()
}
func (w *deliveryWitness) fixedAppRoutes() {
	w.t.Helper()
	f := w.front
	a := w.app
	w.check(a.HeadlessActivate("hall of fame"))
	w.click(image.Pt(576, 432))
	w.requireSource("hall-of-fame")
	w.check(a.HeadlessActivate("new game"))
	if a.Screen() == ui.ScreenPicker {
		w.check(a.HeadlessActivate("@first"))
	}
	w.requireSource("main-menu")
	c := ui.NewChargen(f.ChargenSetup())
	c.CloseTip()
	w.check(a.OpenChargen(c, func(res ui.ChargenResult) (ui.MapOpener, error) { return f.NewGameOpener(10, res), nil }))
	find := func(owner string) image.Point {
		w.t.Helper()
		for y := 0; y < 480; y++ {
			for x := 0; x < 640; x++ {
				if got, ok := ui.PreCreateControlAt(c, image.Pt(x, y)); ok && got == owner {
					return image.Pt(x, y)
				}
			}
		}
		w.t.Fatalf("instrument: no pre-create control %s", owner)
		return image.Point{}
	}
	w.click(find("difficulty 2"))
	w.click(find("choice 0"))
	w.click(find("forward"))
	w.requireSource("character-precreate")
	w.click(image.Pt(142, 64))
	w.requireSource("character-detail")
	w.openTown()
	s := f.townUI
	w.paint(0)
	mask := f.TownSquareArt.Value().Mask
	found := false
	for y := 0; y < 480 && !found; y++ {
		for x := 0; x < 640; x++ {
			if mask.ColorIndexAt(x, y) == 0xb0 {
				w.check(a.HeadlessPointer("hover", x, y))
				found = true
				break
			}
		}
	}
	if !found {
		w.t.Fatal("instrument: no installed star mask pixel")
	}
	w.paint(68 * time.Millisecond)
	w.requireSource("town-exterior")
	w.requireSource("town-crowd")
	w.callerRetryRoutes()
	w.check(a.HeadlessActivate("TAVERN"))
	s.CloseTip()
	w.check(a.HeadlessStep())
	w.paint(0)
	w.paint(11 * time.Second)
	w.requireSource("tavern-interior")
	water := false
	for _, r := range w.owner.Service.Snapshot().Receipts {
		if r.Reason == audio.DeliveryAdmitted && r.Selector == "town/inn/water.wav" {
			water = water || r.Repeat
		}
	}
	if !water {
		w.receipt("water-instrument-failure")
		w.t.Fatal("water did not enter true repeat delivery", s.room, s.tavernPage().Ready(), s.tavernPage().Active(), a.Screen())
	}
	w.check(a.HeadlessKey("escape"))
	w.check(a.HeadlessActivate("SCHOOL"))
	w.check(a.HeadlessStep())
	for n := 0; s.room == roomTalk && n < 64; n++ {
		w.check(a.HeadlessActivate("dialogue"))
	}
	if s.room != roomSchool {
		w.t.Fatalf("instrument: school entry remained %v", s.room)
	}
	s.CloseTip()
	for n := 0; s.schoolTrainingBusy() && n < 64; n++ {
		w.paint(84 * time.Millisecond)
	}
	view := s.TownSurface()
	at := image.Point{}
	found = false
	for y := 0; y < 480 && !found; y++ {
		for x := 0; x < 640; x++ {
			if ctl, ok := ui.TownSurfaceControlAt(view, image.Pt(x, y)); ok && ctl.Kind == ui.TownSurfaceControlCell && ctl.Index == 2 {
				at, found = image.Pt(x, y), true
				break
			}
		}
	}
	if !found {
		w.t.Fatal("instrument: no installed school skill cell")
	}
	w.click(at)
	w.requireSource("school-skill")
	w.click(image.Pt(554, 94))
	w.requireSource("town-response")
	w.requireSource("school-training")
	w.click(image.Pt(614, 459))
	for n := 0; s.schoolTrainingBusy() && n < 64; n++ {
		w.paint(84 * time.Millisecond)
	}
	rotate := false
	for _, r := range w.owner.Service.Snapshot().Receipts {
		rotate = rotate || r.Reason == audio.DeliveryAdmitted && r.Selector == schoolRotateSound
	}
	if !rotate {
		w.t.Fatal("installed class change did not reach rotate delivery")
	}
	s.atSquare()
	if !s.openTownDialogue(TownTavern, TownOffer{Mission: 100}, 25) {
		w.t.Fatal("instrument: installed tavern topic100/npc25 missing")
	}
	w.check(a.HeadlessStep())
	w.requireSource("town-dialogue")
	w.receipt("fixed-app-routes")
}
func (w *deliveryWitness) currentSave() ([]byte, uint64) {
	w.t.Helper()
	snap, label, err := w.front.Snapshot(true)
	w.check(err)
	raw, err := w.front.ExportCurrentSave(snap, label)
	w.check(err)
	return raw, w.front.live.world.Hash()
}

func (w *deliveryWitness) callerRetryRoutes() {
	w.t.Helper()
	oldDraw := w.front.townUI.squareView().RawDraw("wildlife")
	defer func() { w.front.townUI.squareView().SetRawDraw("wildlife", oldDraw) }()
	w.check(w.app.HeadlessFocus(false))
	w.finishOneShots()
	w.holdOneShots = true
	defer func() { w.holdOneShots = false; w.finishOneShots() }()
	scope := w.owner.NewScope()
	defer scope.Destroy()
	var voices []audio.Voice
	for _, member := range chrgenReleaseMembers {
		sample, ok := w.front.SoundBank.NamedSample(member)
		if !ok {
			w.t.Fatal("instrument: retry saturation sample missing", member)
		}
		voice := audio.Dispatch(scope.Player(audio.EffectsChannel), sample,
			audio.FixedRequest("character-detail", member, audio.EffectsChannel, 128, false,
				audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}))
		if voice == nil {
			w.t.Fatal("instrument: retry saturation refused", member)
		}
		voices = append(voices, voice)
	}
	w.check(w.app.HeadlessFocus(true))
	count := func(source string, reason audio.DeliveryReason) int {
		n := 0
		for _, r := range w.owner.Service.Snapshot().Receipts {
			if r.Source == source && r.Reason == reason {
				n++
			}
		}
		return n
	}
	refused := count("town-crowd", audio.DeliveryChannelsBusy)
	if refused == 0 {
		w.t.Fatal("crowd entry did not reach equal-priority shared refusal")
	}
	w.paint(0)
	w.paint(10 * time.Millisecond)
	if count("town-crowd", audio.DeliveryChannelsBusy) != refused {
		w.t.Fatal("crowd retried from ordinary paint after refused entry")
	}
	audio.StopReset(voices[0])
	admitted := count("town-crowd", audio.DeliveryAdmitted)
	w.paint(10 * time.Millisecond)
	if count("town-crowd", audio.DeliveryAdmitted) != admitted {
		w.t.Fatal("crowd refusal became an automatic queue or paint retry")
	}
	w.check(w.app.HeadlessFocus(false))
	w.check(w.app.HeadlessFocus(true))
	if count("town-crowd", audio.DeliveryAdmitted) != admitted+1 {
		w.t.Fatal("crowd did not retry on a fresh entry")
	}
	w.front.townUI.squareView().SetRawDraw("wildlife", func() int { return rawSheet3 })
	w.paint(8 * time.Second)
	w.paint(68 * time.Millisecond)
	if w.front.townUI.sqWildlife().Member("horse").Current != 1 || count("town-horse", audio.DeliveryChannelsBusy) == 0 {
		w.t.Fatal("horse paint did not reach refused A3 frame1", w.front.townUI.sqWildlife().Member("horse"))
	}
	audio.StopReset(voices[1])
	prior := w.owner.Service.Snapshot().Counters.Admitted
	if count("town-horse", audio.DeliveryAdmitted) != 0 {
		w.t.Fatal("horse refusal replayed without caller paint")
	}
	w.paint(10 * time.Millisecond)
	if w.front.townUI.sqWildlife().Member("horse").Current != 1 || count("town-horse", audio.DeliveryAdmitted) != 1 || w.owner.Service.Snapshot().Counters.Admitted != prior+1 {
		w.t.Fatal("horse did not retry at the same paint frame after capacity freed")
	}
	w.receipt("caller-retry")
}

func (w *deliveryWitness) admissionAndFrozenWorld() {
	w.t.Helper()
	f := w.front
	a := w.app
	w.check(a.OpenMission(f.NewGameOpener(20, ui.ChargenResult{Name: "Delivery", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 20, 20}})))
	for n := 0; a.HeadlessNoticeOpen() && n < 32; n++ {
		w.check(a.HeadlessKey("enter"))
	}
	before, hash := w.currentSave()
	scope := w.owner.NewScope()
	defer scope.Destroy()
	centre := audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}
	var voices []audio.Voice
	members := append([]string{}, chrgenReleaseMembers[:11]...)
	members = append(members, "speech/training/npc33s1l1.wav", "speech/training/npc33s2l1.wav", "speech/training/npc33s3l1.wav", "speech/training/npc34s1l1.wav")
	members = append(members, "registry:50")
	w.finishOneShots()
	firstBuffer := len(w.players)
	for _, channel := range w.owner.Service.Snapshot().Channels {
		if channel.Playing {
			w.t.Fatal("instrument: unexpected ongoing map audio before saturation")
		}
	}
	for i, member := range members {
		sample, ok := f.SoundBank.NamedSample(member)
		if i >= 11 && i < 15 {
			sample, ok = f.SpeechBank.Sample(member)
		}
		if i == 15 {
			sample, ok = f.SoundBank.Sample(50)
		}
		if !ok {
			w.t.Fatalf("instrument: missing saturation member %s", member)
		}
		group := audio.EffectsChannel
		source := "character-detail"
		if i >= 11 && i < 15 {
			group = audio.SpeechChannel
			source = "town-response"
		}
		var voice audio.Voice
		if i == 15 {
			cam := f.live.view.Camera()
			geometry := audio.ViewGeometry{Origin: image.Pt(int(cam.X/32), int(cam.Y/32)), Span: image.Pt(int(float64(cam.ViewW)/cam.Zoom/32), int(float64(cam.ViewH)/cam.Zoom/32))}
			request, valid := audio.AmbientRequest("mission-river", member, true, []image.Point{geometry.Origin.Add(geometry.Span.Div(2))}, geometry, false)
			if !valid {
				w.t.Fatal("instrument: saturation ambient geometry invalid")
			}
			voice = ui.RequestAmbient(scope.Ambient(), ui.AmbientRiver, sample, request)
		} else {
			voice = audio.Dispatch(scope.Player(group), sample, audio.FixedRequest(source, member, group, 128, false, centre))
		}
		if voice == nil {
			w.t.Fatalf("mixed admission refused at %d", i)
		}
		voices = append(voices, voice)
	}
	if got := w.owner.Service.Snapshot().Counters.Admitted; got < 16 {
		w.t.Fatal("mixed admission count", got)
	}
	last := w.owner.Service.Snapshot()
	occupied := 0
	for _, ch := range last.Channels {
		if ch.Playing {
			occupied++
		}
	}
	if occupied != 16 {
		w.t.Fatalf("mixed channel count=%d", occupied)
	}
	w.independentStreams()
	w.players[firstBuffer].position = 17 * time.Millisecond
	priorSeek := w.players[firstBuffer].seeks
	sample, ok := f.SoundBank.Sample(100)
	if !ok {
		w.t.Fatal("instrument: missing fixed UI high-priority sample")
	}
	high := audio.Dispatch(scope.Player(audio.EffectsChannel), sample, audio.FixedRequest("fixed-interface", "registry:100", audio.EffectsChannel, 220, false, centre))
	if high == nil || voices[0].Playing() || !voices[1].Playing() {
		w.t.Fatal("strict lower priority did not evict first tied minimum")
	}
	if p := w.players[firstBuffer]; p.position != 0 || p.seeks != priorSeek+1 || p.closes != 0 {
		w.t.Fatal("installed eviction did not stop/rewind/retain first minimum", p)
	}
	if v := audio.Dispatch(scope.Player(audio.EffectsChannel), sample, audio.FixedRequest("fixed-interface", "registry:100", audio.EffectsChannel, 220, false, centre)); v == nil {
		w.t.Fatal("second high request should evict next lower minimum")
	}
	for _, v := range voices {
		audio.StopReset(v)
	}
	audio.StopReset(high)
	scope.Destroy()
	scope = w.owner.NewScope()
	defer scope.Destroy()
	sample, ok = f.SoundBank.NamedSample("chrgen/ok.wav")
	if !ok {
		w.t.Fatal("instrument: missing duplicate sample")
	}
	for i := 0; i < 16; i++ {
		if audio.Dispatch(scope.Player(audio.EffectsChannel), sample, audio.FixedRequest("main-menu", "chrgen/ok.wav", audio.EffectsChannel, 128, false, centre)) == nil {
			w.t.Fatal("duplicate refused before object limit", i)
		}
	}
	prior := w.owner.Service.Snapshot()
	if v := audio.Dispatch(scope.Player(audio.EffectsChannel), sample, audio.FixedRequest("main-menu", "chrgen/ok.wav", audio.EffectsChannel, 128, false, centre)); v != nil {
		w.t.Fatal("sample duplicate cap bypassed")
	}
	next := w.owner.Service.Snapshot()
	tail := next.Receipts[len(next.Receipts)-1]
	if tail.Reason != audio.DeliverySampleBusy || next.Counters.Evicted != prior.Counters.Evicted || next.Counters.BuffersCreated != prior.Counters.BuffersCreated {
		w.t.Fatal("sample-first refusal mutated global channels or buffers", tail)
	}
	after, afterHash := w.currentSave()
	if !bytes.Equal(before, after) || hash != afterHash {
		w.t.Fatal("delivery changed frozen World hash or complete current SAV")
	}
	f.live.world.SetPurse(sim.SelfSlot, f.live.world.Purse(sim.SelfSlot)+1)
	changed, changedHash := w.currentSave()
	if bytes.Equal(before, changed) || changedHash == hash {
		w.t.Fatal("independent live state control did not change both SAV and World")
	}
	proof := map[string]any{"beforeSAV": fmt.Sprintf("%x", sha256.Sum256(before)), "afterSAV": fmt.Sprintf("%x", sha256.Sum256(after)), "changedSAV": fmt.Sprintf("%x", sha256.Sum256(changed)), "World": fmt.Sprintf("%x", hash), "changedWorld": fmt.Sprintf("%x", changedHash), "distinctInstalledObjects": members}
	raw, err := json.MarshalIndent(proof, "", "  ")
	w.check(err)
	w.check(os.WriteFile(filepath.Join(w.dir, "frozen-state.json"), raw, 0600))
	w.receipt("admission")
}

func (w *deliveryWitness) independentStreams() {
	w.t.Helper()
	prior := w.owner.Service.Snapshot()
	track, ok := w.front.MusicBank.Track("town.wav")
	if !ok || w.front.MusicPlayer == nil || w.front.CutsceneAudioPlayer == nil {
		w.t.Fatal("instrument: production independent stream construction missing")
	}
	w.front.MusicPlayer.Start(track)
	_, music, opened := ui.AudioDeviceState(w.front.MusicPlayer)
	if !opened || len(music) != 1 {
		w.t.Fatal("music stream refused under shared saturation")
	}
	sample, ok := w.front.SoundBank.Sample(100)
	if !ok {
		w.t.Fatal("instrument: stream PCM source missing")
	}
	pcm := make([]byte, len(sample.PCM)*2)
	for i, value := range sample.PCM {
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(value))
	}
	w.front.CutsceneAudioPlayer.Start(audio.DeviceRate, 1)
	w.front.CutsceneAudioPlayer.Push(pcm)
	var movie []float64
	for until := time.Now().Add(time.Second); time.Now().Before(until); {
		_, movie, opened = ui.AudioDeviceState(w.front.CutsceneAudioPlayer)
		if opened && len(movie) == 1 {
			break
		}
		runtime.Gosched()
	}
	if !opened || len(movie) != 1 {
		w.t.Fatal("movie stream refused under shared saturation")
	}
	next := w.owner.Service.Snapshot()
	if next.Counters != prior.Counters || len(next.Samples) != len(prior.Samples) {
		w.t.Fatal("music/movie consumed shared samples, channels or admission")
	}
	for _, ch := range next.Channels {
		if !ch.Playing {
			w.t.Fatal("independent stream displaced a shared channel", ch)
		}
	}
	proof, err := json.MarshalIndent(map[string]any{"sharedChannels": len(next.Channels), "before": prior.Counters, "after": next.Counters,
		"musicGains": music, "movieGains": movie, "musicMember": "town.wav", "movieControl": "production streaming device fed installed decoded PCM; no original movie decoder or audibility claim"}, "", "  ")
	w.check(err)
	w.check(os.WriteFile(filepath.Join(w.dir, "independent-streams.json"), proof, 0600))
	w.front.MusicPlayer.Stop()
	w.front.CutsceneAudioPlayer.Stop()
}

func (w *deliveryWitness) installedAmbientRoutes() {
	w.t.Helper()
	f, a := w.front, w.app
	w.check(a.HeadlessKey("0"))
	m := f.live.mission.state.Map
	words := terrain.RenderTileWords(m.Tiles)
	var water, crows []image.Point
	eligibleBirds := 0
	for i, word := range words {
		cell := image.Pt(i%m.Width, i/m.Width)
		if terrain.Resolve(word).Water {
			water = append(water, cell)
		}
		code := m.Overlay[i]
		class := f.Statics.Classes[code]
		if code == 0 || class == nil {
			continue
		}
		if word&0x2000 != 0 && class.Dead != nil {
			class = class.Dead
		}
		if class.FireObject == -2 {
			crows = append(crows, cell)
		}
		if class.FireObject >= 0 {
			eligibleBirds++
		}
	}
	if len(water) == 0 || len(crows) == 0 {
		w.t.Fatal("instrument: installed mission20 has no water/crow source", len(water), len(crows))
	}
	centre := func(cell image.Point) {
		f.live.view.Camera().CenterOn(float64(cell.X)*camera.CellSize, float64(cell.Y)*camera.CellSize)
		w.check(a.HeadlessPointer("hover", 320, 120))
	}
	hash := f.live.world.Hash()
	centre(water[len(water)/2])
	w.requireSource("mission-river")
	centre(crows[len(crows)/2])
	for n := 0; w.sources()["mission-crow"] == 0 && n < 1400; n++ {
		w.check(a.HeadlessStep())
		w.finishOneShots()
	}
	w.requireSource("mission-crow")
	if eligibleBirds != 0 {
		w.t.Fatal("instrument: installed nonnegative FireObject source population changed", eligibleBirds)
	}
	if f.live.world.Hash() != hash {
		w.t.Fatal("paused ambient delivery mutated World")
	}
	proof, err := json.MarshalIndent(map[string]any{"mission": 20, "address": f.live.mission.state.Address, "waterCells": len(water), "crowCells": len(crows), "eligibleBirdCells": eligibleBirds,
		"birdQualification": "no installed nonnegative FireObject source in this mission; asset-free selector control covers that branch", "pausedWorld": fmt.Sprintf("%x", hash)}, "", "  ")
	w.check(err)
	w.check(os.WriteFile(filepath.Join(w.dir, "installed-ambient-population.json"), proof, 0600))
	w.receipt("installed-ambient")
	w.installedLoopRetention()
}

func (w *deliveryWitness) installedLoopRetention() {
	w.t.Helper()
	state := w.owner.BackendState()
	var id uint64
	for _, b := range state.Buffers {
		if b.Repeat && b.Playing {
			id = b.ID
			break
		}
	}
	if id == 0 {
		w.t.Fatal("instrument: no installed river loop for phase control")
	}
	p := w.players[id-1]
	p.position = 17 * time.Millisecond
	seeks := p.seeks
	st, _, ok := ui.SoundDeviceState(w.front.SoundPlayer)
	if !ok {
		w.t.Fatal("instrument: shared settings seam missing")
	}
	w.front.live.view.Camera().Pan(32, 0)
	w.check(w.app.HeadlessPointer("hover", 320, 120))
	w.owner.SetSettings(audio.EffectsChannel, audio.Settings{Master: st.Master / 2})
	next := w.owner.BackendState()
	found, moved := false, false
	for _, b := range next.Buffers {
		if b.ID == id {
			found = true
			if b.Phase != int64(17*time.Millisecond)*audio.DeviceRate/int64(time.Second) || !b.Playing {
				w.t.Fatal("loop phase changed during move/settings", b)
			}
		}
	}
	for _, r := range next.Receipts {
		moved = moved || r.Buffer == id && r.Action == "move"
	}
	if !found || !moved || next.Created != state.Created || p.seeks != seeks || p.closes != 0 {
		w.t.Fatal("installed loop move/settings replaced, rewound or destroyed buffer", next)
	}
	w.owner.SetSettings(audio.EffectsChannel, st)
	w.receipt("installed-loop-retention")
}

func (w *deliveryWitness) installedFireCast() {
	w.t.Helper()
	f, a := w.front, w.app
	party := MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
	if len(party) != 1 || !party[0].Mage {
		w.t.Fatal("instrument: installed mage construction failed")
	}
	w.check(a.OpenMission(f.MissionOpenerWith(41, party)))
	a.Layout(1024, 768)
	for n := 0; a.HeadlessNoticeOpen() && n < 32; n++ {
		w.check(a.HeadlessKey("enter"))
	}
	releasePauseMission(w.t, f, a)
	id := f.live.mission.ids[0]
	e, _ := f.live.entity(id)
	w.t.Logf("installed caster: class=%d charge=%d delay=%d weaponSpell=%d", e.Class, e.AttackCharge, f.live.sounds[e.Class].AttackDelay, e.WeaponSpell)
	for _, actor := range f.live.world.Entities() {
		if actor.KnownSpells != 0 {
			w.t.Logf("caster population id=%d owner=%d class=%d known=%x charge=%d delay=%d", actor.ID, actor.Owner, actor.Class, actor.KnownSpells, actor.AttackCharge, f.live.sounds[actor.Class].AttackDelay)
		}
	}
	book := e.Book
	book.State = sim.BookPresent
	for _, spell := range []uint32{2, 3, 21} {
		rule, ok := f.live.world.Spell(spell)
		if !ok {
			w.t.Fatal("instrument: installed spell missing", spell)
		}
		book.Slots[spell-1] = sim.BookSpell{Range: rule.MaxRange, ManaCost: uint16(rule.ManaCost)}
	}
	w.check(f.live.world.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{ID: id, KnownSpells: e.KnownSpells | 1<<2 | 1<<3 | 1<<21, Book: book}}))
	f.live.push()
	at, ok := cityMageNearest(cityMageCastCells(w.t, f, a, id, 3))
	if !ok {
		w.t.Fatal("instrument: no visible admitted Wall of Fire cell")
	}
	cityMageCastInput(w.t, f, a, id, 3, at)
	w.check(a.HeadlessPointer("right-press", 512, 200))
	w.check(a.HeadlessPointer("right-release", 512, 200))
	activePauseResume(w.t, a)
	for n := 0; w.sources()["mission-fire"] == 0 && n < 240; n++ {
		w.check(a.HeadlessStep())
		if len(wallFireAmbientCells(f.live.world.CellEffects())) != 0 {
			f.live.view.Camera().Pan(32, 0)
			w.check(a.HeadlessPointer("hover", 512, 200))
		}
		w.finishOneShots()
	}
	w.receipt("installed-fire-cast")
	w.requireSource("mission-fire")
	if len(wallFireAmbientCells(f.live.world.CellEffects())) == 0 {
		w.t.Fatal("installed cast did not create canonical Wall of Fire coverage")
	}
	for _, spell := range []uint32{2, 21} {
		releasePauseMission(w.t, f, a)
		current, _ := f.live.entity(id)
		w.check(f.live.world.ImportOriginalActorPools([]sim.OriginalActorPools{{ID: id, HP: current.HP, MaxHP: current.MaxHP, Mana: 1000, MaxMana: 1000}}))
		f.live.push()
		at, ok := cityMageNearest(cityMageCastCells(w.t, f, a, id, spell))
		if !ok {
			w.t.Fatal("instrument: no visible admitted spell cell", spell)
		}
		cityMageCastInput(w.t, f, a, id, spell, at)
		w.check(a.HeadlessPointer("right-press", 512, 200))
		w.check(a.HeadlessPointer("right-release", 512, 200))
		activePauseResume(w.t, a)
		for n := 0; n < 160; n++ {
			w.check(a.HeadlessStep())
			w.finishOneShots()
		}
	}
	w.requireSource("book-cast-direct")
	w.requireSource("deferred-spell-effect")
	w.requireSource("storm-phase")
	w.receipt("installed-spell-casts")
	w.producerObservationControls(id)
}

func (w *deliveryWitness) producerObservationControls(id sim.EntityID) {
	w.t.Helper()
	f, a := w.front, w.app
	releasePauseMission(w.t, f, a)
	mw := f.live
	e, ok := mw.entity(id)
	if !ok || !e.Alive() {
		w.t.Fatal("instrument: observation-control caster unavailable")
	}
	// This controller control retains installed class selectors and PCM. Its
	// authored charge makes the existing delayed hook reachable; mission 41's
	// three ordinary mages have charge 8 equal to their class delay 8.
	e.AttackCharge = 16
	e.HasAttackTarget = false
	e.Facing = e.DesiredFacing
	e.TurnRemaining, e.TurnTotal = 0, 0
	target := e
	target.ID, target.Owner, target.X = id+1, 2, e.X+4
	target.HasAttackTarget = false
	worn, _ := mw.world.EquippedItems(id)
	probe, err := sim.NewStructuredWorld(1, mw.world.Bounds(), sim.ModeCanonical, sim.Terrain{}, []sim.Entity{e, target}, nil,
		mw.world.Relations(), nil, []sim.Stock{{ID: id, EquippedItems: worn}, {ID: target.ID, EquippedItems: worn}}, mw.world.Spells(), sim.GhostTemplate{}, nil)
	w.check(err)
	mw.world = probe
	mw.castRun = make(map[sim.EntityID]castRun)
	mw.spellSoundCues = nil
	mw.push()
	event := sim.CastEvent{Caster: id, Spell: 2, Owner: e.Owner, FromX: e.X, FromY: e.Y, ToX: e.X + 4, ToY: e.Y}
	mw.observeCasts([]sim.CastEvent{event})
	for n := 0; n < 12; n++ {
		mw.advanceSwings()
		mw.advanceSpellSoundCues()
		mw.advanceCastRuns()
		w.check(a.HeadlessPointer("hover", 512, 200))
		w.finishOneShots()
	}
	w.requireSource("book-cast-animation")
	mw.castRun = make(map[sim.EntityID]castRun)
	w.check(probe.ImportOriginalActorActions([]sim.OriginalActorAction{{Entity: id, HasTarget: true, Target: target.ID, Phase: sim.AttackCasting, Countdown: 20}}))
	for n := 0; n < 12; n++ {
		mw.advanceSwings()
		w.check(a.HeadlessPointer("hover", 512, 200))
		w.finishOneShots()
	}
	w.requireSource("weapon-cast-animation")
	event.Weapon = true
	mw.observeCasts([]sim.CastEvent{event})
	w.requireSource("weapon-cast-direct")
	mw.observeScriptCasts([]sim.ScriptCastEvent{{Spell: 2, FromX: e.X, FromY: e.Y, ToX: e.X + 4, ToY: e.Y}})
	w.requireSource("scripted-cast")
	mw.observeAreaPaints([]sim.AreaPaint{{Spell: 4, Owner: e.Owner, Cells: []sim.CellPoint{{X: e.X, Y: e.Y}}}})
	w.requireSource("spell-effect")
	scope := w.owner.NewScope()
	defer scope.Destroy()
	cam := mw.view.Camera()
	geometry := audio.ViewGeometry{Origin: image.Pt(int(cam.X/32), int(cam.Y/32)), Span: image.Pt(int(float64(cam.ViewW)/cam.Zoom/32), int(float64(cam.ViewH)/cam.Zoom/32))}
	request, valid := audio.AmbientRequest("mission-bird", "registry:60", false, []image.Point{image.Pt(int(e.X), int(e.Y))}, geometry, true)
	sample, found := f.SoundBank.Sample(60)
	if !valid || !found || audio.Dispatch(scope.Player(audio.EffectsChannel), sample, request) == nil {
		w.t.Fatal("instrument: installed bird recipe boundary control failed")
	}
	w.requireSource("mission-bird")
	proof, err := json.MarshalIndent(map[string]any{"qualification": "installed selectors and PCM through production observer/App/backend; authored one-actor charge16 observation control, not native or default mission trigger evidence", "mission41MageCharge": 8, "mission41MageClassDelay": 8, "controllerCharge": 16}, "", "  ")
	w.check(err)
	w.check(os.WriteFile(filepath.Join(w.dir, "producer-observation-qualification.json"), proof, 0600))
	w.receipt("producer-observation-controls")
}

func TestReleaseSharedAudioDelivery(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("set AGAINROM_ASSETS to a lawful install")
	}
	out := os.Getenv("AGAINROM_SFX_DELIVERY_WITNESS_DIR")
	if !filepath.IsAbs(out) || !filepath.IsAbs(root) {
		t.Fatal("instrument: AGAINROM_SFX_DELIVERY_WITNESS_DIR must be an absolute existing directory")
	}
	out, err := editorPhysicalDirectory(out)
	if err != nil {
		t.Fatal("instrument:", err)
	}
	install, err := editorPhysicalDirectory(root)
	if err != nil {
		t.Fatal("instrument:", err)
	}
	rel, err := filepath.Rel(install, out)
	if err != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatal("instrument: witness output is inside the install")
	}
	for _, name := range []string{"controlled", "physical"} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(out, filepath.Base(install), name)
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			w := newDeliveryWitness(t, dir, name == "controlled")
			if name == "controlled" {
				w.fixedAppRoutes()
				w.admissionAndFrozenWorld()
				w.installedAmbientRoutes()
				w.installedFireCast()
				w.installedCombatRoutes()
			} else {
				sample, ok := w.front.SoundBank.Sample(100)
				if !ok {
					t.Fatal("instrument: physical sample missing")
				}
				scope := w.owner.NewScope()
				voice := audio.Dispatch(scope.Player(audio.EffectsChannel), sample, audio.FixedRequest("fixed-interface", "registry:100", audio.EffectsChannel, 220, false, audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}))
				if voice == nil || !voice.Playing() {
					t.Fatal("instrument: physically muted construction did not dispatch")
				}
				for _, buffer := range w.owner.BackendState().Buffers {
					if !buffer.Physical || buffer.DeviceGain != 0 {
						t.Fatal("instrument: physical backend is not muted", buffer)
					}
				}
				w.receipt("physical-dispatch")
				scope.Destroy()
			}
			w.app.StopAudio()
			state := w.owner.BackendState()
			if len(state.Buffers) != 0 || state.Created != state.Destroyed {
				t.Fatal("process shutdown leaked retained physical buffers", state)
			}
			w.receipt("shutdown")
		})
	}
}
