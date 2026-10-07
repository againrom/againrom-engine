package game

import (
	"image"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func restoredBatSource(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	doc, err := sav.Open(restoredFacingSource(t, f))
	if err != nil {
		t.Fatal(err)
	}
	store, _, err := doc.Projectiles()
	if err != nil {
		t.Fatal(err)
	}
	store.IDs, store.Items = store.IDs[:1], store.Items[:1]
	store.Items[0].Picture = 7
	if err := doc.SetProjectiles(store); err != nil {
		t.Fatal(err)
	}
	return doc.Marshal()
}

func TestRestoredBatProjectileTravelsAcrossSAVLoad(t *testing.T) {
	f := restoredFacingFront(t)
	app, path := openOriginalSAVApp(t, f, restoredBatSource(t, f), "bat.sav")
	check := func(front *FrontEnd) ui.SpellBolt {
		t.Helper()
		before := front.live.world.Hash()
		draws := front.live.savedProjectileDraws()
		if before != front.live.world.Hash() {
			t.Fatal("reading bat presentation changed world state")
		}
		if len(draws) != 1 || draws[0].Sheet != nil || draws[0].Effect != ui.SpellBackgroundDeformation {
			t.Fatalf("the restored bat has %d sprite-free travelling draws, want 1", len(draws))
		}
		p := front.live.world.SavedProjectiles().Items[0]
		if draws[0].Pos != image.Pt(int(p.X), int(p.Y)) {
			t.Fatal("the bat draw restarted from its departure", draws[0].Pos, p)
		}
		return draws[0]
	}
	first := check(f)
	f.live.tick()
	before := check(f)
	if first.Pos == before.Pos {
		t.Fatal("restored bat did not move")
	}
	_, name, written := menuSAVE(t, f, app, OriginalStore{Dir: filepath.Dir(path)})
	fresh := restoredFacingColdLoad(t, written, name)
	if after := check(fresh); after != before {
		t.Fatal("SAV LOAD changed the travelling draw", before, after)
	}
	for tick := 2; tick < restoredFlight; tick++ {
		f.live.tick()
		fresh.live.tick()
		if check(f) != check(fresh) || f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatal("restored bat continuation differs", tick)
		}
	}
	f.live.tick()
	fresh.live.tick()
	f.live.tick()
	fresh.live.tick()
	if len(f.live.savedProjectileDraws()) != 0 || len(fresh.live.savedProjectileDraws()) != 0 {
		t.Fatal("retired bat still draws")
	}
}

func TestFreshBatProjectileReadsItsMovingTarget(t *testing.T) {
	units, set := shotArchive(t)
	shooter := shotShooter(1, shotClassBat, 1, 4, 16)
	shooter.Reach = 8
	victim := shotVictim(2, 7, 4)
	victim.Speed = 63
	mw := shotWorld(t, units, set, nil, shooter, victim)
	mw.strike(1, 2)
	for tick := 0; len(mw.world.SavedProjectiles().Items) == 0 && tick < 32; tick++ {
		mw.tick()
	}
	if len(mw.world.SavedProjectiles().Items) != 1 {
		t.Fatal("bat did not release")
	}
	mw.enqueue(2, 7, 5)
	moved, retargeted := false, false
	for range 6 {
		mw.tick()
		victim, _ := mw.entity(2)
		moved = moved || victim.Y != 4
		items := mw.world.SavedProjectiles().Items
		if len(items) == 0 {
			break
		}
		retargeted = retargeted || items[0].ActionY != int32(4*ui.ShotScale+ui.ShotScale/2)
		before := mw.world.Hash()
		drawn := 0
		for _, draw := range mw.boltDraws(mw.world.EntityView()) {
			if draw.Effect == ui.SpellBackgroundDeformation {
				drawn++
			}
		}
		if drawn != 1 {
			t.Fatal("moving-target bat disappeared during flight", drawn)
		}
		if before != mw.world.Hash() {
			t.Fatal("reading bat changed simulation")
		}
	}
	if !moved {
		t.Fatal("ordinary move order did not move the target during flight")
	}
	if !retargeted {
		t.Fatal("the record kept the target's old point")
	}
}

func TestBatProjectileNeedsRegistryEntryButNoSheet(t *testing.T) {
	units, set := shotArchive(t)
	delete(set.Sheets, 7)
	mw := shotWorld(t, units, set, nil, shotShooter(1, shotClassBat, 1, 4, 8), shotVictim(2, 5, 4))
	if _, _, drawn := mw.classShot(shotClassBat); !drawn {
		t.Fatal("registered callback depends on sprite art")
	}
	if b, ok := mw.spellDraw(7, image.Pt(1, 4), image.Pt(5, 4), image.Pt(2*ui.ShotScale, 4*ui.ShotScale), 100, 1); !ok || b.Sheet != nil || b.Effect != ui.SpellBackgroundDeformation {
		t.Fatal("registered callback missing", b, ok)
	}
	delete(set.Pictures, 7)
	if _, _, drawn := mw.classShot(shotClassBat); drawn {
		t.Fatal("callback admitted missing registry entry")
	}
	if _, ok := mw.spellDraw(7, image.Pt(1, 4), image.Pt(5, 4), image.Point{}, 0, 1); ok {
		t.Fatal("unregistered callback drawn")
	}
}
