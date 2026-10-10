package game

import (
	"fmt"
	"image"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// TestReleaseTavernDialogueDressesTheStockUnitItsSpeakerAnswersFor walks the
// shipped campaign's towns and, in every tavern, opens each story cell's
// conversation through the cell's own Talk action. A speaker one of the tavern's
// stock mercenary units answers for is drawn as that unit, wearing the set its
// own installed Humans row gives it (DLG-SPEAKER-023, TAVERN-TALKPIC-016). The
// expected picture is built from the installed rows and item tables by the
// candidate-inspection oracle, without the tavern's or the resolver's routines,
// and the dialogue image the player sees must carry it. The shipped mission-41
// offer's swordswoman (npc90) is one of the speakers the walk must find.
func TestReleaseTavernDialogueDressesTheStockUnitItsSpeakerAnswersFor(t *testing.T) {
	f := releaseFront(t)
	_, s := reachabilityWalkArrive(t, f, "Stock Speaker")
	f.Town.gold = 5_000_000
	src := f.Archives.Containers
	brian := false
	answered := map[int]int{}
	alone := map[int]bool{}
	for _, m := range reachabilityWalkStates() {
		s.markWorldSelected(m)
		if _, ok := f.Town.Won(m); !ok {
			t.Fatalf("Won(%d) refused", m)
		}
		f.addChapterCompanions(f.Town.Chapter())
		if m >= 40 && !brian {
			member, ok := mapload.CampaignNPCMember(f.Table, 25, 0, f.Carried)
			if !ok {
				t.Fatal("no Brian to carry")
			}
			f.Carried = append(f.Carried, member)
			brian = true
		}
		s.room = roomTavern
		s.composeShopFaces()
		for i, cell := range s.tavernSurfaceCells() {
			if !strings.HasPrefix(cell.Semantic, "NPC ") {
				continue
			}
			s.room = roomTavern
			s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: i}, true)
			if s.room != roomTalk {
				t.Logf("chapter %d %s: no conversation opened", f.Town.Chapter(), cell.Semantic)
				continue
			}
			payload, audience := s.townPayload(), HeroAudience(f.Carried)
			for part := 1; ; part++ {
				if part > 1 {
					s.AdvanceTownDialogue()
				}
				line, ok := expectedInstalledDialoguePart(t, payload, part, audience)
				if !ok {
					break
				}
				if s.room != roomTalk || s.dialogue.displayPart != part {
					t.Fatalf("chapter %d %s part %d: pager delivered part %d in room %d",
						f.Town.Chapter(), cell.Semantic, part, s.dialogue.displayPart, s.room)
				}
				if s.dialogue.text != line {
					t.Fatalf("chapter %d %s part %d: pager body %q, want installed body %q",
						f.Town.Chapter(), cell.Semantic, part, s.dialogue.text, line)
				}
				speaker, named := EventPartSpeaker(payload, part, audience)
				if !named {
					continue
				}
				rec := f.NPCFaces[int32(speaker)]
				object := s.tavernTalkObject(speaker)
				pic, win, found := s.speakerFace(speaker)
				if object.stock == 0 {
					if object.live < 0 && rec.Kind == data.NPCFigure && found {
						bare, _ := composeUnitFigure(src, data.Equipment{},
							figureID{Dir: rec.Dir, Face: rec.Face, Hero: rec.Tokens.Has(data.NPCTokenHero)})
						alone[speaker] = imagesEqual(pic, bare)
					}
					continue
				}
				want := expectedInstalledMercenaryTalkPicture(t, f, object.stock)
				where := func() string {
					return fmt.Sprintf("chapter %d %s part %d speaker npc%d unit %d",
						f.Town.Chapter(), cell.Semantic, part, speaker, object.stock)
				}
				if !found || !imagesEqual(pic, want) {
					t.Errorf("%s: dialogue figure is not the unit's own composed figure", where())
				}
				// A Platoon record names no figure, so it has no bare sheet.
				if bare, _ := composeUnitFigure(src, data.Equipment{}, figureID{Dir: rec.Dir, Face: rec.Face}); rec.Kind == data.NPCFigure && imagesEqual(pic, bare) {
					t.Errorf("%s: dialogue figure is the bare sheet", where())
				}
				got, ok := s.TownDialogue()
				layout, _, layoutOK := s.townDialogueLayout()
				if !ok || !layoutOK {
					t.Fatalf("%s: dialogue did not compose", where())
				}
				var window image.Rectangle
				if rec.HasWindow {
					window = win
				}
				if !imagesEqual(got, ui.RenderNotice(layout.WithFaceWindow(window), f.Font.Value(), line, want)) {
					t.Errorf("%s: the dialogue image does not carry the unit's figure", where())
				}
				answered[speaker] = object.stock
			}
			s.room = roomTavern
		}
	}
	if _, ok := answered[90]; !ok {
		t.Fatalf("the walk found no npc90 dialogue answered by a stock unit; answered %v", answered)
	}
	var names []string
	for speaker, typ := range answered {
		names = append(names, fmt.Sprintf("npc%d as unit %d", speaker, typ))
	}
	sort.Strings(names)
	t.Logf("speakers drawn as their stock unit: %s", strings.Join(names, ", "))
	var bareOnly []string
	for speaker, bare := range alone {
		if bare {
			bareOnly = append(bareOnly, fmt.Sprintf("npc%d", speaker))
		}
	}
	sort.Strings(bareOnly)
	t.Logf("figure speakers no unit or party member answers, drawn as the sheet alone: %s", strings.Join(bareOnly, ", "))
}
