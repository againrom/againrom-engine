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
	{folder: mod.SpellIcon.Kind, ref: mod.SpellIcon, check: func(b []byte) error { _, err := decodeSpellIcon(b); return err }},
	{folder: mod.SpellEffect.Kind, ref: mod.SpellEffect, check: effectSheetCheck(true)},
	{folder: mod.SpellEffect.Kind, ref: mod.SpellEffect, check: effectSheetCheck(false)},
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

// modAssetRequest is an import that waits for the answer to its question:
// again repeats it with the replacement allowed.
type modAssetRequest struct {
	again func() (*ui.InspectionDocument, error)
}

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
	rel, asset, err := s.modAssetPlace(kind, name)
	if err != nil {
		return "", nil, err
	}
	data, err := x.hostFileBytes(source, mod.MaxFileBytes, asset.check, x.notModAsset(kind, name))
	if err != nil {
		return "", nil, err
	}
	return rel, data, nil
}

// modAssetBytes is modAssetDraft for bytes the editor holds rather than a
// host file: a copy from a game install the editor reads.
func (s *mapInspectionEdits) modAssetBytes(kind, name string, data []byte) (string, error) {
	x := s.inspector
	rel, asset, err := s.modAssetPlace(kind, name)
	if err != nil {
		return "", err
	}
	if int64(len(data)) > mod.MaxFileBytes {
		return "", fmt.Errorf(x.word("editor.asset.err.size"), mod.MaxFileBytes)
	}
	if err := asset.check(data); err != nil {
		return "", x.inLanguage(err)
	}
	return rel, nil
}

// modAssetPlace is the path in the open mod a file of kind named name
// takes and the sort of asset it is.
func (s *mapInspectionEdits) modAssetPlace(kind, name string) (string, *modAsset, error) {
	x := s.inspector
	if err := x.modWritable(); err != nil {
		return "", nil, err
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
		return "", nil, x.notModAsset(kind, name)
	}
	rel, err := asset.ref.Path(mod.AssetsFolder + "/" + kind + "/" + name)
	if err != nil {
		return "", nil, err
	}
	return rel, asset, nil
}

// notModAsset is the refusal of a file named name that is not of a sort
// the mod keeps under assets/<kind>.
func (x *MapInspector) notModAsset(kind, name string) error {
	var takes []string
	for _, a := range modAssets {
		if a.folder == kind {
			takes = append(takes, a.ext)
		}
	}
	return fmt.Errorf(x.word("editor.asset.err.type"), name, mod.AssetsFolder+"/"+kind, strings.Join(takes, ", "))
}

// modWritable refuses an import into no mod, a mod the editor may not
// write, and a packed mod the editor holds as a copy.
func (x *MapInspector) modWritable() error {
	if x.mod == nil {
		return errors.New(x.word("editor.mod.err.nofolder"))
	}
	if err := x.modReadOnly(); err != nil {
		return err
	}
	if x.mod.copyOnly {
		return errors.New(x.word("editor.mod.err.imported"))
	}
	return nil
}

// hostFileBytes is the bytes of the host file source on its way into a
// mod: a file of at most limit bytes, the bound of its kind of mod file,
// that passes check, the loader's own rule for its kind; notFile is the
// refusal of a folder.
func (x *MapInspector) hostFileBytes(source string, limit int64, check func([]byte) error, notFile error) ([]byte, error) {
	st, err := os.Stat(source)
	if err != nil {
		return nil, err
	}
	if st.IsDir() {
		return nil, notFile
	}
	if st.Size() > limit {
		return nil, fmt.Errorf(x.word("editor.asset.err.size"), limit)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}
	if err := check(data); err != nil {
		return nil, x.inLanguage(err)
	}
	return data, nil
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
	return s.importModFile(rel, data, replace, func() (*ui.InspectionDocument, error) {
		_, d, err := s.ImportModAsset(kind, source, true)
		return d, err
	})
}

// importModFile puts data into the open project at rel as one undo step,
// the commit of every import; a held file is replaced only when replace is
// set, otherwise the answer is the question to ask.
// A with edit runs in the import's own draft step, given the file's path.
func (s *mapInspectionEdits) importModFile(rel string, data []byte, replace bool, again func() (*ui.InspectionDocument, error), with ...func(*editorWorkspaceState, string) error) (string, *ui.InspectionDocument, error) {
	var each []func(*editorWorkspaceState, []string) error
	for _, edit := range with {
		each = append(each, func(w *editorWorkspaceState, rels []string) error { return edit(w, rels[0]) })
	}
	rels, d, err := s.importModFiles([]modDraftFile{{rel, data}}, replace, again, each...)
	if err != nil {
		return "", nil, err
	}
	return rels[0], d, nil
}

// modDraftFile is one file an import puts into the open project.
type modDraftFile struct {
	rel  string
	data []byte
}

// importModFiles is importModFile for files put together in one undo
// step; with is given their paths in the order of files.
func (s *mapInspectionEdits) importModFiles(files []modDraftFile, replace bool, again func() (*ui.InspectionDocument, error), with ...func(*editorWorkspaceState, []string) error) ([]string, *ui.InspectionDocument, error) {
	rels := make([]string, len(files))
	for i, f := range files {
		rels[i] = f.rel
		if held, ok := s.heldModFile(f.rel); ok {
			if !replace {
				s.assetAsk = &modAssetRequest{again: again}
				x := s.inspector
				return nil, nil, &ui.WorkspaceConfirmation{Action: modAssetReplaceKey, Message: fmt.Sprintf(x.word("editor.asset.replace"), held), Confirm: x.word("editor.mod.replacefile")}
			}
			rels[i] = held
		}
	}
	m := s.inspector.mod
	d, err := s.changeWorkspace(func(w *editorWorkspaceState) error {
		if w.Project == nil {
			w.Project = m.document()
		}
		for i, f := range files {
			w.Project.Files[rels[i]] = bytes.Clone(f.data)
		}
		for _, edit := range with {
			if err := edit(w, rels); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	s.saveNote = fmt.Sprintf(s.inspector.word("editor.asset.imported"), strings.Join(rels, ", "))
	return rels, d, nil
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
		d, err := ask.again()
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

// importedModFile reports a project file an import or a page drafts: one
// under assets, where the import writes, or a file of a kind the project
// holds whole. A save drops from the folder such a file the last save wrote
// and the project no longer holds.
func importedModFile(name string) bool {
	return strings.HasPrefix(name, mod.AssetsFolder+"/") || mod.HeldDraft(name)
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
