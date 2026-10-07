package mod

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ContentDigest fingerprints the files of a mod folder: every regular file
// under it by slash-separated path and content, in path order, except README.md
// and names that start with a dot. Text files (.toml, .star, .txt, .md) are
// hashed with line ends read as LF so a checkout's line-end setting does not
// change the digest. The result is lowercase hex SHA-256.
func ContentDigest(dir string) (string, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if p != dir && strings.HasPrefix(name, ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || !d.Type().IsRegular() || strings.EqualFold(name, "README.md") && filepath.Dir(p) == dir {
			return nil
		}
		paths = append(paths, p)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(paths, func(i, j int) bool { return filepath.ToSlash(paths[i]) < filepath.ToSlash(paths[j]) })
	h := sha256.New()
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		switch strings.ToLower(filepath.Ext(p)) {
		case ".toml", ".star", ".txt", ".md":
			data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
		}
		rel, _ := filepath.Rel(dir, p)
		fmt.Fprintf(h, "%s\x00%d\x00", filepath.ToSlash(rel), len(data))
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// SetFormat is the version of the mod-set description.
const SetFormat = 1

// SetEntry is one mod of a mod set.
type SetEntry struct {
	ID       string
	Version  string
	Digest   string // ContentDigest of the mod folder
	Settings []SettingValue
}

// Set is the resolved mod set a game runs under: the base it applies to, and
// the mods in load order with the values of their settings. The zero Set is "no
// mods".
type Set struct {
	Base string
	Mods []SetEntry
}

// Empty reports whether no mod is active.
func (s Set) Empty() bool { return len(s.Mods) == 0 }

// Canonical is the one text form of the set that the digest is taken over.
func (s Set) Canonical() string {
	var b strings.Builder
	fmt.Fprintf(&b, "modset %d\nbase %s\n", SetFormat, strconv.Quote(s.Base))
	for _, m := range s.Mods {
		fmt.Fprintf(&b, "mod %s %s %s\n", m.ID, strconv.Quote(m.Version), m.Digest)
		for _, v := range m.Settings {
			fmt.Fprintf(&b, "set %s %s %s\n", v.Key, v.Value.Kind, strconv.Quote(v.Value.String()))
		}
	}
	return b.String()
}

// Digest is the SHA-256 of Canonical, as lowercase hex.
func (s Set) Digest() string {
	sum := sha256.Sum256([]byte(s.Canonical()))
	return hex.EncodeToString(sum[:])
}

// Differences lists how have departs from want, each line naming the mod: a mod
// missing from have, a mod have adds, a different version, different files, a
// different setting value or a different load order. It is empty when the two
// sets are the same.
func Differences(want, have Set) []string {
	var out []string
	if want.Base != have.Base {
		out = append(out, fmt.Sprintf("the base game is %s, the saved game used %s", have.Base, want.Base))
	}
	haveBy := map[string]SetEntry{}
	for _, m := range have.Mods {
		haveBy[m.ID] = m
	}
	wantBy := map[string]SetEntry{}
	for _, m := range want.Mods {
		wantBy[m.ID] = m
		h, ok := haveBy[m.ID]
		if !ok {
			out = append(out, fmt.Sprintf("mod %q %s is missing", m.ID, m.Version))
			continue
		}
		if h.Version != m.Version {
			out = append(out, fmt.Sprintf("mod %q is version %s, the saved game used %s", m.ID, h.Version, m.Version))
		} else if h.Digest != m.Digest {
			out = append(out, fmt.Sprintf("mod %q has different files than the saved game's copy", m.ID))
		}
		hs := map[string]string{}
		for _, v := range h.Settings {
			hs[v.Key] = v.Value.String()
		}
		for _, v := range m.Settings {
			got, ok := hs[v.Key]
			switch {
			case !ok:
				out = append(out, fmt.Sprintf("mod %q setting %s is not declared now; the saved game used %s", m.ID, v.Key, v.Value))
			case got != v.Value.String():
				out = append(out, fmt.Sprintf("mod %q setting %s is %s, the saved game used %s", m.ID, v.Key, got, v.Value))
			}
		}
		for _, v := range h.Settings {
			found := false
			for _, w := range m.Settings {
				found = found || w.Key == v.Key
			}
			if !found {
				out = append(out, fmt.Sprintf("mod %q setting %s is %s, the saved game did not record it", m.ID, v.Key, v.Value))
			}
		}
	}
	for _, m := range have.Mods {
		if _, ok := wantBy[m.ID]; !ok {
			out = append(out, fmt.Sprintf("mod %q is active but the saved game did not use it", m.ID))
		}
	}
	if len(out) == 0 {
		var a, b []string
		for _, m := range want.Mods {
			a = append(a, m.ID)
		}
		for _, m := range have.Mods {
			b = append(b, m.ID)
		}
		if strings.Join(a, ",") != strings.Join(b, ",") {
			out = append(out, fmt.Sprintf("the mods load in the order %s, the saved game used %s", strings.Join(b, ", "), strings.Join(a, ", ")))
		}
	}
	return out
}
