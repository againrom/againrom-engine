package game

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestFlightProducerWithoutWindowConsumer(t *testing.T) {
	units, pictures := shotArchive(t)
	makeWorld := func() *mapWorld {
		mw := shotWorld(t, units, pictures, nil,
			shotShooter(1, shotClassArcher, 2, 4, 4), shotVictim(2, 6, 4))
		mw.bindFlightInputs()
		return mw
	}
	window, headless := makeWorld(), makeWorld()
	window.strike(1, 2)
	var commands = []sim.Command{sim.Attack(1, 2)}
	seen := false
	for tick := 0; tick < 24; tick++ {
		var report sim.Report
		window.tickStep(nil, true, func(r sim.Report) { report = r })
		sim.StepReported(headless.world, commands)
		commands = nil
		want, got := window.world.SavedProjectiles(), headless.world.SavedProjectiles()
		if len(want.Items) > 0 {
			seen = true
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("tick %d: ordinary Attack without window consumer has %+v; ordinary window tick has %+v", tick+1, got, want)
		}
		if headless.world.Hash() != window.world.Hash() {
			t.Fatalf("tick %d: window consumption changed the world hash", tick+1)
		}
		window.observeCasts(report.Casts)
		window.observeScriptCasts(report.ScriptCasts)
		window.observeAreaPaints(report.AreaPaints)
		window.advanceUnitShots()
		if !reflect.DeepEqual(window.world.SavedProjectiles(), want) || headless.world.Hash() != window.world.Hash() {
			t.Fatalf("tick %d: a second window consume changed records or hash", tick+1)
		}
	}
	if !seen {
		t.Fatal("ordinary ranged Attack never built a flight record")
	}
}

