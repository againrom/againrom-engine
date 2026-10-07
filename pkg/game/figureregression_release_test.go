package game

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/draw"
	"os"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Figure regression suite. Every audit below reads what decides a picture: the
// art class a unit resolves to, the composed figure its doll and dialogue use,
// the tier palette of a creature. Each is checked against what the shipped data
// and the promoted claims select (HERO-FIGURE-144, DLG-SPEAKER-022,
// TAVERN-TALKPIC-016, UNIT-APPEAR-030, HERO-DOLL-078, REG-NPC-088), in a fresh
// mission and again after SAVE and a cold LOAD. An audit returns its findings
// as text, so a loss control can hand it a known-defective state and require
// the finding.

type figureHero struct {
	name       string
	sex, class int
}

// figureHeroes covers the four primary-hero directories: the cape belongs to
// the two fighters and never to the two mages.
var figureHeroes = []figureHero{
	{"Man fighter", 0, 0}, {"Woman fighter", 1, 0}, {"Man mage", 0, 1}, {"Woman mage", 1, 1},
}

// figureMissions samples the shipped campaign across its chapters, with the
// missions that hold a placed Hero (40) and a placed Lancer (71, 131).
func figureMissions() []int {
	return []int{10, 30, 40, 41, 60, 70, 71, 90, 100, 120, 131, 140, 150}
}

type figureRig struct {
	t       *testing.T
	f       *FrontEnd
	app     *ui.App
	mission int
	hero    figureHero
	hired   int
	stats   map[string]int
}

// openFigureMission builds the party through the town's own routes (a chosen
// hero, the chapter's companion grants, Brian from chapter 40 on, and every
// squad the tavern will hire) and opens the mission with it.
func openFigureMission(t *testing.T, mission int, hero figureHero) *figureRig {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	result := ui.ChargenResult{Name: hero.name, Choices: []int{hero.sex, hero.class, 3}, Stats: []int{31, 27, 24, 29}}
	generated := f.ChargenParty(result)
	if len(generated) != 1 {
		t.Fatalf("ChargenParty length = %d, want 1", len(generated))
	}
	town := NewTown(f.Campaign.Value())
	var earlier []int
	for number := range f.Campaign.Value().Chapters {
		if number < mission {
			earlier = append(earlier, number)
		}
	}
	slices.Sort(earlier)
	for _, number := range earlier {
		town.Won(number)
	}
	t.Logf("mission %d: chapter %d after winning %v", mission, town.Chapter(), earlier)
	town.gold = 50_000_000
	f.Town = town
	f.Carried = generated
	hired := 0
	// Missions 10 and 20 are entered before the town opens: the party is the
	// hero alone, and the town offers no companion and no squad yet.
	if mission >= 30 {
		f.arriveInTown()
		s, ok := f.TownScreen().(*townScreen)
		if !ok {
			t.Fatal("no town screen")
		}
		f.addChapterCompanions(f.Town.Chapter())
		if mission >= 40 {
			brian, ok := mapload.CampaignNPCMember(f.Table, 25, 0, f.Carried)
			if !ok {
				t.Fatal("no Brian to carry")
			}
			f.Carried = append(f.Carried, brian)
		}
		// The tavern's own squad builder, once per type; the town-state audit
		// hires through the tavern's controls.
		s.room = roomTavern
		for typ := 1; typ <= 15; typ++ {
			members, ok := s.buildMercenarySquad(typ, 1)
			if !ok {
				continue
			}
			f.Carried = mapload.OwnParty(append(f.Carried, members...))
			hired++
		}
		if hired < 10 {
			t.Fatalf("the tavern hired %d squads, want at least 10", hired)
		}
	}
	// The world map's marker click selects the mission to open.
	f.Town.selectMission(mission)
	r := &figureRig{t: t, f: f, mission: mission, hero: hero, hired: hired, stats: map[string]int{}}
	r.app = f.App("figure regression")
	t.Cleanup(r.app.StopAudio)
	r.app.Layout(1024, 768)
	if err := r.app.OpenMission(f.MissionOpenerWith(mission, f.Carried)); err != nil {
		t.Fatalf("open mission %d: %v", mission, err)
	}
	return r
}

