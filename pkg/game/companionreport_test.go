package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const companionReportFlags = "Hero,!Mage,!MySex,Start"

func companionReportRegistry(flags string, definition int32) (*data.NPCDefs, map[int32]data.NPCFace) {
	children := []*reg.Node{{Name: "Flags", Type: reg.TypeString, Str: flags}}
	if definition >= 0 {
		children = append(children, &reg.Node{Name: "DataBinID", Type: reg.TypeInt, Int: definition})
	}
	r := &reg.Reg{Root: &reg.Node{Dir: true, Children: []*reg.Node{{Name: "npc23", Dir: true, Children: children}}}}
	return data.LoadNPCDefs(r), data.LoadNPCFaces(r)
}

func companionReportCast(primary data.FigureDir) speakerCast {
	opposite := data.FigureDirFor(false, !primary.Female())
	return speakerCast{
		playerDir: primary, hasPlayer: true,
		alive: func(sim.EntityID) bool { return true },
		actors: []speakerActor{
			{id: 10, fig: figureID{Dir: opposite}},
			{id: 11, hero: true, fig: figureID{Dir: data.FigureDirFor(true, !primary.Female())}},
			{id: 12, hero: true, fig: figureID{Dir: data.FigureDirFor(false, primary.Female())}},
			{id: 13, hero: true, fig: figureID{Dir: opposite}},
		},
	}
}

func assertCompanionReportPart(t *testing.T, aud EventAudience, part int, want string) {
	t.Helper()
	payload := []byte("<npc=23,part=3,male>he-three<npc=23,part=3,female>she-three<npc=23,part=9,male>he-nine<npc=23,part=9,female>she-nine")
	got, ok := EventPart(payload, part, aud)
	if !ok || got != want {
		t.Fatalf("part %d = %q/%v, want %q", part, got, ok, want)
	}
	if speaker, named := EventPartSpeaker(payload, part, aud); !named || speaker != 23 {
		t.Fatalf("part %d speaker = %d/%v, want npc23", part, speaker, named)
	}
}

func TestCompanionReportAudienceUsesTheLivingOppositeSexFighter(t *testing.T) {
	for _, tc := range []struct {
		name                string
		dir                 data.FigureDir
		female              bool
		wantThree, wantNine string
	}{
		{"male fighter", data.FigureDirManFighter, true, "she-three", "she-nine"},
		{"male mage", data.FigureDirManMage, true, "she-three", "she-nine"},
		{"female fighter", data.FigureDirWomanFighter, false, "he-three", "he-nine"},
		{"female mage", data.FigureDirWomanMage, false, "he-three", "he-nine"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defs, faces := companionReportRegistry(companionReportFlags, 26)
			cast := companionReportCast(tc.dir)
			actor, found := cast.resolve(faces[23])
			if !found || actor.id != 13 {
				t.Fatalf("fixture live actor = %d/%v, want 13", actor.id, found)
			}
			base := speakerAudience(heroAud(tc.dir.Female(), tc.dir.Mage()), faces)
			aud := companionReportAudience(base, 70, 2, defs, faces, cast)
			if female, mage, resolved := aud.Speaker(23); !resolved || female != tc.female || mage {
				t.Fatalf("speaker axes = %v/%v/%v, want %v/false/true", female, mage, resolved, tc.female)
			}
			assertCompanionReportPart(t, aud, 3, tc.wantThree)
			assertCompanionReportPart(t, aud, 9, tc.wantNine)
			if _, _, resolved := base.Speaker(23); resolved || faces[23].Start {
				t.Fatal("override mutated the generic audience or separate Start gate")
			}
		})
	}
}

type companionReportInputs struct {
	mission, event int
	defs           *data.NPCDefs
	faces          map[int32]data.NPCFace
	cast           speakerCast
}

