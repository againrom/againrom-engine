package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// secondMissionStore is a mission file's state store with the keys the EN
// mission files hold (R2-SESSION-244): in battle, View 8/29, Selection [1],
// the given speed, open book and pressed slot.
func secondMissionStore(speed, bookOpen, pressed int32) []byte {
	leaf := func(name string, v int32) synth.RegNode { return synth.RegNode{Name: name, Kind: 0x02, Int: v} }
	return synth.Reg(0x11, []synth.RegNode{
		{Name: "CurrentState", Kind: 0x01, Children: []synth.RegNode{leaf("InBattle", 1)}},
		{Name: "GameOptions", Kind: 0x01, Children: []synth.RegNode{
			leaf("Wimpy", 0), leaf("ShowHP", 0), leaf("FlyingHP", 1), leaf("Formation", 1),
			leaf("Speed", speed), leaf("ShowTimeFlow", 1),
		}},
		{Name: "Inventory", Kind: 0x01, Children: []synth.RegNode{leaf("IsOpen", 0)}},
		{Name: "Objects", Kind: 0x01, Children: []synth.RegNode{{Name: "Selection", Kind: 0x06, Ints: []int32{1}}}},
		{Name: "SpellBook", Kind: 0x01, Children: []synth.RegNode{
			leaf("IsOpen", bookOpen), leaf("Pressed", pressed), {Name: "Shortcuts", Kind: 0x06, Ints: []int32{}},
		}},
		{Name: "View", Kind: 0x01, Children: []synth.RegNode{leaf("X", 8), leaf("Y", 29)}},
	})
}

// secondMissionFile is a mission-like second-game file in mission 10: the
// hero's Player and an empty world half, the store and the mission tail.
func secondMissionFile(hero secondOriginalHero, store []byte) []byte {
	return labelledContainer(sav.SecondGame.Magic, "mission", secondSavedDocument(10, 44, &hero, true),
		append(store, secondSavedTail(0, [2]uint32{10, placeMission})...))
}

// secondMissionMap is a second-game map of w by h zero tiles with no
// objects, units or script nodes: version 1300, the 660-byte type-0 record
// and the first game's other nine records in the corpus order. The type-7
// body is its three count words; a second-game map with no script still
// runs the second game's script state, which holds the campaign bank.
func secondMissionMap(w, h int) []byte { return secondMissionMapOf(w, h, secondMissionContent{}) }

// secondMissionContent is what a fixture mission holds beyond the empty map:
// type-4 objects, type-7 action nodes, type-11c records, files of the main
// archive, and the mission font, without which a notice takes no input.
type secondMissionContent struct {
	objects  []alm.Object
	actions  []alm.ScriptNode
	pointers []alm.StructureAction
	main     []synth.File
	font     bool
}

// secondMissionMapOf is secondMissionMap with c's objects, action nodes and
// type-11c records; the type-11 record follows type 7 when c has any.
func secondMissionMapOf(w, h int, c secondMissionContent) []byte {
	le := binary.LittleEndian
	meta := make([]byte, 660)
	le.PutUint32(meta[0x00:], uint32(w))
	le.PutUint32(meta[0x04:], uint32(h))
	le.PutUint32(meta[0x1c:], 1) // one type-5 group record
	le.PutUint32(meta[0x20:], uint32(len(c.objects)))
	le.PutUint32(meta[0x3c:], uint32(len(c.pointers)))
	copy(meta[0x44:], "second")
	group := make([]byte, 76)
	copy(group[0x0c:], "Self")
	var objects []byte
	for _, o := range c.objects {
		rec := make([]byte, 20)
		le.PutUint32(rec[0x00:], o.X)
		le.PutUint32(rec[0x04:], o.Y)
		le.PutUint32(rec[0x08:], o.Kind)
		le.PutUint16(rec[0x0c:], o.Field0C)
		le.PutUint16(rec[0x12:], o.Field12)
		objects = append(objects, rec...)
	}
	script := le.AppendUint32(nil, uint32(len(c.actions)))
	for _, n := range c.actions {
		rec := make([]byte, 796)
		le.PutUint32(rec[0x40:], n.Opcode)
		le.PutUint32(rec[0x44:], n.ID)
		for k := range n.Value {
			le.PutUint32(rec[0x4c+4*k:], n.Value[k])
			le.PutUint32(rec[0x74+4*k:], n.Type[k])
		}
		script = append(script, rec...)
	}
	script = append(script, make([]byte, 8)...)
	var pointers []byte
	for _, r := range c.pointers {
		pointers = le.AppendUint32(le.AppendUint32(le.AppendUint32(pointers, r.ID), r.Gate), r.Position)
	}
	payloads := map[int][]byte{0: meta, 1: make([]byte, 2*w*h), 2: make([]byte, w*h), 3: make([]byte, w*h),
		4: objects, 5: group, 6: nil, 7: script, 8: nil, 9: make([]byte, 4), 11: pointers}
	order := []int{0, 1, 2, 3, 5, 4, 9, 8, 6, 7}
	if len(c.pointers) != 0 {
		order = append(order, 11)
	}
	out := make([]byte, 20)
	le.PutUint32(out[0:], 0x0052374D)
	le.PutUint32(out[4:], 20)
	le.PutUint32(out[12:], uint32(len(order)))
	le.PutUint32(out[16:], 1300)
	for _, typeID := range order {
		p := payloads[typeID]
		hdr := make([]byte, 20)
		le.PutUint32(hdr[0:], 7)
		le.PutUint32(hdr[4:], 20)
		le.PutUint32(hdr[8:], uint32(len(p)))
		le.PutUint32(hdr[12:], uint32(typeID))
		out = append(append(out, hdr...), p...)
	}
	return out
}

