package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// tailRecord is one shipped type-6 record's raw bytes, read off the map file
// by its unit id without the format package's tail decoder.
func tailRecord(t *testing.T, f *FrontEnd, mission int, unit uint16) []byte {
	t.Helper()
	addr, _ := MissionMap(mission)
	b, err := f.Archives.Containers.ReadFile(addr)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := alm.OpenDocument(b)
	if err != nil {
		t.Fatal(err)
	}
	var p []byte
	for i, id := range doc.RecordTypeIDs() {
		if id == 6 {
			p = doc.RecordPayload(i)
		}
	}
	for at := 0; at+70 <= len(p); at += 70 {
		if binary.LittleEndian.Uint16(p[at+0x40:]) == unit {
			return p[at : at+70]
		}
	}
	t.Fatalf("mission %d has no unit %d", mission, unit)
	return nil
}

// tailValues are the actor fields the override block can store.
func tailValues(e sim.Entity) string {
	return fmt.Sprintf("hp %d/%d mana %d/%d body %d mind %d spirit %d reaction %d defence %d absorption %d skill %v protection %v",
		e.HP, e.MaxHP, e.Mana, e.MaxMana, e.NativeBasis.Body, e.Mind, e.Spirit, e.Reaction, e.Defence, e.Absorption, e.Skill, e.Protection)
}

// assertTailApplied checks that every present byte of rec is the actor's own
// value: the stats, defence, absorption, the five skills and the five
// protections (UNIT-PLACE-034, UNIT-PLACESKILL-086, UNIT-PLACERESIST-087),
// except a Shooting byte that is the record's only skill byte (DIV-3581).
func assertTailApplied(t *testing.T, label string, rec []byte, e sim.Entity) {
	t.Helper()
	got := map[int]int32{0x2c: int32(e.NativeBasis.Body), 0x2d: e.Mind, 0x2e: e.Spirit, 0x2f: e.Reaction,
		0x31: e.Defence, 0x32: e.Absorption}
	for i := 0; i < 5; i++ {
		got[0x36+i], got[0x3b+i] = e.Skill[i+1], e.Protection[i]
	}
	if !slices.ContainsFunc(rec[0x36:0x3a], func(v byte) bool { return v != 0 }) {
		delete(got, 0x3a) // a lone Shooting byte is not stored (DIV-3581)
	}
	stores := 0
	for off, v := range got {
		if rec[off] != 0 {
			stores++
			if v != int32(rec[off]) {
				t.Errorf("%s: file +%#x is %d, the actor holds %d", label, off, rec[off], v)
			}
		}
	}
	if hp := int16(binary.LittleEndian.Uint16(rec[0x20:])); hp != -1 && e.HP != int32(hp) {
		t.Errorf("%s: authored health %d, the actor holds %d", label, hp, e.HP)
	}
	if stores == 0 {
		t.Fatalf("%s: the record stores nothing", label)
	}
}

// tailSaved checks that the SAV's record of map unit u carries the actor's
// override fields: the four stat words, health and mana, and the defence
// block's defence, absorption and five protections, and the attack block's
// skill words.
func tailSaved(t *testing.T, raw []byte, u uint16, e sim.Entity) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range doc.Objects {
		unit, ok := uint32(0), false
		for _, v := range r.Values {
			if v.Name == "T08" {
				unit, ok = v.Value, true
			}
		}
		if !ok || unit != uint32(u) || (r.Class != "Unit" && r.Class != "Human") {
			continue
		}
		le := binary.LittleEndian
		def, attack := savedRecordRawForTest(t, r, "UBE"), savedRecordRawForTest(t, r, "UA6")
		got := fmt.Sprintf("body %d mind %d spirit %d reaction %d health %d mana %d defence %d absorption %d",
			savedRecordValueForTest(t, r, "Body"), savedRecordValueForTest(t, r, "Mind"), savedRecordValueForTest(t, r, "Spirit"),
			savedRecordValueForTest(t, r, "Reaction"), int16(savedRecordValueForTest(t, r, "Health")), int16(savedRecordValueForTest(t, r, "Mana")),
			le.Uint16(def), le.Uint16(def[2:]))
		want := fmt.Sprintf("body %d mind %d spirit %d reaction %d health %d mana %d defence %d absorption %d",
			e.NativeBasis.Body, e.Mind, e.Spirit, e.Reaction, e.HP, e.Mana, e.Defence, e.Absorption)
		for i := 0; i < 5; i++ {
			got += fmt.Sprintf(" p%d %d s%d %d", i, le.Uint16(def[6+2*i:]), i+1, le.Uint16(attack[4+2*i:]))
			want += fmt.Sprintf(" p%d %d s%d %d", i, e.Protection[i], i+1, e.Skill[i+1])
		}
		if got != want {
			t.Fatalf("SAV record of unit %d: got %s, want %s", u, got, want)
		}
		return
	}
	t.Fatalf("the SAV holds no actor record of map unit %d", u)
}

