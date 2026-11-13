package game

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"againrom/pkg/mod"
	"againrom/pkg/namefold"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// modAsset is one sort of file a mod keeps under assets: the folder of its
// data kind, its extension, the rule its path passes and the check the
// loader of that kind runs on its bytes.
type modAsset struct {
	folder, ext string
	ref         mod.Reference
	check       func([]byte) error
}

// modAssets is every sort of file the tree keeps under assets. The import
// takes a file of each; a sort is added here, with its loader's own check.
// The rows of one folder stand in the order of the extensions the tree takes
// there, which is where each row's extension comes from.
var modAssets = []modAsset{
	{folder: "items", ref: mod.ItemSprite, check: func(b []byte) error { _, err := modIconSprite(b); return err }},
	{folder: "bodies", ref: mod.BodySheetPicture, check: func(b []byte) error { _, err := decodeBodySheet(b); return err }},
	{folder: speakerAssets, ref: mod.SpeakerPicture, check: func(b []byte) error { _, err := decodeSpeakerPicture(b); return err }},
	{folder: "towns", ref: mod.TownDescription, check: func(b []byte) error { _, err := town.Parse(b); return err }},
	{folder: "towns", ref: mod.TownArt, check: func(b []byte) error { _, err := decodeModPicture(b, bmpDepth(b)); return err }},
}

func init() {
	seen := map[string]int{}
	for i := range modAssets {
		a := &modAssets[i]
		k, _ := mod.AssetKind(a.folder)
		if _, exts := k.AssetPlace(); seen[a.folder] < len(exts) {
			a.ext = exts[seen[a.folder]]
		}
		seen[a.folder]++
	}
}

// modAssetImportKey starts the mod window's key that imports a file of the
// kind named after it; the value is the host path of the file.
const (
	modAssetImportKey  = "import."
	modAssetReplaceKey = "import-replace"
)

// modAssetRequest is an import that waits for the answer to its question.
type modAssetRequest struct{ kind, source string }

// modAssetName is the name an imported file takes: the picked file's name in
// lower case, its stem cut so the name stays within a packed component.
func modAssetName(source string) string {
	name := namefold.Host(filepath.Base(source))
	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for len(stem)+len(ext) > mod.MaxNameBytes && stem != "" {
		_, size := utf8.DecodeLastRuneInString(stem)
		stem = stem[:len(stem)-size]
	}
	return stem + ext
}

// modAssetDraft is the one check of a file on its way into the open mod: the
// host file source as a draft of kind named name, at assets/<kind>/<name>.
// It answers the draft's path and bytes and changes nothing. The mod must be
// a folder the editor writes, the name pass the kind's reference rule, and
// the bytes pass the check the kind's loader runs.
func (s *mapInspectionEdits) modAssetDraft(kind, source, name string) (string, []byte, error) {
	x := s.inspector
	if x.mod == nil {
		return "", nil, errors.New(x.word("editor.mod.err.nofolder"))
	}
	if err := x.modReadOnly(); err != nil {
		return "", nil, err
	}
	if x.mod.copyOnly {
		return "", nil, errors.New(x.word("editor.mod.err.imported"))
	}
	var takes []string
	var asset *modAsset
	for i, a := range modAssets {
		if a.folder != kind {
			continue
		}
		takes = append(takes, a.ext)
		if namefold.HostEqual(path.Ext(name), a.ext) {
			asset = &modAssets[i]
		}
	}
	if len(takes) == 0 {
		return "", nil, fmt.Errorf(x.word("editor.asset.err.kind"), kind)
	}
	if asset == nil {
		return "", nil, fmt.Errorf(x.word("editor.asset.err.type"), name, mod.AssetsFolder+"/"+kind, strings.Join(takes, ", "))
	}
	rel, err := asset.ref.Path(mod.AssetsFolder + "/" + kind + "/" + name)
	if err != nil {
		return "", nil, err
	}
	st, err := os.Stat(source)
	if err != nil {
		return "", nil, err
	}
	if st.IsDir() {
		return "", nil, fmt.Errorf(x.word("editor.asset.err.type"), name, mod.AssetsFolder+"/"+kind, strings.Join(takes, ", "))
	}
	if st.Size() > mod.MaxFileBytes {
		return "", nil, fmt.Errorf(x.word("editor.asset.err.size"), mod.MaxFileBytes)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return "", nil, err
	}
	if err := asset.check(data); err != nil {
		return "", nil, err
	}
	return rel, data, nil
}

