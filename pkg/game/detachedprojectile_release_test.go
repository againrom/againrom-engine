package game

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReleaseDetachedProjectileTargetSAVRoundTrip(t *testing.T) {
	_, raw := groundCorpusFile(t, "2026-08-15/game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b")
	f, app, _, p := removedCorpseProjectileApp(t, raw)
	_, _, written := menuSAVE(t, f, app, OriginalStore{})
	if got := projectileSAVFields(t, written); got[9] != 157 || got[10] != p.ActionX || got[11] != p.ActionY || got[14] != p.ActionSegments {
		t.Fatal("written detached projectile", got, p)
	}
	path := filepath.Join(t.TempDir(), "detached.sav")
	if err := os.WriteFile(path, written, 0o600); err != nil {
		t.Fatal(err)
	}
	back := loadAreaContinuation(t, path)
	drivers := back.live.world.SavedWorldEffectDrivers()
	if drivers == nil || len(drivers.Projectiles) != 1 {
		t.Fatal("LOAD dropped the detached projectile driver", drivers)
	}
	d := drivers.Projectiles[0]
	if _, live := entityIn(back.live.world.Entities(), d.Target); !d.HasTarget || !d.TargetDetached || live {
		t.Fatal("LOAD did not bind a detached target", d, live)
	}
	if items := back.live.world.SavedProjectiles().Items; len(items) != 1 || items[0] != p {
		t.Fatal("LOAD changed the detached projectile", items, p)
	}
	back.live.tick()
	if items := back.live.world.SavedProjectiles().Items; len(items) != 1 || items[0].ActionX != p.ActionX || items[0].ActionSegments != p.ActionSegments-1 {
		t.Fatal("LOADed detached projectile does not fly toward its saved last point", items)
	}
}
