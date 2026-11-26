package game

import (
	"fmt"
	"strconv"

	"againrom/pkg/ui"
)

const tooltipDelayKey = "TooltipDelay"

func (s OptionsStore) TooltipDelay() (int, error) {
	m, err := s.readAll()
	if err != nil {
		return ui.DefaultTooltipDelay, err
	}
	ms, err := strconv.Atoi(m[tooltipDelayKey])
	if err != nil || !ui.ValidTooltipDelay(ms) {
		return ui.DefaultTooltipDelay, nil
	}
	return ms, nil
}

func (s OptionsStore) SetTooltipDelay(ms int) error {
	if !ui.ValidTooltipDelay(ms) {
		return fmt.Errorf("invalid tooltip delay %d", ms)
	}
	return s.update(func(m map[string]string) error {
		m[tooltipDelayKey] = strconv.Itoa(ms)
		return nil
	})
}