// coldLoad SAVEs the open mission with the ordinary producer and opens the
// bytes in a cold front end.
func (r *figureRig) coldLoad() *figureRig {
	t := r.t
	t.Helper()
	snap, label, err := r.f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	saved, _, err := r.f.playerMissionSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	g := &FrontEnd{InstallResources: r.f.InstallResources}
	g.SetDeterministicFrames(true)
	next := &figureRig{t: t, f: g, mission: r.mission, hero: r.hero, hired: r.hired, stats: map[string]int{}}
	next.app = g.App("figure regression cold")
	t.Cleanup(next.app.StopAudio)
	next.app.Layout(1024, 768)
	reopened, _, err := g.RestoreOriginal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if err := next.app.OpenMission(reopened); err != nil {
		t.Fatal(err)
	}
	return next
}

func (r *figureRig) drawsByID() map[sim.EntityID]ui.MapEntity {
	out := map[sim.EntityID]ui.MapEntity{}
	for _, d := range r.f.live.entityDraws() {
		out[sim.EntityID(d.ID)] = d
	}
	return out
}

func isPCBody(units *terrain.UnitSet, class *terrain.UnitClass) bool {
	if class == nil {
		return false
	}
	for _, body := range units.Bodies {
		if body == class {
			return true
		}
	}
	return false
}

func sameBytes(a, b []byte) bool { return bytes.Equal(a, b) }

// The oracle compositions are memoised across the whole run: a squad of four
// shares one picture, and the install does not change under a test process.
type oracleKey struct {
	root string
	fig  figureID
	eq   data.Equipment
}

var (
	oracleFigures = map[oracleKey]*image.RGBA{}
	oracleCapes   = map[string]*image.RGBA{}
)

// oracleFigure is the compositor's own picture for a figure and a worn set.
func oracleFigure(src entrySource, fig figureID, eq data.Equipment) *image.RGBA {
	key := oracleKey{os.Getenv("AGAINROM_ASSETS"), fig, eq}
	if pic, ok := oracleFigures[key]; ok {
		return pic
	}
	pic, _ := composeUnitFigure(src, eq, fig)
	oracleFigures[key] = pic
	return pic
}

// oracleCaped paints the original warrior background under a figure, from the
// installed sheet read by heroBackgroundExpected.
func oracleCaped(t *testing.T, src entrySource, dir data.FigureDir, front *image.RGBA) *image.RGBA {
	t.Helper()
	key := fmt.Sprint(os.Getenv("AGAINROM_ASSETS"), dir.Female())
	back, ok := oracleCapes[key]
	if !ok {
		back = heroBackgroundExpected(t, src, dir, image.NewRGBA(front.Bounds()))
		oracleCapes[key] = back
	}
	out := image.NewRGBA(front.Bounds())
	draw.Draw(out, out.Bounds(), back, back.Bounds().Min, draw.Src)
	draw.Draw(out, out.Bounds(), front, front.Bounds().Min, draw.Over)
	return out
}

