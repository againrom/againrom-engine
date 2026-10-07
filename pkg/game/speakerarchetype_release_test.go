package game

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseUnansweredHeroSpeakersUseRegistryArchetypes(t *testing.T) {
	dirs := []data.FigureDir{data.FigureDirManFighter, data.FigureDirManMage, data.FigureDirWomanFighter, data.FigureDirWomanMage}
	wants := [4][4]int{{0, 3, 2, 1}, {0, 3, 2, 0}, {0, 1, 0, 3}, {0, 1, 0, 2}}
	for primary, dir := range dirs {
		f := releaseFront(t)
		f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Archetype witness", Choices: []int{primary / 2, primary % 2, 0}, Stats: []int{31, 27, 24, 29}})
		app, town := openFirstTownShopDialogue(t, f, "archetype dialogue")
		t.Cleanup(app.StopAudio)
		checkDialogueBackdrop(t, f, app, town, "main/text/shop/npc31m31.txt")
		t.Logf("primary %s: App load game / @first / SHOP opens installed npc31m31 dialogue", dir)
		for app.Screen() == ui.ScreenTown && town.room == roomTalk {
			if err := app.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
		}
		seen := [4]bool{}
		payloads := 0
		want, _ := composeInventorySubject(f.Archives.Containers, 0, data.Equipment{}, data.FigureDirManFighter, 5)
		want.Figure = heroBackgroundExpected(t, f.Archives.Containers, data.FigureDirManFighter, want.Figure)
		r := speakerResolver{src: f.Archives.Containers, npcFaces: f.NPCFaces, cast: speakerCast{playerDir: dir, hasPlayer: true}}
		got, _, ok := r.SpeakerFace(21)
		if !ok || got == nil || !bytes.Equal(got.Pix, want.Figure.Pix) {
			t.Fatal("installed npc21 synthesis is not MaleFighter5")
		}
		seen[0] = true
		t.Logf("primary %s npc21: installed registry/archive resolver only; no no-live App state claimed", dir)
		for _, entry := range f.Archives.Containers.Entries() {
			addr := strings.ToLower(entry.Address)
			building := TownBuilding(-1)
			switch {
			case strings.HasPrefix(addr, "main/text/shop/"):
				building = TownShop
			case strings.HasPrefix(addr, "main/text/training/"):
				building = TownSchool
			case strings.HasPrefix(addr, "main/text/inn/npc/"):
				building = TownTavern
			default:
				continue
			}
			var npc, mission int
			if _, err := fmt.Sscanf(path.Base(addr), "npc%dm%d.txt", &npc, &mission); err != nil {
				continue
			}
			payload, err := f.Archives.Containers.ReadFile(entry.Address)
			if err != nil {
				t.Fatal(err)
			}
			payloads++
			needed := false
			for part := 1; part <= 64; part++ {
				if _, ok := EventPart(payload, part, HeroAudience(f.Carried)); !ok {
					break
				}
				if speaker, ok := EventPartSpeaker(payload, part, HeroAudience(f.Carried)); ok && speaker >= 21 && speaker <= 24 && !seen[speaker-21] {
					needed = true
				}
			}
			if !needed {
				continue
			}
			town.composeShopFaces()
			if !town.openTownDialogue(building, TownOffer{Mission: mission, Index: -1}, npc) {
				t.Fatal("installed dialogue did not open", addr)
			}
			for part := 1; town.room == roomTalk && part <= 64; part++ {
				speaker, named := EventPartSpeaker(payload, town.said, HeroAudience(f.Carried))
				if named && speaker >= 21 && speaker <= 24 && !seen[speaker-21] {
					if _, live := town.resolver.cast.resolve(f.NPCFaces[int32(speaker)]); live {
						t.Fatal("witness has a live speaker", addr, speaker)
					}
					index := wants[primary][speaker-21]
					face := []int{5, 3, 1, 1}[index]
					want, _ := composeInventorySubject(f.Archives.Containers, 0, data.Equipment{}, dirs[index], face)
					if !dirs[index].Mage() {
						want.Figure = heroBackgroundExpected(t, f.Archives.Containers, dirs[index], want.Figure)
					}
					got, _, ok := town.speakerFace(speaker)
					if !ok || got == nil || want.Figure == nil || !bytes.Equal(got.Pix, want.Figure.Pix) {
						t.Fatalf("primary %s npc%d: not %s/%d", dir, speaker, dirs[index], face)
					}
					checkDialogueBackdrop(t, f, app, town, addr)
					t.Logf("primary %s npc%d: production openTownDialogue %s + App Enter to part%d draws %s/%d; no live actor; chapter reachability not claimed", dir, speaker, addr, town.said, dirs[index], face)
					seen[speaker-21] = true
				}
				if err := app.HeadlessKey("enter"); err != nil {
					t.Fatal(err)
				}
			}
		}
		if seen != [4]bool{true, true, true, true} {
			t.Fatalf("primary %s: installed dialogues covered %v", dir, seen)
		}
		t.Logf("primary %s: searched %d installed shop/training/inn-NPC payloads; no npc21 line in this population", dir, payloads)
		missionApp := f.App("live primary speaker")
		t.Cleanup(missionApp.StopAudio)
		missionApp.Layout(640, 480)
		if err := missionApp.OpenMission(f.MissionOpenerWith(10, f.NextParty())); err != nil {
			t.Fatal(err)
		}
		live := f.live
		cast := speakerCast{actors: live.speakerActors, alive: live.entityAlive, worn: live.equipmentOf, playerDir: dir, hasPlayer: true}
		actor, found := cast.resolve(f.NPCFaces[21])
		if !found || !actor.me || !actor.hero {
			t.Fatal("npc21 did not resolve to the live primary", dir)
		}
		wanted, _ := composeInventoryPortrait(f.Archives.Containers, uint32(actor.id), live.equipmentOf(actor.id), figureID{Dir: actor.fig.Dir, Face: actor.fig.Face, Hero: true})
		actual, _, found := live.SpeakerFace(21)
		if !found || actual == nil || !bytes.Equal(actual.Pix, wanted.Figure.Pix) {
			t.Fatal("npc21 live primary picture changed", dir)
		}
		t.Logf("primary %s: actual mission10 actor%d answers Hero+Me npc21 with his worn figure; no-live state with this living primary is unreachable", dir, actor.id)
	}
}

