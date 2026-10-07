package game

import (
	"fmt"
	"hash/fnv"
	"image"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func TestReleaseRestoredBatProjectileTravelsAcrossSAVLoad(t *testing.T) {
	_, raw := groundCorpusFile(t, "2026-08-15/game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b")
	doc, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	store, _, err := doc.Projectiles()
	if err != nil || len(store.Items) != 1 {
		t.Fatal("restored flight source", err)
	}
	store.Items[0].Picture, store.Items[0].Phase = 7, 0
	if err := doc.SetProjectiles(store); err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	app, path := openOriginalSAVApp(t, f, doc.Marshal(), "bat.sav")
	digest := fnv.New64a()
	check := func(front *FrontEnd) ui.SpellBolt {
		t.Helper()
		before := front.live.world.Hash()
		draws := front.live.savedProjectileDraws()
		if before != front.live.world.Hash() {
			t.Fatal("presentation mutated world")
		}
		if len(draws) != 1 || draws[0].Effect != ui.SpellBackgroundDeformation || draws[0].Sheet != nil {
			t.Fatalf("restored bat draws %+v", draws)
		}
		p := front.live.world.SavedProjectiles().Items[0]
		if p.Picture != 7 || draws[0].Pos != image.Pt(int(p.X), int(p.Y)) || draws[0].Phase != int(p.Phase) {
			t.Fatal("current SAV position/phase lost", p, draws[0])
		}
		fmt.Fprintf(digest, "%d/%x/%v\n", front.live.world.Tick(), before, draws[0].Pos)
		return draws[0]
	}
	first := check(f)
	f.live.tick()
	before := check(f)
	if before.Pos == first.Pos {
		t.Fatal("restored projectile did not travel")
	}
	storeDir, name, _ := menuSAVE(t, f, app, OriginalStore{Dir: filepath.Dir(path)})
	fresh := loadLocalLegacySave(t, storeDir, name)
	if check(fresh) != before {
		t.Fatal("cold SAV LOAD changed bat presentation")
	}
	f.live.tick()
	fresh.live.tick()
	if check(f) != check(fresh) || f.live.world.Hash() != fresh.live.world.Hash() {
		t.Fatal("continued SAV flight differs")
	}
	for range 2 {
		f.live.tick()
		fresh.live.tick()
	}
	if len(f.live.savedProjectileDraws()) != 0 || len(fresh.live.savedProjectileDraws()) != 0 {
		t.Fatal("retired SAV bat draws")
	}
	t.Logf("derived picture-7 Prj266 current route, world digest %016x", digest.Sum64())
}