// auditPartyFigures checks every living party member's map art, doll,
// inventory standing and information-card flag.
func auditPartyFigures(r *figureRig) []string {
	t, f := r.t, r.f
	t.Helper()
	mw, src := f.live, f.Archives.Containers
	var bad []string
	report := func(format string, args ...any) { bad = append(bad, fmt.Sprintf(format, args...)) }
	draws := r.drawsByID()
	heroes, hires := 0, 0
	for i, p := range mw.mission.party {
		id := mw.mission.ids[i]
		e, held := mw.entity(id)
		d, drawn := draws[id]
		if !held || !drawn || !e.Alive() {
			continue
		}
		who := fmt.Sprintf("%s (party %d, entity %d)", p.Name, i, id)
		eq := mw.equipmentOf(id)
		got := mw.unitFigure(id)
		if !p.Hired() {
			heroes++
			// A player's character is drawn from the PC body its worn set
			// names, never from an NPC class record.
			body := mw.units.Bodies[data.HeroBodyKey(p.BodyDir, data.HeroBody(p.Body))]
			switch {
			case body == nil:
				report("%s: the hero has no PC body %q/%q", who, p.BodyDir, p.Body)
			case d.Art != body:
				report("%s: hero drawn as NPC class %q, not its PC body %q", who, artName(d.Art), body.Name)
			case !isPCBody(mw.units, d.Art):
				report("%s: hero art is not a PC body", who)
			}
			if !d.PlayerCharacter {
				report("%s: hero lacks the player-character flag", who)
			}
			dir, face := memberFigure(p)
			want := oracleFigure(src, figureID{Dir: dir, Face: face}, eq)
			if want == nil {
				report("%s: no composed figure for %s/%d", who, dir, face)
				continue
			}
			if !dir.Mage() {
				capeless := want
				want = oracleCaped(t, src, dir, capeless)
				if sameBytes(want.Pix, capeless.Pix) {
					report("%s: fixture hides the whole cape", who)
				}
			}
			if got == nil || !sameBytes(got.Pix, want.Pix) {
				report("%s: hero doll is not the %s figure (warrior cape %t)", who, dir, !dir.Mage())
			}
			continue
		}
		hires++
		// A hired mercenary is drawn as its own unit class, never as a PC body,
		// never with the hero cape, and carries nothing in a pack.
		class := mw.units.Classes[p.Class]
		switch {
		case class == nil:
			report("%s: hired class %d is absent", who, p.Class)
		case d.Art != class:
			report("%s: hired unit drawn as %q, not its own class %q", who, artName(d.Art), class.Name)
		}
		if isPCBody(mw.units, d.Art) {
			report("%s: hired unit drawn on a PC body", who)
		}
		if d.PlayerCharacter {
			report("%s: hired unit carries the player-character flag", who)
		}
		if pack, _ := mw.world.Carried(id); len(pack) != 0 {
			report("%s: hired unit holds a pack of %d items", who, len(pack))
		}
		if !data.ComposesFigure(p.Class) {
			if got != nil {
				report("%s: siege unit has a composed doll", who)
			}
			continue
		}
		if worn := mapload.EquipmentFromParty(p); worn != eq {
			report("%s: hired unit wears %v, its template says %v", who, eq, worn)
		}
		dir, face := memberFigure(p)
		horse := data.FigureHasHorse(p.Class)
		want := oracleFigure(src, figureID{Dir: dir, Face: face, Horse: horse}, eq)
		plain := oracleFigure(src, figureID{Dir: dir, Face: face}, eq)
		if got == nil || want == nil || !sameBytes(got.Pix, want.Pix) {
			report("%s: hired doll is not its own %s figure (horse %t)", who, dir, horse)
			continue
		}
		if horse == sameBytes(want.Pix, plain.Pix) {
			report("%s: horse rider state %t but the doll differs from the plain figure %t", who, horse, !sameBytes(want.Pix, plain.Pix))
		}
		if !dir.Mage() {
			if caped := oracleCaped(t, src, dir, plain); !sameBytes(caped.Pix, plain.Pix) && sameBytes(got.Pix, caped.Pix) {
				report("%s: hired unit is drawn with the hero cape", who)
			}
		}
	}
	if heroes == 0 || hires < r.hired {
		report("party audit saw %d heroes and %d hired units, want a hero and %d hires", heroes, hires, r.hired)
	}
	return bad
}

func artName(c *terrain.UnitClass) string {
	if c == nil {
		return "<nil>"
	}
	return c.Name
}

// placementOf pairs an entity with the one placement of the map that carries
// its MapUnitID, as a loaded session does.
func placementOf(mw *mapWorld) func(sim.Entity) (int, bool) {
	m := mw.mission.state.Map
	byUnit := map[uint16]int{}
	for i, u := range m.Units {
		if u.UnitID == 0 {
			continue
		}
		if _, dup := byUnit[u.UnitID]; dup {
			byUnit[u.UnitID] = -1
		} else {
			byUnit[u.UnitID] = i
		}
	}
	return func(e sim.Entity) (int, bool) {
		i, ok := byUnit[e.MapUnitID]
		return i, ok && i >= 0 && e.MapUnitID != 0
	}
}

