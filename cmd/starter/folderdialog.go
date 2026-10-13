package main

import "againrom/pkg/pathdialog"

type folderDialogResult struct {
	result pathdialog.Result
	err    error
}

func (a *app) chooseBase() {
	if a.folderDialogResults != nil {
		return
	}
	start, ok := a.selectedRoot()
	if !ok {
		start = a.selfDir()
	}
	req := pathdialog.Request{Kind: pathdialog.ChooseFolder, Title: a.t(cChooseBaseFolder), StartDir: start}
	dialog := a.deps.pathDialog
	if dialog == nil {
		a.folderDialogFailed(pathdialog.ErrUnavailable)
		return
	}
	results := make(chan folderDialogResult, 1)
	a.folderDialogResults = results
	a.deps.async(func() {
		result, err := dialog(req)
		results <- folderDialogResult{result: result, err: err}
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
			a.folderDialogFailed(r.err)
		case !r.result.Cancelled:
			a.focus = fNone
			a.addText = r.result.Path
			a.addBase()
		}
	default:
	}
	return true
}

func (a *app) folderDialogFailed(err error) {
	a.focus = fAddBase
	a.fail(cFolderDialogFailed, err)
}
