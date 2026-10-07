package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/sim"
)

func releaseSpellRows(t *testing.T, f *FrontEnd) []data.Spell {
	t.Helper()
	rows, err := data.LoadSpells(f.Table.Spells)
	if err != nil || len(rows) < 28 {
		t.Fatalf("installed spells: %d rows, %v", len(rows), err)
	}
	return rows
}

func underscored(name string) string { return strings.ReplaceAll(name, " ", "_") }

func spellModDir(t *testing.T, names []string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"mod.toml":      "id = \"spell-edit\"\ntitle = \"Spell edit\"\nversion = \"1.0\"\n",
		"main.star":     "def init(game, settings):\n    game.data.add(\"data/spells.toml\")\n",
		"settings.toml": "[mana_num]\ntype = \"int\"\ndefault = 1\nmin = 1\nmax = 10\n",
		"data/spells.toml": "[global]\nmana_mul = [\"@mana_num\", 2]\ndamage_mul = [3, 2]\nheal_mul = [1, 2]\nrange_mul = [3, 2]\nradius_mul = [2, 1]\nduration_mul = [3, 1]\n\n" +
			"[[spell]]\ntarget = \"" + underscored(names[3]) + "\"\nmana = 7\nrange = 21\narea_duration = 40\n\n" +
			"[[spell]]\ntarget = \"" + names[2] + "\"\nradius = 9\ndamage = { min = 5, max = 9 }\n\n" +
			"[[spell]]\ntarget = \"" + names[6] + "\"\nmana = 4\ndamage = { min = 20, max = 30 }\n",
	}
	for name, body := range files {
		p := filepath.Join(dir, "spell-edit", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func spellModFront(t *testing.T, dir string) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if dir == "" {
		return f
	}
	entries, err := mod.Resolve(dir, []string{"spell-edit"})
	if err != nil {
		t.Fatal(err)
	}
	res, err := modrt.Load(entries, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), []mod.SettingFlag{{Mod: "spell-edit", Key: "mana_num", Value: "3"}}, modrt.Options{})
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

func scaled(v int32, num, den, ceiling int64) int32 { return int32(min(int64(v)*num/den, ceiling)) }

func TestReleaseModSpellKeysEditTheInstalledTable(t *testing.T) {
	base := spellModFront(t, "")
	rows := releaseSpellRows(t, base)
	names := make([]string, len(rows)+1)
	for i, r := range rows {
		names[i+1] = r.Name
	}
	baseline := mapload.SpellRules(base.Table)
	if len(baseline) != len(rows) {
		t.Fatalf("%d rules for %d rows", len(baseline), len(rows))
	}

	f := spellModFront(t, spellModDir(t, names))
	got := mapload.SpellRules(f.Table)
	if len(got) != len(baseline) {
		t.Fatalf("%d rules under the mod, %d installed", len(got), len(baseline))
	}
	for i, row := range rows {
		id := i + 1
		b, g := baseline[i], got[i]
		wantMana := scaled(row.ManaCost, 3, 2, mod.SpellMaxMana)
		wantRange := scaled(row.MaxRange, 3, 2, mod.SpellMaxRange)
		wantRadius := uint8(scaled(int32(row.Radius), 2, 1, mod.SpellMaxRadius))
		wantMin, wantMax := row.DamageMin, row.DamageMax
		if row.DamageMin > 0 || row.DamageMax > 0 {
			ratio := [2]int64{3, 2}
			if row.Restorative {
				ratio = [2]int64{1, 2}
			}
			wantMin = scaled(row.DamageMin, ratio[0], ratio[1], mod.SpellMaxDamage)
			wantMax = scaled(row.DamageMax, ratio[0], ratio[1], mod.SpellMaxDamage)
		}
		wantArea := scaled(row.AreaDuration, 3, 1, mod.SpellMaxAreaLife)
		wantDuration := scaled(row.SpellDuration, 3, 1, mod.SpellMaxDuration)
		switch id {
		case 3:
			wantMana, wantRange, wantArea = 7, 21, 40
		case 2:
			wantRadius, wantMin, wantMax = 9, 5, 9
		case 6:
			wantMana, wantMin, wantMax = 4, 20, 30
		}
		if g.ManaCost != wantMana || int32(g.MaxRange) != wantRange || g.Radius != wantRadius || g.DamageMin != wantMin || g.DamageMax != wantMax ||
			g.AreaDuration != wantArea || g.SpellDuration != wantDuration {
			t.Errorf("row %d %q: mana %d range %d radius %d damage %d-%d area %d duration %d; want %d %d %d %d-%d %d %d", id, row.Name,
				g.ManaCost, g.MaxRange, g.Radius, g.DamageMin, g.DamageMax, g.AreaDuration, g.SpellDuration,
				wantMana, wantRange, wantRadius, wantMin, wantMax, wantArea, wantDuration)
		}
		if g.ID != b.ID || g.School != b.School || g.Delivery != b.Delivery || g.EffectSpeed != b.EffectSpeed || g.Complication != b.Complication ||
			g.TargetsUnit != b.TargetsUnit || g.Defensive != b.Defensive || g.Distribution != b.Distribution || g.Area != b.Area ||
			g.EffectKind != b.EffectKind || g.EffectMode != b.EffectMode || g.EffectMagnitude != b.EffectMagnitude {
			t.Errorf("row %d %q: a column no key names changed: %+v vs %+v", id, row.Name, g, b)
		}
	}

	again := mapload.SpellRules(spellModFront(t, "").Table)
	for i := range baseline {
		if again[i] != baseline[i] {
			t.Errorf("unmodded row %d differs between two front ends", i+1)
		}
	}

	book := func(front *FrontEnd, rules []sim.SpellRule) map[uint16][]string {
		e := sim.Entity{Mind: 30, Book: sim.Spellbook{State: sim.BookPresent}}
		for id := uint16(1); id <= 28; id++ {
			sim.LearnBookSpell(&e, id, rules)
		}
		out := map[uint16][]string{}
		for _, s := range spellbookOf(sim.Rules{}, e, rules, spellNamesFrom(front.Table, [29]string{}), testSpellPopupWords(), nil) {
			out[uint16(s.ID)] = s.Info
		}
		return out
	}
	before := book(base, baseline)
	after := book(f, got)
	for _, id := range []uint16{2, 3, 6} {
		if strings.Join(before[id], "|") == strings.Join(after[id], "|") {
			t.Errorf("spell %d popup is unchanged by the mod: %q", id, after[id])
		}
	}
	if !strings.Contains(strings.Join(after[3], "|"), ": 7") || !strings.Contains(strings.Join(after[6], "|"), "20-30") {
		t.Errorf("popup does not state the edited values: wall %q heal %q", after[3], after[6])
	}
}

func TestReleaseModSpellEditsReachACastAndTheSave(t *testing.T) {
	probe := spellModFront(t, "")
	rows := releaseSpellRows(t, probe)
	names := make([]string, len(rows)+1)
	for i, r := range rows {
		names[i+1] = r.Name
	}
	dir := spellModDir(t, names)
	baseline, _ := mapload.SpellRules(probe.Table), 0
	const wall = 3

	f := spellModFront(t, dir)
	skillCapMission(t, f)
	hero := skillCapHero(t, f)
	rule, ok := f.live.world.Spell(wall)
	if !ok || rule.ManaCost != 7 || rule.MaxRange != 21 || rule.ManaCost == baseline[wall-1].ManaCost {
		t.Fatalf("the mission world's spell %d is %+v (installed mana %d)", wall, rule, baseline[wall-1].ManaCost)
	}
	victim := releaseEntity(t, f.live, f.live.mission.ids[1])
	f.live.attackOrCast(uint32(f.live.mission.ids[0]), 0, wall, int(victim.X), int(victim.Y), true)
	for tick := 0; tick < 64; tick++ {
		f.live.tick()
	}
	if spent := hero.Mana - skillCapHero(t, f).Mana; spent < 7 || spent > 7+16 {
		t.Fatalf("the cast spent %d mana, want about the edited 7 (installed %d)", spent, baseline[wall-1].ManaCost)
	}

	control := spellModFront(t, "")
	skillCapMission(t, control)
	controlRange := savedSpellBytes(t, control, wall).rng
	bonus := controlRange - uint32(baseline[wall-1].MaxRange)
	liveRange := uint8(21 + bonus)
	raw := exportSave(t, f)
	got := savedSpellBytes(t, f, wall)
	if got.rng != uint32(liveRange) || got.mana != 7 {
		t.Errorf("the saved Spell record holds range %d mana %d, want %d and the edited 7", got.rng, got.mana, liveRange)
	}

	cold := spellModFront(t, dir)
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("cold load: town %v err %v", town, err)
	}
	if err := cold.App("cold load").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	restored := skillCapHero(t, cold)
	slot, ok := sim.BookRuleFor(restored, mapload.SpellRules(cold.Table)[wall-1])
	if !ok || slot.ManaCost != 7 || slot.MaxRange != 21 {
		t.Errorf("the restored mage's spell is %+v ok=%v, want the edited mana 7 and range 21", slot, ok)
	}

	plain := spellModFront(t, "")
	if _, _, err := plain.RestoreOriginal(raw); err == nil {
		t.Error("a game without the mod loaded the SAV written under it")
	}
}

func exportSave(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

type spellBytes struct{ rng, mana uint32 }

func savedSpellBytes(t *testing.T, f *FrontEnd, id uint32) spellBytes {
	t.Helper()
	doc, err := sav.DecodeDocumentData(exportSave(t, f))
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range doc.Objects {
		if o.Class == "Spell" && savedRecordValueForTest(t, o, "S08") == id {
			return spellBytes{savedRecordValueForTest(t, o, "S09"), savedRecordValueForTest(t, o, "S0C")}
		}
	}
	t.Fatal("the SAV holds no Spell record for the spell")
	return spellBytes{}
}
