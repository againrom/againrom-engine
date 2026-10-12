package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// speakerSexHero is one of the four starting heroes the generator offers.
type speakerSexHero struct {
	name         string
	female, mage bool
	choiceSex    int
	choiceClass  int
}

var speakerSexHeroes = []speakerSexHero{
	{name: "male fighter", choiceSex: 0, choiceClass: 0},
	{name: "female fighter", female: true, choiceSex: 1, choiceClass: 0},
	{name: "male mage", mage: true, choiceSex: 0, choiceClass: 1},
	{name: "female mage", female: true, mage: true, choiceSex: 1, choiceClass: 1},
}

// speakerSexJoins is the mission from which each persistent companion stands
// with the party (docs/1046-joined-heroes/spec.md FR1): npc22 from the
// mission-30 town, npc25 Brian, npc23, npc24 and npc26 Rood from the mission
// whose map hands them over.
var speakerSexJoins = []struct {
	npc     int32
	mission int
}{{22, 30}, {25, 40}, {23, 70}, {24, 100}, {26, 140}}

// speakerSexLine is one event part that names a hero speaker.
type speakerSexLine struct {
	mission, event, part int
	speaker              int
	variant              string // "female", "male" or "" for a part with no speaker sex arm
}

// speakerSexAudience builds the audience a party is read for.
type speakerSexAudience func(party []mapload.PartyMember, faces map[int32]data.NPCFace) EventAudience

func productionSpeakerSexAudience(party []mapload.PartyMember, faces map[int32]data.NPCFace) EventAudience {
	return speakerAudience(HeroAudience(party), faces)
}

// speakerSexParty is the starting hero the generator makes for h, with every
// companion joined by mission m, each built by the production constructors.
func speakerSexParty(t *testing.T, f *FrontEnd, h speakerSexHero, m int) []mapload.PartyMember {
	t.Helper()
	party := f.ChargenParty(ui.ChargenResult{Name: "Hero", Choices: []int{h.choiceSex, h.choiceClass, 0}})
	if len(party) != 1 || !party[0].StartingHero || data.FigureDir(party[0].FigureDir).Female() != h.female || party[0].Mage != h.mage {
		t.Fatalf("%s: generator party = %+v", h.name, party)
	}
	for _, j := range speakerSexJoins {
		if m < j.mission {
			continue
		}
		member, ok := mapload.CampaignNPCMember(f.Table, j.npc, j.mission, party)
		if !ok {
			t.Fatalf("%s: companion npc%d did not build", h.name, j.npc)
		}
		party = append(party, member)
	}
	return party
}

// speakerSexMissions is every first-game mission number with at least one
// event text, and the event numbers that ship for it.
func speakerSexMissions(f *FrontEnd) map[int][]int {
	out := map[int][]int{}
	for m := 1; m <= 200; m++ {
		for e := 1; e <= 99; e++ {
			if _, ok := ReadEventText(f.Archives.Containers, m, e); ok {
				out[m] = append(out[m], e)
			}
		}
	}
	return out
}

// speakerVariant is the speaker sex a selected tag is authored for: the bare
// `female` or `male` arm, after the hero's own `iam*` literals are cut out.
func speakerVariant(tag string) string {
	bare := withoutHeroArms(tag)
	switch {
	case containsFold(bare, "female"):
		return "female"
	case containsFold(bare, "male"):
		return "male"
	}
	return ""
}

func sexWord(female bool) string {
	if female {
		return "female"
	}
	return "male"
}

