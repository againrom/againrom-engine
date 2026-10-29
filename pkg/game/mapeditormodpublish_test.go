package game

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
)

func publishEditorFixture(t *testing.T, e *ui.MapEditor, out string) string {
	t.Helper()
	if err := e.Key("package"); err != nil {
		t.Fatal(err)
	}
	if !e.FocusModField("output") {
		t.Fatal("missing Publish folder")
	}
	_ = e.Key("clear")
	e.TextInput(out)
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if !e.FocusModField("publish") {
		t.Fatal("missing Publish")
	}
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	file := filepath.Join(out, mod.PackedName(s.workspace.Package.ID))
	if st, err := os.Stat(file); err != nil || !st.Mode().IsRegular() {
		t.Fatalf("Publish output: %v", err)
	}
	return file
}

func TestEditorPublishUsesThePackerAndRequiresSaveAndReplaceSeparately(t *testing.T) {
	f, e := workspaceFixture(t)
	source := writeEditorProjectFixture(t, e)
	if err := workspaceField(e, "package", 0, source); err != nil {
		t.Fatal(err)
	}
	working := filepath.Join(t.TempDir(), "ambush")
	if err := workspaceField(e, "package", 0, working); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	out := t.TempDir()
	file := publishEditorFixture(t, e, out)
	first, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	_, packed, err := mod.Pack(working)
	if err != nil || !bytes.Equal(first, packed) {
		t.Fatal("Publish differs from the one packer", err)
	}
	entries, err := mod.Resolve(out, []string{"ambush"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := modrt.Load(entries, BaseID(InspectInstall(f.root)), nil, modrt.Options{})
	if err != nil || len(r.Missions.Missions) != 2 || len(r.Characters.Rows) != 1 {
		t.Fatal("published mod lost declarations", err)
	}
	before := tree(t, working)
	if _, err := s.SetWorkspaceField("package", "gold", "73"); err != nil {
		t.Fatal(err)
	}
	_, err = s.SetWorkspaceField("package", "publish", "")
	var ask *ui.WorkspaceConfirmation
	if !errors.As(err, &ask) || ask.Action != "publish-save" || !strings.Contains(ask.Confirm, "Save") {
		t.Fatal("missing explicit Save and publish", err)
	}
	if !reflect.DeepEqual(before, tree(t, working)) {
		t.Fatal("Publish silently saved")
	}
	_, err = s.SetWorkspaceField("package", "publish-save", "confirmed")
	if !errors.As(err, &ask) || ask.Action != "publish-replace" || s.Dirty() {
		t.Fatal("Save and publish did not ask Replace after checkpoint", err)
	}
	if got, _ := os.ReadFile(file); !bytes.Equal(first, got) {
		t.Fatal("Publish silently replaced RES")
	}
	if _, err := s.SetWorkspaceField("package", "publish-replace", "confirmed"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(file); bytes.Equal(first, got) {
		t.Fatal("Replace did not publish the saved change")
	}
	if !strings.Contains(s.SaveNote(), file) || !strings.Contains(s.SaveNote(), "(2)") || s.inspector.modOutput != out {
		t.Fatal("missing output receipt", s.SaveNote())
	}
}

func TestEditorPublishRefusesBrokenOrAmbiguousInputsWithoutReplacingRES(t *testing.T) {
	for _, kind := range []string{"broken-chain", "missing-map", "duplicate-id", "install-output", "script-omission"} {
		t.Run(kind, func(t *testing.T) {
			f, e := workspaceFixture(t)
			source := writeEditorProjectFixture(t, e)
			if err := workspaceField(e, "package", 0, source); err != nil {
				t.Fatal(err)
			}
			working := filepath.Join(t.TempDir(), "ambush")
			if err := workspaceField(e, "package", 0, working); err != nil {
				t.Fatal(err)
			}
			s := e.Document().Edits.(*mapInspectionEdits)
			out := t.TempDir()
			file := publishEditorFixture(t, e, out)
			before, _ := os.ReadFile(file)
			switch kind {
			case "broken-chain":
				_, _ = s.SetWorkspaceField("package", "after", "26")
			case "missing-map":
				data, err := os.ReadFile(filepath.Join(working, mod.MissionsFile))
				if err != nil {
					t.Fatal(err)
				}
				writeTestPlayInput(t, filepath.Join(working, mod.MissionsFile), bytes.ReplaceAll(data, []byte("second.alm"), []byte("missing.alm")))
				if err := workspaceField(e, "package", 0, working); err != nil {
					t.Fatal(err)
				}
				s = e.Document().Edits.(*mapInspectionEdits)
			case "duplicate-id":
				if err := os.Mkdir(filepath.Join(out, "ambush"), 0o755); err != nil {
					t.Fatal(err)
				}
			case "install-output":
				s.inspector.modOutput = f.root
			case "script-omission":
				writeTestPlayInput(t, filepath.Join(working, "main.star"), []byte("def init(game, settings):\n    pass\n"))
				if err := workspaceField(e, "package", 0, working); err != nil {
					t.Fatal(err)
				}
				s = e.Document().Edits.(*mapInspectionEdits)
				_, _ = s.SetWorkspaceField("package", "gold", "1")
			}
			_, err := s.SetWorkspaceField("package", "publish-save", "confirmed")
			var ask *ui.WorkspaceConfirmation
			if err == nil || errors.As(err, &ask) {
				t.Fatal("invalid Publish reached replacement confirmation", err)
			}
			if got, _ := os.ReadFile(file); !bytes.Equal(got, before) {
				t.Fatal("refusal changed published RES")
			}
		})
	}
}

func TestEditorUnsupportedScriptContextIsShownAndCannotBeCopied(t *testing.T) {
	_, e := workspaceFixture(t)
	source := writeEditorProjectFixture(t, e)
	writeTestPlayInput(t, filepath.Join(source, "main.star"), []byte("def init(game, settings):\n    pass\n"))
	before := tree(t, source)
	if err := workspaceField(e, "package", 0, source); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	if !strings.Contains(s.ModSummary(), "Read-only context:") {
		t.Fatal("unsupported context is not visible", s.ModSummary())
	}
	parent := t.TempDir()
	if _, err := s.saveWorkspacePackage(filepath.Join(parent, "ambush")); err == nil {
		t.Fatal("unsupported context copied with an omitted mission")
	}
	if len(tree(t, parent)) != 0 || !reflect.DeepEqual(before, tree(t, source)) {
		t.Fatal("unsupported context changed source or output")
	}
}
