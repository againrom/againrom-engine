package data

import (
	"fmt"
	"strconv"
	"strings"

	"againrom/pkg/formats/reg"
)

// Sprite paths: the archive entry a class's art sits at, and the overlay entry
// beside it.
//
// The stored path is backslash-separated and extensionless. The sprite entry is
// the registry's own directory, then that path with every backslash turned into
// a forward slash, then ".256"; the overlay entry is the same string with a "b"
// before the extension — one insertion, never a second lookup.
//
// CASE IS PRESERVED and nothing else about the path is touched: no lowercasing
// (the archive folds case on lookup, this layer does not), no cleaning, no
// collapsing of a repeated separator, no "." or ".." element handling. That is
// why path/filepath is absent from this file: these are ARCHIVE paths, and
// filepath's separator is the host's — filepath.ToSlash is a no-op on a Unix
// host, and either Clean rewrites the string on every host.
//
// A units or objects class holds File as an INDEX into its registry's [Files]
// table; a structure holds the path itself, that registry having no [Files].
// File's bound and the [Files] entry's presence and non-emptiness are checked
// here, at load, and nowhere else: this is the only place the resolved index,
// the count and the table are all in hand.
//
// Construction NEVER OPENS AN ARCHIVE AND NEVER CHECKS EXISTENCE. Seven of the
// shipped classes name a sprite whose art does not ship at all, and an absent
// sprite file is valid shipped data, not a failure: this layer resolves classes,
// not art. Both methods are pure string functions of a value fixed at load.

// spriteEnv is what one registry offers a class computing its base: the
// descriptor, and — where the registry has a [Files] table — that table's node
// and the bound a File index must fall inside.
type spriteEnv struct {
	desc      descriptor
	files     *reg.Node // the [Files] section, nil where the registry has none
	fileCount int32     // [Global] FileCount: the exclusive upper bound on File
}

// newSpriteEnv reads what a registry says about its [Files] table, once per
// load. A registry that has none — structures.reg, where each class carries its
// path directly — needs nothing read.
//
// FileCount is a [Global] count exactly as the class count is, and File's legal
// domain is stated as [0, FileCount): a registry with a [Files] table and no
// readable FileCount cannot have that bound evaluated at all, so a missing or
// non-int one is malformed on the same grounds as a missing class count, and is
// reported in the same shape. A missing [Files] SECTION is not rejected here: it
// is a fault only where some class actually references an entry, which is what
// base checks.
func newSpriteEnv(r *reg.Reg, d descriptor) (spriteEnv, error) {
	env := spriteEnv{desc: d}
	if !d.hasFiles {
		return env, nil
	}
	n, state := readKey(findSection(r, "Global"), "FileCount", kindInt)
	switch state {
	case keyPresent:
		env.fileCount = n.Int
	case keyWrongKind:
		return spriteEnv{}, fmt.Errorf("Global: FileCount: wrong kind, want int")
	default:
		return spriteEnv{}, fmt.Errorf("Global: FileCount: missing")
	}
	env.files = findSection(r, "Files")
	return env, nil
}

// base computes one class's sprite base — the entry path without its extension —
// from that class's RESOLVED File node, nil where File is absent after
// inheritance. The section name is carried in only to name the offender in an
// error.
//
// An absent File yields an EMPTY BASE AND NO ERROR: Validation does not name a
// missing File, so rejecting one here would refuse data the contract takes, and
// both path methods answer "" for such a class. A structure whose own File is
// the empty string is the same case rather than the [Files] emptiness fault —
// that registry has no [Files] table for the fault to be about, and the
// alternative is handing back a path the class never had.
func (e spriteEnv) base(section string, file *reg.Node) (string, error) {
	if file == nil {
		return "", nil
	}

	// structures.reg: File is the path itself.
	if !e.desc.hasFiles {
		if file.Str == "" {
			return "", nil
		}
		return e.desc.spritePrefix + slashPath(file.Str), nil
	}

	idx := file.Int
	if idx < 0 || idx >= e.fileCount {
		return "", fmt.Errorf("%s: File: index %d is outside [0, %d)", section, idx, e.fileCount)
	}
	name := "File" + strconv.Itoa(int(idx))
	entry, state := readKey(e.files, name, kindStr)
	if state != keyPresent {
		return "", fmt.Errorf("%s: File: [Files] has no entry %s", section, name)
	}
	if entry.Str == "" {
		return "", fmt.Errorf("%s: File: [Files] entry %s is empty", section, name)
	}
	return e.desc.spritePrefix + slashPath(entry.Str), nil
}

