package main

import "againrom/pkg/pathdialog"

// folderChoice is one folder the starter asks the path window for: the field
// it fills, the window's title and start, the failure line and the answer.
type folderChoice struct {
	field         fieldID
	title, failed captionID
	start         func(a *app) string
	chosen        func(a *app, path string)
}

var (
	baseFolderChoice = folderChoice{field: fAddBase, title: cChooseBaseFolder, failed: cFolderDialogFailed, start: (*app).baseFolderStart, chosen: (*app).baseFolderChosen}
	modsFolderChoice = folderChoice{field: fModsDir, title: cChooseModsFolder, failed: cModsDialogFailed, start: (*app).modsFolderStart, chosen: (*app).modsFolderChosen}
)

func (a *app) baseFolderStart() string {
	if start, ok := a.selectedRoot(); ok {
		return start
	}
	return a.selfDir()
}

func (a *app) modsFolderStart() string { return a.s.modsDir(a.selfDir()) }

func (a *app) baseFolderChosen(path string) {
	a.focus = fNone
	a.addText = path
	a.addBase()
}

func (a *app) modsFolderChosen(path string) {
	a.focus = fNone
	a.s.ModsDir = path
	a.rescan()
	a.ok(cModsFound, len(a.scan), a.s.modsDir(a.selfDir()))
}

type folderDialogResult struct {
	choice folderChoice
	result pathdialog.Result
	err    error
}

func (a *app) chooseBase() { a.chooseFolder(baseFolderChoice) }

// folderQuestion is the folder window c asks, titled in the current language.
func (a *app) folderQuestion(c folderChoice) pathdialog.Question {
	return pathdialog.Question{Kind: pathdialog.ChooseFolder, Title: a.t(c.title)}
}

func (a *app) chooseFolder(c folderChoice) {
	if a.folderDialogResults != nil {
		return
	}
	req := a.folderQuestion(c).Request(c.start(a))
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
		default:
			r.choice.chosen(a, r.result.Path)
		}
	default:
	}
	return true
}

func (a *app) folderDialogFailed(c folderChoice, err error) {
	a.focus = c.field
	a.fail(c.failed, err)
}
