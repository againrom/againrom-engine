package game

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/sim"
)

const radiusModID = "wide-blast"

func radiusModFront(t *testing.T, radius int) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	name := underscored(releaseSpellRows(t, f)[fireBallSpell-1].Name)
	dir := t.TempDir()
	files := map[string]string{
		"mod.toml":         "id = \"" + radiusModID + "\"\ntitle = \"Wide blast\"\nversion = \"1.0\"\n",
		"main.star":        "def init(game, settings):\n    game.data.add(\"data/spells.toml\")\n",
		"data/spells.toml": "[[spell]]\ntarget = \"" + name + "\"\nradius = " + strconv.Itoa(radius) + "\n",
	}
	for rel, body := range files {
		p := filepath.Join(dir, radiusModID, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := mod.Resolve(dir, []string{radiusModID})
	if err != nil {
		t.Fatal(err)
	}
	res, err := modrt.Load(entries, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.SetMods(res.Rules, res.Set, false); err != nil {
		t.Fatal(err)
	}
	if err := f.SetModSpells(res.Spells); err != nil {
		t.Fatal(err)
	}
	return f
}

func radiusMission(t *testing.T, f *FrontEnd) (casterID, allyID sim.EntityID) {
	t.Helper()
	party := MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
	ally := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)[0]
	ally.ID, ally.Name, ally.StartingHero = "wide-blast-ally", "Wide blast ally", false
	f.Carried = append(party, ally)
	app := f.App("wide blast")
	t.Cleanup(app.StopAudio)
	if err := app.OpenMission(f.MissionOpener(41)); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	casterID, allyID = mw.mission.ids[0], mw.mission.ids[1]
	caster, _ := mw.entity(casterID)
	rule, ok := mw.world.Spell(fireBallSpell)
	if !ok || caster.MaxMana == 0 {
		t.Fatalf("fixture: caster %+v rule %v", caster, ok)
	}
	if err := mw.world.HeadlessPlace(allyID, caster.X+5, caster.Y); err != nil {
		t.Fatal(err)
	}
	book := caster.Book
	if !book.HasInstances() {
		book.State = sim.BookPresent
	}
	book.Slots[fireBallSpell-1] = sim.BookSpell{Range: rule.MaxRange, ManaCost: uint16(rule.ManaCost)}
	if err := mw.world.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{
		ID: casterID, KnownSpells: caster.KnownSpells | 1<<fireBallSpell, Book: book,
	}}); err != nil {
		t.Fatal(err)
	}
	return casterID, allyID
}

func fireBallBursts(w *sim.World) int {
	n := 0
	for _, p := range w.SavedProjectiles().Items {
		if p.Picture == fireBallBurstPicture {
			n++
		}
	}
	return n
}

func savedAreaRadii(t *testing.T, raw []byte) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	var out []byte
	var walk func(r sav.DocumentRecordData)
	walk = func(r sav.DocumentRecordData) {
		for _, rd := range r.Raw {
			if rd.Name == "AE48" && len(rd.Bytes) == 4 {
				out = append(out, rd.Bytes[1])
			}
		}
		for _, g := range r.Groups {
			walk(g)
		}
		for _, in := range r.Inline {
			walk(in.Record)
		}
	}
	for _, o := range doc.Objects {
		walk(o)
	}
	return out
}