// auditPlacedFigures checks every living placed person and creature: a placed
// Hero keeps its PC body and takes the cape (fighters only), every other person
// keeps his row's own class and never takes it, a rider keeps his horse, and a
// creature draws the tier palette its row's Face column selects.
func auditPlacedFigures(r *figureRig) []string {
	t, f := r.t, r.f
	t.Helper()
	mw, src := f.live, f.Archives.Containers
	var bad []string
	report := func(format string, args ...any) { bad = append(bad, fmt.Sprintf(format, args...)) }
	draws := r.drawsByID()
	pair := placementOf(mw)
	m := mw.mission.state.Map
	inParty := map[sim.EntityID]bool{}
	for _, id := range mw.mission.ids {
		inParty[id] = true
	}
	tierPalette := map[int32]map[int32][256]uint32{}
	capes, riders, persons, creatures := 0, 0, 0, 0
	for _, e := range mw.world.Entities() {
		if inParty[e.ID] || !e.Alive() {
			continue
		}
		d, drawn := draws[e.ID]
		i, paired := pair(e)
		if !drawn || !paired {
			continue
		}
		u := m.Units[i]
		res := mapload.Resolve(u, f.Table)
		if !res.Found() {
			continue
		}
		who := fmt.Sprintf("placement %d (unit %d, entity %d)", i, u.UnitID, e.ID)
		if res.Arm == mapload.ArmUnits {
			creatures++
			def, err := data.NewUnitDef(f.Table.Units.EntryName(res.Index), f.Table.Units.EntryParams(res.Index))
			if err != nil {
				continue
			}
			class := mw.units.Classes[def.TypeID]
			if class == nil || d.Art != class {
				report("%s: creature drawn as %q, want its class %d %q", who, artName(d.Art), def.TypeID, artName(class))
				continue
			}
			if got := mw.tiers[e.ID]; got != int(def.Face) {
				report("%s: creature tier %d, its row's Face column says %d", who, got, def.Face)
			}
			frames := class.TierFrames(int(def.Face))
			if len(frames) == 0 || d.Frame == nil {
				report("%s: creature has no frame in tier %d", who, def.Face)
				continue
			}
			want := mw.ownerFrame(class, frames[0], e.Owner)
			if d.Frame.Palette != want.Palette {
				report("%s: creature class %d tier %d drawn in another tier's palette", who, def.TypeID, def.Face)
			}
			if tierPalette[def.TypeID] == nil {
				tierPalette[def.TypeID] = map[int32][256]uint32{}
			}
			var key [256]uint32
			for k, c := range d.Frame.Palette {
				key[k] = uint32(c.R)<<24 | uint32(c.G)<<16 | uint32(c.B)<<8 | uint32(c.A)
			}
			tierPalette[def.TypeID][def.Face] = key
			continue
		}
		persons++
		def, err := data.NewHumanDef(f.Table.Humans.EntryName(res.Index), f.Table.Humans.EntryParams(res.Index))
		if err != nil {
			continue
		}
		fig, hasFigure := mw.figures[e.ID]
		if !hasFigure {
			report("%s: placed person has no figure", who)
			continue
		}
		hero := data.FigureIsHero(e.TypeID)
		if hero {
			body := mw.mission.state.Start.Roster[e.ID]
			pc := mw.units.Bodies[data.HeroBodyKey(body.BodyDir, data.HeroBody(body.Body))]
			if pc == nil || d.Art != pc || !isPCBody(mw.units, d.Art) {
				report("%s: placed Hero drawn as %q, not its PC resource", who, artName(d.Art))
			}
		} else if class := mw.units.Classes[e.TypeID]; class == nil || d.Art != class {
			report("%s: placed person drawn as %q, not his row's class %d %q", who, artName(d.Art), e.TypeID, artName(class))
		}
		if isPCBody(mw.units, d.Art) != hero {
			report("%s: PC body %t for a person with Hero %t", who, isPCBody(mw.units, d.Art), hero)
		}
		horse := data.FigureHasHorse(e.TypeID) && !hero
		if fig.Hero != hero || fig.Horse != horse {
			report("%s: figure Hero %t Horse %t, the type id %d says Hero %t Horse %t", who, fig.Hero, fig.Horse, e.TypeID, hero, horse)
		}
		_ = def
		want := oracleFigure(src, figureID{Dir: fig.Dir, Face: fig.Face, Hero: hero, Horse: horse}, mw.equipmentOf(e.ID))
		got := mw.unitFigure(e.ID)
		if want == nil || got == nil || !sameBytes(got.Pix, want.Pix) {
			report("%s: placed person's doll is not the %s figure for Hero %t Horse %t", who, fig.Dir, hero, horse)
		}
		if horse {
			riders++
			if d.Art != mw.units.Classes[e.TypeID] {
				report("%s: rider drawn on foot as %q", who, artName(d.Art))
			}
		}
		if hero && !fig.Dir.Mage() {
			capes++
		}
	}
	r.stats["placed capes"] += capes
	r.stats["riders"] += riders
	r.stats["placed persons"] += persons
	r.stats["creatures"] += creatures
	r.stats["tier families"] += len(tierPalette)
	for class, tiers := range tierPalette {
		var faces []int
		for face := range tiers {
			faces = append(faces, int(face))
		}
		slices.Sort(faces)
		for a := 0; a < len(faces); a++ {
			for b := a + 1; b < len(faces); b++ {
				if tiers[int32(faces[a])] == tiers[int32(faces[b])] {
					report("class %d: tiers %d and %d draw with one palette", class, faces[a], faces[b])
				}
			}
		}
	}
	if persons+creatures == 0 {
		report("mission %d holds no placed person or creature to audit", r.mission)
	}
	return bad
}

