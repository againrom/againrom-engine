package mod

import (
	"slices"
	"strings"

	"againrom/pkg/locale"
	"againrom/pkg/namefold"
)

// DraftContext is when the map editor's draft of a kind's file builds the
// mod's run context again.
type DraftContext uint8

const (
	// DraftNone: the run reads no draft of the kind.
	DraftNone DraftContext = iota
	// DraftAlways: each draft of the kind builds the context again.
	DraftAlways
	// DraftCited: a draft builds it again while the project holds the file
	// of a kind whose records cite text and whose drafts the run reads.
	DraftCited
)

// Drafting is how the map editor's project drafts the files of one kind.
type Drafting struct {
	Context DraftContext
	// Held: the project holds every file of the kind whole from the open,
	// and a save takes out of the folder one the last save wrote and the
	// project no longer holds.
	Held bool
}

// installDrafts is the drafting of the install files: the run reads their
// drafts through OverlayInstallFiles.
var installDrafts = Drafting{Context: DraftAlways, Held: true}

// textLookup answers a text key, as DataReader.Text does.
type textLookup = func(key string) (string, bool)

// DraftKinds is every kind the map editor drafts, in table order.
func DraftKinds() []DataKind {
	var out []DataKind
	for _, k := range dataKinds {
		if k.Draft != (Drafting{}) {
			out = append(out, k)
		}
	}
	return out
}

// DraftNames is the fixed names the kind's files are drafted under: its
// Files and, of a path that names one file per language, that file of
// each language.
func (k DataKind) DraftNames() []string {
	out := slices.Clone(k.Files)
	for _, p := range k.Paths {
		if draftFolder(p) == "" && strings.Contains(p, "{lang}") {
			for _, l := range locale.All() {
				out = append(out, strings.ReplaceAll(p, "{lang}", l.Code))
			}
		}
	}
	return out
}

// draftFolder is the folder, with its slash, that holds the files a path
// names by a pattern: the segments before the first placeholder or
// wildcard. A fixed path, or one whose only placeholder is the language,
// has none.
func draftFolder(p string) string {
	segments := strings.Split(p, "/")
	for i, s := range segments {
		if s != "{lang}" && strings.ContainsAny(s, "*<{") {
			return strings.Join(segments[:i], "/") + "/"
		}
	}
	return ""
}

// DraftFile reports whether name, a file of the editor's project as the
// project spells it, is a file of the kind: one of DraftNames, or one under
// a folder its Paths name.
func (k DataKind) DraftFile(name string) bool {
	if slices.Contains(k.DraftNames(), name) {
		return true
	}
	_, under := k.draftFolderName(name)
	return under
}

// DraftFolderName is the name of a file under one of the kind's folders as the
// kind spells that folder, the rest as given; name itself when none holds it.
// The folder matches as the mod tree matches names, in any letter case.
func (k DataKind) DraftFolderName(name string) string {
	spelled, _ := k.draftFolderName(name)
	return spelled
}

func (k DataKind) draftFolderName(name string) (string, bool) {
	for _, p := range k.Paths {
		f := draftFolder(p)
		if f != "" && len(name) >= len(f) && namefold.ModTree(name[:len(f)]) == namefold.ModTree(f) {
			return f + name[len(f):], true
		}
	}
	return name, false
}

// DraftFolderFile reports whether rel, a file of the mod's listing, is one
// the kind keeps under a folder: a file under a folder its Paths name that
// one of its paths takes.
func (k DataKind) DraftFolderFile(rel string) bool {
	return !slices.Contains(k.DraftNames(), rel) && k.DraftFile(rel) && k.takes(rel)
}

// HeldDraft reports whether name is a file of a kind the project holds
// whole (Drafting.Held).
func HeldDraft(name string) bool {
	return slices.ContainsFunc(dataKinds, func(k DataKind) bool { return k.Draft.Held && k.DraftFile(name) })
}

// RebuildsContext reports whether the editor's draft of the file name, in a
// project that holds the files holds names, builds the mod's run context
// again: a file of a kind the run always reads the drafts of, or of a cited
// kind while the project holds the file of a kind that cites it.
func RebuildsContext(name string, holds func(rel string) bool) bool {
	cited := slices.ContainsFunc(dataKinds, func(k DataKind) bool {
		return k.text && k.Draft.Context == DraftAlways && slices.ContainsFunc(k.DraftNames(), holds)
	})
	return slices.ContainsFunc(dataKinds, func(k DataKind) bool {
		return k.DraftFile(name) && (k.Draft.Context == DraftAlways || k.Draft.Context == DraftCited && cited)
	})
}

