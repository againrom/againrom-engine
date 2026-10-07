// Command versionres regenerates the version resource object of every program
// listed in versionres.Programs from that program's VERSION file.
//
//	go run ./internal/versionres/cmd/versionres
//
// Run it from inside the module after editing a VERSION file, and commit the
// changed .syso files with it.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"againrom/internal/archtest"
	"againrom/internal/versionres"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "versionres:", err)
		os.Exit(1)
	}
	root, err := archtest.FindModuleRoot(wd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "versionres:", err)
		os.Exit(1)
	}
	for _, p := range versionres.Programs {
		dir := filepath.Join(root, filepath.FromSlash(p.Dir))
		version, err := versionres.ReadVersion(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "versionres:", p.Describe()+":", err)
			os.Exit(1)
		}
		data, err := versionres.Syso(version, p)
		if err != nil {
			fmt.Fprintln(os.Stderr, "versionres:", p.Describe()+":", err)
			os.Exit(1)
		}
		out := filepath.Join(dir, versionres.SysoName)
		if err := os.WriteFile(out, data, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "versionres:", err)
			os.Exit(1)
		}
		fmt.Printf("%s %s -> %s\n", p.Exe, version, filepath.ToSlash(filepath.Join(p.Dir, versionres.SysoName)))
	}
}
