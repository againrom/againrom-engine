package game

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/base"
)

func TestInspectInstallReportsMissingArchives(t *testing.T) {
	dir := t.TempDir()
	if got := InspectInstall(filepath.Join(dir, "nosuch")); !reflect.DeepEqual(got.Missing, RequiredArchives()) || got.Valid() {
		t.Fatalf("missing directory: %+v", got)
	}
	for _, name := range []string{"MAIN.RES", "Graphics.res", "scenario.res"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := InspectInstall(dir)
	if got.Valid() || !reflect.DeepEqual(got.Missing, []string{WorldArchive, MoviesArchive}) || got.Language != "" {
		t.Fatalf("partial install: %+v", got)
	}
}

func TestInspectInstallIsValidWithFiveArchivesEvenIfMainIsUnreadable(t *testing.T) {
	dir := t.TempDir()
	for _, name := range RequiredArchives() {
		if err := os.WriteFile(filepath.Join(dir, strings.ToUpper(name)), []byte("not an archive"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := InspectInstall(dir)
	if !got.Valid() || got.Language != "" {
		t.Fatalf("%+v", got)
	}
}

// TestReleaseInspectInstallNamesTheLanguage reads the lawful install named by
// AGAINROM_ASSETS: it is valid and names a language.
func TestReleaseInspectInstallNamesTheLanguage(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: language detection needs a lawful install")
	}
	got := InspectInstall(root)
	if !got.Valid() || (got.Language != "english" && got.Language != "russian") {
		t.Fatalf("%+v", got)
	}
	t.Logf("install %s: %s", root, got.Language)
}

// A root of the five archives that matches no known build is the generic
// profile, valid, and a mod names it "rom1".
func TestInspectInstallNamesTheGenericProfile(t *testing.T) {
	dir := t.TempDir()
	for _, name := range RequiredArchives() {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("not an archive"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := InspectInstall(dir)
	if !got.Valid() || got.Base.ID() != base.ROM1 || got.Base.Exact || BaseID(got) != base.ROM1 {
		t.Fatalf("%+v", got)
	}
	if front := (*FrontEnd)(nil); front.Base().Profile.Known() || len(front.BaseLines()) != 0 {
		t.Fatal("a nil front end names a base")
	}
}

func TestBaseIDPrefersTheDetectedProfile(t *testing.T) {
	cases := []struct {
		info InstallInfo
		want string
	}{
		{InstallInfo{Language: "english"}, "rom1-en"},
		{InstallInfo{Language: "russian"}, "rom1-ru"},
		{InstallInfo{Language: "language selector 4"}, "rom1"},
		{InstallInfo{Language: "english", Base: base.Match{Profile: base.Profile{ID: base.ROM1Demo}}}, base.ROM1Demo},
	}
	for _, c := range cases {
		if got := BaseID(c.info); got != c.want {
			t.Errorf("BaseID(%+v) = %q, want %q", c.info, got, c.want)
		}
	}
}

// A known build that lacks a file of its profile is not valid and names the
// file; with the file present it is the profile, exactly.
func TestInspectInstallRefusesAPartialProfile(t *testing.T) {
	dir := t.TempDir()
	const main = "known main"
	for _, name := range RequiredArchives() {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(main), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	sum := sha256.Sum256([]byte(main))
	old := base.Profiles
	t.Cleanup(func() { base.Profiles = old })
	base.Profiles = append([]base.Profile{{
		ID: "partial-test", Title: "Partial", Files: []string{"extra.res"},
		Builds: []base.Build{{Size: int64(len(main)), SHA256: hex.EncodeToString(sum[:])}},
	}}, old...)

	got := InspectInstall(dir)
	var partial *base.PartialError
	if got.Valid() || !errors.As(got.BaseErr, &partial) || !reflect.DeepEqual(got.Missing, []string{"extra.res"}) {
		t.Fatalf("partial profile: %+v", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "EXTRA.RES"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got = InspectInstall(dir)
	if !got.Valid() || got.Base.ID() != "partial-test" || !got.Base.Exact || BaseID(got) != "partial-test" {
		t.Fatalf("complete profile: %+v", got)
	}
}

func TestInstallInfoErrorsAreTyped(t *testing.T) {
	got := InspectInstall(filepath.Join(t.TempDir(), "nosuch"))
	var miss *base.NotInstallError
	if !errors.As(got.BaseErr, &miss) || got.Valid() {
		t.Fatalf("%+v", got)
	}
}
