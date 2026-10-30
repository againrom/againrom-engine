package ui

import "time"

// showSaveRefusal puts the save dialog admission's reason for a closed save
// point on the map's message line, else the screen's message, and reports
// whether there was one.
func (f *flow) showSaveRefusal(onMap bool) bool {
	if f.saveDialogSeams.Prepare == nil {
		return false
	}
	_, err := f.saveDialogSeams.Prepare(SaveRequest{OnMap: onMap, Admission: true})
	if err == nil {
		return false
	}
	message := f.menuText(err.Error())
	if f.screen == ScreenMap && f.viewer != nil {
		f.viewer.PostMessage(message, MessageWhite, 15*time.Second)
	} else {
		f.msg, f.saveRefused = message, message
	}
	return true
}
