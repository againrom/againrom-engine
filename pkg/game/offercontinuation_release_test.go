package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

func offerTextOverlay(t *testing.T, f *FrontEnd, files []synth.File) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), MainArchive)
	if err := os.WriteFile(name, synth.Archive(files), 0o600); err != nil {
		t.Fatal(err)
	}
	useOfferTextOverlay(t, f, name)
	return name
}

func useOfferTextOverlay(t *testing.T, f *FrontEnd, name string) {
	t.Helper()
	paths := []string{name}
	for _, archive := range RequiredArchives() {
		host, err := cutsceneInstallPath(f.Archives.Root, archive)
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, host)
	}
	src, err := vfs.OpenFileBacked(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := *f.Archives
	a.Containers = src
	f.Archives = &a
}

func coldOfferApp(t *testing.T, raw []byte, overlay string) (*FrontEnd, *ui.App, *townScreen) {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(store.Dir, "town.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	if overlay != "" {
		useOfferTextOverlay(t, f, overlay)
	}
	app := f.App("offer continuation")
	app.Layout(640, 480)
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{}, nil))
	t.Cleanup(app.StopAudio)
	for _, target := range []string{"load game", "@first"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatalf("cold App LOAD %q: %v", target, err)
		}
	}
	s := f.TownScreen().(*townScreen)
	if app.Screen() != ui.ScreenTown || s.room != roomSquare {
		t.Fatal("cold App LOAD did not stop at the square", app.Screen(), s.room)
	}
	s.CloseTip()
	return f, app, s
}

func TestReleaseOfferFirstPartStockCompatibility(t *testing.T) {
	f := releaseFront(t)
	wantNodes := []string{
		"main/text/shop/npc31m110.txt", "main/text/shop/npc31m130.txt",
		"main/text/shop/npc31m31.txt", "main/text/shop/npc31m91.txt",
		"main/text/training/npc34m121.txt", "main/text/training/npc34m131.txt",
		"main/text/training/npc34m61.txt",
	}
	var nodes []string
	for _, entry := range f.Archives.Containers.Entries() {
		for _, prefix := range []string{"main/text/shop/npc31m", "main/text/training/npc34m"} {
			if !strings.HasPrefix(entry.Address, prefix) || !strings.HasSuffix(entry.Address, ".txt") {
				continue
			}
			mission := strings.TrimSuffix(strings.TrimPrefix(entry.Address, prefix), ".txt")
			if n, err := strconv.Atoi(mission); err == nil && n > 0 {
				nodes = append(nodes, entry.Address)
			}
		}
	}
	slices.Sort(nodes)
	if !reflect.DeepEqual(nodes, wantNodes) {
		t.Fatal("installed shop/training population changed", nodes)
	}
	for _, node := range nodes {
		t.Run(node, func(t *testing.T) {
			g := releaseFront(t)
			building, door, room, prefix := TownShop, "SHOP", roomShop, "main/text/shop/npc31m"
			if strings.Contains(node, "/training/") {
				building, door, room, prefix = TownSchool, "SCHOOL", roomSchool, "main/text/training/npc34m"
			}
			var mission int
			if _, err := fmt.Sscanf(strings.TrimPrefix(node, prefix), "%d.txt", &mission); err != nil {
				t.Fatal(err)
			}
			payload, err := g.Archives.Containers.ReadFile(node)
			if err != nil {
				t.Fatal(err)
			}
			app, s := roomExitApp(t, g, mission/10*10)
			t.Cleanup(app.StopAudio)
			s.CloseTip()
			offers := g.Town.Offers(building)
			if len(offers) == 0 || offers[0].Mission != mission {
				t.Fatal("installed chapter does not expose the measured node as its head", mission, offers)
			}
			if err := app.HeadlessActivate(door); err != nil {
				t.Fatal(err)
			}
			first, accepted := EventPart(payload, 1, HeroAudience(g.Carried))
			if !accepted || s.room != roomTalk || s.said != 1 || s.dialogue.displayPart != 1 || strings.TrimSpace(s.dialogue.text) != strings.TrimSpace(first) {
				t.Fatal("stock first part changed", s.room, s.said, s.dialogue.displayPart, accepted)
			}
			if !containsMission(g.Town.Available(), mission) || len(g.Town.Offers(building)) != len(offers)-1 || g.Town.selectedMission() != mission {
				t.Fatal("stock entry did not register and consume its head", mission, g.Town.Offers(building), g.Town.Available())
			}
			pages := 1
			for pages < 64 {
				dialogueKeyPress(t, app, s, "enter")
				next, ok := EventPart(payload, pages+1, HeroAudience(g.Carried))
				if !ok {
					break
				}
				pages++
				if s.room != roomTalk || s.said != pages || s.dialogue.displayPart != pages || strings.TrimSpace(s.dialogue.text) != strings.TrimSpace(next) {
					t.Fatal("stock following part changed", pages, s.room, s.said, s.dialogue.displayPart)
				}
			}
			if s.room != room {
				t.Fatal("stock conversation did not close to its building", pages, s.room)
			}
			t.Logf("installed %s: %d accepted parts; head %d registered on entry", node, pages, mission)
		})
	}
	t.Logf("installed root %s: %d stock offer nodes; EN and RU together cover 14", f.Archives.Root, len(nodes))
}

