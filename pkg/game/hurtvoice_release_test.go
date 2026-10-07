package game

import (
	"fmt"
	"image"
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type hurtVoiceHeard struct {
	*SoundBank
	frame              int
	plays              []hurtVoicePlay
	requests           []audio.Request
	recipients         [][]sim.EntityID
	t                  *testing.T
	front              *FrontEnd
	scope              *ui.AudioScope
	refused            map[audio.DeliveryReason]int
	farRequests        int
	minimumAttenuation int
}

func (h *hurtVoiceHeard) Play(audio.Sample, audio.Placement) {
	h.t.Helper()
	h.t.Fatal("legacy Play bypassed the typed shared-service observer")
}

func (h *hurtVoiceHeard) RequestSample(s audio.Sample, request audio.Request) audio.Voice {
	h.t.Helper()
	var recipients []sim.EntityID
	if request.VolumeTerm == "distance+setting" {
		cam := h.front.live.view.Camera()
		geometry := audio.ViewGeometry{Origin: image.Pt(int(math.Floor(cam.X/32)), int(math.Floor(cam.Y/32))),
			Span: image.Pt(int(float64(cam.ViewW)/cam.Zoom/32), int(float64(cam.ViewH)/cam.Zoom/32))}
		dx := float64(request.SourceFine.X) - float64(geometry.Origin.X*256+geometry.Span.X*128)
		dy := float64(request.SourceFine.Y) - float64(geometry.Origin.Y*256+geometry.Span.Y*128)
		if math.Hypot(dx, dy) >= 40*256 {
			h.farRequests++
		}
		attenuation := int(math.Max(-10000, -(math.Exp(math.Hypot(dx, dy)/2048)-1)*100))
		h.minimumAttenuation = min(h.minimumAttenuation, attenuation)
		pan := int(math.Trunc(dx * 4000 / float64(geometry.Span.X*256)))
		pan = max(-10000, min(10000, pan))
		gain := int64(math.Pow(10, float64(attenuation)/2000) * 10000)
		placement := audio.Placement{Left: int(min(10000, gain*int64(10000-pan)/10000)),
			Right: int(min(10000, gain*int64(10000+pan)/10000))}
		if request.Geometry != geometry || request.Attenuation != attenuation || request.Pan != pan || request.Placement != placement ||
			request.Priority != uint8((10000-absAudioTerm(attenuation))/100) || !request.SpatialKnown {
			h.t.Fatalf("positional request differs from independent camera/fine-coordinate terms: %+v; geometry=%+v attenuation=%d pan=%d placement=%+v", request, geometry, attenuation, pan, placement)
		}
		for _, e := range h.front.live.world.Entities() {
			x, y, ok := h.front.live.world.ActorFinePosition(e.ID)
			fine := image.Pt(int(e.X)*256+128, int(e.Y)*256+128)
			if ok {
				fine = image.Pt(int(e.X)*256+int(x), int(e.Y)*256+int(y))
			}
			if fine == request.SourceFine {
				recipients = append(recipients, e.ID)
			}
		}
		if len(recipients) == 0 {
			h.t.Fatalf("sound fine coordinate %v belongs to no current actor", request.SourceFine)
		}
	}
	h.requests = append(h.requests, request)
	h.recipients = append(h.recipients, recipients)
	name := strings.TrimPrefix(request.Selector, "voice:")
	if strings.HasPrefix(request.Selector, "registry:") {
		name = "slot " + strings.TrimPrefix(request.Selector, "registry:")
	}
	h.plays = append(h.plays, hurtVoicePlay{h.frame, name})
	before := h.scope.Owner().Service.Snapshot()
	var sequence uint64
	for _, receipt := range before.Receipts {
		sequence = max(sequence, receipt.Sequence)
	}
	voice := audio.Dispatch(h.scope.Player(request.Group), s, request)
	snapshot := h.scope.Owner().Service.Snapshot()
	if len(snapshot.Receipts) == 0 {
		h.t.Fatal("typed request produced no shared-service receipt")
	}
	var receipt audio.DeliveryReceipt
	for _, r := range snapshot.Receipts {
		if r.Sequence <= sequence || r.Source != request.Source || r.Selector != request.Selector || r.Group != request.Group {
			continue
		}
		switch r.Reason {
		case audio.DeliveryAdmitted, audio.DeliverySampleBusy, audio.DeliveryChannelsBusy,
			audio.DeliverySampleUnavailable, audio.DeliveryDeviceUnavailable, audio.DeliveryBufferUnavailable, audio.DeliveryInvalidGroup:
			receipt = r
		}
	}
	if receipt.Sequence == 0 || receipt.Source != request.Source || receipt.Selector != request.Selector || receipt.Group != request.Group ||
		(voice != nil) != (receipt.Reason == audio.DeliveryAdmitted) {
		h.t.Fatalf("typed request and shared admission differ: request=%+v receipt=%+v voice=%v", request, receipt, voice)
	}
	if voice == nil {
		h.refused[receipt.Reason]++
	}
	return voice
}

func releaseAudioFromActor(t *testing.T, heard *hurtVoiceHeard, index int, id sim.EntityID) bool {
	t.Helper()
	if !slices.Contains(heard.recipients[index], id) {
		return false
	}
	if len(heard.recipients[index]) != 1 {
		t.Fatalf("target sound coordinate is shared by actors %v; cannot identify actor %d", heard.recipients[index], id)
	}
	return true
}

func absAudioTerm(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func observeReleaseAudio(t *testing.T, f *FrontEnd) *hurtVoiceHeard {
	t.Helper()
	scope := ui.NewAudioScope(f.SoundPlayer)
	if scope == nil {
		t.Fatal("release observer has no shared audio owner")
	}
	h := &hurtVoiceHeard{SoundBank: f.SoundBank, t: t, front: f, scope: scope, refused: map[audio.DeliveryReason]int{}}
	f.live.view.SetAudio(h, h)
	f.live.view.SetSpeechAudio(h)
	t.Cleanup(func() {
		t.Logf("typed sound requests=%d; distance >= 40 cells=%d; minimum attenuation=%d; shared-service admission refusals=%v", len(h.requests), h.farRequests, h.minimumAttenuation, h.refused)
		scope.Destroy()
	})
	return h
}

// hurtVoiceSubject is one actor watched through a release fight: the class it
// is drawn as when the fight starts, and after every frame its health,
// maximum, blow count, life and request eligibility. A subject with after set stands until that
// subject has taken a blow, so that the one who walks out first is the one the
// hostiles strike first.
type hurtVoiceSubject struct {
	name     string
	id       sim.EntityID
	bank     string
	attacks  bool
	after    *hurtVoiceSubject
	ordered  bool
	class    int32
	own      bool
	hp       []int32
	max      []int32
	blows    []uint32
	alive    []bool
	eligible []bool
}

func (s *hurtVoiceSubject) record(f *FrontEnd) {
	e, _ := f.live.entity(s.id)
	if len(s.hp) == 0 {
		s.class = f.live.spellClientClass(e.ID, e.Class)
	}
	s.own = e.Owner == sim.SelfSlot
	s.hp, s.max = append(s.hp, e.HP), append(s.max, e.MaxHP)
	s.blows, s.alive = append(s.blows, f.live.blows[s.id]), append(s.alive, e.Alive())
	s.eligible = append(s.eligible, hurtVoiceRequestEligible(f, e))
}

func hurtVoiceRequestEligible(f *FrontEnd, e sim.Entity) bool {
	if fog := f.live.fog; e.Owner != sim.SelfSlot {
		if i := int(e.Y)*fog.cols + int(e.X); i < 0 || i >= len(fog.visible) || fog.visible[i] == 0 {
			return false
		}
	}
	c := f.live.view.Camera()
	return c != nil && c.Zoom > 0 && int(float64(c.ViewW)/c.Zoom/32) > 0 && int(float64(c.ViewH)/c.Zoom/32) > 0
}

// want is what the subject must voice frame by frame (ANIM-094, ANIM-095),
// and how many wounds and falls were outside the request gates: a blow takes the
// voice timestamp and sounds the easy leaf while the health held before it is
// at least half the maximum and the hard leaf below that, unless that health
// is -10 or less or the timestamp was taken fewer than 75 frames, 1500 ms,
// before, as does a fallen body of the player's that loses a point with no blow;
// the fall requests the die leaf and takes no timestamp. A gated request
// still takes the timestamp. Shared-service admission is measured separately.
func (s *hurtVoiceSubject) want() ([]hurtVoicePlay, int) {
	var out []hurtVoicePlay
	last, unheard := -1, 0
	voice := func(f int, leaf string) {
		if s.eligible[f] {
			out = append(out, hurtVoicePlay{f, s.bank + "/" + leaf + ".wav"})
		} else {
			unheard++
		}
	}
	for f := 1; f < len(s.hp); f++ {
		stored := s.hp[f-1]
		if s.blows[f] > s.blows[f-1] && stored > -10 && (last < 0 || f-last >= 75) {
			leaf := "easy"
			if stored < s.max[f-1]/2 {
				leaf = "hard"
			}
			voice(f, leaf)
			last = f
		} else if s.own && s.blows[f] == s.blows[f-1] && stored < 0 && stored >= -9 && s.hp[f] < stored && (last < 0 || f-last >= 75) {
			// A fallen body of the player's that loses a point with no blow
			// bleeds, through the same gate (ANIM-126).
			voice(f, "hard")
			last = f
		}
		if s.alive[f-1] && !s.alive[f] {
			voice(f, "die")
		}
	}
	return out, unheard
}

// hurtVoiceNearestHostile is the living hostile of from's domain nearest it.
func hurtVoiceNearestHostile(f *FrontEnd, from sim.Entity) (sim.Entity, bool) {
	var best sim.Entity
	distance := int32(-1)
	for _, e := range f.live.world.Entities() {
		dx, dy := e.X-from.X, e.Y-from.Y
		if e.Alive() && e.Domain == from.Domain && f.live.world.Relations().Hostile(from.Owner, e.Owner) &&
			(distance < 0 || dx*dx+dy*dy < distance) {
			best, distance = e, dx*dx+dy*dy
		}
	}
	return best, distance >= 0
}

// hurtVoiceReleaseFight gives the player's attack order to every attacking
// subject against the hostile nearest the first, and a new order as each
// target falls, for frames frames or until the mission's outcome shows. A
// subject waiting on another takes its first order once that one is struck. A
// dialogue notice is dismissed with the player's Enter, which is a frame of
// its own. Every other actor that voices from a subject's bank is watched
// too, so that each bank's plays are all accounted for. The viewer's sounds
// go to the returned recorder, and every watched actor is recorded after each
// frame.
func hurtVoiceReleaseFight(t *testing.T, f *FrontEnd, app *ui.App, subjects []*hurtVoiceSubject, frames int) ([]*hurtVoiceSubject, *hurtVoiceHeard) {
	t.Helper()
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	banks := map[string]bool{}
	for _, s := range subjects {
		banks[s.bank] = true
	}
	for _, e := range f.live.world.Entities() {
		if bank := f.live.voiceBank(e); banks[bank] && !slices.ContainsFunc(subjects, func(s *hurtVoiceSubject) bool { return s.id == e.ID }) {
			subjects = append(subjects, &hurtVoiceSubject{name: fmt.Sprintf("actor %d", e.ID), id: e.ID, bank: bank})
		}
	}
	heard := observeReleaseAudio(t, f)
	for _, s := range subjects {
		s.record(f)
	}
	var target sim.Entity
	for heard.frame = 1; heard.frame <= frames; heard.frame++ {
		if _, kind, up := f.LiveNotice(); up && kind != ui.NoticeDialogue {
			break
		} else if up {
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
		} else {
			lead, _ := f.live.entity(subjects[0].id)
			renewed := false
			if victim, ok := f.live.entity(target.ID); target.ID == 0 || !ok || !victim.Alive() {
				next, ok := hurtVoiceNearestHostile(f, lead)
				if !ok {
					break
				}
				target, renewed = next, true
			}
			for _, s := range subjects {
				if !s.attacks || (s.after != nil && !s.ordered && f.live.blows[s.after.id] == 0) {
					continue
				}
				if renewed || !s.ordered {
					f.live.strike(uint32(s.id), uint32(target.ID))
					s.ordered = true
				}
			}
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		for _, s := range subjects {
			s.record(f)
		}
	}
	return subjects, heard
}

// hurtVoiceCheck compares, bank by bank, what the watched actors voiced with
// what they must voice, and fails on any play of the old route: a subject's
// drawn class's wound or death sample. Actors sharing a bank are compared
// together, each frame's plays in name order.
func hurtVoiceCheck(t *testing.T, f *FrontEnd, heard *hurtVoiceHeard, watched []*hurtVoiceSubject) {
	t.Helper()
	byFrame := func(a, b hurtVoicePlay) int {
		if a.frame != b.frame {
			return a.frame - b.frame
		}
		return strings.Compare(a.src, b.src)
	}
	want := map[string][]hurtVoicePlay{}
	for _, s := range watched {
		mine, unheard := s.want()
		want[s.bank] = append(want[s.bank], mine...)
		t.Logf("%s (%s, drawn as class %d with Sound %v): health %d..%d of %d over %d frames, %d blows; generated %v, %d outside request gates",
			s.name, s.bank, s.class, soundSlots(f.live.sounds, s.class), s.hp[0], slices.Min(s.hp), s.max[0],
			len(s.hp)-1, s.blows[len(s.blows)-1]-s.blows[0], mine, unheard)
		if s.attacks && len(mine) == 0 {
			t.Errorf("%s generated no eligible wound request", s.name)
		}
		if slots := soundSlots(f.live.sounds, s.class); len(slots) > 2 {
			for _, slot := range slots[2:] {
				for _, p := range heard.plays {
					if p.src == "slot "+strconv.Itoa(slot) {
						t.Errorf("%s: frame %d played %s, the drawn class's own wound or death sample", s.name, p.frame, p.src)
					}
				}
			}
		}
	}
	for bank, mine := range want {
		var got []hurtVoicePlay
		for _, p := range heard.plays {
			if strings.HasPrefix(p.src, bank+"/") {
				got = append(got, p)
			}
		}
		slices.SortFunc(got, byFrame)
		slices.SortFunc(mine, byFrame)
		if !slices.Equal(got, mine) {
			t.Errorf("%s: heard %v\nwant %v", bank, got, mine)
		}
	}
}

// TestReleaseHurtVoiceFollowsSexAndClass fights mission 20 with each generated
// hero through the player's attack orders. Danas holds a bow in one run and a
// blade in the other; Naira holds a bow. Each hero's wounds and fall must
// sound his own bank's leaves, whatever he wields.
func TestReleaseHurtVoiceFollowsSexAndClass(t *testing.T) {
	for _, hero := range []struct {
		name    string
		choices []int
		bank    string
	}{
		{"Danas with a bow", []int{0, 0, 4}, "mf_hero"},
		{"Danas with a blade", []int{0, 0, 0}, "mf_hero"},
		{"Naira with a bow", []int{1, 0, 4}, "ff_hero"},
		{"Fergard", []int{0, 1, 0}, "m_mage"},
		{"Reniesta", []int{1, 1, 0}, "f_mage"},
	} {
		t.Run(hero.name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			party := f.ChargenParty(ui.ChargenResult{Name: "Hurt", Choices: hero.choices, Stats: []int{31, 27, 24, 29}})
			app := f.App("hurt voice")
			app.SetCutscenes(nil)
			if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
				t.Fatal(err)
			}
			app.Layout(1024, 768)
			id := f.live.mission.ids[0]
			e, _ := f.live.entity(id)
			if got := f.live.voiceBank(e); got != hero.bank {
				t.Fatalf("%s voices from %q, want %q", hero.name, got, hero.bank)
			}
			subject := &hurtVoiceSubject{name: hero.name, id: id, bank: hero.bank, attacks: true}
			watched, heard := hurtVoiceReleaseFight(t, f, app, []*hurtVoiceSubject{subject}, 1200)
			t.Logf("tick %d", f.live.world.Tick())
			hurtVoiceCheck(t, f, heard, watched)
		})
	}
}

// TestReleaseHurtVoiceOfAHiredHealer hires the healer, mercenary type 4, in
// the first chapter whose tavern offers her, beside Danas with a bow, and
// fights that chapter's mission through the player's attack orders. Her
// wounds sound the female mage bank and his the male hero bank. She is ordered
// first and he once she is struck: the hostiles strike the party member
// nearest them, and a hero who walks out beside her takes every blow until he
// falls, which ends the mission before she is struck at all.
func TestReleaseHurtVoiceOfAHiredHealer(t *testing.T) {
	const healer = 4
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	c := f.Campaign.Value()
	target := hurtVoiceHealerChapter(t, c, healer)
	town := NewTown(c)
	for mission := range c.Chapters {
		if mission < target {
			town.Won(mission)
		}
	}
	town.gold = 1_000_000
	f.Town = town
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Hurt", Choices: []int{0, 0, 4}, Stats: []int{31, 27, 24, 29}})
	f.arriveInTown()
	f.Town.mercEnabled[healer] = true
	s := f.TownScreen().(*townScreen)
	s.room = roomTavern
	s.composeShopFaces()
	if msg, ok := s.toggleMercenary(healer); !ok {
		t.Fatalf("hire type %d in chapter %d refused: %s", healer, target, msg)
	}
	app := f.App("hurt voice healer")
	app.SetCutscenes(nil)
	if err := app.OpenMission(f.MissionOpenerWith(target, f.Carried)); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	var danas, healers []*hurtVoiceSubject
	for i, member := range f.live.mission.party {
		if i >= len(f.live.mission.ids) {
			break
		}
		id := f.live.mission.ids[i]
		e, _ := f.live.entity(id)
		var s *hurtVoiceSubject
		switch {
		case !member.Hired() && len(danas) == 0:
			s = &hurtVoiceSubject{name: "Danas with a bow", id: id, bank: "mf_hero", attacks: true}
			danas = append(danas, s)
		case member.MercenaryType == healer:
			s = &hurtVoiceSubject{name: fmt.Sprintf("healer %s #%d", member.Name, len(healers)+1), id: id, bank: "f_mage", attacks: true}
			healers = append(healers, s)
		default:
			continue
		}
		if got := f.live.voiceBank(e); got != s.bank {
			t.Fatalf("%s voices from %q, want %q", s.name, got, s.bank)
		}
	}
	if len(danas) != 1 || len(healers) == 0 {
		t.Fatalf("mission %d carries %d heroes and %d healers", target, len(danas), len(healers))
	}
	danas[0].after = healers[0]
	subjects := append(healers, danas...)
	watched, heard := hurtVoiceReleaseFight(t, f, app, subjects, 1200)
	t.Logf("mission %d, tick %d", target, f.live.world.Tick())
	hurtVoiceCheck(t, f, heard, watched)
}

// hurtVoiceHealerChapter is the first main chapter whose tavern offers
// mercenary type typ, stocked and unlocked by an earlier chapter.
func hurtVoiceHealerChapter(t *testing.T, c Campaign, typ int) int {
	t.Helper()
	for _, target := range c.Main {
		if !c.offers(target) || !slices.Contains(c.Chapters[target].Mercenaries, typ) ||
			typ > len(c.MercenaryCount) || c.MercenaryCount[typ-1] <= 0 {
			continue
		}
		for mission, ch := range c.Chapters {
			if mission < target && slices.Contains(ch.EnableMercenary, typ) {
				return target
			}
		}
	}
	t.Fatalf("no chapter offers mercenary type %d", typ)
	return 0
}
