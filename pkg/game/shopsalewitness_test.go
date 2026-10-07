package game

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStaffSaleWitnessSeparatesInstalledRoots(t *testing.T) {
	base := t.TempDir()
	output := filepath.Join(base, "output")
	roots := []string{filepath.Join(base, "assets", "en"), filepath.Join(base, "assets", "ru")}
	for i, root := range roots {
		dir := staffSaleWitnessOutput(output, root)
		if again := staffSaleWitnessOutput(output, filepath.Clean(root)); again != dir {
			t.Fatal("same installed root changed witness output", dir, again)
		}
		if err := os.MkdirAll(filepath.Join(dir, "profile", "saves"), 0700); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"identity.json", filepath.Join("profile", "saves", "control.sav")} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte{byte(i)}, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	for i, root := range roots {
		dir := staffSaleWitnessOutput(output, root)
		for _, name := range []string{"identity.json", filepath.Join("profile", "saves", "control.sav")} {
			got, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil || len(got) != 1 || got[0] != byte(i) {
				t.Fatalf("installed root %s output %s was clobbered: %v/%v", root, name, got, err)
			}
		}
	}
}
