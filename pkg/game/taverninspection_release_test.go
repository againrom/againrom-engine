package game

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

var shippedMercenaryTalkTypes = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 13, 14}

// expectedInstalledMercenaryTalkPicture builds the expected candidate picture
// directly from the installed Units/Humans rows and the shared picture
// composers. It does not call buildMercenarySquad, tavernCandidateDetail or
// speakerResolver, so a dialogue that substitutes the party/player picture
// cannot make its own expected result agree.
func expectedInstalledMercenaryTalkPicture(t *testing.T, f *FrontEnd, typ int) *image.RGBA {
	t.Helper()
	if typ <= 2 {
		name := "Catapult"
		if typ == 2 {
			name = "Ballista"
		}
		idx := data.NotFound
		for i := 1; i < f.Table.Units.Len(); i++ {
			if f.Table.Units.EntryName(i) == name {
				idx = i
				break
			}
		}
		if idx == data.NotFound {
			t.Fatalf("type %d expected unit %q is absent", typ, name)
		}
		def, err := data.NewUnitDef(name, f.Table.Units.EntryParams(idx))
		if err != nil {
			t.Fatalf("type %d expected unit %q: %v", typ, name, err)
		}
		class := f.Units.Classes[def.TypeID]
		if class == nil {
			t.Fatalf("type %d expected class %d is absent", typ, def.TypeID)
		}
		addr := graphicsPrefix + data.PortraitTierPath(class.Portrait, def.Face)
		t.Logf("type %d expected flat portrait %s class=%d face=%d", typ, addr, def.TypeID, def.Face)
		pic := loadPortrait(f.Archives.Containers, addr)
		if pic == nil {
			t.Fatalf("type %d expected flat portrait %q is unreadable", typ, addr)
		}
		return pic
	}

	template := fmt.Sprintf("NPC%02d_%d", typ, mercenaryLevel(f.Town.Chapter()))
	idx := data.FindHumanByName(f.Table.Humans, template)
	if idx == data.NotFound {
		t.Fatalf("type %d expected template %q is absent", typ, template)
	}
	def, err := data.NewHumanDef(template, f.Table.Humans.EntryParams(idx))
	if err != nil {
		t.Fatalf("type %d expected template %q: %v", typ, template, err)
	}
	dir, face := data.FigureFor(def.TypeID, def.Face, def.Gender)
	t.Logf("type %d expected figure %s/%d typeID=%d gender=%d", typ, dir, face, def.TypeID, def.Gender)
	wornItems, _, _, err := mapload.HumanRowEquipment(f.Table.Humans.EntryStrings(idx), f.Table)
	if err != nil {
		t.Fatalf("type %d expected template %q equipment: %v", typ, template, err)
	}
	var worn [sim.EquipSlots]uint16
	for slot, item := range wornItems {
		worn[slot] = item.Code
	}
	composed, _ := composeInventorySubject(f.Archives.Containers, uint32(typ),
		equipmentFromSlots(worn), dir, face)
	if composed.Figure == nil {
		t.Fatalf("type %d expected composed figure %s/%d is unreadable", typ, dir, face)
	}
	// Read the horse independently of addFigureHorse and its class predicate.
	// These are the two shipped mounted class ids in HERO-DOLL-078's range.
	if def.TypeID == 19 || def.TypeID == 21 {
		b, err := f.Archives.Containers.ReadFile("graphics/infowindow/horse.bmp")
		if err != nil {
			t.Fatal(err)
		}
		pic := inspectionPortraitBMP(t, b)
		draw.Draw(pic, pic.Bounds(), composed.Figure, composed.Figure.Bounds().Min, draw.Over)
		return pic
	}
	return composed.Figure
}