// placedFacts is what the player sees of every living placed unit, keyed by the
// placement's MapUnitID so a fresh mission and a loaded one compare unit by unit:
// the art class, the creature tier palette, the composed doll and the
// statistics card (the card follows the unit's own placement after a load).
type placedFact struct {
	art, card string
	palette   [32]byte
	doll      [32]byte
}

func placedFacts(r *figureRig) map[uint16]placedFact {
	mw := r.f.live
	draws := r.drawsByID()
	inParty := map[sim.EntityID]bool{}
	for _, id := range mw.mission.ids {
		inParty[id] = true
	}
	out := map[uint16]placedFact{}
	for _, e := range mw.world.Entities() {
		d, drawn := draws[e.ID]
		if inParty[e.ID] || !e.Alive() || e.MapUnitID == 0 || !drawn {
			continue
		}
		fact := placedFact{art: artName(d.Art)}
		if d.Frame != nil {
			h := sha256.New()
			fmt.Fprint(h, d.Frame.Palette)
			copy(fact.palette[:], h.Sum(nil))
		}
		if pic := mw.unitFigure(e.ID); pic != nil {
			fact.doll = sha256.Sum256(pic.Pix)
		}
		c := d.Char
		fact.card = fmt.Sprint(c.Known, c.Band, c.Name, c.Body, c.Reaction, c.Mind, c.Spirit, c.Skills, c.Weapon)
		out[e.MapUnitID] = fact
	}
	return out
}

// comparePlaced requires a loaded mission to show every placed unit as the
// fresh mission does.
func comparePlaced(fresh, cold *figureRig) []string {
	want, got := placedFacts(fresh), placedFacts(cold)
	var bad []string
	ids := make([]int, 0, len(want))
	for id := range want {
		ids = append(ids, int(id))
	}
	slices.Sort(ids)
	for _, id := range ids {
		w, ok := got[uint16(id)]
		switch {
		case !ok:
			bad = append(bad, fmt.Sprintf("unit %d is missing after LOAD", id))
		case w.art != want[uint16(id)].art:
			bad = append(bad, fmt.Sprintf("unit %d art %q after LOAD, %q fresh", id, w.art, want[uint16(id)].art))
		case w.palette != want[uint16(id)].palette:
			bad = append(bad, fmt.Sprintf("unit %d draws another palette after LOAD", id))
		case w.doll != want[uint16(id)].doll:
			bad = append(bad, fmt.Sprintf("unit %d has another doll after LOAD", id))
		case w.card != want[uint16(id)].card:
			bad = append(bad, fmt.Sprintf("unit %d card %s after LOAD, %s fresh", id, w.card, want[uint16(id)].card))
		}
	}
	if len(want) == 0 {
		bad = append(bad, "no placed unit to compare")
	}
	return bad
}