// speakerSexCensus reads every part of every event text of every mission for
// one starting hero. A failure is a hero speaker resolving no member or not
// its own character, a speaker, text variant or portrait of another sex than
// the member's, or a part the hero arms admit that the speaker arms close.
func speakerSexCensus(t *testing.T, f *FrontEnd, faces map[int32]data.NPCFace, missions map[int][]int,
	h speakerSexHero, build speakerSexAudience, portraits bool) (lines []speakerSexLine, failures []string, identity []string) {
	t.Helper()
	numbers := make([]int, 0, len(missions))
	for m := range missions {
		numbers = append(numbers, m)
	}
	sort.Ints(numbers)
	figures := map[figureID]*image.RGBA{}
	for _, m := range numbers {
		party := speakerSexParty(t, f, h, m)
		ids := make([]sim.EntityID, len(party))
		for i := range ids {
			ids[i] = sim.EntityID(i + 1)
		}
		cast := speakerCast{actors: missionSpeakers(nil, nil, nil, party, ids, nil),
			alive: func(sim.EntityID) bool { return true }}
		cast.playerDir, _ = memberFigure(party[0])
		cast.hasPlayer = true
		r := speakerResolver{src: f.Archives.Containers, npcFaces: faces, cast: cast}
		aud := build(party, faces)
		heroOnly := speakerAudience(HeroAudience(party), nil)
		for _, e := range missions[m] {
			payload, _ := ReadEventText(f.Archives.Containers, m, e)
			for n := 1; n < 64; n++ {
				if _, ok := dialoguePart(payload, n, aud); !ok {
					if _, heroSees := dialoguePart(payload, n, heroOnly); heroSees {
						failures = append(failures, fmt.Sprintf("%s m%d e%02d part %d: absent", h.name, m, e, n))
					}
					continue
				}
				speaker, named := EventPartSpeaker(payload, n, aud)
				if !named || speaker < 21 || speaker > 26 {
					continue
				}
				tag, _, _ := eventPartTail(dialoguePayload(payload), n, aud)
				line := speakerSexLine{mission: m, event: e, part: n, speaker: speaker, variant: speakerVariant(tag)}
				lines = append(lines, line)
				where := fmt.Sprintf("%s m%d e%02d part %d npc%d", h.name, m, e, n, speaker)
				rec, ok := faces[int32(speaker)]
				if !ok {
					failures = append(failures, where+": no npc record")
					continue
				}
				actor, live := cast.resolveNPC(int32(speaker), rec)
				if !live {
					failures = append(failures, where+": resolves no party member")
					continue
				}
				member := party[int(actor.id)-1]
				if want := speakerSexWant(speaker, member); want != "" {
					identity = append(identity, where+": "+want)
					failures = append(failures, where+": "+want)
				}
				memberFemale := actor.fig.Dir.Female()
				if female, _, resolved := aud.Speaker(speaker); resolved && female != memberFemale {
					failures = append(failures, fmt.Sprintf("%s: speaker sex %s, member %s", where, sexWord(female), actor.fig.Dir))
				}
				if line.variant != "" && line.variant != sexWord(memberFemale) {
					failures = append(failures, fmt.Sprintf("%s: %s text spoken by %s %s", where, line.variant, actor.fig.Dir, member.ID))
				}
				if !portraits {
					continue
				}
				pic, _, shown := r.SpeakerFace(speaker)
				fig := figureID{Dir: actor.fig.Dir, Face: actor.fig.Face, Hero: actor.hero}
				want, cached := figures[fig]
				if !cached {
					want, _ = composeUnitFigure(f.Archives.Containers, data.Equipment{}, fig)
					figures[fig] = want
				}
				if !shown || pic == nil || want == nil || pic.Bounds() != want.Bounds() || !bytes.Equal(pic.Pix, want.Pix) {
					failures = append(failures, where+": portrait is not the resolved member's figure")
				}
			}
		}
	}
	return lines, failures, identity
}

// nestedEventTags is the first game's tag rule with the speaker arms testing
// the whole body, `iam*` literals included: the loss control for the cut.
type nestedEventTags struct{ firstEventTags }

func (nestedEventTags) accepts(a EventAudience, tag string) bool {
	for _, arm := range eventArms {
		if !containsFold(tag, arm.lit) || arm.guard != "" && containsFold(tag, arm.guard) {
			continue
		}
		have := a.HeroFemale
		if !arm.sex {
			have = a.HeroMage
		}
		if arm.speaker {
			female, mage, gate := a.speakerFacts(tag)
			if !gate {
				continue
			}
			if have = female; !arm.sex {
				have = mage
			}
		}
		if have != arm.want {
			return false
		}
	}
	return true
}

// speakerSexWant names a resolved member that is not the record's own
// character: the starting hero for npc21, the companion carrying the number
// otherwise.
func speakerSexWant(speaker int, member mapload.PartyMember) string {
	if speaker == 21 {
		if member.StartingHero {
			return ""
		}
		return "npc21 resolved " + member.ID
	}
	if member.CompanionNPC == speaker {
		return ""
	}
	return fmt.Sprintf("resolved %s (companion %d)", member.ID, member.CompanionNPC)
}