func TestCompanionReportAudienceRejectsInputsOutsideItsBound(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*companionReportInputs)
	}{
		{"other mission", func(i *companionReportInputs) { i.mission = 71 }},
		{"other event", func(i *companionReportInputs) { i.event = 3 }},
		{"nil definitions", func(i *companionReportInputs) { i.defs = nil }},
		{"direct definition", func(i *companionReportInputs) { i.defs, _ = companionReportRegistry(companionReportFlags, 31) }},
		{"absent definition", func(i *companionReportInputs) { i.defs, _ = companionReportRegistry(companionReportFlags, -1) }},
		{"composed mage", func(i *companionReportInputs) { i.defs, _ = companionReportRegistry("Hero,Mage,!MySex,Start", 26) }},
		{"composed same sex", func(i *companionReportInputs) { i.defs, _ = companionReportRegistry("Hero,!Mage,MySex,Start", 26) }},
		{"missing record", func(i *companionReportInputs) { delete(i.faces, 23) }},
		{"nil records", func(i *companionReportInputs) { i.faces = nil }},
		{"portrait record", func(i *companionReportInputs) { r := i.faces[23]; r.Kind = data.NPCPortrait; i.faces[23] = r }},
		{"no Start token", func(i *companionReportInputs) { r := i.faces[23]; r.Archetype = false; i.faces[23] = r }},
		{"separate Start key", func(i *companionReportInputs) { r := i.faces[23]; r.Start = true; i.faces[23] = r }},
		{"missing Hero token", func(i *companionReportInputs) {
			r := i.faces[23]
			r.Tokens &^= data.NPCTokens(data.NPCTokenHero)
			i.faces[23] = r
		}},
		{"missing class token", func(i *companionReportInputs) {
			r := i.faces[23]
			r.Tokens &^= data.NPCTokens(data.NPCTokenNotMage)
			i.faces[23] = r
		}},
		{"missing sex token", func(i *companionReportInputs) {
			r := i.faces[23]
			r.Tokens &^= data.NPCTokens(data.NPCTokenNotMySex)
			i.faces[23] = r
		}},
		{"extra predicate", func(i *companionReportInputs) {
			r := i.faces[23]
			r.Tokens |= data.NPCTokens(data.NPCTokenMe)
			i.faces[23] = r
		}},
		{"missing liveness", func(i *companionReportInputs) { i.cast.alive = nil }},
		{"missing actors", func(i *companionReportInputs) { i.cast.actors = nil }},
		{"dead matching actor", func(i *companionReportInputs) { i.cast.alive = func(id sim.EntityID) bool { return id != 13 } }},
		{"missing primary", func(i *companionReportInputs) { i.cast.hasPlayer = false }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defs, faces := companionReportRegistry(companionReportFlags, 26)
			i := companionReportInputs{70, 2, defs, faces, companionReportCast(data.FigureDirManFighter)}
			tc.change(&i)
			base := EventAudience{Speaker: func(int) (bool, bool, bool) { return false, true, true }}
			aud := companionReportAudience(base, i.mission, i.event, i.defs, i.faces, i.cast)
			assertCompanionReportPart(t, aud, 3, "he-three")
			assertCompanionReportPart(t, aud, 9, "he-nine")
			if female, mage, resolved := aud.Speaker(23); female || !mage || !resolved {
				t.Fatalf("fallback changed the base closure: %v/%v/%v", female, mage, resolved)
			}
		})
	}
}

func TestCompanionReportAudienceKeepsStartingHeroAndGenericHeroArmsSeparate(t *testing.T) {
	defs, faces := companionReportRegistry(companionReportFlags, 26)
	primary := mapload.PartyMember{ID: "primary", StartingHero: true, PlayerCharacter: true, Mage: true, FigureDir: string(data.FigureDirManMage)}
	companion := mapload.PartyMember{ID: "companion", PlayerCharacter: true, FigureDir: string(data.FigureDirWomanFighter)}
	for _, reordered := range []bool{false, true} {
		party, ids := []mapload.PartyMember{primary, companion}, []sim.EntityID{12, 13}
		wantHero := "male-mage"
		if reordered {
			party, ids = []mapload.PartyMember{companion, primary}, []sim.EntityID{13, 12}
			wantHero = "female-fighter"
		}
		mw := &mapWorld{mission: &missionNotices{party: party}}
		dir, hasPlayer := mw.playerFigureDir()
		if !hasPlayer || dir != data.FigureDirManMage {
			t.Fatal("StartingHero identity changed with party order")
		}
		actors := append([]speakerActor{{id: 1, fig: figureID{Dir: data.FigureDirWomanFighter}}}, missionSpeakers(nil, nil, nil, party, ids, nil)...)
		cast := speakerCast{actors: actors, playerDir: dir, hasPlayer: hasPlayer, alive: func(sim.EntityID) bool { return true }}
		if actor, ok := cast.resolve(faces[23]); !ok || actor.id != 13 {
			t.Fatalf("reordered=%v resolved %d/%v, want companion13", reordered, actor.id, ok)
		}
		aud := companionReportAudience(speakerAudience(HeroAudience(party), faces), 70, 2, defs, faces, cast)
		assertCompanionReportPart(t, aud, 3, "she-three")
		payload := []byte("<part=1,iammale,iammage>male-mage<part=1,iamfemale,iamfighter>female-fighter")
		if got, ok := EventPart(payload, 1, aud); !ok || got != wantHero {
			t.Fatalf("reordered=%v generic hero body = %q/%v, want %q", reordered, got, ok, wantHero)
		}
	}
}

