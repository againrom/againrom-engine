package game

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
)

func (s *mapInspectionEdits) publishEditorMod(action, value string) (err error) {
	x, saved := s.inspector, false
	defer func() {
		var confirmation *ui.WorkspaceConfirmation
		if err != nil && !errors.As(err, &confirmation) && x.mod != nil && x.mod.owned && (saved || !s.Dirty()) {
			err = fmt.Errorf("%s %s: %w", x.word("editor.mod.savecomplete"), x.word("editor.mod.publishfailed"), err)
		}
	}()
	if x.mod == nil || !x.mod.owned || x.mod.copyOnly {
		return errors.New("choose a working mod copy with W before publishing")
	}
	if x.modOutput == "" {
		return errors.New("choose an explicit Publish folder")
	}
	out, err := editorPhysicalDirectory(x.modOutput)
	if err != nil {
		return err
	}
	if err := fenceEditorWrite(x.archives.Root, out, "Publish"); err != nil {
		return err
	}
	if inside, err := sameOrDescendantPhysical(x.mod.dir, out); err != nil {
		return err
	} else if inside {
		return errors.New("the Publish folder must be outside the working mod")
	}
	if _, err := os.Lstat(filepath.Join(out, x.mod.id)); err == nil {
		return fmt.Errorf("%s already contains a folder with mod ID %s; choose another Publish folder", out, x.mod.id)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := s.unchangedModSource(); err != nil {
		return err
	}
	if s.workspace.Project == nil || s.workspace.Project.Selected < 0 {
		return errors.New("add the open map to the mod before publishing")
	}
	if s.Dirty() {
		if action != "publish-save" || value != "confirmed" {
			return &ui.WorkspaceConfirmation{Action: "publish-save", Message: x.word("editor.mod.asksave"), Confirm: x.word("editor.mod.savepublish")}
		}
		if _, err := s.saveWorkspacePackage(x.mod.dir); err != nil {
			return err
		}
		saved = true
	}
	entries, err := mod.Resolve(filepath.Dir(x.mod.dir), []string{x.mod.id})
	if err != nil {
		return err
	}
	result, err := modrt.Load(entries, BaseID(InspectInstall(x.archives.Root)), nil, modrt.Options{})
	if err != nil {
		return err
	}
	if err := checkMissionProjection(s.workspace.Project.Missions, result.Missions); err != nil {
		return err
	}
	if err := x.admitMissionPackage(result); err != nil {
		return err
	}
	target := filepath.Join(out, mod.PackedName(x.mod.id))
	replace := action == "publish-replace" && value == "confirmed"
	if _, err := os.Lstat(target); err == nil && !replace {
		message := x.word("editor.mod.askreplace") + " " + target
		if saved {
			message = x.word("editor.mod.saved") + " " + message
		}
		return &ui.WorkspaceConfirmation{Action: "publish-replace", Message: message, Confirm: x.word("editor.mod.replacefile")}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	file, err := PackMod(x.mod.dir, out, replace)
	if err != nil {
		return err
	}
	s.saveNote = fmt.Sprintf("%s %s (%d)", x.word("editor.mod.published"), file, len(result.Missions.Missions))
	return nil
}