// TestReleaseHeroSpeakersSpeakInTheirOwnSex is the speaker-sex census: every
// first-game event part that names a hero speaker (npc21 to npc26), for each
// of the four starting heroes, resolves a member of that hero's party, and the
// member's sex is the speaker's, the text variant's and the portrait's.
func TestReleaseHeroSpeakersSpeakInTheirOwnSex(t *testing.T) {
	f := releaseFront(t)
	faces := LoadNPCFaces(f.Archives.Containers)
	missions := speakerSexMissions(f)
	if len(missions) == 0 || faces == nil {
		t.Fatal("install has no event texts or no npc registry")
	}
	total, tagged := 0, 0
	for _, h := range speakerSexHeroes {
		lines, failures, identity := speakerSexCensus(t, f, faces, missions, h, productionSpeakerSexAudience, true)
		for _, msg := range failures {
			t.Error(msg)
		}
		perSpeaker := map[int]int{}
		variants := 0
		for _, l := range lines {
			perSpeaker[l.speaker]++
			if l.variant != "" {
				variants++
			}
		}
		total += len(lines)
		tagged += variants
		if h == speakerSexHeroes[0] {
			speakerSexCoverage(t, lines)
		}
		t.Logf("%s: %d hero-speaker lines (npc21..26: %d %d %d %d %d %d), %d with a speaker sex variant, %d resolved to another member of the same sex",
			h.name, len(lines), perSpeaker[21], perSpeaker[22], perSpeaker[23], perSpeaker[24], perSpeaker[25], perSpeaker[26], variants, len(identity))
		for _, msg := range identity {
			t.Log("  identity: " + msg)
		}
	}
	t.Logf("%d missions with event texts; %d lines over four heroes, %d with a speaker sex variant", len(missions), total, tagged)
	if tagged == 0 {
		t.Fatal("census reached no speaker sex variant")
	}

	// Loss control 1: the speaker gate on the Start key alone, the resolver
	// before this fix. It must fail at the owner's line: mission 111 event06
	// part 2 spoken by a male starting hero in the female variant.
	keyOnly := func(party []mapload.PartyMember, faces map[int32]data.NPCFace) EventAudience {
		stripped := make(map[int32]data.NPCFace, len(faces))
		for id, rec := range faces {
			rec.Archetype = false
			stripped[id] = rec
		}
		return speakerAudience(HeroAudience(party), stripped)
	}
	// Loss control 2: a resolver that ignores the starting hero's sex.
	sexless := func(party []mapload.PartyMember, faces map[int32]data.NPCFace) EventAudience {
		aud := speakerAudience(HeroAudience(party), faces)
		inner := speakerAudience(EventAudience{HeroMage: aud.HeroMage}, faces)
		aud.Speaker = inner.Speaker
		return aud
	}
	// Loss control 3: the speaker arms also read the `iam*` literals, which
	// closes the companion's `iamfemale`/`iammale` pairs.
	nested := func(party []mapload.PartyMember, faces map[int32]data.NPCFace) EventAudience {
		aud := speakerAudience(HeroAudience(party), faces)
		aud.tags = nestedEventTags{}
		return aud
	}
	for _, control := range []struct {
		name  string
		build speakerSexAudience
		hero  speakerSexHero
		must  string
	}{
		{"Start key only", keyOnly, speakerSexHeroes[2], "male mage m111 e06 part 2 npc21: female text"},
		{"Start key only", keyOnly, speakerSexHeroes[0], "male fighter m111 e06 part 2 npc21: female text"},
		{"hero sex ignored", sexless, speakerSexHeroes[1], "female fighter m111 e06 part 2 npc21: male text"},
		{"iam literals read by the speaker arms", nested, speakerSexHeroes[0], "male fighter m130 e01 part 2: absent"},
	} {
		_, failures, _ := speakerSexCensus(t, f, faces, missions, control.hero, control.build, false)
		hit := false
		for _, msg := range failures {
			hit = hit || strings.HasPrefix(msg, control.must)
		}
		if !hit {
			t.Errorf("loss control %q for %s did not fail at %q (%d failures)", control.name, control.hero.name, control.must, len(failures))
		}
		t.Logf("loss control %q, %s: %d failures", control.name, control.hero.name, len(failures))
	}
}

// speakerSexCoverage logs each mission's parts and speakers; * marks a variant.
func speakerSexCoverage(t *testing.T, lines []speakerSexLine) {
	t.Helper()
	byMission := map[int][]string{}
	var order []int
	for _, l := range lines {
		if _, seen := byMission[l.mission]; !seen {
			order = append(order, l.mission)
		}
		mark := ""
		if l.variant != "" {
			mark = "*"
		}
		byMission[l.mission] = append(byMission[l.mission], fmt.Sprintf("e%02d.%d:npc%d%s", l.event, l.part, l.speaker, mark))
	}
	for _, m := range order {
		t.Logf("  m%d: %s", m, strings.Join(byMission[m], " "))
	}
}

