package game

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/sim"
)

const (
	targetsModID = "target-filters"
	fireArrowID  = 1
	healSpellID  = 6
)

func targetsModFront(t *testing.T, spells func(name func(id int) string) string) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if spells == nil {
		return f
	}
	rows := releaseSpellRows(t, f)
	body := spells(func(id int) string { return underscored(rows[id-1].Name) })
	dir := t.TempDir()
	files := map[string]string{
		"mod.toml":         "id = \"" + targetsModID + "\"\ntitle = \"Target filters\"\nversion = \"1.0\"\n",
		"main.star":        "def init(game, settings):\n    game.data.add(\"data/spells.toml\")\n",
		"data/spells.toml": body,
	}
	for rel, text := range files {
		p := filepath.Join(dir, targetsModID, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := mod.Resolve(dir, []string{targetsModID})
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

func targetsMission(t *testing.T, f *FrontEnd) (casterID, allyID sim.EntityID) {
	t.Helper()
	casterID, allyID = radiusMission(t, f)
	mw := f.live
	caster, _ := mw.entity(casterID)
	book, known := caster.Book, caster.KnownSpells
	for _, id := range []uint32{fireArrowID, healSpellID} {
		rule, ok := mw.world.Spell(id)
		if !ok {
			t.Fatalf("no spell row %d", id)
		}
		slot := sim.BookSpell{Range: rule.MaxRange, ManaCost: uint16(rule.ManaCost)}
		if rule.Defensive {
			slot.Defensive = 1
		}
		book.Slots[id-1] = slot
		known |= 1 << id
	}
	if err := mw.world.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{ID: casterID, KnownSpells: known, Book: book}}); err != nil {
		t.Fatal(err)
	}
	return casterID, allyID
}

func hostileNear(t *testing.T, mw *mapWorld, party []sim.EntityID, x, y int32) sim.EntityID {
	t.Helper()
	relations := mw.world.Relations()
	for _, e := range mw.world.Entities() {
		if slices.Contains(party, e.ID) || e.MaxHP < 20 || !e.Alive() || e.OffMap || !relations.Hostile(sim.SelfSlot, e.Owner) {
			continue
		}
		if err := mw.world.HeadlessPlace(e.ID, x, y); err == nil {
			return e.ID
		}
	}
	t.Fatal("no hostile unit could be placed next to the party")
	return 0
}

func healMinimum(mw *mapWorld) int32 {
	rule, _ := mw.world.Spell(healSpellID)
	return max(rule.DamageMin, 5)
}

func hpOf(mw *mapWorld, id sim.EntityID) int32 {
	e, _ := mw.entity(id)
	return e.HP
}

