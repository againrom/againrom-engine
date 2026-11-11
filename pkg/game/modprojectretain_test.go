package game

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"againrom/pkg/hostfile"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
)

func reopenProject(t *testing.T, s *mapInspectionEdits, dir string) {
	t.Helper()
	if err := s.inspector.SetMod(dir); err != nil {
		t.Fatal(err)
	}
	s.workspace = nil
	if err := s.prepareWorkspace(); err != nil {
		t.Fatal(err)
	}
}

func TestProjectSaveTouchesOnlyItsChangedFiles(t *testing.T) {
	for _, owned := range []bool{false, true} {
		t.Run(strconv.FormatBool(owned), func(t *testing.T) {
			s, dir := modSpeakerProject(t)
			if owned {
				dir = filepath.Join(t.TempDir(), "ambush")
				if _, err := s.saveWorkspacePackage(dir); err != nil {
					t.Fatal(err)
				}
			}
			note := filepath.Join(dir, "author-notes.txt")
			readOnly := filepath.Join(dir, "reference.txt")
			writeTestPlayInput(t, note, []byte("original notes"))
			writeTestPlayInput(t, readOnly, []byte("kept"))
			stamp := time.Unix(1234567890, 0)
			if err := os.Chtimes(readOnly, stamp, stamp); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(readOnly, 0o444); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(readOnly, 0o644) })
			info, err := os.Stat(readOnly)
			if err != nil {
				t.Fatal(err)
			}
			reopenProject(t, s, dir)
			writeTestPlayInput(t, note, []byte("notes changed elsewhere"))
			missionPath := filepath.Join(dir, mod.MissionsFile)
			declarations, err := os.ReadFile(missionPath)
			if err != nil {
				t.Fatal(err)
			}
			writeTestPlayInput(t, missionPath, append([]byte("# outside edit\n"), declarations...))
			held, err := os.Open(note)
			if err != nil {
				t.Fatal(err)
			}
			defer held.Close()
			before := tree(t, dir)
			if err := s.model.SetDescription("changed map"); err != nil {
				t.Fatal(err)
			}
			if _, err := s.SaveIntoMod(); err != nil {
				t.Fatal(err)
			}
			after := tree(t, dir)
			entry := s.inspector.mod.missions.Missions[0]
			for name, data := range before {
				if name != entry.Map && after[name] != data {
					t.Errorf("changed an unwritten file: %s", name)
				}
			}
			if len(before) != len(after) || after[entry.Map] != string(s.model.Bytes()) || s.inspector.mod.owned != owned {
				t.Fatal("save changed the file list, map or ownership")
			}
			got, err := os.Stat(readOnly)
			if err != nil || got.Mode() != info.Mode() || !got.ModTime().Equal(info.ModTime()) {
				t.Fatalf("unwritten file attributes changed: %v", err)
			}
		})
	}
}

func TestProjectSaveWithTheFolderHeld(t *testing.T) {
	s, dir := modSpeakerProject(t)
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Error(err)
		}
	}()
	if err := s.model.SetDescription("save while held"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
}

func TestProjectSaveRefusesOnlyAnExternallyChangedTarget(t *testing.T) {
	for _, rel := range []string{mod.ManifestName, mod.MissionsFile, "mission/en/map.alm"} {
		t.Run(rel, func(t *testing.T) {
			s, dir := modSpeakerProject(t)
			switch rel {
			case mod.ManifestName:
				if _, err := s.SetWorkspaceField("package", "author", "Draft author"); err != nil {
					t.Fatal(err)
				}
			case mod.MissionsFile:
				s.workspace.Project.Missions[0].Title = "Draft mission title"
			default:
				if err := s.model.SetDescription("draft map"); err != nil {
					t.Fatal(err)
				}
			}
			path := filepath.Join(dir, filepath.FromSlash(rel))
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			writeTestPlayInput(t, path, append(original, []byte("\n# outside\n")...))
			before := tree(t, dir)
			if _, err := s.SaveIntoMod(); err == nil || !strings.Contains(err.Error(), "changed outside the editor") {
				t.Fatalf("changed target: %v", err)
			}
			if !reflect.DeepEqual(before, tree(t, dir)) {
				t.Fatal("refusal wrote a file")
			}
		})
	}
}

