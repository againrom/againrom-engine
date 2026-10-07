package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/game"
)

func TestLaunchProfileKeepsSavesAndOptionsTogether(t *testing.T) {
	for _, mode := range []string{"portable", "fallback", "explicit", "outside"} {
		t.Run(mode, func(t *testing.T) {
			install := t.TempDir()
			for _, name := range game.RequiredArchives() {
				if err := os.WriteFile(filepath.Join(install, name), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			executable := filepath.Join(install, "againrom.exe")
			config := t.TempDir()
			want := filepath.Join(install, "Againrom")
			o, err := parse(nil)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "fallback":
				if err := os.WriteFile(want, []byte("blocked profile"), 0600); err != nil {
					t.Fatal(err)
				}
				want = filepath.Join(config, "Againrom")
			case "explicit":
				want = t.TempDir()
				o, err = parse([]string{"-saves", filepath.Join(want, "custom-saves")})
				if err != nil {
					t.Fatal(err)
				}
			case "outside":
				want = t.TempDir()
				executable = filepath.Join(want, "againrom.exe")
				t.Chdir(want)
			case "portable":
				config = ""
			}
			profile, original, err := loadSources(install, o, executable, config)
			if err != nil {
				t.Fatal(err)
			}
			wantSaves := filepath.Join(want, "saves")
			if o.saves != "" {
				wantSaves = o.saves
			}
			if profile.Directory != want || profile.Saves.Dir != wantSaves || profile.Options.Path != filepath.Join(want, "options.txt") || original.Dir != install {
				t.Fatalf("profile=%+v original=%+v, want profile %q saves %q original %q", profile, original, want, wantSaves, install)
			}
			if mode == "outside" {
				if _, err := os.Stat(filepath.Join(want, "Againrom")); !os.IsNotExist(err) {
					t.Fatalf("outside launch created an install profile: %v", err)
				}
			}
		})
	}
}

func TestLaunchProfileSuppliesStartupPreferences(t *testing.T) {
	install := defaultInstall(t)
	profileDir := filepath.Join(install, "Againrom")
	if err := os.Mkdir(profileDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "options.txt"), []byte("SoundEnabled=0\nMasterVolume=25\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(install, "options.txt"), []byte("SoundEnabled=1\nMasterVolume=75\n"), 0600); err != nil {
		t.Fatal(err)
	}
	o, _ := parse(nil)
	profile, _, err := loadSources(install, o, filepath.Join(install, "againrom.exe"), "")
	if err != nil {
		t.Fatal(err)
	}
	front, err := frontEnd(install, o, profile.Options)
	if err != nil {
		t.Fatal(err)
	}
	if front.Options.Path != filepath.Join(profileDir, "options.txt") || front.Sound != (game.SoundOptions{Enabled: false, Volume: 25}) {
		t.Fatalf("startup options=%+v sound=%+v", front.Options, front.Sound)
	}
}

func TestReadOnlyLaunchDoesNotProbeInstallProfile(t *testing.T) {
	install, config := defaultInstall(t), t.TempDir()
	var out, errOut bytes.Buffer
	code := runWithProfilePaths([]string{"-assets", install, "-check", "-picker"}, noEnv, &out, &errOut, filepath.Join(install, "againrom.exe"), config)
	if code != 0 {
		t.Fatalf("check exit=%d stderr=%s", code, &errOut)
	}
	for _, dir := range []string{install, config} {
		if _, err := os.Stat(filepath.Join(dir, "Againrom")); !os.IsNotExist(err) {
			t.Fatalf("read-only launch created a profile under %s: %v", dir, err)
		}
	}
}

func TestHeadlessLaunchUsesExecutableProfile(t *testing.T) {
	install := defaultInstall(t)
	scenario := filepath.Join(t.TempDir(), "profile.json")
	if err := os.WriteFile(scenario, []byte(`{"version":1,"steps":[{"command":"assert_state","state":{"screen":"menu"}}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	code := runWithProfilePaths([]string{"-assets", install, "-headless", scenario}, noEnv, &out, &errOut, filepath.Join(install, "againrom.exe"), "")
	if code != 0 {
		t.Fatalf("headless exit=%d stderr=%s", code, &errOut)
	}
	if !strings.Contains(errOut.String(), filepath.Join(install, "Againrom", "saves")) {
		t.Fatalf("headless launch did not use its selected profile: %s", &errOut)
	}
}