func TestFlightProducerCastConsumers(t *testing.T) {
	units, _ := shotArchive(t)
	pictures := spWorld(t).projectiles
	pictures.Sheets[17] = &terrain.EffectSheet{Phases: 8, RotationPhases: 1, Frames: spFrames(8, 8)}
	for _, family := range []string{"book", "staff", "weapon-rider", "staged", "fire-ball", "client-lightning", "client-chain"} {
		t.Run(family, func(t *testing.T) {
			spell := uint16(1)
			rule := sim.SpellRule{ID: 1, School: 1, ManaCost: 3, MaxRange: 8,
				TargetsUnit: true, Damaging: true, DamageMin: 4, DamageMax: 4}
			if family == "staged" {
				spell, rule.ID, rule.Area, rule.TargetsUnit, rule.Distribution, rule.Radius = 4, 4, true, false, 5, 2
				rule.SelfCast, rule.MaxRange, rule.AreaDuration = true, 0, 1
			}
			if family == "fire-ball" {
				spell, rule.ID, rule.Area, rule.TargetsUnit, rule.Distribution, rule.Radius = 2, 2, true, false, 3, 1
				rule.Delivery, rule.EffectSpeed = 2, 128
			}
			if family == "client-lightning" || family == "client-chain" {
				spell, rule.ID = 13, 13
				if family == "client-chain" {
					spell, rule.ID = 14, 14
				}
			}
			makeWorld := func() *mapWorld {
				caster := shotShooter(1, shotClassArcher, 14, 14, 4)
				caster.AttackRelax, caster.ScanRange, caster.Mind = 60, 12, 30
				caster.Mana, caster.MaxMana, caster.KnownSpells = 100, 100, 1<<spell
				caster.AlwaysHits, caster.DamageBase = true, 5
				if family == "staff" || family == "weapon-rider" {
					caster.WeaponSpell, caster.WeaponSpellLevel = spell, 30
				}
				if family == "weapon-rider" {
					caster.Mana, caster.MaxMana = 0, 0
				}
				w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical, nil,
					[]sim.Entity{caster, shotVictim(2, 18, 14)}, nil, []sim.SpellRule{rule})
				if err != nil {
					t.Fatal(err)
				}
				viewer, err := ui.NewViewer("flight producer", terrain.Grid{Width: 32, Height: 32, Tiles: make([]uint16, 32*32)}, &terrain.Tileset{})
				if err != nil {
					t.Fatal(err)
				}
				mw := newMapWorld(w, nil, units, viewer)
				mw.projectiles = pictures
				if family == "client-lightning" || family == "client-chain" {
					if err := w.DeclareCellTails([]sim.CellTail{{X: 15, Y: 14, Bytes: [6]byte{byte(spell), 30, 11, 14, 0, 0}}}); err != nil {
						t.Fatal(err)
					}
				}
				mw.bindFlightInputs()
				return mw
			}
			window, headless := makeWorld(), makeWorld()
			if family == "staged" {
				if why := window.world.BookSpellRefusal(1, 1, 4); why != "" {
					t.Fatalf("ordinary staged self cast refused: %s", why)
				}
			}
			command := sim.Cast(1, 2, sim.SpellID(spell))
			if family == "client-lightning" || family == "client-chain" {
				command = sim.MoveTo(1, sim.CellPoint{X: 15, Y: 14})
				window.enqueue(1, 15, 14)
			} else if family == "staff" || family == "weapon-rider" {
				window.strike(1, 2)
				command = sim.Attack(1, 2)
			} else if family == "staged" {
				command = sim.Cast(1, 1, sim.SpellID(spell))
				window.castAt(1, 1, uint32(spell))
			} else {
				window.castAt(1, 2, uint32(spell))
			}
			seen, casts, paints := false, 0, 0
			for tick := 0; tick < 40; tick++ {
				var commands []sim.Command
				if tick == 0 {
					commands = []sim.Command{command}
				}
				var report sim.Report
				window.tickStep(nil, true, func(r sim.Report) { report = r })
				casts, paints = casts+len(report.Casts), paints+len(report.AreaPaints)
				sim.StepReported(headless.world, commands)
				want := window.world.SavedProjectiles()
				seen = seen || len(want.Items) > 0
				if !reflect.DeepEqual(headless.world.SavedProjectiles(), want) || headless.world.Hash() != window.world.Hash() {
					t.Fatalf("tick %d: %s window/headless records or hash differ", tick+1, family)
				}
				window.observeCasts(report.Casts)
				window.observeScriptCasts(report.ScriptCasts)
				window.observeAreaPaints(report.AreaPaints)
				if !reflect.DeepEqual(window.world.SavedProjectiles(), want) || headless.world.Hash() != window.world.Hash() {
					t.Fatalf("tick %d: second %s consumer changed records or hash", tick+1, family)
				}
			}
			if !seen {
				t.Fatalf("ordinary %s commands never produced a record; casts=%d paints=%d records=%+v", family, casts, paints, window.world.SavedProjectiles())
			}
		})
	}
}