// slashPath turns the stored separator into the archive's. Every backslash
// becomes a forward slash and NOTHING ELSE CHANGES — not case, not a repeated
// separator, not a "." element. A plain string replacement, never filepath.
func slashPath(p string) string { return strings.ReplaceAll(p, `\`, "/") }

// spritePath and overlayPath are the two extensions over a base. An empty base —
// a class resolving no File — answers "" from both rather than a bare extension.
//
// What is ours and unbacked is only the path FORMATTER — the separator, the
// prefix, the ".256" and where the "b" sits — which 0016's provenance already
// carries at Medium. Naming the method for the node's identity was a second,
// undisclosed claim on top of that one.
func spritePath(base string) string {
	if base == "" {
		return ""
	}
	return base + ".256"
}

func overlayPath(base string) string {
	if base == "" {
		return ""
	}
	return base + "b.256"
}

// palettePath is the entry one TIER's colour table sits at: the sheet's own
// DIRECTORY, then "palette", then the tier in decimal ONLY above tier 1, then
// ".pal".
//
// The digit's absence at tier 1 is not a formatting nicety — the engine builds
// the name with a "%d" and then strikes the digit off for the first entry with
// an explicit store, so tier 1 is `palette.pal` and tier 2 is `palette2.pal`.
// A formatter that wrote `palette1.pal` would name a node that does not exist
// on either shipped root.
//
// It is the DIRECTORY of the base and not the base itself. The sibling
// formatters above append to the whole base — `swordsman` becomes
// `swordsman.256` — and this one drops the last element, so a class whose sheet
// is `units/monsters/goblin/sprites.256` takes its tables from
// `units/monsters/goblin/palette*.pal`. A base carrying no separator at all
// yields the bare name, which addresses a container's root: the same "no
// directory" answer a path with no directory should give, rather than a special
// case.
//
// An empty base — a class resolving no File — answers "" as both siblings do,
// and so does any tier below 1: there is no tier 0 and no negative tier, and a
// caller asking for one is asking for no file rather than for the first.
//
// It is a pure string function of a value fixed at load. It opens nothing,
// checks no existence and applies no case folding, for the reasons the file
// header states.
func palettePath(base string, tier int) string {
	if base == "" || tier < 1 {
		return ""
	}
	dir := base[:strings.LastIndex(base, "/")+1]
	n := ""
	if tier > 1 {
		n = strconv.Itoa(tier)
	}
	return dir + "palette" + n + ".pal"
}

// PalettePath returns the archive entry this class's tier-N colour table sits
// at, or "" when the class resolves no File or N is below 1. It opens nothing
// and checks nothing.
//
// How many tiers the class actually has is TierCount's answer, not this one's:
// this is the name a tier WOULD have, so a caller that asks past the count gets
// a well-formed address for a node that need not exist, exactly as a class
// naming art that does not ship gets a well-formed sprite address.
func (c *UnitClass) PalettePath(tier int) string { return palettePath(c.base, tier) }

// SpritePath returns the archive entry this class's sprite sits at, or "" when
// the class resolves no File. It opens nothing and checks nothing: an entry
// naming art that does not ship is valid shipped data.
func (c *UnitClass) SpritePath() string { return spritePath(c.base) }

// OverlayPath returns the entry beside SpritePath's — the same path with a "b"
// before the extension — or "" when the class resolves no File. What that entry
// holds is an overlay layer, not a shadow; see spritePath/overlayPath above.
func (c *UnitClass) OverlayPath() string { return overlayPath(c.base) }

// SpritePath returns the archive entry this class's sprite sits at, or "" when
// the class resolves no File. It opens nothing and checks nothing.
func (c *ObjectClass) SpritePath() string { return spritePath(c.base) }

// OverlayPath returns the entry beside SpritePath's — the same path with a "b"
// before the extension — or "" when the class resolves no File.
func (c *ObjectClass) OverlayPath() string { return overlayPath(c.base) }

// SpritePath returns the archive entry this class's sprite sits at, or "" when
// the class carries no path. It opens nothing and checks nothing.
func (c *StructureClass) SpritePath() string { return spritePath(c.base) }

// OverlayPath returns the entry beside SpritePath's — the same path with a "b"
// before the extension — or "" when the class carries no path.
func (c *StructureClass) OverlayPath() string { return overlayPath(c.base) }