// auditInventorySubjects selects every party member through the mission's own
// inventory-subject switch. A hero and a companion become the subject of the
// equipment pane; a hired unit never does, so its pack and worn set are never
// shown as an inventory (MERC-TYPE-001, PARTY-FLAG-003).
func auditInventorySubjects(r *figureRig) []string {
	mw := r.f.live
	var bad []string
	report := func(format string, args ...any) { bad = append(bad, fmt.Sprintf(format, args...)) }
	first := mw.mission.ids[0]
	mw.switchInventorySubject(uint32(first))
	for i, p := range mw.mission.party {
		if i >= len(mw.mission.ids) {
			break
		}
		id := mw.mission.ids[i]
		if e, ok := mw.entity(id); !ok || !e.Alive() {
			continue
		}
		before := mw.invSubject.ID
		mw.switchInventorySubject(uint32(id))
		got := mw.invSubject.ID
		switch {
		case p.Hired() && got != before:
			report("hired %s (entity %d) became the inventory subject", p.Name, id)
		case !p.Hired() && data.ComposesFigure(p.Class) && got != uint32(id):
			report("%s (entity %d) did not become the inventory subject", p.Name, id)
		}
		if p.Hired() {
			r.stats["hired selections"]++
		}
	}
	return bad
}

// auditSpeakers checks the mission dialogue's picture for every person the
// mission holds under an npc number (placed persons through the roster, and
// carried companions): the speaker is drawn as himself, in the worn set the
// world says he wears at this moment, never as the bare record sheet
// (DLG-SPEAKER-022, DLG-FIGURE-021).
func auditSpeakers(r *figureRig) []string {
	t, f := r.t, r.f
	t.Helper()
	mw, src := f.live, f.Archives.Containers
	var bad []string
	report := func(format string, args ...any) { bad = append(bad, fmt.Sprintf(format, args...)) }
	holders := map[int][]sim.EntityID{}
	for id, m := range mw.mission.state.Start.Roster {
		if m.CompanionNPC != 0 {
			holders[m.CompanionNPC] = append(holders[m.CompanionNPC], id)
		}
	}
	for i, p := range mw.mission.party {
		if p.CompanionNPC != 0 && i < len(mw.mission.ids) {
			holders[p.CompanionNPC] = append(holders[p.CompanionNPC], mw.mission.ids[i])
		}
	}
	npcs := make([]int, 0, len(holders))
	for npc := range holders {
		npcs = append(npcs, npc)
	}
	slices.Sort(npcs)
	for _, npc := range npcs {
		var dressed, bare []*image.RGBA
		for _, id := range holders[npc] {
			if !mw.entityAlive(id) {
				continue
			}
			fig, ok := mw.figures[id]
			// The record's own sheet and face must name this person's, as the
			// section's Face term does; a person whose face differs from the
			// record's is not the one the dialogue is about.
			if rec, found := f.NPCFaces[int32(npc)]; !ok || !found || rec.Dir != fig.Dir || int(rec.Face) != fig.Face {
				continue
			}
			dressed = append(dressed, oracleFigure(src, fig, mw.equipmentOf(id)))
			bare = append(bare, oracleFigure(src, fig, data.Equipment{}))
		}
		if len(dressed) == 0 {
			continue
		}
		pic, _, ok := mw.SpeakerFace(npc)
		if !ok || pic == nil {
			report("npc%d: no dialogue picture", npc)
			continue
		}
		matched, naked := false, false
		for i := range dressed {
			if sameBytes(pic.Pix, dressed[i].Pix) {
				matched = true
				naked = !sameBytes(dressed[i].Pix, bare[i].Pix) && sameBytes(pic.Pix, bare[i].Pix)
			}
		}
		if !matched {
			report("npc%d: the dialogue picture is not the live person's figure in his worn set", npc)
		}
		if naked {
			report("npc%d: the speaker is drawn bare while he wears a set", npc)
		}
		r.stats["speakers"]++
		for i := range dressed {
			if !sameBytes(dressed[i].Pix, bare[i].Pix) {
				r.stats["dressed speakers"]++
				break
			}
		}
	}
	// A Figure record no live person of its sheet answers for stands as a
	// synthesised drawable: the face sheet with no worn layer, on the warrior
	// backdrop when the record states Hero and the sheet is not a mage's
	// (DLG-SPEAKER-022, HERO-FIGURE-144).
	sheets := map[figureID]bool{}
	for _, fig := range mw.figures {
		sheets[figureID{Dir: fig.Dir, Face: fig.Face}] = true
	}
	var records []int
	for npc, rec := range f.NPCFaces {
		if rec.Kind == data.NPCFigure {
			records = append(records, int(npc))
		}
	}
	slices.Sort(records)
	for _, npc := range records {
		rec := f.NPCFaces[int32(npc)]
		if sheets[figureID{Dir: rec.Dir, Face: rec.Face}] {
			continue
		}
		hero := rec.Tokens.Has(data.NPCTokenHero)
		want := oracleFigure(src, figureID{Dir: rec.Dir, Face: rec.Face}, data.Equipment{})
		if want == nil {
			continue
		}
		if hero && !rec.Dir.Mage() {
			want = oracleCaped(t, src, rec.Dir, want)
			r.stats["unanswered hero speakers"]++
		}
		pic, _, ok := mw.SpeakerFace(npc)
		if !ok || pic == nil || !sameBytes(pic.Pix, want.Pix) {
			report("npc%d: an unanswered %s record (Hero %t) is not drawn bare on its own backdrop", npc, rec.Dir, hero)
		}
		r.stats["unanswered speakers"]++
	}
	return bad
}