func TestReleaseFlightProducerScenarioMissionWithoutConsumer(t *testing.T) {
	f := releaseFront(t)
	p, _, err := StartScenarioMission(f.Archives, HeadlessScenario{Version: 6, Stage: StageMission, Mission: 41, Mage: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Party) != 1 {
		t.Fatalf("scenario party %v", p.Party)
	}
	caster, _ := p.World.Entity(p.Party[0])
	var target sim.Entity
	for _, e := range p.World.EntityView() {
		if e.Alive() && e.ID != caster.ID && !e.OffMap {
			target = e
			break
		}
	}
	if target.ID == caster.ID || !target.Alive() {
		t.Fatal("mission 41 has no target")
	}
	if err := p.World.HeadlessPlace(target.ID, caster.X+5, caster.Y); err != nil {
		t.Fatal(err)
	}
	flightLearn(t, &mapWorld{world: p.World}, caster.ID, 1)
	seen := false
	for tick := 0; tick < 64; tick++ {
		var commands []sim.Command
		if tick == 0 {
			commands = []sim.Command{sim.Cast(caster.ID, target.ID, 1)}
		}
		report := sim.StepReported(p.World, commands)
		for _, event := range report.Casts {
			if event.Caster != caster.ID || event.Weapon || event.Spell != 1 {
				continue
			}
			if !event.Produced || len(event.Records) != 1 {
				t.Fatalf("ordinary scenario cast produced record identities %+v", event)
			}
			for _, record := range p.World.SavedProjectiles().Items {
				seen = seen || record.ID == event.Records[0] && record.Picture == 10
			}
		}
		if seen {
			break
		}
	}
	if !seen {
		t.Fatalf("ordinary scenario Cast without mapWorld consumer produced %+v", p.World.SavedProjectiles())
	}
}

type producerFrame struct {
	Records sim.SavedProjectiles
	Drivers *sim.SavedWorldEffects
	Hash    uint64
}

type producerResume struct {
	SAV    string
	Route  string
	Frames []producerFrame
}

func producerFrameNow(mw *mapWorld) producerFrame {
	return producerFrame{Records: mw.world.SavedProjectiles(), Drivers: mw.world.SavedWorldEffectDrivers(), Hash: mw.world.Hash()}
}

func producerCheckFrame(t *testing.T, tick int, got, want producerFrame) {
	t.Helper()
	if !reflect.DeepEqual(got.Records, want.Records) || !reflect.DeepEqual(got.Drivers, want.Drivers) || got.Hash != want.Hash {
		t.Fatalf("cold process tick %d: records %+v drivers %+v hash %016x; want records %+v drivers %+v hash %016x",
			tick, got.Records, got.Drivers, got.Hash, want.Records, want.Drivers, want.Hash)
	}
}

func producerColdProcess(t *testing.T, path string) {
	t.Helper()
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var want producerResume
	if err := json.Unmarshal(encoded, &want); err != nil {
		t.Fatal(err)
	}
	if want.Route != "ordinary-headless-tick" {
		t.Fatalf("unexpected cold continuation route %q", want.Route)
	}
	raw, err := os.ReadFile(want.SAV)
	if err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "flight-cold.sav")
	t.Cleanup(app.StopAudio)
	for tick, frame := range want.Frames {
		if tick > 0 {
			f.live.tickStep(nil, false)
		}
		producerCheckFrame(t, tick, producerFrameNow(f.live), frame)
	}
	if len(want.Frames) != flightWindow+1 {
		t.Fatal("cold process did not compare the full continuation")
	}
	t.Logf("fresh process %s compared all record fields, allocator, IDs, drivers and hash for %d ticks", want.Route, len(want.Frames))
}

func producerAssertCurrentSave(t *testing.T, raw []byte, source producerFrame, structures []sim.Structure) {
	t.Helper()
	store := kitProjectileStore(t, raw)
	if source.Records.FreeIndex != store.FreeIndex || !slices.Equal(source.Records.IDs, store.IDs) || len(source.Records.Items) != len(store.Items) {
		t.Fatal("SAV lost the current allocator, identities or record count")
	}
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	buildings, _, err := file.Buildings()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range source.Records.Items {
		expected := p
		if source.Drivers != nil {
			for _, d := range source.Drivers.Projectiles {
				if d.ID != p.ID || !d.HasTarget || !d.TargetStructure {
					continue
				}
				matches := 0
				for _, structure := range structures {
					if structure.ID != sim.StructureID(d.Target) {
						continue
					}
					for _, building := range buildings {
						if int32(building.Col) == structure.Col && int32(building.Row) == structure.Row && uint16(building.Kind) == structure.Kind {
							expected.ActionTarget = int32(building.RuntimeID)
							matches++
						}
					}
				}
				if matches != 1 || expected.ActionTarget == 0 {
					t.Fatalf("record %d target structure has %d saved identities", p.ID, matches)
				}
			}
		}
		found := false
		for _, written := range store.Items {
			if written.ID != p.ID {
				continue
			}
			got := savedProjectileFromSav(written)
			got.ActionTarget = written.ActionTarget
			if got != expected {
				t.Fatalf("SAV record %+v, current record in SAV target namespace %+v", got, expected)
			}
			found = true
		}
		if !found {
			t.Fatalf("SAV dropped current record %d", p.ID)
		}
	}
}

