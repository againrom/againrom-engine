//go:build savdocumentaudit

package sav

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFieldSetters1142CorpusBoundedness(t *testing.T) {
	root := os.Getenv("AGAINROM_DOCUMENT_CORPUS")
	if root == "" {
		t.Fatal("set AGAINROM_DOCUMENT_CORPUS to the explicit read-only SAV directory")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".sav") {
			continue
		}
		count++
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(root, name)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(raw)

			base, err := Open(raw)
			if err != nil {
				t.Fatalf("input %x: %v", digest, err)
			}
			baseline := base.Marshal()

			// measure applies one setter to a fresh copy, bounds the whole-
			// file byte-diff it produces to [min,max], and hands the
			// reopened edited file to verify so the caller can read back
			// the exact value that was set — not just that SOMETHING
			// differed by a plausible amount.
			measure := func(label string, min, max int, apply func(f *File) error, verify func(back *File) error) {
				f, err := Open(raw)
				if err != nil {
					t.Fatalf("%s: reopen: %v", label, err)
				}
				if err := apply(f); err != nil {
					t.Logf("%s: not applicable: %v", label, err)
					return
				}
				out := f.Marshal()
				diff, longer, shorter := diffCount(baseline, out)
				t.Logf("%s: %d bytes differ (of %d/%d), len delta %+d", label, diff, len(baseline), len(out), len(out)-len(baseline))
				_ = longer
				_ = shorter
				if diff < min || diff > max {
					t.Errorf("%s: %d bytes differ, want [%d,%d]", label, diff, min, max)
				}
				back, err := Open(out)
				if err != nil {
					t.Fatalf("%s: edited output does not reopen: %v", label, err)
				}
				if err := verify(back); err != nil {
					t.Errorf("%s: %v", label, err)
				}
			}

			// Fixed-width dword edits: the touched region is a single u32
			// (reg.SetInt for the store leaf; a positional, non-reordering
			// dword patch in the campaign tail for the other three), so no
			// legitimate edit can differ by more than its own 4 bytes.
			// SetPlayerListField writes into the compressed Body rather
			// than an uncompressed region, so its window allows headroom
			// for the RLE stream's own encoding to shift around that one
			// changed word; all 136 preserved saves measure exactly 1.
			measure("SetStoreInt GameOptions/Speed=7", 0, 4, func(f *File) error {
				return f.SetStoreInt("GameOptions", "Speed", 7)
			}, func(back *File) error {
				store, ok := back.StateStore()
				if !ok {
					return fmt.Errorf("reopened file lost its state store")
				}
				if v, ok := store.GetInt("GameOptions", "Speed"); !ok || v != 7 {
					return fmt.Errorf("reopened GameOptions/Speed = %d, %v, want 7", v, ok)
				}
				return nil
			})
			measure("SetStoreIntArray Objects/Selection=nil", 0, 4096, func(f *File) error {
				return f.SetStoreIntArray("Objects", "Selection", nil)
			}, func(back *File) error {
				store, ok := back.StateStore()
				if !ok {
					return fmt.Errorf("reopened file lost its state store")
				}
				if a, ok := store.GetIntArray("Objects", "Selection"); !ok || len(a) != 0 {
					return fmt.Errorf("reopened Objects/Selection = %v, %v, want empty", a, ok)
				}
				return nil
			})
			measure("SetMapName +suffix", 1, 2*len(raw)+64, func(f *File) error {
				return f.SetMapName(f.Head.MapName + "-x")
			}, func(back *File) error {
				if want := base.Head.MapName + "-x"; back.Head.MapName != want {
					return fmt.Errorf("reopened MapName = %q, want %q", back.Head.MapName, want)
				}
				return nil
			})
			measure("SetPlayerListField=9", 0, 64, func(f *File) error {
				return f.SetPlayerListField(9)
			}, func(back *File) error {
				if back.Head.PlayerListField != 9 {
					return fmt.Errorf("reopened PlayerListField = %d, want 9", back.Head.PlayerListField)
				}
				return nil
			})
			measure("SetCampaignScalar[2]=0", 0, 4, func(f *File) error {
				return f.SetCampaignScalar(2, 0)
			}, func(back *File) error {
				got, ok, err := back.Campaign()
				if err != nil || !ok {
					return fmt.Errorf("reopened campaign: ok=%v err=%v", ok, err)
				}
				if got.AutoGetMission != 0 {
					return fmt.Errorf("reopened AutoGetMission = %d, want 0", got.AutoGetMission)
				}
				return nil
			})
			measure("SetCampaignBaseDWord[1]=0", 0, 4, func(f *File) error {
				return f.SetCampaignBaseDWord(1, 0)
			}, func(back *File) error {
				got, ok, err := back.Campaign()
				if err != nil || !ok {
					return fmt.Errorf("reopened campaign: ok=%v err=%v", ok, err)
				}
				if got.Main.MapObject != 0 {
					return fmt.Errorf("reopened Main.MapObject = %d, want 0", got.Main.MapObject)
				}
				return nil
			})
			measure("SetCampaignArray[0]=nil", 0, 2048, func(f *File) error {
				return f.SetCampaignArray(0, nil)
			}, func(back *File) error {
				got, ok, err := back.Campaign()
				if err != nil || !ok {
					return fmt.Errorf("reopened campaign: ok=%v err=%v", ok, err)
				}
				if len(got.Mercenaries) != 0 {
					return fmt.Errorf("reopened Mercenaries = %v, want empty", got.Mercenaries)
				}
				return nil
			})
			measure("SetCampaignDocuments=nil", 0, 2048, func(f *File) error {
				return f.SetCampaignDocuments(nil)
			}, func(back *File) error {
				got, ok, err := back.Campaign()
				if err != nil || !ok {
					return fmt.Errorf("reopened campaign: ok=%v err=%v", ok, err)
				}
				if len(got.Documents) != 0 {
					return fmt.Errorf("reopened Documents = %v, want empty", got.Documents)
				}
				return nil
			})
			measure("SetCampaignChildren=nil", 0, 4096, func(f *File) error {
				return f.SetCampaignChildren(nil)
			}, func(back *File) error {
				got, ok, err := back.Campaign()
				if err != nil || !ok {
					return fmt.Errorf("reopened campaign: ok=%v err=%v", ok, err)
				}
				if len(got.Children) != 0 {
					return fmt.Errorf("reopened Children = %v, want empty", got.Children)
				}
				return nil
			})

			still, err := os.ReadFile(path)
			if err != nil || sha256.Sum256(still) != digest {
				t.Fatalf("read-only corpus changed: %v", err)
			}
		})
	}
	if count == 0 {
		t.Fatal("selected directory contains no SAV subjects")
	}
	t.Logf("read-only setter-boundedness subjects: %d", count)
}

func diffCount(a, b []byte) (diff, longerBy, shorterBy int) {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			diff++
		}
	}
	diff += len(a) - n
	diff += len(b) - n
	if len(b) > len(a) {
		longerBy = len(b) - len(a)
	}
	if len(a) > len(b) {
		shorterBy = len(a) - len(b)
	}
	return diff, longerBy, shorterBy
}