// TestReleaseEveryShippedMercenaryPriceFitsAndMatchesTheHireAction is the
// complete card-price census. The campaign contains 107 mercenary rows. The
// two pre-town rows have a zero unit factor and are not actionable; the other
// 105 use mercenarySquadPrice, the same producer used by affordability and
// hire/return. Every row is still composed so the production font and card
// composer are checked against the complete shipped value population.
func TestReleaseEveryShippedMercenaryPriceFitsAndMatchesTheHireAction(t *testing.T) {
	f := releaseFront(t)
	font := f.tipFont()
	if font == nil {
		t.Fatal("the production compact-card font did not resolve")
	}
	if f.TownTavernArt.Value() == nil || f.TownTavernArt.Value().ManBack == nil {
		t.Fatalf("production tavern card art did not resolve: %v", f.TownTavernArt.Err())
	}

	const cardWidth, cardHeight = 48, 64
	cardText := color.RGBA{R: 0xbd, G: 0x9e, B: 0x4a, A: 0xff}
	rows, actionable, sevenDigit := 0, 0, 0
	maxWidth, maxMission, maxType, maxPrice := 0, 0, 0, 0
	for mission, chapter := range f.Campaign.Value().Chapters {
		for _, typ := range chapter.Mercenaries {
			rows++
			if typ <= 0 || typ > len(f.Campaign.Value().MercenaryCount) {
				t.Fatalf("mission %d row %d has no shipped pool slot", mission, typ)
			}
			count := f.Campaign.Value().MercenaryCount[typ-1]
			terms, ok := f.Table.NPC.Mercenary(int32(typ))
			if !ok || count <= 0 {
				t.Fatalf("mission %d type %d has terms=%v count=%d", mission, typ, ok, count)
			}
			unit := mercenaryUnitPrice(mission)
			price := (int(terms.PriceA) + count*int(terms.PriceB)) * unit
			if unit > 0 {
				got, ok := mercenarySquadPrice(f.Table, mission, typ, count)
				if !ok || got != price {
					t.Fatalf("mission %d type %d action price = %d, %v; want %d", mission, typ, got, ok, price)
				}
				actionable++
			} else if got, ok := mercenarySquadPrice(f.Table, mission, typ, count); ok || got != 0 {
				t.Fatalf("mission %d type %d locked row became actionable at %d", mission, typ, got)
			}

			priceText := ui.GroupDigits(int64(price))
			detailText := fmt.Sprintf("%d/%d", count, count)
			priceWidth, _ := font.Measure(priceText)
			detailWidth, _ := font.Measure(detailText)
			if priceWidth > maxWidth {
				maxWidth, maxMission, maxType, maxPrice = priceWidth, mission, typ, price
			}
			if priceWidth > cardWidth || detailWidth > cardWidth {
				t.Errorf("mission %d type %d card text %q/%q measures %d/%dpx, past %dpx",
					mission, typ, priceText, detailText, priceWidth, detailWidth, cardWidth)
			}
			priceBox := image.Rect(cardWidth-priceWidth, 0, cardWidth, font.Height())
			detailBox := image.Rect(0, cardHeight-font.Height(), detailWidth, cardHeight)
			if !priceBox.In(image.Rect(0, 0, cardWidth, cardHeight)) ||
				!detailBox.In(image.Rect(0, 0, cardWidth, cardHeight)) ||
				!priceBox.Intersect(detailBox).Empty() {
				t.Errorf("mission %d type %d text geometry price=%v count=%v", mission, typ, priceBox, detailBox)
			}

			view := ui.TownSurfaceView{Kind: ui.TownSurfaceTavern, Font: f.Font.Value(), CardFont: font,
				TavernArt: f.TownTavernArt.Value(),
				Cells:     []ui.TownSurfaceCell{{Portrait: true, Price: priceText, Detail: detailText}}}
			got := ui.ComposeTownSurface(view)
			want := image.NewRGBA(image.Rect(0, 0, cardWidth, cardHeight))
			draw.Draw(want, want.Bounds(), f.TownTavernArt.Value().ManBack, f.TownTavernArt.Value().ManBack.Bounds().Min, draw.Src)
			font.Draw(want, priceText, cardWidth-priceWidth, 0, cardText)
			font.Draw(want, detailText, 0, cardHeight-font.Height(), cardText)
			if !equalTranslated(want, got, image.Pt(176, 416)) {
				t.Errorf("mission %d type %d card did not compose exact price %q and count %q at the 48x64 corners",
					mission, typ, priceText, detailText)
			}
			if price >= 1_000_000 {
				sevenDigit++
				if mission != 150 || typ != 13 || price != 1_000_000 {
					t.Errorf("unexpected seven-digit row mission %d type %d price %d", mission, typ, price)
				}
			}
		}
	}
	if rows != 107 || actionable != 105 || sevenDigit != 1 {
		t.Fatalf("shipped price census rows/actionable/seven-digit = %d/%d/%d, want 107/105/1", rows, actionable, sevenDigit)
	}
	if maxMission != 150 || maxType != 13 || maxPrice != 1_000_000 {
		t.Fatalf("widest price is mission %d type %d price %d at %dpx, want mission 150 type 13 price 1000000",
			maxMission, maxType, maxPrice, maxWidth)
	}

	// Drive the widest row through the live cell, selected Hire button and
	// action. All three must retain the same seven-digit value.
	town := NewTown(f.Campaign.Value())
	for _, mission := range f.Campaign.Value().Main {
		if mission < 150 {
			town.Won(mission)
		}
	}
	town.open = true
	town.gold = 2_000_000
	town.mercEnabled[13] = true
	f.Town = town
	s := f.TownScreen().(*townScreen)
	s.room = roomTavern
	candidates := s.tavernCandidates()
	found := false
	for _, candidate := range candidates {
		if candidate.key != (tavernCandidateKey{kind: tavernCandidateMercenary, id: 13}) {
			continue
		}
		found = true
		if candidate.cell.Price != "1,000,000" || candidate.merc.Price != 1_000_000 {
			t.Fatalf("mission 150 type 13 cell/action price = %q/%d, want 1,000,000", candidate.cell.Price, candidate.merc.Price)
		}
		s.tavernSelection = candidate.key
	}
	if !found {
		t.Fatal("mission 150 type 13 did not reach the production candidate stream")
	}
	surface := s.TownSurface()
	if surface.CardFont != font {
		t.Fatal("the live tavern surface did not carry the measured compact-card font")
	}
	if len(surface.Buttons) <= tavernButtonHire || surface.Buttons[tavernButtonHire].Value != "1,000,000" {
		t.Fatalf("selected Hire button price = %q, want 1,000,000", surface.Buttons[tavernButtonHire].Value)
	}
	before := town.Gold()
	if _, ok := s.toggleMercenary(13); !ok {
		t.Fatal("mission 150 type 13 production hire refused")
	}
	if spent := before - town.Gold(); spent != 1_000_000 {
		t.Fatalf("mission 150 type 13 hire spent %d, want displayed 1000000", spent)
	}
	t.Logf("107 shipped rows composed; widest mission %d type %d price %d measures %d/%dpx",
		maxMission, maxType, maxPrice, maxWidth, cardWidth)
}

