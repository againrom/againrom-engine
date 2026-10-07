//go:build sessioncorpusaudit

package game

import (
	"bytes"
	"crypto/sha256"
	"os"
	"strings"
	"testing"
)

// This is writer admission, separate from milestone 2's original reader
// population. Every discovered world is attempted unchanged; every refusal
// names its source and stage. Passing requires actual ordinary SAV publication
// and a source-free cold LOAD, not AGS fallback or only Document decoding.
func TestCurrentWorldSAVAdmission1170(t *testing.T) {
	worlds, loaded, written := 0, 0, 0
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, f *FrontEnd) {
		if !mf.present {
			return
		}
		worlds++
		refuse := func(stage string, err error) {
			t.Logf("WRITER-REFUSED %s sha256=%x stage=%s: %v", mf.rel, sha256.Sum256(mf.raw), stage, err)
		}
		open, town, err := f.RestoreOriginal(mf.raw)
		if err != nil {
			refuse("LOAD", err)
			return
		}
		if town {
			t.Fatal("world resumed as city")
		}
		if _, _, _, _, _, _, _, _, _, _, err = open(); err != nil {
			refuse("COMMIT", err)
			return
		}
		loaded++
		s, _, err := f.Snapshot(true)
		if err != nil {
			refuse("SNAPSHOT", err)
			return
		}
		if _, err = f.ExportCurrentWorldSave(s, "writer admission"); err != nil {
			refuse("CURRENT-WORLD", err)
			return
		}
		store := SaveStore{Dir: t.TempDir()}
		save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
		name, err := save(true)
		if err != nil {
			t.Fatal("admitted ordinary publication", err)
		}
		raw, err := store.Read(name)
		if err != nil || !strings.HasSuffix(name, ".sav") || !bytes.HasPrefix(raw, []byte("Asg&")) {
			t.Fatal("admitted SAVE did not publish SAV", name, err)
		}
		cold, err := NewFrontEnd(os.Getenv("AGAINROM_ASSETS"))
		if err != nil {
			t.Fatal("cold installed FrontEnd", err)
		}
		resume, town, err := cold.RestoreOriginal(raw)
		if err != nil || town {
			t.Fatal("admitted writer produced an unloadable SAV", err)
		}
		if _, _, _, _, _, _, _, _, _, _, err = resume(); err != nil {
			t.Fatal("admitted writer produced an uncommittable SAV", err)
		}
		written++
		t.Logf("WRITER-PASS %s sha256=%x output=%d bytes", mf.rel, sha256.Sum256(mf.raw), len(raw))
	})
	if worlds == 0 || written == 0 {
		t.Fatal("writer admission did not exercise any real current world")
	}
	t.Logf("WRITER-CENSUS discovered-worlds=%d imported=%d ordinary-SAV-and-cold-LOAD=%d refused=%d", worlds, loaded, written, worlds-written)
}

// Structural qualification is not provenance. Name any unchanged original
// crossing newly eligible for the native stride boundary policy, independently
// of the writer census or a current-world producer's output.
func TestCurrentWorldSAVStrideQualification1170(t *testing.T) {
	worlds, motions, active, matched := 0, 0, 0, 0
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, f *FrontEnd) {
		if !mf.present {
			return
		}
		worlds++
		open, town, err := f.RestoreOriginal(mf.raw)
		if err != nil || town {
			t.Fatalf("original stride census import: %v", err)
		}
		if _, _, _, _, _, _, _, _, _, _, err = open(); err != nil {
			t.Fatal(err)
		}
		rows, _, _, present := f.live.world.SavedActorMotions()
		if !present {
			t.Fatal("original stride census lost its motion carrier")
		}
		motions += len(rows)
		for _, m := range rows {
			if !m.Current || !m.Active || m.Issue != "" {
				continue
			}
			active++
			if m.NativeStrideCompatible() {
				matched++
				t.Logf("ORIGINAL-STRIDE-MATCH %s sha256=%x actor=%d position=%+v mover=%x dynamic=%v", mf.rel, sha256.Sum256(mf.raw), m.Entity, m.Position, m.Mover, m.DynamicRoute)
			}
		}
	})
	if worlds == 0 || motions == 0 {
		t.Fatal("original stride census is empty")
	}
	t.Logf("ORIGINAL-STRIDE-CENSUS worlds=%d retained-motions=%d active=%d structurally-qualified=%d", worlds, motions, active, matched)
}
