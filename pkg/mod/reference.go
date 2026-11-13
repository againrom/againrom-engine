package mod

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"againrom/pkg/namefold"
)

type referenceRole uint8

const (
	referenceFile referenceRole = iota
	referenceMap
	referencePages
	referenceTown
)

// Reference is the one rule a value that names a file of the mod passes: a
// row field of a data file, or a "mod:" art key of a town description. The
// value is a relative slash path inside the mod with no dotted component,
// each component within the packed-name bound, and the extension the kind's
// reader takes when it takes one.
type Reference struct {
	// Kind is the data kind whose rows hold the value.
	Kind string
	// tables and fields are the rows and keys the tree description reads
	// the value from; none for a value that is not a row's.
	tables, fields []string
	// exts are the extensions the kind's reader takes; none takes any.
	exts []string
	// loose reads an unclean spelling as its clean path; otherwise it is
	// refused.
	loose bool
	role  referenceRole
}

// The values of a mod's files that name another file of the mod.
var (
	ItemSprite       = Reference{Kind: "items", tables: []string{"item", "change", "weapon", "material"}, fields: []string{"sprite"}, exts: []string{".png"}}
	BodySheetPicture = Reference{Kind: "body sheets", tables: []string{"body"}, fields: []string{"heroes", "heroes_l"}, exts: []string{".png"}}
	MissionMap       = Reference{Kind: "missions", tables: []string{"mission"}, fields: []string{"map"}, loose: true, role: referenceMap}
	MissionTexts     = Reference{Kind: "missions", tables: []string{"mission"}, fields: []string{"texts"}, loose: true, role: referencePages}
	TownDescription  = Reference{Kind: "towns", tables: []string{"town"}, fields: []string{"description"}, exts: []string{".json"}, loose: true, role: referenceTown}
	TownArt          = Reference{Kind: "towns", loose: true}
	SpeakerPicture   = Reference{Kind: "speakers", tables: []string{"speaker"}, fields: []string{"picture"}, loose: true}
)

const errReferenceInside = "%q is not a path inside the mod folder"

// Names reports whether value carries an extension the kind's reader takes,
// without case: the test that tells a file of the mod from another name.
func (r Reference) Names(value string) bool {
	return slices.ContainsFunc(r.exts, func(ext string) bool { return namefold.HostEqual(path.Ext(value), ext) })
}

// Path is the clean path inside the mod that value names, or why it names
// none.
func (r Reference) Path(value string) (string, error) {
	if value == "" || strings.ContainsAny(value, "\\:\x00") || path.IsAbs(value) {
		return "", fmt.Errorf(errReferenceInside, value)
	}
	rel := path.Clean(value)
	if outside := rel == "." || rel == ".." || strings.HasPrefix(rel, "../"); !r.loose && (outside || rel != value) {
		return "", fmt.Errorf("%q is not a clean path inside the mod folder", value)
	} else if outside {
		return "", fmt.Errorf(errReferenceInside, value)
	}
	if len(r.exts) != 0 && !r.Names(rel) {
		return "", fmt.Errorf("%q is not a %s file", value, strings.Join(r.exts, " or "))
	}
	if err := refuseDotted(rel); err != nil {
		return "", err
	}
	for _, c := range strings.Split(rel, "/") {
		if err := checkPackedComponent(c); err != nil {
			return "", fmt.Errorf("%s: %v", rel, err)
		}
	}
	return rel, nil
}

// Check is Path for the value of field at line of file, its refusal naming
// them.
func (r Reference) Check(file string, line int, field, value string) (string, error) {
	rel, err := r.Path(value)
	if err != nil {
		return "", &ItemFileError{File: file, Line: line, Msg: field + " " + err.Error()}
	}
	return rel, nil
}

// Read reads the file value names from the tree, with its clean path. A
// value the rule refuses and a file the tree does not hold are both errors.
func (r Reference) Read(t Tree, value string) ([]byte, string, error) {
	rel, err := r.Path(value)
	if err != nil {
		return nil, "", err
	}
	return t.ReadInside(rel)
}

// ReadIn is Read from the mod at dir, a folder or a packed file.
func (r Reference) ReadIn(dir, value string) ([]byte, string, error) {
	t, err := OpenTree(dir)
	if err != nil {
		return nil, "", err
	}
	return r.Read(t, value)
}

// AssetPlace is the folder under assets a mod's own files of the kind lie in
// and the extensions the tree takes there; "" for a kind with none.
func (k DataKind) AssetPlace() (folder string, exts []string) {
	for _, p := range k.Paths {
		parts := strings.Split(p, "/")
		if len(parts) < 3 || parts[0] != AssetsFolder {
			continue
		}
		folder = parts[0] + "/" + parts[1]
		exts = append(exts, path.Ext(p))
	}
	return folder, exts
}

// AssetsFolder holds a mod's own pictures and other files, one folder a
// kind.
const AssetsFolder = "assets"

// MaxNameBytes is the most bytes one component of a path in a mod holds.
const MaxNameBytes = maxPackedNameBytes

// AssetKind is the data kind whose own files lie in assets/<folder>.
func AssetKind(folder string) (DataKind, bool) {
	for _, k := range dataKinds {
		if place, _ := k.AssetPlace(); place == AssetsFolder+"/"+folder {
			return k, true
		}
	}
	return DataKind{}, false
}

// inPlace reports whether rel lies where the tree keeps the kind's own
// files; a kind with no such place keeps them anywhere.
func (k DataKind) inPlace(rel string) bool {
	place, _ := k.AssetPlace()
	return place == "" || slices.ContainsFunc(k.Paths, func(p string) bool { return treeMatch(p, rel) })
}

// inFolder reports whether rel lies in the kind's folder under assets,
// whatever its extension: the place of a file an art key names, which the
// game reads in every art format.
func (k DataKind) inFolder(rel string) bool {
	place, _ := k.AssetPlace()
	return place == "" || strings.HasPrefix(namefold.ModTree(rel), namefold.ModTree(place)+"/")
}

// folderAdvice names where a file an art key names as rel belongs.
func (k DataKind) folderAdvice(rel string) string {
	place, _ := k.AssetPlace()
	return place + "/<name>" + path.Ext(rel)
}

// placeAdvice names where a file of the kind named as rel belongs.
func (k DataKind) placeAdvice(rel string) string {
	place, exts := k.AssetPlace()
	ext := exts[0]
	for _, e := range exts {
		if namefold.HostEqual(path.Ext(rel), e) {
			ext = e
		}
	}
	return place + "/<name>" + ext
}