func TestReleaseModHealHostileAndSelfCastReachTheCastAndSurviveSaveAndLoad(t *testing.T) {
	filters := func(name func(int) string) string {
		return "[[spell]]\ntarget = \"" + name(healSpellID) + "\"\nheal_hostile = true\n\n" +
			"[[spell]]\ntarget = \"" + name(fireArrowID) + "\"\nself_cast = true\n"
	}
	f := targetsModFront(t, filters)
	casterID, allyID := targetsMission(t, f)
	mw := f.live
	checkRows := func(w *sim.World) {
		t.Helper()
		if rule, _ := w.Spell(healSpellID); !rule.HealHostile || rule.SelfCast {
			t.Fatalf("heal row %+v", rule)
		}
		if rule, _ := w.Spell(fireArrowID); !rule.SelfCast || rule.HealHostile {
			t.Fatalf("fire arrow row %+v", rule)
		}
	}
	checkRows(mw.world)

	if refusal := mw.world.BookSpellRefusal(casterID, casterID, fireArrowID); refusal != "" {
		t.Fatalf("a self cast of the damaging row is refused: %q", refusal)
	}
	hp0 := hpOf(mw, casterID)
	mw.castAt(uint32(casterID), uint32(casterID), fireArrowID)
	for tick := 0; hpOf(mw, casterID) >= hp0; tick++ {
		if tick > 300 {
			t.Fatalf("the self cast never hurt the caster (hp %d)", hpOf(mw, casterID))
		}
		mw.tick()
	}

	caster, _ := mw.entity(casterID)
	enemy := hostileNear(t, mw, []sim.EntityID{casterID, allyID}, caster.X+2, caster.Y)
	if err := mw.world.HeadlessDamage(enemy, hpOf(mw, enemy)/2); err != nil {
		t.Fatal(err)
	}
	wounded := hpOf(mw, enemy)
	hostileBoth := func(w *mapWorld) bool {
		e, _ := w.entity(enemy)
		r := w.world.Relations()
		return r.Hostile(sim.SelfSlot, e.Owner) && r.Hostile(e.Owner, sim.SelfSlot)
	}
	if !hostileBoth(mw) {
		t.Fatal("fixture: the placed unit is not hostile both ways")
	}
	manaBefore := caster.Mana
	minHeal := healMinimum(mw)
	mw.castAt(uint32(casterID), uint32(enemy), healSpellID)
	for tick := 0; hpOf(mw, enemy) < wounded+minHeal; tick++ {
		if tick > 300 {
			t.Fatalf("the heal never reached the enemy (hp %d, wounded at %d)", hpOf(mw, enemy), wounded)
		}
		mw.tick()
	}
	if now, _ := mw.entity(casterID); now.Mana >= manaBefore {
		t.Errorf("the heal on the enemy cost no mana (%d to %d)", manaBefore, now.Mana)
	}
	if !hostileBoth(mw) {
		t.Error("the heal changed the diplomacy between the participant and the enemy")
	}

	saved := exportSave(t, f)
	g := targetsModFront(t, filters)
	open, town, err := g.RestoreOriginal(saved)
	if err != nil || town {
		t.Fatalf("cold load: town %v err %v", town, err)
	}
	if err := g.App("cold load").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	checkRows(g.live.world)
	if got := hpOf(g.live, enemy); got != hpOf(mw, enemy) {
		t.Errorf("the restored enemy holds %d health, saved %d", got, hpOf(mw, enemy))
	}
	if err := g.live.world.HeadlessDamage(enemy, hpOf(g.live, enemy)/2); err != nil {
		t.Fatal(err)
	}
	wounded = hpOf(g.live, enemy)
	g.live.castAt(uint32(casterID), uint32(enemy), healSpellID)
	for tick := 0; hpOf(g.live, enemy) < wounded+minHeal; tick++ {
		if tick > 300 {
			t.Fatalf("after the cold LOAD the heal never reached the enemy (hp %d, wounded at %d)", hpOf(g.live, enemy), wounded)
		}
		g.live.tick()
	}

	plain := releaseFront(t)
	if _, _, err := plain.RestoreOriginal(saved); err == nil {
		t.Error("a game without the mod loaded the SAV written under it")
	}
}

func TestReleaseWithoutTheFiltersAnEnemyHealAndASelfCastDoNothing(t *testing.T) {
	f := targetsModFront(t, nil)
	casterID, allyID := targetsMission(t, f)
	mw := f.live
	if refusal := mw.world.BookSpellRefusal(casterID, casterID, fireArrowID); refusal == "" {
		t.Error("the installed game admits a damaging self cast")
	}
	caster, _ := mw.entity(casterID)
	enemy := hostileNear(t, mw, []sim.EntityID{casterID, allyID}, caster.X+2, caster.Y)
	if err := mw.world.HeadlessDamage(enemy, hpOf(mw, enemy)/2); err != nil {
		t.Fatal(err)
	}
	wounded := hpOf(mw, enemy)
	minHeal := healMinimum(mw)
	mw.castAt(uint32(casterID), uint32(enemy), healSpellID)
	paid := false
	for tick := 0; tick < 60; tick++ {
		mw.tick()
		if now, _ := mw.entity(casterID); now.Mana < caster.Mana {
			paid = true
		}
		if hpOf(mw, enemy) >= wounded+minHeal {
			t.Fatalf("the installed game healed the enemy (%d from %d)", hpOf(mw, enemy), wounded)
		}
	}
	if !paid {
		t.Error("the refused heal was not paid, as the installed game pays it")
	}
}

func targetsArea(t *testing.T, filters string) (*FrontEnd, sim.EntityID, sim.EntityID, sim.EntityID) {
	t.Helper()
	f := targetsModFront(t, func(name func(int) string) string {
		return "[[spell]]\ntarget = \"" + name(fireBallSpell) + "\"\nradius = 3\n" + filters
	})
	casterID, allyID := targetsMission(t, f)
	mw := f.live
	caster, _ := mw.entity(casterID)
	if err := mw.world.HeadlessPlace(allyID, caster.X+2, caster.Y); err != nil {
		t.Fatal(err)
	}
	ally, _ := mw.entity(allyID)
	enemy := hostileNear(t, mw, []sim.EntityID{casterID, allyID}, ally.X, ally.Y+1)
	return f, casterID, allyID, enemy
}

