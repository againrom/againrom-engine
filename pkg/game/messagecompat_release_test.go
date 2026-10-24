package game

import (
	"crypto/sha256"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseMessageReaderCompatibility(t *testing.T) {
	f := releaseFront(t)
	src := f.Archives.Containers
	var addresses []string
	for _, entry := range src.Entries() {
		name := strings.ToLower(entry.Address)
		if strings.HasPrefix(name, "main/text/") && strings.HasSuffix(name, ".txt") {
			addresses = append(addresses, entry.Address)
		}
	}
	sort.Strings(addresses)
	h := sha256.New()
	files, parts := 0, 0
	events := map[int][]int{}
	for _, address := range addresses {
		payload, err := src.ReadFile(address)
		if err != nil {
			t.Fatal(err)
		}
		files++
		var mission, event int
		if _, err := fmt.Sscanf(strings.ToLower(address), "main/text/battle/m%d/event%d.txt", &mission, &event); err == nil {
			events[mission] = append(events[mission], event)
		}
		for _, female := range []bool{false, true} {
			for _, mage := range []bool{false, true} {
				audience := EventAudience{HeroFemale: female, HeroMage: mage}
				for n := 0; n <= 32; n++ {
					tag, body, ok := eventPartTag(payload, n, audience)
					speaker, named := EventPartSpeaker(payload, n, audience)
					fmt.Fprintf(h, "%s/%t/%t/%d/%t/%t/%d/%q/%q\n", address, female, mage, n, ok, named, speaker, tag, body)
					if ok {
						parts++
					}
				}
			}
		}
	}
	defs, err := LoadDefinitions(src)
	if err != nil {
		t.Fatal(err)
	}
	party := MissionPartyAs(false, defs.StartWeapon, defs.Bodies, defs.Table)
	worlds := sha256.New()
	maps := 0
	for number := 10; number <= 290; number++ {
		ms, err := StartMission(src, number, defs.Table, mapload.DifficultyNormal, party)
		if err != nil || ms.World.Script() == nil {
			continue
		}
		maps++
		for tick := 0; tick <= 32; tick++ {
			fmt.Fprintf(worlds, "%d/%d/%x\n", number, tick, ms.World.Hash())
			sim.Step(ms.World, nil)
		}
	}
	if files == 0 || parts == 0 || maps < 28 {
		t.Fatal("empty compatibility population", files, parts, maps)
	}
	frames := sha256.New()
	var missions []int
	for mission := range events {
		missions = append(missions, mission)
	}
	sort.Ints(missions)
	count := 0
	for _, mission := range missions {
		front := missionTipFront(t)
		app, live := openMissionTipMap(t, front, mission)
		for live.view.NoticeOpen() {
			if err := app.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
		}
		for _, event := range events[mission] {
			if !live.openDialogue(event) {
				continue
			}
			for page := 0; live.view.NoticeOpen() && page < 64; page++ {
				body, _, _ := front.LiveNotice()
				portrait, face := live.view.NoticeSpeaker()
				layout := front.Words.OnLayout(ui.AuthoredDialogueLayout())
				layout.Frame = front.gameMenuArt()
				frame := image.NewRGBA(image.Rect(0, 0, 640, 480))
				ui.ComposeDialogueNotice(frame, layout.WithPortrait(portrait), front.Font.Value(), body, face, layout.Box.Min)
				fmt.Fprintf(frames, "%d/%d/%d/%x\n", mission, event, page, sha256.Sum256(frame.Pix))
				count++
				if err := app.HeadlessKey("enter"); err != nil {
					t.Fatal(err)
				}
			}
		}
		app.StopAudio()
	}
	receipt := fmt.Sprintf("files=%d accepted_parts=%d reader_sha256=%x\nworld_maps=%d ticks_per_map=33 world_sha256=%x\nmessage_frames=%d frames_sha256=%x\n", files, parts, h.Sum(nil), maps, worlds.Sum(nil), count, frames.Sum(nil))
	t.Log(receipt)
	dir := os.Getenv("AGAINROM_MESSAGE_PICKER_FRAMES")
	if dir == "" {
		t.Fatal("message witness output is required")
	}
	if err := os.WriteFile(filepath.Join(dir, "message-compatibility.txt"), []byte(receipt), 0600); err != nil {
		t.Fatal(err)
	}
	x := installedInspector(t)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	if err := e.Key("help"); err != nil {
		t.Fatal(err)
	}
	_ = e.Key("end")
	messageFrame(t, e, x.pageLanguage(), "f1-dialogue-help-compat")
}
