package game

import (
	"os"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/vfs"
)

// Mission 111's event parts have no voice resource and the mission notice path
// plays none (DLG-VOICE-074): speech.res holds no battle path, and no event
// text path composes a speech path.
func TestReleaseMissionEventDialogueHasNoVoice(t *testing.T) {
	f := releaseFront(t)
	file, err := cutsceneInstallPath(os.Getenv("AGAINROM_ASSETS"), "speech.res")
	if err != nil {
		t.Fatal(err)
	}
	src, err := vfs.OpenFileBacked([]string{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range src.Entries() {
		if strings.Contains(e.Address, "battle") {
			t.Fatalf("speech.res holds a battle voice %q", e.Address)
		}
	}
	parts := 0
	for event := 1; event <= 8; event++ {
		path, _ := EventTextPath(111, event)
		payload, err := f.Archives.Containers.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for n := 1; n <= 12; n++ {
			if _, _, ok := eventPartTag(payload, n, EventAudience{}); !ok {
				continue
			}
			parts++
			if p, voiced := TownSpeechPath(strings.ToLower(path), payload, n, EventAudience{}); voiced {
				t.Fatalf("%s part %d composes speech path %q", path, n, p)
			}
		}
	}
	if parts == 0 {
		t.Fatal("no mission 111 event parts found")
	}
}

// Mission 110 event 02: the parts tagged npc67 show the placed bandit (Humans
// row F_BrigandLeader3, map unit 57) and no party member (DLG-SPEAKER-075).
// Parts tagged npc25 resolve a hero; the RU Part5 speaker is left as the claim
// states it.
func TestReleaseMission110BanditPartsShowTheBandit(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("event speakers")
	app.Layout(1024, 768)
	t.Cleanup(app.StopAudio)
	party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
	if err := app.OpenMission(f.MissionOpenerWith(110, party)); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	path, _ := EventTextPath(110, 2)
	payload, err := f.Archives.Containers.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cast := speakerCast{actors: mw.speakerActors, alive: mw.entityAlive, worn: mw.equipmentOf}
	if dir, ok := mw.playerFigureDir(); ok {
		cast.playerDir, cast.hasPlayer = dir, true
	}
	seen := 0
	for n := 1; n <= 5; n++ {
		speaker, named := EventPartSpeaker(payload, n, mw.mission.dialogueAudience)
		if !named {
			t.Fatalf("part %d names no speaker", n)
		}
		actor, found := cast.resolve(f.NPCFaces[int32(speaker)])
		if !found && speaker == 67 {
			t.Fatalf("part %d speaker npc%d resolves no live actor", n, speaker)
		}
		if !found {
			t.Logf("part %d: npc%d resolves no live actor and shows the synthesised hero figure", n, speaker)
			continue
		}
		e, _ := mw.entity(actor.id)
		t.Logf("part %d: npc%d -> entity %d map unit %d hero %v", n, speaker, actor.id, e.MapUnitID, actor.hero)
		if speaker == 67 {
			seen++
			if actor.hero || e.MapUnitID == 0 || int(actor.id) >= len(mw.mission.state.Map.Units) {
				t.Fatalf("part %d npc67 resolved %+v, want a placed non-hero unit", n, actor)
			}
			name := f.Table.Humans.EntryName(mapload.Resolve(mw.mission.state.Map.Units[actor.id], f.Table).Index)
			if name != "F_BrigandLeader3" {
				t.Fatalf("part %d npc67 shows Humans row %q, want F_BrigandLeader3", n, name)
			}
		} else if !actor.hero {
			t.Fatalf("part %d npc%d resolved a non-hero %+v", n, speaker, actor)
		}
	}
	if seen < 2 {
		t.Fatalf("%d npc67 parts, want at least Part2 and Part3", seen)
	}
}
