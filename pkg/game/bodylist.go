package game

import "againrom/pkg/data"

// HeroPictureAddress is the address the shipped body list is read from:
// mainPrefix — the same prefix EventTextPath composes from — and the
// payload's own name under main.res's text tree (research HERO-APPEAR-052,
// High for the load).
//
// It is spelt beside mainPrefix's own use in eventtext.go for the reason
// every address in this package is: one spelling, so a renamed container or
// a renamed payload moves the one place that names it.
const HeroPictureAddress = mainPrefix + "text/heropicture.txt"

// ReadBodyList reads the shipped body list off src and reports whether it is
// there — ReadEventText's EXACT SIGNATURE AND EXACT CONTRACT, one file
// over (the read half).
//
// THERE IS NO ERROR RETURN, for ReadEventText's own reason: a front end that
// cannot read the list still opens missions.
func ReadBodyList(src entrySource) (data.BodyList, bool) {
	if src == nil {
		return nil, false
	}
	b, err := src.ReadFile(HeroPictureAddress)
	if err != nil {
		return nil, false
	}
	return data.ParseBodyList(b), true
}
