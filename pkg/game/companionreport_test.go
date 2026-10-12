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

func companionReportRegistry(flags string) map[int32]data.NPCFace {
	children := []*reg.Node{{Name: "Flags", Type: reg.TypeString, Str: flags}, {Name: "DataBinID", Type: reg.TypeInt, Int: 26}}
	r := &reg.Reg{Root: &reg.Node{Dir: true, Children: []*reg.Node{{Name: "npc23", Dir: true, Children: children}}}}
	return data.LoadNPCFaces(r)
}

const companionReportPayload = "<npc=23,part=3,male>he-three<npc=23,part=3,female>she-three<npc=23,part=9,male>he-nine<npc=23,part=9,female>she-nine"

func assertCompanionReportPart(t *testing.T, aud EventAudience, part int, want string) {
	t.Helper()
	payload := []byte(companionReportPayload)
	got, ok := EventPart(payload, part, aud)
	if !ok || got != want {
		t.Fatalf("part %d = %q/%v, want %q", part, got, ok, want)
	}
	if speaker, named := EventPartSpeaker(payload, part, aud); !named || speaker != 23 {
		t.Fatalf("part %d speaker = %d/%v, want npc23", part, speaker, named)
	}
}

func TestStartTokenSpeakerTakesTheOppositeSexOfTheStartingHero(t *testing.T) {
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
			faces := companionReportRegistry(companionReportFlags)
			if faces[23].Start || !faces[23].Archetype {
				t.Fatal("fixture record must carry the Start token and no Start key")
			}
			aud := speakerAudience(heroAud(tc.dir.Female(), tc.dir.Mage()), faces)
			if female, mage, resolved := aud.Speaker(23); !resolved || female != tc.female || mage {
				t.Fatalf("speaker axes = %v/%v/%v, want %v/false/true", female, mage, resolved, tc.female)
			}
			assertCompanionReportPart(t, aud, 3, tc.wantThree)
			assertCompanionReportPart(t, aud, 9, tc.wantNine)
		})
	}
}

func TestSpeakerGateNeedsAStartKeyOrToken(t *testing.T) {
	faces := companionReportRegistry("Hero,!Mage,!MySex")
	aud := speakerAudience(heroAud(false, false), faces)
	if _, _, resolved := aud.Speaker(23); resolved {
		t.Fatal("a record with neither Start key nor token opened the speaker gate")
	}
	assertCompanionReportPart(t, aud, 3, "he-three")
}

func TestHeroArmsDoNotAlsoTestTheSpeaker(t *testing.T) {
	faces := map[int32]data.NPCFace{22: {Archetype: true, Tokens: tokens(data.NPCTokenHero, data.NPCTokenMage, data.NPCTokenNotMySex)}}
	payload := []byte("<npc=22,part=2,iamfemale>to-her<npc=22,part=2,iammale>to-him")
	for _, female := range []bool{false, true} {
		want := "to-him"
		if female {
			want = "to-her"
		}
		if got, ok := EventPart(payload, 2, speakerAudience(heroAud(female, false), faces)); !ok || got != want {
			t.Fatalf("female hero=%v: part 2 = %q/%v, want %q", female, got, ok, want)
		}
	}
}