func TestProjectSaveDoesNotRecordTheAuthorsTree(t *testing.T) {
	for _, count := range []int{1300, 4200} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			s, dir := modSpeakerProject(t)
			for n := 0; n < count; n++ {
				prefix := filepath.Join(".git", "objects")
				if count > 4096 {
					prefix = "author-notes"
				}
				writeTestPlayInput(t, filepath.Join(dir, prefix, fmt.Sprintf("%040x", n)), []byte("synthetic object"))
			}
			reopenProject(t, s, dir)
			before := tree(t, dir)
			if err := s.model.SetDescription("many author files"); err != nil {
				t.Fatal(err)
			}
			if _, err := s.SaveIntoMod(); err != nil {
				t.Fatal(err)
			}
			after := tree(t, dir)
			if len(after) != len(before) || s.inspector.mod.owned {
				t.Fatal("handwritten mod became owned")
			}
			for name, data := range before {
				if name != "mission/en/map.alm" && after[name] != data {
					t.Fatalf("changed %s", name)
				}
			}
			if _, err := os.Stat(filepath.Join(dir, editorPackageRecord)); !os.IsNotExist(err) {
				t.Fatalf("handwritten ownership record: %v", err)
			}
		})
	}
}

func TestManifestChoiceOptionsNeverWriteAnUnknownChoiceNumber(t *testing.T) {
	starts := [][]string{{"common"}, {"rom1"}, {"rom1-en"}, {"rom1-ru"}, {"rom2-en"}, {"rom2-ru"}, {"rom1-en", "rom1-ru"}, {"future-base"}}
	for _, start := range starts {
		p := editorPackageSpec{ID: "choice-mod", Title: "Choice mod", Manifest: defaultPackageManifest()}
		p.Manifest.AppliesTo = start
		source, err := p.manifestBytes(nil)
		if err != nil {
			t.Fatal(err)
		}
		s := &mapInspectionEdits{inspector: &MapInspector{}}
		s.workspace = &editorWorkspaceState{Package: p, Project: &editorModDocument{Manifest: source, Files: map[string][]byte{}, Selected: -1}}
		s.workspaceImage = workspaceBytes(s.workspace)
		original := cloneWorkspace(s.workspace)
		choices := 0
		for _, field := range s.manifestFields() {
			for _, choice := range field.Choices {
				choices++
				s.workspace = cloneWorkspace(original)
				s.workspaceImage = workspaceBytes(s.workspace)
				if _, err := s.SetWorkspaceField("package", field.Key, strconv.FormatInt(choice.Value, 10)); err != nil {
					t.Fatal(err)
				}
				out, err := s.workspace.Package.manifestBytes(source)
				if err != nil {
					t.Fatal(err)
				}
				if choice.Value == -1 {
					if !bytes.Equal(source, out) {
						t.Fatalf("unknown source choice changed bytes: %s", out)
					}
					continue
				}
				if bytes.Equal(source, out) {
					continue
				}
				root := t.TempDir()
				dir := filepath.Join(root, p.ID)
				writeTestPlayInput(t, filepath.Join(dir, mod.ManifestName), out)
				entry := mod.Load(dir)
				base := manifestBases[choice.Value]
				if base == "common" || base == "rom1" {
					base = "rom1-en"
				}
				if entry.Err != nil {
					t.Fatal(entry.Err)
				}
				if _, err := modrt.Load([]mod.Entry{entry}, base, nil, modrt.Options{}); err != nil {
					t.Fatalf("%v -> %s %d: %v", start, field.Key, choice.Value, err)
				}
			}
		}
		if choices < len(manifestBases) {
			t.Fatal("choice fields were not walked")
		}
	}
}

func TestManifestUnknownChoiceLeavesTheWholeDraftUnchanged(t *testing.T) {
	for _, list := range [][]string{{"rom1-en", "rom1-ru"}, {"future-base"}} {
		p := editorPackageSpec{ID: "choice-mod", Title: "Choice mod", Manifest: defaultPackageManifest()}
		p.Manifest.AppliesTo = list
		source, err := p.manifestBytes(nil)
		if err != nil {
			t.Fatal(err)
		}
		s := &mapInspectionEdits{inspector: &MapInspector{}}
		s.workspace = &editorWorkspaceState{Package: p, Project: &editorModDocument{Manifest: source, Files: map[string][]byte{}, Selected: -1}}
		s.workspaceImage = workspaceBytes(s.workspace)
		before := workspaceBytes(s.workspace)
		if _, err := s.SetWorkspaceField("package", "applies-to", "-1"); err != nil {
			t.Fatal(err)
		}
		out, err := s.workspace.Package.manifestBytes(source)
		if err != nil || !bytes.Equal(source, out) || !bytes.Equal(before, workspaceBytes(s.workspace)) || s.CanUndo() {
			t.Fatalf("source choice edited the draft: %v", err)
		}
	}
}

