//go:build sessioncorpusaudit

package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCorpusDyingMoverSpeedRoundTrip(t *testing.T) {
	const path = "2026-08-15/game0017.sav"
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Fatal("AGAINROM_SAVE_CORPUS is required for dying corpus witness")
	}
	raw, err := os.ReadFile(filepath.Join(corpus, path))
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != "eafce5d6575d54fdddc7a35f57531cd3df9317006c80f7c4085866c1b02b4fe0" {
		t.Fatalf("dying corpus fixture SHA %s", got)
	}
	assets := os.Getenv("AGAINROM_ASSETS")
	if assets == "" {
		t.Fatal("AGAINROM_ASSETS is required for dying corpus witness")
	}
	c := savGateCase{name: path + " dying", raw: raw, ticks: 30, kill: true, dying: true}
	f, _, refusal, town := savGateOpen(t, assets, "dying mover speed", c, raw)
	if refusal != "" || town {
		t.Fatalf("open refusal=%q town=%v", refusal, town)
	}
	start, ok := f.live.world.Entity(0)
	if !ok || start.SourceNow().MoverSpeed != 18 {
		t.Fatalf("original actor's starting speed=%d, held=%v", start.SourceNow().MoverSpeed, ok)
	}
	savGatePlay(t, f, c)
	actor, ok := f.live.world.Entity(0)
	if !ok || actor.SourceBinding.Identity != 94856992 || actor.SourceBinding.ArchiveIndex != 36 || actor.SourceBinding.RuntimeID != 79 || actor.HP >= 0 || actor.ActorLoad.Source.Class != 2 || actor.SourceNow().MoverSpeed != 19 {
		t.Fatalf("corpus actor before SAVE: held=%v ID=%d identity=%d archive=%d runtime=%d HP=%d class=%d speed=%d", ok, actor.ID, actor.SourceBinding.Identity, actor.SourceBinding.ArchiveIndex, actor.SourceBinding.RuntimeID, actor.HP, actor.ActorLoad.Source.Class, actor.SourceNow().MoverSpeed)
	}
	motions, _, _, _ := f.live.world.SavedActorMotions()
	matched := false
	for _, m := range motions {
		if m.Entity != actor.ID {
			continue
		}
		matched = true
		if m.Current || m.Issue != "native death supersedes original movement" || m.Mover[10] != actor.SourceNow().MoverSpeed {
			t.Fatalf("corpus actor motion Current=%v Issue=%q speed=%d, live speed=%d", m.Current, m.Issue, m.Mover[10], actor.SourceNow().MoverSpeed)
		}
	}
	if !matched {
		t.Fatal("corpus actor has no imported motion")
	}
	g, _, _, refusal := savGateSaveLoad(t, assets, c, f)
	if refusal != "" {
		t.Fatal(refusal)
	}
	loaded, ok := g.live.world.Entity(actor.ID)
	if !ok || loaded.SourceBinding != actor.SourceBinding || loaded.ActorLoad.Source.MoverSpeed != 19 {
		t.Fatalf("dying actor source speed after cold LOAD = %d, held=%v", loaded.ActorLoad.Source.MoverSpeed, ok)
	}
}