// TestReleaseOwnerCatapultLineSpeaksInTheSpeakersSex loads the owner's mission
// 111 save, a male mage starting hero with Naira in the party, and opens
// event06 through the production dialogue handler: part 1 is Naira's female
// variant with her portrait, part 2 the hero's male variant with his.
func TestReleaseOwnerCatapultLineSpeaksInTheSpeakersSex(t *testing.T) {
	path := os.Getenv("AGAINROM_SPEAKER_SEX_SAV")
	if path == "" {
		t.Skip("AGAINROM_SPEAKER_SEX_SAV is not set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if after, err := os.ReadFile(path); err != nil || !bytes.Equal(raw, after) {
			t.Error("owner save bytes changed", err)
		}
	})
	f := shopOrderFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "owner-mission111.sav")
	t.Cleanup(app.StopAudio)
	live := f.live
	if f.liveMission != 111 || live == nil {
		t.Fatal("owner save did not load mission 111", f.liveMission)
	}
	dir, has := live.playerFigureDir()
	if !has || dir != data.FigureDirManMage {
		t.Fatal("owner save starting hero is not a male mage", dir, has)
	}
	payload, ok := ReadEventText(f.Archives.Containers, 111, 6)
	if !ok {
		t.Fatal("mission 111 has no event06")
	}
	want := map[int]struct {
		speaker int
		variant string
		dir     data.FigureDir
	}{1: {23, "female", data.FigureDirWomanFighter}, 2: {21, "male", data.FigureDirManMage}}
	before := live.world.Hash()
	if !live.openDialogue(6) {
		t.Fatal("event06 did not open")
	}
	for part := 1; part <= 2; part++ {
		w := want[part]
		body, kind, open := live.view.NoticeState()
		tag, _, _ := eventPartTail(dialoguePayload(payload), part, live.mission.dialogueAudience)
		expect, _ := dialoguePart(payload, part, live.mission.dialogueAudience)
		speaker, named := EventPartSpeaker(payload, part, live.mission.dialogueAudience)
		if !open || kind != ui.NoticeDialogue || live.mission.part != part || !named || speaker != w.speaker || speakerVariant(tag) != w.variant || body != expect {
			t.Fatalf("part %d: open=%v speaker=%d tag=%q, want npc%d %s", part, open, speaker, tag, w.speaker, w.variant)
		}
		if independent := speakerSexIndependentTag(t, payload, part, w.variant); independent != tag {
			t.Fatalf("part %d: shown tag %q, the installed %s variant is %q", part, tag, w.variant, independent)
		}
		cast := speakerCast{actors: live.speakerActors, alive: live.entityAlive, worn: live.equipmentOf, playerDir: dir, hasPlayer: true}
		actor, found := cast.resolveNPC(int32(speaker), live.npcFaces[int32(speaker)])
		if !found || actor.fig.Dir != w.dir {
			t.Fatalf("part %d: speaker resolves %v %s, want %s", part, found, actor.fig.Dir, w.dir)
		}
		pic, _, shown := live.SpeakerFace(speaker)
		portrait, face := live.view.NoticeSpeaker()
		if !shown || !portrait || face == nil || !bytes.Equal(face.Pix, pic.Pix) {
			t.Fatalf("part %d: window portrait is not the resolved speaker's", part)
		}
		t.Logf("part %d: npc%d %s variant, speaker figure %s", part, speaker, w.variant, actor.fig.Dir)
		live.advanceNotice()
	}
	if live.view.NoticeOpen() || live.world.Hash() != before {
		t.Fatal("event06 did not close after two parts or changed the world")
	}
}

// speakerSexIndependentTag finds part n's tag of the given speaker variant by
// a plain scan of the installed bytes, independent of the audience.
func speakerSexIndependentTag(t *testing.T, payload []byte, part int, variant string) string {
	t.Helper()
	s := string(dialoguePayload(payload))
	for {
		open := strings.IndexByte(s, '<')
		if open < 0 {
			break
		}
		end := strings.IndexByte(s[open:], '>')
		if end < 0 {
			break
		}
		tag := s[open+1 : open+end]
		s = s[open+end+1:]
		lower := strings.ToLower(strings.ReplaceAll(tag, " ", ""))
		has := strings.Contains(lower, ","+variant) && (variant == "female" || !strings.Contains(lower, "female"))
		if strings.Contains(lower, fmt.Sprintf("part=%d,", part)) && has {
			return tag
		}
	}
	t.Fatalf("installed event06 has no part %d %s variant", part, variant)
	return ""
}
