package game

import (
	"bytes"
	"testing"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Controlled installed-data arena, not an original-game runtime witness or a
// claim about campaign acquisition. Only actor population/placement/book and
// presentation fog are fixtures. The map, Building definitions, terrain, spell
// rows, art, App input, damage, inspection and native SaveStore are production.
func TestReleaseMultiCellStructureAreaSpell(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
	hero.Skill[1] = 100
	party := []mapload.PartyMember{{ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true,
		Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero, KnownSpells: 1 << 2,
		Saved: &mapload.Saved{Cell: mapload.Cell{X: 29, Y: 50}, HP: 100, MaxHP: 100, Mana: 1000, MaxMana: 1000, HealthRegenPeriod: 100, ManaRegenPeriod: 50}}}
	a := f.App("1086 multicell spell witness")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	live := f.live
	actor, ok := live.entity(live.mission.ids[0])
	if !ok {
		t.Fatal("no mage")
	}
	const sid = 4
	target := live.world.Structures()[sid]
	if target.Col != 33 || target.Row != 49 || target.Width != 3 || target.Height != 3 || target.Attach != 511 || target.Field42 != 1000 || target.MaxHealth != 1000 {
		t.Fatalf("installed mission-10 target4 changed: %+v", target)
	}
	actor.X, actor.Y = 29, 50
	actor.TargetX, actor.TargetY, actor.HasTarget = 0, 0, false
	actor.Transit, actor.TransitTotal = 0, 0
	m := live.mission.state.Map
	w, err := sim.NewStructuredWorld(1086, live.world.Bounds(), sim.ModeCanonical,
		sim.Terrain{Block: mapload.PassabilityWith(m, f.Table), Cost: mapload.Cost(m), Height: mapload.Height(m)},
		[]sim.Entity{actor}, nil, live.world.Relations(), nil, nil, mapload.SpellRules(f.Table), sim.GhostTemplate{}, live.world.Structures())
	if err != nil {
		t.Fatal(err)
	}
	live.world = w
	live.commanded[actor.ID] = true
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	live.push()
	inspectionCentre(live, int(actor.X), int(actor.Y))
	if err := a.HeadlessSelectEntity(uint32(actor.ID)); err != nil {
		t.Fatal(err)
	}
	live.push()
	if _, _, err := a.HeadlessSpellPoint(2); err != nil {
		if err := a.HeadlessKey("book"); err != nil {
			t.Fatal(err)
		}
	}
	inspectionCentre(live, 34, 50)
	ref := ui.InspectionSubject{Kind: ui.InspectionStructure, ID: sid}
	releaseHoverInspection(t, a, live, ref)
	card, err := a.HeadlessMissionCard()
	if err != nil {
		t.Fatal(err)
	}
	beforePixels := append([]byte(nil), card.Pix...)
	if entries, ruined := live.view.StructureRuinFrames(sid); entries == 0 || ruined != 0 {
		t.Fatalf("intact art=%d/%d", ruined, entries)
	}
	multiCellSave1086(t, f, "before", sid)

	// Locate the target's centre cell through the headless inverse hit test,
	// then send real App pointer events. No live HP assignment occurs here.
	px, py := -1, -1
	for y := 100; y < 650 && px < 0; y += 3 {
		for x := 100; x < 800; x += 3 {
			col, row, e := a.HeadlessDropCell(x, y)
			if e == nil && col == 34 && row == 50 {
				px, py = x, y
				break
			}
		}
	}
	if px < 0 {
		t.Fatal("target centre not in map viewport")
	}
	sx, sy, e := a.HeadlessSpellPoint(2)
	if e != nil {
		t.Fatal(e)
	}
	for _, edge := range []string{"press", "release"} {
		if e := a.HeadlessPointer(edge, sx, sy); e != nil {
			t.Fatal(e)
		}
	}
	casts := 0
	for casts < 20 && live.world.Structures()[sid].Field42 != 0 {
		for _, edge := range []string{"press", "release"} {
			if e := a.HeadlessPointer(edge, px, py); e != nil {
				t.Fatal(e)
			}
		}
		if len(live.pending) != 1 || live.pending[0].Kind != sim.KindCastAt || live.pending[0].Spell != 2 || live.pending[0].X != 34 || live.pending[0].Y != 50 {
			t.Fatalf("App spell click queued %+v", live.pending)
		}
		before := live.world.Structures()[sid].Field42
		live.tick()
		if casts == 0 {
			multiCellSave1086(t, f, "pending book windup", sid)
		}
		for tick := 0; tick < 512; tick++ {
			live.tick()
			state, _ := live.entity(actor.ID)
			if _, _, active := live.world.CastingSpell(actor.ID); !active && state.CastWait == 0 {
				break
			}
		}
		casts++
		after := live.world.Structures()[sid].Field42
		if after >= before {
			actorNow, _ := live.entity(actor.ID)
			t.Fatalf("cast%d did not damage structure: %d->%d actor=%+v", casts, before, after, actorNow)
		}
		t.Logf("installed Fire Ball cast%d target4 HP %d->%d (3x3 mask 0x1ff)", casts, before, after)
	}
	if got := live.world.Structures()[sid].Field42; got != 0 {
		t.Fatalf("multicell structure survived at HP%d", got)
	}
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	live.push()
	releaseHoverInspection(t, a, live, ref)
	panel, ok := live.view.InspectionPanel()
	if !ok || panel.HP != 0 || panel.MaxHP != 1000 {
		t.Fatalf("ruined HP panel=%+v", panel)
	}
	card, err = a.HeadlessMissionCard()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(beforePixels, card.Pix) {
		t.Fatal("HP card pixels unchanged")
	}
	entries, ruined := live.view.StructureRuinFrames(sid)
	if entries == 0 || entries != ruined {
		t.Fatalf("ruin art=%d/%d", ruined, entries)
	}
	multiCellSave1086(t, f, "after ruin", sid)
	t.Logf("target4 HP1000->0, casts=%d, ruin frames=%d/%d, HP card pixels changed; before/pending/after SaveStore keeps structure HP and ruin frames", casts, ruined, entries)
}

func multiCellSave1086(t *testing.T, f *FrontEnd, stage string, sid uint32) {
	t.Helper()
	dir := t.TempDir()
	save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, func() time.Time { return time.Unix(1000, 0) })
	name, err := save(true)
	if err != nil {
		t.Fatalf("%s save: %v", stage, err)
	}
	restored := releaseFront(t)
	_, _, load := restored.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	open, town, err := load(localOriginalSaveToken(name))
	if err != nil || town || open == nil {
		t.Fatalf("%s restore: %v town%v", stage, err, town)
	}
	ra := restored.App("1086 restored " + stage)
	ra.Layout(1024, 768)
	if err := ra.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	live, back := f.live.world.Structures(), restored.live.world.Structures()
	if len(live) != len(back) {
		t.Fatalf("%s structures %d -> %d", stage, len(live), len(back))
	}
	for i := range live {
		if live[i].Field42 != back[i].Field42 || live[i].MaxHealth != back[i].MaxHealth {
			t.Fatalf("%s structure%d HP %d/%d -> %d/%d", stage, i, live[i].Field42, live[i].MaxHealth, back[i].Field42, back[i].MaxHealth)
		}
	}
	restored.live.push()
	entries, ruined := f.live.view.StructureRuinFrames(sid)
	backEntries, backRuined := restored.live.view.StructureRuinFrames(sid)
	if entries == 0 || entries != backEntries || ruined != backRuined {
		t.Fatalf("%s ruin frames %d/%d -> %d/%d", stage, ruined, entries, backRuined, backEntries)
	}
}
