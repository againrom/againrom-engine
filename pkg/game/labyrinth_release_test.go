package game

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const (
	// labyrinthMission is the side mission of chapter 30 titled "The
	// Labyrinth" in the EN install and "Лабиринт" in the RU install. It
	// places three lost villagers, seventeen squirrels, nine goblins and
	// one goblin slinger.
	labyrinthMission = 31
	labyrinthChapter = 30
	// maceBearer is the hired Human ClubMan, mercenary type 14.
	maceBearer = 14
)

// labyrinthFront opens the mission with the generated hero and three hired
// mace-bearers, whom the tavern of chapter 30 offers.
func labyrinthFront(t *testing.T) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	c := f.Campaign.Value()
	town := NewTown(c)
	for mission := range c.Chapters {
		if mission < labyrinthChapter {
			town.Won(mission)
		}
	}
	town.gold = 1_000_000
	f.Town = town
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Labyrinth", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}})
	f.arriveInTown()
	f.Town.mercEnabled[maceBearer] = true
	s := f.TownScreen().(*townScreen)
	s.room = roomTavern
	s.composeShopFaces()
	for range 3 {
		if msg, ok := s.toggleMercenary(maceBearer); !ok {
			t.Fatalf("hiring mercenary type %d in chapter %d refused: %s", maceBearer, labyrinthChapter, msg)
		}
	}
	app := f.App("labyrinth")
	app.SetCutscenes(nil)
	if err := app.OpenMission(f.MissionOpenerWith(labyrinthMission, f.Carried)); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	return f, app
}

// labyrinthMercenaries are the hired mace-bearers of the mission's party.
func labyrinthMercenaries(t *testing.T, f *FrontEnd) []sim.EntityID {
	t.Helper()
	var ids []sim.EntityID
	for i, member := range f.live.mission.party {
		if i < len(f.live.mission.ids) && member.Hired() {
			ids = append(ids, f.live.mission.ids[i])
		}
	}
	if len(ids) != 3 {
		t.Fatalf("the party holds %d hired mace-bearers, want 3", len(ids))
	}
	return ids
}

// labyrinthFrame is one ordinary frame. A dialogue takes the player's Enter as
// a frame of its own; a notice that ends the mission is an error.
func labyrinthFrame(f *FrontEnd, app *ui.App) error {
	if text, kind, up := f.LiveNotice(); up {
		if kind != ui.NoticeDialogue {
			return fmt.Errorf("the mission ended with the notice %q (kind %d)", text, kind)
		}
		return app.HeadlessActivate("notice")
	}
	return app.HeadlessStep()
}

// labyrinthLive is the first living entity of ids, if any.
func labyrinthLive(f *FrontEnd, ids []sim.EntityID) (sim.Entity, bool) {
	for _, id := range ids {
		if e, ok := f.live.entity(id); ok && e.Alive() {
			return e, true
		}
	}
	return sim.Entity{}, false
}

// labyrinthBeside reports whether creature id stands within one cell of a living
// mace-bearer.
func labyrinthBeside(f *FrontEnd, mercs []sim.EntityID, id sim.EntityID) bool {
	e, ok := f.live.entity(id)
	if !ok || !e.Alive() {
		return false
	}
	for _, mid := range mercs {
		if m, ok := f.live.entity(mid); ok && m.Alive() && max(releaseAbs32(e.X-m.X), releaseAbs32(e.Y-m.Y)) <= 1 {
			return true
		}
	}
	return false
}

// labyrinthStriker is the hostile creature nearest lead that stands within one
// cell of a mace-bearer it holds as its victim.
func labyrinthStriker(f *FrontEnd, mercs []sim.EntityID, lead sim.Entity) (sim.Entity, bool) {
	var striker sim.Entity
	distance := int32(-1)
	for _, e := range f.live.world.Entities() {
		if !e.Alive() || !e.HasAttackTarget || !slices.Contains(mercs, e.AttackTarget) ||
			!f.live.world.Relations().Hostile(lead.Owner, e.Owner) {
			continue
		}
		if m, ok := f.live.entity(e.AttackTarget); !ok || !m.Alive() || max(releaseAbs32(e.X-m.X), releaseAbs32(e.Y-m.Y)) > 1 {
			continue
		}
		if d := (e.X-lead.X)*(e.X-lead.X) + (e.Y-lead.Y)*(e.Y-lead.Y); distance < 0 || d < distance {
			striker, distance = e, d
		}
	}
	return striker, distance >= 0
}

