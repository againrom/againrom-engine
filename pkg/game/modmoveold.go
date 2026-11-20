package game

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"againrom/pkg/mod"
	"againrom/pkg/namefold"
	"againrom/pkg/ui"
)

// modMoveOldKey is the key of the mod window's button that carries the files
// rows name on old paths into the tree's places.
const modMoveOldKey = "moveold"

// oldPathPlan is a move of a mod's old-path files: the files copied to their
// places, the old files removed after, the row files made new whole, and the
// row values that follow in the data files.
type oldPathPlan struct {
	moves   []mod.FileMove
	removes []string
	whole   map[string][]byte
	values  map[string][]mod.ValueEdit
	// held is the targets that already hold the same bytes: they are not
	// copied again.
	held map[string]bool
}

func (p oldPathPlan) empty() bool {
	return len(p.moves) == 0 && len(p.whole) == 0 && len(p.values) == 0
}

// planOldPaths is the plan for the rows of the open mod that name a file on
// an old path. A data row's file takes the import's place and name
// (modAssetBytes) and the row's value is rewritten by mod.RewriteValues. A
// mission row's map and pages take the place rule's (mod.TreeLayout), its
// rows written through the missions kind (writeHeld), and a speaker bound to a moved map
// follows (mod.MoveSpeakerMaps).
func (s *mapInspectionEdits) planOldPaths(tree mod.Tree, old []mod.OldPath) (oldPathPlan, error) {
	x := s.inspector
	plan := oldPathPlan{whole: map[string][]byte{}, values: map[string][]mod.ValueEdit{}, held: map[string]bool{}}
	for _, o := range old {
		data, _, err := tree.ReadInside(o.Rel)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return plan, err
		}
		to, err := s.modAssetBytes(o.Folder, modAssetName(o.Rel), data)
		if err != nil {
			return plan, err
		}
		if move := (mod.FileMove{From: o.Rel, To: to}); !slices.Contains(plan.moves, move) {
			plan.moves = append(plan.moves, move)
			plan.removes = append(plan.removes, o.Rel)
		}
		plan.values[o.File] = append(plan.values[o.File], mod.ValueEdit{Line: o.Line, Value: to})
	}
	layout := mod.TreeLayout(x.mod.missions.Missions, x.mod.files, nil)
	if len(layout.Places) != 0 {
		rows, err := writeHeld(mod.MissionsFile, x.mod.source, mod.Data{
			Missions: mod.MissionData{Missions: layout.Missions},
			Before:   mod.DataBefore{Missions: x.mod.missions.Missions},
		})
		if err != nil {
			return plan, err
		}
		plan.whole[mod.MissionsFile] = rows
		plan.moves = append(plan.moves, layout.Moves...)
		plan.removes = append(plan.removes, layout.Removed...)
		if src, err := x.mod.speakersFile(); err != nil {
			return plan, err
		} else if src != nil {
			if b, moved, err := mod.MoveSpeakerMaps(src, layout.Moves); err != nil {
				return plan, err
			} else if moved {
				plan.whole[mod.SpeakersFile] = b
			}
		}
	}
	spelled := map[string]string{}
	err := tree.WalkDir(func(rel string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			spelled[namefold.ModTree(rel)] = rel
		}
		return err
	})
	if err != nil {
		return plan, err
	}
	target := map[string]mod.FileMove{}
	for _, m := range plan.moves {
		key := namefold.ModTree(m.To)
		if other, shared := target[key]; shared && other.From != m.From {
			return plan, fmt.Errorf(x.word("editor.mod.err.moveclash"), m.From, other.From, m.To)
		}
		target[key] = m
		if there, exists := spelled[key]; exists {
			want, _, err := tree.ReadInside(m.From)
			if err != nil {
				return plan, err
			}
			have, _, err := tree.ReadInside(there)
			if err != nil {
				return plan, err
			}
			if !bytes.Equal(want, have) {
				return plan, fmt.Errorf(x.word("editor.mod.err.moveclash"), m.From, there, m.To)
			}
			plan.held[m.To] = true
		}
	}
	return plan, nil
}