// OverlayDrafts puts the map editor's drafts in view over d, the run of mod
// r.Mod's folder in language r.Lang, through each kind's draft reader; a
// kind whose records cite text reads it from the drafted strings.
func OverlayDrafts(r DataReader, d *Data, view DraftView) error {
	text := func() (textLookup, error) {
		for _, k := range dataKinds {
			if k.Draft.Context == DraftCited && k.lookup != nil {
				return k.lookup(r.Lang, view.Draft)
			}
		}
		return nil, nil
	}
	for _, k := range dataKinds {
		if k.draft == nil {
			continue
		}
		if err := k.draft(r, d, view, text); err != nil {
			return err
		}
	}
	return nil
}

// DraftText is the text lookup of a game in lang over the strings files
// file answers: the strings kind's read of its drafts.
func DraftText(lang string, file func(rel string) ([]byte, bool)) (textLookup, error) {
	return Lookup(lang, func(l string) (Strings, error) {
		rel := StringsFile(l)
		b, _ := file(rel)
		return ParseStrings(rel, b)
	})
}

// draftItems puts the draft of the items file over d through the kind's
// reader, its names read from the drafted strings, and hands the item layer
// the drafted sprites in place of the folder's. A project that holds no
// items file has none, whatever the folder holds: an undo past a save takes
// the saved file out. Records an older mod reads from another file stay.
func draftItems(r DataReader, d *Data, view DraftView, text func() (textLookup, error)) error {
	src, _ := view.Draft(ItemsFile)
	fixed := func(file string) bool { return namefold.ModTree(file) == namefold.ModTree(ItemsFile) }
	items := &d.Items
	items.Rows = slices.DeleteFunc(items.Rows, func(r ItemRow) bool { return fixed(r.File) })
	items.Changes = slices.DeleteFunc(items.Changes, func(c ItemChange) bool { return fixed(c.File) })
	items.Materials = slices.DeleteFunc(items.Materials, func(r ItemMaterial) bool { return fixed(r.File) })
	items.Weapons = slices.DeleteFunc(items.Weapons, func(r WeaponRow) bool { return fixed(r.File) })
	if items.Empty() {
		items.Mods = nil
	}
	if src != nil {
		look, err := text()
		if err != nil {
			return err
		}
		r.Text = look
		if err := readItems(r, d, ItemsFile, src); err != nil {
			return err
		}
	}
	for _, rel := range view.Drafted() {
		if b, _ := view.Draft(rel); strings.HasPrefix(namefold.ModTree(rel), AssetsFolder+"/"+ItemSprite.Kind+"/") && b != nil {
			if d.Items.Pictures == nil {
				d.Items.Pictures = map[string][]byte{}
			}
			d.Items.Pictures[rel] = b
		}
	}
	return nil
}

func draftCharacters(r DataReader, d *Data, view DraftView, text func() (textLookup, error)) error {
	src, _ := view.Draft(CharactersFile)
	d.Characters.Rows = slices.DeleteFunc(d.Characters.Rows, func(c CharacterRow) bool {
		return namefold.ModTree(c.File) == namefold.ModTree(CharactersFile)
	})
	if src == nil {
		return nil
	}
	look, err := text()
	if err != nil {
		return err
	}
	r.Text = look
	return readCharacters(r, d, CharactersFile, src)
}

// draftSpells puts the draft of the spells file over d's spells: the run of
// one mod reads its spells from the one file, so a project that holds no
// spells file has none, whatever the folder holds.
func draftSpells(r DataReader, d *Data, view DraftView, _ func() (textLookup, error)) error {
	src, _ := view.Draft(SpellsFile)
	d.Spells, d.spellGlobals, d.spellFields = SpellData{Order: d.Spells.Order}, nil, nil
	if src == nil {
		return nil
	}
	if err := readSpells(r, d, SpellsFile, src); err != nil {
		return err
	}
	for _, rel := range view.Drafted() {
		if b, _ := view.Draft(rel); strings.HasPrefix(namefold.ModTree(rel), AssetsFolder+"/"+SpellIcon.Kind+"/") && b != nil {
			if d.Spells.Pictures == nil {
				d.Spells.Pictures = map[string][]byte{}
			}
			d.Spells.Pictures[rel] = b
		}
	}
	return nil
}

// draftTerrain puts the draft of the terrain file over d's groups.
func draftTerrain(r DataReader, d *Data, view DraftView, _ func() (textLookup, error)) error {
	src, ok := view.Draft(TerrainFile)
	if !ok {
		return nil
	}
	d.Terrain = TerrainData{}
	return readTerrain(r, d, TerrainFile, src)
}

// draftSpeakers puts the draft of the speakers file, with the drafted
// pictures it names, over d's records.
func draftSpeakers(r DataReader, d *Data, view DraftView, _ func() (textLookup, error)) error {
	src, ok := view.Draft(SpeakersFile)
	if !ok {
		return nil
	}
	speakers, err := ReadSpeakers(r.Mod, r.Dir, SpeakersFile, src, view.Draft)
	if err != nil {
		return err
	}
	d.Speakers = speakers
	return nil
}
