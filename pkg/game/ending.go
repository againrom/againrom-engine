package game

import (
	"errors"
	"strings"

	"againrom/pkg/formats/fame"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// TerminalMission is the installed flag plus the original main-mission gate,
// not the absence of a successor (REG-SCN-063, SAV-890).
func (c Campaign) TerminalMission(n int) bool {
	return n > 0 && n%10 == 0 && c.Last[n]
}

// A finished campaign is neither a city nor a mission, so no SAV holds it.
var (
	errCompletedCampaignFile = errors.New("this save holds a completed campaign; the game has no save point after the campaign ends, so it cannot be loaded")
	errCompletedCampaignSave = errors.New("the campaign is complete; there is no save point after its ending")
)

func (c Campaign) completedBy(t *Town) bool {
	if t == nil {
		return false
	}
	for _, n := range c.Main {
		if c.TerminalMission(n) && t.Done(n) {
			return true
		}
	}
	return false
}

func (f *FrontEnd) completedCampaign() bool {
	return f.Campaign.Value().completedBy(f.Town)
}

// Installed art is shared by the campaign ending and the read-only hall view.
func (f *FrontEnd) endingArt() *ui.EndingView {
	if f.endingAssets == nil {
		f.endingAssets = &ui.EndingView{}
		view := f.endingAssets
		view.HallFont = f.documentFont()
		if f.Archives != nil && f.Archives.Containers != nil {
			if raw, err := f.Archives.Containers.ReadFile(mainPrefix + "text/credits.txt"); err == nil {
				view.Credits = strings.Split(strings.ReplaceAll(strings.TrimSpace(vfs.DecodeText(raw)), "\r\n", "\n"), "\n")
			}
			if raw, err := f.Archives.Containers.ReadFile(mainPrefix + "graphics/famehall/hall.bmp"); err == nil {
				if art, err := chargenRGBA(raw, "hall.bmp"); err == nil {
					view.HallBackground = art
				}
			}
		}
	}
	return f.endingAssets
}

func (f *FrontEnd) hallOfFame() ui.EndingView {
	view := *f.endingArt()
	store := f.hallStore
	if store.OriginalDir == "" && f.Archives != nil {
		store.OriginalDir = f.Archives.Root
	}
	if records, err := store.read(); err == nil {
		view.HallAvailable = true
		for _, row := range records {
			view.Hall = append(view.Hall, ui.EndingHallRow{Name: vfs.DecodeText([]byte(row.Name)), Score: row.Score})
		}
	}
	return view
}

// A captured result remains pending until the profile table has been committed.
func (f *FrontEnd) campaignEnding() (ui.EndingView, bool) {
	if !f.completedCampaign() {
		return ui.EndingView{}, false
	}
	store := f.hallStore
	if store.OriginalDir == "" && f.Archives != nil {
		store.OriginalDir = f.Archives.Root
	}
	if result := f.fame.Result; result != nil {
		if !result.Recorded && store.Dir != "" {
			if _, err := store.add(fame.Record{Name: result.Name, Score: result.Score}); err == nil {
				result.Recorded = true
			}
		}
	}
	view := f.hallOfFame()
	if result := f.fame.Result; result != nil {
		view.HasScore, view.Score, view.Recorded = true, result.Score, result.Recorded
	}
	for _, member := range f.Carried {
		if member.PlayerCharacter {
			view.Hero = member.Name
			break
		}
	}
	view.Gold = f.Town.Gold()
	return view, true
}

// leaveCampaignEnding is the terminal route's campaign reset (FAME-029). A
// result the hall store has not accepted keeps the completed campaign so the
// hall can retry it.
func (f *FrontEnd) leaveCampaignEnding() {
	if result := f.fame.Result; result != nil && !result.Recorded {
		return
	}
	f.resetSessionForNewGame()
}