// heldModFile is the spelling under which the open project holds the file
// rel names, matched as the mod tree matches names; false when it holds none.
func (s *mapInspectionEdits) heldModFile(rel string) (string, bool) {
	var d *editorModDocument
	if s.workspace != nil {
		d = s.workspace.Project
	}
	if d == nil {
		d = &editorModDocument{}
	}
	for _, name := range s.projectFiles(d) {
		if namefold.ModTree(name) == namefold.ModTree(rel) {
			return name, true
		}
	}
	if m := s.inspector.mod; m != nil && d.Files[rel] == nil {
		if _, removed := d.Files[rel]; !removed {
			if st, err := os.Stat(filepath.Join(m.dir, filepath.FromSlash(rel))); err == nil && !st.IsDir() {
				return rel, true
			}
		}
	}
	return "", false
}

// ImportModAsset copies the host file source into the open mod as a draft at
// assets/<kind>/<name>, the name being the file's own in lower case. It is
// one undo step and Ctrl+S writes it with the rest of the project. A file
// the project holds under that name is replaced only when replace is set;
// otherwise the answer is the question to ask.
func (s *mapInspectionEdits) ImportModAsset(kind, source string, replace bool) (string, *ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return "", nil, err
	}
	rel, data, err := s.modAssetDraft(kind, source, modAssetName(source))
	if err != nil {
		return "", nil, err
	}
	if held, ok := s.heldModFile(rel); ok {
		if !replace {
			s.assetAsk = &modAssetRequest{kind: kind, source: source}
			x := s.inspector
			return "", nil, &ui.WorkspaceConfirmation{Action: modAssetReplaceKey, Message: fmt.Sprintf(x.word("editor.asset.replace"), held), Confirm: x.word("editor.mod.replacefile")}
		}
		rel = held
	}
	m := s.inspector.mod
	d, err := s.changeWorkspace(func(w *editorWorkspaceState) error {
		if w.Project == nil {
			w.Project = m.document()
		}
		w.Project.Files[rel] = bytes.Clone(data)
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	s.saveNote = fmt.Sprintf(s.inspector.word("editor.asset.imported"), rel)
	return rel, d, nil
}

// setModAsset is the mod window's route to the import: the key names the
// kind and the value the host file, and the answer to a replace question
// repeats the import that asked.
func (s *mapInspectionEdits) setModAsset(key, value string) (*ui.InspectionDocument, bool, error) {
	if key == modAssetReplaceKey {
		ask := s.assetAsk
		s.assetAsk = nil
		if ask == nil || value != "confirmed" {
			return nil, true, nil
		}
		_, d, err := s.ImportModAsset(ask.kind, ask.source, true)
		return d, true, err
	}
	kind, ok := strings.CutPrefix(key, modAssetImportKey)
	if !ok {
		return nil, false, nil
	}
	s.assetAsk = nil
	_, d, err := s.ImportModAsset(kind, strings.TrimSpace(value), false)
	return d, true, err
}

// isModAssetKey reports a key of the import route.
func isModAssetKey(key string) bool {
	return key == modAssetReplaceKey || strings.HasPrefix(key, modAssetImportKey)
}

// modAssetKinds is the kinds the import takes, in the order of modAssets.
func modAssetKinds() []string {
	var out []string
	for _, a := range modAssets {
		if !slices.Contains(out, a.folder) {
			out = append(out, a.folder)
		}
	}
	return out
}
