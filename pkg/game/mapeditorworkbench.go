package game

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"againrom/pkg/launch"
	"againrom/pkg/mod"
	"againrom/pkg/namefold"
)

// editorWorkbenchFolder is the folder of the player profile the mod window
// proposes working mods in, beside the profile's saves and maps.
const editorWorkbenchFolder = "workbench"

// profileFolder is the player profile's subfolder name, made when absent, and
// the grant that admits writes beneath it when the profile sits inside the
// install. It answers none when the editor has no player profile.
func (x *MapInspector) profileFolder(name string) (string, *runtimeProfileAccess) {
	if x.testPlay == nil || x.testPlay.Profile == nil {
		return "", nil
	}
	profile, err := x.testPlay.Profile()
	if err != nil {
		return "", nil
	}
	root := ""
	if x.archives != nil {
		root = x.archives.Root
	}
	dir, grant, err := profile.folder(name, root)
	if err != nil {
		return "", nil
	}
	return dir, grant
}

// hostFolderName reports whether every host keeps a folder named id: Windows
// keeps none that ends in a dot or whose name before its first dot is a device.
func hostFolderName(id string) bool {
	stem, _, _ := strings.Cut(id, ".")
	switch stem {
	case "con", "prn", "aux", "nul":
		return false
	}
	if len(stem) == 4 && (stem[:3] == "com" || stem[:3] == "lpt") && stem[3] >= '0' && stem[3] <= '9' {
		return false
	}
	return !strings.HasSuffix(id, ".")
}

// proposedModFolder is the working folder the mod window proposes for a mod
// that has none yet: the folder id of the profile's workbench. It makes no
// folder, and answers none without a player profile or an id that can name a
// folder on every host.
func (x *MapInspector) proposedModFolder(id string) string {
	if x.testPlay == nil || x.testPlay.Profile == nil || !mod.ValidID(id) || !hostFolderName(id) {
		return ""
	}
	profile, err := x.testPlay.Profile()
	if err != nil || profile.Directory == "" {
		return ""
	}
	return filepath.Join(profile.Directory, editorWorkbenchFolder, id)
}

// modWriteGrants admits a mod's writes, with the programs inside an install,
// beneath the workbench and beneath the resolved mods folder when it is the
// one launch.ModsFolder names beside the programs; both are made when absent.
// Elsewhere it grants nothing.
func (x *MapInspector) modWriteGrants() []*runtimeProfileAccess {
	if x.testPlay == nil || x.testPlay.Profile == nil {
		return nil
	}
	profile, err := x.testPlay.Profile()
	if err != nil || profile.Saves.profile == nil {
		return nil
	}
	root := ""
	if x.archives != nil {
		root = x.archives.Root
	}
	_, workbench, err := profile.folder(editorWorkbenchFolder, root)
	if err != nil || workbench == nil {
		return nil
	}
	grants := []*runtimeProfileAccess{workbench}
	named := strings.TrimSpace(x.testPlay.ModsFolder)
	if named == "" {
		return grants
	}
	mods, err := filepath.Abs(named)
	if err != nil {
		return grants
	}
	parent, err := editorPhysicalDirectory(filepath.Dir(mods))
	beside := launch.ModsFolder("", filepath.Dir(workbench.root.path))
	if err != nil || !sameSaveName(filepath.Join(parent, filepath.Base(mods)), beside) {
		return grants
	}
	dir, err := createProfileDirectory(beside)
	if err != nil {
		return grants
	}
	grant := &runtimeProfileAccess{root: dir, directory: dir, descendants: true}
	if allowed, err := grant.allows(dir.path); err != nil || !allowed {
		return grants
	}
	return append(grants, grant)
}

// makeWorkbench makes the profile's workbench when dir is a proposed working
// folder, so the folder a new mod is saved into has its parent.
func (x *MapInspector) makeWorkbench(dir string) {
	want := x.proposedModFolder(filepath.Base(dir))
	if want != "" && namefold.HostEqual(filepath.Clean(want), filepath.Clean(dir)) {
		x.profileFolder(editorWorkbenchFolder)
	}
}

// publishFolder is the folder Publish writes the packed mod into: the one
// typed in the mod window, else the mods folder the editor's program
// resolved, which the starter and the game read. typed tells them apart.
func (x *MapInspector) publishFolder() (dir string, typed bool) {
	if x.modOutput != "" {
		return x.modOutput, true
	}
	if x.testPlay == nil {
		return "", false
	}
	return strings.TrimSpace(x.testPlay.ModsFolder), false
}

// setPublishFolder takes in the Publish folder field. A blank value, or the
// resolved mods folder itself, returns Publish to the resolved folder; any
// other must be an existing folder.
func (x *MapInspector) setPublishFolder(value string) error {
	value = strings.TrimSpace(value)
	x.modOutput = ""
	if resolved, _ := x.publishFolder(); value == "" || resolved != "" && namefold.HostEqual(filepath.Clean(resolved), filepath.Clean(value)) {
		return nil
	}
	dir, err := editorPhysicalDirectory(value)
	if err != nil {
		return err
	}
	x.modOutput = dir
	return nil
}

// resolvedPublishFolder is the physical folder Publish writes into and makes
// nothing. A typed folder must exist. The resolved mods folder may be absent:
// then it is its physical parent with its name, absent is set, and Publish
// makes it only when it writes the packed mod.
func (x *MapInspector) resolvedPublishFolder() (out string, absent bool, err error) {
	dir, typed := x.publishFolder()
	if dir == "" {
		return "", false, errors.New(x.word("editor.mod.err.nooutput"))
	}
	if _, err := os.Lstat(dir); !typed && errors.Is(err, os.ErrNotExist) {
		parent, err := editorPhysicalDirectory(filepath.Dir(dir))
		if err != nil {
			return "", false, err
		}
		return filepath.Join(parent, filepath.Base(dir)), true, nil
	}
	out, err = editorPhysicalDirectory(dir)
	return out, false, err
}
