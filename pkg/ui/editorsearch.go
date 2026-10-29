package ui

func (e *MapEditor) editorSearchText(value string) string {
	if e.hostDecode != nil {
		return e.hostDecode(value)
	}
	return value
}
