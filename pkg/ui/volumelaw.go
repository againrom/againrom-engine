package ui

import "math"

// The Sound Options sliders map a position to a volume by a square law
// (MENU-076): volume = trunc(-max * ((position - max) / max)^2) in hundredths of
// a decibel of attenuation, 0 at the right end. The channel preferences hold a
// linear percentage, so the law is applied as an amplitude: percent =
// 100 * 10^(volume / 2000).
const soundSliderRange = 5000

// soundSliderVolume is the attenuation, in hundredths of a decibel, of a
// slider position of the given range.
func soundSliderVolume(position, max int) int {
	d := float64(position-max) / float64(max)
	return int(-float64(max) * d * d)
}

// soundSliderPositionOf is the inverse used when the dialog is built:
// max - trunc(max * sqrt(-volume / max)).
func soundSliderPositionOf(volume, max int) int {
	if volume >= 0 {
		return max
	}
	return max - int(float64(max)*math.Sqrt(float64(-volume)/float64(max)))
}

// soundSliderPercent is the channel percentage a position selects.
func soundSliderPercent(position int) int {
	position = min(max(position, 0), soundSliderRange)
	volume := soundSliderVolume(position, soundSliderRange)
	return int(math.Round(100 * math.Pow(10, float64(volume)/2000)))
}

// soundPercentSlider is the position that shows a stored percentage.
func soundPercentSlider(percent int) int {
	if percent <= 0 {
		return 0
	}
	if percent >= 100 {
		return soundSliderRange
	}
	volume := int(math.Round(2000 * math.Log10(float64(percent)/100)))
	return soundSliderPositionOf(max(volume, -soundSliderRange), soundSliderRange)
}