// secondMissionFront is a second-game front over a 40x40 mission 10 map.
func secondMissionFront(t *testing.T) *FrontEnd {
	t.Helper()
	return secondMissionFrontOf(t, secondMissionContent{})
}

// secondMissionFrontOf is secondMissionFront over a mission holding c.
func secondMissionFrontOf(t *testing.T, c secondMissionContent) *FrontEnd {
	t.Helper()
	f := currentPoolFixtureFront(t)
	dir := t.TempDir()
	path := filepath.Join(dir, ScenarioArchive)
	b := synth.Archive([]synth.File{{Path: "10.alm", Data: secondMissionMapOf(40, 40, c)}, {Path: "npc.reg", Data: synth.NPCReg(nil)}})
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	paths := []string{path}
	if len(c.main) != 0 {
		paths = append(paths, filepath.Join(dir, MainArchive))
		if err := os.WriteFile(paths[1], synth.Archive(c.main), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fs, err := vfs.Open(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.Archives.Containers = fs
	profile, ok := base.Find(base.ROM2EN)
	if !ok {
		t.Fatal("missing ROM2 profile")
	}
	f.Archives.Base = base.Match{Profile: profile}
	if c.font {
		f.Font = resolved(missionFont(), nil)
	}
	// The second game's spell table has 29 rows; Slow is the last, a speed
	// effect with no damage.
	rows := dbCollection{{}}
	for id := 1; id <= 29; id++ {
		p := make([]int32, 19)
		p[1], p[4], p[6], p[8], p[16], p[17], p[18] = 1, 1, 10, 1, 1, 1, 1
		row := dbEntry{name: "spell", params: p}
		if id == 29 {
			p[14], p[16], p[17] = 5, 0, 0
			row.strings = []string{"speed=-3:duration 40"}
		}
		rows = append(rows, row)
	}
	f.Table.Spells = rows
	f.Table = readUnder(base.GameROM2, f.Table)
	return f
}

// secondMissionLoad LOADs name from dir into a fresh second-game front
// whose save directory is dir.
func secondMissionLoad(t *testing.T, dir, name string) (*FrontEnd, *ui.App) {
	t.Helper()
	return secondMissionLoadOf(t, secondMissionContent{}, dir, name)
}

// secondMissionLoadOf is secondMissionLoad over a mission holding c.
func secondMissionLoadOf(t *testing.T, c secondMissionContent, dir, name string) (*FrontEnd, *ui.App) {
	t.Helper()
	f := secondMissionFrontOf(t, c)
	app := f.App("second-mission")
	app.Layout(1024, 768)
	// The second game lists its own directory, its originals among them.
	save, list, load := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, name)
	return f, app
}

// secondMissionLoadFile writes b as game0001.sav and LOADs it.
func secondMissionLoadFile(t *testing.T, b []byte) (*FrontEnd, *ui.App, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	f, app := secondMissionLoad(t, dir, "game0001.sav")
	return f, app, dir
}

// secondMissionFixtureHero is the fixture hero at cell 11/21 with his staff
// in the hand and nothing else worn, as the EN game0004 hero.
func secondMissionFixtureHero() secondOriginalHero {
	h := secondFixtureHero()
	h.held, h.worn, h.cell = &h.worn[0], nil, [2]uint8{11, 21}
	return h
}

// secondMissionActor is the player's one living actor.
func secondMissionActor(t *testing.T, w *sim.World) sim.Entity {
	t.Helper()
	var hero sim.Entity
	n := 0
	for _, e := range w.Entities() {
		if e.Owner == sim.SelfSlot && e.Alive() {
			hero, n = e, n+1
		}
	}
	if n != 1 {
		t.Fatalf("%d living actors of the player, want the hero", n)
	}
	return hero
}

// secondMissionState is what a LOAD of the fixture hero shows: the hero's
// place, pools, attributes, skills, holdings and book, and the first frame.
type secondMissionState struct {
	x, y, hp, maxHP, mana, maxMana int32
	attributes                     [4]int32
	skills                         [6]int32
	known                          uint32
	book                           sim.BookState
	held                           uint16
	carried                        []uint16
	view                           ui.SaveApplicationState
}

func secondMissionStateOf(t *testing.T, f *FrontEnd) secondMissionState {
	t.Helper()
	w := f.live.world
	e := secondMissionActor(t, w)
	eq, _ := w.Equipped(e.ID)
	carried, _ := w.Carried(e.ID)
	party := f.LiveParty()
	if len(party) != 1 {
		t.Fatalf("party %d, want the hero", len(party))
	}
	h := party[0].Hero
	view, ok := f.LiveApplication()
	if !ok {
		t.Fatal("no live application")
	}
	s := secondMissionState{x: e.X, y: e.Y, hp: e.HP, maxHP: e.MaxHP, mana: e.Mana, maxMana: e.MaxMana,
		attributes: [4]int32{h.Body, h.Reaction, h.Mind, h.Spirit}, known: uint32(e.KnownSpells), book: e.Book.State,
		held: eq[0], carried: slices.Clone(carried), view: view}
	for i, v := range e.Skill {
		s.skills[i] = int32(v)
	}
	return s
}

// The fixture mission file LOADs its hero with his values, and its first
// frame follows the state store: View 8/29, the hero selected, the book open
// or closed, the inventory closed and Speed 4 or 8 as the 62 or 31 ms period
// (R2-SESSION-244, R2-ENGINE-620).
func TestSecondMissionFileLoadsTheHeroAndTheFirstFrame(t *testing.T) {
	for _, c := range []struct {
		speed, open, pressed int32
		period               int
		spell                uint32
	}{{4, 0, -1, 62000, 0}, {8, 1, 6, 31000, 21}} {
		f, _, _ := secondMissionLoadFile(t, secondMissionFile(secondMissionFixtureHero(), secondMissionStore(c.speed, c.open, c.pressed)))
		s := secondMissionStateOf(t, f)
		e := secondMissionActor(t, f.live.world)
		if s.x != 11 || s.y != 21 || s.hp != 29 || s.maxHP != 29 || s.mana != 75 || s.maxMana != 75 {
			t.Fatalf("hero place and pools %+v", s)
		}
		if s.attributes != [4]int32{19, 15, 43, 25} || s.skills[4] != 20 || s.skills[5] != 10 {
			t.Fatalf("hero attributes %v skills %v", s.attributes, s.skills)
		}
		if s.held != 0x812d || !slices.Equal(s.carried, []uint16{0x812d}) {
			t.Fatalf("hero holds %#x carries %v", s.held, s.carried)
		}
		if s.known != 1<<24|1<<27 || s.book != sim.BookPresent {
			t.Fatalf("book %#x state %v", s.known, s.book)
		}
		v := s.view
		if !slices.Equal(v.Selection, []uint32{uint32(e.ID)}) || v.ViewX != 8 || v.ViewY != 29 || v.InventoryOpen ||
			v.SpellBookOpen != (c.open != 0) || v.PeriodUS != c.period || v.PressedSpell != c.spell {
			t.Fatalf("speed %d first frame %+v", c.speed, v)
		}
	}
}

// A changed attribute and a changed position in the file reach the loaded
// hero.
func TestSecondMissionFileLoadFollowsTheFile(t *testing.T) {
	hero := secondMissionFixtureHero()
	hero.stats[sav.StatMind], hero.cell = 44, [2]uint8{3, 5}
	f, _, _ := secondMissionLoadFile(t, secondMissionFile(hero, secondMissionStore(4, 0, -1)))
	s := secondMissionStateOf(t, f)
	if s.attributes[2] != 44 || s.x != 3 || s.y != 5 {
		t.Fatalf("mind %d at %d/%d, want 44 at 3/5", s.attributes[2], s.x, s.y)
	}
}

// After a mission file's LOAD our SAVE writes our own document; its cold
// LOAD shows the same hero, book and first frame and the next order runs to
// the same World.
func TestSecondMissionFileSaveColdLoadAndNextOrder(t *testing.T) {
	hero := secondMissionFixtureHero()
	hero.spells, hero.bookSlots = []uint8{24, 29}, 30
	f, app, dir := secondMissionLoadFile(t, secondMissionFile(hero, secondMissionStore(8, 1, 6)))
	loaded := secondMissionStateOf(t, f)
	if loaded.known != 1<<24|1<<29 || loaded.book != sim.BookNativePresent {
		t.Fatalf("a book holding Slow %#x state %v", loaded.known, loaded.book)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := SaveStore{Dir: dir}.List()
	if err != nil || len(entries) != 1 || entries[0].Name == "game0001.sav" {
		t.Fatalf("our SAVE: %+v %v: %s", entries, err, app.HeadlessMessage())
	}
	written, err := os.ReadFile(filepath.Join(dir, entries[0].Name))
	if err != nil {
		t.Fatal(err)
	}
	if p, ok := sav.ProfileOf(written); !ok || p != sav.FirstGame {
		t.Fatal("our SAVE is not our own document")
	}
	if err := os.Remove(filepath.Join(dir, "game0001.sav")); err != nil {
		t.Fatal(err)
	}
	cold, _ := secondMissionLoad(t, dir, entries[0].Name)
	got := secondMissionStateOf(t, cold)
	want := secondMissionStateOf(t, f)
	if got.known != want.known || got.book != want.book || got.x != want.x || got.y != want.y || got.attributes != want.attributes ||
		got.hp != want.hp || got.mana != want.mana || got.held != want.held || !slices.Equal(got.carried, want.carried) {
		t.Fatalf("cold LOAD %+v, want %+v", got, want)
	}
	if got.view.ViewX != want.view.ViewX || got.view.ViewY != want.view.ViewY || got.view.SpellBookOpen != want.view.SpellBookOpen ||
		got.view.PeriodUS != want.view.PeriodUS || got.view.PressedSpell != want.view.PressedSpell || len(got.view.Selection) != 1 {
		t.Fatalf("cold first frame %+v, want %+v", got.view, want.view)
	}
	if cold.live.world.Hash() != f.live.world.Hash() {
		t.Fatal("the cold LOAD changed the World")
	}
	id := uint32(secondMissionActor(t, f.live.world).ID)
	f.live.enqueue(id, 15, 21)
	cold.live.enqueue(uint32(secondMissionActor(t, cold.live.world).ID), 15, 21)
	for range 32 {
		f.live.tick()
		cold.live.tick()
		if f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatalf("the next order diverged at tick %d", f.live.world.Tick())
		}
	}
	if e := secondMissionActor(t, cold.live.world); e.X == 11 {
		t.Fatal("the hero did not move")
	}
}

// A projectile a mission file carries without a driver does not refuse our
// SAVE, alone or beside a driven one: the document leaves it out, and the
// cold LOAD holds and continues only the driven one (DIV-4175).
func TestSecondMissionFileSaveLeavesAnUndrivenProjectileOut(t *testing.T) {
	for _, driven := range []bool{false, true} {
		f, app, dir := secondMissionLoadFile(t, secondMissionFile(secondMissionFixtureHero(), secondMissionStore(4, 0, -1)))
		if f.live.world.SavedWorldEffectDrivers() != nil {
			t.Fatal("the fixture already drives an effect")
		}
		w := f.live.world
		w.SetSavedProjectiles(sim.SavedProjectiles{FreeIndex: 8, IDs: []uint16{7}, Items: []sim.SavedProjectile{{ID: 7}}})
		want := []uint16{}
		if driven {
			hero := secondMissionActor(t, w)
			id, ok := w.ReleaseCast(sim.CastRecord{Caster: hero.ID, Picture: 1, Phases: 1, Segments: 40,
				X: int32(hero.X)*256 + 128, Y: int32(hero.Y)*256 + 128, AimX: int32(hero.X+8)*256 + 128, AimY: int32(hero.Y)*256 + 128})
			if !ok || len(w.SavedProjectiles().Items) != 2 || len(w.SavedWorldEffectDrivers().Projectiles) != 1 {
				t.Fatalf("driven %v: the cast built %d, %v: %+v", driven, id, ok, w.SavedProjectiles())
			}
			want = []uint16{id}
		}
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessGameMenuAction("save"); err != nil {
			t.Fatal(err)
		}
		entries, err := SaveStore{Dir: dir}.List()
		if err != nil || len(entries) != 1 {
			t.Fatalf("driven %v: our SAVE: %+v %v: %s", driven, entries, err, app.HeadlessMessage())
		}
		cold, _ := secondMissionLoad(t, dir, entries[0].Name)
		got := cold.live.world.SavedProjectiles()
		ids := []uint16{}
		for _, p := range got.Items {
			ids = append(ids, p.ID)
		}
		if !slices.Equal(ids, want) || !slices.Equal(append([]uint16{}, got.IDs...), want) {
			t.Fatalf("driven %v: the cold LOAD holds %v, IDs %v, want %v", driven, ids, got.IDs, want)
		}
		if !driven {
			continue
		}
		start := got.Items[0]
		for range 4 {
			f.live.tick()
			cold.live.tick()
		}
		var live sim.SavedProjectile
		for _, p := range w.SavedProjectiles().Items {
			if p.ID == want[0] {
				live = p
			}
		}
		moved := cold.live.world.SavedProjectiles().Items
		if len(moved) != 1 || moved[0].X == start.X || moved[0].X != live.X || moved[0].Y != live.Y {
			t.Fatalf("the cold LOAD's projectile %+v from %+v, the live one %+v", moved, start, live)
		}
	}
}

// A cast of Slow, the spell past the book's slots, never refuses our SAVE:
// at each tick from the order through the wind-up, the release and the
// effect's life, the SAVE writes and its cold LOAD holds the selection, the
// cast and the effect, and continues as the live World does (DIV-4174).
func TestSecondMissionFileSlowCastSavesAndColdLoads(t *testing.T) {
	hero := secondMissionFixtureHero()
	hero.spells, hero.bookSlots = []uint8{24, 29}, 30
	f, _, dir := secondMissionLoadFile(t, secondMissionFile(hero, secondMissionStore(4, 0, -1)))
	save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	caster := secondMissionActor(t, f.live.world)
	if why := f.live.world.BookSpellRefusal(caster.ID, caster.ID, 29); why != "" {
		t.Fatalf("Slow on the hero is refused: %s", why)
	}
	f.live.pending = append(f.live.pending, sim.Cast(caster.ID, caster.ID, 29))
	saved := map[string]bool{}
	selected, slowed := false, false
	var prior *FrontEnd
	for range 24 {
		f.live.tick()
		w := f.live.world
		if prior != nil {
			if prior.live.tick(); prior.live.world.Hash() != w.Hash() {
				t.Fatalf("tick %d: the prior cold LOAD's next tick diverged", w.Tick())
			}
		}
		selected = selected || secondMissionActor(t, w).AdmittedBookSpell == 29
		slowed = slowed || slices.ContainsFunc(w.ActiveEffects(), func(e sim.ActiveEffect) bool { return e.Spell == 29 })
		if _, err := save(true); err != nil {
			t.Fatalf("tick %d: our SAVE: %v", w.Tick(), err)
		}
		entries, err := SaveStore{Dir: dir}.List()
		if err != nil {
			t.Fatal(err)
		}
		name := ""
		for _, e := range entries {
			if !saved[e.Name] {
				name = e.Name
			}
		}
		if name == "" {
			t.Fatalf("tick %d: our SAVE wrote nothing", w.Tick())
		}
		saved[name] = true
		cold, _ := secondMissionLoad(t, dir, name)
		if got, want := secondMissionActor(t, cold.live.world).AdmittedBookSpell, secondMissionActor(t, w).AdmittedBookSpell; got != want {
			t.Fatalf("tick %d: the cold LOAD's selection is %d, want %d", w.Tick(), got, want)
		}
		if cold.live.world.Hash() != w.Hash() {
			t.Fatalf("tick %d: the cold LOAD changed the World", w.Tick())
		}
		prior = cold
	}
	if !selected || !slowed {
		t.Fatalf("the cast selected Slow %v and slowed the hero %v", selected, slowed)
	}
}

// The book page shows the cells of the second game's slot table: a book
// holding Slow (row 29) shows 24 cells and none for Slow (R2-ENGINE-619,
// DIV-4174).
func TestSecondBookPageLeavesSlowOut(t *testing.T) {
	var table []sim.SpellRule
	for id := uint16(1); id <= 29; id++ {
		table = append(table, sim.SpellRule{ID: id})
	}
	e := sim.Entity{KnownSpells: 1<<24 | 1<<29}
	rows := data.SpellPresentationOf(rules.SpellGameSecond).BookRows
	book, fixed := selectedSpellbook(sim.Rules{}, []sim.Entity{e}, table, nil, ui.Words{}, nil, rows)
	if !fixed || len(book) != 24 {
		t.Fatalf("%d cells fixed %t, want the 24 of the table", len(book), fixed)
	}
	for _, c := range book {
		if c.ID == 29 {
			t.Fatal("the page shows Slow")
		}
	}
}