func preparedOfferTown(t *testing.T, chapter, mission int, building TownBuilding) *FrontEnd {
	t.Helper()
	f := currentTown(t, nil, nil)
	p := f.Town.progress
	if p == nil {
		t.Fatal("mission return has no live campaign progress")
	}
	for p.main.mission < chapter {
		p.advanceMain(f.Campaign.Value(), p.main.mission)
	}
	if p.main.mission != chapter || p.record(mission) == nil {
		t.Fatal("installed chapter has no live offered record", chapter, mission)
	}
	rows := f.Campaign.Value().Chapters[chapter].Shop
	if building == TownSchool {
		rows = f.Campaign.Value().Chapters[chapter].School
	}
	if !containsMission(rows, mission) {
		t.Fatal("chosen installed chapter does not offer the witness mission", rows, mission)
	}
	p.selected, p.markers, p.markerSelected = chapter, nil, false
	p.record(mission).announced = false
	if building == TownShop {
		p.shopMission = []int{mission, mission}
	} else {
		p.tcMission = []int{mission, mission}
	}
	f.Town.offerLabels = nil
	clearTownTestGateLatches(f.Town)
	f.Town.taken = map[offerRef]bool{}
	s := f.TownScreen().(*townScreen)
	s.worldSelectedOnce = nil
	return f
}

func offerRecord(t *testing.T, c sav.CampaignProjection, mission int) sav.CampaignRecord {
	t.Helper()
	if c.Main.Mission == uint32(mission) {
		return c.Main
	}
	for _, child := range c.Children {
		if child.Mission == uint32(mission) {
			return child
		}
	}
	t.Fatal("saved campaign lacks the selected record", mission)
	return sav.CampaignRecord{}
}

func offerMarkersEqual(g *FrontEnd, want []sav.CampaignMarker) bool {
	return reflect.DeepEqual(g.Town.progress.projection().Markers, want)
}

func assertOfferAppPanel(t *testing.T, f *FrontEnd, app *ui.App, body string) {
	t.Helper()
	layout := f.Words.OnLayout(ui.AuthoredDialogueLayout()).WithPortrait(false)
	layout.Frame = f.gameMenuArt()
	want := ui.RenderNotice(layout, f.Font.Value(), body, nil)
	wrong := ui.RenderNotice(layout, f.Font.Value(), "Different words", nil)
	actual, note, err := app.HeadlessFrame()
	if err != nil || note != "" || actual == nil || want == nil || wrong == nil {
		t.Fatal("actual App frame did not compose the installed panel", note, err)
	}
	x0, y0 := (actual.Bounds().Dx()-want.Bounds().Dx())/2, (actual.Bounds().Dy()-want.Bounds().Dy())/2
	different := 0
	for y := 0; y < want.Bounds().Dy(); y++ {
		for x := 0; x < want.Bounds().Dx(); x++ {
			pixel := want.RGBAAt(x, y)
			if pixel.A != 255 {
				continue
			}
			if actual.RGBAAt(x+x0, y+y0) != pixel {
				t.Fatalf("actual App panel differs from literal %q at %d,%d", body, x, y)
			}
			if wrong.RGBAAt(x, y) != pixel {
				different++
			}
		}
	}
	if different == 0 {
		t.Fatal("installed font does not discriminate expected and wrong panel bodies")
	}
}

