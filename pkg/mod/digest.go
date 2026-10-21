package mod

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// ContentDigest fingerprints the files of the mod at dir, its folder or its
// RES file: every regular file under it by slash-separated path and content,
// in path order, except README.md and names that start with a dot. Text
// files (.toml, .star, .txt, .md) are hashed with line ends read as LF so a
// checkout's line-end setting does not change the digest. A mod and its
// packed RES give one digest. The result is lowercase hex SHA-256.
func ContentDigest(dir string) (string, error) {
	t, err := OpenTree(dir)
	if err != nil {
		return "", err
	}
	return t.Digest()
}

// SetFormat is the version of the mod-set description.
const SetFormat = 1

// SetEntry is one mod of a mod set.
type SetEntry struct {
	ID       string
	Version  string
	Digest   string // ContentDigest of the mod folder
	Settings []SettingValue
	// StateFree reports that the mod changes no game state: its script sets
	// no rule and adds no data file, and it serves no text file, so it
	// carries pictures and sounds at most. It
	// is not part of the set's canonical form.
	StateFree bool
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

// Marked is the set a save's mod mark records: the mods that change game
// state, in load order. A set of state-free mods alone marks nothing.
func (s Set) Marked() Set {
	out := Set{Base: s.Base}
	for _, m := range s.Mods {
		if !m.StateFree {
			out.Mods = append(out.Mods, m)
		}
	}
	return out
}

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

// DifferenceKind names one way a mod set departs from another.
type DifferenceKind int

const (
	// DifferenceBase: another base game. Saved and Current are the bases.
	DifferenceBase DifferenceKind = iota + 1
	// DifferenceMissing: a mod the saved set holds is not active. Saved is
	// its version.
	DifferenceMissing
	// DifferenceAdded: an active mod the saved set does not hold. Current is
	// its version.
	DifferenceAdded
	// DifferenceVersion: the same mod in another version.
	DifferenceVersion
	// DifferenceFiles: the same version with other files. Saved and Current
	// are the version.
	DifferenceFiles
	// DifferenceSettingGone: a saved setting the active mod does not declare.
	// Saved is its saved value.
	DifferenceSettingGone
	// DifferenceSetting: a setting with another value.
	DifferenceSetting
	// DifferenceSettingNew: an active setting the saved set does not record.
	// Current is its value.
	DifferenceSettingNew
	// DifferenceOrder: the same mods in another load order. Saved and Current
	// are the mod ids joined by ", ".
	DifferenceOrder
)

// Difference is one way have departs from want. Mod is the mod's id and
// Setting the setting's key where the kind names one; Saved is the value want
// holds and Current the value have holds.
type Difference struct {
	Kind           DifferenceKind
	Mod, Setting   string
	Saved, Current string
}

// String is the difference as one line of prose naming the mod.
func (d Difference) String() string {
	switch d.Kind {
	case DifferenceBase:
		return fmt.Sprintf("the base game is %s, the saved game used %s", d.Current, d.Saved)
	case DifferenceMissing:
		return fmt.Sprintf("mod %q %s is missing", d.Mod, d.Saved)
	case DifferenceAdded:
		return fmt.Sprintf("mod %q is active but the saved game did not use it", d.Mod)
	case DifferenceVersion:
		return fmt.Sprintf("mod %q is version %s, the saved game used %s", d.Mod, d.Current, d.Saved)
	case DifferenceFiles:
		return fmt.Sprintf("mod %q has different files than the saved game's copy", d.Mod)
	case DifferenceSettingGone:
		return fmt.Sprintf("mod %q setting %s is not declared now; the saved game used %s", d.Mod, d.Setting, d.Saved)
	case DifferenceSetting:
		return fmt.Sprintf("mod %q setting %s is %s, the saved game used %s", d.Mod, d.Setting, d.Current, d.Saved)
	case DifferenceSettingNew:
		return fmt.Sprintf("mod %q setting %s is %s, the saved game did not record it", d.Mod, d.Setting, d.Current)
	case DifferenceOrder:
		return fmt.Sprintf("the mods load in the order %s, the saved game used %s", d.Current, d.Saved)
	}
	return fmt.Sprintf("mod %q differs", d.Mod)
}

// Differences lists how have departs from want, each line naming the mod:
// Compare's differences as prose.
func Differences(want, have Set) []string {
	var out []string
	for _, d := range Compare(want, have) {
		out = append(out, d.String())
	}
	return out
}

// Compare lists how have departs from want: a mod missing from have, a mod
// have adds, a different version, different files, a different setting value
// or a different load order. It is empty when the two sets are the same.
func Compare(want, have Set) []Difference {
	var out []Difference
	if want.Base != have.Base {
		out = append(out, Difference{Kind: DifferenceBase, Saved: want.Base, Current: have.Base})
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
			out = append(out, Difference{Kind: DifferenceMissing, Mod: m.ID, Saved: m.Version})
			continue
		}
		if h.Version != m.Version {
			out = append(out, Difference{Kind: DifferenceVersion, Mod: m.ID, Saved: m.Version, Current: h.Version})
		} else if h.Digest != m.Digest {
			out = append(out, Difference{Kind: DifferenceFiles, Mod: m.ID, Saved: m.Version, Current: h.Version})
		}
		hs := map[string]string{}
		for _, v := range h.Settings {
			hs[v.Key] = v.Value.String()
		}
		for _, v := range m.Settings {
			got, ok := hs[v.Key]
			switch {
			case !ok:
				out = append(out, Difference{Kind: DifferenceSettingGone, Mod: m.ID, Setting: v.Key, Saved: v.Value.String()})
			case got != v.Value.String():
				out = append(out, Difference{Kind: DifferenceSetting, Mod: m.ID, Setting: v.Key, Saved: v.Value.String(), Current: got})
			}
		}
		for _, v := range h.Settings {
			found := false
			for _, w := range m.Settings {
				found = found || w.Key == v.Key
			}
			if !found {
				out = append(out, Difference{Kind: DifferenceSettingNew, Mod: m.ID, Setting: v.Key, Current: v.Value.String()})
			}
		}
	}
	for _, m := range have.Mods {
		if _, ok := wantBy[m.ID]; !ok {
			out = append(out, Difference{Kind: DifferenceAdded, Mod: m.ID, Current: m.Version})
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
			out = append(out, Difference{Kind: DifferenceOrder, Saved: strings.Join(a, ", "), Current: strings.Join(b, ", ")})
		}
	}
	return out
}