func TestReleaseFlightProducerSaveColdProcess(t *testing.T) {
	if path := os.Getenv("AGAINROM_FLIGHT_CHILD"); path != "" {
		producerColdProcess(t, path)
		return
	}
	for _, family := range []string{"book", "staff", "staged", "fire-ball", "client-lightning", "client-chain", "unit", "structure"} {
		t.Run(family, func(t *testing.T) {
			if family == "unit" || family == "structure" {
				producerPhysical(t, family)
				return
			}
			f, app, dir, caster, target := flightMageArena(t, family == "staff")
			mw := f.live
			spell, picture := uint32(1), int32(10)
			switch family {
			case "client-lightning", "client-chain":
				spell = 13
				if family == "client-chain" {
					spell = 14
				}
				picture = int32(2*spell + 8)
				actor, _ := mw.entity(caster)
				if err := mw.world.DeclareCellTails([]sim.CellTail{{X: actor.X + 1, Y: actor.Y,
					Bytes: [6]byte{byte(spell), 30, byte(actor.X - 3), byte(actor.Y), 0, 0}}}); err != nil {
					t.Fatal(err)
				}
				mw.enqueue(uint32(caster), int(actor.X+1), int(actor.Y))
			case "staff":
				e, _ := mw.entity(caster)
				if e.WeaponSpell != 1 {
					t.Fatalf("installed staff spell %d", e.WeaponSpell)
				}
				mw.strike(uint32(caster), uint32(target.ID))
			case "staged":
				spell, picture = 4, 17
				flightLearn(t, mw, caster, spell)
				mw.castAt(uint32(caster), uint32(caster), spell)
			default:
				if family == "fire-ball" {
					spell, picture = 2, 13
				}
				flightLearn(t, mw, caster, spell)
				mw.castAt(uint32(caster), uint32(target.ID), spell)
			}
			born := flightAwait(t, f, app, func(p sim.SavedProjectile, _ sim.SavedProjectileDriver) bool {
				return p.Picture == picture && p.ActionSegments > 1
			})
			if family == "client-lightning" || family == "client-chain" {
				target, _ = mw.entity(caster)
				if born.ActionPhase != 0 || born.Dir != 0 || born.ActionDir != 0 {
					t.Fatalf("cell-entry client record %+v", born)
				}
			}
			source, structures := producerFrameNow(mw), mw.world.Structures()
			raw, run := flightWitness(t, app, dir, mw, born, unitEffect(target), "flight producer "+family)
			producerAssertCurrentSave(t, raw, source, structures)
			if (family == "book" || family == "staff") && flightLanded(run) < 0 {
				t.Fatal("the witnessed release never damaged its target")
			}
			producerFreshProcess(t, raw, dir)
			t.Logf("%s record %d picture %d: SAV fields, drop/change controls, cold LOAD and fresh process passed", family, born.ID, born.Picture)
		})
	}
}

func producerFreshProcess(t *testing.T, raw []byte, dir string) {
	t.Helper()
	baseline := releaseFront(t)
	loaded, _ := openOriginalSAVApp(t, baseline, raw, "flight-baseline.sav")
	t.Cleanup(loaded.StopAudio)
	want := producerResume{SAV: filepath.Join(dir, "producer.sav"), Route: "ordinary-headless-tick"}
	for tick := 0; tick <= flightWindow; tick++ {
		if tick > 0 {
			baseline.live.tick()
		}
		want.Frames = append(want.Frames, producerFrameNow(baseline.live))
	}
	if err := os.WriteFile(want.SAV, raw, 0600); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "producer-expectation.json")
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command(executable, "-test.run=^TestReleaseFlightProducerSaveColdProcess$", "-test.v")
	child.Env = append(os.Environ(), "AGAINROM_FLIGHT_CHILD="+path)
	if output, err := child.CombinedOutput(); err != nil {
		t.Fatalf("fresh-process SAV continuation: %v\n%s", err, output)
	} else {
		t.Logf("%s", output)
	}
}

