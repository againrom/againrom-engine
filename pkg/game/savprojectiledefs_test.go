package game

import (
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mod"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
	"againrom/pkg/vfs"
)

func projectileDefsFront(t *testing.T, modded bool) *FrontEnd {
	t.Helper()
	f := restoredFacingFront(t)
	f.Archives.Base = base.Match{Profile: base.Profile{Game: base.GameROM1}}
	f.Table.Edition = &base.Edition{Game: base.GameROM1, Spells: "rom1"}
	rows := dbCollection{{}}
	for _, s := range rules.SpellProfile(rules.SpellGameFirst) {
		rows = append(rows, dbEntry{name: strings.ReplaceAll(s.Name, "_", " "),
			params: []int32{0, 7, 1, 0, 0, 0, 20, 0, 3, 1, 0, 0, 0, 0, 0, 0, 2, 2, 0}})
	}
	f.Table.Spells = rows
	in, err := loadFlightInputs(f.Archives.Containers, data.FirstSpellPresentation(), firstGameFiles.flight)
	if err != nil {
		t.Fatal(err)
	}
	f.Table.FlightInputs = in
	if modded {
		gold := color.RGBA{R: 0xd0, G: 0xb0, A: 0xff}
		src := "[[spell]]\ntarget = \"Fire_Arrow\"\n" +
			"cast = \"assets/spells/fire-cast.256\"\n" +
			"cast_row = { phases = 7, rotation_phases = 1 }\n" +
			"burst = \"assets/spells/fire-burst.256\"\n"
		files := map[string][]byte{
			mod.SpellsFile:                 []byte(src),
			"assets/spells/fire-cast.256":  spellPictureSheet256(7, gold),
			"assets/spells/fire-burst.256": spellPictureSheet256(3, gold),
		}
		if err := f.SetModSpells(spellPicturesMod(t, files)); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func TestMissionAndLoadedSAVShareProjectileDefinitions(t *testing.T) {
	for _, tc := range []struct {
		name       string
		modded     bool
		wantPhases uint16
	}{
		{name: "install", modded: false, wantPhases: restoredBoltPhases},
		{name: "modded", modded: true, wantPhases: 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mission := projectileDefsFront(t, tc.modded)
			app := mission.App("mission projectile defs")
			app.Layout(1024, 768)
			t.Cleanup(app.StopAudio)
			if err := app.OpenMission(mission.MissionOpener(10)); err != nil {
				t.Fatal(err)
			}
			mage := sim.Entity{HP: 100, MaxHP: 100, Owner: sim.SelfSlot, TokenSize: 1,
				Mind: 40, Mana: 100, MaxMana: 100, KnownSpells: 1 << 1,
				Reach: 1, AttackCharge: 2, AttackRelax: 60, ScanRange: 30,
				Book: sim.Spellbook{State: sim.BookPresent}}
			mage.Book.Slots[0] = sim.BookSpell{Range: 20, ManaCost: 1}
			id, err := mission.live.world.CheatSummon(mage, nil, [sim.EquipSlots]sim.ItemInstance{}, 8, 8)
			if err != nil {
				t.Fatal(err)
			}
			mission.live.attackOrCast(uint32(id), 0, 1, 20, 8, true)
			for tick := 0; tick < 32 && len(mission.live.world.SavedProjectiles().Items) == 0; tick++ {
				mission.live.tick()
			}
			if items := mission.live.world.SavedProjectiles().Items; len(items) == 0 || items[0].Picture != restoredBoltPicture {
				t.Fatalf("mission cast produced %+v, want picture %d", items, restoredBoltPicture)
			}
			missionDrivers := mission.live.world.SavedWorldEffectDrivers()
			if missionDrivers == nil || len(missionDrivers.Projectiles) == 0 {
				t.Fatal("mission projectile has no driver")
			}
			if got := missionDrivers.Projectiles[0].Phases; got != tc.wantPhases {
				t.Fatalf("mission driver phases = %d, want %d", got, tc.wantPhases)
			}

			raw := phaseSoundSave(t, mission)
			loaded := projectileDefsFront(t, tc.modded)
			loadedApp, _ := openOriginalSAVApp(t, loaded, raw, "projectile-defs.sav")
			t.Cleanup(loadedApp.StopAudio)

			if !reflect.DeepEqual(mission.Table.FlightInputs.Pictures[restoredBoltPicture], loaded.Table.FlightInputs.Pictures[restoredBoltPicture]) {
				t.Fatalf("flight picture differs: mission=%+v loaded=%+v",
					mission.Table.FlightInputs.Pictures[restoredBoltPicture],
					loaded.Table.FlightInputs.Pictures[restoredBoltPicture])
			}
			mSheet, lSheet := mission.Projectiles.Sheet(restoredBoltPicture), loaded.Projectiles.Sheet(restoredBoltPicture)
			if mSheet == nil || lSheet == nil ||
				mSheet.Phases != lSheet.Phases || mSheet.RotationPhases != lSheet.RotationPhases ||
				mSheet.Flip != lSheet.Flip || mSheet.Clock != lSheet.Clock ||
				mSheet.CenterX != lSheet.CenterX || mSheet.CenterY != lSheet.CenterY ||
				len(mSheet.Frames) != len(lSheet.Frames) {
				t.Fatalf("projectile art differs: mission=%+v loaded=%+v", mSheet, lSheet)
			}
			loadedDrivers := loaded.live.world.SavedWorldEffectDrivers()
			if loadedDrivers == nil || len(loadedDrivers.Projectiles) != len(missionDrivers.Projectiles) {
				t.Fatalf("loaded drivers = %+v, want %+v", loadedDrivers, missionDrivers)
			}
			if got := loadedDrivers.Projectiles[0].Phases; got != missionDrivers.Projectiles[0].Phases {
				t.Fatalf("loaded SAV driver phases = %d, mission driver phases = %d", got, missionDrivers.Projectiles[0].Phases)
			}
			for tick := range 10 {
				mission.live.tick()
				loaded.live.tick()
				if !reflect.DeepEqual(mission.live.world.SavedProjectiles(), loaded.live.world.SavedProjectiles()) {
					t.Fatalf("tick %d: mission and loaded SAV projectiles diverged: mission=%+v loaded=%+v",
						tick, mission.live.world.SavedProjectiles(), loaded.live.world.SavedProjectiles())
				}
				md, ld := mission.live.savedProjectileDraws(), loaded.live.savedProjectileDraws()
				if len(md) != len(ld) {
					t.Fatalf("tick %d: draw count mission=%d loaded=%d", tick, len(md), len(ld))
				}
				for i := range md {
					if md[i].Pos != ld[i].Pos || md[i].Frame != ld[i].Frame ||
						md[i].Mirror != ld[i].Mirror || md[i].Effect != ld[i].Effect ||
						md[i].Phase != ld[i].Phase {
						t.Fatalf("tick %d: draw %d diverged: mission=%+v loaded=%+v", tick, i, md[i], ld[i])
					}
				}
			}
		})
	}
}

func projectileRegistryContainers(t *testing.T, regBytes []byte, includeReg bool) *vfs.FS {
	t.Helper()
	dir := t.TempDir()
	write := func(name string, b []byte) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, b, 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	scenario := write(ScenarioArchive, synth.Archive([]synth.File{
		{Path: "10.alm", Data: poolFixtureMap(91)},
		{Path: "npc.reg", Data: synth.NPCReg(nil)},
	}))
	var gfxFiles []synth.File
	if includeReg {
		gfxFiles = append(gfxFiles, synth.File{Path: "projectiles/projectiles.reg", Data: regBytes})
	}
	graphics := write(GraphicsArchive, synth.Archive(gfxFiles))
	containers, err := vfs.Open([]string{scenario, graphics}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return containers
}

func TestCorruptProjectileRegistryHandledLikeMissionLoader(t *testing.T) {
	valid := restoredFacingFront(t)
	ev, err := InspectProjectileArt(valid.Archives.Containers, restoredBoltPicture)
	if err != nil || ev.Path != "graphics/projectiles/bolt/sprites.256" || ev.Phases != restoredBoltPhases || ev.Frames != restoredBoltPhases*9 || ev.PayloadSHA256 == ([32]byte{}) {
		t.Fatalf("InspectProjectileArt = %+v, err=%v", ev, err)
	}
	raw := restoredFacingSource(t, valid)

	for _, tc := range []struct {
		name       string
		regBytes   []byte
		includeReg bool
		wantErr    bool
	}{
		{name: "malformed bytes", regBytes: []byte("not a reg"), includeReg: true, wantErr: true},
		{name: "missing count", regBytes: synth.Reg(0, nil), includeReg: true, wantErr: true},
		{name: "absent registry", includeReg: false, wantErr: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			containers := projectileRegistryContainers(t, tc.regBytes, tc.includeReg)

			_, missionErr := loadFlightInputs(containers, data.FirstSpellPresentation(), firstGameFiles.flight)
			if (missionErr != nil) != tc.wantErr {
				t.Fatalf("mission loader err = %v, wantErr = %v", missionErr, tc.wantErr)
			}
			_, artErr := LoadProjectiles(containers)
			if artErr == nil {
				t.Fatal("LoadProjectiles accepted missing/corrupt registry")
			}
			_, evidenceErr := InspectProjectileArt(containers, restoredBoltPicture)
			if evidenceErr == nil {
				t.Fatal("InspectProjectileArt accepted missing/corrupt registry")
			}

			f := restoredFacingFront(t)
			f.Archives.Containers = containers
			f.Projectiles = nil
			f.Table.FlightInputs = nil
			open, town, loadErr := f.RestoreOriginal(raw)
			if loadErr == nil {
				if town {
					t.Fatal("RestoreOriginal returned town for mission SAV")
				}
				app := f.App("corrupt registry sav load")
				app.Layout(1024, 768)
				t.Cleanup(app.StopAudio)
				loadErr = app.OpenMission(open)
			}
			if (loadErr != nil) != tc.wantErr {
				t.Fatalf("SAV import err = %v, wantErr = %v (mission loader err = %v)", loadErr, tc.wantErr, missionErr)
			}
			if !tc.wantErr {
				d := f.live.world.SavedWorldEffectDrivers()
				if d == nil || len(d.Projectiles) == 0 || d.Projectiles[0].Phases != 0 {
					t.Fatalf("absent registry SAV drivers = %+v, want armed with 0 phases", d)
				}
			}
		})
	}
}
