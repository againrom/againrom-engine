package main

import "againrom/pkg/pathdialog"

// folderChoice is one folder the starter asks the path window for: the field
// the folder fills, the window's title and the line a failed window shows.
type folderChoice struct {
	field         fieldID
	title, failed captionID
}

var (
	baseFolderChoice = folderChoice{field: fAddBase, title: cChooseBaseFolder, failed: cFolderDialogFailed}
	modsFolderChoice = folderChoice{field: fModsDir, title: cChooseModsFolder, failed: cModsDialogFailed}
)

type folderDialogResult struct {
	choice folderChoice
	result pathdialog.Result
	err    error
}

func (a *app) chooseBase() { a.chooseFolder(baseFolderChoice) }

func (a *app) chooseFolder(c folderChoice) {
	if a.folderDialogResults != nil {
		return
	}
	start, ok := a.selectedRoot()
	if !ok {
		start = a.selfDir()
	}
	if c.field == fModsDir {
		start = a.s.modsDir(a.selfDir())
	}
	req := pathdialog.Request{Kind: pathdialog.ChooseFolder, Title: a.t(c.title), StartDir: start}
	dialog := a.deps.pathDialog
	if dialog == nil {
		a.folderDialogFailed(c, pathdialog.ErrUnavailable)
		return
	}
	results := make(chan folderDialogResult, 1)
	a.folderDialogResults = results
	a.deps.async(func() {
		result, err := dialog(req)
		results <- folderDialogResult{choice: c, result: result, err: err}
	})
}

func (a *app) pollFolderDialog() bool {
	if a.folderDialogResults == nil {
		return false
	}
	select {
	case r := <-a.folderDialogResults:
		a.folderDialogResults = nil
		switch {
		case r.err != nil:
			a.folderDialogFailed(r.choice, r.err)
		case r.result.Cancelled:
		case r.choice.field == fModsDir:
			a.focus = fNone
			a.s.ModsDir = r.result.Path
			a.rescan()
			a.ok(cModsFound, len(a.scan), a.s.modsDir(a.selfDir()))
		default:
			a.focus = fNone
			a.addText = r.result.Path
			a.addBase()
		}
	default:
	}
	return true
}

func (a *app) folderDialogFailed(c folderChoice, err error) {
	a.focus = c.field
	a.fail(c.failed, err)
}
