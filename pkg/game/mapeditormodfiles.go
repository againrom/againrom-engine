package game

import (
	"errors"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/namefold"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// The files page's keys: "filter" narrows the install's paths, an import is
// "<language>:<install path>" and a removal "drop.<file in the mod>".
const (
	filesFilterKey = "filter"
	filesDropKey   = "drop."
)

// filesShown is the most install paths the files page lists at once.
const filesShown = 40

// installListing is the install's paths a mod may replace, listed once for
// the archives from.
type installListing struct {
	from  *vfs.FS
	paths []string
}

// editorSounds is the install's sound archive at root, opened once.
type editorSounds struct {
	root   string
	opened bool
	fs     *vfs.FS
}

// installSounds is the sound archive of the editor's install, nil when it
// has none.
func (x *MapInspector) installSounds() *vfs.FS {
	if x.archives == nil {
		return nil
	}
	if !x.sounds.opened || x.sounds.root != x.archives.Root {
		x.sounds = editorSounds{root: x.archives.Root, opened: true}
		if fs, err := openSoundArchive(x.archives.Root); err == nil {
			x.sounds.fs = fs
		}
	}
	return x.sounds.fs
}

// installPaths is the files of the install's archives and sound archive of
// a kind a mod replaces, in address order.
func (x *MapInspector) installPaths() []string {
	if x.archives == nil || x.archives.Containers == nil {
		return nil
	}
	if x.listing != nil && x.listing.from == x.archives.Containers {
		return x.listing.paths
	}
	entries := x.archives.Containers.Entries()
	if s := x.installSounds(); s != nil {
		entries = append(entries, s.Entries()...)
	}
	var paths []string
	for _, e := range entries {
		if _, ok := mod.InstallFileKind(e.Address); ok {
			paths = append(paths, e.Address)
		}
	}
	slices.Sort(paths)
	x.listing = &installListing{from: x.archives.Containers, paths: paths}
	return paths
}

// serveModFiles serves files on f's own file layer, a sound on a fork of the
// install's sound archive, as the game's launch serves them.
func (x *MapInspector) serveModFiles(f *FrontEnd, files mod.FileData) error {
	var sounds *vfs.FS
	if s := x.installSounds(); s != nil {
		sounds = s.Fork()
	}
	return f.modFileLayer(files.Order, sounds).serveInstallFiles(f.Archives, files.Files)
}

// heldInstallFiles is the files of files/ the project d holds, by
// "<language>:<install path>" folded. The project holds each such file of
// the folder with its bytes, so a file an undo took out is not held.
func (s *mapInspectionEdits) heldInstallFiles(d *editorModDocument) map[string]string {
	held := map[string]string{}
	for name, b := range d.Files {
		rest, ok := strings.CutPrefix(name, mod.FilesDir+"/")
		lang, address, two := strings.Cut(rest, "/")
		if ok && two && b != nil {
			held[namefold.Host(lang)+":"+mod.FoldAddress(address)] = name
		}
	}
	return held
}

// filesFields is W's files page: the filter, the files the mod replaces,
// the install's paths the filter matches with an import per language
// folder, and the override lines; none with no mod folder open.
func (s *mapInspectionEdits) filesFields() []ui.InspectionField {
	x := s.inspector
	if !s.modPageHolds() {
		return nil
	}
	d := s.workspace.Project
	if d == nil {
		d = x.mod.document()
	}
	line := x.pageLine
	held := s.heldInstallFiles(d)
	filter := workspaceText(filesFilterKey, x.word("editor.files.filter"), x.filesFilter)
	filter.Meaning = x.word("editor.files.filter.hint")
	paths := x.installPaths()
	want := mod.FoldAddress(x.filesFilter)
	var match []string
	for _, p := range paths {
		if strings.Contains(p, want) {
			match = append(match, p)
		}
	}
	out := []ui.InspectionField{filter, line("summary", "", fmt.Sprintf(x.word("editor.files.summary"), len(held), len(match), len(paths)))}
	group := x.word("editor.files.held")
	for _, name := range slices.Sorted(maps.Values(held)) {
		out = append(out, ui.InspectionField{Key: filesDropKey + name, Group: group, Label: fmt.Sprintf(x.word("editor.files.drop"), name), Action: true})
	}
	play := []string{mod.CommonFiles, modrt.LanguageFor(x.mod.base)}
	for _, p := range match[:min(len(match), filesShown)] {
		for _, lang := range mod.InstallLanguages() {
			f := workspaceText(lang+":"+p, fmt.Sprintf(x.word("editor.files.import"), languageLabel(lang)), "")
			f.Group, f.Pick = p, []string{strings.TrimPrefix(path.Ext(p), ".")}
			f.Meaning = x.word("editor.files.install")
			if name, ok := held[lang+":"+p]; ok {
				f.Meaning = fmt.Sprintf(x.word("editor.files.replaced"), name)
			}
			if !slices.Contains(play, lang) {
				f.Meaning += "; " + fmt.Sprintf(x.word("editor.files.otherlang"), languageLabel(lang))
			}
			out = append(out, f)
		}
	}
	if len(match) > filesShown {
		out = append(out, line("shown", "", fmt.Sprintf(x.word("editor.files.shown"), filesShown, len(match))))
	}
	group = x.word("editor.files.overrides")
	lines := s.filesOverrides(d)
	if len(lines) == 0 {
		lines = []string{x.word("editor.files.nooverrides")}
	}
	for i, l := range lines {
		out = append(out, line(fmt.Sprintf("override.%d", i), group, l))
	}
	if x.modNotEnabled(x.mod) {
		out = append(out, line(fmt.Sprintf("override.%d", len(lines)), group, x.word("editor.files.notenabled")))
	}
	return out
}

// setFilesField makes one edit of the files page: the filter narrows the
// list, an import or a removal is a draft step.
func (s *mapInspectionEdits) setFilesField(key, value string) (*ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	x := s.inspector
	if key == filesFilterKey {
		x.filesFilter = strings.TrimSpace(value)
		return nil, nil
	}
	if err := x.modWritable(); err != nil {
		return nil, err
	}
	if isModAssetKey(key) {
		d, _, err := s.setModAsset(key, value)
		return d, err
	}
	if rel, ok := strings.CutPrefix(key, filesDropKey); ok {
		held, ok := s.heldModFile(rel)
		if !ok || !strings.HasPrefix(held, mod.FilesDir+"/") {
			return nil, fmt.Errorf(x.word("editor.files.err.gone"), key)
		}
		return s.editKind(func(k kindFiles) error { return k.drop(held) })
	}
	lang, address, ok := strings.Cut(key, ":")
	if !ok {
		return nil, fmt.Errorf(x.word("editor.files.err.gone"), key)
	}
	_, d, err := s.ImportInstallFile(lang, address, strings.TrimSpace(value), false)
	return d, err
}

// ImportInstallFile copies the host file source into the open mod as its
// replacement of the install file at address in language folder lang,
// held to the folder's rules and, through the draft's context, to the rule
// the game serves it by. It is one undo step, through the one file import.
func (s *mapInspectionEdits) ImportInstallFile(lang, address, source string, replace bool) (string, *ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return "", nil, err
	}
	x := s.inspector
	if err := x.modWritable(); err != nil {
		return "", nil, err
	}
	if source == "" {
		return "", nil, errors.New(x.word("editor.files.err.nofile"))
	}
	if !slices.Contains(mod.InstallLanguages(), lang) {
		return "", nil, fmt.Errorf(x.word("editor.files.err.gone"), lang+":"+address)
	}
	rel := mod.FilesDir + "/" + lang + "/" + address
	check := func(data []byte) error {
		_, err := mod.DraftInstallFile(x.mod.id, rel, data)
		return err
	}
	data, err := x.hostFileBytes(source, mod.MaxInstallFileBytes, check, fmt.Errorf(x.word("editor.files.err.file"), source))
	if err != nil {
		return "", nil, err
	}
	return s.importModFile(rel, data, replace, func() (*ui.InspectionDocument, error) {
		_, d, err := s.ImportInstallFile(lang, address, source, true)
		return d, err
	})
}