// Each visible creature fall requests Sound[4] on its fall frame.
func TestReleaseClassVoicedFallsPlaySoundFourInTheLabyrinth(t *testing.T) {
	f, app := labyrinthFront(t)
	mercs := labyrinthMercenaries(t, f)
	heard := observeReleaseAudio(t, f)
	alive := map[sim.EntityID]bool{}
	for _, e := range f.live.world.Entities() {
		alive[e.ID] = e.Alive()
	}
	type fall struct {
		frame    int
		class    int32
		slot     int
		eligible bool
	}
	var falls []fall
	eligibleClass := map[int32]bool{}
	var target sim.Entity
	since, hostile := 0, 0
	for heard.frame = 1; heard.frame <= 16000; heard.frame++ {
		lead, ok := labyrinthLive(f, mercs)
		if !ok {
			t.Fatalf("frame %d: every mace-bearer fell before each creature class generated an eligible fall request; falls (frame, class, slot, eligible): %v", heard.frame, falls)
		}
		victim, found := f.live.entity(target.ID)
		fresh := target.ID == 0 || !found || !victim.Alive()
		answered := false
		if !fresh && !labyrinthBeside(f, mercs, target.ID) {
			if striker, ok := labyrinthStriker(f, mercs, lead); ok {
				target, since, answered = striker, heard.frame, true
			}
		}
		if fresh || answered || (heard.frame-since)%200 == 0 {
			if fresh {
				var nearest sim.Entity
				distance := int32(-1)
				hostile = 0
				for _, e := range f.live.world.Entities() {
					if !e.Alive() || !f.live.world.Relations().Hostile(lead.Owner, e.Owner) {
						continue
					}
					hostile++
					if eligibleClass[f.live.spellClientClass(e.ID, e.Class)] {
						continue
					}
					if d := (e.X-lead.X)*(e.X-lead.X) + (e.Y-lead.Y)*(e.Y-lead.Y); distance < 0 || d < distance {
						nearest, distance = e, d
					}
				}
				if distance < 0 {
					break
				}
				target, since = nearest, heard.frame
			}
			for _, id := range mercs {
				if m, _ := f.live.entity(id); m.Alive() {
					f.live.strike(uint32(id), uint32(target.ID))
				}
			}
		}
		if err := labyrinthFrame(f, app); err != nil {
			t.Fatalf("frame %d: %v", heard.frame, err)
		}
		for _, e := range f.live.world.Entities() {
			if alive[e.ID] && !e.Alive() && e.Owner != sim.SelfSlot {
				class := f.live.spellClientClass(e.ID, e.Class)
				slots := soundSlots(f.live.sounds, class)
				if len(slots) < 5 || slots[4] == 0 {
					t.Fatalf("entity %d, drawn as class %d, has no Sound[4]: %v", e.ID, class, slots)
				}
				falls = append(falls, fall{heard.frame, class, slots[4], hurtVoiceRequestEligible(f, e)})
				eligibleClass[class] = eligibleClass[class] || falls[len(falls)-1].eligible
			}
			alive[e.ID] = e.Alive()
		}
		if eligibleClass[64] && eligibleClass[74] && eligibleClass[79] {
			break
		}
	}
	if hostile == 0 && len(falls) == 0 {
		t.Fatalf("no hostile creature stood in the mission")
	}
	var want, got []hurtVoicePlay
	slotOf := map[int32]int{}
	slots := map[string]bool{}
	classes := map[int32][2]int{}
	for _, fl := range falls {
		slotOf[fl.class] = fl.slot
		slots["slot "+strconv.Itoa(fl.slot)] = true
		n := classes[fl.class]
		n[0]++
		if fl.eligible {
			n[1]++
			want = append(want, hurtVoicePlay{fl.frame, "slot " + strconv.Itoa(fl.slot)})
		}
		classes[fl.class] = n
	}
	for _, p := range heard.plays {
		if slots[p.src] {
			got = append(got, p)
		}
	}
	byFrame := func(a, b hurtVoicePlay) int {
		if a.frame != b.frame {
			return a.frame - b.frame
		}
		return strings.Compare(a.src, b.src)
	}
	slices.SortFunc(want, byFrame)
	slices.SortFunc(got, byFrame)
	for _, class := range []int32{64, 74, 79} {
		n, ok := classes[class]
		t.Logf("drawn class %d: %d fell, %d eligible requests, index 4 = slot %d of Sound %v",
			class, n[0], n[1], slotOf[class], soundSlots(f.live.sounds, class))
		if !ok || n[1] == 0 {
			t.Errorf("class %d: %d falls, %d eligible requests, want at least one", class, n[0], n[1])
		}
	}
	if len(classes) != 3 {
		t.Errorf("%d creature classes fell, want the mission's 3: %v", len(classes), classes)
	}
	if !slices.Equal(got, want) {
		t.Errorf("requests of each class's Sound[4]: %v\nwant one on each eligible fall frame: %v", got, want)
	}
	t.Logf("%d falls over %d frames, tick %d", len(falls), heard.frame-1, f.live.world.Tick())
}

