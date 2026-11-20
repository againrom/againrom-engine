package ui

import "image"

// replyPress is the reply row a primary press began on and the pointer
// position last seen, so a stationary pointer does not take the light back
// from the keys.
type replyPress struct {
	row     int
	pointer image.Point
}

// replyDigit is the reply number a digit key chose this tick, or 0.
func replyDigit(in appInput, rows int) int {
	if !in.GroupKey || in.Viewer.Ctrl || in.Viewer.Alt || in.Viewer.Shift {
		return 0
	}
	if in.GroupDigit >= 1 && in.GroupDigit <= rows {
		return in.GroupDigit
	}
	return 0
}

// stepDialogueReplies drives a dialogue page that shows replies and answers
// the reply chosen this tick, or 0. Keys 1 to N choose; Up and Down move the
// light and Enter chooses the lit reply; a primary press and release on the
// same row chooses it. Escape is not read here: it does nothing while the
// rows stand. Every pointer edge is consumed, as the one button's are.
func (a *App) stepDialogueReplies(in *appInput) int {
	v := a.flow.viewer
	rows := v.NoticeReplyCount()
	lit := v.noticeReplyLit
	at := image.Pt(in.CursorX, in.CursorY)
	row := v.noticeReplyAt(in.CursorX, in.CursorY)
	if v.NoticeReplyOff(row) {
		row = 0
	}
	if at != a.reply.pointer {
		a.reply.pointer = at
		if row > 0 {
			lit = row
		}
	}
	chosen := 0
	switch {
	case in.Close || in.Unfocused:
		a.reply.row = 0
	case replyDigit(*in, rows) > 0:
		if d := replyDigit(*in, rows); !v.NoticeReplyOff(d) {
			chosen = d
		}
	case in.Up != in.Down:
		for range rows {
			switch {
			case lit == 0 && in.Up:
				lit = rows
			case lit == 0:
				lit = 1
			case in.Up:
				lit = (lit-2+rows)%rows + 1
			default:
				lit = lit%rows + 1
			}
			if !v.NoticeReplyOff(lit) {
				break
			}
		}
	case in.Enter && lit > 0:
		chosen = lit
	}
	if in.PrimaryPressed && row > 0 {
		a.reply.row, lit = row, row
	}
	if in.PrimaryReleased {
		if chosen == 0 && row > 0 && row == a.reply.row {
			chosen = row
		}
		a.reply.row = 0
	}
	in.PrimaryPressed, in.PrimaryReleased = false, false
	if chosen > 0 {
		a.reply.row = 0
		return chosen
	}
	v.SetDialogueReplyLit(lit)
	if row > 0 && row == a.reply.row {
		v.SetDialogueReplyHeld(row)
	} else {
		v.SetDialogueReplyHeld(0)
	}
	return 0
}