func TestCompanionReportAudiencePreservesOtherSpeakersAndStartKeyGates(t *testing.T) {
	defs, faces := companionReportRegistry(companionReportFlags, 26)
	for _, npc := range []int32{21, 22, 24, 51} {
		faces[npc] = data.NPCFace{Tokens: tokens(data.NPCTokenFemale)}
	}
	faces[52] = data.NPCFace{Start: true, Tokens: tokens(data.NPCTokenFemale, data.NPCTokenMage)}
	base := speakerAudience(heroAud(false, false), faces)
	aud := companionReportAudience(base, 70, 2, defs, faces, companionReportCast(data.FigureDirManFighter))
	for _, npc := range []int{21, 22, 24, 51, 99} {
		if _, _, resolved := aud.Speaker(npc); resolved {
			t.Fatalf("npc%d acquired a separate Start gate", npc)
		}
	}
	if female, mage, resolved := aud.Speaker(52); !female || !mage || !resolved {
		t.Fatalf("generic Start speaker = %v/%v/%v, want true/true/true", female, mage, resolved)
	}
	if got, ok := EventPart([]byte("<npc=21,part=1,male>first<npc=21,part=1,female>second"), 1, aud); !ok || got != "first" {
		t.Fatalf("ungated other speaker = %q/%v, want first", got, ok)
	}
	if got, ok := EventPart([]byte("<npc=52,part=1,male>first<npc=52,part=1,female,mage>second"), 1, aud); !ok || got != "second" {
		t.Fatalf("gated other speaker = %q/%v, want second", got, ok)
	}
}

func TestCompanionReportDialogueContextIsPagedAndClearedBetweenEvents(t *testing.T) {
	defs, faces := companionReportRegistry(companionReportFlags, 26)
	primary := mapload.PartyMember{StartingHero: true, PlayerCharacter: true, FigureDir: string(data.FigureDirManFighter)}
	cast := companionReportCast(data.FigureDirManFighter)
	w := heroWorld(t, nil, nil, []sim.Entity{{ID: 13, HP: 10, MaxHP: 10}})
	v := worldFixtureViewer(t, worldFixtureMap())
	v.SetFont(missionFont())
	payload := []byte("<npc=23,part=1,male>\r\nhe-one\r\n<npc=23,part=1,female>\r\nshe-one\r\n<npc=23,part=2,male>\r\nhe-two\r\n<npc=23,part=2,female>\r\nshe-two")
	src := missionSource{}
	for _, event := range []int{1, 2, 3} {
		src[missionEvent(t, 70, event)] = payload
	}
	dot := faceDot()
	mw := &mapWorld{world: w, view: v, npcFaces: faces, speakerActors: cast.actors}
	mw.mission = &missionNotices{number: 70, src: src, party: []mapload.PartyMember{primary},
		table: &mapload.Table{NPC: defs}, audience: speakerAudience(HeroAudience([]mapload.PartyMember{primary}), faces),
		faces: missionFaces{23: dot}}
	before := w.Hash()
	for _, tc := range []struct {
		event    int
		one, two string
	}{
		{1, "he-one", "he-two"}, {2, "she-one", "she-two"}, {3, "he-one", "he-two"},
	} {
		if !mw.openDialogue(tc.event) {
			t.Fatalf("event%d did not open", tc.event)
		}
		if body, kind, open := v.NoticeState(); !open || kind != ui.NoticeDialogue || body != tc.one {
			t.Fatalf("event%d opening = %q/%v/%v, want %q", tc.event, body, kind, open, tc.one)
		}
		if _, picture := v.NoticeSpeaker(); picture != dot {
			t.Fatalf("event%d body lost its selected speaker", tc.event)
		}
		if tc.event == 2 {
			mw.speakerActors = nil
		}
		mw.advanceNotice()
		if body, _, open := v.NoticeState(); !open || body != tc.two {
			t.Fatalf("event%d page2 = %q/%v, want %q", tc.event, body, open, tc.two)
		}
		mw.advanceNotice()
		if mw.mission.open || mw.mission.payload != nil || mw.mission.dialogueAudience.Speaker != nil {
			t.Fatalf("event%d retained its dialogue context after close", tc.event)
		}
		if _, _, resolved := mw.mission.audience.Speaker(23); resolved || w.Hash() != before {
			t.Fatalf("event%d changed the generic audience or world hash", tc.event)
		}
	}
}