func TestReleaseZeroModeNPCFaceByteSurvivesSaveAndLoad(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	app := f.App("zero mode face")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(100)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	placed := 0
	for i, unit := range live.mission.state.Map.Units {
		if unit.UnitID == 168 {
			placed = i
		}
	}
	u := live.mission.state.Map.Units[placed]
	if u.UnitID != 168 || u.Flags&1 == 0 || u.ClassSubID != 55 || f.Table.NPC.Hero(55) {
		t.Fatalf("mission100 npc55 placement changed: %+v", u)
	}
	r := mapload.Resolve(u, f.Table)
	row, err := data.NewHumanDef(f.Table.Humans.EntryName(r.Index), f.Table.Humans.EntryParams(r.Index))
	if err != nil || f.Table.Humans.EntryName(r.Index) != "F_BrigandLeader3" || row.Face != 25 || row.Gender != 0 {
		t.Fatalf("npc55 row changed: %+v %v", row, err)
	}
	people := []placedFacePerson{{entity: sim.EntityID(placed), unit: 168, control: true, dir: data.FigureDirManFighter, face: 25, name: "F_BrigandLeader3", row: row, faceByte: 25}}
	before := live.world.Hash()
	file, raw := writeOrdinarySAV(t, f, "zero-mode.sav")
	placedFaceRoutes(t, live, "mission100 fresh", people)
	placedFacePanes(t, f, app, live, "mission100 fresh", people)
	placedFaceSaved(t, raw, 100, people)
	decoded, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := decoded.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	for _, actor := range graph.Actors {
		if actor.MapUnitID == 168 && actor.Face != 25 {
			t.Fatal("saved constructor face", actor.Face)
		}
	}
	g, loaded := lancerLoad(t, filepath.Dir(file), filepath.Base(file))
	placedFaceRoutes(t, g.live, "mission100 loaded", people)
	placedFacePanes(t, g, loaded, g.live, "mission100 loaded", people)
	t.Logf("mission100 npc55: World before inspection %x after %x; SAV sha256 %x", before, live.world.Hash(), sha256.Sum256(raw))
}