// TestReleaseEveryMercenaryUsesInstalledTalkPortraitAndInspectionArt runs once
// for each lawful EN/RU root selected by the release gate. It enumerates the
// whole candidate population rather than sampling one convenient type.
func TestReleaseEveryMercenaryUsesInstalledTalkPortraitAndInspectionArt(t *testing.T) {
	f := releaseFront(t)
	if f.TownTavernArt.Value() == nil {
		t.Fatalf("production tavern art did not resolve: %v", f.TownTavernArt.Err())
	}
	f.Carried = f.NextParty()
	s := f.TownScreen().(*townScreen)
	s.room = roomTavern
	installedTypes := make(map[int]bool)
	for _, chapter := range f.Campaign.Value().Chapters {
		for _, typ := range chapter.Mercenaries {
			installedTypes[typ] = true
		}
	}
	if len(installedTypes) != len(shippedMercenaryTalkTypes) {
		t.Fatalf("installed candidate type population = %v, want %v", installedTypes, shippedMercenaryTalkTypes)
	}
	art := f.TownTavernArt.Value()
	for _, typ := range tavernTalkSheets {
		if len(art.UnitFrames[typ]) == 0 {
			t.Fatalf("shipped talk sheet Unit%d did not load", typ)
		}
	}
	if len(art.HeroFrames[0]) == 0 || len(art.HeroFrames[1]) == 0 {
		t.Fatal("shipped HeroFighter/HeroMage talk sheets did not load")
	}
	// TAVERN-TALKPIC-016 over every InnNPC element at its own stage, with no
	// party: the Hero records take the synthesised arm, npc90 and npc59 the
	// live stock mercenary their record admits (Medium in the claim), and the
	// rest their own Unit<id> sheet.
	heroSheet := map[int]int{22: 1, 23: 0, 25: 0}
	stockType := map[int]int{90: 10, 59: 8}
	maxTalkProducers := 0
	carried := f.Carried
	f.Carried = nil
	for _, mission := range f.Campaign.Value().Main {
		ch := f.Campaign.Value().Chapters[mission]
		if len(ch.InnNPC) == 0 {
			continue
		}
		town := NewTown(f.Campaign.Value())
		for _, prior := range f.Campaign.Value().Main {
			if prior >= mission {
				break
			}
			town.Won(prior)
		}
		if town.Chapter() != mission {
			t.Fatalf("stage %d: town reached chapter %d", mission, town.Chapter())
		}
		f.Town = town
		ts := f.TownScreen().(*townScreen)
		if len(ch.InnNPC) > maxTalkProducers {
			maxTalkProducers = len(ch.InnNPC)
		}
		for _, npc := range ch.InnNPC {
			o := ts.tavernTalkObject(npc)
			frames := tavernTalkFrames(art, o)
			hero, isHero := heroSheet[npc]
			typ, isStock := stockType[npc]
			switch {
			case isHero:
				if o.live >= 0 || len(frames) == 0 || &frames[0] != &art.HeroFrames[hero][0] {
					t.Fatalf("stage %d npc%d = %+v, want the synthesised Hero sheet %d", mission, npc, o, hero)
				}
			case isStock:
				if o.stock != typ || !o.statistics() || len(frames) == 0 || &frames[0] != &art.UnitFrames[typ][0] {
					t.Fatalf("stage %d npc%d = %+v, want the live stock unit of type %d on Unit%d", mission, npc, o, typ, typ)
				}
			default:
				if o.live >= 0 || len(frames) == 0 || &frames[0] != &art.UnitFrames[npc][0] {
					t.Fatalf("stage %d npc%d = %+v, want the synthesised object on Unit%d", mission, npc, o, npc)
				}
			}
		}
	}
	f.Carried = carried
	if maxTalkProducers < 3 {
		t.Fatalf("installed talk-only producer census reached only %d; want the shipped three-offer chapter covered", maxTalkProducers)
	}
	for _, typ := range shippedMercenaryTalkTypes {
		if !installedTypes[typ] {
			t.Fatalf("installed campaign never produces candidate type %d", typ)
		}
	}

	for _, typ := range shippedMercenaryTalkTypes {
		t.Run(fmt.Sprintf("type-%02d", typ), func(t *testing.T) {
			path, ok := TownMercenaryTextPath(typ)
			if !ok {
				t.Fatalf("type %d has no source mapping", typ)
			}
			payload, err := f.Archives.Containers.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			line, ok := expectedInstalledDialoguePart(t, payload, 1, HeroAudience(f.Carried))
			if !ok || line == "" {
				t.Fatalf("%s has no authored part 1", path)
			}
			speaker, named := EventPartSpeaker(payload, 1, HeroAudience(f.Carried))
			if !named {
				t.Fatalf("%s part 1 names no portrait speaker", path)
			}

			s.openMercenaryDialogue(typ)
			if s.talkDiagnostic != "" || s.mercenaryTalk != typ || s.room != roomTalk {
				t.Fatalf("open type %d = room %v source %d diagnostic %q", typ, s.room, s.mercenaryTalk, s.talkDiagnostic)
			}
			face := expectedInstalledMercenaryTalkPicture(t, f, typ)
			var win image.Rectangle
			rec, found := f.NPCFaces[int32(speaker)]
			if !found {
				t.Fatalf("%s speaker %d has no installed portrait record", path, speaker)
			}
			if rec.HasWindow {
				win = ui.NoticeFaceWindow(rec.WindowX, rec.WindowY)
			}
			layout, modalPayload, ok := s.townDialogueLayout()
			if !ok || !reflect.DeepEqual(modalPayload, payload) {
				t.Fatalf("modal source for type %d is not %s", typ, path)
			}
			got, ok := s.TownDialogue()
			if !ok || got == nil {
				t.Fatalf("type %d produced no modal", typ)
			}
			want := ui.RenderNotice(layout.WithFaceWindow(win), f.Font.Value(), line, face)
			if !imagesEqual(got, want) {
				t.Fatalf("type %d modal is not its installed text plus independently composed candidate picture", typ)
			}
			s.room, s.mercenaryTalk, s.talkDiagnostic = roomTavern, 0, ""

			frames := f.TownTavernArt.Value().UnitFrames[typ]
			if len(frames) < 2 {
				t.Fatalf("type %d frames = %d, want at least two", typ, len(frames))
			}
			if frames[0].Bounds().Dx() != 48 || frames[0].Bounds().Dy() != 64 {
				t.Fatalf("type %d first frame bounds = %v, want 48x64", typ, frames[0].Bounds())
			}
			v := ui.TownSurfaceView{Kind: ui.TownSurfaceTavern, TavernArt: f.TownTavernArt.Value(),
				Cells: []ui.TownSurfaceCell{
					{Portrait: true, Selected: true, Frames: frames},
					{Portrait: true, Frames: frames},
				}}
			phase0 := ui.ComposeTownSurface(v)
			v.AnimationFrame = 1
			phase1 := ui.ComposeTownSurface(v)
			if equalRegion(phase0, phase1, image.Rect(176, 416, 224, 480)) {
				t.Fatalf("type %d selected card did not advance to shipped frame 1", typ)
			}
			if !equalRegion(phase0, phase1, image.Rect(224, 416, 272, 480)) {
				t.Fatalf("type %d unselected card consumed animation phase", typ)
			}
			expected := image.NewRGBA(image.Rect(0, 0, 48, 64))
			draw.Draw(expected, expected.Bounds(), f.TownTavernArt.Value().ManBack, f.TownTavernArt.Value().ManBack.Bounds().Min, draw.Src)
			draw.Draw(expected, expected.Bounds(), frames[0], frames[0].Bounds().Min, draw.Over)
			if !equalTranslated(expected, phase0, image.Pt(176, 416)) {
				t.Fatalf("type %d phase-0 card changed shipped palette, scale, crop or alignment", typ)
			}

			candidate, mask, info, ok := s.tavernCandidateDetail(typ)
			if !ok || !candidate.HasSubject {
				t.Fatalf("type %d produced no actual stat projection", typ)
			}
			if !imagesEqual(candidate.Figure, face) {
				t.Fatalf("type %d inspection did not use the independently expected actor portrait", typ)
			}
			if typ <= 2 && mask != nil {
				t.Fatal("siege portrait acquired a human equipment mask")
			}
			// Optional owner-only render proof. The release gate otherwise writes
			// no installed art and compares the production pixels in memory.
			if out := os.Getenv("AGAINROM_PORTRAIT_OUTPUT"); out != "" {
				if !filepath.IsAbs(out) {
					t.Fatal("portrait output must be absolute")
				}
				if err := os.MkdirAll(out, 0o755); err != nil {
					t.Fatal(err)
				}
				file, err := os.Create(filepath.Join(out, fmt.Sprintf("candidate-%02d.png", typ)))
				if err != nil {
					t.Fatal(err)
				}
				pic := ui.ComposeTownSurface(ui.TownSurfaceView{Kind: ui.TownSurfaceTavern,
					TavernArt: f.TownTavernArt.Value(), Candidate: candidate, CandidatePixels: s.tavernDetailPixels, Font: f.Font.Value()})
				encodeErr := png.Encode(file, pic)
				closeErr := file.Close()
				if encodeErr != nil || closeErr != nil {
					t.Fatalf("portrait PNG: %v / %v", encodeErr, closeErr)
				}
			}
			members, ok := s.buildMercenarySquad(typ, 1)
			if !ok || len(members) != 1 {
				t.Fatalf("type %d production hire template is unavailable", typ)
			}
			expectedSubject := partyPanelSubject(members[0], f.Table, f.Words)
			nameIndex := int(members[0].Class)
			if typ > 2 {
				template := fmt.Sprintf("NPC%02d_%d", typ, mercenaryLevel(f.Town.Chapter()))
				if i := data.FindHumanByName(f.Table.Humans, template); i != data.NotFound {
					if def, err := data.NewHumanDef(template, f.Table.Humans.EntryParams(i)); err == nil {
						nameIndex = int(def.TypeID)
					}
				}
			}
			expectedSubject.UnitNameIndex, expectedSubject.Char.UnitNameIndex = nameIndex, nameIndex
			expectedSubject.Name, expectedSubject.Char.Name = "", ""
			if name, named := ui.PanelSubjectName(expectedSubject); named {
				expectedSubject.Name, expectedSubject.Char.Name = name, name
			}
			expectedSubject.Role = f.Words.PanelCaptions[mainSlotMercenary]
			expectedSubject.Unplaced, expectedSubject.Selected = true, 1
			// HERO-SIGHT-007's word, ftol(((mind+reaction)/25 + 4) x 256), with
			// the whole cells the derived sight carries.
			if c := expectedSubject.Char; c.Mind+c.Reaction > 0 {
				word := int(float64(c.Mind+c.Reaction)/25*256 + 4*256)
				expectedSubject.Char.Sight256 = uint16(c.Sight<<8 | word&0xff)
			}
			if !reflect.DeepEqual(candidate.Subject, expectedSubject) {
				t.Errorf("type %d candidate statistics differ from the production hire template", typ)
			}
			if name, named := ui.PanelSubjectName(candidate.Subject); !named || name == "" {
				t.Errorf("type %d candidate statistics have no installed display name", typ)
			}
			items := mapload.MemberItemEquipment(members[0], f.Table)
			view := ui.TownSurfaceView{Kind: ui.TownSurfaceTavern, Candidate: candidate,
				CandidateSlotMask: mask, CandidateSlotInfo: info}
			for slot, item := range items {
				if item.Code == 0 {
					continue
				}
				wantLines := itemInstanceInfoLines(item, f.Table, f.Words)
				if !reflect.DeepEqual(info[slot], wantLines) {
					t.Errorf("type %d slot %d tooltip = %q, want normal full lines %q", typ, slot+1, info[slot], wantLines)
				}
				p, found := maskPoint(mask, uint8(slot+1))
				if !found {
					eq := mapload.EquipmentFromParty(members[0])
					dir, face := memberFigure(members[0])
					_, unread := composeInventorySubject(f.Archives.Containers, uint32(typ), eq, dir, face)
					t.Errorf("type %d worn slot %d code %d (%q) has no composed doll pixel; unread=%q", typ, slot+1, item.Code, wantLines, unread)
					continue
				}
				p = p.Add(image.Pt((160-mask.W)/2+8, 238+(242-mask.H)/2+1))
				if lines, ok := ui.TownCandidateHoverLines(view, p); !ok || !reflect.DeepEqual(lines, wantLines) {
					t.Errorf("type %d slot %d production hover = %q, %v; want %q", typ, slot+1, lines, ok, wantLines)
				}
			}
		})
	}
}