// labyrinthRecordings are the select and command recordings of the eight human
// voice banks, by name, each decoded and not silent.
func labyrinthRecordings(t *testing.T, f *FrontEnd) map[string]audio.Sample {
	t.Helper()
	installed := map[string]audio.Sample{}
	for _, banks := range humanVoiceBanks {
		for _, bank := range banks {
			for _, leaf := range []string{"select1", "select2", "command1", "command2", "command3"} {
				name := bank + "/" + leaf + ".wav"
				sample, ok := f.SoundBank.namedSample(name)
				if !ok || !slices.ContainsFunc(sample.PCM, func(v int16) bool { return v != 0 }) {
					t.Fatalf("%s absent or silent", name)
				}
				installed[name] = sample
			}
		}
	}
	return installed
}

// labyrinthReplied names the recordings requested from the speech device
// since mark. A sample that several recordings share names all of them, so a
// bank the check cannot tell apart from another fails it.
func labyrinthReplied(installed map[string]audio.Sample, voices *acknowledgmentRecorder, mark int) []string {
	var out []string
	for _, s := range voices.samples[mark:] {
		var names []string
		for name, in := range installed {
			if slices.Equal(in.PCM, s.PCM) {
				names = append(names, name)
			}
		}
		slices.Sort(names)
		out = append(out, strings.Join(names, "|"))
	}
	return out
}

// labyrinthRescue walks the mace-bearers to villager id with ordinary move
// orders, restarted every 200 frames, until the villager joins the player and
// the dialogue the meeting opens is answered.
func labyrinthRescue(t *testing.T, f *FrontEnd, app *ui.App, mercs []sim.EntityID, id sim.EntityID) {
	t.Helper()
	for n := 0; n < 4000; n++ {
		v, _ := f.live.entity(id)
		if v.Owner == sim.SelfSlot {
			for ; n < 4000; n++ {
				if _, _, up := f.LiveNotice(); !up {
					return
				}
				if err := labyrinthFrame(f, app); err != nil {
					t.Fatalf("villager %d, frame %d: %v", id, n, err)
				}
			}
			t.Fatalf("villager %d: a notice stayed open", id)
		}
		if n%200 == 0 {
			for _, m := range mercs {
				if e, _ := f.live.entity(m); e.Alive() {
					f.live.enqueue(uint32(m), int(v.X), int(v.Y))
				}
			}
		}
		if err := labyrinthFrame(f, app); err != nil {
			t.Fatalf("villager %d, frame %d: %v", id, n, err)
		}
	}
	t.Fatalf("villager %d did not join the player within 4000 frames", id)
}