func TestHeroAudienceIsTheStartingHeroInAnySlot(t *testing.T) {
	faces := companionReportRegistry(companionReportFlags)
	primary := mapload.PartyMember{ID: "primary", StartingHero: true, PlayerCharacter: true, Mage: true, FigureDir: string(data.FigureDirManMage)}
	companion := mapload.PartyMember{ID: "companion", PlayerCharacter: true, FigureDir: string(data.FigureDirWomanFighter)}
	for _, reordered := range []bool{false, true} {
		party, ids := []mapload.PartyMember{primary, companion}, []sim.EntityID{12, 13}
		if reordered {
			party, ids = []mapload.PartyMember{companion, primary}, []sim.EntityID{13, 12}
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
		aud := speakerAudience(HeroAudience(party), faces)
		assertCompanionReportPart(t, aud, 3, "she-three")
		payload := []byte("<part=1,iammale,iammage>male-mage<part=1,iamfemale,iamfighter>female-fighter")
		if got, ok := EventPart(payload, 1, aud); !ok || got != "male-mage" {
			t.Fatalf("reordered=%v hero body = %q/%v, want male-mage", reordered, got, ok)
		}
	}
}

func TestSpeakerGatePreservesOtherSpeakers(t *testing.T) {
	faces := companionReportRegistry(companionReportFlags)
	for _, npc := range []int32{21, 22, 24, 51} {
		faces[npc] = data.NPCFace{Tokens: tokens(data.NPCTokenFemale)}
	}
	faces[52] = data.NPCFace{Start: true, Tokens: tokens(data.NPCTokenFemale, data.NPCTokenMage)}
	aud := speakerAudience(heroAud(false, false), faces)
	for _, npc := range []int{21, 22, 24, 51, 99} {
		if _, _, resolved := aud.Speaker(npc); resolved {
			t.Fatalf("npc%d acquired a speaker gate", npc)
		}
	}
	if female, mage, resolved := aud.Speaker(52); !female || !mage || !resolved {
		t.Fatalf("Start key speaker = %v/%v/%v, want true/true/true", female, mage, resolved)
	}
	if got, ok := EventPart([]byte("<npc=21,part=1,male>first<npc=21,part=1,female>second"), 1, aud); !ok || got != "first" {
		t.Fatalf("ungated other speaker = %q/%v, want first", got, ok)
	}
	if got, ok := EventPart([]byte("<npc=52,part=1,male>first<npc=52,part=1,female,mage>second"), 1, aud); !ok || got != "second" {
		t.Fatalf("gated other speaker = %q/%v, want second", got, ok)
	}
}

func TestSpeakerVariantIsPagedForEveryEvent(t *testing.T) {
	faces := companionReportRegistry(companionReportFlags)
	primary := mapload.PartyMember{StartingHero: true, PlayerCharacter: true, FigureDir: string(data.FigureDirManFighter)}
	w := heroWorld(t, nil, nil, []sim.Entity{{ID: 13, HP: 10, MaxHP: 10}})
	v := worldFixtureViewer(t, worldFixtureMap())
	v.SetFont(missionFont())
	payload := []byte("<npc=23,part=1,male>\r\nhe-one\r\n<npc=23,part=1,female>\r\nshe-one\r\n<npc=23,part=2,male>\r\nhe-two\r\n<npc=23,part=2,female>\r\nshe-two")
	src := missionSource{}
	for _, event := range []int{1, 2, 3} {
		src[missionEvent(t, 111, event)] = payload
	}
	dot := faceDot()
	mw := &mapWorld{world: w, view: v, npcFaces: faces}
	mw.mission = &missionNotices{number: 111, src: src, party: []mapload.PartyMember{primary},
		audience: speakerAudience(HeroAudience([]mapload.PartyMember{primary}), faces),
		faces:    missionFaces{23: dot}}
	before := w.Hash()
	for _, event := range []int{1, 2, 3} {
		if !mw.openDialogue(event) {
			t.Fatalf("event%d did not open", event)
		}
		if body, kind, open := v.NoticeState(); !open || kind != ui.NoticeDialogue || body != "she-one" {
			t.Fatalf("event%d opening = %q/%v/%v, want she-one", event, body, kind, open)
		}
		if _, picture := v.NoticeSpeaker(); picture != dot {
			t.Fatalf("event%d body lost its selected speaker", event)
		}
		mw.advanceNotice()
		if body, _, open := v.NoticeState(); !open || body != "she-two" {
			t.Fatalf("event%d page2 = %q/%v, want she-two", event, body, open)
		}
		mw.advanceNotice()
		if mw.mission.open || mw.mission.payload != nil || mw.mission.dialogueAudience.Speaker != nil {
			t.Fatalf("event%d retained its dialogue context after close", event)
		}
		if w.Hash() != before {
			t.Fatalf("event%d changed the world hash", event)
		}
	}
}

func TestScenarioAnnouncementTakesTheSpeakerSexWhateverTheActor(t *testing.T) {
	faces := companionReportRegistry(companionReportFlags)
	primary := mapload.PartyMember{ID: "primary", StartingHero: true, PlayerCharacter: true, FigureDir: string(data.FigureDirManFighter)}
	companion := mapload.PartyMember{ID: "npc:23", PlayerCharacter: true, CompanionNPC: 23, Class: 14, FigureDir: string(data.FigureDirWomanFighter)}
	payload := []byte("<part=1>one<part=2>two" +
		"<npc=23,part=3,male>he-three<npc=23,part=3,female>she-three" +
		"<part=4>four<part=5>five<part=6>six<part=7>seven<part=8>eight" +
		"<npc=23,part=9,male>he-nine<npc=23,part=9,female>she-nine")
	src := missionSource{missionEvent(t, 70, 2): payload}
	for _, tc := range []struct {
		name    string
		present bool
		hp      int32
	}{
		{"live translated actor", true, 10},
		{"absent translated actor", false, 10},
		{"dead translated actor", true, -1},
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
			p.watchAnnouncements(ms, src, faces)
			if female, mage, resolved := p.audience.Speaker(23); !female || mage || !resolved {
				t.Fatalf("speaker = %v/%v/%v, want true/false/true", female, mage, resolved)
			}
			report := p.announcementOf(2)
			if !report.Shipped || !report.Shown || len(report.Parts) != 9 {
				t.Fatalf("announcement = %+v, want nine shipped parts", report)
			}
			if report.Parts[2] != "she-three" || report.Parts[8] != "she-nine" {
				t.Fatalf("parts = %q/%q, want she-three/she-nine", report.Parts[2], report.Parts[8])
			}
			if w.Hash() != before {
				t.Fatal("announcement changed the world hash")
			}
		})
	}
}