func TestReleaseModFireBallRadiusCoversTheMapAndSurvivesSaveAndLoad(t *testing.T) {
	const radius = 255
	f := radiusModFront(t, radius)
	casterID, allyID := radiusMission(t, f)
	mw := f.live
	if rule, _ := mw.world.Spell(fireBallSpell); rule.Radius != radius {
		t.Fatalf("the mission world's Fire_Ball radius is %d", rule.Radius)
	}
	bounds := mw.world.Bounds()
	cells := int(bounds.Width) * int(bounds.Height)
	tiles := int((bounds.Width+4)/3) * int((bounds.Height+4)/3)
	caster, _ := mw.entity(casterID)
	ally0, _ := mw.entity(allyID)

	mw.castAt(uint32(casterID), uint32(allyID), fireBallSpell)
	var inFlight []byte
	for tick := 0; fireBallBursts(mw.world) == 0; tick++ {
		if tick > 400 {
			t.Fatal("no burst was built")
		}
		now, _ := mw.entity(casterID)
		if inFlight == nil && now.Mana < caster.Mana {
			if len(mw.world.ScorchedCells()) != 0 {
				t.Fatal("the area had landed before the in-flight SAVE")
			}
			if raw := exportSave(t, f); len(savedAreaRadii(t, raw)) != 0 {
				inFlight = raw
			}
		}
		mw.tick()
	}
	if inFlight == nil {
		t.Fatal("no in-flight SAVE was taken")
	}
	if radii := savedAreaRadii(t, inFlight); !slices.Contains(radii, byte(radius)) {
		t.Fatalf("the in-flight SAV holds area radii %v, want %d", radii, radius)
	}
	if got := len(mw.world.ScorchedCells()); got != cells {
		t.Fatalf("%d scorched cells, want the map's %d", got, cells)
	}
	bursts := fireBallBursts(mw.world)
	if bursts < 2 || bursts > tiles {
		t.Fatalf("%d bursts, want between 2 and the map's %d tiles", bursts, tiles)
	}
	now, _ := mw.entity(casterID)
	ally, _ := mw.entity(allyID)
	if now.HP >= caster.HP || ally.HP >= ally0.HP {
		t.Fatalf("hp caster %d->%d ally %d->%d: the area hit neither", caster.HP, now.HP, ally0.HP, ally.HP)
	}
	t.Logf("map %dx%d: %d bursts (%d tiles), %d scorched", bounds.Width, bounds.Height, bursts, tiles, len(mw.world.ScorchedCells()))

	during := exportSave(t, f)
	cold := func(raw []byte) *FrontEnd {
		g := radiusModFront(t, radius)
		open, town, err := g.RestoreOriginal(raw)
		if err != nil || town {
			t.Fatalf("cold load: town %v err %v", town, err)
		}
		if err := g.App("cold load").OpenMission(open); err != nil {
			t.Fatal(err)
		}
		return g
	}
	restored := cold(during)
	if got := len(restored.live.world.ScorchedCells()); got != cells {
		t.Fatalf("cold LOAD during the bursts: %d scorched, want %d", got, cells)
	}
	if got := fireBallBursts(restored.live.world); got != bursts {
		t.Fatalf("cold LOAD during the bursts: %d bursts, want %d", got, bursts)
	}
	if rule, _ := restored.live.world.Spell(fireBallSpell); rule.Radius != radius {
		t.Fatalf("restored Fire_Ball radius %d", rule.Radius)
	}

	late := cold(inFlight)
	if got := len(late.live.world.ScorchedCells()); got != 0 {
		t.Fatalf("the in-flight SAV restored %d scorched cells", got)
	}
	for tick := 0; fireBallBursts(late.live.world) == 0; tick++ {
		if tick > 400 {
			t.Fatal("the restored cast never blasted")
		}
		late.live.tick()
	}
	if got := len(late.live.world.ScorchedCells()); got != cells {
		t.Fatalf("restored in-flight cast scorched %d cells, want %d", got, cells)
	}
	if got := fireBallBursts(late.live.world); got != bursts {
		t.Fatalf("restored in-flight cast drew %d bursts, want %d", got, bursts)
	}

	plain := releaseFront(t)
	if _, _, err := plain.RestoreOriginal(during); err == nil {
		t.Error("a game without the mod loaded the SAV written under it")
	}
}

func TestReleaseFireBallInstalledRadiusStaysOneBurstOverNineCells(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	casterID, allyID := radiusMission(t, f)
	mw := f.live
	if rule, _ := mw.world.Spell(fireBallSpell); rule.Radius != 1 {
		t.Fatalf("installed Fire_Ball radius %d, want 1", rule.Radius)
	}
	mw.castAt(uint32(casterID), uint32(allyID), fireBallSpell)
	for tick := 0; fireBallBursts(mw.world) == 0; tick++ {
		if tick > 400 {
			t.Fatal("no burst was built")
		}
		mw.tick()
	}
	if n, s := fireBallBursts(mw.world), len(mw.world.ScorchedCells()); n != 1 || s != 9 {
		t.Fatalf("%d bursts and %d scorched cells, want 1 and 9", n, s)
	}
}
