package modrt

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

func TestRowReferencesKeepFolderAndPackEquivalent(t *testing.T) {
	for _, tc := range []struct {
		name, data, row, read, held string
	}{
		{"town dot", mod.TownsFile, "[[town]]\nid = 9\ndescription = \"towns/t.json\"\n", "./art/x.bmp", "art/x.bmp"},
		{"town brackets", mod.TownsFile, "[[town]]\nid = 9\ndescription = \"towns/t.json\"\n", "art/y[1].bmp", "art/y[1].bmp"},
		{"sprite language token", mod.ItemsFile, "[[material]]\nitem = \"Pike\"\ntier = \"Common\"\nmaterial = \"Bone\"\nallow = true\nsprite = \"art/{lang}.png\"\n", "art/{lang}.png", "art/{lang}.png"},
		{"body language token", mod.BodiesFile, bodyReferenceRows("art/{lang}.png", "assets/bodies/light.png"), "art/{lang}.png", "art/{lang}.png"},
		{"light body language token", mod.BodiesFile, bodyReferenceRows("assets/bodies/heroes.png", "art/{lang}.png"), "art/{lang}.png", "art/{lang}.png"},
		{"portrait language token", mod.SpeakersFile, "[[speaker]]\nid = 20000\nname = \"Speaker\"\npicture = \"art/{lang}.png\"\n", "art/{lang}.png", "art/{lang}.png"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{tc.data: tc.row, tc.held: "synthetic image bytes"}
			if tc.data == mod.TownsFile {
				files["towns/t.json"] = fmt.Sprintf(`{"art":[{"name":"a","format":"picture","key":%q}]}`, mod.ModArtPrefix+tc.read)
			}
			if tc.data == mod.BodiesFile {
				files["assets/bodies/heroes.png"] = "heroes"
				files["assets/bodies/light.png"] = "light"
			}
			entries := makeMods(t, map[string]map[string]string{"references": files})
			dir := entries[0].Dir
			folder, err := mod.OpenTree(dir)
			if err != nil {
				t.Fatal(err)
			}
			want, err := folder.Describe()
			if err != nil {
				t.Fatal(err)
			}
			if !want.Admits(tc.held) || len(want.Warnings) != 0 {
				t.Errorf("reader's file is excluded or warned: %+v", want)
			}
			_, packedBytes, err := mod.Pack(dir)
			if err != nil {
				t.Fatal(err)
			}
			packedPath := filepath.Join(t.TempDir(), mod.PackedName("references"))
			if err := os.WriteFile(packedPath, packedBytes, 0o644); err != nil {
				t.Fatal(err)
			}
			packed, err := mod.OpenTree(packedPath)
			if err != nil {
				t.Fatal(err)
			}
			got, err := packed.Describe()
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Errorf("folder/pack descriptions differ: %+v / %+v: %v", want, got, err)
			}
			folderFiles, err := folder.Files()
			if err != nil {
				t.Fatal(err)
			}
			packedFiles, err := packed.Files()
			if err != nil || !reflect.DeepEqual(folderFiles, packedFiles) {
				t.Errorf("folder/pack trees differ: %v / %v: %v", folderFiles, packedFiles, err)
			}
			digest, err := folder.Digest()
			if err != nil {
				t.Fatal(err)
			}
			if got, err := packed.Digest(); err != nil || got != digest {
				t.Errorf("folder/pack digests differ: %s / %s: %v", digest, got, err)
			}
			for _, tree := range []mod.Tree{folder, packed} {
				if got, err := tree.ReadFile(tc.held); err != nil || string(got) != files[tc.held] {
					t.Errorf("%s does not hold the reader's file: %q, %v", tree.Path(), got, err)
				}
				if got, rel, err := tree.ReadInside(tc.read); err != nil || rel != tc.held || string(got) != files[tc.held] {
					t.Errorf("%s reader bytes: %q at %s: %v", tree.Path(), got, rel, err)
				}
				for _, base := range []string{"rom1-en", "rom1-ru"} {
					res, err := Load([]mod.Entry{mod.Load(tree.Path())}, base, nil, Options{})
					if err != nil || len(res.Warnings) != 0 {
						t.Errorf("%s %s check warnings: %v: %v", tree.Path(), base, res.Warnings, err)
					}
				}
			}
			if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(tc.held)), []byte("changed image bytes"), 0o644); err != nil {
				t.Fatal(err)
			}
			if got, err := folder.Digest(); err != nil || got == digest {
				t.Errorf("editing the reader's file did not move digest: %s: %v", got, err)
			}
		})
	}
}

func bodyReferenceRows(heroes, light string) string {
	return fmt.Sprintf("[[body]]\nname = \"custom\"\nheroes = %q\nheroes_l = %q\nframe = [64, 80]\norigin = [32, 70]\ndirections = 8\nmove = { frames = 1, ticks = 1 }\nattack = { frames = 1, ticks = 1 }\n", heroes, light)
}

func TestPackRefusesMalformedInstallFilesLikeTheFolder(t *testing.T) {
	for _, file := range []string{"files/common/graphics/x.png", "files/de/graphics/x.bmp", "files"} {
		t.Run(strings.ReplaceAll(file, "/", "_"), func(t *testing.T) {
			entries := makeMods(t, map[string]map[string]string{"broken": {file: "bad install file"}})
			_, folderErr := Load(entries, "rom1-en", nil, Options{})
			if folderErr == nil {
				t.Fatal("folder accepted broken install files")
			}
			_, _, packErr := mod.Pack(entries[0].Dir)
			if packErr == nil || packErr.Error() != folderErr.Error() {
				t.Fatalf("folder refusal %v, pack refusal %v", folderErr, packErr)
			}
		})
	}
}
