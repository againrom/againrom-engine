// Command restool inspects .res archives (developer tool).
//
// Usage:
//
//	restool list <archive>            list entries (size and path), in node order
//	restool cat <archive> <path>      write one entry's bytes to stdout
//	restool extract <archive> <dir>   write every entry under <dir>
//
// restool is a developer-run tool for verifying the reader against a lawful game
// install; it is never part of the test suite. Point extract at a git-ignored
// directory: extracted bytes are game assets and must never be committed.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"againrom/pkg/formats/res"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "restool:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: restool <list|cat|extract> <archive> [args]")
	}
	cmd, archivePath := args[0], args[1]
	a, err := res.Open(archivePath)
	if err != nil {
		return err
	}
	switch cmd {
	case "list":
		return doList(a)
	case "cat":
		if len(args) != 3 {
			return fmt.Errorf("usage: restool cat <archive> <path>")
		}
		return doCat(a, args[2])
	case "extract":
		if len(args) != 3 {
			return fmt.Errorf("usage: restool extract <archive> <dir>")
		}
		return doExtract(a, args[2])
	default:
		return fmt.Errorf("unknown command %q (want list, cat, or extract)", cmd)
	}
}

func doList(a *res.Archive) error {
	entries := a.Entries()
	for _, e := range entries {
		fmt.Printf("%12d  %s\n", e.Size, e.Path)
	}
	fmt.Fprintf(os.Stderr, "restool: %d entries\n", len(entries))
	return nil
}

func doCat(a *res.Archive, path string) error {
	b, err := a.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(b)
	return err
}

func doExtract(a *res.Archive, dir string) error {
	root, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	entries := a.Entries()
	for _, e := range entries {
		dest := filepath.Join(root, filepath.FromSlash(e.Path))
		// Path-escape guard: a crafted name must not write outside <dir>.
		rel, relErr := filepath.Rel(root, dest)
		if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("entry %q escapes the output directory", e.Path)
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		b, err := a.ReadFile(e.Path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dest, b, 0o644); err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "restool: extracted %d entries to %s\n", len(entries), root)
	return nil
}
