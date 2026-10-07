package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"

	"golang.org/x/text/encoding/charmap"
)

func TestReleaseMission70CompanionReportUsesLiveSpeakerVariants(t *testing.T) {
	output := os.Getenv("AGAINROM_NAIRA_DIALOGUE_WITNESS_DIR")
	if output == "" {
		t.Skip("AGAINROM_NAIRA_DIALOGUE_WITNESS_DIR is required")
	}
	for _, primaryDir := range []data.FigureDir{data.FigureDirManFighter, data.FigureDirManMage, data.FigureDirWomanFighter, data.FigureDirWomanMage} {
		t.Run(string(primaryDir), func(t *testing.T) {
			f := releaseFront(t)
			root, err := filepath.EvalSymlinks(f.Archives.Root)
			if err != nil || !filepath.IsAbs(root) || !filepath.IsAbs(output) {
				t.Fatal("absolute resolved install and witness roots required", err)
			}
			rel, err := filepath.Rel(root, filepath.Clean(output))
			if err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)) {
				t.Fatal("dialogue witness output is inside the install")
			}
			namespace := fmt.Sprintf("%x", sha256.Sum256([]byte(filepath.ToSlash(root))))
			dir := filepath.Join(output, namespace, strings.ReplaceAll(t.Name(), "/", "_"))
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			f.Options = OptionsStore{Path: filepath.Join(dir, "options.txt")}
			f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer, f.SpeechPlayer = nil, nil, nil, nil, nil
			f.SetDeterministicFrames(true)
			primary := f.NextParty()[0]
			primary.Mage, primary.FigureDir = primaryDir.Mage(), string(primaryDir)
			primary.StartingHero, primary.PlayerCharacter = true, true
			template, templateOK := mapload.CampaignNPCMember(f.Table, 23, 70, []mapload.PartyMember{primary})
			app := f.App("companion report")
			t.Cleanup(app.StopAudio)
			app.Layout(640, 480)
			if err := app.OpenMission(f.MissionOpenerWith(70, []mapload.PartyMember{primary})); err != nil {
				t.Fatal(err)
			}
			live := f.live
			id := releaseRosterNPC(t, live.mission.state, 23)
			member := live.mission.state.Start.Roster[id]
			wantDir, wantClass, wantRow := data.FigureDirWomanFighter, int32(14), "PC_Naira_2"
			if primaryDir.Female() {
				wantDir, wantClass, wantRow = data.FigureDirManFighter, 3, "PC_Danath_2"
			}
			cast := speakerCast{actors: live.speakerActors, alive: live.entityAlive, worn: live.equipmentOf, playerDir: primaryDir, hasPlayer: true}
			actor, found := cast.resolve(live.npcFaces[23])
			if !templateOK || template.Class != wantClass || template.Name != wantRow || !found || actor.id != id || !actor.hero || actor.fig.Dir != wantDir || member.CompanionNPC != 23 || !live.entityAlive(id) {
				t.Fatal("installed NPC23 live branch differs", actor, template.Class, template.Name)
			}
			fig := actor.fig
			fig.Hero = true
			portrait, _ := composeUnitFigure(f.Archives.Containers, live.equipmentOf(id), fig)
			if portrait == nil {
				t.Fatal("resolved live speaker has no installed portrait")
			}
			payload, err := f.Archives.Containers.ReadFile("main/text/battle/m70/event02.txt")
			if err != nil {
				t.Fatal(err)
			}
			ru := fmt.Sprintf("%x", sha256.Sum256(payload)) == "7f37e564e6c1ec4a3a488ed93f5f39819e344f2e1ae3780705b6dfc1b52d68f0"
			if !ru && fmt.Sprintf("%x", sha256.Sum256(payload)) != "e3a91ecf3a1c705eeee50af3d97de354a0dbc1d326623579606e5c700fd2ede1" {
				t.Fatal("installed event02 source changed")
			}
			registry, err := f.Archives.Containers.ReadFile(NPCRegistry)
			if err != nil || fmt.Sprintf("%x", sha256.Sum256(registry)) != "c91cff00c382617fb4867f734a176eb2b0358c6896e4e7b346384d3ceecfb223" {
				t.Fatal("installed NPC registry changed", err)
			}
			if err := app.HeadlessKey("0"); err != nil {
				t.Fatal(err)
			}
			for n := 0; live.view.NoticeOpen() && n < 32; n++ {
				if err := app.HeadlessKey("enter"); err != nil {
					t.Fatal(err)
				}
			}
			if !live.stopped || live.view.NoticeOpen() {
				t.Fatal("controlled dialogue did not settle on a stopped map")
			}
			before := live.world.Hash()
			if !live.openDialogue(2) || live.world.Hash() != before {
				t.Fatal("ordinary dialogue open failed or changed World")
			}
			played := NewPlayWorld(live.mission.state)
			played.watchAnnouncements(live.mission.state, f.Archives.Containers, live.npcFaces, f.Table)
			report := played.announcementOf(2)
			proof := map[string]any{"root": root, "root_path_sha256": namespace, "source_sha256": fmt.Sprintf("%x", sha256.Sum256(payload)), "registry_sha256": fmt.Sprintf("%x", sha256.Sum256(registry)), "primary": primaryDir, "speaker": id, "figure": actor.fig.Dir, "class": member.Class, "template_class": template.Class, "template_row": template.Name, "portrait_sha256": fmt.Sprintf("%x", sha256.Sum256(portrait.Pix)), "world_before": fmt.Sprintf("%x", before)}
			for part := 1; live.view.NoticeOpen() && part < 32; part++ {
				if part == 3 || part == 9 {
					t.Run(fmt.Sprintf("part%d", part), func(t *testing.T) {
						body, kind, open := live.view.NoticeState()
						want := companionNativeBody(t, payload, part, actor.fig.Dir.Female())
						if !open || kind != ui.NoticeDialogue || body != want || live.mission.part != part {
							t.Errorf("part%d delivered body hash %x, want independent installed variant %x", part, sha256.Sum256([]byte(body)), sha256.Sum256([]byte(want)))
						}
						if len(report.Parts) < part || strings.TrimSpace(report.Parts[part-1]) != decodeInstallText(want) {
							t.Error("scenario announcement disagrees with independent installed variant", part)
						}
						pane, pic := live.view.NoticeSpeaker()
						if !pane || pic == nil || pic.Bounds() != portrait.Bounds() || !bytes.Equal(pic.Pix, portrait.Pix) {
							t.Fatal("ordinary dialogue portrait differs from the resolved live actor")
						}
						layout := f.Words.OnLayout(ui.AuthoredDialogueLayout()).WithPortrait(true)
						layout.Frame = f.gameMenuArt()
						frame := image.NewRGBA(image.Rect(0, 0, 640, 480))
						ui.ComposeDialogueNotice(frame, layout, f.Font.Value(), body, pic, layout.Box.Min)
						checkDialogueText(t, "installed companion report", frame, f.Font.Value(), want, ui.NoticeLayoutOf(layout, f.Font.Value(), want))
						if ru {
							female, male := "оказалась", "оказался"
							if part == 9 {
								female, male = "устроила", "устроил"
							}
							word, wrong := male, female
							if actor.fig.Dir.Female() {
								word, wrong = female, male
							}
							companionGenderPixels(t, f, frame, layout, want, word, wrong)
							if part == 3 && !strings.Contains(decodeInstallText(body), "Умойpа") {
								t.Error("installed Latin p in the place name was rewritten")
							}
						}
						proof[fmt.Sprintf("part%d_body_sha256", part)] = fmt.Sprintf("%x", sha256.Sum256([]byte(body)))
						proof[fmt.Sprintf("part%d_widget_pix_sha256", part)] = fmt.Sprintf("%x", sha256.Sum256(frame.Pix))
						companionWidgetPNG(t, filepath.Join(dir, fmt.Sprintf("part%d-dialogue-widget.png", part)), frame)
					})
				}
				if err := app.HeadlessKey("enter"); err != nil {
					t.Fatal(err)
				}
			}
			if live.view.NoticeOpen() || proof["part3_body_sha256"] == nil || proof["part9_body_sha256"] == nil {
				t.Fatal("ordinary Enter did not reach both parts and close the dialogue")
			}
			if live.world.Hash() != before || app.Screen() != ui.ScreenMap || f.live != live {
				t.Fatal("dialogue paging changed World or map identity")
			}
			if primaryDir == data.FigureDirManFighter {
				t.Run("current SAV cold LOAD", func(t *testing.T) {
					companionColdReport(t, f, app, payload, portrait, id, dir, ru)
				})
			}
			start := live.world.Tick()
			if err := app.HeadlessKey("0"); err != nil {
				t.Fatal(err)
			}
			for range 8 {
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			if live.world.Tick() <= start {
				t.Fatal("ordinary App ticks did not resume after dialogue")
			}
			proof["tick_after"] = live.world.Tick()
			encoded, err := json.MarshalIndent(proof, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "witness.json"), append(encoded, '\n'), 0600); err != nil {
				t.Fatal(err)
			}
			t.Logf("live NPC23 actor%d %s class%d portrait%x; source%x, root namespace %s; ordinary ticks %d -> %d", id, actor.fig.Dir, member.Class, sha256.Sum256(portrait.Pix), sha256.Sum256(payload), namespace, start, live.world.Tick())
		})
	}
}

