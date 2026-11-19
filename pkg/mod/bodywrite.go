package mod

import (
	"fmt"
	"strconv"
	"strings"
)

// writeBodies is the body sheets kind's writer: source made to hold
// d.Bodies.Bodies, each [[body]] table kept where it stands with its lines
// but for the keys it changes. A sheet the source does not hold, its Line
// zero, is appended; one d lacks leaves with its table.
func writeBodies(_ string, source []byte, d Data) ([]byte, error) {
	before, err := ParseBodies("", BodiesFile, source)
	if err != nil {
		return nil, err
	}
	after := bodySourceRows(d.Bodies.Bodies)
	out, err := writeRows(source, bodySourceRows(before), after, func(key string) string { return key })
	if err != nil {
		return nil, err
	}
	back, err := ParseBodies("", BodiesFile, out)
	if err != nil {
		return nil, err
	}
	if !sameRows(bodySourceRows(back), after) {
		return nil, fmt.Errorf("%s: the written rows do not read back as the rows given", BodiesFile)
	}
	return out, nil
}

// writeWeaponBodies is the weapon bodies kind's writer over d.Bodies.Weapons,
// as writeBodies is for the sheets; rel is the file's name as the mod holds it.
func writeWeaponBodies(rel string, source []byte, d Data) ([]byte, error) {
	before, err := ParseWeaponBodies("", rel, source)
	if err != nil {
		return nil, err
	}
	after := weaponSourceRows(d.Bodies.Weapons)
	out, err := writeRows(source, weaponSourceRows(before), after, func(key string) string { return key })
	if err != nil {
		return nil, err
	}
	back, err := ParseWeaponBodies("", rel, out)
	if err != nil {
		return nil, err
	}
	if !sameRows(weaponSourceRows(back), after) {
		return nil, fmt.Errorf("%s: the written rows do not read back as the rows given", rel)
	}
	return out, nil
}

// bodySourceRows is the sheets as the row writer sees them: a kept sheet is
// matched by its line, so a rename changes one key and moves no table.
func bodySourceRows(bodies []BodySheet) []sourceRow {
	rows, lines := make([]sourceRow, len(bodies)), make([]int, len(bodies))
	for i, b := range bodies {
		pairs := []missionPair{{"name", strconv.Quote(b.Name)}}
		switch {
		case b.Off:
			pairs = append(pairs, missionPair{OffField, "true"})
		default:
			pairs = append(pairs,
				missionPair{"heroes", strconv.Quote(b.Heroes)},
				missionPair{"heroes_l", strconv.Quote(b.HeroesLight)},
				missionPair{"frame", fmt.Sprintf("[%d, %d]", b.Width, b.Height)},
				missionPair{"origin", fmt.Sprintf("[%d, %d]", b.OriginX, b.OriginY)},
				missionPair{"directions", strconv.Itoa(b.Directions)},
				missionPair{"move", bodyActionValue(b.Move)},
				missionPair{"attack", bodyActionValue(b.Attack)})
			if b.Idle.Frames > 0 || b.Idle.Line != 0 {
				pairs = append(pairs, missionPair{"idle", bodyActionValue(b.Idle)})
			}
			if b.WeaponLast {
				pairs = append(pairs, missionPair{"weapon-last", "true"})
			}
			if s := b.Selection; b.HasSelection {
				pairs = append(pairs, missionPair{"selection", fmt.Sprintf("[%d, %d, %d, %d]", s[0], s[1], s[2], s[3])})
			}
		}
		rows[i], lines[i] = sourceRow{table: "body", pairs: pairs}, b.Line
	}
	numberRows(rows, lines)
	return rows
}

// bodyActionValue is an action as the file writes it: ticks is one count when
// every frame holds the same, else one count per frame.
func bodyActionValue(a BodyAction) string {
	ticks := "1"
	if len(a.Ticks) > 0 {
		same := true
		parts := make([]string, len(a.Ticks))
		for i, t := range a.Ticks {
			same = same && t == a.Ticks[0]
			parts[i] = strconv.Itoa(t)
		}
		ticks = strconv.Itoa(a.Ticks[0])
		if !same {
			ticks = "[" + strings.Join(parts, ", ") + "]"
		}
	}
	return fmt.Sprintf("{ frames = %d, ticks = %s }", a.Frames, ticks)
}

// weaponSourceRows is the weapon bodies as the row writer sees them.
func weaponSourceRows(weapons []WeaponBody) []sourceRow {
	rows, lines := make([]sourceRow, len(weapons)), make([]int, len(weapons))
	for i, w := range weapons {
		var pairs []missionPair
		if w.Weapon != "" {
			pairs = append(pairs, missionPair{"weapon", strconv.Quote(w.Weapon)})
		}
		if w.Row != 0 {
			pairs = append(pairs, missionPair{"row", strconv.Itoa(w.Row)})
		}
		pairs = append(pairs, missionPair{"body", strconv.Quote(w.Body)})
		rows[i], lines[i] = sourceRow{table: "weapon", pairs: pairs}, w.Line
	}
	numberRows(rows, lines)
	return rows
}
