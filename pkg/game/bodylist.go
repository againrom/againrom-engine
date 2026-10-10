package game

import (
	"fmt"
	"strconv"
	"strings"

	"againrom/pkg/data"
)

// HeroPictureAddress is the address the shipped body list is read from:
// mainPrefix — the same prefix EventTextPath composes from — and the
// payload's own name under main.res's text tree (research HERO-APPEAR-052,
// High for the load).
//
// It is spelt beside mainPrefix's own use in eventtext.go for the reason
// every address in this package is: one spelling, so a renamed container or
// a renamed payload moves the one place that names it.
const HeroPictureAddress = mainPrefix + "text/heropicture.txt"

// ModBodyChoiceAddress is where a front end running mods serves what the mods
// choose about bodies, on its own forked filesystem (SetModBodies). No install
// holds it, so an install without mods reads the shipped list alone.
const ModBodyChoiceAddress = "againrom/mod-bodies.txt"

// ReadBodyList reads the shipped body list off src and reports whether it is
// there — ReadEventText's EXACT SIGNATURE AND EXACT CONTRACT, one file
// over (the read half). The mods' choices, when src serves them, ride on the
// list (data.BodyList.WithWeaponBody, WithModBody).
//
// THERE IS NO ERROR RETURN, for ReadEventText's own reason: a front end that
// cannot read the list still opens missions.
func ReadBodyList(src entrySource) (data.BodyList, bool) {
	if src == nil {
		return data.BodyList{}, false
	}
	b, err := src.ReadFile(HeroPictureAddress)
	if err != nil {
		return data.BodyList{}, false
	}
	list := data.ParseBodyList(b)
	if choice, err := src.ReadFile(ModBodyChoiceAddress); err == nil {
		list = applyBodyChoice(list, choice)
	}
	return list, true
}

// encodeBodyChoice writes the mods' choices on list as the lines
// "weapon <row> <body>" and "body <name> <weapon-last 0|1>".
func encodeBodyChoice(list data.BodyList) []byte {
	var b strings.Builder
	for _, row := range list.WeaponRows() {
		body, _ := list.WeaponBody(row)
		fmt.Fprintf(&b, "weapon %d %s\n", row, body)
	}
	for _, name := range list.ModBodies() {
		m, _ := list.ModBodyOf(name)
		last := 0
		if m.WeaponLast {
			last = 1
		}
		fmt.Fprintf(&b, "body %s %d\n", name, last)
	}
	for _, name := range list.ShieldlessBodies() {
		fmt.Fprintf(&b, "bare %s 1\n", name)
	}
	return []byte(b.String())
}

// applyBodyChoice is encodeBodyChoice read back onto list. The text is this
// front end's own; a line it cannot read is skipped.
func applyBodyChoice(list data.BodyList, text []byte) data.BodyList {
	for _, line := range strings.Split(string(text), "\n") {
		f := strings.Fields(line)
		if len(f) != 3 {
			continue
		}
		switch f[0] {
		case "weapon":
			if row, err := strconv.Atoi(f[1]); err == nil {
				list = list.WithWeaponBody(row, data.HeroBody(f[2]))
			}
		case "bare":
			list = list.WithoutShieldForm(data.HeroBody(f[1]))
		case "body":
			list = list.WithModBody(data.HeroBody(f[1]), data.ModBody{WeaponLast: f[2] == "1"})
		}
	}
	return list
}