// TestReleaseRescuedVillagersReplyAndHurtInThePeasantBank rescues the lost
// villager the mace-bearers reach first, then plays it as the player would.
// The mission places three villagers with no weapon, so every one takes the
// peasant bank for its wounds. The rescued one, selected with a click and then
// ordered, replies from that bank and not from the mace-bearers' mercenary
// bank, and a mace-bearer selected beside it still replies from the mercenary
// bank. Ordered into a fight it is wounded: each wound and its fall must sound
// the peasant leaves of the same bank, wherever the camera lets them be heard.
func TestReleaseRescuedVillagersReplyAndHurtInThePeasantBank(t *testing.T) {
	script := &scriptedDraw{}
	previous := newViewerVoiceDraw
	newViewerVoiceDraw = func() voiceDraw { return script.draw }
	t.Cleanup(func() { newViewerVoiceDraw = previous })
	f, app := labyrinthFront(t)
	mercs := labyrinthMercenaries(t, f)
	voices := &acknowledgmentRecorder{}
	f.SpeechPlayer = voices
	installed := labyrinthRecordings(t, f)
	var villagers []sim.EntityID
	for _, e := range f.live.world.Entities() {
		if e.Humanoid && e.Owner != sim.SelfSlot {
			villagers = append(villagers, e.ID)
		}
	}
	if len(villagers) != 3 {
		t.Fatalf("the mission places %d people beside the party, want 3 villagers", len(villagers))
	}
	for _, d := range f.live.entityDraws() {
		if slices.Contains(villagers, sim.EntityID(d.ID)) && d.Voice != "m_peasant" {
			t.Errorf("villager %d wounds sound from %q, want m_peasant", d.ID, d.Voice)
		}
	}
	id := villagers[0]
	labyrinthRescue(t, f, app, mercs, id)
	// The joined villager guards on its own and walks into the monsters; the
	// player's Stand Ground keeps it where the clicks below expect it. The
	// monsters already fighting the party are removed: their groups choose
	// the nearest foe, the lowest id on a tie, so the villager beside the
	// mace-bearers may be struck while the clicks run, and a blow within
	// 1500 ms before the fight below would silence its first wound there.
	for _, e := range f.live.world.Entities() {
		if !e.Alive() || e.Owner == sim.SelfSlot || !e.HasAttackTarget || e.AttackTargetKind != sim.AttackTargetUnit {
			continue
		}
		if victim, ok := f.live.entity(e.AttackTarget); ok && victim.Owner == sim.SelfSlot {
			f.live.pending = append(f.live.pending, sim.Kill(e.ID))
		}
	}
	f.live.pending = append(f.live.pending, sim.GroupStance(id, sim.OrderStandGround, 0))
	for range 8 {
		if err := labyrinthFrame(f, app); err != nil {
			t.Fatal(err)
		}
	}
	v, _ := f.live.entity(id)
	x, y, err := app.HeadlessMinimapPoint(int(v.X), int(v.Y))
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	clock := func() {
		t.Helper()
		if err := headlessReplyClock(app); err != nil {
			t.Fatal(err)
		}
	}
	expect := func(what string, mark int, want string) {
		t.Helper()
		got := labyrinthReplied(installed, voices, mark)
		t.Logf("%s: %v", what, got)
		if !slices.Equal(got, []string{want}) {
			t.Errorf("%s requested %v, want %s", what, got, want)
		}
	}
	mark := len(voices.samples)
	script.set(0, 0)
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	expect("clicking the villager", mark, "m_peasant/select1.wav")
	clock()
	mark = len(voices.samples)
	script.set(0, 16384)
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	expect("clicking the villager again", mark, "m_peasant/select2.wav")
	clock()
	x, y, err = app.HeadlessGroundPoint()
	if err != nil {
		t.Fatal(err)
	}
	mark = len(voices.samples)
	script.set(0, 0)
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	expect("ordering the villager", mark, "m_peasant/command1.wav")
	clock()
	var picked sim.EntityID
	mark = len(voices.samples)
	script.set(0, 0)
	for _, m := range mercs {
		if err := app.HeadlessSelectEntity(uint32(m)); err == nil {
			picked = m
			break
		}
	}
	if picked == 0 {
		t.Fatal("no mace-bearer stands in the camera's view")
	}
	expect(fmt.Sprintf("clicking mace-bearer %d", picked), mark, "mf_merc/select1.wav")
	subject := &hurtVoiceSubject{name: "the rescued villager", id: id, bank: "m_peasant", attacks: true}
	watched, heard := hurtVoiceReleaseFight(t, f, app, []*hurtVoiceSubject{subject}, 900)
	t.Logf("villager %d, tick %d", id, f.live.world.Tick())
	hurtVoiceCheck(t, f, heard, watched)
}