func TestReleaseModAreaHitsKeepsABlastToHostileUnitsAcrossSaveAndLoad(t *testing.T) {
	f, casterID, allyID, enemyID := targetsArea(t, "area_hits = \"hostile\"\n")
	mw := f.live
	if rule, _ := mw.world.Spell(fireBallSpell); rule.AreaHits != sim.AreaHitsHostile || rule.Radius != 3 {
		t.Fatalf("mission world Fire_Ball: %+v", rule)
	}
	ids := [3]sim.EntityID{casterID, allyID, enemyID}
	var before [3]int32
	for i, id := range ids {
		before[i] = hpOf(mw, id)
	}
	caster, _ := mw.entity(casterID)
	mw.castAt(uint32(casterID), uint32(allyID), fireBallSpell)
	var inFlight []byte
	for tick := 0; fireBallBursts(mw.world) == 0; tick++ {
		if tick > 400 {
			t.Fatal("no burst was built")
		}
		if now, _ := mw.entity(casterID); inFlight == nil && now.Mana < caster.Mana && len(mw.world.ScorchedCells()) == 0 {
			if raw := exportSave(t, f); len(savedAreaRadii(t, raw)) != 0 {
				inFlight = raw
			}
		}
		mw.tick()
	}
	if inFlight == nil {
		t.Fatal("no in-flight SAVE was taken")
	}
	if hpOf(mw, enemyID) >= before[2] {
		t.Errorf("the blast left the enemy at %d health of %d", hpOf(mw, enemyID), before[2])
	}
	if hpOf(mw, casterID) < before[0] || hpOf(mw, allyID) < before[1] {
		t.Errorf("the blast hit the party: caster %d of %d, ally %d of %d",
			hpOf(mw, casterID), before[0], hpOf(mw, allyID), before[1])
	}

	g := targetsModFront(t, func(name func(int) string) string {
		return "[[spell]]\ntarget = \"" + name(fireBallSpell) + "\"\nradius = 3\narea_hits = \"hostile\"\n"
	})
	open, town, err := g.RestoreOriginal(inFlight)
	if err != nil || town {
		t.Fatalf("cold load: town %v err %v", town, err)
	}
	if err := g.App("cold load").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	late := g.live
	var restored [3]int32
	for i, id := range ids {
		restored[i] = hpOf(late, id)
	}
	for tick := 0; fireBallBursts(late.world) == 0; tick++ {
		if tick > 400 {
			t.Fatal("the restored cast never blasted")
		}
		late.tick()
	}
	if hpOf(late, enemyID) >= restored[2] {
		t.Errorf("the restored blast left the enemy at %d health of %d", hpOf(late, enemyID), restored[2])
	}
	if hpOf(late, casterID) < restored[0] || hpOf(late, allyID) < restored[1] {
		t.Errorf("the restored blast hit the party: caster %d of %d, ally %d of %d",
			hpOf(late, casterID), restored[0], hpOf(late, allyID), restored[1])
	}

	plain := releaseFront(t)
	if _, _, err := plain.RestoreOriginal(inFlight); err == nil {
		t.Error("a game without the mod loaded the SAV written under it")
	}
}

func TestReleaseWithoutAreaHitsTheSameBlastHitsEveryone(t *testing.T) {
	f, casterID, allyID, enemyID := targetsArea(t, "")
	mw := f.live
	ids := [3]sim.EntityID{casterID, allyID, enemyID}
	var before [3]int32
	for i, id := range ids {
		before[i] = hpOf(mw, id)
	}
	mw.castAt(uint32(casterID), uint32(allyID), fireBallSpell)
	for tick := 0; fireBallBursts(mw.world) == 0; tick++ {
		if tick > 400 {
			t.Fatal("no burst was built")
		}
		mw.tick()
	}
	for i, id := range ids {
		if hpOf(mw, id) >= before[i] {
			t.Errorf("unit %d holds %d of %d health: the blast missed it", id, hpOf(mw, id), before[i])
		}
	}
}
