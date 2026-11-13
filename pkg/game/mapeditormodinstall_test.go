package game

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/launch"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// modInstallFixture is the mod window of an editor whose programs stand in the
// install root.
func modInstallFixture(t *testing.T, modsFolder string) (install string, e *ui.MapEditor, s *mapInspectionEdits) {
	t.Helper()
	t.Setenv("AGAINROM_PROFILE", "")
	f, e := workspaceFixture(t)
	install = f.root
	if resolved, err := filepath.EvalSymlinks(install); err == nil {
		install = resolved
	}
	for _, name := range []string{GraphicsArchive, WorldArchive, MoviesArchive} {
		if err := os.WriteFile(filepath.Join(install, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if modsFolder == "" {
		modsFolder = launch.ModsFolder("", install)
	}
	exe := filepath.Join(install, "mapedit.exe")
	f.x.testPlay.Profile = func() (RuntimeProfile, error) { return ResolveRuntimeProfile("", exe, t.TempDir(), install) }
	f.x.testPlay.ModsFolder = modsFolder
	if err := e.Key("package"); err != nil {
		t.Fatal(err)
	}
	return install, e, e.Document().Edits.(*mapInspectionEdits)
}

// Save and Publish write beneath the two admitted folders and nowhere else.
func TestModSaveAndPublishBesideProgramsInsideTheInstall(t *testing.T) {
	install, e, s := modInstallFixture(t, "")
	before := installFiles(t, install)
	working := filepath.Join(install, "Againrom", "workbench", "authored-map")
	if got := modWindowField(t, e, "folder").TextValue; got != working {
		t.Fatalf("proposed working folder %q, want %q", got, working)
	}
	if _, err := s.SetWorkspaceField("package", "save", ""); err != nil {
		t.Fatalf("Save into the proposed working folder: %v", err)
	}
	s = e.Document().Edits.(*mapInspectionEdits)
	if _, err := os.Stat(filepath.Join(working, mod.ManifestName)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetWorkspaceField("package", "publish", ""); err != nil {
		t.Fatalf("Publish into the mods folder: %v", err)
	}
	file := filepath.Join(install, "mods", "mod-authored-map.res")
	if st, err := os.Stat(file); err != nil || !st.Mode().IsRegular() {
		t.Fatalf("Publish wrote no %s: %v", file, err)
	}
	after := installFiles(t, install)
	for p, b := range before {
		if after[p] != b {
			t.Fatalf("install file changed: %s", p)
		}
	}
	for p := range after {
		if _, had := before[p]; had {
			continue
		}
		if !insideDir(filepath.Join(install, "Againrom"), p) && !insideDir(filepath.Join(install, "mods"), p) {
			t.Fatalf("a write outside the two admitted folders: %s", p)
		}
	}
}

// A typed folder elsewhere in the install is refused in the fence's words.
func TestModWritesElsewhereInTheInstallStayRefused(t *testing.T) {
	install, e, s := modInstallFixture(t, "")
	sub := filepath.Join(install, "sub")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	want := fenceEditorWrite(install, sub, "Save mod package")
	if want == nil {
		t.Fatal("the fence admits a folder of the install")
	}
	if _, err := s.SetWorkspaceField("package", "folder", filepath.Join(sub, "authored-map")); err == nil || err.Error() != want.Error() {
		t.Fatalf("Save into %s answered %v, want %v", sub, err, want)
	}
	if _, err := s.SetWorkspaceField("package", "save", ""); err != nil {
		t.Fatal(err)
	}
	s = e.Document().Edits.(*mapInspectionEdits)
	if _, err := s.SetWorkspaceField("package", "output", sub); err != nil {
		t.Fatal(err)
	}
	want = fenceEditorWrite(install, sub, "Publish")
	if _, err := s.SetWorkspaceField("package", "publish", ""); err == nil || !strings.HasSuffix(err.Error(), ": "+want.Error()) {
		t.Fatalf("Publish into %s answered %v, want %v", sub, err, want)
	}
	if entries, err := os.ReadDir(sub); err != nil || len(entries) != 0 {
		t.Fatalf("the typed folder holds %v, %v", entries, err)
	}
}

// A mods folder named as the install root is refused.
func TestModPublishRefusesAModsFolderAtTheInstallRoot(t *testing.T) {
	install, e, s := modInstallFixture(t, "")
	s.inspector.testPlay.ModsFolder = install
	if _, err := s.SetWorkspaceField("package", "save", ""); err != nil {
		t.Fatal(err)
	}
	s = e.Document().Edits.(*mapInspectionEdits)
	if _, err := s.SetWorkspaceField("package", "publish", ""); !errors.Is(err, errWriteUnderInstall) {
		t.Fatalf("Publish into the install root answered %v", err)
	}
	if _, err := os.Lstat(filepath.Join(install, "mod-authored-map.res")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("Publish wrote into the install root", err)
	}
}

// The other writers refuse the two admitted folders.
func TestOtherWritersRefuseTheAdmittedFolders(t *testing.T) {
	install, e, s := modInstallFixture(t, "")
	if _, err := s.SetWorkspaceField("package", "save", ""); err != nil {
		t.Fatal(err)
	}
	s = e.Document().Edits.(*mapInspectionEdits)
	if len(s.inspector.modWriteGrants()) != 2 {
		t.Fatal("the editor's mod writers hold no grant for both folders")
	}
	working := filepath.Join(install, "Againrom", "workbench", "authored-map")
	mods := filepath.Join(install, "mods")
	fe := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Root: install}}}
	for _, dir := range []string{filepath.Join(install, "Againrom", "workbench"), mods} {
		if err := refuseBaseDumpInInstall(install, filepath.Join(dir, "base")); !errors.Is(err, ErrBaseDumpInstall) {
			t.Errorf("base dump into %s answered %v", dir, err)
		}
		if err := fe.checkHeadlessFramePath(filepath.Join(dir, "frame.png")); !errors.Is(err, ErrHeadlessFrameInstall) {
			t.Errorf("headless frame into %s answered %v", dir, err)
		}
		if _, err := PackMod(working, dir, false); !errors.Is(err, ErrPackInstall) {
			t.Errorf("-pack-mod into %s answered %v", dir, err)
		}
	}
	for _, dir := range []string{working, mods} {
		if _, err := AssembleModMission(install, "en", 1, dir); err == nil || !strings.Contains(err.Error(), "is inside the install") {
			t.Errorf("modassemble -out %s answered %v", dir, err)
		}
	}
	if entries, err := os.ReadDir(mods); err != nil || len(entries) != 0 {
		t.Fatalf("the mods folder holds %v, %v", entries, err)
	}
	if strings.TrimSpace(s.inspector.testPlay.ModsFolder) != mods {
		t.Fatalf("mods folder %q", s.inspector.testPlay.ModsFolder)
	}
}