func companionColdReport(t *testing.T, f *FrontEnd, app *ui.App, payload []byte, portrait *image.RGBA, speaker sim.EntityID, dir string, ru bool) {
	t.Helper()
	dir = filepath.Join(dir, "current-sav-cold-load")
	store := SaveStore{Dir: filepath.Join(dir, "saves")}
	save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	before, tick := f.live.world.Hash(), f.live.world.Tick()
	if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenGameMenu {
		t.Fatal("ordinary current SAV failed", err, app.Screen())
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 || !IsOriginal(entries[0].Name) {
		t.Fatal("ordinary F2 did not emit one current SAV", entries, err)
	}
	raw, err := store.Read(entries[0].Name)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenMap || f.live.world.Hash() != before {
		t.Fatal("ordinary SAVE return changed the World", err)
	}
	cold := releaseFront(t)
	cold.Options = OptionsStore{Path: filepath.Join(dir, "cold-options.txt")}
	cold.SoundPlayer, cold.MusicPlayer, cold.AmbientPlayer, cold.CutsceneAudioPlayer, cold.SpeechPlayer = nil, nil, nil, nil, nil
	cold.SetDeterministicFrames(true)
	coldApp := cold.App("current SAV companion report")
	t.Cleanup(coldApp.StopAudio)
	coldApp.Layout(640, 480)
	coldSave, coldList, coldLoad := cold.SaveSeams(store, OriginalStore{}, nil)
	coldApp.SetSaveSeams(coldSave, coldList, coldLoad)
	groundAppLoad(t, coldApp, coldList, entries[0].Name)
	live := cold.live
	if live == nil || live.mission.number != 70 || live.world.Tick() != tick {
		t.Fatal("cold LOAD did not restore this generated mission cut")
	}
	if !live.stopped {
		if err := coldApp.HeadlessKey("0"); err != nil {
			t.Fatal(err)
		}
	}
	for n := 0; live.view.NoticeOpen() && n < 32; n++ {
		if err := coldApp.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	playerDir, hasPlayer := live.playerFigureDir()
	cast := speakerCast{actors: live.speakerActors, alive: live.entityAlive, playerDir: playerDir, hasPlayer: hasPlayer}
	actor, resolved := cast.resolve(live.npcFaces[23])
	if !live.stopped || live.view.NoticeOpen() || !hasPlayer || playerDir != data.FigureDirManFighter || !resolved || actor.id != speaker || !actor.hero || actor.fig.Dir != data.FigureDirWomanFighter {
		t.Fatal("cold LOAD lost the live female NPC23 or StartingHero identity", actor)
	}
	before = live.world.Hash()
	if !live.openDialogue(2) {
		t.Fatal("cold LOAD event02 did not open")
	}
	played := NewPlayWorld(live.mission.state)
	played.watchAnnouncements(live.mission.state, cold.Archives.Containers, live.npcFaces, cold.Table)
	report := played.announcementOf(2)
	proof := map[string]any{"generated_sav_sha256": fmt.Sprintf("%x", sha256.Sum256(raw)), "generated_sav": entries[0].Name, "speaker": actor.id, "figure": actor.fig.Dir, "class": live.mission.state.Start.Roster[actor.id].Class, "world_before": fmt.Sprintf("%x", before), "tick_before": tick}
	for part := 1; live.view.NoticeOpen() && part < 32; part++ {
		if part == 3 || part == 9 {
			t.Run(fmt.Sprintf("part%d", part), func(t *testing.T) {
				body, kind, open := live.view.NoticeState()
				want := companionNativeBody(t, payload, part, true)
				pane, pic := live.view.NoticeSpeaker()
				if !open || kind != ui.NoticeDialogue || body != want || live.mission.part != part || !pane || pic == nil || pic.Bounds() != portrait.Bounds() || !bytes.Equal(pic.Pix, portrait.Pix) {
					t.Fatal("ordinary cold LOAD dialogue differs from the independent female variant or live portrait")
				}
				if len(report.Parts) < part || strings.TrimSpace(report.Parts[part-1]) != decodeInstallText(want) {
					t.Fatal("cold LOAD scenario report differs from the independent female variant")
				}
				layout := cold.Words.OnLayout(ui.AuthoredDialogueLayout()).WithPortrait(true)
				layout.Frame = cold.gameMenuArt()
				frame := image.NewRGBA(image.Rect(0, 0, 640, 480))
				ui.ComposeDialogueNotice(frame, layout, cold.Font.Value(), body, pic, layout.Box.Min)
				checkDialogueText(t, "cold current companion report", frame, cold.Font.Value(), want, ui.NoticeLayoutOf(layout, cold.Font.Value(), want))
				if ru {
					word, wrong := "оказалась", "оказался"
					if part == 9 {
						word, wrong = "устроила", "устроил"
					}
					companionGenderPixels(t, cold, frame, layout, want, word, wrong)
				}
				proof[fmt.Sprintf("part%d_body_sha256", part)] = fmt.Sprintf("%x", sha256.Sum256([]byte(body)))
				proof[fmt.Sprintf("part%d_widget_pix_sha256", part)] = fmt.Sprintf("%x", sha256.Sum256(frame.Pix))
				companionWidgetPNG(t, filepath.Join(dir, fmt.Sprintf("part%d-dialogue-widget.png", part)), frame)
			})
		}
		if err := coldApp.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if live.view.NoticeOpen() || proof["part3_body_sha256"] == nil || proof["part9_body_sha256"] == nil || live.world.Hash() != before || coldApp.Screen() != ui.ScreenMap {
		t.Fatal("cold LOAD dialogue did not close cleanly or changed World")
	}
	if err := coldApp.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	for range 8 {
		if err := coldApp.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if live.world.Tick() <= tick {
		t.Fatal("ordinary cold LOAD App ticks did not resume")
	}
	proof["tick_after"] = live.world.Tick()
	encoded, err := json.MarshalIndent(proof, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "witness.json"), append(encoded, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("generated current SAV %x; cold live female NPC23 actor%d, ordinary ticks %d -> %d", sha256.Sum256(raw), actor.id, tick, live.world.Tick())
}

func companionNativeBody(t *testing.T, source []byte, part int, female bool) string {
	t.Helper()
	lines := strings.Split(string(source), "\r\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, "<") || !strings.HasSuffix(strings.TrimSpace(line), ">") {
			continue
		}
		npc, number, sex := -1, -1, ""
		for _, term := range strings.Split(strings.Trim(strings.TrimSpace(line), "<>"), ",") {
			term = strings.ToLower(strings.TrimSpace(term))
			switch {
			case strings.HasPrefix(term, "npc="):
				npc, _ = strconv.Atoi(strings.TrimPrefix(term, "npc="))
			case strings.HasPrefix(term, "part="):
				number, _ = strconv.Atoi(strings.TrimPrefix(term, "part="))
			case term == "female" || term == "male":
				sex = term
			}
		}
		if npc != 23 || number != part || sex == "female" && !female || sex == "male" && female {
			continue
		}
		if i+2 >= len(lines) || !strings.HasPrefix(lines[i+2], "<") {
			t.Fatal("independent native source part is not the pinned single-line body")
		}
		return lines[i+1]
	}
	t.Fatal("independent native variant missing", part, female)
	return ""
}

func companionGenderPixels(t *testing.T, f *FrontEnd, frame *image.RGBA, layout ui.NoticeLayout, source, word, wrong string) {
	t.Helper()
	raw, err := charmap.CodePage866.NewEncoder().String(word)
	if err != nil {
		t.Fatal(err)
	}
	counterfeit, err := charmap.CodePage866.NewEncoder().String(wrong)
	if err != nil {
		t.Fatal(err)
	}
	font := f.Font.Value()
	placed := dialogueLines(t, "gender literal", source, ui.NoticeLayoutOf(layout, font, source))
	for row, line := range placed {
		x, width := dlgTextLeft, dlgTextWidth
		if line.first {
			x, width = x+dlgIndent, width-dlgIndent
		}
		sum := 0
		for _, token := range line.words {
			sum += font.Advance(token)
		}
		gap := float64(font.Advance(" "))
		if line.justified {
			gap = float64(width-sum) / float64(len(line.words)-1)
		}
		cursor := float64(x)
		for _, token := range line.words {
			if strings.Trim(token, ",.") == raw {
				origin := image.Pt(int(cursor), dlgTextTop+dlgPitch*row)
				correct := dialogueRunPixels(font, raw, origin.X, origin.Y, dlgWhite, 1)
				counterfeitRun := dialogueRunPixels(font, counterfeit, origin.X, origin.Y, dlgWhite, 1)
				background := image.NewRGBA(frame.Bounds())
				ui.ComposeDialogueNotice(background, layout, font, "", nil, layout.Box.Min)
				points := map[image.Point]bool{}
				for p := range correct {
					points[p] = true
				}
				for p := range counterfeitRun {
					points[p] = true
				}
				qualified := 0
				for p := range points {
					want, present := correct[p]
					if !present {
						want = background.RGBAAt(p.X, p.Y)
					}
					counterfeit, present := counterfeitRun[p]
					if !present {
						counterfeit = background.RGBAAt(p.X, p.Y)
					}
					if want != counterfeit && frame.RGBAAt(p.X, p.Y) == want {
						qualified++
					}
				}
				if !runMatches(frame, correct) || qualified == 0 {
					t.Error("independently authored gender glyph does not distinguish the delivered variant", word, origin)
				}
				for _, shift := range []int{-1, 1} {
					if runMatches(frame, dialogueRunPixels(font, raw, origin.X+shift, origin.Y, dlgWhite, 1)) {
						t.Error("gender glyph accepts a one-pixel displacement", shift)
					}
				}
				t.Logf("authored gender glyph at %v: %d pixels distinguish %s from %s, including absent ink; one-pixel displacement controls reject", origin, qualified, word, wrong)
				return
			}
			cursor = cursor + float64(font.Advance(token)) + gap
		}
	}
	t.Error("authored gender literal absent from visible installed lines", word)
}

func companionWidgetPNG(t *testing.T, path string, frame *image.RGBA) {
	t.Helper()
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(out, frame); err != nil {
		out.Close()
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}