func TestCompanionReportAnnouncementsResolveTranslatedSavedPlacement(t *testing.T) {
	defs, faces := companionReportRegistry(companionReportFlags, 26)
	primary := mapload.PartyMember{ID: "primary", StartingHero: true, PlayerCharacter: true, FigureDir: string(data.FigureDirManFighter)}
	companion := mapload.PartyMember{ID: "npc:23", PlayerCharacter: true, CompanionNPC: 23, Class: 14, FigureDir: string(data.FigureDirWomanFighter)}
	payload := []byte("<part=1>one<part=2>two" +
		"<npc=23,part=3,male>he-three<npc=23,part=3,female>she-three" +
		"<part=4>four<part=5>five<part=6>six<part=7>seven<part=8>eight" +
		"<npc=23,part=9,male>he-nine<npc=23,part=9,female>she-nine")
	src := missionSource{missionEvent(t, 70, 2): payload}
	for _, tc := range []struct {
		name                string
		present             bool
		hp                  int32
		resolved            bool
		wantThree, wantNine string
	}{
		{"live translated actor", true, 10, true, "she-three", "she-nine"},
		{"absent translated actor", false, 10, false, "he-three", "he-nine"},
		{"dead translated actor", true, -1, false, "he-three", "he-nine"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := worldFixtureMap()
			m.Units = []alm.Unit{{UnitID: 42, Flags: 1, ClassSubID: 23}}
			entities := []sim.Entity{{ID: 18, Owner: sim.SelfSlot, HP: 10, MaxHP: 10}}
			if tc.present {
				entities = append(entities, sim.Entity{ID: 17, MapUnitID: 42, Owner: sim.SelfSlot, HP: tc.hp, MaxHP: 10})
			}
			w := heroWorld(t, nil, nil, entities)
			ms := &Mission{Number: 70, Map: m, World: w, Party: []mapload.PartyMember{primary},
				Start:         mapload.Start{IDs: []sim.EntityID{18}, Roster: map[sim.EntityID]mapload.PartyMember{17: companion}},
				savedDocument: &SnapshotSAVDocument{}}
			p := NewPlayWorld(ms)
			before := w.Hash()
			p.watchAnnouncements(ms, src, faces, &mapload.Table{NPC: defs})
			if female, mage, resolved := p.eventAudience(2).Speaker(23); female != tc.resolved || mage || resolved != tc.resolved {
				t.Fatalf("saved speaker = %v/%v/%v, want %v/false/%v", female, mage, resolved, tc.resolved, tc.resolved)
			}
			report := p.announcementOf(2)
			if !report.Shipped || !report.Shown || len(report.Parts) != 9 {
				t.Fatalf("announcement = %+v, want nine shipped parts", report)
			}
			if report.Parts[2] != tc.wantThree || report.Parts[8] != tc.wantNine {
				t.Fatalf("saved parts = %q/%q, want %q/%q", report.Parts[2], report.Parts[8], tc.wantThree, tc.wantNine)
			}
			if w.Hash() != before {
				t.Fatal("saved announcement changed the world hash")
			}
		})
	}
}