func TestUnchangedProjectFilesAreNeverOpened(t *testing.T) {
	baseline := map[string][32]byte{mod.ManifestName: sha256.Sum256([]byte("manifest")), mod.MissionsFile: sha256.Sum256([]byte("missions"))}
	files := map[string][]byte{mod.ManifestName: []byte("manifest"), mod.MissionsFile: []byte("missions"), mod.SpeakersFile: nil}
	changed, err := changedProjectFiles("an absent folder", files, baseline)
	if err != nil || len(changed) != 0 {
		t.Fatalf("unchanged files reached the filesystem: %v %v", changed, err)
	}
}

type refusedProjectPublish struct{ hostfile.OS }

func (refusedProjectPublish) Publish(old, target string) (hostfile.PublishResult, error) {
	return hostfile.PublishResult{}, errors.New("synthetic publish refusal")
}

func TestProjectFilesShareSiblingStagingAndRollback(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(strconv.FormatBool(fail), func(t *testing.T) {
			dir := t.TempDir()
			writeTestPlayInput(t, filepath.Join(dir, "a.txt"), []byte("prior"))
			baseline := map[string][32]byte{"a.txt": sha256.Sum256([]byte("prior"))}
			files := map[string][]byte{"a.txt": []byte("replacement"), "nested/z.txt": []byte("new")}
			set, cleanup, err := editorProjectSet(dir, files, baseline)
			if err != nil {
				t.Fatal(err)
			}
			defer cleanup()
			if len(set.Files) != 2 || set.Dir != "" {
				t.Fatal("the transaction does not stage beside each target")
			}
			for _, row := range set.Files {
				if row.Folder != "" || row.Staging != hostfile.Sibling || row.Overwrite != hostfile.Backup {
					t.Fatal("a changed file bypasses sibling staging and rollback")
				}
			}
			if fail {
				set.Ops = refusedProjectPublish{}
			}
			_, err = hostfile.Commit(set)
			if fail {
				if err == nil || !strings.Contains(err.Error(), "synthetic publish refusal") {
					t.Fatal(err)
				}
				cleanup()
				if got := tree(t, dir); len(got) != 1 || got["a.txt"] != "prior" {
					t.Fatalf("rollback lost the preceding files: %v", got)
				}
				if _, err := os.Stat(filepath.Join(dir, "nested")); !os.IsNotExist(err) {
					t.Fatalf("rollback kept the new empty folder: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			} else if got := tree(t, dir); !reflect.DeepEqual(got, map[string]string{"a.txt": "replacement", "nested/z.txt": "new"}) {
				t.Fatalf("publication: %v", got)
			}
		})
	}
}

func TestMapSaveAsIntoAModUsesProjectAdmission(t *testing.T) {
	for _, broken := range []bool{false, true} {
		t.Run(strconv.FormatBool(broken), func(t *testing.T) {
			s, dir := modSpeakerProject(t)
			if broken {
				writeTestPlayInput(t, filepath.Join(dir, mod.EntryName), []byte("def init(game, settings):\n    fail('synthetic refusal')\n"))
			}
			before := tree(t, dir)
			target := filepath.Join(dir, "mission", "en", "copy.alm")
			_, err := s.SaveAs(target)
			if broken {
				if err == nil || !strings.Contains(err.Error(), "synthetic refusal") || !reflect.DeepEqual(before, tree(t, dir)) {
					t.Fatalf("mod admission did not precede the loose save: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(target)
			if err != nil || !bytes.Equal(data, s.model.Bytes()) {
				t.Fatalf("saved map: %v", err)
			}
			for name, data := range before {
				if tree(t, dir)[name] != data {
					t.Fatalf("Save As changed %s", name)
				}
			}
			if _, err := os.Stat(filepath.Join(dir, editorPackageRecord)); !os.IsNotExist(err) {
				t.Fatalf("Save As took ownership: %v", err)
			}
		})
	}
}

func TestProjectTestPlayCanReplaceItsOwnRetainedFiles(t *testing.T) {
	f, e := workspaceFixture(t)
	dir := writeEditorProjectFixture(t, e)
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	for range 2 {
		if _, err := s.TestPlay(); err != nil {
			t.Fatal(err)
		}
	}
	private := filepath.Join(f.profile, TestPlayModID, "ambush")
	if tree(t, private)[".author"] == "" || len(f.started) != 2 {
		t.Fatal("repeated test play lost the retained files")
	}
	writeTestPlayInput(t, filepath.Join(private, "foreign.txt"), []byte("not copied or generated"))
	before := tree(t, private)
	if _, err := s.TestPlay(); err == nil || !strings.Contains(err.Error(), "not a recorded test play file") || len(f.started) != 2 || !reflect.DeepEqual(before, tree(t, private)) {
		t.Fatalf("unrecorded destination: %v", err)
	}
}