func requireClean(t *testing.T, label string, problems []string) {
	t.Helper()
	if len(problems) > 0 {
		if len(problems) > 12 {
			problems = append(problems[:12], fmt.Sprintf("... and %d more", len(problems)-12))
		}
		t.Errorf("%s:\n  %s", label, strings.Join(problems, "\n  "))
	}
}

func TestReleaseFigureRegressionMissions(t *testing.T) {
	total := map[string]int{}
	ran := 0
	for n, mission := range figureMissions() {
		hero := figureHeroes[n%len(figureHeroes)]
		t.Run(fmt.Sprintf("mission %d %s", mission, hero.name), func(t *testing.T) {
			fresh := openFigureMission(t, mission, hero)
			ran++
			requireClean(t, "fresh party", auditPartyFigures(fresh))
			requireClean(t, "fresh placed", auditPlacedFigures(fresh))
			requireClean(t, "fresh speakers", auditSpeakers(fresh))
			cold := fresh.coldLoad()
			requireClean(t, "loaded party", auditPartyFigures(cold))
			requireClean(t, "loaded placed", auditPlacedFigures(cold))
			requireClean(t, "loaded speakers", auditSpeakers(cold))
			requireClean(t, "placed units, loaded against fresh", comparePlaced(fresh, cold))
			requireClean(t, "fresh inventory subjects", auditInventorySubjects(fresh))
			requireClean(t, "loaded inventory subjects", auditInventorySubjects(cold))
			for k, v := range fresh.stats {
				total[k] += v
			}
		})
	}
	if ran == 0 {
		return
	}
	// The audits are only evidence over a population that holds the cases.
	for key, floor := range map[string]int{"placed capes": 1, "riders": 2, "placed persons": 50,
		"creatures": 500, "tier families": 50, "speakers": 10, "dressed speakers": 5,
		"unanswered hero speakers": 1, "hired selections": 20} {
		if total[key] < floor {
			t.Errorf("the mission walk audited %d %s, want at least %d", total[key], key, floor)
		}
	}
}