// TestReleaseEveryReachableMercenaryWeaponReachesStatsAndDoll is the exact
// Humans-row population the live tavern can construct over all four level
// bands. The old builder copied the row and then deliberately replaced cell
// zero with an empty generated weapon, making every authored weapon disappear
// from combat statistics and from the inspection doll. This census starts at
// the installed campaign's main chapters, not at a remembered type list, and
// checks every distinct reachable template once. Empty authored weapon cells
// remain empty; a hotfix must not invent equipment the row does not name.
func TestReleaseEveryReachableMercenaryWeaponReachesStatsAndDoll(t *testing.T) {
	f := releaseFront(t)
	seen := make(map[string]bool)
	templates, authoredWeapons, magicStaffs, mageStaffs := 0, 0, 0, 0
	for _, mission := range f.Campaign.Value().Main {
		chapter := f.Campaign.Value().Chapters[mission]
		if len(chapter.Mercenaries) == 0 {
			continue
		}
		town := NewTown(f.Campaign.Value())
		for _, prior := range f.Campaign.Value().Main {
			if prior >= mission {
				break
			}
			town.Won(prior)
		}
		if town.Chapter() != mission {
			// Missions 10 and 20 are the pre-town campaign stretch. Their
			// scenario rows may name stock, but no tavern can present it.
			continue
		}
		f.Town = town
		s := f.TownScreen().(*townScreen)
		s.room = roomTavern
		for _, typ := range chapter.Mercenaries {
			if typ <= 2 {
				continue
			}
			template := fmt.Sprintf("NPC%02d_%d", typ, mercenaryLevel(mission))
			if seen[template] {
				continue
			}
			seen[template] = true
			templates++
			row := data.FindHumanByName(f.Table.Humans, template)
			if row == data.NotFound {
				t.Fatalf("mission %d type %d template %s is absent", mission, typ, template)
			}
			cells := f.Table.Humans.EntryStrings(row)
			members, ok := s.buildMercenarySquad(typ, 1)
			if !ok || len(members) != 1 {
				t.Fatalf("mission %d type %d production template is unavailable", mission, typ)
			}
			member := members[0]
			if len(cells) == 0 || cells[0] == "" {
				if member.Weapon != nil || member.Worn[0] != 0 || member.WornItems[0].Code != 0 {
					t.Fatalf("mission %d type %d bare template %s invented weapon %+v/0x%04x/0x%04x",
						mission, typ, template, member.Weapon, member.Worn[0], member.WornItems[0].Code)
				}
				continue
			}
			authoredWeapons++
			if member.Weapon == nil || member.Worn[0] == 0 || member.WornItems[0].Code != member.Worn[0] {
				t.Fatalf("mission %d type %d weapon %q reached pointer/code/item = %v/0x%04x/0x%04x",
					mission, typ, cells[0], member.Weapon, member.Worn[0], member.WornItems[0].Code)
			}
			if code, present := mapload.EquipmentFromParty(member).Code(1); !present || uint16(code) != member.Worn[0] {
				t.Fatalf("mission %d type %d doll slot 1 = 0x%04x, %v; want 0x%04x",
					mission, typ, uint16(code), present, member.Worn[0])
			}
			candidate, mask, info, ok := s.tavernCandidateDetail(typ)
			if !ok || !candidate.HasSubject || candidate.Subject.Char.Weapon == "" || len(info[0]) == 0 {
				t.Fatalf("mission %d type %d stats/tooltip omitted authored weapon %q", mission, typ, cells[0])
			}
			if _, found := maskPoint(mask, 1); !found {
				t.Fatalf("mission %d type %d doll omitted authored weapon %q", mission, typ, cells[0])
			}
			wantLines := itemInstanceInfoLines(member.WornItems[0], f.Table, f.Words)
			if !reflect.DeepEqual(info[0], wantLines) {
				t.Fatalf("mission %d type %d weapon tooltip = %q, want %q", mission, typ, info[0], wantLines)
			}
			if member.Weapon.SpellName == "" {
				continue
			}
			magicStaffs++
			if member.Mage {
				mageStaffs++
			}
			foundCast := false
			for _, effect := range member.WornItems[0].Effects {
				foundCast = foundCast || effect.Kind == 41
			}
			if !foundCast || member.Weapon.SpellPower == 0 {
				t.Fatalf("mission %d type %d magic staff %q lost castSpell state: %+v effects=%+v",
					mission, typ, cells[0], member.Weapon, member.WornItems[0].Effects)
			}
		}
	}
	if templates != 33 || authoredWeapons != 32 || magicStaffs != 7 || mageStaffs != 7 {
		t.Fatalf("reachable template/weapon/magic-staff/mage-staff population = %d/%d/%d/%d, want 33/32/7/7",
			templates, authoredWeapons, magicStaffs, mageStaffs)
	}
	t.Logf("checked %d reachable templates: %d authored weapons, %d magic staffs", templates, authoredWeapons, magicStaffs)
}

func equalRegion(a, b *image.RGBA, r image.Rectangle) bool {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				return false
			}
		}
	}
	return true
}

func equalTranslated(want, got *image.RGBA, at image.Point) bool {
	for y := 0; y < want.Bounds().Dy(); y++ {
		for x := 0; x < want.Bounds().Dx(); x++ {
			if want.RGBAAt(x, y) != got.RGBAAt(at.X+x, at.Y+y) {
				return false
			}
		}
	}
	return true
}

func maskPoint(mask *ui.SlotMask, slot uint8) (image.Point, bool) {
	if mask == nil {
		return image.Point{}, false
	}
	for y := 0; y < mask.H; y++ {
		for x := 0; x < mask.W; x++ {
			if mask.Slot[y*mask.W+x] == slot {
				return image.Pt(x, y), true
			}
		}
	}
	return image.Point{}, false
}