func tailEntity(t *testing.T, w *sim.World, unit uint16) sim.Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.MapUnitID == unit {
			return e
		}
	}
	t.Fatalf("no actor of map unit %d", unit)
	return sim.Entity{}
}

// TestReleasePlacementTailSurvivesSaveColdLoad starts each mission through
// the mission door, checks the applied override bytes on its actors, saves
// SAV through F2, loads it cold and steps both sessions: the values hold.
func TestReleasePlacementTailSurvivesSaveColdLoad(t *testing.T) {
	for _, c := range []struct {
		mission int
		units   []uint16
	}{
		{141, []uint16{5}},       // a Human: four stats, the second derivation
		{140, []uint16{28, 202}}, // two creatures: stats, skills, protections
		{41, []uint16{38}},       // a Human: five skills over the trained levels
	} {
		t.Run(fmt.Sprint(c.mission), func(t *testing.T) {
			f := releaseFront(t)
			f.Options = OptionsStore{}
			f.SetDeterministicFrames(true)
			party := f.ChargenParty(ui.ChargenResult{Name: "TailProbe", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
			app := f.App("placement tail SAVE")
			t.Cleanup(app.StopAudio)
			app.Layout(1024, 768)
			f.ConfigureSaveSeams(app, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
			if err := app.OpenMission(f.missionOpener(c.mission, party, nil, nil, nil)); err != nil {
				t.Fatal(err)
			}
			app.Layout(1024, 768)
			if err := app.HeadlessKey("0"); err != nil {
				t.Fatal(err)
			}
			for k := 0; k < 8; k++ {
				if _, _, open := f.LiveNotice(); !open {
					break
				}
				if err := app.HeadlessKey("enter"); err != nil {
					t.Fatal(err)
				}
			}
			for _, u := range c.units {
				assertTailApplied(t, fmt.Sprintf("mission %d unit %d", c.mission, u), tailRecord(t, f, c.mission, u), tailEntity(t, f.live.world, u))
			}
			dir, raw := castOrderF2Save(t, f, app, "tail")
			for _, u := range c.units {
				tailSaved(t, raw, u, tailEntity(t, f.live.world, u))
			}
			g, _ := castOrderSession(t, dir)
			for tick := 0; tick < 2; tick++ {
				for _, u := range c.units {
					want, got := tailValues(tailEntity(t, f.live.world, u)), tailValues(tailEntity(t, g.live.world, u))
					if want != got {
						t.Fatalf("tick %d unit %d: cold LOAD holds %s, the session %s", tick, u, got, want)
					}
				}
				f.live.tick()
				g.live.tick()
			}
		})
	}
}

// tailMap reads one installed map's bytes: a root file or a scenario entry.
func tailMap(t *testing.T, f *FrontEnd, source string) []byte {
	t.Helper()
	read := f.Archives.Containers.ReadFile
	if !strings.HasPrefix(source, "scenario/") {
		read = f.Archives.Loose.ReadFile
	}
	b, err := read(source)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// tailSpawn builds the map in b at Normal difficulty through the game's
// spawn and returns the world and the decoded map.
func tailSpawn(t *testing.T, f *FrontEnd, b []byte) (*sim.World, *alm.Map) {
	t.Helper()
	m, err := alm.Open(b)
	if err != nil {
		t.Fatal(err)
	}
	w, err := mapload.FromALMWith(m, f.Table, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	return w, m
}

// tailPayload is the type-6 payload of map bytes b, sliced from b itself.
func tailPayload(t *testing.T, b []byte) []byte {
	t.Helper()
	doc, err := alm.OpenDocument(b)
	if err != nil {
		t.Fatal(err)
	}
	for i, id := range doc.RecordTypeIDs() {
		if id == 6 {
			p := doc.RecordPayload(i)
			at := bytes.Index(b, p)
			if at < 0 {
				t.Fatal("type-6 payload not found in the file")
			}
			return b[at : at+len(p)]
		}
	}
	t.Fatal("no type-6 section")
	return nil
}

// humanHealthMaximum is HERO-HP-072's health maximum of a Human with no
// equipment health term, computed here from the row's skill columns.
func humanHealthMaximum(body int32, fighter bool, skill [data.SkillSlots]int32) int32 {
	body = min(body, 50)
	mult := 1.0
	if fighter {
		mult = 2
	}
	xp := 0.0
	for _, s := range skill {
		xp += math.Trunc((math.Pow(1.1, float64(s)) - 1) * 1000)
	}
	h := float64(body) * mult
	if h != 0 {
		h = math.Trunc(h + math.Log(xp/5000+1)/math.Log(1.1)*mult)
		h = math.Trunc(h * (math.Pow(1.1, float64(body))/100 + 1))
	}
	return int32(h)
}

// TestReleasePlacementTailWitnesses: the shipped records the stores reach.
// Every present byte of every Cross.ALM tail record (EN) is its actor's
// value; scn:141's Body 15 Human takes the health maximum HERO-HP-072
// computes from Body 15 and the lesser live health; scn:151's NPC-arm Lord
// keeps his derived absorption and protections (DIV-3580). Bit 7 ships on no
// record (ALM-FLAGCORP-110), so a synthetic bit on scn:10's brigands removes
// the row's equipment on the type-key arm and leaves a creature's weapon.
func TestReleasePlacementTailWitnesses(t *testing.T) {
	f := releaseFront(t)
	if f.Base().Profile.Language != "russian" {
		b := tailMap(t, f, "Cross.ALM")
		w, m := tailSpawn(t, f, b)
		p := tailPayload(t, b)
		n := 0
		for i := range m.Units {
			rec := p[i*70 : (i+1)*70]
			if slices.ContainsFunc(rec[0x2c:0x40], func(v byte) bool { return v != 0 }) {
				e, _ := w.Entity(sim.EntityID(i))
				assertTailApplied(t, fmt.Sprintf("Cross.ALM record %d", i), rec, e)
				n++
			}
		}
		if n != 23 {
			t.Fatalf("Cross.ALM has %d tail records, want 23 (UNIT-PLACERESIST-087)", n)
		}
	}

	addr, _ := MissionMap(141)
	w, m := tailSpawn(t, f, tailMap(t, f, addr))
	e := tailEntity(t, w, 5)
	r := mapload.Resolve(m.Units[e.ID], f.Table)
	row, err := data.NewHumanDef(f.Table.Humans.EntryName(r.Index), f.Table.Humans.EntryParams(r.Index))
	if err != nil {
		t.Fatal(err)
	}
	template := humanHealthMaximum(row.Body, row.Profile().Fighter, row.Skill)
	want := humanHealthMaximum(15, row.Profile().Fighter, row.Skill)
	if e.NativeBasis.Body != 15 || e.MaxHP != want || e.HP != min(want, template) || want == template {
		t.Fatalf("scn:141 unit 5: body %d health %d/%d, want body 15 and health %d/%d (template maximum %d)",
			e.NativeBasis.Body, e.HP, e.MaxHP, min(want, template), want, template)
	}

	// scn:151's NPC-arm Lord keeps his derived absorption and protections
	// against his record's 240 and 100s (DIV-3580); the control is the same
	// record off the NPC arm, which stores them.
	addr, _ = MissionMap(151)
	b151 := slices.Clone(tailMap(t, f, addr))
	w, m = tailSpawn(t, f, b151)
	lord := tailEntity(t, w, 205)
	if r := mapload.Resolve(m.Units[lord.ID], f.Table); r.Arm != mapload.ArmNPC || lord.Absorption == 240 || lord.Protection[3] == 100 {
		t.Fatalf("scn:151 unit 205 on the %v arm: absorption %d protections %v, want the derived values on the NPC arm", r.Arm, lord.Absorption, lord.Protection)
	}
	tailPayload(t, b151)[int(lord.ID)*70+0x0c] &^= 1
	w, _ = tailSpawn(t, f, b151)
	if e := tailEntity(t, w, 205); e.Absorption != 240 || e.Protection != [5]int32{100, 100, 100, 100, 100} {
		t.Fatalf("scn:151 unit 205 off the NPC arm: absorption %d protections %v, want 240 and 100s", e.Absorption, e.Protection)
	}

	// Bit 7 on the first armed placement of each population: a type-key or
	// definition-id Human loses the row's equipment, a creature keeps its
	// weapon (the Units constructor takes no gate, ALM-FLAGPATH-109).
	for _, c := range []struct {
		mission int
		human   bool
	}{{10, true}, {140, false}} {
		addr, _ = MissionMap(c.mission)
		b := slices.Clone(tailMap(t, f, addr))
		w, m = tailSpawn(t, f, b)
		at := -1
		for i, u := range m.Units {
			arm := mapload.Resolve(u, f.Table).Arm
			worn, _ := w.EquippedItems(sim.EntityID(i))
			armed := slices.ContainsFunc(worn[:], func(it sim.ItemInstance) bool { return !it.Empty() })
			if armed && (c.human && (arm == mapload.ArmHumansByType || arm == mapload.ArmServerID) || !c.human && arm == mapload.ArmUnits) {
				at = i
				break
			}
		}
		if at < 0 {
			t.Fatalf("scn:%d has no armed placement of the population", c.mission)
		}
		tailPayload(t, b)[at*70+0x0c] |= 0x80
		bare, _ := tailSpawn(t, f, b)
		before, _ := w.EquippedItems(sim.EntityID(at))
		worn, _ := bare.EquippedItems(sim.EntityID(at))
		carried, _ := bare.CarriedItems(sim.EntityID(at))
		skipped := !slices.ContainsFunc(worn[:], func(it sim.ItemInstance) bool { return !it.Empty() }) && len(carried) == 0
		if c.human && !skipped || !c.human && fmt.Sprint(worn) != fmt.Sprint(before) {
			t.Fatalf("scn:%d placement %d with bit 7 holds %v and carries %v; without it %v", c.mission, at, worn, carried, before)
		}
	}
}

// loneShootingOriginals are the owner's EN SAVs of the original taken in
// mission 110, by name and SHA-256.
var loneShootingOriginals = []struct{ name, sum string }{
	{"game0023.sav", "39677b8b59e0d02484b66d728ec311124a9975fde234c48520dc739061c275c7"},
	{"game0025.sav", "e3b1e5d04894edf2abeed0b64a8c6509c77909e0892b4ad9eead259f45af9784"},
	{"game0026.sav", "6ca703a1068bcbb3d99b7e1e74ca5c48aa5c6bcb788a5659804cc6d3dda038bf"},
	{"game0027.sav", "e5604929a483ef3352b9929d928126f948252ff38854cca46ec33f1b54c41958"},
	{"game0028.sav", "7a228a381079647410c1b6230034bfc49f3824bffd5ceadac78e60d29bd8442e"},
}

// TestReleaseLoneShootingBytesKeepTheOriginalsShooting: scn:110's 28
// records whose only skill byte is Shooting 3 start with the skills their
// template derives at every difficulty, and on EN those six skill words
// equal the original's state in five original SAVs of the mission
// (DIV-3581).
func TestReleaseLoneShootingBytesKeepTheOriginalsShooting(t *testing.T) {
	f := releaseFront(t)
	addr, _ := MissionMap(110)
	b := tailMap(t, f, addr)
	p := tailPayload(t, b)
	bare := slices.Clone(b)
	lone := map[uint16]bool{}
	for at := 0; at+70 <= len(p); at += 70 {
		rec := p[at : at+70]
		if rec[0x3a] != 0 && !slices.ContainsFunc(rec[0x36:0x3a], func(v byte) bool { return v != 0 }) {
			lone[binary.LittleEndian.Uint16(rec[0x40:])] = true
			tailPayload(t, bare)[at+0x3a] = 0
		}
	}
	if len(lone) != 28 {
		t.Fatalf("scn:110 has %d lone Shooting records, want 28 (UNIT-PLACESKILL-086)", len(lone))
	}
	m, err := alm.Open(bare)
	if err != nil {
		t.Fatal(err)
	}
	starts := map[mapload.Difficulty]*sim.World{}
	for _, d := range []mapload.Difficulty{mapload.DifficultyEasy, mapload.DifficultyNormal, mapload.DifficultyHard} {
		ms, err := StartMission(f.Archives.Containers, 110, f.Table, d, nil)
		if err != nil {
			t.Fatal(err)
		}
		template, err := mapload.FromALMWith(m, f.Table, d)
		if err != nil {
			t.Fatal(err)
		}
		for u := range lone {
			got, want := tailEntity(t, ms.World, u), tailEntity(t, template, u)
			if got.Skill != want.Skill {
				t.Errorf("difficulty %d unit %d: skills %v, template %v", d, u, got.Skill, want.Skill)
			}
		}
		starts[d] = ms.World
	}
	if f.Base().Profile.Language == "russian" {
		return
	}
	root := os.Getenv("AGAINROM_SAVE_CORPUS")
	if root == "" {
		root = filepath.Join(os.Getenv("AGAINROM_ASSETS"), "..", "saves")
	}
	le := binary.LittleEndian
	for _, o := range loneShootingOriginals {
		raw, err := os.ReadFile(filepath.Join(root, "2026-10-14", "exp-teleport-series", o.name))
		if err != nil {
			t.Fatal(err)
		}
		if sum := sha256.Sum256(raw); hex.EncodeToString(sum[:]) != o.sum {
			t.Fatalf("%s: SHA-256 %x, want %s", o.name, sum, o.sum)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		d, err := campaignDifficulty(int64(doc.Head.Difficulty))
		if err != nil || doc.Head.Mission != 110 {
			t.Fatalf("%s: mission %d difficulty %d (%v)", o.name, doc.Head.Mission, doc.Head.Difficulty, err)
		}
		n := 0
		for _, r := range doc.Objects {
			if r.Class != "Unit" && r.Class != "Human" {
				continue
			}
			for _, v := range r.Values {
				if v.Name != "T08" || !lone[uint16(v.Value)] {
					continue
				}
				a := savedRecordRawForTest(t, r, "UA6")
				var original [data.SkillSlots]int32
				for k := range original {
					original[k] = int32(int16(le.Uint16(a[2+2*k:])))
				}
				if e := tailEntity(t, starts[d], uint16(v.Value)); e.Skill != original {
					t.Errorf("%s unit %d: the original holds skills %v, the engine's start %v", o.name, v.Value, original, e.Skill)
				}
				n++
			}
		}
		if n != 28 {
			t.Fatalf("%s holds %d actors of the lone Shooting records, want 28", o.name, n)
		}
	}
}
