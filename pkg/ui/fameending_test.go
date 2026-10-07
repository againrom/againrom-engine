package ui

import (
	"againrom/pkg/render/text"
	"testing"
)

// The hall entry retries an unrecorded result once, and the page says so
// until it is recorded.
func TestEndingHallEntryRetriesResultWithoutDrawWrites(t *testing.T) {
	for _, selector := range []int{0, text.SelectorConverting} {
		a := endingTestApp(t)
		a.flow.menuFont.Selector = selector
		calls := 0
		a.SetCampaignEnding(func() (EndingView, bool) {
			calls++
			return EndingView{HasScore: true, Score: -23, Recorded: calls > 1}, true
		})
		a.flow.showCampaignEnding()
		if err := a.HeadlessActivate(a.flow.endingWords().Back); err != nil {
			t.Fatal(err)
		}
		pending := func() bool {
			for _, label := range a.endingPaint().texts {
				if label.text == a.flow.endingWords().PendingHall {
					return true
				}
			}
			return false
		}
		before := calls
		if a.flow.endingPage != 2 || !a.flow.ending.Recorded || pending() || calls != before {
			t.Fatal("hall entry did not refresh the result", calls, a.flow.endingPage)
		}
	}
}

func TestEndingHallShowsPendingResultNote(t *testing.T) {
	a := endingTestApp(t)
	a.SetCampaignEnding(func() (EndingView, bool) { return EndingView{HasScore: true, Score: 3}, true })
	a.flow.showCampaignEnding()
	if err := a.HeadlessActivate(a.flow.endingWords().Back); err != nil {
		t.Fatal(err)
	}
	for _, label := range a.endingPaint().texts {
		if label.text == a.flow.endingWords().PendingHall {
			return
		}
	}
	t.Fatal("an unrecorded result left no note on the hall page")
}
