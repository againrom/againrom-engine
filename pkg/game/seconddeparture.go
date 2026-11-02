package game

import "slices"

// secondDepartureExit is a location an ordinary departure adds when its gates
// hold.
type secondDepartureExit struct {
	target partyPlace
	gates  []SecondGate
}

// secondDepartureWrite is a bank slot a departure stores.
type secondDepartureWrite struct {
	slot  int
	value int32
}

// secondDepartureMovie is an output DWORD a departure answers when its gates
// hold; the first held one answers.
type secondDepartureMovie struct {
	movie int
	gates []SecondGate
}

// secondDepartureRow declares one ordinary-departure case: the locations it
// adds, the slots it stores and the movie it answers (R2-ENGINE-145,
// R2-ENGINE-146, R2-ENGINE-148). Every gate reads the bank as the mission
// left it, before the row's writes.
type secondDepartureRow struct {
	from   int
	exits  []secondDepartureExit
	writes []secondDepartureWrite
	output []secondDepartureMovie
}

func secondMission(id int) partyPlace { return partyPlace{kind: placeMission, id: id} }
func secondTown(id int) partyPlace    { return partyPlace{kind: placeTown, id: id} }

// secondDepartures is the one declaration of every ordinary departure, in
// case-body order. A mission with no row stores no output and adds nothing.
var secondDepartures = []secondDepartureRow{
	{from: 10, exits: []secondDepartureExit{{target: secondMission(20)}}, output: []secondDepartureMovie{{movie: 1}}},
	{from: 20,
		exits:  []secondDepartureExit{{target: secondTown(2)}, {target: secondMission(21), gates: []SecondGate{{772, true}}}},
		writes: []secondDepartureWrite{{532, 1}, {552, 2}}},
	{from: 30, output: []secondDepartureMovie{{movie: 2}}},
	{from: 31, exits: []secondDepartureExit{{target: secondMission(32)}}},
	{from: 40,
		exits:  []secondDepartureExit{{target: secondMission(50)}, {target: secondMission(60)}},
		writes: []secondDepartureWrite{{773, 23}}},
	// The bank780-gated fixed record is mission 52 (R2-ENGINE-419).
	{from: 50,
		exits:  []secondDepartureExit{{target: secondTown(3)}, {target: secondMission(52), gates: []SecondGate{{780, true}}}},
		writes: []secondDepartureWrite{{771, 1}, {534, 1}, {554, 3}}},
	{from: 60,
		exits:  []secondDepartureExit{{target: secondMission(80)}},
		writes: []secondDepartureWrite{{537, 1}, {557, 2}, {774, 1}}},
	{from: 70,
		writes: []secondDepartureWrite{{535, 1}, {555, 3}, {777, 1}, {770, 1}},
		output: []secondDepartureMovie{{movie: 3, gates: []SecondGate{{777, false}, {778, true}}}}},
	{from: 80,
		writes: []secondDepartureWrite{{778, 1}},
		output: []secondDepartureMovie{{movie: 3, gates: []SecondGate{{778, false}, {777, true}}}}},
	{from: 90},
	{from: 100, writes: []secondDepartureWrite{{538, 1}, {558, 2}}},
	{from: 110, output: []secondDepartureMovie{
		{movie: 5, gates: []SecondGate{{779, true}}},
		{movie: 4, gates: []SecondGate{{779, false}}}}},
}

// secondDepartureFor answers mission n's departure row.
func secondDepartureFor(n int) (secondDepartureRow, bool) {
	i := slices.IndexFunc(secondDepartures, func(d secondDepartureRow) bool { return d.from == n })
	if i < 0 {
		return secondDepartureRow{}, false
	}
	return secondDepartures[i], true
}

func secondGatesHold(bank *[1024]int32, gates []SecondGate) bool {
	for _, g := range gates {
		if (bank[g.Slot] != 0) != g.Nonzero {
			return false
		}
	}
	return true
}

// run applies the row to the campaign's bank: it adds each exit whose gates
// hold through add, stores the writes and answers the movie output, -1 when
// none holds.
func (d secondDepartureRow) run(c *secondCampaign, add func(partyPlace)) int {
	output := -1
	for _, e := range d.exits {
		if secondGatesHold(&c.bank, e.gates) {
			add(e.target)
		}
	}
	for _, m := range d.output {
		if output < 0 && secondGatesHold(&c.bank, m.gates) {
			output = m.movie
		}
	}
	for _, w := range d.writes {
		c.bank[w.slot] = w.value
	}
	return output
}

// enterLocation clears the slots a location entry owns, 752..767, in the
// bank a mission seats or a town entry holds (R2-ENGINE-048).
func enterLocation(bank *[1024]int32) { clear(bank[752:768]) }

// secondDepartureReset clears the slot a departure row may store a pending
// value in, before any row runs.
func secondDepartureReset(bank *[1024]int32) { bank[773] = 0 }