// filesOverrides is the override lines the mod check prints for the open
// mod with the mods the starter enables, the project d's files/ in place of
// the folder's; made once per context.
func (s *mapInspectionEdits) filesOverrides(d *editorModDocument) []string {
	m := s.inspector.mod
	if m.filesCheck == nil {
		lines := s.inspector.modFileOverrides(m, projectView(m.files, d.Files))
		m.filesCheck = &lines
	}
	return *m.filesCheck
}

// modNotEnabled is whether the starter's list is known and m is not in it.
func (x *MapInspector) modNotEnabled(m *editorMod) bool {
	h := x.testPlay
	return h != nil && h.EnabledMods != nil && !slices.Contains(h.EnabledMods(), m.id)
}

// modFileOverrides serves the install files of the enabled mods, with drafts
// over m's when m is enabled, as the game's launch serves them, and answers
// its override lines, or the refusal as the one line.
func (x *MapInspector) modFileOverrides(m *editorMod, view draftView) []string {
	h := x.testPlay
	if h == nil || h.EnabledMods == nil || strings.TrimSpace(h.ModsFolder) == "" || x.archives == nil || x.archives.Containers == nil {
		return nil
	}
	var entries []mod.Entry
	for _, id := range h.EnabledMods() {
		if id == m.id {
			entries = append(entries, mod.Load(m.dir))
			continue
		}
		e, err := mod.Resolve(strings.TrimSpace(h.ModsFolder), []string{id})
		if err != nil {
			return []string{err.Error()}
		}
		entries = append(entries, e...)
	}
	if len(entries) < 2 {
		return nil
	}
	result, err := modrt.Load(entries, m.base, nil, modrt.Options{})
	if err != nil {
		return []string{err.Error()}
	}
	lang := modrt.LanguageFor(m.base)
	files := mod.FileData{Order: mod.PrecedenceOf(result.Ordered)}
	for _, e := range result.Ordered {
		var of []mod.InstallFile
		for _, f := range result.Files.Files {
			if f.Mod == e.Manifest.ID {
				of = append(of, f)
			}
		}
		if e.Manifest.ID == m.id {
			if of, err = mod.OverlayInstallFiles(m.id, lang, of, view); err != nil {
				return []string{err.Error()}
			}
		}
		files.Files = append(files.Files, of...)
	}
	archives := *x.archives
	archives.mods = nil
	f := &FrontEnd{InstallResources: InstallResources{Archives: &archives}}
	if err := x.serveModFiles(f, files); err != nil {
		return []string{err.Error()}
	}
	var out []string
	for _, o := range f.ModFileOverrides() {
		out = append(out, "override: "+o)
	}
	return out
}