func TestReleaseOfferFirstPartSAVAndNextAction(t *testing.T) {
	for _, q := range []struct {
		name, door       string
		chapter, mission int
		building         TownBuilding
		room             townRoom
	}{{"shop", "SHOP", 30, 31, TownShop, roomShop}, {"school", "SCHOOL", 60, 61, TownSchool, roomSchool}} {
		for _, first := range []string{"absent", "rejected"} {
			t.Run(q.name+"/"+first, func(t *testing.T) {
				f := preparedOfferTown(t, q.chapter, q.mission, q.building)
				path, _ := TownBuildingTextPath(q.building, q.mission)
				payload := "<part=2>\r\nAccepted second part\r\n"
				if first == "rejected" {
					condition := "iamfemale"
					if HeroAudience(f.Carried).HeroFemale {
						condition = "iammale"
					}
					payload = "<part=1 " + condition + ">\r\nRejected first part\r\n" + payload
				}
				overlay := offerTextOverlay(t, f, []synth.File{{Path: strings.TrimPrefix(path, mainPrefix), Data: []byte(payload)}})
				initial := currentTownSave(t, f)
				initialCampaign := autoGetCampaign(t, initial)
				if initialCampaign.SelectedMission == uint32(q.mission) || offerRecord(t, initialCampaign, q.mission).Announced || len(initialCampaign.Markers) != 0 {
					t.Fatal("initial SAV already satisfies registration", initialCampaign.SelectedMission, initialCampaign.Markers)
				}
				f, app, s := coldOfferApp(t, initial, overlay)
				if got := f.Town.Offers(q.building); !reflect.DeepEqual(got, []TownOffer{{Index: 0, Mission: q.mission}, {Index: 1, Mission: q.mission}}) {
					t.Fatal("cold initial offers differ", got)
				}
				gold := f.Town.Gold()
				if err := app.HeadlessActivate(q.door); err != nil {
					t.Fatal(err)
				}
				wantOffer := []TownOffer{{Index: 1, Mission: q.mission}}
				if s.room != roomTalk || s.said != 1 || s.dialogue.displayPart != 0 || s.dialogue.text != "Nothing to say" {
					t.Fatal("failed first lookup dropped the default constructed panel", s.room, s.said, s.dialogue)
				}
				assertOfferAppPanel(t, f, app, "Nothing to say")
				if !reflect.DeepEqual(f.Town.Offers(q.building), wantOffer) || f.Town.selectedMission() != q.mission || !f.Town.progress.record(q.mission).announced {
					t.Fatal("normal builder return did not register independently of part 1", f.Town.Offers(q.building), f.Town.selectedMission())
				}
				dialogueKeyPress(t, app, s, "enter")
				if s.room != roomTalk || s.said != 2 || s.dialogue.displayPart != 2 || strings.TrimSpace(s.dialogue.text) != "Accepted second part" {
					t.Fatal("next App input did not accept part 2", s.room, s.said, s.dialogue)
				}
				assertOfferAppPanel(t, f, app, "Accepted second part\r\n")
				dialogueKeyPress(t, app, s, "escape")
				if s.room != q.room {
					t.Fatal("missing part 3 did not close to the building", s.room)
				}
				if err := app.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
				raw := saveInnApp(t, f, app)
				c := autoGetCampaign(t, raw)
				rows := c.ShopMission
				if q.building == TownSchool {
					rows = c.TCMission
				}
				if !reflect.DeepEqual(rows, []uint16{uint16(q.mission)}) || c.SelectedMission != uint32(q.mission) || !offerRecord(t, c, q.mission).Announced {
					t.Fatal("App SAVE lost ordinary offer, selection or announcement", rows, c.SelectedMission)
				}
				picture, hasPicture := nativeCityMarkerPicture(f, q.mission)
				wantMarkers := []sav.CampaignMarker(nil)
				if hasPicture {
					wantMarkers = []sav.CampaignMarker{{Value: uint32(q.mission), Picture: picture}}
				}
				if !reflect.DeepEqual(c.Markers, wantMarkers) {
					t.Fatal("App SAVE marker differs from the installed picture rule", c.Markers, wantMarkers)
				}
				g, gapp, gs := coldOfferApp(t, raw, overlay)
				if !reflect.DeepEqual(g.Town.Offers(q.building), wantOffer) || g.Town.selectedMission() != q.mission || !g.Town.progress.record(q.mission).announced || !offerMarkersEqual(g, wantMarkers) || g.Town.Gold() != gold {
					t.Fatal("cold App LOAD lost a registered obligation")
				}
				if _, ok := g.Town.Take(q.building, 0); ok {
					t.Fatal("cold App LOAD resurrected consumed stable index 0")
				}
				doc, err := sav.DecodeDocumentData(raw)
				if err != nil {
					t.Fatal(err)
				}
				if !hasPicture {
					priorPicture, ok := nativeCityMarkerPicture(f, 50)
					if !ok {
						t.Fatal("installed control mission has no marker picture")
					}
					doc.Campaign.Markers = []sav.CityCampaignMarkerData{{Value: 50, Text: append([]byte(priorPicture), 0)}}
					wantMarkers = []sav.CampaignMarker{{Value: 50, Picture: priorPicture}}
				}
				doc.Campaign.Markers[0].Tail = [8]byte{17, 0, 0, 0, 29, 0, 0, 0}
				raw, err = sav.EncodeDocumentData(doc)
				if err != nil {
					t.Fatal(err)
				}
				wantMarkers[0].Field0, wantMarkers[0].Field1 = 17, 29
				g, gapp, gs = coldOfferApp(t, raw, overlay)
				if err := gapp.HeadlessActivate(q.door); err != nil {
					t.Fatal(err)
				}
				if gs.room != roomTalk || gs.offer.Index != 1 || gs.offer.Mission != q.mission || len(g.Town.Offers(q.building)) != 0 || g.Town.selectedMission() != q.mission || !g.Town.progress.record(q.mission).announced || g.Town.Gold() != gold {
					t.Fatal("next city action did not consume only the remaining stable identity", gs.offer, g.Town.Offers(q.building))
				}
				dialogueKeyPress(t, gapp, gs, "click")
				if gs.room != roomTalk || gs.said != 2 || gs.dialogue.displayPart != 2 {
					t.Fatal("post-LOAD App button did not attempt part 2")
				}
				dialogueKeyPress(t, gapp, gs, "enter")
				if err := gapp.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
				next := autoGetCampaign(t, saveInnApp(t, g, gapp))
				if !reflect.DeepEqual(next.Markers, wantMarkers) {
					t.Fatal("next registration duplicated or replaced an existing marker", next.Markers, wantMarkers)
				}
				losses := []string{"offer", "index", "selected", "announce", "marker"}
				for _, loss := range losses {
					t.Run("loss="+loss, func(t *testing.T) {
						doc, err := sav.DecodeDocumentData(raw)
						if err != nil {
							t.Fatal(err)
						}
						switch loss {
						case "offer":
							i := 5
							if q.building == TownSchool {
								i = 4
							}
							doc.Campaign.Arrays[i] = []uint16{uint16(q.chapter)}
						case "index":
							a, err := readCurrentActions(&doc)
							if err != nil || a == nil || a.Session == nil || a.Session.OfferLabels == nil {
								t.Fatal("saved stable labels missing", err)
							}
							a.Session.OfferLabels.Buildings[q.building][0].Index = 0
							data, err := json.Marshal(a)
							if err != nil {
								t.Fatal(err)
							}
							if err := sav.SetNativeActions(&doc.State, data); err != nil {
								t.Fatal(err)
							}
						case "selected":
							doc.Campaign.Scalars[0] = uint32(q.chapter)
						case "announce":
							found := false
							for i := range doc.Campaign.Children {
								if doc.Campaign.Children[i].Base.DWords[0] == uint32(q.mission) {
									doc.Campaign.Children[i].Base.DWords[5], found = 0, true
								}
							}
							if !found {
								t.Fatal("announce loss control has no child")
							}
						case "marker":
							doc.Campaign.Markers = nil
						}
						changed, err := sav.EncodeDocumentData(doc)
						if err != nil {
							t.Fatal(err)
						}
						lost, _, _ := coldOfferApp(t, changed, overlay)
						detected := false
						switch loss {
						case "offer":
							detected = len(lost.Town.Offers(q.building)) == 1 && lost.Town.Offers(q.building)[0].Mission != q.mission
						case "index":
							detected = len(lost.Town.Offers(q.building)) == 1 && lost.Town.Offers(q.building)[0].Index != 1
						case "selected":
							detected = lost.Town.selectedMission() != q.mission
						case "announce":
							detected = !lost.Town.progress.record(q.mission).announced
						case "marker":
							detected = !offerMarkersEqual(lost, wantMarkers)
						}
						if !detected {
							t.Fatal("cold App LOAD did not discriminate the named saved loss", loss)
						}
					})
				}
				t.Logf("%s part1 %s: SAVE/cold App LOAD/next city action; remaining index 1; selected %d; announced; markers %+v; %d independent losses", q.name, first, q.mission, wantMarkers, len(losses))
			})
		}
	}
}