// moveOldPaths carries every file a row names on an old path to its place
// under the tree and rewrites the row. The mod's folder is changed, not a
// draft, so edits not yet saved are refused first. The copies, the rewritten
// row files and the removals of the old files are one file-set transaction
// (installMissionPackage), so a failure puts every file back. A target that
// holds a different file is never written over: the move refuses, naming both
// paths, and changes nothing.
func (s *mapInspectionEdits) moveOldPaths() (*ui.InspectionDocument, error) {
	x := s.inspector
	s.saveNote = ""
	if err := x.modWritable(); err != nil {
		return nil, err
	}
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	if s.Dirty() {
		return nil, errors.New(x.word("editor.mod.err.movedirty"))
	}
	dir := x.mod.dir
	tree, err := mod.OpenTree(dir)
	if err != nil {
		return nil, err
	}
	described, err := tree.Describe()
	if err != nil {
		return nil, err
	}
	plan, err := s.planOldPaths(tree, described.OldPaths)
	if err != nil {
		return nil, err
	}
	if plan.empty() {
		s.saveNote = x.word("editor.mod.nomove")
		return nil, nil
	}
	root := ""
	if x.archives != nil {
		root = x.archives.Root
	}
	physical, err := editorPhysicalDirectory(dir)
	if err != nil {
		return nil, err
	}
	if err := fenceEditorWrite(root, physical, "Move mod files", x.modWriteGrants()...); err != nil {
		return nil, err
	}
	if err := applyOldPaths(dir, x.mod.id, tree, plan); err != nil {
		return nil, err
	}
	if err := s.reloadMoved(dir, plan); err != nil {
		return nil, err
	}
	d, err := s.project(s.model)
	if err != nil {
		return nil, err
	}
	s.saveNote = fmt.Sprintf(x.word("editor.mod.moved"), len(plan.moves))
	return d, nil
}

// reloadMoved makes the editor hold the mod's folder as the move left it, as
// a save does: the mod read again, the project its rows and files, the
// open maps' paths the moved ones, and the state clean. The undo history,
// which names the old rows, ends here.
func (s *mapInspectionEdits) reloadMoved(dir string, plan oldPathPlan) error {
	x := s.inspector
	selected := x.mod.selected
	loose := ""
	if s.workspace.Project != nil {
		loose = s.workspace.Project.Loose
	}
	if err := x.SetMod(dir); err != nil {
		return err
	}
	s.joinMod()
	x.mod.selected = selected
	s.workspace.Project = x.mod.document()
	s.workspace.Project.Selected, s.workspace.Project.Loose = -1, loose
	for i, entry := range x.mod.missions.Missions {
		if entry.RowID() == selected {
			s.workspace.Project.Selected = i
			s.workspace.Package = modPackageSpec(entry, s.workspace.Package)
			break
		}
	}
	moved := func(source string) string {
		for _, m := range plan.moves {
			if filepath.Join(dir, filepath.FromSlash(m.From)) == source {
				return filepath.Join(dir, filepath.FromSlash(m.To))
			}
		}
		return source
	}
	s.source = moved(s.source)
	for _, t := range x.openMaps {
		t.source = moved(t.source)
	}
	s.undoWorkspace, s.redoWorkspace = nil, nil
	s.workspaceImage = workspaceBytes(s.workspace)
	s.savedWorkspace = s.workspaceImage
	if s.model != nil {
		s.saved, s.savedInside = s.model.Bytes(), s.TextsInside()
	}
	return nil
}

// applyOldPaths writes plan into the mod folder dir.
func applyOldPaths(dir, id string, tree mod.Tree, plan oldPathPlan) error {
	changed := map[string][]byte{}
	for _, m := range plan.moves {
		if plan.held[m.To] {
			continue
		}
		data, _, err := tree.ReadInside(m.From)
		if err != nil {
			return err
		}
		changed[m.To] = data
	}
	files := slices.Sorted(func(yield func(string) bool) {
		for f := range plan.whole {
			if !yield(f) {
				return
			}
		}
		for f := range plan.values {
			if _, both := plan.whole[f]; !both && !yield(f) {
				return
			}
		}
	})
	for _, file := range files {
		source, _, err := tree.ReadInside(file)
		if err != nil {
			return err
		}
		out := source
		if whole, ok := plan.whole[file]; ok {
			out = whole
		}
		if edits := plan.values[file]; len(edits) != 0 {
			if out, err = mod.RewriteValues(out, edits); err != nil {
				return fmt.Errorf("%s: %w", file, err)
			}
		}
		changed[file] = out
	}
	for _, rel := range plan.removes {
		changed[rel] = nil
	}
	return installMissionPackage("", dir, id, false, missionPackagePlan{InPlace: true, Changed: changed})
}