func producerPhysical(t *testing.T, family string) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	bow, err := resolveWeaponForSlot(false, false, f.Table.Shapes, f.Table.Materials, f.Table.Weapons, data.SkillShoot)
	if err != nil {
		t.Fatal(err)
	}
	party := MissionPartyAs(false, bow, f.Bodies, f.Table)
	fighter := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)[0]
	fighter.ID, fighter.Name, fighter.StartingHero = "flight-target", "Flight target", false
	f.Carried = append(party[:1], fighter)
	mission := 41
	if family == "structure" {
		mission = 10
		f.Carried = party[:1]
	}
	app := f.App("physical flight producer")
	t.Cleanup(app.StopAudio)
	dir := t.TempDir()
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	if err := app.OpenMission(f.MissionOpener(mission)); err != nil {
		t.Fatal(err)
	}
	for k := 0; k < 16 && app.HeadlessNoticeOpen(); k++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	mw := f.live
	shooter := mw.mission.ids[0]
	hero, _ := mw.entity(shooter)
	_, delay, drawn := mw.classShot(mw.spellClientClass(shooter, hero.Class))
	if !drawn || delay < 2 {
		t.Fatalf("installed bow release drawn=%v delay=%d; pre-release SAV requires at least two ticks", drawn, delay)
	}
	var effect flightEffect
	if family == "structure" {
		const id = sim.StructureID(4)
		var target sim.Structure
		for _, s := range mw.world.Structures() {
			if s.ID == id {
				target = s
			}
		}
		if target.Width == 0 {
			t.Fatal("mission 10 has no structure 4")
		}
		if err := mw.world.HeadlessPlace(shooter, target.Col-6, target.Row); err != nil {
			t.Fatal(err)
		}
		mw.pending = append(mw.pending, sim.AttackStructure(shooter, id))
		mw.commanded[shooter] = true
		effect = structureEffect(id)
	} else {
		target, _ := mw.entity(mw.mission.ids[1])
		if err := mw.world.HeadlessPlace(target.ID, hero.X+5, hero.Y); err != nil {
			t.Fatal(err)
		}
		target, _ = mw.entity(target.ID)
		mw.strike(uint32(shooter), uint32(target.ID))
		effect = unitEffect(target)
	}
	charging := false
	for tick := 0; tick < 120; tick++ {
		mw.tick()
		e, _ := mw.entity(shooter)
		if e.AttackPhase == sim.AttackCharging && e.FlightRun.Active && e.FlightRun.Elapsed == 0 {
			charging = true
			break
		}
	}
	if !charging {
		e, _ := mw.entity(shooter)
		t.Fatalf("ordinary bow attack never entered its pre-release wind-up: %+v", e)
	}
	if mw.world.SavedProjectiles().FreeIndex != 0 {
		t.Fatal("ordinary physical attack released before its installed delay")
	}
	for k := 0; k < 8 && app.HeadlessNoticeOpen(); k++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	pre := cityRosterF2Save(t, app, SaveStore{Dir: dir}, "physical before release")
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	before := releaseFront(t)
	loaded, _ := openOriginalSAVApp(t, before, pre, "before-release.sav")
	t.Cleanup(loaded.StopAudio)
	seen := false
	for tick := 0; tick <= delay+4; tick++ {
		if !reflect.DeepEqual(mw.world.SavedProjectiles(), before.live.world.SavedProjectiles()) {
			t.Fatalf("pre-release SAV tick %d changed records: %+v, want %+v", tick, before.live.world.SavedProjectiles(), mw.world.SavedProjectiles())
		}
		seen = seen || len(mw.world.SavedProjectiles().Items) > 0
		mw.tick()
		before.live.tick()
	}
	if !seen {
		t.Fatal("pre-release SAV continuation never released the pending physical shot")
	}
	born := flightAwait(t, f, app, func(p sim.SavedProjectile, d sim.SavedProjectileDriver) bool {
		return d.TargetStructure == (family == "structure") && p.Picture <= 12 && p.ActionSegments > 1
	})
	source, structures := producerFrameNow(mw), mw.world.Structures()
	raw, _ := flightWitness(t, app, dir, mw, born, effect, "physical "+family)
	producerAssertCurrentSave(t, raw, source, structures)
	producerFreshProcess(t, raw, dir)
	t.Logf("%s installed delay %d: pre-release SAV, in-flight current fields, loss controls and fresh-process continuation passed", family, delay)
}
